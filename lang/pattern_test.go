package lang

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"harp/dsl"
)

// hitsText is a pattern step by step: the hits of each step as their offsets (and lengths when they have one), . for none
func hitsText(t *testing.T, src string, steps int) string {
	t.Helper()
	dsl.Seed(1)
	if !strings.Contains(src, "\n") {
		src = "g = " + src
	}
	v := evalLast(t, "seed = 1\n"+src)
	var g dsl.Gen[[]dsl.Hit]
	switch x := v.V.(type) {
	case dsl.Gen[[]dsl.Hit]:
		g = x
	case dsl.Gen[bool]:
		g = dsl.HitsOf(x)
	default:
		t.Fatalf("%s is a %s", src, v.T)
	}
	var out []string
	for i := 0; i < steps; i++ {
		var hs []string
		for _, h := range g.Generate() {
			s := fmt.Sprintf("%.2f", h.Offset)
			if h.Len > 0 {
				s += fmt.Sprintf("/%g", h.Len)
			}
			hs = append(hs, s)
		}
		if len(hs) == 0 {
			out = append(out, ".")
		} else {
			out = append(out, strings.Join(hs, "+"))
		}
	}
	return strings.Join(out, " ")
}

func TestThePatternTextHasMoreThanHitsAndRests(t *testing.T) {
	for src, want := range map[string]string{
		// plain: x and . (and the others that psalm reads)
		`pattern("x..x")`:          "0.00 . . 0.00",
		`pattern("X _ - o | 1 0")`: "0.00 . . 0.00 0.00 .",
		// a roll: three hits in a step
		`pattern("x3 .")`: "0.00+0.33+0.67 .",
		// a group fills one step: a triplet, and with rests
		`pattern("[x x x] .")`:   "0.00+0.33+0.67 .",
		`pattern("[x . x x]")`:   "0.00+0.50+0.75",
		`pattern("[x [x x]] .")`: "0.00+0.50+0.75 .",
		// how long: x*2 lasts two steps (the step after it has none)
		`pattern("x*2 . x")`: "0.00/2 . . 0.00",
		// 1 and 0 are single steps, never a roll
		`pattern("1010")`: "0.00 . 0.00 .",
		// a chance: ?0 never plays, ?100 always
		`pattern("x?0 x?100")`: ". 0.00",
		// a condition: the second time through in every four, and the other three
		`pattern("x%2/4")`:     ". 0.00 . . . 0.00 . .",
		`pattern("x%!2/4")`:    "0.00 . 0.00 0.00 0.00 . 0.00 0.00",
		`pattern("x%first .")`: "0.00 . . . . . . .",
		// functions: euclid, additive, and a euclid that turns every pass
		`pattern("E(3,8)")`:    "0.00 . . 0.00 . . 0.00 .",
		`pattern("A(3+3+2)")`:  "0.00 . . 0.00 . . 0.00 .",
		`pattern("E(3,8,+1)")`: "0.00 . . 0.00 . . 0.00 . . 0.00 . . 0.00 . . 0.00",
	} {
		n := len(strings.Fields(want))
		if got := hitsText(t, src, n); got != want {
			t.Errorf("%s:\n got  %s\n want %s", src, got, want)
		}
	}
}

func TestAPatternMutatesAndDrifts(t *testing.T) {
	steady := hitsText(t, `pattern("x . . . x . . .")`, 32)
	mutated := hitsText(t, `pattern("x . . . x . . .", mutate: 1, seed: 7)`, 32)
	if steady == mutated {
		t.Errorf("a mutating pattern is the same every pass: %s", mutated)
	}
	// the same seed makes the same changes, another seed other ones
	if again := hitsText(t, `pattern("x . . . x . . .", mutate: 1, seed: 7)`, 32); again != mutated {
		t.Errorf("the same seed, other changes")
	}
	if other := hitsText(t, `pattern("x . . . x . . .", mutate: 1, seed: 8)`, 32); other == mutated {
		t.Errorf("another seed, the same changes")
	}
	// a hit with ! stays: the first step has a hit in every pass
	for i, f := range strings.Fields(hitsText(t, `pattern("x! . . . . . . .", mutate: 1, seed: 3)`, 64)) {
		if i%8 == 0 && f == "." {
			t.Fatalf("a kept hit fell out in pass %d", i/8)
		}
	}
	// frozen after 2 passes: passes 2 and after are the same
	f := strings.Fields(hitsText(t, `pattern("x . . . x . . .", mutate: 1, seed: 7, freeze: 2)`, 40))
	if strings.Join(f[16:24], " ") != strings.Join(f[24:32], " ") || strings.Join(f[24:32], " ") != strings.Join(f[32:40], " ") {
		t.Errorf("it goes on changing after it is frozen: %v", f)
	}
	// a drift moves the hits late by up to the drift, smoothly
	for _, s := range strings.Fields(hitsText(t, `pattern("x x x x", drift: 0.2)`, 24)) {
		var off float64
		fmt.Sscanf(s, "%f", &off)
		if off < 0 || off > 0.2+1e-9 {
			t.Fatalf("a drift of 0.2 moved a hit by %v", off)
		}
	}
}

func TestAPatternInASong(t *testing.T) {
	// %fill: only in the last bar of a section (a song of 2 bars of 4 steps, then 1 bar)
	got := hitsText(t, "form = song(\"A 2, B 1\", barsteps: 4)\nx = pattern(\"x%fill . . .\", song: form)", 12)
	if got != ". . . . 0.00 . . . 0.00 . . ." {
		t.Errorf("%s", got)
	}
}

func TestMistakesInPatterns(t *testing.T) {
	for src, want := range map[string]string{
		`pattern("x.q")`:            "unexpected",
		`pattern("  ")`:             "no steps",
		`pattern("[x x")`:           "without its ]",
		`pattern("x x]")`:           "without its [",
		`pattern("[]")`:             "empty group",
		`pattern("x100")`:           "1 to 64",
		`pattern("x*0")`:            "above 0",
		`pattern("x?200")`:          "0 to 100",
		`pattern("x%3/2")`:          "condition",
		`pattern("x", mutate: 2)`:   "mutate is from 0 to 1",
		`pattern("x", drift: 1)`:    "drift",
		`pattern("E(9,8)")`:         "E(9,8)",
		`pattern("x", song: "A 1")`: "song is a song",
	} {
		_, err := Compile("seed = 1\nx = " + src + "\nout = sine(440).dup()\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}

func TestHitsInsideAStepArePlayed(t *testing.T) {
	step := 4410 // bpm 150, 4 steps a beat, 44100 Hz
	// play: a roll of three in a step, then a rest of three steps
	at, _ := onsets(renderPatch(t, grooveHead+"out = play(inst, rhythm: pattern(\"x3 . . .\"), legato: 0.2)", 1.2)[0])
	want := []int{step / 3, 2 * step / 3} // the first hit, on sample 0, is not seen as a rise
	for i, w := range want {
		if d := at[i] - w; d < -70 || d > 70 {
			t.Errorf("hit %d starts at %d, wanted about %d (%v)", i+1, at[i], w, at[:4])
		}
	}
	if d := at[2] - 4*step; d < -70 || d > 70 {
		t.Errorf("the next pass starts at %d, wanted about %d", at[2], 4*step)
	}
	// a pattern with a hit that lasts two steps holds the gate for as long (legato 0.5 of 2 steps: one step)
	g := renderPatch(t, grooveHead+"out = play(inst, rhythm: pattern(\"x*2 . . .\"), legato: 0.5)", 1)[0]
	high := 0
	for _, v := range g[:3*step] {
		if v != 0 {
			high++
		}
	}
	if math.Abs(float64(high)-float64(step)) > 150 {
		t.Errorf("held for %d samples, wanted about %d", high, step)
	}
	// a triplet in melody: the notes have the offsets, and the pitches go one for each hit
	if got := noteText(t, `melody([60 62 64 65], pattern("[x x x] ."), velocity: 1, length: 1)`, 2); !strings.HasPrefix(got, "{60@1/1,62@1/1,64@1/1}") {
		t.Errorf("%s", got)
	}
	// ... and a synth plays them late by their offsets: a short drum on every hit of a roll of two
	hat := renderPatch(t, "bpm = 150\nsteps = 4\nout = drumsynth(melody([drum(\"hat\")], pattern(\"x2 . . .\"), velocity: 1, length: 1), voices: 4)\n", 1)[0]
	e := func(a, b int) float64 { return energy(hat[a:b]) }
	if e(2205, 2205+600) < 10*e(1300, 1900) || e(100, 700) < 10*e(1300, 1900) {
		t.Errorf("a hit on 0 and one half a step later: %v %v and between %v", e(100, 700), e(2205, 2805), e(1300, 1900))
	}
}
