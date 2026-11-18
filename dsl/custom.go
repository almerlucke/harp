package dsl

import "fmt"

// Custom nodes: how a processor of your own (or one from another package, like the blocks of psalm)
// becomes a node in the graph. A custom node expands over channels like the built in ones, so
// Spec.New is called once for every channel, and every channel has a state of its own.

// Process computes the first n samples of out from the first n samples of every input. in has the
// buffers of Spec.Inputs, for this channel
type Process func(in [][]float64, out []float64, n int)

// Spec describes a custom node
type Spec struct {
	// Name is shown in a listing of the program
	Name string
	// Key says what the node computes from its inputs: the parameters that are not inputs (a mode,
	// a size). Nodes with the same Key and the same inputs are one node, so the key must name
	// everything that makes the output different. An empty Key is never merged: use it if the
	// process takes values from somewhere else (a generator, a file, a random stream)
	Key string
	// Inputs are the signals the node reads. Channels expand like in every node: one channel goes
	// to all the channels of the others, otherwise the numbers must be equal
	Inputs []Sig
	// New makes the state of one channel, when the program is built
	New func(rt *Rt) Process
}

// Custom makes a node of your own, for every channel of its inputs. Signal numbers: use Signal for
// a parameter that may be a number or a signal
func Custom(s Spec) Sig {
	if gate, ok := currentReset(); ok {
		s = s.resetting(gate)
	}
	return expand(s.Name, s.Inputs, func(in []*node) *node {
		return &node{name: s.Name, key: s.Key, in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			p := s.New(rt)
			return func(n int) { p(b, out, n) }
		}}
	})
}

// Signal makes a signal of a number or a signal: for a function that takes a Param
func Signal[P Param](p P) Sig { return sig(p) }

// Pos is the position in the stream of the first sample of the block that is being computed: the
// number of samples since the start. Use it to do something every so many samples (update filter
// coefficients) at the same samples whatever the size of the blocks, which depends on where steps
// start and on the size of the buffers of the device
func (rt *Rt) Pos() int64 { return rt.pos }

// ProcessN is Process for a node with several outputs: out has a buffer for each of them
type ProcessN func(in [][]float64, out [][]float64, n int)

// SpecN describes a custom node with several outputs: a stereo reverb that takes a left and a right
// and gives a left and a right, a crossover with a low and a high output
type SpecN struct {
	Name string
	// Inputs are the signals the node reads. They expand like in Spec: the node is made once for every
	// channel of the widest, and a mono input goes to all of them
	Inputs []Sig
	// Key says what the node is, for a program that takes over from another one (Runner.Swap): the same key is the same node, with its state kept.
	// Empty: it is never kept. A node that takes values from generators puts the fingerprint of them in it (dsl.Fingerprint)
	Key string
	// State is the state that the outputs of an instance share, if the caller has it (otherwise the instances make their own): a program that takes
	// over from another can put a new generator in it (it is a rebaser, see swap.go)
	State any
	// Outputs is the number of outputs of the node. The result has Outputs channels for every
	// instance, the outputs of an instance together: out 0 and 1 of the first, out 0 and 1 of the second...
	Outputs int
	// New makes the state of one instance, when the program is built
	New func(rt *Rt) ProcessN
}

// CustomN makes a node of your own with several outputs. It is never merged with another: its
// outputs belong together, and there is no key to tell what it computes
func CustomN(s SpecN) Sig {
	if gate, ok := currentReset(); ok {
		s = s.resetting(gate)
	}
	if s.Outputs < 1 {
		panic("dsl: CustomN: a node needs at least one output")
	}
	var out Sig
	for c := 0; c < width(s.Name, s.Inputs...); c++ {
		in := make([]*node, len(s.Inputs))
		for i, sig := range s.Inputs {
			in[i] = sig.at(c).ch[0]
		}
		// The first output computes all of them, the others copy theirs from what it leaves in
		// its buffers: they have the first as an input, so it always runs before them
		var bufs [][]float64
		var group any = &struct{ _ byte }{} // the outputs of an instance share state: they are kept together
		if s.State != nil {
			group = s.State
		}
		lead := &node{name: s.Name, id: s.Key, group: group, in: in, build: func(rt *Rt, b [][]float64, first []float64) Kernel {
			mine := make([][]float64, s.Outputs)
			mine[0] = first
			for k := 1; k < len(mine); k++ {
				mine[k] = make([]float64, BlockSize)
			}
			bufs = mine
			p := s.New(rt)
			return func(n int) { p(b, mine, n) }
		}}
		out.ch = append(out.ch, lead)
		for k := 1; k < s.Outputs; k++ {
			out.ch = append(out.ch, &node{name: fmt.Sprintf("%s[%d]", s.Name, k), id: s.Key, group: group, in: []*node{lead},
				build: func(_ *Rt, _ [][]float64, o []float64) Kernel {
					src := bufs[k] // set when the first output was built, which is before this one
					return func(n int) { copy(o[:n], src[:n]) }
				}})
		}
	}
	return out
}

// WithReset runs build, and every custom node that it makes starts again from its state at the start (a new run of New) on the
// samples where gate rises from 0 to above 0: as a synth of SuperCollider that is made new for every note. The nodes get the gate as
// one more input (the last: the buffers of the others are as they were). Nodes are made one at a time, so this is a plain scope
func WithReset(gate Sig, build func()) {
	resetGates = append(resetGates, gate)
	defer func() { resetGates = resetGates[:len(resetGates)-1] }()
	build()
}

var resetGates []Sig

func currentReset() (Sig, bool) {
	if len(resetGates) == 0 {
		return Sig{}, false
	}
	return resetGates[len(resetGates)-1], true
}

// segments calls f for every stretch of the n samples without a rise of the gate in it, with the offset it starts at; before a
// stretch that starts at a rise, fresh is called
func segments(g []float64, prev *float64, n int, fresh func(), f func(from, to int)) {
	start := 0
	for i := 0; i < n; i++ {
		if *prev <= 0 && g[i] > 0 && i > 0 {
			f(start, i)
			start = i
			fresh()
		} else if *prev <= 0 && g[i] > 0 {
			fresh()
		}
		*prev = g[i]
	}
	f(start, n)
}

func (s Spec) resetting(gate Sig) Spec {
	inner := s.New
	nin := len(s.Inputs)
	s.Inputs = append(append([]Sig(nil), s.Inputs...), gate)
	s.New = func(rt *Rt) Process {
		p := inner(rt)
		var prev float64
		sub := make([][]float64, nin)
		return func(in [][]float64, out []float64, n int) {
			segments(in[nin], &prev, n, func() { p = inner(rt) }, func(from, to int) {
				if to <= from {
					return
				}
				for k := range sub {
					sub[k] = in[k][from:to]
				}
				rt.pos += int64(from)
				p(sub, out[from:to], to-from)
				rt.pos -= int64(from)
			})
		}
	}
	return s
}

func (s SpecN) resetting(gate Sig) SpecN {
	inner := s.New
	nin := len(s.Inputs)
	s.Inputs = append(append([]Sig(nil), s.Inputs...), gate)
	s.New = func(rt *Rt) ProcessN {
		p := inner(rt)
		var prev float64
		sub := make([][]float64, nin)
		subOut := make([][]float64, s.Outputs)
		return func(in [][]float64, out [][]float64, n int) {
			segments(in[nin], &prev, n, func() { p = inner(rt) }, func(from, to int) {
				if to <= from {
					return
				}
				for k := range sub {
					sub[k] = in[k][from:to]
				}
				for k := range subOut {
					subOut[k] = out[k][from:to]
				}
				rt.pos += int64(from)
				p(sub, subOut, to-from)
				rt.pos -= int64(from)
			})
		}
	}
	return s
}
