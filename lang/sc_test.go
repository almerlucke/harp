package lang

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The examples in examples/supercollider are patches that were written for SuperCollider, written again in
// harp. SuperCollider is not here to listen to, so what is checked is what those patches are known to do:
// the formulas of the UGens they use, from their documentation, and the sound that follows from them.

const scSR = 44100.0

func scSource(t *testing.T, name string) string {
	t.Helper()
	src, err := os.ReadFile("examples/supercollider/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}

func scRender(t *testing.T, src string, seconds float64) [][]float64 {
	t.Helper()
	p := mustBuild(t, src)
	_, ch := p.Settings().Render(p.Out, seconds)
	return ch
}

func goertzel(x []float64, hz float64) float64 {
	w := 2 * math.Pi * hz / scSR
	c := 2 * math.Cos(w)
	var s1, s2 float64
	for _, v := range x {
		s0 := v + c*s1 - s2
		s2, s1 = s1, s0
	}
	return (s1*s1 + s2*s2 - c*s1*s2) / float64(len(x)*len(x))
}

func rmsOf(x []float64) float64 {
	var s float64
	for _, v := range x {
		s += v * v
	}
	return math.Sqrt(s / float64(len(x)))
}

// 1: SinOsc.ar(SinOsc.kr(5, 0, 10, 440), 0, 0.2) ! 2
func TestSC1Vibrato(t *testing.T) {
	ch := scRender(t, scSource(t, "1_vibrato.harp"), 2)
	if len(ch) != 2 || maxDiff(ch[0], ch[1]) != 0 {
		t.Fatalf("%d channels, and ! 2 is two of the same", len(ch))
	}
	// the sine of a frequency of 440 + 10 * the sine of 5 Hz, the phase being what the frequency adds up to
	var phase, lfoPhase, worst float64
	for _, got := range ch[0] {
		want := 0.2 * math.Sin(2*math.Pi*phase)
		worst = math.Max(worst, math.Abs(got-want))
		phase += (440 + 10*math.Sin(2*math.Pi*lfoPhase)) / scSR
		lfoPhase += 5 / scSR
	}
	if worst > 1e-9 {
		t.Errorf("differs from the formula by %g", worst)
	}
	if p := peak(ch[0]); math.Abs(p-0.2) > 1e-3 {
		t.Errorf("peak %v, the mul of 0.2", p)
	}
}

// 2: RLPF.ar(Saw.ar([100, 101], 0.1), SinOsc.kr(0.2).range(300, 3000), 0.2)
func TestSC2RLPFSaws(t *testing.T) {
	ch := scRender(t, scSource(t, "2_rlpf_saws.harp"), 5)
	if len(ch) != 2 {
		t.Fatalf("%d channels", len(ch))
	}
	// the cutoff of the lfo is 3000 at 1.25 s and 300 at 3.75 s: the sound is much duller at the low point.
	// (Not quieter: a resonance of Q 5 raises the harmonic that is at the cutoff, so the level is not what
	// tells. The brightness is: the size of the steps between samples is what the high frequencies make)
	window := func(c []float64, at float64) []float64 { return c[int(at*scSR):int((at+0.1)*scSR)] }
	steps := func(x []float64) float64 {
		d := make([]float64, len(x)-1)
		for i := range d {
			d[i] = x[i+1] - x[i]
		}
		return rmsOf(d)
	}
	for c := 0; c < 2; c++ {
		bright, dull := steps(window(ch[c], 1.2)), steps(window(ch[c], 3.7))
		if dull > bright*0.5 { // clearly duller: less than half of the high frequency content
			t.Errorf("channel %d: the steps are %v with the cutoff at 300 Hz and %v at 3000 Hz: not a sweep", c, dull, bright)
		}
		// the energy at 5 kHz is far below that of the fundamental when the cutoff is low
		if goertzel(window(ch[c], 3.7), 5000) > goertzel(window(ch[c], 3.7), 100)/1000 {
			t.Errorf("channel %d: there is much above the cutoff", c)
		}
	}
	// the saw of SuperCollider swings over 0.05 with this mul (half of what 0.1 says): a resonance does not make it a lot louder
	if p := peak(ch[0]); p < 0.03 || p > 0.3 {
		t.Errorf("peak %v", p)
	}
	if maxDiff(ch[0], ch[1]) < 1e-3 {
		t.Error("100 and 101 Hz are the same sound")
	}
}

// 3: analog bubbles, by McCartney
func TestSC3AnalogBubbles(t *testing.T) {
	src := scSource(t, "3_analog_bubbles.harp")
	// the frequencies, f: LFSaw.kr(0.4, 0, 24, LFSaw.kr([8, 7.23], 0, 3, 80)).midicps. LFSaw starts at 0, rises
	// to 1 and jumps to -1: its value at sample n is ((2 f n / sr + 1) mod 2) - 1
	lfsaw := func(f float64, n int) float64 { return math.Mod(2*f*float64(n)/scSR+1, 2) - 1 }
	fOnly := strings.Replace(src, "out = sine(f).gain(0.04).comb(delay: 0.2, decay: 4)", "out = f", 1)
	f := scRender(t, fOnly, 3)
	if len(f) != 2 {
		t.Fatalf("[8, 7.23] gives %d channels", len(f))
	}
	for c, rate := range []float64{8, 7.23} {
		var worst float64
		for n := 0; n < len(f[c]); n += 7 {
			note := lfsaw(0.4, n)*24 + lfsaw(rate, n)*3 + 80
			want := 440 * math.Exp2((note-69)/12)
			worst = math.Max(worst, math.Abs(f[c][n]-want)/want)
		}
		if worst > 1e-6 {
			t.Errorf("channel %d: the frequency differs from the formula by %g (relative)", c, worst)
		}
	}
	// at the start both are note 80: 830.6 Hz
	if math.Abs(f[0][0]-440*math.Exp2(11.0/12)) > 1e-9 {
		t.Errorf("starts at %v Hz", f[0][0])
	}

	ch := scRender(t, src, 6)
	// CombN with a delay of 0.2: only the delayed sound is heard, so there is nothing for 0.2 s
	for i := 0; i < int(0.2*scSR)-1; i++ {
		if math.Abs(ch[0][i]) > 1e-12 {
			t.Fatalf("sound at sample %d, before the delay of 0.2 s", i)
		}
	}
	// and then the sine of 0.04 and its echoes that ring for 4 s (60 dB): at most 0.04 / (1 - 0.708) in the end
	if p := peak(ch[0]); p < 0.04 || p > 0.04/(1-math.Pow(0.001, 0.2/4))+1e-9 {
		t.Errorf("peak %v, want between 0.04 and %v", p, 0.04/(1-math.Pow(0.001, 0.2/4)))
	}
	if maxDiff(ch[0], ch[1]) < 1e-3 {
		t.Error("8 and 7.23 are the same bubbles")
	}
}

// 4: Decay2.ar(Impulse.ar(XLine.kr(1, 50, 20), 0.25), 0.01, 0.2, FSinOsc.ar(600))
func TestSC4Decay2Pings(t *testing.T) {
	src := scSource(t, "4_decay2_pings.harp")
	// the impulses speed up from 1 a second to 50 a second in 20 s: the number of them is the integral of
	// 50^(t/20), from 0 to t: 20 (50^(t/20) - 1) / ln 50
	count := func(a, b float64) int {
		pulses := scRender(t, "out = impulse(xline(1, 50, 20))", 20)[0]
		n := 0
		for _, v := range pulses[int(a*scSR):int(b*scSR)] {
			n += int(v)
		}
		return n
	}
	total := func(t float64) float64 { return 20 * (math.Pow(50, t/20) - 1) / math.Log(50) }
	for _, w := range [][2]float64{{0, 5}, {5, 10}, {10, 15}, {15, 20}} {
		want := total(w[1]) - total(w[0])
		if got := count(w[0], w[1]); math.Abs(float64(got)-want) > 2 {
			t.Errorf("between %v s and %v s: %d impulses, want about %.1f", w[0], w[1], got, want)
		}
	}
	// a ping: the first of them, alone in the first half second. It rises to the top of the difference of two
	// exponentials, at the time and the height that come from the decay times of 60 dB
	ping := scRender(t, strings.Replace(src, "* sine(600)", "", 1), 0.5)[0]
	ta, td := 0.01/math.Log(1000), 0.2/math.Log(1000)
	tTop := ta * td / (td - ta) * math.Log(td/ta)
	hTop := math.Exp(-tTop/td) - math.Exp(-tTop/ta)
	var top float64
	var at int
	for i, v := range ping {
		if v > top {
			top, at = v, i
		}
	}
	if math.Abs(top-hTop) > 5e-3 || math.Abs(float64(at)/scSR-tTop) > 2/scSR {
		t.Errorf("the top is %v at %v s, want %v at %v s", top, float64(at)/scSR, hTop, tTop)
	}
	// with the sine of 600 Hz it is that sound: the energy is at 600 Hz
	full := scRender(t, src, 0.3)[0]
	if goertzel(full, 600) < 100*goertzel(full, 1500) {
		t.Error("the energy is not at 600 Hz")
	}
}

// 5: Resonz.ar(WhiteNoise.ar(0.5), LFNoise1.kr(1).expRange(200, 5000), 0.05)
func TestSC5ResonantNoise(t *testing.T) {
	src := scSource(t, "5_resonant_noise.harp")
	a, b := scRender(t, src, 4), scRender(t, src, 4)
	if maxDiff(a[0], b[0]) != 0 || maxDiff(a[0], a[1]) != 0 {
		t.Error("seeded noise is not the same again, or the two channels differ")
	}
	if r := rmsOf(a[0]); r < 0.001 || r > 0.3 {
		t.Errorf("rms %v", r)
	}
	// with the center held at 1000 Hz the noise is a narrow band there: the energy at 1000 Hz is far above
	// that an octave on either side, and the band has a gain of 0 dB at the center
	held := strings.Replace(src, "lfNoise(1).expRange(200, 5000)", "1000", 1)
	n := scRender(t, held, 4)[0][int(0.5*scSR):]
	center, below, above := goertzel(n, 1000), goertzel(n, 500), goertzel(n, 2000)
	if center < 30*below || center < 30*above {
		t.Errorf("power at 1000 Hz %g, at 500 Hz %g, at 2000 Hz %g: not a narrow band", center, below, above)
	}
	white := scRender(t, "seed = 1\nout = noise().gain(0.5)", 4)[0][int(0.5*scSR):]
	want := goertzel(white, 1000) // what the 0.5 of white noise has there, with a unit gain at the center, a tone of that power
	if ratio := center / want; ratio < 0.1 || ratio > 30 {
		t.Errorf("at 1000 Hz the filter makes the power %vx that of the noise", ratio)
	}
}

// 6: Pan2.ar(Pulse.ar(100, 0.3, 0.1), LFTri.kr(0.2))
func TestSC6PanPulse(t *testing.T) {
	src := scSource(t, "6_pan_pulse.harp")
	ch := scRender(t, src, 5)
	tri := scRender(t, "out = triangle(0.2)", 5)[0]
	mono := scRender(t, "out = (pulse(100, width: 0.3) + (1 - 2 * 0.3)) * -0.05", 5)[0]
	const win = 2205 // 0.05 s: five periods of the pulse
	power := func(x []float64, at int) float64 {
		var s float64
		for _, v := range x[at : at+win] {
			s += v * v
		}
		return s / win
	}
	for _, at := range []float64{0.2, 0.7, 1.25, 2.0, 2.6, 3.75, 4.4} {
		i := int(at * scSR)
		pl, pr, pm := power(ch[0], i), power(ch[1], i), power(mono, i)
		// the power is that of the pulse, wherever it is: equal power
		if math.Abs((pl+pr)/pm-1) > 0.05 {
			t.Errorf("at %v s: power %v + %v is not that of the pulse %v", at, pl, pr, pm)
		}
		// and the split follows the law: right / left = tan^2 of the angle (pos + 1) pi / 4
		pos := math.Max(-1, math.Min(1, tri[i+win/2]))
		theta := (pos + 1) * math.Pi / 4
		if want := math.Tan(theta) * math.Tan(theta); pl > 1e-9 && math.Abs(pr/pl/want-1) > 0.35 {
			t.Errorf("at %v s the triangle is at %.2f: right/left power %v, the law gives %v", at, pos, pr/pl, want)
		}
	}
}

// 7: (SinOsc.ar(1200) * Decay2.ar(Dust.ar(4), 0.005, 0.15) * 0.3) ! 2
func TestSC7DustPings(t *testing.T) {
	src := scSource(t, "7_dust_pings.harp")
	a, b := scRender(t, src, 20), scRender(t, src, 20)
	if maxDiff(a[0], b[0]) != 0 || maxDiff(a[0], a[1]) != 0 {
		t.Error("seeded dust is not the same again, or the channels differ")
	}
	// four a second on average for 20 s: 80 events, with a spread of 9. The first source of random numbers
	// of a seeded patch is the same one in the patch of the dust alone
	events := 0
	for _, v := range scRender(t, "seed = 4\nout = dust(4)", 20)[0] {
		if v != 0 {
			events++
		}
	}
	if events < 80-5*9 || events > 80+5*9 {
		t.Errorf("%d events in 20 seconds, want about 80", events)
	}
	// pings can ring over each other, so the top can be more than the 0.3 of one: but not much more
	if p := peak(a[0]); p < 0.05 || p > 0.6 {
		t.Errorf("peak %v", p)
	}
	if goertzel(a[0], 1200) < 50*goertzel(a[0], 700) {
		t.Error("the energy is not at 1200 Hz")
	}
}

// the SuperCollider patches and the others are all patches: they compile, and sound at levels that are right
func TestSCExamplesAreAllThere(t *testing.T) {
	entries, err := os.ReadDir("examples/supercollider")
	if err != nil || len(entries) != 12 {
		t.Fatalf("%d examples: %v", len(entries), err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".harp" { // the script that makes the reference of the gabber is one
			continue
		}
		ch := scRender(t, scSource(t, e.Name()), 3)
		for c := range ch {
			if pk := peak(ch[c]); pk < 0.005 || pk > 1 || math.IsNaN(pk) {
				t.Errorf("%s: channel %d peaks at %v", e.Name(), c, pk)
			}
		}
	}
}

// What SuperCollider 3.14.1 does, measured by running it (see examples/supercollider/6_pan_pulse.harp): Pulse.ar(100, 0.3, 0.1)
// has a mean of 0 (+0.00005), an rms of 0.0459, is positive for the long part (0.03) and negative for the short one
// (-0.07). The pulse of harp is not that, and the translation says how to get there
func TestSCPulseAsMeasuredInSuperCollider(t *testing.T) {
	const scMean, scRMS = 0.00005, 0.0459 // measured with sclang and scsynth 3.14.1, Pulse.ar(100, 0.3, 0.1), 2 s
	plain := scRender(t, "out = pulse(100, width: 0.3).gain(0.1)", 2)[0]
	if m := meanOf(plain); math.Abs(m-(-0.04)) > 0.002 || math.Abs(rmsOf(plain)-0.0998) > 0.002 {
		t.Errorf("the pulse of harp: mean %v and rms %v, want -0.04 and 0.0998 (between -1 and 1, a DC offset)", m, rmsOf(plain))
	}
	sc := scRender(t, "out = (pulse(100, width: 0.3) + (1 - 2 * 0.3)) * -0.05", 2)[0]
	if m := meanOf(sc); math.Abs(m-scMean) > 5e-4 {
		t.Errorf("the translation has a mean of %v, SuperCollider's has %v", m, scMean)
	}
	if r := rmsOf(sc); math.Abs(r/scRMS-1) > 0.03 {
		t.Errorf("the translation has an rms of %v, SuperCollider's has %v", r, scRMS)
	}
	// the short part (30%) is the negative one, the long part positive: look in the middle of both, over many periods
	var short, long float64
	for p := 20; p < 120; p++ {
		short += sc[p*441+66]
		long += sc[p*441+264]
	}
	if short/100 > -0.06 || long/100 < 0.025 {
		t.Errorf("the short part is at %v and the long part at %v, want about -0.07 and +0.03", short/100, long/100)
	}
	// psalm's rectangle is the pulse with a mean of 0, in its own size
	r := scRender(t, "out = rectangle(100, width: 0.3).gain(0.1)", 2)[0]
	if m := meanOf(r); math.Abs(m) > 1e-3 {
		t.Errorf("the rectangle of psalm has a mean of %v, want 0", m)
	}
}

func minOf(x []float64) (m float64) {
	m = 9
	for _, v := range x {
		m = math.Min(m, v)
	}
	return
}

func maxOf(x []float64) (m float64) {
	m = -9
	for _, v := range x {
		m = math.Max(m, v)
	}
	return
}

func meanOf(x []float64) (m float64) {
	for _, v := range x {
		m += v
	}
	return m / float64(len(x))
}
