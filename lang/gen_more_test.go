package lang

import (
	"strings"
	"testing"
)

func TestNumberSequencesAndTime(t *testing.T) {
	for src, want := range map[string]string{
		`fibonacci(8)`:                     "0 1 1 2 3 5 0 5 5 2 7 1 0",
		`thuemorse()`:                      "0 1 1 0 1 0 0 1",
		`primes()`:                         "2 3 5 7 11 13",
		`collatz(6)`:                       "6 3 10 5 16 8 4 2 1 6",
		`divisible(fibonacci(8), 2)`:       "x..x..x..x..",
		`divide(3)`:                        "x..x..x..",
		`divide(4, 2)`:                     "..x...x.",
		`talea([3 2 3])`:                   "x..x.x..x..x.x..",
		`ratchets([1 2], 4)`:               "x...x.x.x...x.x.",
		`ratchets([0 4], 4)`:               "....xxxx....xxxx",
		`jump([0 1 2 3 4 5 6 7], 3, 1, 4)`: "0 1 2 4 5 6 0 1 2",
	} {
		n := len(strings.Fields(want))
		if !strings.Contains(want, " ") {
			n = len(want)
		}
		if got := pull(t, src, n); got != want {
			t.Errorf("%s:\n got  %s\n want %s", src, got, want)
		}
	}
}

func TestDurationsAreCutUp(t *testing.T) {
	if got := listText(t, `metric(4, 4)`); got != "1 1 1 1" {
		t.Errorf("%s", got)
	}
	total := 0.0
	for _, it := range evalLast(t, "seed = 2\nx = split(4, 0.25, 0.7, 2)").Items() {
		v := it.Float()
		total += v
		if v < 0.25-1e-9 {
			t.Errorf("a part under the lower limit: %v", v)
		}
	}
	if total < 4-1e-9 || total > 4+1e-9 {
		t.Errorf("the parts add up to the length: %v", total)
	}
}

func TestSectionsEvolutionAndMasks(t *testing.T) {
	if got := pull(t, `sections([degrees(60, major, [0]), degrees(72, major, [0])], [2 3])`, 10); got != "60 60 72 72 72 60 60 72 72 72" {
		t.Errorf("%s", got)
	}
	// a phrase of 8 notes in range, that is the same every round (the best phrase), and a smoother one than chance
	ph := nums(t, `evolve(length: 8, lo: 0, hi: 7, generations: 0)`, 16)
	for i := 0; i < 8; i++ {
		if ph[i] != ph[i+8] || ph[i] < 0 || ph[i] > 7 {
			t.Errorf("%v", ph)
			break
		}
	}
	jumps := func(p []float64) (n float64) {
		for i := 1; i < len(p); i++ {
			if d := p[i] - p[i-1]; d > 0 {
				n += d
			} else {
				n -= d
			}
		}
		return
	}
	evolved := nums(t, `evolve(length: 12, lo: 0, hi: 15, generations: 60, smooth: 1)`, 12*8)
	if j := jumps(evolved[12*7:]); j > 30 {
		t.Errorf("after 7 rounds a smooth phrase should have small steps: %v", j)
	}
	// the band of a mask: over 8 values from a narrow band to a wide one
	m := within(t, `tendency([[8, 60, 60, 48, 72]], loop: false)`, 8, 48, 72)
	if m[0] != 60 {
		t.Errorf("the first band is one note: %v", m[0])
	}
}

func TestMistakesInNumbersAndTime(t *testing.T) {
	for src, want := range map[string]string{
		`fibonacci(1)`:                "2 to 4096",
		`collatz(0)`:                  "from 1",
		`divide(0)`:                   "1 or more",
		`talea([0 2])`:                "1 to 256",
		`ratchets([2], 0)`:            "1 to 64",
		`ratchets([1.5])`:             "whole number",
		`jump([1 2], 0)`:              "chunk",
		`split(4, 0, 0.5, 2)`:         "above 0",
		`metric(0, 2)`:                "above 0",
		`sections([every(2)], [4 4])`: "a length for each",
		`sections([every(2)], [0])`:   "from 1",
		`evolve(smooth: 0)`:           "weight above 0",
		`evolve(lo: 5, hi: 5)`:        "lo is below hi",
		`tendency([[8, 60, 60, 48]])`: "segment 1",
		`tendency([])`:                "no segments",
	} {
		_, err := Compile("seed = 1\nx = " + src + "\nout = sine(440).dup()\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}
