package lang

import (
	"strings"
	"testing"
)

func TestMoreScalesAndTunings(t *testing.T) {
	// the scales that were named
	if got := pull(t, `degrees(60, phrygian, [0 1 2 7])`, 4); got != "60 61 63 72" {
		t.Errorf("phrygian: %s", got)
	}
	if got := pull(t, `degrees(60, scale([0 2 3 7 8]), [0 1 2 3 4 5])`, 6); got != "60 62 63 67 68 72" {
		t.Errorf("a scale of your own: %s", got)
	}
	if got := pull(t, `degrees(60, equal(24), [0 1 2 24])`, 4); got != "60 60.5 61 72" {
		t.Errorf("quarter tones: %s", got)
	}
	// a just major scale: the fifth is the pure 3/2: 7.0196 semitones
	got := nums(t, `degrees(60, just([1 1.125 1.25 1.3333333333 1.5]), [0 1 2 3 4])`, 5)
	if d := got[4] - 60 - 7.0195500086; d > 1e-6 || d < -1e-6 {
		t.Errorf("the pure fifth: %v", got)
	}
}

func TestChordsAndLeading(t *testing.T) {
	x := evalLast(t, `x = chord(60, "minor")`)
	if x.T.Kind != KList || len(x.Items()) != 3 || x.Items()[1].Float() != 63 {
		t.Errorf("%v", x.V)
	}
	if inv := evalLast(t, `x = chord(60, "major", inversion: 1)`).Items(); inv[0].Float() != 64 || inv[2].Float() != 72 {
		t.Errorf("the first inversion of C major: %v", inv)
	}
	// the triads of A minor on i VI iv v
	if got := pull(t, `triads(57, minor, [0 5 3 4])`, 4); got != "[57 60 64] [65 69 72] [62 65 69] [64 67 71]" {
		t.Errorf("%s", got)
	}
	if got := pull(t, `triads(57, minor, [0], size: 4)`, 1); got != "[57 60 64 67]" {
		t.Errorf("sevenths: %s", got)
	}
	// the voice leader keeps the chords near each other: no voice leaps more than a fifth between chords
	led := pull(t, `voiceLead(triads(57, minor, [0 5 3 4 0 5 3 4]), 48, 72)`, 8)
	prev := []float64(nil)
	for _, c := range strings.Split(strings.Trim(led, "[]"), "] [") {
		var cur []float64
		for _, f := range strings.Fields(c) {
			var v float64
			if _, err := fmtSscan(f, &v); err != nil {
				t.Fatal(err)
			}
			cur = append(cur, v)
		}
		if prev != nil {
			if span := cur[len(cur)-1] - prev[len(prev)-1]; span > 7 || span < -7 {
				t.Errorf("the top voice leaps %v: %s", span, led)
			}
		}
		prev = cur
	}
}

func TestArpeggiosProgressionsAndRows(t *testing.T) {
	if got := pull(t, `arp([60 64 67 72])`, 8); got != "60 64 67 72 60 64 67 72" {
		t.Errorf("%s", got)
	}
	if got := pull(t, `arp([60 64 67], mirror: "exclusive")`, 8); !strings.HasPrefix(got, "60 64 67 64") {
		t.Errorf("%s", got)
	}
	if got := pull(t, `progression("classical")`, 30); !strings.Contains(got, "4 0") {
		t.Errorf("a dominant goes to the tonic in the classical style: %s", got)
	}
	// a tone row has each pitch class once; its retrograde is the row backwards, its inversion is the mirror
	r := evalLast(t, "seed = 3\nx = toneRow()").Items()
	seen := map[float64]bool{}
	for _, v := range r {
		seen[v.Float()] = true
	}
	if len(r) != 12 || len(seen) != 12 {
		t.Errorf("a row: %v", r)
	}
	if got := listText(t, `rowForm([0 2 7 1], "retro")`); got != "1 7 2 0" {
		t.Errorf("retrograde: %s", got)
	}
	if got := listText(t, `rowForm([0 2 7], "prime", 5)`); got != "5 7 12" {
		t.Errorf("transposed: %s", got)
	}
	// counterpoint: a consonant line against the cantus, as long as the cantus, starting on a perfect consonance
	cf := []int{60, 62, 64, 62, 60}
	line := evalLast(t, "seed = 1\nx = counterpoint([60 62 64 62 60])").Items()
	if len(line) != len(cf) {
		t.Fatalf("%v", line)
	}
	for i, v := range line {
		d := (int(v.Float()) - cf[i]) % 12
		if d < 0 {
			d += 12
		}
		if d == 1 || d == 2 || d == 5 || d == 6 || d == 10 || d == 11 {
			t.Errorf("note %d is a dissonance against the cantus (%d semitones)", i+1, d)
		}
	}
	// a melody that walks stays in range, and snapped numbers are notes of the scale
	for _, v := range within(t, `stepwise(60, major, lo: -3, hi: 3, step: 1, leap: 0.3, maxLeap: 3)`, 100, 55, 67) {
		if v != float64(int(v)) {
			t.Errorf("a note of C major is a whole number: %v", v)
		}
	}
	for _, v := range nums(t, `snap(wander(60, 72, 0.5), 60, wholetone)`, 60) {
		if int(v)%2 != 0 {
			t.Errorf("not in the whole tone scale: %v", v)
		}
	}
}

func TestMistakesInPitch(t *testing.T) {
	for src, want := range map[string]string{
		`scale([1 2 3])`:                  "first step is 0",
		`scale([0 2 2])`:                  "rise",
		`scale([0 12])`:                   "below 12",
		`equal(0)`:                        "1 to 96",
		`just([1 1.5 1.2])`:               "rise",
		`just([1.1 1.5])`:                 "first ratio is 1",
		`stepwise(60, major, start: 20)`:  "between them",
		`chord(60, "minr")`:               "did you mean minor",
		`triads(60, major, [0], size: 1)`: "2 to 7",
		`arp([60 64], mode: "sideways")`:  "one of",
		`progression("polka")`:            "one of",
		`rowForm([1 2], "upside")`:        "one of",
		`counterpoint([60.5 62])`:         "whole",
	} {
		_, err := Compile("seed = 1\nx = " + src + "\nout = sine(440).dup()\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}
