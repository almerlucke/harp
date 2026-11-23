// Package dsl is an embedded DSL for sound synthesis with generators.
//
// There are two rates, and the Go types keep them apart:
//
//   - event rate: Gen[T] (genny generators), pulled once per step of the clock, on a control
//     goroutine ahead of the audio (see Runner.Go), never in the audio loop
//   - audio rate: Sig, a node in a signal graph, computed per sample
//
// Lifts (Hold, Pitch, Gate, Notes) turn a Gen into a Sig. Building a patch only builds a graph; Compile
// turns the graph into a flat Program with preallocated buffers, and that program is what runs.
package dsl

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/almerlucke/genny"
)

// Gen is an event rate value source: genny's generator interface
type Gen[T any] = genny.Generator[T]

// Seed seeds all generators created after it, and the numbers that are drawn while a patch is built
func Seed(n uint64) {
	genny.Seed(n)
	SeedBuild(n)
}

// the numbers that are drawn while a patch is built (Random, the seeds of the noises) come from a stream of their own, so that a patch that
// is built again while it plays (a live edit) can draw the same numbers again without moving the stream that its generators are drawing from
var build struct {
	mu   sync.Mutex
	rng  *rand.Rand
	seed uint64
}

// SeedBuild seeds the numbers that are drawn while a patch is built, and only those
func SeedBuild(n uint64) {
	build.mu.Lock()
	build.rng, build.seed = rand.New(rand.NewPCG(n, n^0x9e3779b97f4a7c15)), n
	build.mu.Unlock()
}

// BuildSeed is the seed that the numbers of the build were last seeded with (0 if they never were)
func BuildSeed() uint64 {
	build.mu.Lock()
	defer build.mu.Unlock()
	return build.seed
}

func buildFloat() float64 {
	build.mu.Lock()
	defer build.mu.Unlock()
	if build.rng == nil {
		build.rng = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 1))
	}
	return build.rng.Float64()
}

func buildIntn(n int) int {
	build.mu.Lock()
	defer build.mu.Unlock()
	if build.rng == nil {
		build.rng = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 1))
	}
	return build.rng.IntN(n)
}

// BlockSize is the number of samples a program computes per kernel call at most
const BlockSize = 64

// Rt is what a node can use while it is being compiled
type Rt struct {
	SampleRate float64
	// StepSamples is the length of a step in samples (fractional)
	StepSamples float64
	lanes       []lane
	// Live says that the sound is made for a device, on time: the generators run on a goroutine ahead of it (Runner.Go). When it is false the
	// patch is rendered as fast as it can be, which a node that waits for something outside (a reader of files) has to know
	Live bool
	sink *[]lane // while a node is built: where the lanes it registers go (the node owns them)
	pos  int64   // the first sample of the block that runs, see Pos
	// skipTo is a sample before which no event starts: while a runner skips ahead (Runner.Skip) the notes that would have started in what is skipped are not played
	skipTo int64
}

// Kernel computes n samples (n <= BlockSize)
type Kernel func(n int)

type opKind uint8

const (
	opOpaque opKind = iota // a kernel of its own: build says how
	opConst                // a constant, a
	opAdd                  // in[0] + in[1]
	opMul                  // in[0] * in[1]
	opAffine               // in[0] * a + b
)

// node is one operation in the audio graph. The elementwise operations (const, add, mul, affine)
// are known to the optimizer, which folds and fuses them; every other node is opaque: build is
// called once by Compile, with the buffers of the inputs and its own output buffer, and returns
// the kernel that runs every block (nil: nothing to run)
//
// Nodes are never changed once made: the optimizer makes new ones, so a Sig can be used in
// several patches
type node struct {
	name  string
	op    opKind
	in    []*node
	build func(rt *Rt, in [][]float64, out []float64) Kernel

	// opaque nodes only: key names what the node computes from its inputs (the parameters, not
	// the inputs), so two nodes with the same key and the same inputs give the same signal and
	// are merged. Empty: never merged (a node that takes its values from a generator)
	key string
	// id, gens and group are for a program that takes over from another one (Runner.Swap), see swap.go. id names what the node is when it
	// is not its key (a node that has no key, because it is never merged, can still have an id); gens is the fingerprint of the generators
	// whose values it takes (empty: not known); group is the state that several nodes share (they are kept, or not, together)
	id, gens string
	lane     bool // the node takes values from generators: it is only kept if their fingerprint (gens) is known
	group    any
	// buildPost is build for a node that can scale and offset its own output (out*a+b) as part of
	// its loop; the optimizer moves a following affine into it
	buildPost func(rt *Rt, in [][]float64, out []float64, a, b float64) Kernel

	// const: the value (a). affine: in[0]*a + b. With post set, a and b are an affine applied to
	// what the node computes (add, mul or a postable opaque node): (x+y)*a + b
	a, b float64
	post bool
}

// Sig is a signal of one or more channels: audio rate values, one per sample per channel. A mono
// signal has one channel, a stereo one two. Every channel is a node of its own in the graph, so
// what the engine runs is mono nodes only: the channels cost nothing but the nodes they are
type Sig struct{ ch []*node }

// Channels is the number of channels of the signal
func (s Sig) Channels() int { return len(s.ch) }

// Ch is one channel of the signal, as a mono signal
func (s Sig) Ch(i int) Sig {
	if i < 0 || i >= len(s.ch) {
		panic(fmt.Sprintf("dsl: Ch(%d) of a signal with %d channels", i, len(s.ch)))
	}
	return Sig{ch: s.ch[i : i+1 : i+1]}
}

// mono is a signal of one node
func mono(n *node) Sig { return Sig{ch: []*node{n}} }

// Param is anything that can be given where a signal is wanted: a Sig, or a constant
type Param interface {
	float64 | int | Sig
}

// N makes a constant signal
func N(v float64) Sig { return mono(&node{name: fmt.Sprintf("const(%g)", v), op: opConst, a: v}) }

func sig[P Param](p P) Sig {
	switch v := any(p).(type) {
	case Sig:
		return v
	case float64:
		return N(v)
	default: // int, the only other type in Param
		return N(float64(v.(int)))
	}
}

// Program is a compiled signal graph: a flat list of kernels over preallocated buffers
type Program struct {
	rt        *Rt
	kernels   []Kernel
	lanes     []lane      // all the lanes of the program, in the order of the records of the control side
	entries   []*entry    // the nodes that were built, in run order: what a program that takes over from this one is matched with
	roots     []*entry    // the entries of the outputs
	groupSize map[any]int // how many nodes of each group of nodes that share state the program has
	rebases   []func()    // generators that changed inside nodes that are kept: put in place by the control goroutine when it hands the program over
	fills     []func()    // the constants that are written when the program takes over from another (their buffers are in use until then)
	names     []string
	outs      [][]float64 // the buffer of every output channel: one for mono, two for stereo
	consts    int         // buffers that are filled once, when the program is made: they run no kernel
	// Opt is what the optimizer did (zero if it was off)
	Opt Report
}

// String lists the program, one kernel per line in run order
func (p *Program) String() string {
	var b strings.Builder
	for i, n := range p.names {
		fmt.Fprintf(&b, "%2d %s\n", i, n)
	}
	if p.consts > 0 {
		fmt.Fprintf(&b, "   (+ %d constants, filled once)\n", p.consts)
	}
	return b.String()
}

// Kernels is the number of kernels that run every block
func (p *Program) Kernels() int { return len(p.kernels) }

// Compile optimizes the graph under out and builds the kernels
func Compile(out Sig, sampleRate, stepSamples float64) *Program {
	return compile(out, sampleRate, stepSamples, true)
}

// compile sorts the graph under out topologically and builds the kernels. A node used in several
// places is built (and computed) once
func compile(out Sig, sampleRate, stepSamples float64, optimized bool) *Program {
	return compileFrom(nil, out, sampleRate, stepSamples, optimized)
}

// build makes the kernel of one node, and its name for the listing
func (p *Program) build(n *node, in [][]float64, out []float64) (Kernel, string) {
	post := ""
	if n.post {
		post = fmt.Sprintf(" *%g%+g", n.a, n.b)
	}
	switch n.op {
	case opConst:
		for i := range out {
			out[i] = n.a
		}
		p.consts++
		return nil, ""
	case opAdd:
		return addKernel(in[0], in[1], out, n.post, n.a, n.b), "add" + post
	case opMul:
		return mulKernel(in[0], in[1], out, n.post, n.a, n.b), "mul" + post
	case opAffine:
		return affineKernel(in[0], out, n.a, n.b), fmt.Sprintf("affine *%g%+g", n.a, n.b)
	}
	if n.post && n.buildPost != nil {
		return n.buildPost(p.rt, in, out, n.a, n.b), n.name + post
	}
	return n.build(p.rt, in, out), n.name
}

func affineKernel(x, out []float64, a, b float64) Kernel {
	switch {
	case b == 0:
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = x[i] * a
			}
		}
	case a == 1:
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = x[i] + b
			}
		}
	}
	return func(n int) {
		for i := 0; i < n; i++ {
			out[i] = x[i]*a + b
		}
	}
}

func mulKernel(x, y, out []float64, post bool, a, b float64) Kernel {
	switch {
	case !post:
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = x[i] * y[i]
			}
		}
	case b == 0:
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = x[i] * y[i] * a
			}
		}
	}
	return func(n int) {
		for i := 0; i < n; i++ {
			out[i] = x[i]*y[i]*a + b
		}
	}
}

func addKernel(x, y, out []float64, post bool, a, b float64) Kernel {
	switch {
	case !post:
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = x[i] + y[i]
			}
		}
	case b == 0:
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = (x[i] + y[i]) * a
			}
		}
	}
	return func(n int) {
		for i := 0; i < n; i++ {
			out[i] = (x[i]+y[i])*a + b
		}
	}
}

// Random is a number from 0 up to 1 from the stream of Seed: the one the generators draw from, for what is decided while the patch is built
func Random() float64 { return buildFloat() }
