package dsl

import (
	"math"
	"testing"
)

func renderMono(t *testing.T, out Sig, sr, seconds float64) []float64 {
	t.Helper()
	_, ch := Patch{SampleRate: sr}.Render(out, seconds)
	if len(ch) != 1 {
		t.Fatalf("%d channels", len(ch))
	}
	return ch[0]
}

func mean(x []float64) (m float64) {
	for _, v := range x {
		m += v
	}
	return m / float64(len(x))
}

func rmsOf(x []float64) float64 {
	var s float64
	for _, v := range x {
		s += v * v
	}
	return math.Sqrt(s / float64(len(x)))
}

func TestNoise(t *testing.T) {
	Seed(5)
	n := renderMono(t, Noise(), 44100, 3)
	lo, hi := 2.0, -2.0
	for _, v := range n {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	// evenly spread from -1 to 1: mean 0, rms 1/sqrt(3)
	if math.Abs(mean(n)) > 0.01 || math.Abs(rmsOf(n)-1/math.Sqrt(3)) > 0.01 || lo < -1 || hi >= 1 || lo > -0.99 || hi < 0.99 {
		t.Errorf("mean %v rms %v (want 0.577) range %v..%v", mean(n), rmsOf(n), lo, hi)
	}
	// two noises are two noises: not correlated, and not merged by the optimizer
	Seed(5)
	two := Patch{SampleRate: 44100}.Start(Chans(Noise(), Noise()))
	a, b := make([]float64, 20000), make([]float64, 20000)
	two.Process(a, b)
	var corr float64
	for i := range a {
		corr += a[i] * b[i]
	}
	if c := corr / float64(len(a)) / (1.0 / 3); math.Abs(c) > 0.03 {
		t.Errorf("two noises are correlated: %v", c)
	}
	// seeded: the same every time
	Seed(9)
	x := renderMono(t, Noise(), 1000, 1)
	Seed(9)
	y := renderMono(t, Noise(), 1000, 1)
	for i := range x {
		if x[i] != y[i] {
			t.Fatal("the same seed gives another noise")
		}
	}
	// and compiling the same noise again is the same noise: its state is made when the program is built
	nz := Noise()
	if u, v := renderMono(t, nz, 1000, 1), renderMono(t, nz, 1000, 1); u[10] != v[10] || u[999] != v[999] {
		t.Error("one noise compiled twice is not the same")
	}
}

func TestLFNoise(t *testing.T) {
	Seed(1)
	// without interpolation it steps: freq values a second
	stepped := renderMono(t, LFNoise(10.0, false), 1000, 4)
	changes := 0
	for i := 1; i < len(stepped); i++ {
		if stepped[i] != stepped[i-1] {
			changes++
		}
	}
	if changes < 38 || changes > 41 { // 10 a second for 4 seconds
		t.Errorf("%d steps in 4 seconds at 10 Hz", changes)
	}
	// with interpolation it moves in lines: no step is bigger than the line it is on (2 over 100 samples at most)
	smooth := renderMono(t, LFNoise(10.0, true), 1000, 4)
	var biggest float64
	for i := 1; i < len(smooth); i++ {
		biggest = math.Max(biggest, math.Abs(smooth[i]-smooth[i-1]))
	}
	if biggest > 2.0/100+1e-9 {
		t.Errorf("a step of %v between samples", biggest)
	}
	if rmsOf(smooth) < 0.2 {
		t.Errorf("rms %v", rmsOf(smooth))
	}
	// every channel has noise of its own
	_, ch := Patch{SampleRate: 1000}.Render(LFNoise(Chans(N(10), N(10)), true), 2)
	if maxAbsDiff(ch[0], ch[1]) < 0.1 {
		t.Error("two channels of lfnoise are the same noise")
	}
}

func maxAbsDiff(a, b []float64) (d float64) {
	for i := range a {
		d = math.Max(d, math.Abs(a[i]-b[i]))
	}
	return
}

func TestImpulse(t *testing.T) {
	x := renderMono(t, Impulse(10.0), 1000, 1)
	var at []int
	for i, v := range x {
		if v != 0 && v != 1 {
			t.Fatalf("sample %d is %v", i, v)
		}
		if v == 1 {
			at = append(at, i)
		}
	}
	// an impulse at the start, and then one every 100 samples
	if len(at) != 10 || at[0] != 0 {
		t.Fatalf("impulses at %v", at)
	}
	for i := 1; i < len(at); i++ {
		if at[i]-at[i-1] != 100 {
			t.Errorf("impulses %d and %d are %d apart", i-1, i, at[i]-at[i-1])
		}
	}
	// a frequency that is a signal moves the next impulse
	fast := renderMono(t, Impulse(Ramp(1.0).Range(10, 50)), 1000, 1)
	var count int
	for _, v := range fast {
		count += int(v)
	}
	if count < 20 || count > 40 {
		t.Errorf("%d impulses from a sweep of 10 to 50 Hz", count)
	}
}

func TestDust(t *testing.T) {
	Seed(3)
	x := renderMono(t, Dust(100.0), 10000, 20)
	var count int
	var sum float64
	for _, v := range x {
		if v < 0 || v > 1 {
			t.Fatalf("a value of %v", v)
		}
		if v != 0 {
			count++
			sum += v
		}
	}
	// 100 a second on average for 20 seconds: 2000, with a spread of sqrt(2000) = 45
	if count < 2000-5*45 || count > 2000+5*45 {
		t.Errorf("%d impulses, want about 2000", count)
	}
	if m := sum / float64(count); math.Abs(m-0.5) > 0.03 {
		t.Errorf("the values average %v, want 0.5", m)
	}
}

// Decay falls 60 dB in the time it is given
func TestDecay(t *testing.T) {
	x := renderMono(t, Decay(Impulse(0.5), 0.4), 1000, 1) // an impulse at 0 and at 2 s: only the first in 1 s
	if x[0] != 1 {
		t.Fatalf("starts at %v", x[0])
	}
	if got := x[400]; math.Abs(got-0.001) > 1e-9 { // 0.4 s later: a thousandth
		t.Errorf("after 0.4 s: %v, want 0.001", got)
	}
	if got := x[200]; math.Abs(got-math.Sqrt(0.001)) > 1e-9 { // half the time: half the dB
		t.Errorf("after 0.2 s: %v, want %v", got, math.Sqrt(0.001))
	}
	// impulses add up on what is left
	y := renderMono(t, Decay(Impulse(10.0), 0.4), 1000, 0.5)
	if y[100] <= x[100] {
		t.Errorf("a second impulse does not add: %v", y[100])
	}
	// a time of zero is no decay at all
	z := renderMono(t, Decay(Impulse(10.0), 0.0), 1000, 0.3)
	if z[0] != 1 || z[1] != 0 || z[100] != 1 {
		t.Errorf("decay of 0: %v %v %v", z[0], z[1], z[100])
	}
}

func TestDecay2(t *testing.T) {
	const sr, att, dec = 10000.0, 0.01, 0.3
	x := renderMono(t, Decay2(Impulse(1.0), att, dec), sr, 1)
	if x[0] != 0 {
		t.Errorf("starts at %v, the two decays start together", x[0])
	}
	var peakAt int
	var peak float64
	for i, v := range x {
		if v > peak {
			peak, peakAt = v, i
		}
	}
	// the difference of two exponentials, e^(-t/ta) and e^(-t/td) with a 60 dB time of att and dec
	ta, td := att/math.Log(1000), dec/math.Log(1000)
	want := ta * td / (td - ta) * math.Log(td/ta) // the time of the top
	if got := float64(peakAt) / sr; math.Abs(got-want) > 2/sr {
		t.Errorf("the top is at %v s, want %v", got, want)
	}
	// and it falls after: 60 dB in dec, as the decay does
	if x[int(0.3*sr)] > 0.003 || x[int(0.3*sr)] < 0.0009 {
		t.Errorf("after the decay time the level is %v", x[int(0.3*sr)])
	}
}

// Ramp is LFSaw: it starts at 0, rises to 1, and jumps to -1
func TestRamp(t *testing.T) {
	x := renderMono(t, Ramp(1.0), 100, 1)
	if x[0] != 0 || math.Abs(x[1]-0.02) > 1e-12 || math.Abs(x[49]-0.98) > 1e-12 {
		t.Errorf("start: %v %v %v", x[0], x[1], x[49])
	}
	if math.Abs(x[50]+1) > 1e-12 || math.Abs(x[51]+0.98) > 1e-12 { // 1.0 jumps to -1
		t.Errorf("the jump: %v %v", x[50], x[51])
	}
	if math.Abs(x[99]-(-0.02)) > 1e-12 || math.Abs(x[100%len(x)]-0) > 1e-12 {
		t.Errorf("the end of the period: %v %v", x[99], x[0])
	}
	// a negative frequency runs the other way
	y := renderMono(t, Ramp(-1.0), 100, 0.1)
	if y[0] != 0 || math.Abs(y[1]+0.02) > 1e-12 {
		t.Errorf("backwards: %v %v", y[0], y[1])
	}
}

func TestHz(t *testing.T) {
	for note, want := range map[float64]float64{69: 440, 57: 220, 81: 880, 60: 261.6255653005986, 0: 8.175798915643707} {
		got := renderMono(t, Hz(note), 100, 0.05)[2]
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("note %v: %v Hz, want %v", note, got, want)
		}
	}
	// a sweep of notes is a sweep of frequencies, per sample
	sweep := renderMono(t, Hz(Ramp(1.0).Range(57, 81)), 100, 1)
	if math.Abs(sweep[50]-220) > 1e-9 { // the middle of the sweep, at the ramp's 0
		t.Errorf("middle of the sweep: %v", sweep[0])
	}
	if got := Hz(Chans(N(57), N(69))).Channels(); got != 2 {
		t.Errorf("%d channels", got)
	}
}

func TestLines(t *testing.T) {
	x := renderMono(t, Line(1, 50, 2), 100, 3)
	if x[0] != 1 || math.Abs(x[100]-25.5) > 1e-9 || math.Abs(x[200]-50) > 1e-9 || x[299] != 50 { // halfway: the middle; then it stays
		t.Errorf("line: %v %v %v %v", x[0], x[100], x[200], x[299])
	}
	y := renderMono(t, XLine(1, 100, 2), 100, 3)
	if y[0] != 1 || math.Abs(y[100]-10) > 1e-9 || y[200] != 100 || y[299] != 100 { // halfway: the geometric mean
		t.Errorf("xline: %v %v %v %v", y[0], y[100], y[200], y[299])
	}
	if msg := func() (m any) { defer func() { m = recover() }(); XLine(0, 1, 1); return }(); msg == nil {
		t.Error("XLine from 0 does not panic")
	}
}

func TestExpRange(t *testing.T) {
	// the lfo of a ramp from -1 to 1: the middle is the geometric mean of the ends, the ends are the ends
	z := renderMono(t, Ramp(1.0).ExpRange(100, 10000), 100, 1)
	if math.Abs(z[0]-1000) > 1e-9 { // the ramp starts at 0, in the middle
		t.Errorf("the middle: %v, want the geometric mean 1000", z[0])
	}
	if math.Abs(z[50]-100) > 1e-9 || math.Abs(z[25]-3162.2776601683795) > 1e-6 { // at -1 (after the jump), and a quarter: 3/4 of the way up the curve
		t.Errorf("the low end %v, a quarter %v", z[50], z[25])
	}
}

// Pulse is SuperCollider's: +1 for the width, -1 for the rest, so the mean is 2*width-1
func TestPulse(t *testing.T) {
	const sr = 44100.0
	for _, w := range []float64{0.1, 0.3, 0.5, 0.8} {
		x := renderMono(t, Pulse(100.0, w), sr, 2)
		var up int
		lo, hi := 2.0, -2.0
		for _, v := range x {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
			if v > 0 {
				up++
			}
		}
		if frac := float64(up) / float64(len(x)); math.Abs(frac-w) > 0.01 {
			t.Errorf("width %v: it is up %v of the time", w, frac)
		}
		if m := mean(x); math.Abs(m-(2*w-1)) > 0.01 {
			t.Errorf("width %v: the mean is %v, want %v", w, m, 2*w-1)
		}
		// between -1 and 1, but for the little overshoot of the edges that are smoothed (a Gibbs ear)
		if lo < -1.2 || hi > 1.2 || lo > -0.99 || hi < 0.99 {
			t.Errorf("width %v: range %v .. %v", w, lo, hi)
		}
		// the level of a wave of -1 and 1: rms of 1 (a little under for the smoothed edges)
		if r := rmsOf(x); r < 0.97 || r > 1.01 {
			t.Errorf("width %v: rms %v", w, r)
		}
	}
	// the pitch: 100 rises a second
	x := renderMono(t, Pulse(100.0, 0.3), sr, 1)
	rises := 0
	for i := 1; i < len(x); i++ {
		if x[i-1] < 0 && x[i] >= 0 {
			rises++
		}
	}
	if rises < 99 || rises > 101 {
		t.Errorf("%d periods in a second at 100 Hz", rises)
	}
	// width 0 and 1 are a constant, the edges cancel
	for w, want := range map[float64]float64{0: -1, 1: 1, -0.5: -1, 2: 1} {
		y := renderMono(t, Pulse(440.0, w), sr, 0.05)
		if maxAbsDiff(y, constant(len(y), want)) > 1e-9 {
			t.Errorf("width %v is not the constant %v", w, want)
		}
	}
	// a width that is a signal moves the edge: the ramp (0 at the start, 1 at half a second, then -1) as a width
	// of 0.5 + 0.4 * ramp makes the mean 0.8 * ramp: +0.4 a quarter of the way, -0.4 three quarters of the way
	pwm := renderMono(t, Pulse(100.0, Ramp(1.0).Range(0.1, 0.9)), sr, 1)
	around := func(at float64) float64 { i := int(at * sr); return mean(pwm[i-2205 : i+2205]) }
	if q1, q3 := around(0.25), around(0.75); math.Abs(q1-0.4) > 0.1 || math.Abs(q3+0.4) > 0.1 {
		t.Errorf("the mean is %v at 0.25 s and %v at 0.75 s, want 0.4 and -0.4", q1, q3)
	}
	// band limited: a pulse at a high note has much less above the Nyquist frequency folded back than a naive one
	const f = 7919.0
	band := renderMono(t, Pulse(f, 0.3), sr, 0.5)
	naive := make([]float64, len(band))
	var ph float64
	for i := range naive {
		naive[i] = -1
		if ph < 0.3 {
			naive[i] = 1
		}
		ph += f / sr
		ph -= math.Floor(ph)
	}
	hf := func(x []float64) (e float64) { // the energy of the first difference: what is close to Nyquist
		for i := 1; i < len(x); i++ {
			d := x[i] - x[i-1]
			e += d * d
		}
		return
	}
	if hf(band) >= hf(naive) {
		t.Errorf("not smoother than a naive pulse: %v against %v", hf(band), hf(naive))
	}
}

func constant(n int, v float64) []float64 {
	x := make([]float64, n)
	for i := range x {
		x[i] = v
	}
	return x
}

// LinExp as SuperCollider 3.14.1 does it (measured): 100..1000 gives 316.23 for an input of 0, 3162.3 for an input of 2 (no clipping)
func TestLinExp(t *testing.T) {
	ramp := Ramp(0.5).Mul(N(2)) // 0 at the start, rising to 2 at 1 s: past the range -1..1
	x := renderMono(t, LinExp(ramp, -1, 1, 100.0, 1000.0), 1000, 2)
	if math.Abs(x[0]-316.2277660168379) > 1e-6 { // what SuperCollider gives for an input of 0
		t.Errorf("at the start %v, want the geometric mean 316.23", x[0])
	}
	// the formula, at any point, with no limit at the ends: 100 * 10^((input+1)/2)
	for _, i := range []int{0, 125, 250, 499, 750, 1000, 1499} {
		in := renderMono(t, ramp, 1000, 2)[i]
		want := 100 * math.Pow(10, (in+1)/2)
		if math.Abs(x[i]-want) > want*1e-9 {
			t.Errorf("sample %d: input %v gives %v, want %v", i, in, x[i], want)
		}
	}
	if mx := maxOf(x); mx < 3000 { // the input goes to 2: 3162
		t.Errorf("the output is clipped at %v: SuperCollider's goes on to 3162", mx)
	}
	// a range that goes down, ranges that are signals, channels
	down := renderMono(t, LinExp(N(0), -1, 1, 1000.0, 100.0), 100, 0.1)
	near(t, "reversed range", down[3], 316.2277660168379, 1e-6)
	sigs := renderMono(t, LinExp(N(0), -1, 1, N(10), Mul(N(10), 4.0)), 100, 0.1)
	near(t, "signal ranges", sigs[3], 20, 1e-9) // the geometric middle of 10 and 40
	if got := LinExp(Chans(N(0), N(1)), -1, 1, 1.0, 100.0).Channels(); got != 2 {
		t.Errorf("%d channels", got)
	}
	zero := renderMono(t, LinExp(N(0), -1, 1, 0.0, 100.0), 100, 0.1)
	if zero[3] != 0 {
		t.Errorf("a range from 0: %v", zero[3])
	}
}

func maxOf(x []float64) (m float64) {
	m = math.Inf(-1)
	for _, v := range x {
		m = math.Max(m, v)
	}
	return
}

func TestClip2ProductReverse(t *testing.T) {
	x := renderMono(t, Clip2(Ramp(0.5).Mul(N(3)), 1.0), 1000, 2) // a ramp of 3 clipped at 1
	if minOf(x) != -1 || maxOf(x) != 1 || math.Abs(x[100]-0.3) > 1e-9 {
		t.Errorf("clip2: %v .. %v, at 0.1 s %v (0.3 is not clipped)", minOf(x), maxOf(x), x[100])
	}
	// the level can be a signal and a negative level is the same as the positive one
	y := renderMono(t, Clip2(N(5), N(-2)), 100, 0.1)
	if y[3] != 2 {
		t.Errorf("a level of -2: %v", y[3])
	}
	_, ch := Patch{SampleRate: 1000}.Render(Product(Chans(Mul(Ramp(1.0), 0.5), N(0.5), N(4))), 0.5)
	if len(ch) != 1 || math.Abs(ch[0][100]-0.5*(2*0.1+1-1)*0.5*4) > 1e-9 { // the ramp is 0.2 at 0.1 s
		t.Errorf("product: %d channels, %v", len(ch), ch[0][100])
	}
	if Product(N(3)).Channels() != 1 {
		t.Error("the product of one channel")
	}
	_, r := Patch{SampleRate: 100}.Render(Reverse(Chans(N(0.1), N(0.2), N(0.3))), 0.1)
	if len(r) != 3 || r[0][2] != 0.3 || r[1][2] != 0.2 || r[2][2] != 0.1 {
		t.Errorf("reverse: %v %v %v", r[0][2], r[1][2], r[2][2])
	}
}

func minOf(x []float64) (m float64) {
	m = math.Inf(1)
	for _, v := range x {
		m = math.Min(m, v)
	}
	return
}

// SCSaw against SuperCollider 3.14.1 (Saw.ar(440) at 44100 Hz: rms 0.28966, peak 0.6637, mean 0.00503, to the sample), and the ringing
// that a fast sweep makes in SuperCollider (Saw.ar(LFSaw.kr(0.5).exprange(6, 11000)): peak 1.81)
func TestSCSaw(t *testing.T) {
	_, ch := Patch{SampleRate: 44100}.Render(SCSaw(440.0), 4)
	x := ch[0]
	var sum, sq, peak float64
	for _, v := range x {
		sum, sq, peak = sum+v, sq+v*v, math.Max(peak, math.Abs(v))
	}
	n := float64(len(x))
	near(t, "rms", math.Sqrt(sq/n), 0.28966, 1e-4)
	near(t, "peak", peak, 0.6637, 1e-4)
	near(t, "mean", sum/n, 0.00503, 1e-4)
	_, ch = Patch{SampleRate: 44100}.Render(SCSaw(Ramp(0.5).Mul(N(1.0)).ExpRange(6, 11000)), 4)
	peak = 0
	for _, v := range ch[0] {
		peak = math.Max(peak, math.Abs(v))
	}
	if peak < 1.5 || peak > 2.2 {
		t.Errorf("a sweep gives a peak of %v, SuperCollider's is 1.81", peak)
	}
	if got := SCSaw(Chans(N(100), N(200))).Channels(); got != 2 {
		t.Errorf("%d channels", got)
	}
}

func TestLFNoiseControl(t *testing.T) {
	for _, interp := range []bool{true, false} {
		_, ch := Patch{SampleRate: 6400}.Render(LFNoiseControl(5.0, interp), 4) // blocks of 64 samples: 100 a second
		x := ch[0]
		for i := range x {
			if i%64 != 0 && x[i] != x[i-1] {
				t.Fatalf("interp %v: sample %d changes inside a block", interp, i)
			}
		}
		if interp {
			// in straight lines from block to block: the second difference of the held values is 0 except at the new values
			// (5 a second: every 20 blocks)
			bad := 0
			for b := 2; b < len(x)/64; b++ {
				if d := (x[b*64] - x[(b-1)*64]) - (x[(b-1)*64] - x[(b-2)*64]); math.Abs(d) > 1e-9 {
					bad++
				}
			}
			if bad > 2*4*5 { // at most two blocks at each of the 20 new values
				t.Errorf("not straight lines: %d bends", bad)
			}
		} else {
			changes := 0
			for i := 1; i < len(x); i++ {
				if x[i] != x[i-1] {
					changes++
				}
			}
			if changes < 15 || changes > 25 { // 5 a second for 4 s
				t.Errorf("steps: %d changes, want about 20", changes)
			}
		}
		if mx := maxOf(x); mx >= 1 || minOf(x) < -1 {
			t.Errorf("range %v", mx)
		}
	}
	if LFNoiseControl(Chans(N(1), N(2)), true).Channels() != 2 {
		t.Error("channels")
	}
}

func TestRampControl(t *testing.T) {
	_, ch := Patch{SampleRate: 6400}.Render(RampControl(1.0), 2) // 100 blocks a second: 100 steps of 0.02 for a ramp of 1 Hz
	x := ch[0]
	for i := range x {
		if i%64 != 0 && x[i] != x[i-1] {
			t.Fatalf("sample %d changes inside a block", i)
		}
	}
	near(t, "start", x[0], 0, 1e-12)
	near(t, "block 10", x[640], 0.2, 1e-9)
	if x[50*64] < -1 || x[50*64] > -0.9 { // 0 -> 1 in half a second, then -1 and up
		t.Errorf("after the jump: %v", x[50*64])
	}
	if got := RampControl(Chans(N(1), N(2))).Channels(); got != 2 {
		t.Error("channels")
	}
}
