package lang

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"harp/dsl"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/arpeggio"
	"github.com/almerlucke/genny/chord"
	"github.com/almerlucke/genny/counterpoint"
	"github.com/almerlucke/genny/progression"
	"github.com/almerlucke/genny/row"
	"github.com/almerlucke/genny/scale"
	"github.com/almerlucke/genny/transform"
)

// Pitch: more scales, tunings, chords and how they are led, arpeggios, progressions, tone rows and counterpoint

func init() {
	for name, steps := range map[string][]float64{
		"phrygian": scale.Phrygian, "lydian": scale.Lydian, "mixolydian": scale.Mixolydian, "locrian": scale.Locrian,
		"harmonicminor": scale.HarmonicMinor, "melodicminor": scale.MelodicMinor, "majorpentatonic": scale.MajorPentatonic,
		"minorpentatonic": scale.MinorPentatonic, "blues": scale.Blues, "wholetone": scale.WholeTone, "diminished": scale.Diminished,
		"chromatic": scale.Chromatic, "mode1": scale.Mode1, "mode2": scale.Mode2, "mode3": scale.Mode3, "mode4": scale.Mode4,
		"mode5": scale.Mode5, "mode6": scale.Mode6, "mode7": scale.Mode7,
	} {
		consts[name] = Value{TScale, steps}
	}
}

var chordKinds = map[string][]float64{
	"major": chord.Major, "minor": chord.Minor, "dim": chord.Diminished, "aug": chord.Augmented, "sus2": chord.Sus2, "sus4": chord.Sus4,
	"major6": chord.Major6, "minor6": chord.Minor6, "major7": chord.Major7, "minor7": chord.Minor7, "dom7": chord.Dominant7,
	"minmaj7": chord.MinorMajor7, "halfdim": chord.HalfDiminished, "dim7": chord.Diminished7, "add9": chord.Add9,
	"major9": chord.Major9, "minor9": chord.Minor9, "dom9": chord.Dominant9,
}

var progressions = map[string]progression.Table{"pop": progression.Pop, "classical": progression.Classical, "jazz": progression.Jazz, "blues": progression.Blues}
var arpModes = map[string]arpeggio.Mode{"up": arpeggio.Up, "converge": arpeggio.Converge, "alternate": arpeggio.Alternate, "random": arpeggio.Random}
var arpMirrors = map[string]arpeggio.MirrorMode{"none": arpeggio.None, "exclusive": arpeggio.Exclusive, "inclusive": arpeggio.Inclusive}
var rowForms = map[string]row.Op{"prime": row.Prime, "retro": row.Retro, "invert": row.Invert, "retroinvert": row.RetroInvert}

func scaleSteps(a Args, name string) []float64 { return a.value(name).V.([]float64) }

func theScale(a Args) *scale.Scale {
	return scale.New(a.Num("root"), scaleSteps(a, "scale"))
}

func numList(xs []float64) Value {
	out := make([]Value, len(xs))
	for i, x := range xs {
		out[i] = numV(x)
	}
	return listV(out)
}

// degreeChords plays the chord on every scale degree that a generator gives, built of thirds of the scale
type degreeChords struct {
	s    *scale.Scale
	g    dsl.Gen[int]
	size int
}

func (d *degreeChords) Generate() []float64 { return chord.Diatonic(d.s, d.g.Generate(), d.size) }
func (d *degreeChords) Continuous() bool    { return d.g.Continuous() }
func (d *degreeChords) Done() bool          { return d.g.Done() }
func (d *degreeChords) Reset()              { d.g.Reset() }

// snapGen moves the numbers of a generator to the nearest note of a scale
type snapGen struct {
	s *scale.Scale
	g dsl.Gen[float64]
}

func (d *snapGen) Generate() float64 { return d.s.Quantize(d.g.Generate()) }
func (d *snapGen) Continuous() bool  { return d.g.Continuous() }
func (d *snapGen) Done() bool        { return d.g.Done() }
func (d *snapGen) Reset()            { d.g.Reset() }

func init() {
	add(&Func{Name: "scale", Result: TScale, Doc: "a scale of your own: the steps, in semitones above the root, from 0 and rising, within an octave (scale([0 2 3 7 8]) is a scale of 5 notes; fractions are microtones: scale([0 1.5 3 4.5 6 7.5 9 10.5]))",
		Params: []Param{{"steps", ListOf(TNum), nil}},
		Call: func(a Args) (Value, error) {
			x, err := numbersOf(a.value("steps"), "steps")
			if err != nil {
				return Value{}, err
			}
			if x[0] != 0 {
				return Value{}, fmt.Errorf("the first step is 0 (the root)")
			}
			for i, v := range x {
				if v < 0 || v >= 12 || math.IsNaN(v) || (i > 0 && v <= x[i-1]) {
					return Value{}, fmt.Errorf("the steps rise, from 0 up to below 12 (step %d is %g)", i+1, v)
				}
			}
			return Value{TScale, x}, nil
		}})
	add(&Func{Name: "equal", Result: TScale, Doc: "a scale that divides the octave in n equal steps: equal(12) is every semitone, equal(24) quarter tones, equal(19) and equal(31) are popular tunings. The pitches are in fractions of a semitone",
		Params: []Param{{"n", TNum, nil}},
		Call: func(a Args) (Value, error) {
			n := a.Int("n")
			if n < 1 || n > 96 {
				return Value{}, fmt.Errorf("n is from 1 to 96")
			}
			x := make([]float64, n)
			for i := range x {
				x[i] = 12 * float64(i) / float64(n)
			}
			return Value{TScale, x}, nil
		}})
	add(&Func{Name: "just", Result: TScale, Doc: "a scale from frequency ratios (just intonation): just([1, 9/8, 5/4, 4/3, 3/2, 5/3, 15/8]) written with the ratios as numbers (1.125, 1.25, ...); the first is 1, all are below 2",
		Params: []Param{{"ratios", ListOf(TNum), nil}},
		Call: func(a Args) (Value, error) {
			r, err := numbersOf(a.value("ratios"), "ratios")
			if err != nil {
				return Value{}, err
			}
			for i, v := range r {
				if !(v >= 1) || v >= 2 || (i > 0 && v <= r[i-1]) {
					return Value{}, fmt.Errorf("the ratios rise from 1 to below 2 (ratio %d is %g)", i+1, v)
				}
			}
			if r[0] != 1 {
				return Value{}, fmt.Errorf("the first ratio is 1 (the root)")
			}
			return Value{TScale, scale.Semitones(r...)}, nil
		}})
	add(&Func{Name: "stepwise", Result: TGenFloat, Doc: "a melody that walks over the degrees of a scale, mostly a step up or down, now and then a leap: it stays between the degrees lo and hi (bouncing off the edges), takes steps of at most step degrees, and with the chance leap a jump of up to maxleap degrees",
		Params: []Param{{"root", TNum, nil}, {"scale", TScale, nil}, {"start", TNum, def(0)}, {"lo", TNum, def(-7)}, {"hi", TNum, def(7)}, {"step", TNum, def(1)}, {"leap", TNum, def(0.1)}, {"maxleap", TNum, def(4)}},
		Call: func(a Args) (Value, error) {
			lo, hi, st := a.Int("lo"), a.Int("hi"), a.Int("start")
			if lo > hi || st < lo || st > hi {
				return Value{}, fmt.Errorf("lo is at most hi, and start is between them")
			}
			if a.Int("step") < 1 || a.Int("maxleap") < a.Int("step") {
				return Value{}, fmt.Errorf("step is 1 or more, maxleap at least step")
			}
			return Value{TGenFloat, scale.NewWalk(theScale(a), st, lo, hi, a.Int("step"), a.Num("leap"), a.Int("maxleap"))}, nil
		}})
	add(&Func{Name: "snap", Result: TGenFloat, Doc: "the numbers of a generator moved to the nearest note of a scale on a root: snap(remap(drift(), -1, 1, 48, 72), a2, minor) is a melody that wanders in A minor",
		Params: []Param{{"gen", TGenFloat, nil}, {"root", TNum, nil}, {"scale", TScale, nil}},
		Call: func(a Args) (Value, error) {
			return Value{TGenFloat, &snapGen{theScale(a), a.value("gen").V.(dsl.Gen[float64])}}, nil
		}})
	add(&Func{Name: "chord", Result: ListOf(TNum), Doc: "the notes of a chord, as a list: chord(a3, \"minor7\"); kind is " + names(chordKinds) + "; inversion turns it that many times (the lowest note goes an octave up), open spreads it over two octaves",
		Params: []Param{{"root", TNum, nil}, {"kind", TStr, defStr("major")}, {"inversion", TNum, def(0)}, {"open", TBool, defBool(false)}},
		Call: func(a Args) (Value, error) {
			iv, ok := chordKinds[a.Str("kind")]
			if !ok {
				return Value{}, fmt.Errorf("there is no chord kind %q: %s%s", a.Str("kind"), names(chordKinds), didYouMean(a.Str("kind"), presetNames(chordKinds)))
			}
			c := chord.New(a.Num("root"), iv)
			if n := a.Int("inversion"); n != 0 {
				c = c.Invert(n)
			}
			if a.value("open").V.(bool) {
				c = c.Open()
			}
			return numList(c), nil
		}})
	add(&Func{Name: "triads", Result: TGenChord, Doc: "the chords on the scale degrees that a generator gives, built of thirds of the scale: triads(a2, minor, [0 5 3 4]) is the chords i, vi, iv, v (size: 4 for seventh chords). The generator can be a progression",
		Params: []Param{{"root", TNum, nil}, {"scale", TScale, nil}, {"degrees", TGenInt, nil}, {"size", TNum, def(3)}},
		Call: func(a Args) (Value, error) {
			if s := a.Int("size"); s < 2 || s > 7 {
				return Value{}, fmt.Errorf("size is from 2 to 7 notes")
			}
			return Value{TGenChord, &degreeChords{theScale(a), a.value("degrees").V.(dsl.Gen[int]), a.Int("size")}}, nil
		}})
	add(&Func{Name: "voicelead", Result: TGenChord, Doc: "chords with smooth voice leading: each chord is turned and moved by octaves to be nearest the one before it, and kept between low and high",
		Params: []Param{{"chords", TGenChord, nil}, {"low", TNum, def(48)}, {"high", TNum, def(72)}},
		Call: func(a Args) (Value, error) {
			return Value{TGenChord, transform.NewWithTransformer(a.value("chords").V.(dsl.Gen[[]float64]), chord.NewVoiceLeader(a.Num("low"), a.Num("high")))}, nil
		}})
	add(&Func{Name: "arp", Result: TGenFloat, Doc: "an arpeggio of a list of notes (a chord): mode is " + names(arpModes) + "; mirror (" + names(arpMirrors) + ") plays it back reversed after the first time; reverse starts from the top",
		Params: []Param{{"notes", ListOf(TNum), nil}, {"mode", TStr, defStr("up")}, {"mirror", TStr, defStr("none")}, {"reverse", TBool, defBool(false)}},
		Call: func(a Args) (Value, error) {
			x, err := numbersOf(a.value("notes"), "notes")
			if err != nil {
				return Value{}, err
			}
			mode, ok := arpModes[a.Str("mode")]
			if !ok {
				return Value{}, fmt.Errorf("mode is one of %s%s", names(arpModes), didYouMean(a.Str("mode"), presetNames(arpModes)))
			}
			mirror, ok := arpMirrors[a.Str("mirror")]
			if !ok {
				return Value{}, fmt.Errorf("mirror is one of %s", names(arpMirrors))
			}
			return Value{TGenFloat, arpeggio.NewLoop(x, mode, mirror, a.value("reverse").V.(bool))}, nil
		}})
	add(&Func{Name: "progression", Result: TGenInt, Doc: "chord changes by chance, as scale degrees (0 is I): a table of what can follow what, in the style of " + names(progressions) + ". It starts on degree start. Use it with triads: triads(a2, minor, progression(\"pop\"))",
		Params: []Param{{"style", TStr, defStr("pop")}, {"start", TNum, def(0)}},
		Call: func(a Args) (Value, error) {
			t, ok := progressions[a.Str("style")]
			if !ok {
				return Value{}, fmt.Errorf("style is one of %s%s", names(progressions), didYouMean(a.Str("style"), presetNames(progressions)))
			}
			if _, ok := t[a.Int("start")]; !ok {
				return Value{}, fmt.Errorf("start is a degree of the table (0 to 6)")
			}
			return Value{TGenInt, progression.New(t, a.Int("start"))}, nil
		}})
	add(&Func{Name: "tonerow", Result: ListOf(TNum), Doc: "a twelve tone row: the pitch classes 0 to 11 in a random order (made when the patch is built, so seed it). Use it as notes, or through rowform",
		Params: nil,
		Call:   func(a Args) (Value, error) { return numList(row.Random(12)), nil }})
	add(&Func{Name: "rowform", Result: ListOf(TNum), Doc: "a form of a tone row (or any list of notes): form is " + names(rowForms) + ", moved up transpose semitones",
		Params: []Param{{"row", ListOf(TNum), nil}, {"form", TStr, defStr("prime")}, {"transpose", TNum, def(0)}},
		Call: func(a Args) (Value, error) {
			x, err := numbersOf(a.value("row"), "row")
			if err != nil {
				return Value{}, err
			}
			op, ok := rowForms[a.Str("form")]
			if !ok {
				return Value{}, fmt.Errorf("form is one of %s%s", names(rowForms), didYouMean(a.Str("form"), presetNames(rowForms)))
			}
			return numList(row.Apply(x, row.Form{Op: op, Transpose: a.Num("transpose")})), nil
		}})
	add(&Func{Name: "counterpoint", Result: ListOf(TNum), Doc: "a line to go with another, written by the rules of strict counterpoint (note against note): the cantus is a list of notes (whole numbers), the line above it (or below with below: true) is consonant, moves by step and small leaps, and begins and ends on a perfect consonance. It is made when the patch is built, and by chance: seed it. If no line keeps all the rules the best one is given",
		Params: []Param{{"cantus", ListOf(TNum), nil}, {"below", TBool, defBool(false)}, {"maxleap", TNum, def(9)}},
		Call: func(a Args) (Value, error) {
			x, err := numbersOf(a.value("cantus"), "cantus")
			if err != nil {
				return Value{}, err
			}
			cf := make([]int, len(x))
			for i, v := range x {
				if !isWhole(v) {
					return Value{}, fmt.Errorf("the cantus is whole notes: item %d is %g", i+1, v)
				}
				cf[i] = int(v)
			}
			cfg := counterpoint.Config{MaxLeap: a.Int("maxleap")}
			if a.value("below").V.(bool) {
				cfg.Side = counterpoint.Below
			}
			line, err := counterpoint.Generate(cf, cfg)
			if err != nil && !errors.Is(err, counterpoint.ErrNoSolution) {
				return Value{}, err
			}
			out := make([]float64, len(line))
			for i, v := range line {
				out[i] = float64(v)
			}
			return numList(out), nil
		}})
}

var _ = sort.Strings
var _ genny.Generator[float64] = (*snapGen)(nil)
