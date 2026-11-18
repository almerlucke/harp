package lang

import (
	"fmt"
	"strings"
	"testing"

	"github.com/almerlucke/genny/harmony"
)

// voicingsOf is the four voices of a harmonize, as the notes of each chord
func voicingsOf(t *testing.T, args string) []harmony.Voicing {
	t.Helper()
	var voices [4][]float64
	for k := 0; k < 4; k++ {
		for _, e := range evalLast(t, fmt.Sprintf("seed = 5\nx = harmonize(%s, voice: %d)", args, k)).V.(*Score).Events {
			voices[k] = append(voices[k], e.Pitch)
		}
	}
	out := make([]harmony.Voicing, len(voices[0]))
	for i := range out {
		for k := 0; k < 4; k++ {
			out[i].Notes[k] = int(voices[k][i])
		}
	}
	return out
}

func TestHarmonizeWritesFourParts(t *testing.T) {
	// I IV V I in C major (degrees 0 3 4 0): the bass is on the roots, and the rules of four part writing are kept
	v := voicingsOf(t, "60, degrees: [0 3 4 0]")
	if len(v) != 4 {
		t.Fatalf("%d chords", len(v))
	}
	for i, pc := range []int{0, 5, 7, 0} {
		if v[i].Notes[3]%12 != pc {
			t.Errorf("chord %d: the bass is %d, the root is %d", i+1, v[i].Notes[3]%12, pc)
		}
		for k := 0; k < 3; k++ {
			if v[i].Notes[k] < v[i].Notes[k+1] {
				t.Errorf("chord %d: the voices cross: %v", i+1, v[i].Notes)
			}
		}
	}
	for i, d := range []int{0, 3, 4, 0} {
		v[i].Chord = harmony.Diatonic(harmony.Key{Tonic: 0}, d, false)
	}
	if broken := harmony.Check(v); len(broken) > 0 {
		t.Errorf("rules broken: %v", broken)
	}
	// chords that are made by the grammar of functions: 12 of them, in a minor key, no rule broken, and they begin and end on the tonic
	g := voicingsOf(t, "57, mode: \"minor\", n: 12, cadence: \"authentic\", sevenths: 0.5")
	if len(g) != 12 {
		t.Fatalf("%d chords", len(g))
	}
	if g[0].Notes[3]%12 != 9 || g[len(g)-1].Notes[3]%12 != 9 {
		t.Errorf("not on the tonic: %v ... %v", g[0].Notes, g[len(g)-1].Notes)
	}
	// (what the rules of four part writing could not keep is said by harp check)
	prog, _ := Parse("seed = 5\nx = harmonize(57, mode: \"minor\", n: 12, cadence: \"authentic\", sevenths: 0.5)\nout = sine(1).dup()\n")
	p, err := buildPatch(prog, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if info := strings.Join(p.Info, "\n"); strings.Contains(info, "had to be broken") || !strings.Contains(info, "A minor") {
		t.Errorf("harp check says: %q", info)
	}
	if a, b := scoreText(t, "harmonize(60, n: 6)"), scoreText(t, "harmonize(60, n: 6)"); a != b || a == "" {
		t.Errorf("the same seed is other chords")
	}
}

func TestHarmonizeIsAScoreForSynthsAndSaysWhatItMade(t *testing.T) {
	prog, _ := Parse("seed = 2\nx = harmonize(60, degrees: [0 4 0])\nout = subtractive(x, preset: \"pad\", voices: 12).gain(0.1)\n")
	p, err := buildPatch(prog, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if info := strings.Join(p.Info, "\n"); !strings.Contains(info, "I V I in C major") {
		t.Errorf("harp check says: %q", info)
	}
	ch := renderPatch(t, "bpm = 120\nx = harmonize(60, degrees: [0 3 4 0], chordSteps: 8)\nout = subtractive(x, preset: \"pad\", voices: 8)\n", 3)
	if e := energy(ch[0][4000:60000]); e < 1e-5 {
		t.Errorf("silent: %v", e)
	}
}

func TestMistakesInHarmonize(t *testing.T) {
	for src, want := range map[string]string{
		`harmonize(60, mode: "dorian")`:         "major or minor",
		`harmonize(60, degrees: [7])`:           "from 0 (I) to 6",
		`harmonize(60, degrees: [1.5])`:         "from 0 (I) to 6",
		`harmonize(60, n: 0)`:                   "1 to 256",
		`harmonize(60, cadence: "halt")`:        "did you mean half",
		`harmonize(60, sevenths: 2)`:            "chance",
		`harmonize(60, degrees: [0], voice: 7)`: "4 voices",
		`harmonize(60, chordSteps: 0)`:          "chordSteps",
	} {
		_, err := Compile("seed = 1\nx = " + src + "\nout = sine(440).dup()\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}
