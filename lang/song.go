package lang

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"harp/blocks"
	"harp/dsl"
)

// A song is the form of a piece: sections with a number of bars each (Intro 8, Drop 16, Break 8). It gives signals that follow it: during(song,
// "Drop") is 1 in the drop and 0 elsewhere (to put a layer in and out), curve(song, "Intro: 0 -> 1") a value that moves through the sections.
// It goes round when it has gone through.

// Song is the sections of a piece in steps
type Song struct {
	Text     string
	BarSteps int
	Names    []string
	Start    []float64 // of each section, in steps
	Len      []float64
	Total    float64
}

func (s *Song) fingerprint() string { return fmt.Sprintf("song(%q,%d)", s.Text, s.BarSteps) }

func parseSong(text string, barSteps int) (*Song, error) {
	s := &Song{Text: text, BarSteps: barSteps}
	for _, part := range strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == '\n' || r == ';' }) {
		f := strings.Fields(part)
		if len(f) == 0 {
			continue
		}
		bars, err := strconv.ParseFloat(f[len(f)-1], 64)
		if len(f) < 2 || err != nil || bars <= 0 || bars > 4096 || math.IsNaN(bars) {
			return nil, fmt.Errorf("%q: a section is a name and its bars, like \"Drop 16\"", strings.TrimSpace(part))
		}
		s.Names = append(s.Names, strings.Join(f[:len(f)-1], " "))
		s.Start = append(s.Start, s.Total)
		s.Len = append(s.Len, bars*float64(barSteps))
		s.Total += bars * float64(barSteps)
	}
	if len(s.Names) == 0 {
		return nil, fmt.Errorf("the song has no sections: \"Intro 8, Drop 16\"")
	}
	return s, nil
}

// index is the sections that a name (any case) is: a name can come twice
func (s *Song) sectionsNamed(name string) []int {
	var out []int
	for i, n := range s.Names {
		if strings.EqualFold(n, name) {
			out = append(out, i)
		}
	}
	return out
}

func (s *Song) known(name string) ([]int, error) {
	idx := s.sectionsNamed(name)
	if len(idx) == 0 {
		return nil, fmt.Errorf("the song has no section %q: it has %s%s", name, strings.Join(s.Names, ", "), didYouMean(name, s.Names))
	}
	return idx, nil
}

// which sections a text says: "Drop, Verse", "not Intro", "all"
func (s *Song) select_(text string) ([]bool, error) {
	on := make([]bool, len(s.Names))
	t := strings.TrimSpace(text)
	if strings.EqualFold(t, "all") {
		for i := range on {
			on[i] = true
		}
		return on, nil
	}
	not := false
	if low := strings.ToLower(t); strings.HasPrefix(low, "not ") {
		not, t = true, strings.TrimSpace(t[4:])
	}
	for _, name := range strings.FieldsFunc(t, func(r rune) bool { return r == ',' || r == ';' }) {
		idx, err := s.known(strings.TrimSpace(name))
		if err != nil {
			return nil, err
		}
		for _, i := range idx {
			on[i] = true
		}
	}
	if not {
		for i := range on {
			on[i] = !on[i]
		}
	}
	return on, nil
}

// at is the section a step is in (the song goes round), and how far through it (0 to 1)
func (s *Song) at(step float64) (int, float64) {
	t := math.Mod(step, s.Total)
	if t < 0 {
		t += s.Total
	}
	for i := len(s.Names) - 1; i >= 0; i-- {
		if t >= s.Start[i] {
			return i, (t - s.Start[i]) / s.Len[i]
		}
	}
	return 0, 0
}

func clamp01(x float64) float64 { return math.Max(0, math.Min(1, x)) }

// gate is 1 in the sections that are on, with a ramp of ramp steps around the edges of them (half before, half after: the sound that is let in a
// little early goes on a little after)
func (s *Song) gate(on []bool, ramp float64) func(step float64) float64 {
	type span struct{ a, b float64 }
	// the spans of the sections that are on, for the song before this time, this time and the next (it goes round); the ones that follow each other
	// are one, so that there is no dip where a section ends and the next begins
	var spans []span
	for _, cycle := range [3]float64{-1, 0, 1} {
		for i := range on {
			if !on[i] {
				continue
			}
			a := s.Start[i] + cycle*s.Total
			b := a + s.Len[i]
			if n := len(spans); n > 0 && spans[n-1].b == a {
				spans[n-1].b = b
				continue
			}
			spans = append(spans, span{a, b})
		}
	}
	return func(step float64) float64 {
		t := math.Mod(step, s.Total)
		if t < 0 {
			t += s.Total
		}
		v := 0.0
		for _, sp := range spans {
			if ramp <= 0 {
				if t >= sp.a && t < sp.b {
					v = 1
				}
				continue
			}
			v = math.Max(v, clamp01((t-sp.a)/ramp+0.5)*clamp01((sp.b-t)/ramp+0.5))
		}
		return v
	}
}

type curvePart struct {
	a, b  float64
	shape string
}

func (s *Song) curveOf(text string) ([]curvePart, error) {
	entries := map[string]curvePart{}
	for _, part := range strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == '\n' || r == ';' }) {
		name, rest, ok := strings.Cut(part, ":")
		if !ok {
			return nil, fmt.Errorf("%q: a curve is like \"Intro: 0 -> 1 smooth\"", strings.TrimSpace(part))
		}
		if _, err := s.known(strings.TrimSpace(name)); err != nil {
			return nil, err
		}
		rest = strings.NewReplacer("→", " -> ", "->", " -> ").Replace(rest)
		f := strings.Fields(rest)
		cp := curvePart{shape: "lin"}
		if n := len(f); n > 0 {
			switch f[n-1] {
			case "lin", "exp", "smooth", "step":
				cp.shape, f = f[n-1], f[:n-1]
			}
		}
		nums := func(xs ...string) ([]float64, error) {
			var out []float64
			for _, x := range xs {
				v, err := strconv.ParseFloat(x, 64)
				if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
					return nil, fmt.Errorf("%q in %q is not a number", x, strings.TrimSpace(part))
				}
				out = append(out, v)
			}
			return out, nil
		}
		switch {
		case len(f) == 1:
			v, err := nums(f[0])
			if err != nil {
				return nil, err
			}
			cp.a, cp.b = v[0], v[0]
		case len(f) == 3 && f[1] == "->":
			v, err := nums(f[0], f[2])
			if err != nil {
				return nil, err
			}
			cp.a, cp.b = v[0], v[1]
		default:
			return nil, fmt.Errorf("%q: a value (\"Drop: 1\") or two with an arrow (\"Intro: 0 -> 1\"), and perhaps lin, exp, smooth or step", strings.TrimSpace(part))
		}
		if cp.shape == "exp" && (cp.a <= 0 || cp.b <= 0) {
			return nil, fmt.Errorf("%q: an exp curve goes between values above 0", strings.TrimSpace(part))
		}
		entries[strings.ToLower(strings.TrimSpace(name))] = cp
	}
	// a section with no curve keeps the value where the one before it ended
	parts := make([]curvePart, len(s.Names))
	prev := 0.0
	for i, n := range s.Names {
		if e, ok := entries[strings.ToLower(n)]; ok {
			parts[i] = e
		} else {
			parts[i] = curvePart{prev, prev, "lin"}
		}
		prev = parts[i].b
	}
	return parts, nil
}

func (c curvePart) at(p float64) float64 {
	switch c.shape {
	case "smooth":
		return c.a + (c.b-c.a)*p*p*(3-2*p)
	case "exp":
		return c.a * math.Pow(c.b/c.a, p)
	case "step":
		return c.a
	}
	return c.a + (c.b-c.a)*p
}

// sectionGen picks from generators by the section the step is in: it is pulled once a step
type sectionGen[T any] struct {
	song    *Song
	which   []int // for each section, the generator it plays (-1 none)
	gens    []dsl.Gen[T]
	restart bool
	step    float64
	was     int
	zero    T
}

func (g *sectionGen[T]) Generate() T {
	sec, _ := g.song.at(g.step)
	g.step++
	k := g.which[sec]
	if k < 0 {
		return g.zero
	}
	if g.restart && g.was != sec {
		g.gens[k].Reset()
	}
	g.was = sec
	return g.gens[k].Generate()
}
func (g *sectionGen[T]) Continuous() bool { return true }
func (g *sectionGen[T]) Done() bool       { return false }
func (g *sectionGen[T]) Reset() {
	g.step, g.was = 0, -1
	for _, c := range g.gens {
		c.Reset()
	}
}

func sectionsOf[T any](song *Song, which []int, gens []dsl.Gen[T], restart bool) dsl.Gen[T] {
	return &sectionGen[T]{song: song, which: which, gens: gens, restart: restart, was: -1}
}

func init() {
	add(&Func{Name: "song", Result: TSong, Doc: "the form of a piece: sections with their bars, song(\"Intro 8, Verse 8, Drop 16, Break 8, Drop 16, Outro 8\"). barSteps is the steps of a bar (16). It goes round when it has gone through. Use it with during (a layer in and out), curve (a value over the sections) and bySection (other rhythms in other sections)",
		Params: []Param{{"sections", TStr, nil}, {"barSteps", TNum, def(16)}},
		Call: func(a Args) (Value, error) {
			if a.Int("barSteps") < 1 || a.Int("barSteps") > 256 {
				return Value{}, fmt.Errorf("barSteps is from 1 to 256")
			}
			s, err := parseSong(a.Str("sections"), a.Int("barSteps"))
			if err != nil {
				return Value{}, err
			}
			return Value{TSong, s}, nil
		}})
	add(&Func{Name: "during", Result: TSig, Doc: "1 in the sections of a song that are named, 0 elsewhere, as a signal: .gain(during(form, \"Drop, Chorus\")) puts a layer in and out. \"not Intro\" is all but that one, \"all\" every section. glide is how many steps the change takes, around the edge of a section (2)",
		Params: []Param{{"song", TSong, nil}, {"sections", TStr, nil}, {"glide", TNum, def(2)}},
		Call: func(a Args) (Value, error) {
			s := a.value("song").V.(*Song)
			on, err := s.select_(a.Str("sections"))
			if err != nil {
				return Value{}, err
			}
			if a.Num("glide") < 0 || a.Num("glide") > 256 {
				return Value{}, fmt.Errorf("glide is from 0 to 256 steps")
			}
			return sigV(dsl.Timeline(fmt.Sprintf("during %s | %v | %g", s.fingerprint(), on, a.Num("glide")), s.gate(on, a.Num("glide")))), nil
		}})
	add(&Func{Name: "curve", Result: TSig, Doc: "a value that moves through the sections of a song, as a signal: curve(form, \"Intro: 0 -> 1 smooth, Drop: 1, Break: 1 -> 0.3\"). A section is a value (held), or two with an arrow, and lin (the default), exp (between values above 0), smooth or step. A section with no curve keeps the value where the one before it ended; the sections that have a name twice share the curve",
		Params: []Param{{"song", TSong, nil}, {"curves", TStr, nil}},
		Call: func(a Args) (Value, error) {
			s := a.value("song").V.(*Song)
			parts, err := s.curveOf(a.Str("curves"))
			if err != nil {
				return Value{}, err
			}
			return sigV(dsl.Timeline(fmt.Sprintf("curve %s | %q", s.fingerprint(), a.Str("curves")), func(step float64) float64 {
				i, p := s.at(step)
				return parts[i].at(p)
			})), nil
		}})
	add(&Func{Name: "bySection", Result: TAny, Doc: "other generators in other sections of a song: bySection(form, [\"Intro, Verse\", \"Drop\"], [every(4), euclid(16, 9)]) plays the first in the intro and the verse and the second in the drop (a section that is in no name plays nothing). It takes a value from the generator of the section for every step, so it is for rhythms; restart: true starts a generator from its start whenever its section comes",
		Params: []Param{{"song", TSong, nil}, {"sections", ListOf(TStr), nil}, {"gens", anyList, nil}, {"restart", TBool, defBool(false)}},
		Call: func(a Args) (Value, error) {
			s := a.value("song").V.(*Song)
			names := a.value("sections").Items()
			if len(names) != len(a.value("gens").Items()) {
				return Value{}, fmt.Errorf("%d names and %d generators: a name (or names) for each", len(names), len(a.value("gens").Items()))
			}
			which := make([]int, len(s.Names))
			for i := range which {
				which[i] = -1
			}
			for k, n := range names {
				on, err := s.select_(n.V.(string))
				if err != nil {
					return Value{}, err
				}
				for i := range on {
					if on[i] {
						which[i] = k
					}
				}
			}
			r := a.value("restart").V.(bool)
			return genLists{
				B: func(g []dsl.Gen[bool]) dsl.Gen[bool] { return sectionsOf(s, which, g, r) },
				I: func(g []dsl.Gen[int]) dsl.Gen[int] { return sectionsOf(s, which, g, r) },
				F: func(g []dsl.Gen[float64]) dsl.Gen[float64] { return sectionsOf(s, which, g, r) },
				C: func(g []dsl.Gen[[]float64]) dsl.Gen[[]float64] { return sectionsOf(s, which, g, r) },
				N: func(g []dsl.Gen[[]blocks.Note]) dsl.Gen[[]blocks.Note] { return sectionsOf(s, which, g, r) },
			}.of(a.value("gens"), "bySection")
		}})
}
