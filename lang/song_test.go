package lang

import (
	"math"
	"strings"
	"testing"
)

// 150 bpm and 4 steps a beat at 44100 Hz: a step is 4410 samples. The song is A (1 bar of 4 steps) and B (2 bars: 8 steps): 12 steps, round
const songHead = "bpm = 150\nsteps = 4\nform = song(\"A 1, B 2\", barsteps: 4)\n"

func TestASongGivesSignalsThatFollowItsSections(t *testing.T) {
	at := func(x []float64, step float64) float64 { return x[int(step*4410)] }
	// during: 1 in B, 0 in A; with no glide it changes on the step, with a glide of 2 steps it is half way at the edge
	hard := renderPatch(t, songHead+"out = during(form, \"B\", glide: 0)", 3)[0]
	if at(hard, 2) != 0 || at(hard, 4) != 1 || at(hard, 11.9) != 1 || at(hard, 12.1) != 0 || at(hard, 16) != 1 {
		t.Errorf("hard: %v %v %v %v %v", at(hard, 2), at(hard, 4), at(hard, 11.9), at(hard, 12.1), at(hard, 16))
	}
	soft := renderPatch(t, songHead+"out = during(form, \"B\", glide: 2)", 3)[0]
	if at(soft, 3) != 0 || math.Abs(at(soft, 4)-0.5) > 1e-6 || at(soft, 5) != 1 || at(soft, 9) != 1 {
		t.Errorf("soft: %v %v %v %v", at(soft, 3), at(soft, 4), at(soft, 5), at(soft, 9))
	}
	// the song goes round: A, B and A again; the end of B is the start of A, and the ramp at the seam is not a dip when two sections follow
	both := renderPatch(t, songHead+"out = during(form, \"all\", glide: 2)", 3)[0]
	for _, s := range []float64{0.5, 3.9, 4.1, 11.9, 12.1, 16} {
		if at(both, s) != 1 {
			t.Errorf("all is 1 everywhere: %v at step %v", at(both, s), s)
		}
	}
	not := renderPatch(t, songHead+"out = during(form, \"not B\", glide: 0)", 2)[0]
	if at(not, 2) != 1 || at(not, 6) != 0 {
		t.Errorf("not B: %v %v", at(not, 2), at(not, 6))
	}
	// curve: A goes from 0 to 1, B from 1 to 0 in a smooth way; a section with no curve keeps the value
	c := renderPatch(t, songHead+"out = curve(form, \"A: 0 -> 1, B: 1 -> 0 smooth\")", 3)[0]
	if math.Abs(at(c, 2)-0.5) > 1e-3 || math.Abs(at(c, 8)-0.5) > 1e-3 || at(c, 4.001) < 0.99 || math.Abs(at(c, 14)-0.5) > 1e-3 {
		t.Errorf("curve: %v %v %v %v", at(c, 2), at(c, 8), at(c, 4.001), at(c, 14))
	}
	held := renderPatch(t, songHead+"out = curve(form, \"A: 0.25 -> 0.75\")", 2)[0]
	if math.Abs(at(held, 8)-0.75) > 1e-6 { // B has none: it keeps where A ended
		t.Errorf("held: %v", at(held, 8))
	}
	// the sections that have a name twice share the curve; a layer goes in and out through gain
	twice := "bpm = 150\nsteps = 4\nform = song(\"A 1, B 1, A 1\", barsteps: 4)\nout = during(form, \"A\", glide: 0)"
	x := renderPatch(t, twice, 2)[0]
	if at(x, 1) != 1 || at(x, 5) != 0 || at(x, 9) != 1 {
		t.Errorf("twice: %v %v %v", at(x, 1), at(x, 5), at(x, 9))
	}
}

func TestSectionsChooseGenerators(t *testing.T) {
	// the first rhythm in A, nothing in B (a section that is in no name plays nothing)
	if got := pull(t, "form = song(\"A 1, B 2\", barsteps: 4)\nx = bysection(form, [\"A\"], [every(1)])", 24); got != "xxxx........xxxx........" {
		t.Errorf("%s", got)
	}
	// two generators, B has its own, and restart starts it again each time its section comes
	src := "form = song(\"A 1, B 1, A 1, B 1\", barsteps: 4)\nx = bysection(form, [\"A\", \"B\"], [every(1), pattern(\"x...\")]%s)"
	if got := pull(t, strings.Replace(src, "%s", "", 1), 16); got != "xxxxx...xxxxx..." {
		t.Errorf("%s", got)
	}
	if got := pull(t, "form = song(\"A 1, B 1, A 1, B 1\", barsteps: 4)\nx = bysection(form, [\"A\", \"B\"], [every(1), pattern(\"x.x.\")], restart: true)", 16); got != "xxxxx.x.xxxxx.x." {
		t.Errorf("%s", got)
	}
}

func TestMistakesInSongs(t *testing.T) {
	for src, want := range map[string]string{
		`x = song("Intro")`:                                  "a section is a name and its bars",
		`x = song("Intro x")`:                                "a section is a name and its bars",
		`x = song("")`:                                       "no sections",
		`x = song("A 1", barsteps: 0)`:                       "barsteps",
		`x = during(song("A 1"), "B")`:                       "no section \"B\"",
		`x = during(song("Drop 1"), "Dorp")`:                 "did you mean Drop",
		`x = curve(song("A 1"), "A 0 1")`:                    "a curve is like",
		`x = curve(song("A 1"), "A: 0 => 1")`:                "a value",
		`x = curve(song("A 1"), "A: 0 -> 1 exp")`:            "above 0",
		`x = curve(song("A 1"), "A: x")`:                     "not a number",
		`x = bysection(song("A 1"), ["A", "B"], [every(1)])`: "a name",
	} {
		_, err := Compile("x0 = 1\n" + src + "\nout = sine(440).dup()\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}
