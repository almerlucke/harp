package compare

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// For a patch that is made by chance when it is built (random choices, a seed), one seed in SuperCollider and one in harp are not
// the same patch: the random numbers are other numbers. What can be the same is the sort of patch: over many seeds, how loud
// they are, how bright, and where the energy lies. Seeds compares the two sets of runs by their distributions.

// Metric is one number for every run (the level, the centre of the spectrum, the share of an octave band), for both sides
type Metric struct {
	Name string
	A, B []float64 // for each run, sorted
	// D is the largest distance between the two distributions (Kolmogorov and Smirnov, 0 for the same, 1 for apart), Limit what it
	// is allowed to be to still be the chance of runs of this number (5% chance of a wrong "different")
	D, Limit float64
	Same     bool
}

// SeedReport is the comparison of the runs of many seeds
type SeedReport struct {
	NameA, NameB string
	Runs         int
	Metrics      []Metric
	Verdict      string
}

// perRun is what is taken from a run
func perRun(ch [][]float64, rate float64) (level, centroid float64, share []float64) {
	var sum float64
	var n int
	for _, c := range ch {
		for _, v := range c {
			sum += v * v
		}
		n += len(c)
	}
	level = db(math.Sqrt(sum / float64(max(n, 1))))
	p := spectrum(ch[0])
	var tot, wsum float64
	for k, v := range p {
		f := float64(k) * rate / spectrumSize
		tot += v
		wsum += v * f
	}
	if tot > 0 {
		centroid = wsum / tot
	}
	bin := func(f float64) int { return int(math.Round(f * spectrumSize / rate)) }
	for i := 0; i+1 < len(edges) && edges[i] < rate/2; i++ {
		hi := math.Min(edges[i+1], rate/2)
		var s float64
		for k := bin(edges[i]); k < bin(hi) && k < len(p); k++ {
			s += p[k]
		}
		if s <= 0 || tot <= 0 {
			share = append(share, -100)
		} else {
			share = append(share, math.Max(-100, 10*math.Log10(s/tot)))
		}
	}
	return
}

// Seeds compares the runs a (one entry for each seed, as the channels of a render) with the runs b
func Seeds(a, b [][][]float64, o Options) SeedReport {
	r := SeedReport{NameA: o.NameA, NameB: o.NameB, Runs: min(len(a), len(b))}
	if len(a) == 0 || len(b) == 0 {
		r.Verdict = "There are no runs to compare."
		return r
	}
	collect := func(runs [][][]float64) (lv, ce []float64, sh [][]float64) {
		for _, run := range runs {
			l, c, s := perRun(run, o.Rate)
			lv, ce = append(lv, l), append(ce, c)
			for i, v := range s {
				if len(sh) <= i {
					sh = append(sh, nil)
				}
				sh[i] = append(sh[i], v)
			}
		}
		return
	}
	la, ca, sa := collect(a)
	lb, cb, sb := collect(b)
	r.Metrics = append(r.Metrics, metric("level (dB)", la, lb), metric("centre of the spectrum (Hz)", ca, cb))
	for i := 0; i < len(sa) && i < len(sb); i++ {
		name := Band{Lo: edges[i], Hi: edges[i+1]}.name()
		// an octave that is silent in both has nothing to compare
		if median(sa[i]) < -60 && median(sb[i]) < -60 {
			continue
		}
		r.Metrics = append(r.Metrics, metric("share of "+name+" (dB)", sa[i], sb[i]))
	}
	differ := 0
	for _, m := range r.Metrics {
		if !m.Same {
			differ++
		}
	}
	switch {
	case differ == 0:
		r.Verdict = fmt.Sprintf("The same sort of patch: over %d and %d seeds the distributions of the level, the brightness and the bands are the same, within what chance does.", len(a), len(b))
	case m0(r.Metrics) && differ <= 1 && len(r.Metrics) >= 6:
		r.Verdict = fmt.Sprintf("Nearly the same sort of patch: level and brightness are the same; one of %d numbers differs, which with this many numbers can be chance.", len(r.Metrics))
	case !r.Metrics[0].Same:
		r.Verdict = "Not the same sort of patch: the levels are not distributed alike."
	default:
		r.Verdict = fmt.Sprintf("Not the same sort of patch: %d of %d numbers are distributed differently.", differ, len(r.Metrics))
	}
	if r.Runs < 10 {
		r.Verdict += " (With fewer than 10 seeds only a big difference is seen.)"
	}
	return r
}

func m0(ms []Metric) bool { return ms[0].Same && ms[1].Same }

func metric(name string, a, b []float64) Metric {
	a, b = append([]float64(nil), a...), append([]float64(nil), b...)
	sort.Float64s(a)
	sort.Float64s(b)
	d := ks(a, b)
	limit := 1.36 * math.Sqrt(float64(len(a)+len(b))/float64(len(a)*len(b)))
	return Metric{Name: name, A: a, B: b, D: d, Limit: limit, Same: d <= limit}
}

// ks is the largest distance between the distribution functions of two sorted sets
func ks(a, b []float64) float64 {
	var d float64
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		x := math.Min(a[i], b[j])
		for i < len(a) && a[i] <= x {
			i++
		}
		for j < len(b) && b[j] <= x {
			j++
		}
		d = math.Max(d, math.Abs(float64(i)/float64(len(a))-float64(j)/float64(len(b))))
	}
	return d
}

func median(s []float64) float64 {
	c := append([]float64(nil), s...)
	sort.Float64s(c)
	return quantile(c, 0.5)
}

// quantile of a sorted set, between the entries
func quantile(s []float64, q float64) float64 {
	if len(s) == 0 {
		return math.NaN()
	}
	p := q * float64(len(s)-1)
	i := int(p)
	if i+1 >= len(s) {
		return s[len(s)-1]
	}
	return s[i] + (s[i+1]-s[i])*(p-float64(i))
}

func (r SeedReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d seeds, the distribution of each number over them (lowest, a quarter, median, three quarters, highest):\n\n", r.Runs)
	w := 0
	for _, m := range r.Metrics {
		w = max(w, len(m.Name))
	}
	row := func(s []float64) string {
		var p []string
		for _, q := range []float64{0, 0.25, 0.5, 0.75, 1} {
			p = append(p, fmt.Sprintf("%8.1f", quantile(s, q)))
		}
		return strings.Join(p, " ")
	}
	for _, m := range r.Metrics {
		verdict := "same"
		if !m.Same {
			verdict = "different"
		}
		fmt.Fprintf(&b, "%-*s  %-13s %s\n%-*s  %-13s %s   %s (distance %.2f, up to %.2f is chance)\n", w, m.Name, r.NameA, row(m.A), w, "", r.NameB, row(m.B), verdict, m.D, m.Limit)
	}
	fmt.Fprintf(&b, "\n%s\n", r.Verdict)
	return b.String()
}
