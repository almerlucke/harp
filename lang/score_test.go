package lang

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"harp/blocks"
	"harp/dsl"

	"github.com/almerlucke/genny/midi"
	"github.com/almerlucke/genny/note"
)

// scoreText is what a score has, as text: start:pitch/length for each note
func scoreText(t *testing.T, src string) string {
	t.Helper()
	v := evalLast(t, "seed = 1\nx = "+src)
	if v.T.Kind != KScore {
		t.Fatalf("%s is a %s", src, v.T)
	}
	var out []string
	for _, e := range v.V.(*Score).Events {
		out = append(out, fmt.Sprintf("%g:%g/%g", e.Start, e.Pitch, e.Duration))
	}
	return strings.Join(out, " ")
}

func TestAScoreIsMadeAndChanged(t *testing.T) {
	for src, want := range map[string]string{
		`phrase([60 62 -1 64], [2 1 1 4])`:                           "0:60/2 2:62/1 4:64/4",
		`phrase([60 62], [2], legato: 0.5)`:                          "0:60/1 2:62/1",
		`transpose(phrase([60 62], [1]), 7)`:                         "0:67/1 1:69/1",
		`stretch(phrase([60 62], [1 2]), 2)`:                         "0:60/2 2:62/4",
		`retro(phrase([60 62 64], [1]))`:                             "0:64/1 1:62/1 2:60/1",
		`shift(phrase([60], [1]), 3)`:                                "3:60/1",
		`together([phrase([60], [2]), shift(phrase([67], [2]), 1)])`: "0:60/2 1:67/2",
		`cut(phrase([60 62 64 65], [1]), 1, 3)`:                      "0:62/1 1:64/1",
		`tile(phrase([60 62], [1]), 2)`:                              "0:60/1 1:62/1 2:60/1 3:62/1",
		`pad(phrase([60 62 64], [1]), 2)`:                            "0:60/1 1:62/1",
		`tune(phrase([61 63], [1]), 60, major)`:                      "0:60/1 1:62/1",
		`mirror(phrase([60 64], [1]), 60)`:                           "0:60/1 1:56/1",
		`canon(phrase([60 62], [1]), 4, 7, 2)`:                       "0:60/1 1:62/1 4:67/1 5:69/1",
		`canon(phrase([60 62], [1]), 4, 7, 2, voice: 1)`:             "4:67/1 5:69/1",
		`thin(phrase([60 62 64 65], [1]), [true false true true])`:   "0:60/1 2:64/1 3:65/1",
		`isorhythm([2 -1 1], [60 62], 8)`:                            "0:60/2 3:60/1 4:62/2 7:62/1",
	} {
		if got := scoreText(t, src); got != want {
			t.Errorf("%s:\n got  %s\n want %s", src, got, want)
		}
	}
}

func TestAScorePlaysOnSynthsAndGoesRound(t *testing.T) {
	v := evalLast(t, "x = phrase([60 -1 64], [1 1 2], velocity: 0.5)")
	g, err := coerce(v, TGenNotes)
	if err != nil {
		t.Fatal(err)
	}
	gen := g.V.(dsl.Gen[[]blocks.Note])
	var got []string
	for i := 0; i < 8; i++ {
		var ps []string
		for _, n := range gen.Generate() {
			ps = append(ps, fmt.Sprintf("%g@%g/%g", n.Pitch, n.Velocity, n.Length))
		}
		got = append(got, strings.Join(ps, ","))
	}
	// 4 steps (1 + 1 + 2), round and round: a note on the steps 0 and 2
	if want := "60@0.5/1  64@0.5/2  60@0.5/1  64@0.5/2 "; strings.Join(got, " ") != want {
		t.Errorf("%q", strings.Join(got, " "))
	}
}

func TestAMidiFileIsAScore(t *testing.T) {
	dir := t.TempDir()
	f := midi.File{Tempo: 120, StepsPerBeat: 4, Tracks: []midi.Track{
		{Name: "a", Events: []note.Event{{Pitch: 60, Velocity: 0.8, Start: 0, Duration: 2}, {Pitch: 64, Velocity: 0.8, Start: 4, Duration: 2}}},
		{Name: "b", Channel: 1, Events: []note.Event{{Pitch: 36, Velocity: 0.9, Start: 0, Duration: 8}}},
	}}
	if err := midi.WriteFile(filepath.Join(dir, "song.mid"), f); err != nil {
		t.Fatal(err)
	}
	text := func(arg string) string {
		v := evalLastIn(t, dir, "steps = 4\nx = midifile(\"song.mid\""+arg+")")
		var out []string
		for _, e := range v.V.(*Score).Events {
			out = append(out, fmt.Sprintf("%g:%g/%g", e.Start, e.Pitch, e.Duration))
		}
		return strings.Join(out, " ")
	}
	if got := text(", track: 0"); got != "0:60/2 4:64/2" {
		t.Errorf("track 0: %s", got)
	}
	if got := text(", track: 1"); got != "0:36/8" {
		t.Errorf("track 1: %s", got)
	}
	if got := text(""); got != "0:60/2 0:36/8 4:64/2" && got != "0:36/8 0:60/2 4:64/2" {
		t.Errorf("all the tracks: %s", got)
	}
	// the notes are the ones in the file, in the steps of the patch (8 steps a beat doubles the steps)
	prog, _ := Parse("steps = 8\nx = midifile(\"song.mid\", track: 0)\nout = sine(1).dup()\n")
	p, err := buildPatch(prog, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Info) == 0 || !strings.Contains(strings.Join(p.Info, "\n"), "2 tracks, 2 notes") {
		t.Errorf("check says nothing about the file: %v", p.Info)
	}
	if _, err := CompileDir("x = midifile(\"song.mid\", track: 5)\nout = sine(1).dup()\n", dir); err == nil || !strings.Contains(err.Error(), "2 tracks") {
		t.Errorf("a track that is not there: %v", err)
	}
	if _, err := CompileDir("x = midifile(\"nothing.mid\")\nout = sine(1).dup()\n", dir); err == nil || !strings.Contains(err.Error(), "there is no file") {
		t.Errorf("a file that is not there: %v", err)
	}
}

func TestPiecesThatAreScores(t *testing.T) {
	// a canon of a fugue subject in the key of C: every voice has notes, the fugue has the voices that were asked for, and the same seed is the same fugue
	subject := `fugue([60 62 64 62 60 67 65 64], [2 2 2 2 4 2 2 4], voices: 3, root: 60, scale: major, voice: %d)`
	for v := 0; v < 3; v++ {
		if got := scoreText(t, fmt.Sprintf(subject, v)); got == "" {
			t.Errorf("voice %d has no notes", v)
		}
	}
	if a, b := scoreText(t, fmt.Sprintf(subject, -1)), scoreText(t, fmt.Sprintf(subject, -1)); a != b || a == "" {
		t.Errorf("the same seed is another fugue")
	}
	// phasing: the second voice drifts, and with a drift of 1 on a pattern of 4 they are together again after 4 cycles
	if got := scoreText(t, `phasing(phrase([60 62 64 65], [1]), 1, voice: 1)`); !strings.HasPrefix(got, "0:60/1 1:62/1 2:64/1 3:65/1 3:60/1") {
		t.Errorf("%s", got)
	}
	// a melody over chords: one bar of 16 steps each, notes in range, a chord tone to start each bar
	if got := scoreText(t, `chordtones([[60 64 67] [65 69 72]], 60, major)`); !strings.HasPrefix(got, "0:") || strings.Count(got, " ") < 8 {
		t.Errorf("%s", got)
	}
}

func TestAScoreForPlay(t *testing.T) {
	s := `s = phrase([60 62 -1 64], [1 1 1 1])` + "\n"
	if got := pull(t, s+"x = onsets(s)", 8); got != "xx.xxx.x" {
		t.Errorf("onsets: %s", got)
	}
	if got := pull(t, s+"x = pitches(s)", 4); got != "60 62 64 60" {
		t.Errorf("pitches: %s", got)
	}
	if got := pull(t, s+"x = lengths(s)", 3); got != "1 1 1" {
		t.Errorf("lengths: %s", got)
	}
	// a chord: the highest note is the one that play gets
	if got := pull(t, "s = together([phrase([60], [1]), phrase([67], [1])])\nx = pitches(s)", 2); got != "67 67" {
		t.Errorf("%s", got)
	}
	// it plays: a synth with a score, and play with its rhythm and pitches
	for _, src := range []string{
		"out = subtractive(phrase([60 62 64 65], [2]), preset: \"pluck\", voices: 3)",
		"fn inst(gate, n) { sine(hz(n)) * gate * 0.2 }\ns = phrase([60 62 64 65], [2])\nout = play(inst, rhythm: onsets(s), legato: 0.5, fields: [pitches(s)]).dup()",
	} {
		ch := renderPatch(t, "bpm = 120\n"+src+"\n", 2)
		if e := energy(ch[0][4000:60000]); e < 1e-5 {
			t.Errorf("%s is silent: %v", src, e)
		}
	}
}

func TestMistakesInScores(t *testing.T) {
	for src, want := range map[string]string{
		`phrase([60], [0])`: "above 0",
		`phrase([60 62], [1], velocity: 0.5, legato: 0.5)`:       "",
		`stretch(phrase([60], [1]), 0)`:                          "above 0",
		`cut(phrase([60], [1]), 5, 2)`:                           "after from",
		`tile(phrase([60], [1]), 0)`:                             "n is from 1",
		`pad(phrase([60], [1]), 0)`:                              "steps is from 1",
		`canon(phrase([60], [1]), 4, voices: 2, voice: 5)`:       "2 voices",
		`phasing(phrase([60 62], [1]), 5)`:                       "below the length",
		`isorhythm([0], [60], 4)`:                                "no notes",
		`fugue([61], [4], voices: 2, root: 60, scale: major)`:    "key",
		`fugue([60 62], [4], voices: 2, root: 60, scale: major)`: "a duration for each",
		`chordtones([], 60, major)`:                              "no chords",
		`midifile("x.mid")`:                                      "there is no file",
		`transpose(5, 1)`:                                        "score",
	} {
		_, err := Compile("seed = 1\nx = " + src + "\nout = sine(440).dup()\n")
		if want == "" {
			if err != nil {
				t.Errorf("%s: %v", src, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}
