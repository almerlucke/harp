package blocks

import (
	"math"
	"testing"

	"harp/dsl"

	"github.com/almerlucke/psalm/blocks/delay"
	"github.com/almerlucke/psalm/blocks/filters/biquad"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
)

const sr = 44100.0

func render(out dsl.Sig, seconds float64) [][]float64 {
	_, ch := dsl.Patch{SampleRate: sr}.Render(out, seconds)
	return ch
}

func maxDiff(a, b []float64) float64 {
	var d float64
	for i := range a {
		d = math.Max(d, math.Abs(a[i]-b[i]))
	}
	return d
}

func rms(x []float64) float64 {
	var s float64
	for _, v := range x {
		s += v * v
	}
	return math.Sqrt(s / float64(len(x)))
}

// impulse is 1 at the first sample of the stream and 0 after it (made with a custom node)
func impulse() dsl.Sig {
	return dsl.Custom(dsl.Spec{Name: "impulse", Key: "impulse", Inputs: []dsl.Sig{dsl.N(0)}, New: func(rt *dsl.Rt) dsl.Process {
		return func(_ [][]float64, out []float64, n int) {
			for i := 0; i < n; i++ {
				out[i] = 0
				if rt.Pos()+int64(i) == 0 {
					out[i] = 1
				}
			}
		}
	}})
}

// The node is the block of psalm, no more and no less
func TestLowpassIsThePsalmBlock(t *testing.T) {
	saw := render(dsl.Saw(220), 0.2)[0]
	got := render(Lowpass(dsl.Saw(220), 1000, 0.5), 0.2)[0]
	f := svf.New(1000, 0.5, sr)
	want := make([]float64, len(saw))
	for i, x := range saw {
		want[i] = f.Process(x, svf.LowPass)
	}
	if d := maxDiff(got, want); d != 0 {
		t.Errorf("differs from the block by %g", d)
	}
}

func TestModes(t *testing.T) {
	dc := func(out dsl.Sig) float64 { c := render(out, 0.2)[0]; return c[len(c)-1] }
	for _, c := range []struct {
		name string
		got  float64
		want float64
	}{
		{"lowpass passes DC", dc(Lowpass(dsl.N(1), 1000, 0.5)), 1},
		{"highpass stops DC", dc(Highpass(dsl.N(1), 1000, 0.5)), 0},
		{"bandpass stops DC", dc(Bandpass(dsl.N(1), 1000, 0.5)), 0},
		{"notch passes DC", dc(Notch(dsl.N(1), 1000, 0.5)), 1},
		{"allpass passes DC", dc(Allpass(dsl.N(1), 1000, 0.5)), 1},
		{"peak is low minus high: DC passes", dc(Peak(dsl.N(1), 1000, 0.5)), 1},
	} {
		if math.Abs(c.got-c.want) > 1e-6 {
			t.Errorf("%s: %v, want %v", c.name, c.got, c.want)
		}
	}
	// a sine far above the cutoff: the lowpass takes it away, the highpass leaves it
	high := dsl.Sine(10000.0)
	lp, hp := rms(render(Lowpass(high, 500, 0.5), 0.2)[0][4410:]), rms(render(Highpass(high, 500, 0.5), 0.2)[0][4410:])
	if lp > 0.01 || hp < 0.6 {
		t.Errorf("rms of 10 kHz through 500 Hz: lowpass %v, highpass %v", lp, hp)
	}
}

// Parameters are read at samples of the stream, not of the block: the sound is the same however
// the blocks are cut (by steps, by the buffers of a device)
func TestControlGridDoesNotDependOnTheBlocks(t *testing.T) {
	patch := func() dsl.Sig {
		cut := dsl.Lfo(7.0).Range(200, 6000)
		return Ladder(Lowpass(dsl.Saw(110), cut, dsl.Lfo(3.0).Range(0.1, 0.9)), ladder.LowPass24, cut, 0.6, 0.3)
	}
	p := dsl.Patch{SampleRate: sr, BPM: 137, StepsPerBeat: 3}
	const total = 20000
	ref := make([]float64, total)
	p.Start(patch()).Process(ref)
	for _, piece := range []int{1, 37, 64, 100, 512, 4999} {
		got := make([]float64, total)
		r := p.Start(patch())
		for i := 0; i < total; i += piece {
			r.Process(got[i:min(i+piece, total)])
		}
		if d := maxDiff(got, ref); d != 0 {
			t.Errorf("pieces of %d differ from one piece by %g", piece, d)
		}
	}
}

// A stereo input is two filters, each with a state of its own
func TestChannelsHaveTheirOwnState(t *testing.T) {
	both := render(Lowpass(dsl.Chans(dsl.Saw(220), dsl.Saw(331)), dsl.Lfo(2.0).Range(300, 3000), 0.7), 0.3)
	a := render(Lowpass(dsl.Saw(220), dsl.Lfo(2.0).Range(300, 3000), 0.7), 0.3)[0]
	b := render(Lowpass(dsl.Saw(331), dsl.Lfo(2.0).Range(300, 3000), 0.7), 0.3)[0]
	if maxDiff(both[0], a) != 0 || maxDiff(both[1], b) != 0 {
		t.Error("a channel differs from the filter alone")
	}
}

// What a node is is in its key: the same node twice is one, another mode or another size is not
func TestKeysMerge(t *testing.T) {
	k := func(out dsl.Sig) int { return dsl.Patch{}.Start(out).Program().Kernels() }
	x := dsl.Saw(220)
	same := k(dsl.Add(Lowpass(x, 1000, 0.5), Lowpass(x, 1000, 0.5)))
	mode := k(dsl.Add(Lowpass(x, 1000, 0.5), Highpass(x, 1000, 0.5)))
	cutoff := k(dsl.Add(Lowpass(x, 1000, 0.5), Lowpass(x, 2000, 0.5)))
	// saw, the filters, and the add (a constant is a buffer, it runs no kernel)
	if same != 3 || mode != 4 || cutoff != 4 {
		t.Errorf("kernels: same %d (3), other mode %d (4), other cutoff %d (4)", same, mode, cutoff)
	}
	drive := k(dsl.Add(Ladder(x, ladder.LowPass24, 1000, 0.5, 0), Ladder(x, ladder.LowPass24, 1000, 0.5, 0.5)))
	if drive != 4 {
		t.Errorf("ladders with another drive merged: %d kernels", drive)
	}
}

func TestLadderIsSteeperThanSVF(t *testing.T) {
	tone := dsl.Sine(4000.0)
	two := rms(render(Lowpass(tone, 1000, 0.3), 0.3)[0][4410:])
	four := rms(render(Ladder(tone, ladder.LowPass24, 1000, 0.3, 0), 0.3)[0][4410:])
	if !(four < two/3) {
		t.Errorf("rms two poles %v, four poles %v: the ladder should be much lower", two, four)
	}
}

func TestBiquadIsThePsalmBlock(t *testing.T) {
	saw := render(dsl.Saw(220), 0.1)[0]
	got := render(Biquad(dsl.Saw(220), biquad.Peak, 800, 1.5, 6), 0.1)[0]
	f := biquad.New(biquad.Peak, 800, 1.5, 6, sr)
	want := make([]float64, len(saw))
	for i, x := range saw {
		want[i] = f.Process(x)
	}
	if d := maxDiff(got, want); d != 0 {
		t.Errorf("differs from the block by %g", d)
	}
}

func TestFormant(t *testing.T) {
	a := render(Formant(dsl.Saw(110), 0, 10), 0.3)[0]
	i := render(Formant(dsl.Saw(110), 2, 10), 0.3)[0]
	if rms(a) < 0.01 || rms(i) < 0.01 || maxDiff(a, i) < 0.1 {
		t.Errorf("rms %v and %v, differ by %v: the vowels should sound and differ", rms(a), rms(i), maxDiff(a, i))
	}
	for _, v := range a {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatal("not finite")
		}
	}
}

func TestDelayIsExact(t *testing.T) {
	const small = 1000.0
	p := dsl.Patch{SampleRate: small}
	for _, c := range []struct {
		seconds float64
		at      int
	}{{0.010, 10}, {0.001, 1}, {0.0, 1} /* the shortest is one sample */, {0.050, 50}} {
		_, ch := p.Render(Delay(impulse(), 0.1, c.seconds), 0.2)
		for i, v := range ch[0] {
			want := 0.0
			if i == c.at {
				want = 1
			}
			if math.Abs(v-want) > 1e-12 {
				t.Fatalf("delay %v: sample %d is %v, want %v", c.seconds, i, v, want)
			}
		}
	}
	// a time that is not a whole number of samples is between two
	_, ch := p.Render(Delay(impulse(), 0.1, 0.0105), 0.2)
	if ch[0][10] <= 0 || ch[0][11] <= 0 || math.Abs(ch[0][10]+ch[0][11]-1) > 0.3 {
		t.Errorf("a delay of 10.5 samples: %v %v", ch[0][10], ch[0][11])
	}
	// every channel has a line of its own, with the time of both
	_, st := p.Render(Delay(dsl.Chans(impulse(), dsl.Mul(impulse(), 0.5)), 0.1, 0.020), 0.2)
	if st[0][20] != 1 || st[1][20] != 0.5 {
		t.Errorf("stereo delay: %v %v", st[0][20], st[1][20])
	}
}

func TestEchoFeedsBack(t *testing.T) {
	p := dsl.Patch{SampleRate: 1000}
	_, only := p.Render(Echo(impulse(), 0.1, 0.010, 0.5, 1.0), 0.1) // only the echoes
	for _, c := range []struct {
		at   int
		want float64
	}{{0, 0}, {10, 1}, {20, 0.5}, {30, 0.25}, {40, 0.125}, {15, 0}} {
		if math.Abs(only[0][c.at]-c.want) > 1e-12 {
			t.Errorf("echoes only, sample %d: %v, want %v", c.at, only[0][c.at], c.want)
		}
	}
	_, dry := p.Render(Echo(impulse(), 0.1, 0.010, 0.5, 0.0), 0.1) // none of them
	if dry[0][0] != 1 || dry[0][10] != 0 || dry[0][20] != 0 {
		t.Errorf("dry: %v %v %v", dry[0][0], dry[0][10], dry[0][20])
	}
	_, half := p.Render(Echo(impulse(), 0.1, 0.010, 0.5, 0.5), 0.1)
	if math.Abs(half[0][0]-0.5) > 1e-12 || math.Abs(half[0][10]-0.5) > 1e-12 {
		t.Errorf("half: %v %v", half[0][0], half[0][10])
	}
}

// A chorus: a delay with a time that moves
func TestDelayWithAMovingTime(t *testing.T) {
	chorus := dsl.Add(dsl.Saw(220), Delay(dsl.Saw(220), 0.05, dsl.Lfo(0.5).Range(0.010, 0.020)))
	for _, v := range render(chorus, 0.5)[0] {
		if math.IsNaN(v) || math.Abs(v) > 2.1 {
			t.Fatalf("sample %v", v)
		}
	}
}

func TestOscIsThePsalmBlock(t *testing.T) {
	got := render(Osc(blosc.Sawtooth, 440, 0.5), 0.1)[0]
	o := blosc.New(440, sr, 0, 0.5, 1, blosc.Sawtooth)
	want := make([]float64, len(got))
	for i := range want {
		want[i] = o.Tick(0, false, 0)
	}
	if d := maxDiff(got, want); d != 0 {
		t.Errorf("differs from the block by %g", d)
	}
	// a frequency that is a signal is read every sample: fm
	fm := render(Osc(blosc.Sine, dsl.Add(440, dsl.Mul(dsl.Sine(110.0), 200)), 0.5), 0.1)[0]
	plain := render(Osc(blosc.Sine, 440, 0.5), 0.1)[0]
	if maxDiff(fm, plain) < 0.1 {
		t.Error("fm sounds like the carrier")
	}
	// the pulse width moves the square
	narrow, wide := render(Osc(blosc.Rectangle, 100, 0.1), 0.1)[0], render(Osc(blosc.Rectangle, 100, 0.9), 0.1)[0]
	var hi1, hi2 int
	for i := range narrow {
		if narrow[i] > 0 {
			hi1++
		}
		if wide[i] > 0 {
			hi2++
		}
	}
	if hi1 == hi2 {
		t.Errorf("pulse width has no effect: %d and %d samples high", hi1, hi2)
	}
}

// the raw delay of psalm reads the sample that was written n samples ago, which the node relies on
func TestPsalmDelayAssumption(t *testing.T) {
	d := delay.New(16)
	for i := 0; i < 10; i++ {
		d.Write(float64(i))
	}
	if got := d.ReadHermite(3); got != 7 { // before writing the 11th, 3 back is the 8th: 7
		t.Errorf("ReadHermite(3) = %v, want 7", got)
	}
}

// The same patch with the lowpass of dsl (coefficients every sample) and the one of psalm (every 16)
func BenchmarkLowpass(b *testing.B) {
	cut := func() dsl.Sig { return dsl.Lfo(0.3).Range(300, 4000) }
	for _, c := range []struct {
		name  string
		build func() dsl.Sig
	}{
		{"dsl", func() dsl.Sig { return dsl.Saw(110).Lowpass(cut(), dsl.N(0.9)) }},
		{"psalm", func() dsl.Sig { return Lowpass(dsl.Saw(110), cut(), 0.5) }},
	} {
		b.Run(c.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				render(c.build(), 5)
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/(5*sr), "ns/sample")
		})
	}
}
