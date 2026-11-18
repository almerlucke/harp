package dsl

import (
	"math"
	"strings"
	"testing"
)

func program(out Sig, noOpt bool) *Program {
	return Patch{NoOptimize: noOpt}.Start(out).Program()
}

func maxDiff(a, b []float64) float64 {
	var d float64
	for i := range a {
		d = math.Max(d, math.Abs(a[i]-b[i]))
	}
	return d
}

// the same patch, with and without the optimizer
func both(t *testing.T, build func() Sig, seconds float64) (opt, plain []float64) {
	t.Helper()
	Seed(3)
	_, opt = Patch{BPM: 110}.render1(build(), seconds)
	Seed(3)
	_, plain = Patch{BPM: 110, NoOptimize: true}.render1(build(), seconds)
	return
}

func TestOptimizedSoundsTheSame(t *testing.T) {
	opt, plain := both(t, lead, 4)
	// folding affines reorders floating point operations: the same sound, to a rounding
	if d := maxDiff(opt, plain); d > 1e-9 {
		t.Errorf("optimized differs from plain by %g", d)
	}
	if program(lead(), false).Kernels() >= program(lead(), true).Kernels() {
		t.Error("the optimizer did not take out a kernel")
	}
}

func TestConstantsFold(t *testing.T) {
	prog, s := Patch{}.render1(Mul(N(2), N(3)), 0.01)
	if prog.Kernels() != 0 || s[0] != 6 || s[len(s)-1] != 6 {
		t.Errorf("%d kernels, first sample %v: want none and 6", prog.Kernels(), s[0])
	}
	if prog.Opt.Folded != 1 {
		t.Errorf("report: %+v", prog.Opt)
	}
}

// A constant is a buffer filled once, whether the optimizer is on or not
func TestConstantsRunNoKernel(t *testing.T) {
	prog := program(Saw(220), true)
	if prog.Kernels() != 1 {
		t.Errorf("%d kernels: %s", prog.Kernels(), prog)
	}
}

func TestIdentitiesTakenOut(t *testing.T) {
	prog := program(Add(Mul(Saw(220), 1), 0), false)
	if prog.Kernels() != 1 || prog.Opt.Identities != 2 {
		t.Errorf("%d kernels, report %+v", prog.Kernels(), prog.Opt)
	}
}

func TestAffineMovesIntoOscillator(t *testing.T) {
	prog := program(Mul(Saw(220), 0.4), false)
	if prog.Kernels() != 1 || !strings.Contains(prog.names[0], "saw") || prog.Opt.Absorbed != 1 {
		t.Errorf("want one saw kernel, got %d: %s%v", prog.Kernels(), prog, prog.Opt)
	}
	// Range of an lfo is an affine of a sine: one kernel
	if k := program(Lfo(0.3).Range(300, 4000), false).Kernels(); k != 1 {
		t.Errorf("lfo range is %d kernels", k)
	}
	opt, plain := both(t, func() Sig { return Lfo(0.3).Range(300, 4000) }, 1)
	if d := maxDiff(opt, plain); d > 1e-9 {
		t.Errorf("differs by %g", d)
	}
}

func TestAffinesFoldToOne(t *testing.T) {
	build := func() Sig { return Add(Mul(Add(Mul(Saw(220), 0.5), 1), 0.5), 0.25) } // ((x*.5)+1)*.5+.25
	prog := program(build(), false)
	if prog.Kernels() != 1 {
		t.Errorf("%d kernels: %s", prog.Kernels(), prog)
	}
	opt, plain := both(t, build, 0.2)
	if d := maxDiff(opt, plain); d > 1e-12 {
		t.Errorf("differs by %g", d)
	}
}

func TestMulKeepsItsScale(t *testing.T) {
	// two signals multiplied and scaled: one kernel, not two
	build := func() Sig { return Mul(Mul(Saw(220), Sine(3)), 0.4) }
	if k := program(build(), false).Kernels(); k != 3 { // saw, sine, mul*0.4
		t.Errorf("%d kernels", k)
	}
	opt, plain := both(t, build, 0.2)
	if d := maxDiff(opt, plain); d != 0 { // same operations in the same order: the same bits
		t.Errorf("differs by %g", d)
	}
}

func TestDuplicatesMerge(t *testing.T) {
	build := func() Sig { return Add(Lfo(0.3), Lfo(0.3)) }
	prog := program(build(), false)
	var sines int
	for _, n := range prog.names {
		if strings.HasPrefix(n, "sine") {
			sines++
		}
	}
	if sines != 1 || prog.Opt.Merged < 1 {
		t.Errorf("%d sines, report %+v", sines, prog.Opt)
	}
	opt, plain := both(t, build, 0.2)
	if d := maxDiff(opt, plain); d != 0 {
		t.Errorf("differs by %g", d)
	}
}

// A node that more than one node uses is computed as it is: an affine after it must not change it
func TestSharedNodeIsNotChanged(t *testing.T) {
	build := func() Sig {
		s := Saw(220)
		return Add(Mul(s, 0.5), s)
	}
	opt, plain := both(t, build, 0.2)
	if d := maxDiff(opt, plain); d > 1e-12 {
		t.Errorf("differs by %g", d)
	}
	if prog := program(build(), false); strings.Contains(prog.names[0], "*") {
		t.Errorf("an affine moved into a node with two users: %s", prog)
	}
}

// Nodes that take values from a generator are never merged: two generators are two signals
func TestGeneratorNodesAreNotMerged(t *testing.T) {
	g1, g2 := Loop(1.0, 2.0), Loop(1.0, 2.0)
	prog := program(Add(Hold(g1), Hold(g2)), false)
	if prog.Opt.Merged != 0 || prog.Kernels() != 3 {
		t.Errorf("%d kernels, report %+v", prog.Kernels(), prog.Opt)
	}
	h := Hold(g1) // one node used twice stays one node: lifted once
	if k := program(Add(h, h), false).Kernels(); k != 2 {
		t.Errorf("%d kernels", k)
	}
}

type logGen struct {
	name string
	log  *[]string
}

func (g logGen) Generate() float64 { *g.log = append(*g.log, g.name); return 1 }
func (logGen) Continuous() bool    { return true }
func (logGen) Done() bool          { return false }
func (logGen) Reset()              {}

// The optimizer must not change the order in which generators are pulled: the same seed has to
// draw the same random numbers for the same notes
func TestGeneratorOrderIsKept(t *testing.T) {
	order := func(noOpt bool) string {
		var log []string
		a, b, c := logGen{"a", &log}, logGen{"b", &log}, logGen{"c", &log}
		x := Hold(b)
		patch := Add(Mul(Hold(a), 0.5), Add(Mul(x, Mul(Hold(c), 2)), Mul(x, 0.25)))
		Patch{SampleRate: 100, BPM: 60, StepsPerBeat: 1, NoOptimize: noOpt}.render1(patch, 3)
		return strings.Join(log, "")
	}
	if o, p := order(false), order(true); o != p {
		t.Errorf("optimized pulls %q, plain pulls %q", o, p)
	}
}

// The optimizer builds a new graph: what you made is as you made it
func TestOptimizerLeavesTheGraphAlone(t *testing.T) {
	saw := Saw(220)
	m := Mul(saw, 0.4)
	in0, a := m.ch[0].in[0], m.ch[0].in[1]
	program(m, false)
	if m.ch[0].op != opMul || m.ch[0].in[0] != in0 || m.ch[0].in[1] != a || m.ch[0].post || saw.ch[0].post {
		t.Error("the optimizer changed a node")
	}
	// and it can be compiled again, either way
	if program(m, true).Kernels() != 2 || program(m, false).Kernels() != 1 {
		t.Error("compiling again gives something else")
	}
}

// What the optimizer does to a patch, and what it is worth
func BenchmarkRender(b *testing.B) {
	stack := func() Sig { // a pad: eight detuned saws, each scaled, mixed and scaled again
		var mix Sig = N(0)
		for i := 0; i < 8; i++ {
			mix = Add(mix, Mul(Saw(110*(1+0.003*float64(i))), 0.1))
		}
		return Mul(Add(Mul(mix, 0.5), 0), Lfo(0.2).Range(0.5, 1))
	}
	for _, c := range []struct {
		name  string
		build func() Sig
	}{{"lead", lead}, {"stack", stack}} {
		for _, noOpt := range []bool{true, false} {
			name := c.name + "/optimized"
			if noOpt {
				name = c.name + "/plain"
			}
			b.Run(name, func(b *testing.B) {
				const seconds = 5
				p := Patch{BPM: 110, NoOptimize: noOpt}
				var prog *Program
				for i := 0; i < b.N; i++ {
					Seed(7)
					prog, _ = p.render1(c.build(), seconds)
				}
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/(seconds*44100), "ns/sample")
				b.ReportMetric(float64(prog.Kernels()), "kernels")
			})
		}
	}
}
