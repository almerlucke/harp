package lang

import (
	"fmt"
	"strings"
	"testing"

	"harp/blocks"
	"harp/dsl"
)

// noteText is the notes of a generator of notes, step by step: pitch@velocity/length, notes of a step together, {} for a rest
func noteText(t *testing.T, src string, steps int) string {
	t.Helper()
	dsl.Seed(1)
	v := evalLast(t, "seed = 1\ng = "+src)
	g := v.V.(dsl.Gen[[]blocks.Note])
	var out []string
	for i := 0; i < steps; i++ {
		var ns []string
		for _, n := range g.Generate() {
			ns = append(ns, fmt.Sprintf("%g@%g/%g", n.Pitch, n.Velocity, n.Length))
		}
		out = append(out, "{"+strings.Join(ns, ",")+"}")
	}
	return strings.Join(out, " ")
}

func TestVelocityAndLengthAreSequences(t *testing.T) {
	for src, want := range map[string]string{
		// numbers are as they were
		`melody([60], every(1), velocity: 0.5, length: 2)`: "{60@0.5/2} {60@0.5/2}",
		`melody([60], every(1))`:                           "{60@0.8/1} {60@0.8/1}",
		// a list: one for each note, in turn, round and round
		`melody([60 62 64], every(1), velocity: [1 0.5], length: [1 3])`: "{60@1/1} {62@0.5/3} {64@1/1} {60@0.5/3}",
		// the values are taken with the pitch, when the rhythm plays: a rest takes none
		`melody([60 62 64], pattern("x.x."), velocity: [1 0.5 0.25])`: "{60@1/1} {} {62@0.5/1} {}",
		// a velocity of 0 is a rest: the note is not played (the pitch and the length go on)
		`melody([60 62 64], every(1), velocity: [1 0 1], length: [1 2 3])`: "{60@1/1} {} {64@1/3} {60@1/1}",
		// chords: a value for each chord, the same for all of its notes
		`chords([[60 64] [62 65]], every(2), velocity: [1 0.5], length: [2 4])`: "{60@1/2,64@1/2} {} {62@0.5/4,65@0.5/4} {}",
		// a generator
		`melody([60], every(1), velocity: jump([0.2 0.6 1], 1, 1, 1), length: 1)`: "{60@0.2/1} {60@0.6/1} {60@1/1}",
	} {
		n := strings.Count(want, "{")
		if got := noteText(t, src, n); got != want {
			t.Errorf("%s:\n got  %s\n want %s", src, got, want)
		}
	}
	// a generator of numbers that moves: wander
	for _, n := range strings.Fields(noteText(t, `melody([60], every(1), velocity: wander(0.3, 0.9), length: wander(1, 4))`, 30)) {
		var p, v, l float64
		fmt.Sscanf(strings.Trim(n, "{}"), "%g@%g/%g", &p, &v, &l)
		if v < 0.3-1e-9 || v > 0.9+1e-9 || l < 1-1e-9 || l > 4+1e-9 {
			t.Fatalf("outside the range: %s", n)
		}
	}
}

func TestAPhraseHasVelocitiesAndLegatos(t *testing.T) {
	// a value for each pitch, a rest has one too: the pitches 60 62 -1 64 have the velocities 1 .5 1 .5 and the legatos 1 .5 1 .5
	if got := scoreText(t, `phrase([60 62 -1 64], [1], velocity: [1 0.5], legato: [1 0.5])`); got != "0:60/1 1:62/0.5 3:64/0.5" {
		t.Errorf("%s", got)
	}
	vs := ""
	for _, e := range evalLast(t, `x = phrase([60 62 64], [1], velocity: [1 0.5 0])`).V.(*Score).Events {
		vs += fmt.Sprintf("%g:%g ", e.Start, e.Velocity)
	}
	if vs != "0:1 1:0.5 " { // a velocity of 0 is a rest
		t.Errorf("%q", vs)
	}
}

func TestMistakesInNoteSequences(t *testing.T) {
	for src, want := range map[string]string{
		`melody([60], every(1), velocity: "loud")`:    "velocity",
		`melody([60], every(1), length: [[1 2]])`:     "length",
		`chords([[60 64]], every(1), velocity: true)`: "velocity",
		`phrase([60], [1], velocity: "loud")`:         "velocity",
		`phrase([60], [1], legato: [0 1])`:            "legato is above 0",
	} {
		_, err := Compile("seed = 1\nx = " + src + "\nout = sine(440).dup()\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}

// a synth plays the velocities, and a live edit of them is heard: the notes are quieter after it
func TestALiveEditOfTheVelocitiesIsHeard(t *testing.T) {
	src := `bpm = 120
steps = 4
out = subtractive(melody([60], every(4), velocity: [0.9 0.9], length: 1), preset: "pluck", voices: 4)`
	l := startLive(t, src)
	before := l.play(8000)
	l.swapTo(strings.Replace(src, "[0.9 0.9]", "[0.1 0.1]", 1))
	l.play(8000)
	after := l.play(8000)
	loud, quiet := 0.0, 0.0
	for _, v := range before[:8000] {
		loud += v * v
	}
	for _, v := range after {
		quiet += v * v
	}
	if loud == 0 || quiet > 0.2*loud {
		t.Errorf("the velocities are not heard: %v then %v", loud, quiet)
	}
}
