package lang

import (
	"math"
	"strings"
	"testing"
)

// onsets are the samples where the gate of a played pattern rises, and the level the instrument has there
func onsets(x []float64) (at []int, level []float64) {
	for i := 1; i < len(x); i++ {
		if x[i-1] == 0 && x[i] != 0 {
			at = append(at, i)
			level = append(level, x[i])
		}
	}
	return
}

// bpm 150, steps 4: a step is 0.1 s = 4410 samples
const grooveHead = "bpm = 150\nsteps = 4\nfn inst(gate) { gate }\n"

func TestSwingDelaysEveryOtherStep(t *testing.T) {
	step := 4410
	straight, _ := onsets(renderPatch(t, grooveHead+"out = play(inst, rhythm: every(1), legato: 0.3)", 1)[0])
	swung, _ := onsets(renderPatch(t, grooveHead+"out = play(inst, rhythm: every(1), legato: 0.3, swing: 0.5)", 1)[0])
	if len(straight) < 8 || len(swung) < 8 {
		t.Fatalf("%d and %d notes", len(straight), len(swung))
	}
	for i := 0; i < 8; i++ {
		want := straight[i]
		if i%2 == 0 { // the first note that is seen is on the step 1 (the one on step 0 starts at the first sample)
			want += step / 2
		}
		if d := swung[i] - want; d < -64 || d > 64 { // the block of 64 samples that SuperCollider has too
			t.Errorf("note %d starts at %d, wanted about %d", i, swung[i], want)
		}
	}
	// a swung note is held as long
	g := renderPatch(t, grooveHead+"out = play(inst, rhythm: every(1), legato: 0.3, swing: 0.5)", 1)[0]
	high := 0
	for i := 4410; i < 4410+4410; i++ { // the second step: the late note, half a step in
		if g[i] != 0 {
			high++
		}
	}
	if math.Abs(float64(high)-0.3*float64(step)) > 200 {
		t.Errorf("the late note is high for %d samples, wanted about %d", high, step/4)
	}
	// on a rhythm of 2 steps nothing is late (all the notes are on the even steps)
	even, _ := onsets(renderPatch(t, grooveHead+"out = play(inst, rhythm: every(2), legato: 0.3, swing: 0.5)", 1)[0])
	plain, _ := onsets(renderPatch(t, grooveHead+"out = play(inst, rhythm: every(2), legato: 0.3)", 1)[0])
	for i := range even {
		if i < len(plain) && even[i] != plain[i] {
			t.Errorf("note %d on an even step moved: %d against %d", i, even[i], plain[i])
		}
	}
}

func TestHumanizeMovesNotesAtRandomButNotFar(t *testing.T) {
	human, _ := onsets(renderPatch(t, "seed = 2\n"+grooveHead+"out = play(inst, rhythm: every(1), legato: 0.3, humanize: 0.2)", 2)[0])
	moved := 0
	for i := range human {
		d := human[i] - int(math.Round(float64(human[i])/4410))*4410 // how far from the nearest step
		if d != 0 {
			moved++
		}
		if math.Abs(float64(d)) > 0.2*4410+130 {
			t.Errorf("note %d is %d samples off", i, d)
		}
	}
	if moved < len(human)/2 {
		t.Errorf("only %d of %d notes moved", moved, len(human))
	}
}

func TestAccentsAreHowStrongTheNotesAre(t *testing.T) {
	// groups of 3 3 2: the levels of the steps are 1 .55 .55 .8 .55 .55 .8 .55
	_, lv := onsets(renderPatch(t, grooveHead+"out = play(inst, rhythm: every(1), legato: 0.3, accent: accents([3 3 2]))", 1.6)[0])
	// (the note on the first step starts at the first sample, where no rise is seen: the levels are those of the steps from 1)
	want := []float64{0.55, 0.55, 0.8, 0.55, 0.55, 0.8, 0.55, 1}
	for i, w := range want {
		if i >= len(lv) || math.Abs(lv[i]-w) > 1e-9 {
			t.Fatalf("levels %v, wanted %v", lv, want)
		}
	}
	// the accent goes by the step, not by the note: a rhythm with rests has the notes on the steps that are strong
	_, lv = onsets(renderPatch(t, grooveHead+"out = play(inst, rhythm: euclid(4, 2, 1), legato: 0.3, accent: [1 0.2 0.6 0.2])", 1.2)[0])
	if len(lv) < 4 || lv[0] != 0.2 || lv[1] != 0.2 || lv[2] != 0.2 {
		t.Errorf("notes on the steps 1 and 3: %v", lv)
	}
	// by dur too: the note takes the accent of the step it starts on
	_, lv = onsets(renderPatch(t, grooveHead+"out = play(inst, dur: [0.5 0.25 0.25], legato: 0.3, accent: [1 0.1 0.2 0.3 0.4])", 1.5)[0])
	if len(lv) < 4 || lv[0] != 0.2 || lv[1] != 0.3 || lv[2] != 0.4 || lv[3] != 0.1 {
		// notes at the steps 0, 2, 3, 4, 6: after the first, the levels of the steps 2 3 4 and 6 (6 mod 5 is 1)
		t.Errorf("%v", lv)
	}
}

func TestMistakesInTheGroove(t *testing.T) {
	for src, want := range map[string]string{
		"out = play(inst, rhythm: every(1), swing: 1.5)":     "swing is from 0",
		"out = play(inst, rhythm: every(1), humanize: 0.9)":  "humanize from 0 to 0.5",
		`out = play(inst, rhythm: every(1), accent: "loud")`: "accent",
		"out = accents([0 2])":                               "from 1",
	} {
		_, err := Compile(grooveHead + src + "\n")
		if err == nil || !containsStr(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// a live edit of the swing is heard from the next notes, and the pattern goes on where it was
func TestALiveEditOfTheSwingKeepsThePatternAndSwingsTheNextNotes(t *testing.T) {
	src := `bpm = 120
steps = 4
fn inst(gate, n) { gate * n }
out = play(inst, rhythm: [true true true true], legato: 0.25, fields: [[1 2 3 4 5 6 7 8 9 10 11 12 13 14 15]], swing: 0)`
	l := startLive(t, src)
	l.play(4500)
	l.swapTo(strings.Replace(src, "swing: 0", "swing: 0.5", 1))
	if kept, fresh := l.r.LastSwap(); fresh != 0 {
		t.Errorf("kept %d, %d started again", kept, fresh)
	}
	x := l.play(13000)
	// a step is 1000 samples; the notes that were made before the swap are as they were (the control side is some steps ahead), the ones after it
	// have the swing: on the odd steps 500 samples late. The numbers go on (n counts 1 2 3...: the note on step 9 is the 10th)
	level := func(at int) float64 { return x[at-4500] }
	if level(9000+100) != 0 || level(9000+500+100) != 10 {
		t.Errorf("the note on step 9: %v at its step, %v half a step later (wanted 0 and 10)", level(9100), level(9600))
	}
	if level(10000+100) != 11 {
		t.Errorf("the note on step 10 is on time: %v", level(10100))
	}
}

func TestALiveEditOfTheAccentGoesOnWithThePattern(t *testing.T) {
	src := `bpm = 120
steps = 4
fn inst(gate) { gate }
out = play(inst, rhythm: [true true true true], legato: 0.25, accent: [1 0.5])`
	l := startLive(t, src)
	l.play(4500)
	l.swapTo(strings.Replace(src, "[1 0.5]", "[1 0.25]", 1))
	if kept, fresh := l.r.LastSwap(); fresh != 0 {
		t.Errorf("kept %d, %d started again", kept, fresh)
	}
	x := l.play(13000)
	at := func(s int) float64 { return x[s-4500] }
	// the notes on the steps 9 and 10 (the accent goes by the step, and the new one is on the same steps)
	if at(9000+100) != 0.25 || at(10000+100) != 1 {
		t.Errorf("%v %v (wanted 0.25 and 1)", at(9100), at(10100))
	}
	// an accent where there was none is another pattern of nodes: it is not an error
	l.swapTo(strings.Replace(src, ", accent: [1 0.25]", "", 1))
}

func TestANamedGrooveGivesTheStepsTheirFeel(t *testing.T) {
	step := 4410
	plain, _ := onsets(renderPatch(t, grooveHead+"out = play(inst, rhythm: every(1), legato: 0.3)", 1.2)[0])
	// mpc 62: every other step is 0.24 of a step late, and the late ones are softer (0.88)
	feel, lv := onsets(renderPatch(t, grooveHead+`out = play(inst, rhythm: every(1), legato: 0.3, groove: "mpc 62")`, 1.2)[0])
	for i := 0; i < 8; i++ {
		want := plain[i]
		if i%2 == 0 { // the first note that is seen is on step 1
			want += int(0.24 * float64(step))
		}
		if d := feel[i] - want; d < -70 || d > 70 {
			t.Errorf("note %d starts at %d, wanted about %d", i, feel[i], want)
		}
		if w := map[bool]float64{true: 0.88, false: 1}[i%2 == 0]; lv[i] < w-1e-9 || lv[i] > w+1e-9 {
			t.Errorf("note %d has the level %v, wanted %v", i, lv[i], w)
		}
	}
	if _, err := Compile(grooveHead + `out = play(inst, rhythm: every(1), groove: "mpc 99")` + "\n"); err == nil || !strings.Contains(err.Error(), "no groove") {
		t.Errorf("a groove that is not there: %v", err)
	}
}

func TestLanesGiveEveryStepAValueOnALengthOfItsOwn(t *testing.T) {
	step := 4410
	// prob: the steps of a lane of 2: 100 percent, 0 percent: the odd steps do not play (only the even ones are heard: every second step)
	x := renderPatch(t, grooveHead+"out = play(inst, rhythm: every(1), legato: 0.3, lanes: \"prob: 100 0\")", 1.7)[0]
	at, _ := onsets(x)
	for i := 1; i < len(at); i++ {
		if d := at[i] - at[i-1]; d < 2*step-70 || d > 2*step+70 {
			t.Fatalf("notes %d samples apart, wanted %d: %v", d, 2*step, at)
		}
	}
	if len(at) < 3 {
		t.Fatalf("%v", at)
	}
	// vel: percent, a lane of 3 against the steps (polymeter): 100 50 25 round and round
	_, lv := onsets(renderPatch(t, grooveHead+"out = play(inst, rhythm: every(1), legato: 0.3, lanes: \"vel: 100 50 25\")", 1.5)[0])
	for i, w := range []float64{0.5, 0.25, 1, 0.5, 0.25, 1} { // the first note seen is on step 1
		if lv[i] != w {
			t.Fatalf("levels %v, wanted %v", lv, w)
		}
	}
	// shift: 50 percent of a step late on a lane of 4 with gaps, and len: the note is held half as long
	plain, _ := onsets(renderPatch(t, grooveHead+"out = play(inst, rhythm: every(1), legato: 0.4)", 1.5)[0])
	moved, _ := onsets(renderPatch(t, grooveHead+"out = play(inst, rhythm: every(1), legato: 0.4, lanes: \"shift: 50 . . .; len: 50\")", 1.5)[0])
	// the lane has a value on every fourth step (0, 4, 8): the note on step 0 is late too (and is seen, as it is not at sample 0); the note on step 4 is
	// the fifth that is seen, and the fourth of the plain ones
	if d := moved[4] - plain[3]; d < step/2-70 || d > step/2+70 {
		t.Errorf("a shift of 50 percent: %d samples late, wanted about %d (%v %v)", d, step/2, plain[:4], moved[:4])
	}
	g := renderPatch(t, grooveHead+"out = play(inst, rhythm: every(1), legato: 0.4, lanes: \"len: 50\")", 1)[0]
	high := 0
	for i := 5 * step; i < 6*step; i++ {
		if g[i] != 0 {
			high++
		}
	}
	if math.Abs(float64(high)-0.2*float64(step)) > 150 { // 0.4 of a step held, half of it
		t.Errorf("held for %d samples, wanted about %d", high, step/5)
	}
	for src, want := range map[string]string{
		`lanes: "prob 100"`:    "a lane is like",
		`lanes: "pitch: 1 2"`:  "one of prob, vel, shift, len",
		`lanes: "prob: 100 x"`: "prob",
	} {
		_, err := Compile(grooveHead + "out = play(inst, rhythm: every(1), " + src + ")\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}
