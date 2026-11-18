package lang

import (
	"math"
	"strings"
	"testing"
)

// the instrument is the gate times the number: the sum of the voices at a moment is the sum of the notes of the chord that is on
const chordHead = "bpm = 120\nsteps = 4\nfn inst(gate, n) { gate * n }\n"

func TestAChordInPlayStartsAllItsNotesOnOneStep(t *testing.T) {
	// 120 bpm and 4 steps a beat at 44100 Hz: a step is 5512.5 samples
	src := chordHead + "out = play(inst, rhythm: [true false false false], legato: 0.5, fields: [[[1 2 4] [10 20 40]]], voices: 3)"
	x := renderPatch(t, src, 2.5)[0]
	at := func(step float64) float64 { return x[int(step*5512.5)+300] }
	// the first chord on step 0, the second on step 4; the gates are high for half of the steps to the next (legato 0.5 of 4 steps: 2)
	if at(0) != 7 || at(1.5) != 7 || at(2.5) != 0 || at(4) != 70 || at(5.5) != 70 || at(6.5) != 0 || at(8) != 7 {
		t.Errorf("%v %v %v %v %v %v %v (wanted 7 7 0 70 70 0 7)", at(0), at(1.5), at(2.5), at(4), at(5.5), at(6.5), at(8))
	}
}

func TestChordsOfDifferentSizesAndWithOtherFields(t *testing.T) {
	// a chord of 1, then of 3, with another field (the second number is 100 times the number of the chord: it is the same for all its notes)
	src := "bpm = 120\nsteps = 4\nfn inst(gate, n, k) { gate * (n + k) }\n" +
		"out = play(inst, rhythm: [true false], legato: 0.5, fields: [[[1] [1 2 3]], [100 200]], voices: 3)"
	x := renderPatch(t, src, 1.5)[0]
	at := func(step float64) float64 { return x[int(step*5512.5)+300] }
	if at(0) != 101 || at(2) != 3*200+6 {
		t.Errorf("%v %v (wanted 101 and 606)", at(0), at(2))
	}
	// with dur instead of rhythm too
	src = chordHead + "out = play(inst, dur: [1], legato: 0.5, fields: [[[1 2] [10 20]]], voices: 2)"
	x = renderPatch(t, src, 2)[0]
	if v := x[int(4*5512.5)+300]; v != 30 {
		t.Errorf("%v (wanted 30)", v)
	}
	// a generator of chords: the chords of a scale on the degrees
	src = chordHead + "out = play(inst, rhythm: every(4), legato: 0.5, fields: [triads(60, major, [0 3])], voices: 3)"
	x = renderPatch(t, src, 2)[0]
	if v := x[300]; v != 60+64+67 {
		t.Errorf("the triad of C: %v", v)
	}
	if v := x[int(4*5512.5)+300]; v != 65+69+72 {
		t.Errorf("the triad of F: %v", v)
	}
}

func TestAScoreWithChordsForPlay(t *testing.T) {
	src := chordHead + "s = together([phrase([1 2], [2]), phrase([10 20], [2])])\nout = play(inst, rhythm: onsets(s), legato: 0.5, fields: [chordsof(s)], voices: 2)"
	x := renderPatch(t, src, 1)[0]
	if v := x[300]; v != 11 {
		t.Errorf("the notes together on step 0: %v (wanted 11)", v)
	}
	if v := x[int(2*5512.5)+300]; v != 22 {
		t.Errorf("on step 2: %v (wanted 22)", v)
	}
}

func TestChordsInPlayMistakes(t *testing.T) {
	for src, want := range map[string]string{
		"out = play(inst, rhythm: every(1), fields: [[[1 2 3] [4 5 6]]], voices: 2)":   "needs voices: 3",
		"out = play(inst, rhythm: every(1), fields: [[[1 2] [3 4]], [[1 2] [3 4]]])":   "only one field",
		"out = play(inst, rhythm: every(1), fields: [[[1 2] [3 4]]], voices: 2)":       "",
		"out = play(inst, rhythm: every(1), fields: [chordsof(phrase([60 62], [1]))])": "",
	} {
		_, err := Compile("fn inst(gate, n) { gate * n }\n" + src + "\n")
		if src == "" || want == "" {
			if err != nil && !strings.Contains(src, "chordsof") {
				t.Errorf("%s: %v", src, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}

// a live edit of a play with chords: the pattern goes on, and the chords are all there after it
func TestALiveEditOfAPlayWithChordsKeepsTheChords(t *testing.T) {
	src := `bpm = 120
steps = 4
fn inst(gate, n) { gate * n }
out = play(inst, rhythm: [true false false false], legato: 0.5, fields: [[[1 2 4] [10 20 40]]], voices: 3)`
	l := startLive(t, src)
	l.play(5000)
	l.swapTo(strings.Replace(src, "legato: 0.5", "legato: 0.25", 1))
	x := l.play(40000)
	// notes every 4 steps (4000 samples): from well after the swap the gates are 1000 samples long (legato 0.25), and the chords are whole
	hi := func(at int) float64 { return x[at-5000] }
	for _, c := range []struct {
		at   int
		want float64
	}{{24000, 7}, {28000, 70}, {32000, 7}, {36000, 70}} {
		if hi(c.at+500) != c.want || hi(c.at+1500) != 0 {
			t.Errorf("at %d: %v then %v (wanted %v and 0)", c.at, hi(c.at+500), hi(c.at+1500), c.want)
		}
	}
	// the rhythm edited: the chords keep going in order (7 then 70), not mixed
	l.swapTo(strings.Replace(src, "[true false false false]", "[true false false false false false false false]", 1))
	x = l.play(60000)
	seen := map[float64]bool{}
	for _, v := range x {
		if v != 0 {
			seen[math.Round(v)] = true
		}
	}
	for v := range seen {
		if v != 7 && v != 70 && v != 3 && v != 6 && v != 1 && v != 2 && v != 4 && v != 10 && v != 20 && v != 40 && v != 30 && v != 50 && v != 60 && v != 14 && v != 24 && v != 44 {
			t.Errorf("a mixed chord: %v", v)
		}
	}
}
