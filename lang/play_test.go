package lang

import (
	"math"
	"strings"
	"testing"
)

func renderPatch(t *testing.T, src string, seconds float64) [][]float64 {
	t.Helper()
	p := mustBuild(t, src)
	_, ch := p.Settings().Render(p.Out, seconds)
	return ch
}

func TestEnvBuiltin(t *testing.T) {
	// the perc of the kick: 1 ms up, 80 ms down, curve -1: SuperCollider's values at samples 1, 2 and 100 (0.0354688242, 0.0701424181, 0.973799884)
	ch := renderPatch(t, "out = env(1, [0 1 0], [0.001 0.08], -1)", 0.1)
	for i, w := range map[int]float64{1: 0.0354688242, 2: 0.0701424181, 100: 0.973799884, 441: 0.830539167} {
		if math.Abs(ch[0][i]-w) > 1e-6 {
			t.Errorf("env at %d: %v, want %v", i, ch[0][i], w)
		}
	}
	// a list of curves, a shape, the control rate
	ch = renderPatch(t, "out = env(1, [-5 6 0], [0.1 1.7], [0 -4], control: true)", 0.2)
	if ch[0][0] != -5 || ch[0][63] != -5 || ch[0][64] == -5 {
		t.Errorf("control rate: %v %v %v", ch[0][0], ch[0][63], ch[0][64])
	}
	ch = renderPatch(t, "out = env(1, [1 0.8], [0.3], shape: \"exp\", control: true)", 0.5)
	if want := math.Pow(0.8, 0.15/0.3); math.Abs(ch[0][int(0.15*44100)/64*64]-want) > 1e-3 {
		t.Errorf("exp: %v, want %v", ch[0][int(0.15*44100)/64*64], want)
	}
	for src, mention := range map[string]string{
		"out = env(1, [0 1 0], [0.1])":                  "times",
		"out = env(1, [0], [])":                         "times",
		"out = env(1, [0 1], [0.1], [0 1])":             "curve",
		"out = env(1, [0 1], [0.1], shape: \"bounce\")": "shape",
		"out = env(1, [0 1 0], [0.1 0.1], release: 5)":  "release",
		"out = env(1, \"a\", [0.1])":                    "levels",
		"out = env(1, [0 1 [2]], [0.1 0.1])":            "not a number",
	} {
		if _, err := build(t, src); err == nil || !strings.Contains(err.Error(), mention) {
			t.Errorf("%s: %v, want a mention of %q", src, err, mention)
		}
	}
}

func TestPlay(t *testing.T) {
	// two notes in turn, on two voices: the gates, and a field that counts
	src := `
bpm = 210
steps = 4
fn inst(gate, n) { gate * n }
out = play(inst, dur: [1 0.5], legato: 0.5, fields: [[1 2 3]], voices: 2)
`
	ch := renderPatch(t, src, 4*0.2857142857142857)
	x := ch[0]
	step := 3150
	// the notes start at 0, 1 beat (4 steps), 1.5 beats, 2.5 beats...: voice 0 has the first (n = 1), voice 1 the second (n = 2), voice 0 the third (n = 3)
	at := func(s int) float64 { return x[s] }
	if at(100) != 1 || at(4*step+100) != 2 || at(6*step+100) != 3 {
		t.Errorf("the notes: %v %v %v, want 1 2 3", at(100), at(4*step+100), at(6*step+100))
	}
	// the gate is high for half of a note: 2 steps of the first (to 6300 -> the block at 6272), low then
	if at(2*step-100) != 1 || at(2*step+100) != 0 {
		t.Errorf("legato: %v %v", at(2*step-100), at(2*step+100))
	}
	// sustain in beats
	ch = renderPatch(t, "bpm = 210\nsteps = 4\nfn inst(gate) { gate }\nout = play(inst, dur: [4], sustain: 2)", 4*0.2857142857142857)
	beat := 0.2857142857142857 * 44100
	if ch[0][int(1.9*beat)] != 1 || ch[0][int(2.1*beat)] != 0 {
		t.Error("sustain: 2 beats")
	}
	// an instrument that makes two channels
	ch = renderPatch(t, "bpm = 210\nfn inst(gate) { [gate, gate * 2] }\nout = play(inst, dur: [1])", 0.1)
	if len(ch) != 2 || ch[1][100] != 2*ch[0][100] {
		t.Error("stereo instrument")
	}
}

func TestPlayErrors(t *testing.T) {
	for _, c := range []struct{ src, mention string }{
		{"fn i(g) { g }\nout = play(i, dur: [0.3])", "whole number of steps"},
		{"fn i(g) { g }\nout = play(i, dur: [])", "dur"},
		{"fn i(g) { g }\nout = play(i, dur: [1], fields: [[1]])", "takes the gate and 0 numbers"},
		{"fn i(g, a) { g }\nout = play(i, dur: [1])", "takes the gate and 1 numbers"},
		{"fn i(g) { g }\nout = play(i, dur: [1], voices: 0)", "voices"},
		{"out = play(sine, dur: [1])", "function"},
		{"fn i(g) { 1 }\nfn j(g) { g }\nout = play(i, dur: [1])", ""},
		{"fn i(g) { [g, g] }\nout = play(i, dur: [1], voices: 2) + sine(100)", ""}, // two channels and one go together
		{"fn i(g) { true }\nout = play(i, dur: [1])", "boolean"},
		{"fn i(g, a) { g }\nout = play(i, dur: [1], fields: [[true]])", "fields"},
	} {
		_, err := build(t, c.src)
		if c.mention == "" {
			if err != nil {
				t.Errorf("%q: %v", c.src, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.mention) {
			t.Errorf("%q: %v, want a mention of %q", c.src, err, c.mention)
		}
	}
}

// every note is a new synth: what the instrument makes starts again with the rise of the gate, and every voice has randomness of its own
func TestPlayStartsEveryNoteAnew(t *testing.T) {
	src := `
bpm = 210
steps = 4
fn inst(gate) { scsin(100, 0) * gate }
out = play(inst, dur: [4], legato: 0.5)
`
	x := renderPatch(t, src, 2)[0]
	beat := 12600
	// the sine of every note starts at phase 0: the same first samples, whatever the time of the note
	second := 4*beat - 4*beat%64 // the note starts in the block its time falls in
	a, b := x[1:20], x[second+1:second+20]
	if math.Abs(a[5]) < 1e-3 {
		t.Fatal("no sound")
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > 1e-9 {
			t.Fatalf("the second note does not start at phase 0: sample %d is %v and %v", i, a[i], b[i])
		}
	}
	// random choices: made for every voice
	src = "seed = 3\nbpm = 210\nfn inst(gate) { gate * rand(0, 1) }\nout = play(inst, dur: [1], voices: 2)"
	y := renderPatch(t, src, 0.3)[0]
	v0, v1 := y[100], y[12600+100]
	if v0 == v1 || v0 == 0 || v1 == 0 {
		t.Errorf("the voices have the same random number: %v %v", v0, v1)
	}
}

func TestFillGeomDbamp(t *testing.T) {
	if v := value(t, "out = dbamp(-20)"); math.Abs(v-0.1) > 1e-12 {
		t.Errorf("dbamp(-20) is %v", v)
	}
	if v := value(t, "out = dbamp(0)"); v != 1 {
		t.Errorf("dbamp(0) is %v", v)
	}
	channels := func(src string) []float64 {
		ch := renderPatch(t, src, 0.1)
		out := make([]float64, len(ch))
		for i := range ch {
			out[i] = ch[i][3]
		}
		return out
	}
	for src, want := range map[string][]float64{
		"out = geom(5, 100, 1600)":                         {100, 200, 400, 800, 1600},
		"out = geom(1, 7, 9)":                              {7},
		"fn sq(i) { i * i }\nout = fill(sq, 4)":            {0, 1, 4, 9},
		"fn plus(i, k) { i + k }\nout = fill(plus, 3, 10)": {10, 11, 12},
	} {
		got := channels(src)
		if len(got) != len(want) {
			t.Errorf("%q: %v", src, got)
			continue
		}
		for i := range want {
			if math.Abs(got[i]-want[i]) > 1e-9*math.Max(1, want[i]) {
				t.Errorf("%q: %v, want %v", src, got, want)
				break
			}
		}
	}
	for src, mention := range map[string]string{
		"out = geom(0, 1, 2)":                   "n is 1 or more",
		"out = geom(3, 0, 2)":                   "above 0",
		"fn f(a, b, c) { a }\nout = fill(f, 3)": "second argument",
		"fn f(a) { a }\nout = fill(f, 0)":       "1 to 1000",
		"out = fill(sine, 3)":                   "function",
	} {
		if _, err := build(t, src); err == nil || !strings.Contains(err.Error(), mention) {
			t.Errorf("%q: %v, want a mention of %q", src, err, mention)
		}
	}
}

// generators drive the notes: a rhythm of booleans, a generator of numbers for a field
func TestPlayWithGenerators(t *testing.T) {
	src := `
bpm = 210
steps = 4
fn inst(gate, n) { gate * n }
out = play(inst, rhythm: [false true false false true], legato: 1, fields: [degrees(60, major, [0 1 2])], voices: 1)
`
	// one beat is 12600 samples, a step 3150: the notes are at steps 1 and 4 (and then 6, 9 ... the rhythm is 5 steps), 60, 62, 64
	x := renderPatch(t, src, 10*3150.0/44100)[0]
	if x[3150-100] != 0 || x[3150+100] != 60 || x[4*3150+100] != 62 || x[6*3150+100] != 64 {
		t.Errorf("notes: %v %v %v %v", x[3150-100], x[3150+100], x[4*3150+100], x[6*3150+100])
	}
	// a generator of beats for the lengths, a generator of whole numbers as a field
	src = `
bpm = 210
steps = 4
fn inst(gate, n) { gate * n }
out = play(inst, dur: [1], fields: [mutate([5 6 7], swap: 0)])
`
	x = renderPatch(t, src, 0.6)[0]
	if x[100] != 5 || x[12600+100] != 6 {
		t.Errorf("mutate as a field: %v %v", x[100], x[12600+100])
	}
	for src, mention := range map[string]string{
		"fn i(g) { g }\nout = play(i)":                                           "when are the notes",
		"fn i(g) { g }\nout = play(i, dur: [1], rhythm: euclid(4, 1))":           "give one",
		"fn i(g) { g }\nout = play(i, rhythm: [1 2])":                            "rhythm",
		"fn i(g, a) { g }\nout = play(i, rhythm: euclid(4, 1), fields: [\"x\"])": "fields",
	} {
		if _, err := build(t, src); err == nil || !strings.Contains(err.Error(), mention) {
			t.Errorf("%q: %v, want a mention of %q", src, err, mention)
		}
	}
}

// a generator is a signal where a signal is wanted: it steps, a new value every step
func TestGeneratorsAsParameters(t *testing.T) {
	// 100 bpm, 4 steps a beat: a step is 6615 samples
	src := "bpm = 100\nsteps = 4\nout = [1 2 3].gain(1) * 0 + seq([10 20 30])"
	ch := renderPatch(t, src, 4*6615.0/44100)
	// three channels (the list) plus the seq on each: the values are 10, 20, 30, 10
	for i, w := range []float64{10, 20, 30, 10} {
		if got := ch[0][i*6615+10]; got != w {
			t.Errorf("seq at step %d: %v, want %v", i, got, w)
		}
	}
	// a generator as it is: degrees, mutate, euclid, in arithmetic and as an argument
	x := renderPatch(t, "bpm = 100\nsteps = 4\nout = degrees(60, major, [0 2 4]) + 1", 3*6615.0/44100)[0]
	if x[10] != 61 || x[6615+10] != 65 || x[2*6615+10] != 68 {
		t.Errorf("degrees: %v %v %v", x[10], x[6615+10], x[2*6615+10])
	}
	x = renderPatch(t, "bpm = 100\nsteps = 4\nout = mutate([3 4], swap: 0) * 2", 2*6615.0/44100)[0]
	if x[10] != 6 || x[6615+10] != 8 {
		t.Errorf("mutate: %v %v", x[10], x[6615+10])
	}
	// glide moves to the new value instead of jumping: halfway through 40 ms it is between, and later it is there
	x = renderPatch(t, "bpm = 100\nsteps = 4\nout = seq([0 1], glide: 40)", 2*6615.0/44100)[0]
	at := 6615 + int(0.04*44100) // one time constant after the jump: about two thirds
	if x[6615-1] != 0 || x[6615+5] <= 0 || x[6615+5] > 0.1 || x[at] < 0.55 || x[at] > 0.75 || x[2*6615-1] < 0.95 {
		t.Errorf("glide: %v %v %v %v %v", x[6615-1], x[6615+5], x[at], x[2*6615-1], x[2*6615-1])
	}
	// glide of a signal
	x = renderPatch(t, "bpm = 100\nsteps = 4\nout = glide(seq([0 1]), 40)", 2*6615.0/44100)[0]
	if x[6615+5] <= 0 || x[6615+5] > 0.1 {
		t.Errorf("glide(): %v", x[6615+5])
	}
	// pulses: a gate of a rhythm, high for part of the step
	x = renderPatch(t, "bpm = 100\nsteps = 4\nout = pulses([true false true false], length: 0.5)", 4*6615.0/44100)[0]
	if x[10] != 1 || x[3300] != 1 || x[3400] != 0 || x[6615+10] != 0 || x[2*6615+10] != 1 {
		t.Errorf("pulses: %v %v %v %v %v", x[10], x[3300], x[3400], x[6615+10], x[2*6615+10])
	}
	for src, mention := range map[string]string{
		"out = seq([true false])":         "values",
		"out = seq([1 2], glide: -5)":     "glide",
		"out = glide(1, 0)":               "ms",
		"out = pulses([true], length: 0)": "length",
		"out = pulses([true], length: 2)": "length",
		"out = pulses(5)":                 "",
	} {
		_, err := build(t, src)
		if mention == "" {
			continue
		}
		if err == nil || !strings.Contains(err.Error(), mention) {
			t.Errorf("%q: %v, want a mention of %q", src, err, mention)
		}
	}
}

// a generator that has a name is one stream: used as a signal in two places, it is pulled once a step and both have the same values
func TestAGeneratorWithANameIsOneStream(t *testing.T) {
	steps := func(src string, n int) [][]float64 {
		ch := renderPatch(t, "bpm = 100\nsteps = 4\nseed = 9\n"+src, float64(n)*6615/44100)
		out := make([][]float64, len(ch))
		for c := range ch {
			for i := 0; i < n; i++ {
				out[c] = append(out[c], ch[c][i*6615+100])
			}
		}
		return out
	}
	same := func(a, b []float64) bool {
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	// one name, two places: the same values
	ch := steps("p = mutate([0 1 2 3 4 5 6 7], swap: 1)\nout = [p, p * 1]", 24)
	if len(ch) != 2 || !same(ch[0], ch[1]) {
		t.Errorf("the same phrase in two places is not the same: %v\n%v", ch[0], ch[1])
	}
	// two separate phrases are not (with the swaps of a seed, they move on independently)
	ch = steps("out = [mutate([0 1 2 3 4 5 6 7], swap: 1), mutate([0 1 2 3 4 5 6 7], swap: 1)]", 24)
	if same(ch[0], ch[1]) {
		t.Error("two separate phrases are the same")
	}
	// the generator as a signal and in a seq are one stream too
	ch = steps("p = degrees(60, major, [0 1 2 3 4])\nout = [p, seq(p)]", 12)
	if !same(ch[0], ch[1]) {
		t.Errorf("seq of a generator is another stream: %v %v", ch[0], ch[1])
	}
	// a rhythm and its gate: the same name, the same length, one gate
	ch = steps("h = euclid(8, 3)\nout = [pulses(h, length: 0.5), pulses(h, length: 0.5)]", 16)
	if !same(ch[0], ch[1]) {
		t.Errorf("one rhythm made two gates: %v %v", ch[0], ch[1])
	}
	// another length of the same rhythm is a gate of its own (it moves the rhythm on too: say so, not hide it)
	if _, err := build(t, "h = euclid(8, 3)\nout = [pulses(h, length: 0.5), pulses(h, length: 0.2)]"); err != nil {
		t.Error(err)
	}
	// every program starts again: a name in one program does not carry its signal into the next
	a := steps("p = degrees(60, major, [0 1 2 3])\nout = p", 4)[0]
	b := steps("p = degrees(70, major, [0 1 2 3])\nout = p", 4)[0]
	if a[0] != 60 || b[0] != 70 {
		t.Errorf("programs share: %v %v", a, b)
	}
}

// a signal as a field is read at the start of every note: one stream can drive the notes and an effect, at the same step
func TestPlayReadsASignalField(t *testing.T) {
	src := `
bpm = 100
steps = 4
stream = seq(degrees(60, major, [0 1 2 3 4 5 6 7]))
fn inst(gate, n) { gate * n }
notes = play(inst, rhythm: [true false], legato: 1, fields: [stream])
out = [notes, stream]
`
	// the notes are on every second step (0, 2, 4, 6): they have the value of the stream at their step: 60, 64, 67, 71. The stream itself
	// steps through 60 62 64 65 67 69 71 72
	ch := renderPatch(t, src, 8*6615.0/44100)
	step := 6615
	for i, w := range []float64{60, 64, 67, 71} {
		if got := ch[0][2*i*step+200]; got != w {
			t.Errorf("note %d: %v, want %v (the value of the stream at its step)", i, got, w)
		}
	}
	if ch[1][2*step+200] != 64 || ch[1][3*step+200] != 65 {
		t.Errorf("the stream: %v %v", ch[1][2*step+200], ch[1][3*step+200])
	}
	// a generator as a field is another thing: it moves on once for every note (60, 62, 64, 65)
	src2 := "bpm = 100\nsteps = 4\nfn inst(gate, n) { gate * n }\nout = play(inst, rhythm: [true false], legato: 1, fields: [degrees(60, major, [0 1 2 3 4 5 6 7])])"
	x := renderPatch(t, src2, 8*6615.0/44100)[0]
	for i, w := range []float64{60, 62, 64, 65} {
		if got := x[2*i*step+200]; got != w {
			t.Errorf("generator, note %d: %v, want %v", i, got, w)
		}
	}
	// a signal of two channels is not a number
	if _, err := build(t, "bpm = 100\nfn inst(gate, n) { gate }\nout = play(inst, dur: [1], fields: [sine([1, 2])])"); err == nil || !strings.Contains(err.Error(), "2 channels") {
		t.Errorf("two channels: %v", err)
	}
	// a constant signal (a number is a generator of itself, a lfo is a signal)
	y := renderPatch(t, "bpm = 100\nsteps = 4\nfn inst(gate, n) { gate * n }\nout = play(inst, dur: [1], fields: [lfo(0).range(5, 5)])", 0.3)[0]
	if y[300] != 5 {
		t.Errorf("a constant signal: %v", y[300])
	}
}
