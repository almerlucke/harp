package compare

import (
	"math"
	"math/rand"
	"strings"
	"testing"
)

const rate = 44100.0

func sine(freq, amp float64, seconds float64) []float64 {
	x := make([]float64, int(seconds*rate))
	for i := range x {
		x[i] = amp * math.Sin(2*math.Pi*freq*float64(i)/rate)
	}
	return x
}

func scale(x []float64, g float64) []float64 {
	y := make([]float64, len(x))
	for i := range x {
		y[i] = x[i] * g
	}
	return y
}

func noise(seed int64, amp, seconds float64) []float64 {
	r := rand.New(rand.NewSource(seed))
	x := make([]float64, int(seconds*rate))
	for i := range x {
		x[i] = amp * (r.Float64()*2 - 1)
	}
	return x
}

func near(t *testing.T, what string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %v, want %v (+-%v)", what, got, want, tol)
	}
}

func rich(seconds float64) []float64 { // a few partials, so that there are several bands
	x := sine(110, 0.3, seconds)
	for i, f := range []float64{330, 990, 2970} {
		y := sine(f, 0.15/float64(i+1), seconds)
		for k := range x {
			x[k] += y[k]
		}
	}
	return x
}

func TestTheSameSignal(t *testing.T) {
	x := rich(3)
	r := Compare([][]float64{x}, [][]float64{append([]float64(nil), x...)}, Options{})
	c := r.Channels[0]
	near(t, "level", c.LevelDB, 0, 1e-9)
	near(t, "correlation", c.Correlation, 1, 1e-9)
	if c.Lag != 0 || c.NullDB > -100 || c.EnvCorrelation < 0.999 {
		t.Errorf("lag %d, null %v, envelope %v", c.Lag, c.NullDB, c.EnvCorrelation)
	}
	if r.Verdict != "The same sound." || r.Level != "same" || r.Spectrum != "same" || r.Waveform != "the same" {
		t.Errorf("verdict %q (%s, %s, %s)", r.Verdict, r.Level, r.Spectrum, r.Waveform)
	}
}

// the same wave with another gain: the level is said, and the shape of the spectrum is still the same
func TestAGain(t *testing.T) {
	x := rich(3)
	r := Compare([][]float64{x}, [][]float64{scale(x, 0.5)}, Options{})
	c := r.Channels[0]
	near(t, "level", c.LevelDB, -6.0206, 1e-3)
	near(t, "the gain that makes it alike", c.GainDB, 6.0206, 1e-3)
	near(t, "correlation", c.Correlation, 1, 1e-9)
	if r.Level != "different" || r.Spectrum != "same" || r.Waveform != "the same" {
		t.Errorf("%s %s %s", r.Level, r.Spectrum, r.Waveform)
	}
	if !strings.Contains(r.Verdict, "The same waves, but 6.0 dB apart in level") {
		t.Errorf("verdict %q", r.Verdict)
	}
	for _, b := range c.Bands {
		if !math.IsNaN(b.A) && !math.IsNaN(b.B) && math.Abs(b.B-b.A+6.0206) > 0.05 {
			t.Errorf("band %s differs by %v, want -6.02", b.name(), b.B-b.A)
		}
	}
}

func TestADelayAndAnInversion(t *testing.T) {
	x := rich(3)
	delayed := append(make([]float64, 3), x[:len(x)-3]...) // b[i] = a[i-3]
	r := Compare([][]float64{x}, [][]float64{delayed}, Options{})
	if c := r.Channels[0]; c.Lag != -3 || c.Correlation < 0.9999 {
		t.Errorf("a delay of 3: lag %d, correlation %v", c.Lag, c.Correlation)
	}
	later := append(make([]float64, 40), x[:len(x)-40]...) // a is the one that is earlier
	r = Compare([][]float64{later}, [][]float64{x}, Options{})
	if c := r.Channels[0]; c.Lag != 40 {
		t.Errorf("the reference is 40 samples later: lag %d, want 40", c.Lag)
	}
	r = Compare([][]float64{x}, [][]float64{scale(x, -1)}, Options{})
	c := r.Channels[0]
	near(t, "inverted", c.Correlation, -1, 1e-9)
	if r.Waveform != "the same, upside down" || !strings.Contains(r.String(), "upside down") {
		t.Errorf("waves: %q", r.Waveform)
	}
}

// two noises are never the same, but they are the same kind: the level and the bands are, the waves are not
func TestTwoNoises(t *testing.T) {
	r := Compare([][]float64{noise(1, 0.3, 8)}, [][]float64{noise(2, 0.3, 8)}, Options{})
	c := r.Channels[0]
	if math.Abs(c.LevelDB) > 0.2 || r.MaxBandDiff > 1 || math.Abs(c.Correlation) > 0.1 {
		t.Errorf("level %v, bands %v, correlation %v", c.LevelDB, r.MaxBandDiff, c.Correlation)
	}
	if r.Waveform != "different" || !strings.Contains(r.Verdict, "not in the shape of the waves") {
		t.Errorf("waves %q, verdict %q", r.Waveform, r.Verdict)
	}
	// asked for the long run only
	r = Compare([][]float64{noise(1, 0.3, 8)}, [][]float64{noise(2, 0.3, 8)}, Options{Statistical: true})
	if r.Channels[0].Compared || r.Waveform != "not compared" || !strings.Contains(r.Verdict, "a random patch") {
		t.Errorf("statistical: compared %v, waves %q, verdict %q", r.Channels[0].Compared, r.Waveform, r.Verdict)
	}
}

func TestAnotherTone(t *testing.T) {
	r := Compare([][]float64{sine(440, 0.3, 3)}, [][]float64{sine(1760, 0.3, 3)}, Options{})
	if r.Level != "same" || r.Spectrum != "different" || !strings.Contains(r.Verdict, "energy lies differently") {
		t.Errorf("%s %s: %q", r.Level, r.Spectrum, r.Verdict)
	}
}

// where the sound is between left and right
func TestPosition(t *testing.T) {
	x := rich(3)
	pan := func(deg float64) [][]float64 {
		th := deg * math.Pi / 180
		return [][]float64{scale(x, math.Cos(th)), scale(x, math.Sin(th))}
	}
	r := Compare(pan(20), pan(20), Options{})
	if r.Position == nil || r.Position.MaxDB > 1e-6 {
		t.Errorf("the same pan: %+v", r.Position)
	}
	r = Compare(pan(20), pan(24), Options{}) // 20 degrees: right over left is 20log10(tan 20) = -8.8 dB; 24: -7.1 dB
	near(t, "a different pan", r.Position.MeanDB, 20*math.Log10(math.Tan(24*math.Pi/180)/math.Tan(20*math.Pi/180)), 0.01)
	swapped := pan(20)
	swapped[0], swapped[1] = swapped[1], swapped[0]
	r = Compare(pan(20), swapped, Options{})
	near(t, "the sides swapped", r.Position.MaxDB, 2*math.Abs(20*math.Log10(math.Tan(20*math.Pi/180))), 0.01)
}

// SuperCollider plays a function of one channel on the left only: the right channel is silent, and harp has one channel
func TestChannelsThatAreNotTheSame(t *testing.T) {
	x := rich(2)
	r := Compare([][]float64{x, make([]float64, len(x))}, [][]float64{x}, Options{NameA: "SuperCollider", NameB: "harp"})
	if !r.LeftOnlyA || len(r.Channels) != 1 || r.Verdict != "The same sound." {
		t.Errorf("left only %v, %d channels, %q", r.LeftOnlyA, len(r.Channels), r.Verdict)
	}
	out := r.String()
	for _, want := range []string{"harp against SuperCollider", "left only", "The same sound."} {
		if !strings.Contains(out, want) {
			t.Errorf("the report does not say %q:\n%s", want, out)
		}
	}
	// one is silent
	r = Compare([][]float64{x}, [][]float64{make([]float64, len(x))}, Options{})
	if !strings.Contains(r.Verdict, "silent") || !strings.Contains(r.String(), "is silent") {
		t.Errorf("silent: %q", r.Verdict)
	}
	// nothing
	if r := Compare(nil, nil, Options{}); !strings.Contains(r.Verdict, "nothing to compare") {
		t.Errorf("nothing: %q", r.Verdict)
	}
	// signals of other lengths are compared over the shorter one
	r = Compare([][]float64{x}, [][]float64{x[:len(x)/2]}, Options{})
	near(t, "seconds", r.Seconds, 1, 0.001)
}

func TestTheBandsAddUp(t *testing.T) {
	// the bands are the power of the signal: together they are its mean square (a sine of 0.5 is 0.125)
	x := sine(1000, 0.5, 4)
	c := Compare([][]float64{x}, [][]float64{x}, Options{}).Channels[0]
	var sum float64
	for _, b := range c.Bands {
		if !math.IsNaN(b.A) {
			sum += math.Pow(10, b.A/10)
		}
	}
	near(t, "the sum of the bands", sum, 0.125, 0.125*0.02)
	// and the band that has the sine is the one of 707 to 1414
	for _, b := range c.Bands {
		if b.Lo == 707 && (math.IsNaN(b.A) || b.A < -9.5) {
			t.Errorf("the band of 707-1414 Hz: %v dB, want about -9", b.A)
		}
	}
}

func noiseRuns(seed0, runs int, gain float64, rate float64) [][][]float64 {
	var out [][][]float64
	for r := 0; r < runs; r++ {
		x := make([]float64, int(rate)*2)
		s := uint64(seed0+r)*2654435761 + 1
		for i := range x {
			s ^= s << 13
			s ^= s >> 7
			s ^= s << 17
			x[i] = gain * (float64(s%2000001)/1000000 - 1) * (0.5 + 0.5*float64(r%5)/4)
		}
		out = append(out, [][]float64{x})
	}
	return out
}

func TestSeedsTheSameKind(t *testing.T) {
	o := Options{Rate: 44100, NameA: "A", NameB: "B"}
	r := Seeds(noiseRuns(1, 20, 0.3, 44100), noiseRuns(100, 20, 0.3, 44100), o)
	if !strings.HasPrefix(r.Verdict, "The same sort") {
		t.Errorf("%s\n%s", r.Verdict, r)
	}
	// twice as loud: the levels are apart, the shape of the spectrum is the same
	r = Seeds(noiseRuns(1, 20, 0.3, 44100), noiseRuns(100, 20, 1.2, 44100), o)
	if !strings.HasPrefix(r.Verdict, "Not the same sort") || r.Metrics[0].Same {
		t.Errorf("%s\n%s", r.Verdict, r)
	}
	if !r.Metrics[1].Same {
		t.Error("the centre of the spectrum is the same for noise of any level")
	}
	if got := Seeds(nil, nil, o).Verdict; !strings.Contains(got, "no runs") {
		t.Error(got)
	}
	if !strings.Contains(r.String(), "20 seeds") {
		t.Error(r.String())
	}
}

func TestKS(t *testing.T) {
	if d := ks([]float64{1, 2, 3}, []float64{1, 2, 3}); d != 0 {
		t.Error(d)
	}
	if d := ks([]float64{1, 2, 3}, []float64{4, 5, 6}); d != 1 {
		t.Error(d)
	}
}
