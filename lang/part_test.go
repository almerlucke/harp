package lang

import (
	"fmt"
	"strings"
	"testing"
)

func TestChordsWrittenAsText(t *testing.T) {
	for src, want := range map[string]string{
		// symbols, from C3: the root is the first of its note at or above, a chord lasts a bar (the list has one for each bar)
		`changes("Dm7 G7*2 Cmaj7")`: "50 53 57 60 | 55 59 62 65 | 55 59 62 65 | 48 52 55 59",
		// roman numerals in a key
		`changes("i VI", key: "A minor")`: "57 60 64 | 53 57 60",
		// a - holds the chord for another bar, and . is a rest
		`changes("C - . F")`: "48 52 55 | 48 52 55 |  | 53 57 60",
		// a slash chord has its bass below the root
		`changes("G/B")`: "47 55 59 62",
		// the lowest note
		`changes("C", lowest: 60)`: "60 64 67",
	} {
		var bars []string
		for _, c := range evalLast(t, "x = "+src).Items() {
			var ps []string
			for _, n := range c.Items() {
				ps = append(ps, strings.TrimSuffix(strings.TrimSuffix(formatNum(n.Float()), ".0"), ".00"))
			}
			bars = append(bars, strings.Join(ps, " "))
		}
		if got := strings.Join(bars, " | "); got != want {
			t.Errorf("%s:\n got  %s\n want %s", src, got, want)
		}
	}
	// lead: each chord is the voicing nearest the one before: it moves less than in root position
	move := func(lead string) float64 {
		cs := evalLast(t, "x = changes(\"C Am F G\", lead: "+lead+")").Items()
		total := 0.0
		for i := 1; i < len(cs); i++ {
			a, b := cs[i-1].Items(), cs[i].Items()
			for k := range a {
				total += abs(a[k].Float() - b[k].Float())
			}
		}
		return total
	}
	if move("true") >= move("false") {
		t.Errorf("voice leading moves %v, root position %v", move("true"), move("false"))
	}
	for src, want := range map[string]string{
		`changes("Dm7*0.5")`:       "whole number of bars",
		`changes("")`:              "no chords",
		`changes("Hm7")`:           "a chord",
		`changes("- C")`:           "without a chord before it",
		`changes("C", key: "X y")`: "the key is like",
	} {
		_, err := Compile("seed = 1\nx = " + src + "\nout = sine(440).dup()\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func formatNum(x float64) string {
	return strings.TrimRight(strings.TrimRight(strings.Replace(fmt.Sprintf("%.3f", x), ",", ".", 1), "0"), ".")
}

func TestAPartIsAMelodyAndItsRhythm(t *testing.T) {
	for src, want := range map[string]string{
		// notes, a rest, a hold (the note before goes on to the end of the step)
		`part("C3 Eb3 G3 - Bb3 . G3 Eb3")`: "0:48/0.9 1:51/0.9 2:55/2 4:58/0.9 6:55/0.9 7:51/0.9",
		// numbers, and names with a sharp, and a negative octave
		`part("60 c#4 bb3")`: "0:60/0.9 1:61/0.9 2:58/0.9",
		// a chord, and an octave up and down
		`part("C3+E3+G3 C3' C3,")`: "0:48/0.9 0:52/0.9 0:55/0.9 1:60/0.9 2:36/0.9",
		// how long: *2 two steps
		`part("C3*2 D3")`: "0:48/2 2:50/0.9",
		// chance: ?0 never
		`part("C3?0 D3")`: "1:50/0.9",
		// the conditions: the text is played as many times as they need (4 passes of 2 steps: C3 only in the second)
		`part("C3%2/4 D3")`: "1:50/0.9 2:48/0.9 3:50/0.9 5:50/0.9 7:50/0.9",
	} {
		if got := scoreText(t, src); got != want {
			t.Errorf("%s:\n got  %s\n want %s", src, got, want)
		}
	}
	// a group fills a step, a roll too: three notes in it
	for _, src := range []string{`part("[C3 D3 E3] .")`, `part("C3^3 .")`} {
		ev := evalLast(t, "x = "+src).V.(*Score).Events
		if len(ev) != 3 || ev[0].Start != 0 || ev[1].Start < 0.33 || ev[1].Start > 0.34 || ev[2].Start < 0.66 || ev[2].Start > 0.67 {
			t.Errorf("%s: %v", src, ev)
		}
	}
	// chord tones that follow the chords: Am for a bar of 4 steps, then F
	for src, want := range map[string]string{
		`part("@1 @3 @1' @c @1 @3 @2 @4", chords: "Am F", barsteps: 4)`: "0:57/0.9 1:64/0.9 2:69/0.9 3:57/0.9 3:60/0.9 3:64/0.9 4:53/0.9 5:60/0.9 6:57/0.9 7:65/0.9",
		`part("@b @b", chords: "Am F", barsteps: 1)`:                    "0:45/0.9 1:41/0.9",
		`part("@-1 @1", chords: "C", barsteps: 2)`:                      "0:43/0.9 1:48/0.9",
		`part("@1 @1", chords: "i VI", key: "A minor", barsteps: 1)`:    "0:57/0.9 1:53/0.9",
		`part("@1 @1", chords: ". C", barsteps: 1)`:                     "1:48/0.9",
	} {
		if got := scoreText(t, src); got != want {
			t.Errorf("%s:\n got  %s\n want %s", src, got, want)
		}
	}
	// it plays on a synth, and with the notes it has
	if x := renderPatch(t, "bpm = 120\nout = subtractive(part(\"C3 Eb3 G3 - Bb3 . G3 Eb3\"), preset: \"pluck\", voices: 4)\n", 3)[0]; energy(x[2000:90000]) < 1e-6 {
		t.Error("a part is silent on a synth")
	}
	for src, want := range map[string]string{
		`part("H3")`:               "is not a note",
		`part("@1")`:               "no chords",
		`part("@9x", chords: "C")`: "@1 is the root",
		`part("C3", chords: "Hm")`: "chords:",
		`part("[C3")`:              "without its ]",
		`part("C3*0")`:             "above 0",
		`part("C3", passes: 100)`:  "1 to 64",
		`part("C3", velocity: 2)`:  "velocity from 0 to 1",
	} {
		_, err := Compile("seed = 1\nx = " + src + "\nout = sine(440).dup()\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}

func TestTheOrdersOfAList(t *testing.T) {
	for src, want := range map[string]string{
		`order([1 2 3 4])`:                "1 2 3 4 1 2 3 4",
		`order([1 2 3 4], "backwards")`:   "4 3 2 1 4 3 2 1",
		`order([1 2 3 4], "converge")`:    "1 4 2 3 1 4 2 3",
		`order([1 2 3 4], "alternate")`:   "1 2 1 3 1 4 1 2",
		`order([1 2 3 4], "up and down")`: "1 2 3 4 3 2 1 2",
		`order([1 2 3], "shuffled")`:      "",
	} {
		if want == "" {
			continue
		}
		if got := pull(t, src, len(strings.Fields(want))); got != want {
			t.Errorf("%s:\n got  %s\n want %s", src, got, want)
		}
	}
	// shuffled: every value once in each deck of 4, random: any
	for deck := 0; deck < 1; deck++ {
		f := strings.Fields(pull(t, `order([1 2 3 4], "shuffled")`, 16))
		for d := 0; d < 4; d++ {
			seen := map[string]bool{}
			for _, v := range f[d*4 : d*4+4] {
				seen[v] = true
			}
			if len(seen) != 4 {
				t.Errorf("a deck has a card twice: %v", f[d*4:d*4+4])
			}
		}
		if strings.Join(f[:4], "") == strings.Join(f[4:8], "") && strings.Join(f[4:8], "") == strings.Join(f[8:12], "") && strings.Join(f[8:12], "") == strings.Join(f[12:], "") {
			t.Errorf("four decks in the same order: %v", f)
		}
	}
	if f := strings.Fields(pull(t, `pick([1 2 3])`, 60)); len(f) != 60 {
		t.Errorf("%v", f)
	} else {
		seen := map[string]bool{}
		for _, v := range f {
			seen[v] = true
		}
		if len(seen) != 3 {
			t.Errorf("60 picks of 3 values: %v", seen)
		}
	}
	for src, want := range map[string]string{
		`order([1 2 3], "sideways")`: "there is no order",
		`order([1], "converge")`:     "two values or more",
		`order([])`:                  "no numbers",
	} {
		_, err := Compile("seed = 1\nx = " + src + "\nout = sine(440).dup()\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}
