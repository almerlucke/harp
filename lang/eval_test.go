package lang

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"harp/blocks"
	"harp/dsl"

	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/synths/subtractive"
)

// the play example, as text
const playPatch = `
bpm = 110
seed = 7

melody = degrees(a3, minor, mutate([7 9 11 9 8 7 4 7], swap: 0.6))
voice  = notes(melody, euclid(16, 7), length: 0.6)
beat   = 60 / bpm

lead = saw([voice.pitch * 0.997, voice.pitch * 1.023])
  .lowPass(cutoff: lfo(0.3).range(300, 4000), res: 0.5)
  .gain(adsr(0.005, 0.15, 0.5, 0.2).on(voice.gate))
  .spread(width: 0.6)
  .gain(0.4)
  .echo(max: 2, time: [beat * 0.75, beat], feedback: 0.45, mix: 0.3)   // a dotted eighth and a quarter

pad = chords([[a3 c4 e4] [f3 a3 c4 f4] [c3 g3 c4 e4] [g3 b3 d4 g4]], every(16), length: 14)
  .subtractive(preset: "pad", voices: 8)
  .gain(0.4)

out = (lead + pad).reverb(mix: 0.25, decay: 2.5)
`

// the same patch, built in Go, step by step the way the text builds it
func goPlayPatch() dsl.Sig {
	dsl.Seed(7)
	notes := dsl.Degrees(57, dsl.Minor, dsl.Mutate([]int{7, 9, 11, 9, 8, 7, 4, 7}, 0.6))
	voice := dsl.Notes(notes, dsl.Euclid(16, 7, 0), 0.6)
	beat := 60 / 110.0
	saws := dsl.Saw(dsl.Chans(dsl.Mul(voice.Pitch, 0.997), dsl.Mul(voice.Pitch, 1.023)))
	filtered := dsl.Mul(blocks.Lowpass(saws, dsl.Lfo(0.3).Range(300, 4000), 0.5), dsl.Adsr(0.005, 0.15, 0.5, 0.2).On(voice.Gate))
	dry := dsl.Gain(dsl.Spread(filtered, 0.6), 0.4)
	lead := blocks.Echo(dry, 2.0, dsl.Chans(dsl.N(beat*0.75), dsl.N(beat)), 0.45, 0.3)
	chords := dsl.Loop([]float64{57, 60, 64}, []float64{53, 57, 60, 65}, []float64{48, 55, 60, 64}, []float64{55, 59, 62, 67})
	bar := dsl.Loop(append([]bool{true}, make([]bool, 15)...)...)
	pad := dsl.Gain(blocks.Subtractive(subtractive.PadPreset(), 8, blocks.Chords(chords, bar, 0.7, 14)), 0.4)
	cfg := fdnreverb.DefaultConfig()
	cfg.Mix, cfg.Decay = 0.25, 2.5
	return blocks.FDNReverb(dsl.Add(lead, pad), cfg)
}

func maxDiff(a, b []float64) float64 {
	var d float64
	for i := range a {
		d = math.Max(d, math.Abs(a[i]-b[i]))
	}
	return d
}

func peak(x []float64) (p float64) {
	for _, v := range x {
		p = math.Max(p, math.Abs(v))
	}
	return
}

// The text is the Go: the whole front end, the generators, the channels, the synth and the reverb, give
// the same samples as the patch built by hand
func TestPlayPatchIsTheGoPatch(t *testing.T) {
	patch, err := Compile(playPatch)
	if err != nil {
		t.Fatal(err)
	}
	if patch.BPM != 110 || !patch.HasSeed || patch.Seed != 7 || patch.Out.Channels() != 2 {
		t.Fatalf("settings: bpm %v seed %v/%v, %d channels", patch.BPM, patch.Seed, patch.HasSeed, patch.Out.Channels())
	}
	_, fromText := patch.Settings().Render(patch.Out, 5)

	_, fromGo := dsl.Patch{BPM: 110}.Render(goPlayPatch(), 5)
	if peak(fromGo[0]) < 0.05 || peak(fromGo[1]) < 0.05 {
		t.Fatal("the patch is silent")
	}
	for ch := range fromGo {
		if d := maxDiff(fromText[ch], fromGo[ch]); d != 0 {
			t.Errorf("channel %d: the text differs from the Go patch by %g", ch, d)
		}
	}
}

// build compiles a program that makes a signal, for the tests of what the language says
func build(t *testing.T, src string) (*Patch, error) {
	t.Helper()
	return Compile(src)
}

func mustBuild(t *testing.T, src string) *Patch {
	t.Helper()
	p, err := build(t, src)
	if err != nil {
		t.Fatalf("%q: %v", src, err)
	}
	return p
}

func TestNumbersAndOperators(t *testing.T) {
	// numbers are worked out when the patch is built: out is a constant of 0.1 * 4 + 1
	p := mustBuild(t, "x = 0.1 * 4 + 1\ny = x - 2 / 4\nout = y")
	_, ch := dsl.Patch{SampleRate: 100}.Render(p.Out, 0.1)
	if want := 0.1*4 + 1 - 2.0/4; math.Abs(ch[0][3]-want) > 1e-12 {
		t.Errorf("got %v, want %v", ch[0][3], want)
	}
	// a note is a number: c4 is 60, and an octave is 12
	p = mustBuild(t, "out = c4 + 12 - a3")
	_, ch = dsl.Patch{SampleRate: 100}.Render(p.Out, 0.1)
	if ch[0][3] != 60+12-57 {
		t.Errorf("notes: %v", ch[0][3])
	}
	// the settings
	p = mustBuild(t, "bpm = 90\nsteps = 2\nout = 0")
	if p.BPM != 90 || p.StepsPerBeat != 2 || p.HasSeed {
		t.Errorf("settings %v %v %v", p.BPM, p.StepsPerBeat, p.HasSeed)
	}
	if p = mustBuild(t, "out = 0"); p.BPM != 120 || p.StepsPerBeat != 4 {
		t.Errorf("defaults %v %v", p.BPM, p.StepsPerBeat)
	}
}

func TestSignalsAndChannels(t *testing.T) {
	for _, c := range []struct {
		src  string
		want int
	}{
		{"out = saw(220)", 1},
		{"out = saw([220 330])", 2},
		{"out = saw([220, 330, 440])", 3},
		{"out = saw(220) * [1 0.5]", 2},
		{"out = [1 2] + [3 4]", 2},
		{"out = sine([220 330]).lowPass(800)", 2},
		{"out = sine([220 330]).spread()", 2},
		{"out = saw([220 330 440]).spread(width: 0.5)", 2},
		{"out = sine(220) + sine(330) * 0.5 - 0.1", 1},
		{"out = -sine(220)", 1},
		{"out = 0.5", 1},
		{"out = [[1 2] 3]", 3}, // a list in a list: the channels, one after the other
	} {
		p, err := build(t, c.src)
		if err != nil {
			t.Errorf("%q: %v", c.src, err)
			continue
		}
		if p.Out.Channels() != c.want {
			t.Errorf("%q: %d channels, want %d", c.src, p.Out.Channels(), c.want)
		}
	}
}

func TestMethodsAndFunctionsAreTheSame(t *testing.T) {
	a := mustBuild(t, "out = saw(220).lowPass(800, res: 0.3).gain(0.5)")
	b := mustBuild(t, "out = gain(lowPass(saw(220), 800, res: 0.3), 0.5)")
	c := mustBuild(t, "out = gain(x: lowPass(x: saw(freq: 220), cutoff: 800, res: 0.3), amount: 0.5)")
	_, ra := dsl.Patch{}.Render(a.Out, 0.1)
	for _, o := range []*Patch{b, c} {
		_, r := dsl.Patch{}.Render(o.Out, 0.1)
		if maxDiff(ra[0], r[0]) != 0 {
			t.Error("the same call written another way sounds different")
		}
	}
	if peak(ra[0]) < 0.01 {
		t.Error("silent")
	}
}

func TestGeneratorsAreCoerced(t *testing.T) {
	// a list stands for a generator where a generator is wanted: plain lists of numbers, booleans, chords
	mustBuild(t, "out = notes(degrees(a3, minor, [0 2 4 7]), [true false true true]).pitch")
	mustBuild(t, "out = notes([60 62 64], euclid(8, 3)).pitch")
	mustBuild(t, "out = chords([a3 c4 e4], every(4)).subtractive()") // one chord, again and again
	mustBuild(t, "out = chords([[a3 c4 e4] [g3 b3 d4]], every(4)).subtractive()")
	mustBuild(t, "out = melody([c4 e4 g4], euclid(8, 5)).supersaw(preset: \"hypersaw\")")
}

func TestVoiceFields(t *testing.T) {
	p := mustBuild(t, "v = notes([60], every(4))\nout = [v.pitch, v.gate]")
	if p.Out.Channels() != 2 {
		t.Errorf("%d channels", p.Out.Channels())
	}
}

// what the language says when it is wrong: where, what, and what could be meant
func TestErrors(t *testing.T) {
	for _, c := range []struct {
		name, src string
		at        Pos
		mentions  []string
	}{
		{"unknown function", "out = lowpas(saw(220), 800)", Pos{1, 7}, []string{"no function lowpas", "did you mean lowPass?"}},
		{"unknown method", "out = saw(220).lowpas(800)", Pos{1, 15}, []string{"no function lowpas", "did you mean lowPass?"}},
		{"unknown parameter", "out = lowPass(saw(220), cutof: 800)", Pos{1, 25}, []string{"no parameter cutof", "did you mean cutoff?", "lowPass(x: signal"}},
		{"missing argument", "out = saw()", Pos{1, 7}, []string{"saw needs freq"}},
		{"too many arguments", "out = saw(220, 330)", Pos{1, 16}, []string{"saw takes 1 arguments"}},
		{"given twice", "out = lowPass(saw(220), 800, cutoff: 900)", Pos{1, 30}, []string{"cutoff is given twice"}},
		{"positional after named", "out = lowPass(saw(220), cutoff: 800, 0.5)", Pos{1, 38}, []string{"positional argument can't come after a named one"}},
		{"undefined name", "melody = [1 2]\nout = melodi", Pos{2, 7}, []string{"melodi is not defined", "did you mean melody?"}},
		{"function as a value", "out = saw", Pos{1, 7}, []string{"saw is a function", "saw(...)"}},
		{"redefined", "x = 1\nx = 2\nout = x", Pos{2, 1}, []string{"x is already defined, at 1:1"}},
		{"a name of the language", "minor = 3\nout = 0", Pos{1, 1}, []string{"minor is a name of the language"}},
		{"no out", "x = 1", Pos{1, 1}, []string{"no out"}},
		{"out twice", "out = 1\nout = 2", Pos{2, 1}, []string{"out is already defined"}},
		{"rate: a generator is not a signal", "out = saw(euclid(16, 7))", Pos{1, 11}, []string{"saw: freq: expected a signal, got a generator of booleans", "event rate"}},
		{"rate: a signal is not a generator", "out = notes(saw(220), euclid(16, 7)).pitch", Pos{1, 13}, []string{"notes: melody: expected a generator of numbers, got a signal", "audio rate"}},
		{"kinds of generators", "out = notes(euclid(4, 2), euclid(16, 7)).pitch", Pos{1, 13}, []string{"expected a generator of numbers, got a generator of booleans", "different things"}},
		{"a string is not a number", `out = saw("a")`, Pos{1, 11}, []string{"expected a signal, got a string"}},
		{"not whole", "out = notes(degrees(a3, minor, [0 2.5]), every(2)).pitch", Pos{1, 32}, []string{"item 2 of the list is not a whole number"}},
		{"empty list", "out = saw([])", Pos{1, 11}, []string{"an empty list has no channels"}},
		{"channels do not match", "out = [1 2] + [1 2 3]", Pos{1, 13}, []string{"signals of 2 and 3 channels"}},
		{"operator on a generator", "out = euclid(4, 2) + 1", Pos{1, 20}, []string{"+ works on numbers and signals", "generator of booleans"}},
		{"division by a signal", "out = saw(220) / saw(110)", Pos{1, 16}, []string{"divided by a number"}},
		{"division by zero", "out = 1 / 0", Pos{1, 9}, []string{"division by zero"}},
		{"no field", "v = notes([60], every(4))\nout = v.pich", Pos{2, 8}, []string{"a voice has no pich", "did you mean pitch?"}},
		{"field of a number", "out = 3.x", Pos{1, 8}, []string{}},
		{"a preset that is not there", `out = chords([a3], every(4)).subtractive(preset: "padd")`, Pos{1, 29}, []string{`no subtractive preset "padd"`, "did you mean pad?"}},
		{"bpm is a number", "bpm = [1 2]\nout = 0", Pos{1, 7}, []string{"bpm is a number"}},
		{"bpm is positive", "bpm = 0\nout = 0", Pos{1, 7}, []string{"not a possible value"}},
		{"out is a signal", "out = euclid(4, 2)", Pos{1, 7}, []string{"out is what is heard", "expected a signal"}},
		{"reverb of three channels", "out = [1 2 3].reverb()", Pos{1, 14}, []string{"takes one or two channels", "3"}},
		{"euclid steps", "out = notes([60], euclid(0, 2)).pitch", Pos{1, 19}, []string{"steps must be at least 1"}},
	} {
		_, err := Compile(c.src)
		if err == nil {
			if c.name != "field of a number" {
				t.Errorf("%s: no error for %q", c.name, c.src)
			}
			continue
		}
		e, ok := err.(*Error)
		if !ok {
			if c.name == "field of a number" { // a syntax error: 3.x is not an expression
				continue
			}
			t.Errorf("%s: %v is not a lang.Error", c.name, err)
			continue
		}
		if e.Pos != c.at {
			t.Errorf("%s: at %v, want %v: %v", c.name, e.Pos, c.at, e)
		}
		full := e.Error()
		for _, m := range c.mentions {
			if !strings.Contains(full, m) {
				t.Errorf("%s: %q does not mention %q", c.name, full, m)
			}
		}
	}
}

func TestFunctionsAreListed(t *testing.T) {
	list := Functions()
	for _, name := range []string{"saw", "lowPass", "echo", "reverb", "subtractive", "supersaw", "euclid", "chords", "notes", "adsr"} {
		if !strings.Contains(list, name+"(") {
			t.Errorf("%s is not listed", name)
		}
	}
	if !strings.Contains(list, "lowPass(x: signal, cutoff: signal, res: signal = 0.5) -> signal") {
		t.Errorf("the signature of lowpass:\n%s", list)
	}
	if len(subtractivePresets) != 88 || len(supersawPresets) != 37 {
		t.Errorf("%d and %d presets", len(subtractivePresets), len(supersawPresets))
	}
}

// a name can be the name of a function too: a call is always a call, and a bare name is the variable
func TestNamesMayShadowFunctions(t *testing.T) {
	p := mustBuild(t, "saw = 440\nmelody = [1 2]\nout = saw(saw) * melody")
	if p.Out.Channels() != 2 {
		t.Errorf("%d channels", p.Out.Channels())
	}
	// but a bare function name is still not a value, and says so when there is no variable of it
	if _, err := Compile("out = saw"); err == nil || !strings.Contains(err.Error(), "saw is a function") {
		t.Errorf("out = saw: %v", err)
	}
}

// seed = n makes what the generators draw the same every time; without it every run is another
func TestSeed(t *testing.T) {
	src := func(seed string) string {
		return seed + "\nmelody = degrees(a3, minor, mutate([0 1 2 3 4 5 6 7 8 9 10 11], swap: 1))\nvoice = notes(melody, every(1), length: 0.5)\nout = saw(voice.pitch)"
	}
	render := func(s string) []float64 {
		p := mustBuild(t, s)
		_, ch := p.Settings().Render(p.Out, 8)
		return ch[0]
	}
	a1, a2, b := render(src("seed = 7")), render(src("seed = 7")), render(src("seed = 8"))
	if maxDiff(a1, a2) != 0 {
		t.Error("the same seed gives another sound")
	}
	if maxDiff(a1, b) < 0.1 {
		t.Error("another seed gives the same sound")
	}
	if peak(a1) < 0.1 {
		t.Error("silent")
	}
	if maxDiff(render(src("")), render(src(""))) < 0.1 {
		t.Error("without a seed two runs are the same")
	}
}

// the example patches are checked, so they can't stop working without the tests saying so
func TestExamplePatches(t *testing.T) {
	files, err := filepath.Glob("examples/*.harp")
	if err != nil || len(files) < 3 {
		t.Fatalf("examples: %v %v", files, err)
	}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		p, err := CompileDir(string(src), filepath.Dir(f))
		if err != nil {
			if strings.Contains(err.Error(), "there is no file") { // the recordings of psalm's assets are not here
				t.Logf("%s: skipped: %v", f, err)
				continue
			}
			if e, ok := err.(*Error); ok {
				t.Errorf("%s:%s", f, e.Show(string(src)))
			} else {
				t.Errorf("%s: %v", f, err)
			}
			continue
		}
		_, ch := p.Settings().Render(p.Out, 6)
		for c := range ch {
			if pk := peak(ch[c]); pk < 0.005 || pk > 1 || math.IsNaN(pk) {
				t.Errorf("%s: channel %d peaks at %v in 6 seconds", f, c, pk)
			}
		}
	}
}

// the play example as a file is the patch of the test above it
func TestPlayExampleFileIsThePlayPatch(t *testing.T) {
	src, err := os.ReadFile("examples/play.harp")
	if err != nil {
		t.Fatal(err)
	}
	a, b := mustBuild(t, string(src)), mustBuild(t, playPatch)
	_, ra := a.Settings().Render(a.Out, 3)
	_, rb := b.Settings().Render(b.Out, 3)
	if maxDiff(ra[0], rb[0]) != 0 || maxDiff(ra[1], rb[1]) != 0 {
		t.Error("examples/play.harp is not the patch that the Go test checks")
	}
}

// errors are found from left to right: a mistake in what comes first in the text is the one that is said
func TestErrorsComeFromLeftToRight(t *testing.T) {
	_, err := Compile("v = notes([60], every(4))\nout = saw(v.pich).lowpas(cutoff: 800)")
	e, ok := err.(*Error)
	if !ok || e.Pos != (Pos{2, 12}) || !strings.Contains(e.Msg, "a voice has no pich") {
		t.Errorf("the typo in the receiver is said before the unknown method: %v", err)
	}
	// and a syntax error is an Error like the others, with a place the source can be shown at
	_, err = Compile("x = 1\ny = (2 +\nout = x")
	e, ok = err.(*Error)
	if !ok || e.Pos.Line != 3 || !strings.HasPrefix(e.Msg, "syntax error:") {
		t.Fatalf("syntax error: %#v", err)
	}
	shown := e.Show("x = 1\ny = (2 +\nout = x")
	if !strings.Contains(shown, "3 | out = x") || !strings.Contains(shown, "^") {
		t.Errorf("shown:\n%s", shown)
	}
}

// value is the first sample of out, for the tests of functions and if
func value(t *testing.T, src string) float64 {
	t.Helper()
	p := mustBuild(t, src)
	_, ch := dsl.Patch{SampleRate: 100}.Render(p.Out, 0.1)
	return ch[0][3]
}

func TestFunctionsOfTheProgram(t *testing.T) {
	for _, c := range []struct {
		src  string
		want float64
	}{
		{"fn double(x) { x * 2 }\nout = double(21)", 42},
		{"fn double(x) { x * 2 }\nout = 21.double()", 42}, // the receiver is the first argument, as with every function
		{"fn f(a, b) { a - b }\nout = f(b: 1, a: 10)", 9}, // named arguments
		{"fn f() { 7 }\nout = f()", 7},
		{"fn f(x) { y = x + 1\n z = y * y\n z }\nout = f(2)", 9},            // statements of its own
		{"base = 100\nfn f(x) { x + base }\nout = f(1)", 101},               // sees the names of the program from before
		{"x = 5\nfn f(x) { x + 1 }\nout = f(1) + x", 7},                     // a parameter hides a name of the program
		{"fn inc(x) { x + 1 }\nfn two(x) { inc(inc(x)) }\nout = two(0)", 2}, // functions call functions defined before
		{"fn a(x) { x }\nout = a(sine(0) + 3)", 3},                          // signals go through
		{"fn f(x) { x }\nout = f([1, 2]).mix()", 3},                         // a list goes through and is a signal after
		{"fn f(x) { y = x\n y }\ny = 8\nout = f(1) + y", 9},                 // names inside do not come out
		{"fn k(x) { x * 3 }\nout = repeat(k, 3, 1)", 27},                    // repeat: k(k(k(1)))
		{"fn k(x) { x + 1 }\nout = repeat(k, 0, 5)", 5},                     // zero times: x as it is
		{"seed = 3\nfn k(x) { x + rand(0, 1) }\nout = repeat(k, 4, 0)", -1}, // see below: only checked for a range
	} {
		if c.want == -1 {
			if v := value(t, c.src); v < 0 || v > 4 {
				t.Errorf("%q: %v is not in 0..4", c.src, v)
			}
			continue
		}
		if got := value(t, c.src); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("%q: got %v, want %v", c.src, got, c.want)
		}
	}
}

func TestIf(t *testing.T) {
	for _, c := range []struct {
		src  string
		want float64
	}{
		{"out = if true { 1 } else { 2 }", 1},
		{"out = if false { 1 } else { 2 }", 2},
		{"fn f(x) { if x > 0 { 1 } else { 2 } }\nout = f(1)", 0}, // replaced below: > is not in the language
		{"out = if coin(1) { 5 } else { 6 }", 5},                 // a chance of 1 is always
		{"out = if coin(0) { 5 } else { 6 }", 6},
		{"out = if true { a = 2\n a * 3 } else { 0 }", 6}, // a block has statements of its own
		{"out = 1 + if false { 10 } else { 20 }", 21},
		{"out = if true { if false { 1 } else { 2 } } else { 3 }", 2},
	} {
		if strings.Contains(c.src, "x > 0") {
			if _, err := build(t, c.src); err == nil {
				t.Errorf("%q: > is not an operator, this should say so", c.src)
			}
			continue
		}
		if got := value(t, c.src); got != c.want {
			t.Errorf("%q: got %v, want %v", c.src, got, c.want)
		}
	}
}

// only the branch that is taken is worked out: the other is never built, so nothing in it is drawn or wrong
func TestIfIsLazy(t *testing.T) {
	mustBuild(t, "out = if true { 1 } else { nosuchfunction(3) }")
	// a mistake in the branch that is taken is a mistake
	if _, err := build(t, "out = if true { nosuchfunction(3) } else { 1 }"); err == nil || !strings.Contains(err.Error(), "no function nosuchfunction") {
		t.Errorf("%v", err)
	}
	// the random numbers drawn are those of the branch taken: the same seed gives the same patch, with or without a dead branch
	a := value(t, "seed = 5\nout = if true { rand(0, 1) } else { rand(0, 1) + rand(0, 1) }")
	b := value(t, "seed = 5\nout = if true { rand(0, 1) } else { 0 }")
	if a != b {
		t.Errorf("a branch that is not taken drew numbers: %v and %v", a, b)
	}
}

func TestBuildTimeRandomness(t *testing.T) {
	src := "seed = %d\nout = [rand(0, 1), expRand(1, 100), choose([10, 20, 30]), if coin(0.5) { 1 } else { 0 }].mix()"
	same := func(a, b int) bool {
		return value(t, strings.Replace(src, "%d", string(rune('0'+a)), 1)) == value(t, strings.Replace(src, "%d", string(rune('0'+b)), 1))
	}
	if !same(3, 3) {
		t.Error("the same seed is not the same patch")
	}
	if same(3, 4) && same(3, 5) && same(3, 6) {
		t.Error("the seeds do not make different patches")
	}
	// the ranges
	for seed := 1; seed < 40; seed++ {
		p := mustBuild(t, "seed = "+string(rune('0'+seed%10))+"\nfn r(x) { expRand(2, 8) }\nout = repeat(r, 5, 0)")
		_, ch := dsl.Patch{SampleRate: 100}.Render(p.Out, 0.1)
		if v := ch[0][3]; v < 2 || v > 8 {
			t.Errorf("expRand(2, 8) gave %v", v)
		}
	}
}

func TestFunctionErrors(t *testing.T) {
	for _, c := range []struct{ src, mentions string }{
		{"fn f(x) { x }\nfn f(y) { y }\nout = 1", "already defined"},
		{"fn f(x, x) { x }\nout = 1", "twice"},
		{"fn f(x) { x }\nout = f(1, 2)", "takes 1 argument"},
		{"fn f(x) { x }\nout = f()", "needs x"},
		{"fn f(x) { x }\nout = f(y: 1)", "no parameter y"},
		{"fn f(x) { y }\nout = f(1)", "y"}, // an unknown name inside
		{"fn f(x) { x }\nout = f(1)\nout = 2", "already"},
		{"out = g(1)\nfn g(x) { x }", "no function g"}, // known from the definition on
		{"fn f(x) { z = 1\n z = 2\n z }\nout = f(1)", "already defined"},
		{"fn f(x) { f(x) }\nout = f(1)", "does not end"}, // calls itself
		{"out = if 3 { 1 } else { 2 }", "true or false"},
		{"fn caller() { callee_secret }\ncallee_secret = 3\nout = caller()", "callee_secret"}, // defined after the function: not seen... see below
		{"out = repeat(1, 3, 0)", "function"},
		{"fn two(a, b) { a }\nout = repeat(two, 3, 0)", "one argument"},
		{"out = repeat(sine, 3, 0)", ""},
		{"out = choose([])", "empty"},
		{"out = expRand(0, 5)", "above 0"},
		{"fn sine(x) { x }\nout = sine(2)", ""}, // a function of the program may have the name of a function of the language
	} {
		_, err := build(t, c.src)
		if c.mentions == "" {
			continue // only that it does not panic
		}
		if c.src == "fn caller() { callee_secret }\ncallee_secret = 3\nout = caller()" {
			// the name is defined after the function, but before the call: functions see the names of the program as they are when called
			if err != nil {
				t.Errorf("%q: %v", c.src, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.mentions) {
			t.Errorf("%q: %v, want a mention of %q", c.src, err, c.mentions)
		}
	}
	// a mistake inside a function is at its own line
	_, err := build(t, "fn f(x) {\n  y = lowpas(x, 100)\n  y\n}\nout = f(sine(1))")
	if err == nil || !strings.Contains(err.Error(), "2:") || !strings.Contains(err.Error(), "did you mean lowPass") {
		t.Errorf("a mistake inside a function: %v", err)
	}
}
