package blocks

import (
	"math"
	"strings"
	"testing"

	"harp/dsl"

	"github.com/almerlucke/psalm/blocks/oversample"
	"github.com/almerlucke/psalm/blocks/shapechain"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
)

// goertzel is the power of a signal at one frequency
func goertzel(x []float64, hz float64) float64 {
	w := 2 * math.Pi * hz / sr
	c := 2 * math.Cos(w)
	var s1, s2 float64
	for _, v := range x {
		s0 := v + c*s1 - s2
		s2, s1 = s1, s0
	}
	return (s1*s1 + s2*s2 - c*s1*s2) / float64(len(x)*len(x))
}

func crossings(x []float64) (n int) {
	for i := 1; i < len(x); i++ {
		if (x[i-1] < 0) != (x[i] < 0) {
			n++
		}
	}
	return
}

func nothing() func(re, im []float64) { return func(re, im []float64) {} }

func near(t *testing.T, what string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// With a function that does nothing the sound is what went in, size-1 samples later
func TestSpectralIsTransparent(t *testing.T) {
	for _, size := range []int{16, 256, 1024} {
		in := render(dsl.Add(dsl.Saw(220), dsl.Mul(dsl.Sine(1234.5), 0.3)), 0.2)[0]
		out := render(Spectral(dsl.Add(dsl.Saw(220), dsl.Mul(dsl.Sine(1234.5), 0.3)), size, nothing), 0.2)[0]
		var d float64
		for p := size - 1; p < len(in); p++ {
			d = math.Max(d, math.Abs(out[p]-in[p-(size-1)]))
		}
		if d > 1e-9 {
			t.Errorf("size %d: differs from the delayed input by %g", size, d)
		}
		for p := 0; p < size-1; p++ { // nothing comes out before the first frame has gone through, but the rounding of the fft
			if math.Abs(out[p]) > 1e-9 {
				t.Fatalf("size %d: sound %v at sample %d, before it went in", size, out[p], p)
			}
		}
	}
}

func TestSpectralChangesTheBins(t *testing.T) {
	const size = 1024
	cut := int(math.Round(2000 * float64(size) / sr)) // the bin of 2 kHz
	lowpass := func() func(re, im []float64) {
		return func(re, im []float64) {
			for k := cut; k < len(re); k++ {
				re[k], im[k] = 0, 0
			}
		}
	}
	low := rms(render(Spectral(dsl.Sine(500.0), size, lowpass), 0.3)[0][4410:])
	high := rms(render(Spectral(dsl.Sine(10000.0), size, lowpass), 0.3)[0][4410:])
	if math.Abs(low-1/math.Sqrt2) > 0.02 || high > 0.01 {
		t.Errorf("rms of 500 Hz %v (want 0.707), of 10 kHz %v (want 0)", low, high)
	}
}

func TestSpectralStateIsPerChannel(t *testing.T) {
	made := 0
	render(Spectral(dsl.Chans(dsl.Saw(100), dsl.Saw(200), dsl.Saw(300)), 64, func() func(re, im []float64) {
		made++
		return func(re, im []float64) {}
	}), 0.05)
	if made != 3 {
		t.Errorf("the function was made %d times for 3 channels", made)
	}
	for _, size := range []int{0, 8, 100, 1000} {
		if msg := func() (m string) {
			defer func() { m, _ = recover().(string) }()
			Spectral(dsl.N(0), size, nothing)
			return
		}(); !strings.Contains(msg, "power of 2") {
			t.Errorf("size %d: %q", size, msg)
		}
	}
}

func TestSaturate(t *testing.T) {
	dc := func(out dsl.Sig) float64 { c := render(out, 0.1)[0]; return c[len(c)-1] }
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"tanh(2*0.5)", dc(Saturate(dsl.N(0.5), 2, 1)), math.Tanh(1)},
		{"hard drive saturates", dc(Saturate(dsl.N(0.5), 50, 1)), 1},
		{"no drive is silence", dc(Saturate(dsl.N(0.5), 0, 1)), 0},
		{"drive is a signal", dc(Saturate(dsl.N(0.5), dsl.N(2), 1)), math.Tanh(1)},
		{"negative", dc(Saturate(dsl.N(-0.5), 2, 1)), -math.Tanh(1)},
		{"oversampled, settled", dc(Saturate(dsl.N(0.5), 2, 4)), math.Tanh(1)},
	} {
		if math.Abs(c.got-c.want) > 2e-3 {
			t.Errorf("%s: %v, want %v", c.name, c.got, c.want)
		}
	}
}

// A hard shaper on a high tone makes harmonics over the Nyquist frequency that fold back to a
// frequency that is not a harmonic: 3 times 9 kHz is 27 kHz, which is heard at 44.1-27 = 17.1 kHz.
// Oversampling takes it away
func TestOversamplingTakesAwayAliasing(t *testing.T) {
	tone := dsl.Mul(dsl.Sine(9000.0), 0.9)
	alias := func(factor int) float64 {
		return goertzel(render(Saturate(tone, 12, factor), 0.5)[0][4410:], 17100)
	}
	plain, over := alias(1), alias(4)
	if plain < 1e-6 || over > plain/30 {
		t.Errorf("power at 17.1 kHz: %g without oversampling, %g with x4: want at least 30 times less", plain, over)
	}
}

func TestShapeIsTheFunction(t *testing.T) {
	saw := render(dsl.Saw(100), 0.1)[0]
	got := render(Shape(dsl.Saw(100), func(x float64) float64 { return x * x }, 1), 0.1)[0]
	for i := range saw {
		if got[i] != saw[i]*saw[i] {
			t.Fatalf("sample %d: %v, want %v", i, got[i], saw[i]*saw[i])
		}
	}
	// oversampled, the function of a straight line is a line: the input, delayed by the filters
	lat := oversample.New(4, oversample.TapsPerPhase).Latency()
	in := render(dsl.Sine(300.0), 0.1)[0]
	out := render(Shape(dsl.Sine(300.0), func(x float64) float64 { return x }, 4), 0.1)[0]
	var d float64
	for p := lat + 200; p < len(in); p++ {
		d = math.Max(d, math.Abs(out[p]-in[p-lat]))
	}
	if d > 1e-3 {
		t.Errorf("x4 with the identity differs from the input delayed by %d by %g", lat, d)
	}
	// a shapechain is a shaper: its Shape method is the function
	chain := shapechain.MustParse("tanh 3")
	shaped := render(Shape(dsl.Sine(300.0), chain.Shape, 1), 0.05)[0]
	if shaped[20] <= in[20] { // tanh(3x)/tanh(3) is steeper than x on the way up
		t.Errorf("chain tanh: %v is not above %v", shaped[20], in[20])
	}
}

func TestFastSine(t *testing.T) {
	fast, exact := render(FastSine(440.0), 0.1)[0], render(dsl.Sine(440.0), 0.1)[0]
	if d := maxDiff(fast, exact); d > 1e-6 { // psalm says its table is accurate to about 3e-7
		t.Errorf("the table sine is off by %g", d)
	}
	if math.Abs(rms(fast)-1/math.Sqrt2) > 0.01 {
		t.Errorf("rms %v", rms(fast))
	}
}

func TestChainOsc(t *testing.T) {
	sin := shapechain.MustParse("sin")
	// the sin of a chain is psalm's table sine, accurate to about 3e-7
	if d := maxDiff(render(ChainOsc(sin, 440.0), 0.1)[0], render(dsl.Sine(440.0), 0.1)[0]); d > 1e-6 {
		t.Errorf("the chain sin is not a sine: differs by %g", d)
	}
	// a macro moves the wave: mult m1[1,5] | sin makes 1 cycle at 0 and 5 at 1
	chain := shapechain.MustParse("mult m1[1,5] | sin")
	one := crossings(render(ChainOsc(chain, 100.0, dsl.N(0)), 0.5)[0])
	five := crossings(render(ChainOsc(chain, 100.0, dsl.N(1)), 0.5)[0])
	if one < 95 || one > 105 || five < 475 || five > 525 {
		t.Errorf("zero crossings in half a second of 100 Hz: macro 0 gives %d (want about 100), macro 1 gives %d (about 500)", one, five)
	}
	// macros left out are 0.5: 3 cycles
	if mid := crossings(render(ChainOsc(chain, 100.0), 0.5)[0]); mid < 285 || mid > 315 {
		t.Errorf("macro left out gives %d crossings, want about 300", mid)
	}
	if msg := func() (m string) {
		defer func() { m, _ = recover().(string) }()
		ChainOsc(chain, 100.0, dsl.N(0), dsl.N(0), dsl.N(0), dsl.N(0), dsl.N(0))
		return
	}(); !strings.Contains(msg, "macros") {
		t.Errorf("too many macros: %q", msg)
	}
}

func TestCrossfade(t *testing.T) {
	dc := func(dry, wet, mix float64) float64 {
		c := render(Crossfade(dsl.N(dry), dsl.N(wet), mix), 0.01)[0]
		return c[5]
	}
	near(t, "all dry", dc(1, 7, 0), 1, 1e-3)
	near(t, "all wet", dc(7, 1, 1), 1, 1e-3)
	near(t, "halfway, dry only", dc(1, 0, 0.5), math.Cos(math.Pi/4), 2e-3)
	near(t, "halfway, wet only", dc(0, 1, 0.5), math.Sin(math.Pi/4), 2e-3)
	near(t, "halfway, both the same: as loud as the sum of the gains", dc(1, 1, 0.5), math.Sqrt2, 3e-3)
	near(t, "beyond the ends is the end", dc(1, 7, -3), 1, 1e-3)
}

func TestSmooth(t *testing.T) {
	// 1000 samples a second, a step of a second: 0 for a step, then 1
	p := dsl.Patch{SampleRate: 1000, BPM: 60, StepsPerBeat: 1}
	_, ch := p.Render(Smooth(dsl.Hold(dsl.Loop(0.0, 1.0)), 100), 2)
	s := ch[0]
	if s[0] != 0 || s[999] != 0 {
		t.Errorf("before the step: %v %v", s[0], s[999])
	}
	if math.Abs(s[1099]-(1-math.Exp(-1))) > 0.02 { // 100 ms is two thirds of the way
		t.Errorf("100 ms after the step %v, want about 0.63", s[1099])
	}
	if s[1999] < 0.99 || s[1999] > 1 {
		t.Errorf("a second after the step: %v", s[1999])
	}
	// it starts at the value it gets, not at 0
	_, ch = p.Render(Smooth(dsl.N(0.8), 100), 0.1)
	if ch[0][0] != 0.8 {
		t.Errorf("starts at %v", ch[0][0])
	}
}

func TestPitchShift(t *testing.T) {
	tone := dsl.Sine(440.0)
	tail := func(out dsl.Sig) []float64 { return render(out, 0.6)[0][int(0.3*sr):] }
	up := tail(PitchShift(tone, 12, 50))
	if up440, up880 := goertzel(up, 440), goertzel(up, 880); up880 < up440*10 {
		t.Errorf("an octave up: power at 880 Hz %g, at 440 Hz %g", up880, up440)
	}
	down := tail(PitchShift(tone, -12, 50))
	if d440, d220 := goertzel(down, 440), goertzel(down, 220); d220 < d440*10 {
		t.Errorf("an octave down: power at 220 Hz %g, at 440 Hz %g", d220, d440)
	}
	same := tail(PitchShift(tone, 0, 50))
	if g := goertzel(same, 440); g < 0.05 {
		t.Errorf("no shift: power at 440 Hz %g", g)
	}
}

func TestMergingOfTheNewNodes(t *testing.T) {
	k := func(out dsl.Sig) int { return dsl.Patch{}.Start(out).Program().Kernels() }
	x := dsl.Saw(220)
	if a, b := k(dsl.Add(Saturate(x, 3, 1), Saturate(x, 3, 1))), k(dsl.Add(Saturate(x, 3, 1), Saturate(x, 3, 4))); a != 3 || b != 4 {
		t.Errorf("saturate: same %d (3), another factor %d (4)", a, b)
	}
	f := func(v float64) float64 { return v }
	if n := k(dsl.Add(Shape(x, f, 1), Shape(x, f, 1))); n != 4 { // saw, two shapes, add: a function is never merged
		t.Errorf("shapes of a function: %d kernels (4)", n)
	}
	if n := k(dsl.Add(Spectral(x, 64, nothing), Spectral(x, 64, nothing))); n != 4 {
		t.Errorf("spectral: %d kernels (4)", n)
	}
	chain := shapechain.MustParse("sin")
	if n := k(dsl.Add(ChainOsc(chain, 100.0), ChainOsc(shapechain.MustParse("sin"), 100.0))); n != 2 { // the same text is the same wave
		t.Errorf("chain oscillators of the same text: %d kernels (2)", n)
	}
}

// What the audio thread runs allocates nothing, for the new nodes too
func TestNewNodesDoNotAllocate(t *testing.T) {
	if raceOn {
		t.Skip("the race detector allocates")
	}
	for name, build := range map[string]func() dsl.Sig{
		"saturate x4": func() dsl.Sig { return Saturate(dsl.Saw(220), dsl.Lfo(0.5).Range(1, 8), 4) },
		"shape x2":    func() dsl.Sig { return Shape(dsl.Saw(220), math.Tanh, 2) },
		"spectral":    func() dsl.Sig { return Spectral(dsl.Saw(220), 512, nothing) },
		"pitchshift":  func() dsl.Sig { return PitchShift(dsl.Saw(220), dsl.Lfo(0.3).Range(-5, 5), 50) },
		"chainosc": func() dsl.Sig {
			return ChainOsc(shapechain.MustParse("mult m1[1,5] | sin"), 110, dsl.Lfo(0.2).Range(0, 1))
		},
		"crossfade": func() dsl.Sig { return Crossfade(dsl.Saw(220), dsl.Sine(330.0), dsl.Lfo(0.4).Range(0, 1)) },
		"smooth":    func() dsl.Sig { return Smooth(dsl.Saw(220), 20) },
		"fastsine":  func() dsl.Sig { return FastSine(dsl.Lfo(1.0).Range(100, 300)) },
	} {
		r := dsl.Patch{}.Start(dsl.Chans(build(), build())) // two channels
		l, rr := make([]float64, 512), make([]float64, 512)
		r.Process(l, rr)
		if n := testing.AllocsPerRun(100, func() { r.Process(l, rr) }); n != 0 {
			t.Errorf("%s: Process allocates %v times per call", name, n)
		}
	}
}

// The reverb is the block of psalm: what the module of psalm computes for a sample, in the harp graph
func TestFDNReverbIsThePsalmCore(t *testing.T) {
	cfg := fdnreverb.DefaultConfig()
	in := dsl.Chans(impulse(), dsl.Mul(impulse(), 0.5))
	got := render(FDNReverbWet(in, cfg), 0.5)
	if len(got) != 2 {
		t.Fatalf("%d channels", len(got))
	}
	ref := newFDNReverb(sr, cfg)
	for i := range got[0] {
		x := 0.0
		if i == 0 {
			x = 1
		}
		l, r := ref.ProcessSample(x, x*0.5)
		if got[0][i] != l || got[1][i] != r {
			t.Fatalf("sample %d: %v %v, the block gives %v %v", i, got[0][i], got[1][i], l, r)
		}
	}
	if maxDiff(got[0], got[1]) < 1e-3 {
		t.Error("left and right are the same: no stereo image")
	}
}

func TestFDNReverbTail(t *testing.T) {
	cfg := fdnreverb.DefaultConfig()
	cfg.Decay, cfg.Damping, cfg.PreDelay = 1.0, 0, 20
	wet := render(FDNReverbWet(impulse(), cfg), 1.6)[0]
	ms := func(from, to float64) []float64 { return wet[int(from*sr):int(to*sr)] }
	// nothing before the pre-delay
	for i, v := range ms(0, 0.018) {
		if v != 0 {
			t.Fatalf("sound %v at sample %d, before the pre-delay of 20 ms", v, i)
		}
	}
	if rms(ms(0.02, 0.2)) < 1e-4 {
		t.Error("no reverb after the pre-delay")
	}
	// RT60 of a second: the level falls about 60 dB a second, with no damping. 0.9 s between two windows is about 54 dB
	early, late := rms(ms(0.15, 0.25)), rms(ms(1.05, 1.15))
	drop := 20 * math.Log10(early/late)
	if drop < 40 || drop > 70 {
		t.Errorf("the tail fell %.0f dB in a second (RT60 of 1 s: about 54)", drop)
	}
}

func TestFDNReverbInputs(t *testing.T) {
	cfg := fdnreverb.DefaultConfig()
	cfg.Mix = 1
	// mono goes to both sides, as in psalm's sockets
	mono := render(FDNReverbWet(impulse(), cfg), 0.3)
	both := render(FDNReverbWet(dsl.Dup(impulse(), 2), cfg), 0.3)
	if maxDiff(mono[0], both[0]) != 0 || maxDiff(mono[1], both[1]) != 0 {
		t.Error("a mono input is not the same as the signal on both sides")
	}
	// width 0 is mono
	cfg.Width = 0
	narrow := render(FDNReverbWet(dsl.Chans(impulse(), dsl.Mul(impulse(), 0.3)), cfg), 0.3)
	if d := maxDiff(narrow[0], narrow[1]); d > 1e-12 {
		t.Errorf("width 0: the sides differ by %g", d)
	}
	// more than two channels have no rule
	if msg := func() (m string) {
		defer func() { m, _ = recover().(string) }()
		FDNReverbWet(dsl.Chans(dsl.N(0), dsl.N(0), dsl.N(0)), cfg)
		return
	}(); !strings.Contains(msg, "3 channels to 2") {
		t.Errorf("three channels: %q", msg)
	}
}

func TestFDNReverbMix(t *testing.T) {
	cfg := fdnreverb.DefaultConfig()
	in := dsl.Chans(dsl.Saw(220), dsl.Saw(330))
	dry, wet := render(in, 0.3), render(FDNReverbWet(in, cfg), 0.3)
	cfg.Mix = 0
	if d := maxDiff(render(FDNReverb(in, cfg), 0.3)[0], dry[0]); d > 1e-3 {
		t.Errorf("mix 0 differs from the dry sound by %g", d)
	}
	cfg.Mix = 1
	if d := maxDiff(render(FDNReverb(in, cfg), 0.3)[1], wet[1]); d > 1e-3 {
		t.Errorf("mix 1 differs from the reverb by %g", d)
	}
	// a mono input comes out stereo
	cfg.Mix = 0.3
	if got := FDNReverb(dsl.Saw(220), cfg).Channels(); got != 2 {
		t.Errorf("%d channels from a mono input", got)
	}
}

// Every reverb has a state of its own, and two of them don't disturb each other
func TestFDNReverbsAreIndependent(t *testing.T) {
	cfg := fdnreverb.DefaultConfig()
	a := render(FDNReverbWet(dsl.Saw(220), cfg), 0.3)
	both := render(dsl.Chans(FDNReverbWet(dsl.Saw(220), cfg), FDNReverbWet(dsl.Saw(331), cfg)), 0.3)
	if maxDiff(both[0], a[0]) != 0 || maxDiff(both[1], a[1]) != 0 {
		t.Error("a reverb is not what it is alone")
	}
	if maxDiff(both[0], both[2]) < 1e-3 {
		t.Error("two reverbs on different input sound the same")
	}
}

func TestFDNReverbDoesNotAllocate(t *testing.T) {
	if raceOn {
		t.Skip("the race detector allocates")
	}
	r := dsl.Patch{}.Start(FDNReverb(dsl.Chans(dsl.Saw(220), dsl.Saw(330)), fdnreverb.DefaultConfig()))
	l, rr := make([]float64, 512), make([]float64, 512)
	r.Process(l, rr)
	if n := testing.AllocsPerRun(100, func() { r.Process(l, rr) }); n != 0 {
		t.Errorf("Process allocates %v times per call", n)
	}
}
