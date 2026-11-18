// Package compare says how much two renderings of a patch are the same sound: the level, the way the energy lies over
// the frequencies, the shape of the waves, how the level moves in time, and for two channels where the sound is
// between left and right. It is made to compare harp with SuperCollider, but it takes any two sets of samples.
//
// The first signal is called the reference (SuperCollider), the second the other (harp), and a difference is
// always the other minus the reference: a level of +6 dB means the other is 6 dB louder.
package compare

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/almerlucke/psalm/blocks/fft"
)

// Options are what can be asked of a comparison
type Options struct {
	Rate        float64 // the sample rate of both, default 44100
	Statistical bool    // do not look at the waves: the patch is random (noise, dust), only what holds for the sound in the long run
	NameA       string  // the name of the reference, default "reference"
	NameB       string  // the name of the other, default "other"
}

// Band is the energy in an octave band of both, in dB of the power: that is the mean of the square of the samples, so a sine
// wave of full scale is -3 dB. NaN is nothing there
type Band struct {
	Lo, Hi float64
	A, B   float64
}

// Channel is what is said of one channel of the two
type Channel struct {
	Index        int
	PeakA, PeakB float64
	RMSA, RMSB   float64
	MeanA, MeanB float64
	LevelDB      float64 // the rms of the other over that of the reference, in dB
	SilentA      bool
	SilentB      bool

	// the waves, at the best alignment, looked for within a few milliseconds
	Lag         int     // A[i] is about B[i-Lag]: positive if the reference is later
	Correlation float64 // from -1 to 1 at that lag: negative if the wave is upside down
	GainDB      float64 // the gain of the other that makes it most like the reference
	NullDB      float64 // what is left of the reference when the other, with that gain, is taken off it, relative to it. -30 or less is alike
	Compared    bool    // false in a statistical comparison

	// the level over time, every 10 ms
	EnvCorrelation float64
	EnvMedianDB    float64 // the median difference of the level of the windows, as a number above 0
	EnvMaxDB       float64

	Bands []Band
}

// Track is where the sound is between left and right over time, the level of the right channel over the left in dB every
// 10 ms, and how the two differ
type Track struct {
	MaxDB, MeanDB float64
	Windows       int
}

// Report is the result
type Report struct {
	Options     Options
	Seconds     float64
	ChannelsA   int
	ChannelsB   int
	Channels    []Channel
	Position    *Track
	Level       string // the verdicts, one word each, and the whole in a sentence
	Spectrum    string
	Waveform    string
	Verdict     string
	LeftOnlyA   bool // the reference is a signal of one channel, heard on the left only (as .play does)
	MaxBandDiff float64
}

// Compare compares two sets of channels. The channels that both have are compared
func Compare(a, b [][]float64, o Options) Report {
	if o.Rate == 0 {
		o.Rate = 44100
	}
	if o.NameA == "" {
		o.NameA = "reference"
	}
	if o.NameB == "" {
		o.NameB = "other"
	}
	r := Report{Options: o, ChannelsA: len(a), ChannelsB: len(b)}
	n := min(len(a), len(b))
	if n == 0 {
		r.Verdict = "There is nothing to compare."
		return r
	}
	frames := len(a[0])
	for c := 0; c < n; c++ {
		frames = min(frames, len(a[c]), len(b[c]))
	}
	r.Seconds = float64(frames) / o.Rate
	for c := 0; c < n; c++ {
		r.Channels = append(r.Channels, compareChannel(c, a[c][:frames], b[c][:frames], o))
	}
	r.LeftOnlyA = len(a) >= 2 && r.Channels[0].SilentA == false && silent(a[1]) && len(b) == 1
	if n >= 2 && !r.Channels[0].SilentA && !r.Channels[0].SilentB && !r.Channels[1].SilentA && !r.Channels[1].SilentB {
		r.Position = position(a[0][:frames], a[1][:frames], b[0][:frames], b[1][:frames], o)
	}
	r.judge()
	return r
}

func silent(x []float64) bool {
	for _, v := range x {
		if math.Abs(v) > 1e-9 {
			return false
		}
	}
	return true
}

func rms(x []float64) float64 {
	if len(x) == 0 {
		return 0
	}
	var s float64
	for _, v := range x {
		s += v * v
	}
	return math.Sqrt(s / float64(len(x)))
}

func mean(x []float64) (m float64) {
	for _, v := range x {
		m += v
	}
	return m / math.Max(float64(len(x)), 1)
}

func peak(x []float64) (p float64) {
	for _, v := range x {
		p = math.Max(p, math.Abs(v))
	}
	return
}

func db(x float64) float64 {
	if x < 1e-12 {
		return math.Inf(-1)
	}
	return 20 * math.Log10(x)
}

func compareChannel(idx int, a, b []float64, o Options) Channel {
	c := Channel{Index: idx, PeakA: peak(a), PeakB: peak(b), RMSA: rms(a), RMSB: rms(b), MeanA: mean(a), MeanB: mean(b)}
	c.SilentA, c.SilentB = c.PeakA < 1e-9, c.PeakB < 1e-9
	if c.SilentA || c.SilentB {
		return c
	}
	c.LevelDB = db(c.RMSB) - db(c.RMSA)
	if !o.Statistical {
		c.Compared = true
		c.Lag = bestLag(a, b, int(o.Rate/100)) // up to 10 ms
		c.Correlation, c.GainDB, c.NullDB = shape(a, b, c.Lag)
	}
	c.EnvCorrelation, c.EnvMedianDB, c.EnvMaxDB = envelope(a, b, int(o.Rate/100))
	c.Bands = bands(a, b, o.Rate)
	return c
}

// bestLag looks for the shift of b, within +-max samples, that makes it most like a, on the part of a that is loudest (a second of it)
func bestLag(a, b []float64, max int) int {
	seg := min(len(a), 44100)
	best, start := -1.0, 0
	for s := 0; s+seg <= len(a); s += seg / 4 {
		if e := rms(a[s : s+seg]); e > best {
			best, start = e, s
		}
	}
	bestC, lag := -1.0, 0
	for l := -max; l <= max; l++ {
		var ab, bb float64
		for i := start; i < start+seg; i++ {
			j := i - l
			if j < 0 || j >= len(b) {
				continue
			}
			ab += a[i] * b[j]
			bb += b[j] * b[j]
		}
		if bb > 0 && math.Abs(ab)/math.Sqrt(bb) > bestC { // the size of the correlation, up to the fixed energy of a
			bestC, lag = math.Abs(ab)/math.Sqrt(bb), l
		}
	}
	return lag
}

// shape gives the correlation, the best gain (in dB) and the null residual of the whole of a and b, with b shifted by lag
func shape(a, b []float64, lag int) (corr, gainDB, nullDB float64) {
	var ab, aa, bb float64
	lo, hi := max(0, lag), min(len(a), len(b)+lag)
	for i := lo; i < hi; i++ {
		x, y := a[i], b[i-lag]
		ab += x * y
		aa += x * x
		bb += y * y
	}
	if aa == 0 || bb == 0 {
		return 0, 0, 0
	}
	corr = ab / math.Sqrt(aa*bb)
	g := ab / bb
	gainDB = db(math.Abs(g))
	var res float64
	for i := lo; i < hi; i++ {
		d := a[i] - g*b[i-lag]
		res += d * d
	}
	nullDB = math.Max(10*math.Log10(res/aa+1e-30), -120)
	return
}

// envelope: the rms of windows of 10 ms for both, how alike they move, and how far apart they are, where both are loud enough to be heard
func envelope(a, b []float64, w int) (corr, median, max float64) {
	var ea, eb []float64
	for i := 0; i+w <= len(a); i += w {
		ea, eb = append(ea, rms(a[i:i+w])), append(eb, rms(b[i:i+w]))
	}
	corr = pearson(ea, eb)
	var diffs []float64
	for i := range ea {
		if ea[i] > 1e-4 && eb[i] > 1e-4 {
			diffs = append(diffs, math.Abs(db(eb[i])-db(ea[i])))
		}
	}
	if len(diffs) == 0 {
		return corr, 0, 0
	}
	sort.Float64s(diffs)
	return corr, diffs[len(diffs)/2], diffs[len(diffs)-1]
}

func pearson(x, y []float64) float64 {
	n := min(len(x), len(y))
	if n < 2 {
		return 0
	}
	mx, my := mean(x[:n]), mean(y[:n])
	var sxy, sxx, syy float64
	for i := 0; i < n; i++ {
		sxy += (x[i] - mx) * (y[i] - my)
		sxx += (x[i] - mx) * (x[i] - mx)
		syy += (y[i] - my) * (y[i] - my)
	}
	if sxx == 0 || syy == 0 {
		return 1 // both are the same level all the way: as alike as can be
	}
	return sxy / math.Sqrt(sxx*syy)
}

// the octave bands, by their edges (the highest is up to the Nyquist frequency)
var edges = []float64{22, 44, 88, 177, 353, 707, 1414, 2828, 5657, 11314, 22627}

// bands is the power in every octave band of both, from the average of windowed transforms (Welch)
func bands(a, b []float64, rate float64) []Band {
	pa, pb := spectrum(a), spectrum(b)
	bin := func(f float64) int { return int(math.Round(f * spectrumSize / rate)) }
	var out []Band
	for i := 0; i+1 < len(edges); i++ {
		lo, hi := edges[i], edges[i+1]
		if lo >= rate/2 {
			break
		}
		hi = math.Min(hi, rate/2)
		sum := func(p []float64) float64 {
			var s float64
			for k := bin(lo); k < bin(hi) && k < len(p); k++ {
				s += p[k]
			}
			return s
		}
		bd := Band{Lo: lo, Hi: hi, A: math.NaN(), B: math.NaN()}
		if x := sum(pa); x > 1e-14 {
			bd.A = 10 * math.Log10(x)
		}
		if x := sum(pb); x > 1e-14 {
			bd.B = 10 * math.Log10(x)
		}
		out = append(out, bd)
	}
	return out
}

const spectrumSize = 8192

// spectrum is the power of every bin of a signal, averaged over windows of 8192 samples that overlap by half: summed over a band it
// is the mean of the square of the samples that lie in it
func spectrum(x []float64) []float64 {
	n := spectrumSize
	f := fft.NewReal(n)
	w := fft.HannWindow(n)
	var wsum float64
	for _, v := range w {
		wsum += v * v
	}
	acc := make([]float64, n/2+1)
	buf, re, im := make([]float64, n), make([]float64, n/2+1), make([]float64, n/2+1)
	var count float64
	for s := 0; s+n <= len(x); s += n / 2 {
		for i := range buf {
			buf[i] = x[s+i] * w[i]
		}
		f.Forward(buf, re, im)
		for k := range acc {
			p := re[k]*re[k] + im[k]*im[k]
			if k != 0 && k != n/2 {
				p *= 2
			}
			acc[k] += p / (float64(n) * wsum)
		}
		count++
	}
	if count > 0 {
		for k := range acc {
			acc[k] /= count
		}
	}
	return acc
}

// position is where the sound is between left and right over time: the level of the right over the left every 10 ms
func position(al, ar, bl, br []float64, o Options) *Track {
	w := int(o.Rate / 100)
	t := &Track{}
	var sum float64
	for i := 0; i+w <= len(al); i += w {
		la, ra, lb, rb := rms(al[i:i+w]), rms(ar[i:i+w]), rms(bl[i:i+w]), rms(br[i:i+w])
		pa, pb := db(ra)-db(la), db(rb)-db(lb)
		if la < 1e-4 || ra < 1e-4 || lb < 1e-4 || rb < 1e-4 || math.Abs(pa) > 30 || math.Abs(pb) > 30 {
			continue // only where both are loud enough on both sides
		}
		d := math.Abs(pb - pa)
		t.MaxDB = math.Max(t.MaxDB, d)
		sum += d
		t.Windows++
	}
	if t.Windows == 0 {
		return nil
	}
	t.MeanDB = sum / float64(t.Windows)
	return t
}

// judge puts the numbers in words. The limits are in the text of the verdict, so that it is clear what "the same" means
func (r *Report) judge() {
	var worstLevel, worstBand float64
	worstNull := math.Inf(-1) // the worst is the highest: it starts below everything
	worstCorr := 1.0
	compared, waves := 0, 0
	inverted := false
	for _, c := range r.Channels {
		if c.SilentA || c.SilentB {
			continue
		}
		compared++
		worstLevel = math.Max(worstLevel, math.Abs(c.LevelDB))
		if c.Compared {
			waves++
			worstCorr = math.Min(worstCorr, math.Abs(c.Correlation))
			worstNull = math.Max(worstNull, c.NullDB)
			inverted = inverted || c.Correlation < 0
		}
		// the bands that count: within 40 dB of the loudest of the channel, in either of the two
		counts := c.counted()
		for i, b := range c.Bands {
			if !counts[i] {
				continue
			}
			// the shape of the spectrum: how the bands differ once the difference in level of the whole is taken off
			worstBand = math.Max(worstBand, math.Abs(nan(b.B)-nan(b.A)-c.LevelDB))
		}
	}
	r.MaxBandDiff = worstBand
	if compared == 0 {
		r.Level, r.Spectrum, r.Waveform = "-", "-", "-"
		r.Verdict = "One of the two is silent: there is nothing to compare."
		return
	}
	r.Level = grade(worstLevel, 0.5, 2, "level")
	r.Spectrum = grade(worstBand, 1, 3, "spectrum")
	switch {
	case waves == 0:
		r.Waveform = "not compared"
	case worstCorr >= 0.99 && worstNull <= -30:
		r.Waveform = "the same"
		if inverted {
			r.Waveform = "the same, upside down"
		}
	case worstCorr >= 0.9:
		r.Waveform = "similar"
	default:
		r.Waveform = "different"
	}
	same := func(g string) bool { return g == "same" }
	close := func(g string) bool { return g == "same" || g == "close" }
	switch {
	case same(r.Level) && same(r.Spectrum) && strings.HasPrefix(r.Waveform, "the same"):
		r.Verdict = "The same sound."
	case close(r.Level) && close(r.Spectrum) && (r.Waveform == "similar" || strings.HasPrefix(r.Waveform, "the same")):
		r.Verdict = "The same sound, with small differences."
	case close(r.Level) && close(r.Spectrum) && r.Waveform == "not compared":
		r.Verdict = "The same in level and in the way the energy lies over the frequencies (the waves were not compared: a random patch)."
	case close(r.Level) && close(r.Spectrum):
		r.Verdict = "The same in level and spectrum, but not in the shape of the waves: that is what a random source does (two noises are never the same), " +
			"and what differences in phase and timing do, such as SuperCollider's control signals that step once per 64 samples. Compare with -stats to look at the long run only."
	case strings.HasPrefix(r.Waveform, "the same") && close(r.Spectrum):
		r.Verdict = fmt.Sprintf("The same waves, but %.1f dB apart in level.", worstLevel)
	case close(r.Spectrum):
		r.Verdict = fmt.Sprintf("The same spectrum, but %.1f dB apart in level.", worstLevel)
	case close(r.Level):
		r.Verdict = fmt.Sprintf("The same level, but the energy lies differently over the frequencies (up to %.1f dB in a band).", worstBand)
		if r.Options.Statistical {
			r.Verdict += " With a random source a few dB in a band can be chance, less the longer the render is."
		}
	default:
		r.Verdict = "Different sounds."
	}
}

func nan(v float64) float64 {
	if math.IsNaN(v) {
		return -200
	}
	return v
}

func grade(x, same, close float64, _ string) string {
	switch {
	case x <= same:
		return "same"
	case x <= close:
		return "close"
	}
	return "different"
}

// name is the band in Hz, with k for thousands: 22-44, 707-1.4k, 5.7k-11k
func (b Band) name() string {
	f := func(v float64) string {
		switch {
		case v < 1000:
			return fmt.Sprintf("%.0f", v)
		case v < 10000:
			return fmt.Sprintf("%.1fk", v/1000)
		}
		return fmt.Sprintf("%.0fk", v/1000)
	}
	return f(b.Lo) + "-" + f(b.Hi)
}

// counted says which bands of a channel count: those within 40 dB of the loudest band of the two. The others hold no more than the
// rounding of the samples and the leakage of the transform, and what they say is not about the sound
func (c Channel) counted() []bool {
	loud := math.Inf(-1)
	for _, b := range c.Bands {
		loud = math.Max(loud, math.Max(nan(b.A), nan(b.B)))
	}
	out := make([]bool, len(c.Bands))
	for i, b := range c.Bands {
		out[i] = math.Max(nan(b.A), nan(b.B)) >= loud-40
	}
	return out
}

// String prints the report
func (r Report) String() string {
	var s strings.Builder
	a, b := r.Options.NameA, r.Options.NameB
	fmt.Fprintf(&s, "%s against %s: %.1f seconds at %.0f Hz\n", b, a, r.Seconds, r.Options.Rate)
	if r.ChannelsA != r.ChannelsB {
		fmt.Fprintf(&s, "channels: %s has %d, %s has %d; the first %d are compared", a, r.ChannelsA, b, r.ChannelsB, len(r.Channels))
		if r.LeftOnlyA {
			fmt.Fprintf(&s, " (%s plays a function of one channel on the left only, as .play does: its right channel is silent)", a)
		}
		s.WriteString("\n")
	}
	for _, c := range r.Channels {
		fmt.Fprintf(&s, "\nchannel %d", c.Index+1)
		switch {
		case c.SilentA && c.SilentB:
			s.WriteString(": both are silent\n")
			continue
		case c.SilentA:
			fmt.Fprintf(&s, ": %s is silent, %s is not (peak %.4f)\n", a, b, c.PeakB)
			continue
		case c.SilentB:
			fmt.Fprintf(&s, ": %s is silent, %s is not (peak %.4f)\n", b, a, c.PeakA)
			continue
		}
		s.WriteString("\n")
		fmt.Fprintf(&s, "  level      rms %.5f against %.5f: %+.2f dB     peak %.4f against %.4f     mean %+.5f against %+.5f\n", c.RMSB, c.RMSA, c.LevelDB, c.PeakB, c.PeakA, c.MeanB, c.MeanA)
		if c.Compared {
			up := ""
			if c.Correlation < 0 {
				up = " (upside down)"
			}
			fmt.Fprintf(&s, "  waves      aligned %+d samples; correlation %+.4f%s; after the best gain (%+.2f dB) what is left of %s is %.1f dB\n", c.Lag, c.Correlation, up, c.GainDB, a, c.NullDB)
		}
		fmt.Fprintf(&s, "  level in time (every 10 ms)   correlation %.3f, median difference %.2f dB, largest %.1f dB\n", c.EnvCorrelation, c.EnvMedianDB, c.EnvMaxDB)
		if len(c.Bands) > 0 {
			counts := c.counted()
			fmt.Fprintf(&s, "  octave band (Hz)  ")
			for _, bd := range c.Bands {
				fmt.Fprintf(&s, "%11s", bd.name())
			}
			fmt.Fprintf(&s, "\n  %-18s", a+" dB")
			for _, bd := range c.Bands {
				fmt.Fprintf(&s, "%11s", fnum(bd.A))
			}
			fmt.Fprintf(&s, "\n  %-18s", b+" dB")
			for _, bd := range c.Bands {
				fmt.Fprintf(&s, "%11s", fnum(bd.B))
			}
			fmt.Fprintf(&s, "\n  %-18s", "difference")
			for i, bd := range c.Bands {
				switch {
				case math.IsNaN(bd.A) || math.IsNaN(bd.B) || !counts[i]:
					fmt.Fprintf(&s, "%11s", ".")
				default:
					fmt.Fprintf(&s, "%+11.1f", bd.B-bd.A)
				}
			}
			s.WriteString("\n  (a . is a band more than 40 dB below the loudest: not about the sound, and not counted)\n")
		}
	}
	if r.Position != nil {
		fmt.Fprintf(&s, "\nposition between left and right (level of right over left, every 10 ms, %d windows): differs by %.2f dB on average, %.2f dB at most\n", r.Position.Windows, r.Position.MeanDB, r.Position.MaxDB)
	}
	fmt.Fprintf(&s, "\nlevel: %s (within 0.5 dB is the same, within 2 dB close)\n", r.Level)
	fmt.Fprintf(&s, "spectrum: %s (the shape: the loud octave bands, once the difference in level is taken off, within 1 dB is the same, within 3 dB close; the biggest is %.1f dB)\n", r.Spectrum, r.MaxBandDiff)
	fmt.Fprintf(&s, "waves: %s (correlation 0.99 or more and what is left 30 dB down is the same, 0.9 or more is similar)\n", r.Waveform)
	fmt.Fprintf(&s, "\n%s\n", r.Verdict)
	return s.String()
}

func fnum(v float64) string {
	if math.IsNaN(v) {
		return "-"
	}
	return fmt.Sprintf("%.1f", v)
}
