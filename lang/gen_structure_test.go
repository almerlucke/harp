package lang

import (
	"strings"
	"testing"
)

func TestAnLSystemIsAPhrase(t *testing.T) {
	// Lindenmayer's algae: A becomes AB, B becomes A: ABAABABAABAAB...
	got := pull(t, `degrees(60, major, lsystem("A", ["A=AB", "B=A"], 5, ["A=0", "B=1"]))`, 13)
	if want := "60 62 60 60 62 60 62 60 60 62 60 60 62"; got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	// symbols without a value (the brackets) are left out
	if got := pull(t, `degrees(60, major, lsystem("A[B]", ["A=AA"], 1, ["A=0", "B=2"]))`, 4); got != "60 60 64 60" {
		t.Errorf("%s", got)
	}
}

func TestAGrammarMakesAPhraseEveryRound(t *testing.T) {
	// a rule without choices is always the same
	if got := pull(t, `grammar(["S=A B", "A=60 62", "B=67"])`, 6); got != "60 62 67 60 62 67" {
		t.Errorf("%s", got)
	}
	// choices with weights: 0 is never taken
	got := pull(t, `grammar(["S=1: 60 | 0: 99"], depth: 2)`, 20)
	if strings.Contains(got, "99") {
		t.Errorf("a weight of 0: %s", got)
	}
	// both choices come up
	got = pull(t, `grammar(["S=60 | 72"])`, 40)
	if !strings.Contains(got, "60") || !strings.Contains(got, "72") {
		t.Errorf("%s", got)
	}
}

func TestAutomataAndLifeMakeRhythms(t *testing.T) {
	// rule 90 from one cell: a triangle; the first row is the cell in the middle
	if got := pull(t, `cellular(90, 8)`, 16); len(got) != 16 || strings.Count(got[:8], "x") != 1 || strings.Count(got[8:], "x") != 2 {
		t.Errorf("the first row is one cell, the second has its two neighbours: %s", got)
	}
	// rule 0 kills everything after the first row
	if got := pull(t, `cellular(0, 8)`, 16); strings.Count(got[8:], "x") != 0 {
		t.Errorf("%s", got)
	}
	if got := pull(t, `life(16, 4, density: 0.5)`, 64); !strings.Contains(got, "x") || !strings.Contains(got, ".") {
		t.Errorf("a grid with half of its cells alive: %s", got)
	}
	if got := pull(t, `scatter([1])`, 8); got != "xxxxxxxx" {
		t.Errorf("%s", got)
	}
	if got := pull(t, `scatter([0])`, 8); got != "........" {
		t.Errorf("%s", got)
	}
	if h := strings.Count(pull(t, `scatter(sweep(0, 1, 100))`, 100), "x"); h < 25 || h > 75 {
		t.Errorf("a chance that rises from 0 to 1 hits half the time over 100 steps: %d", h)
	}
}

func TestChainsOfChance(t *testing.T) {
	// a state that can only go to the other: 60 62 60 62
	if got := pull(t, `markov([60 62], [[0 1] [1 0]])`, 6); got != "60 62 60 62 60 62" {
		t.Errorf("%s", got)
	}
	// 60 can stay: after 60 comes 60 or 62
	if got := pull(t, `markov([60 62], [[1 1] [1 0]])`, 40); !strings.Contains(got, "60 60") {
		t.Errorf("%s", got)
	}
	// an example that is a cycle gives the cycle (order 1 of 1 2 3 1 2 3: every value has one follower)
	if got := pull(t, `learn([1 2 3 1 2 3 1 2 3], 1)`, 9); got != "2 3 1 2 3 1 2 3 1" {
		t.Errorf("%s", got)
	}
	// a walk takes steps to a neighbour
	x := nums(t, `stroll([0 1 2 3 4 5 6 7])`, 60)
	for i := 1; i < len(x); i++ {
		if d := x[i] - x[i-1]; d != 1 && d != -1 {
			t.Fatalf("a step of %v at %d: %v", d, i, x)
		}
	}
	// a grid: a step is one cell along one direction (a neighbour in the row or in the column)
	g := nums(t, `stroll([0 1 2 3 4 5 6 7 8], shape: [3 3])`, 60)
	for i := 1; i < len(g); i++ {
		if d := g[i] - g[i-1]; d != 1 && d != -1 && d != 3 && d != -3 {
			t.Fatalf("a step of %v at %d: %v", d, i, g)
		}
	}
}

func TestMistakesInStructure(t *testing.T) {
	for src, want := range map[string]string{
		`lsystem("A", ["A=AB"], 3, ["C=1"])`:       "no symbol",
		`lsystem("A", ["AA=B"], 3, ["A=1"])`:       "one symbol",
		`lsystem("A", ["A"], 3, ["A=1"])`:          "like",
		`lsystem("A", ["A=A"], 99, ["A=1"])`:       "0 to 32",
		`lsystem("A", ["A=A"], 3, ["A=x"])`:        "value is like",
		`grammar(["S=A"])`:                         "not a number",
		`grammar(["S=1"], start: "T")`:             "no rule T",
		`grammar(["S=x: 60"])`:                     "weight",
		`cellular(300)`:                            "0 to 255",
		`cellular(30, start: "many")`:              "one or random",
		`life(8, 8, row: 8)`:                       "row",
		`life(8, 8, rule: "xyz")`:                  "life",
		`markov([60 62], [[1 1]])`:                 "a row for each",
		`markov([60 62], [[0 0] [1 1]])`:           "row 1 has no weight",
		`markov([60 62], [[1 1] [1 1]], start: 5)`: "start is from",
		`learn([1 2 3], 3)`:                        "order is from 1 to 2",
		`stroll([1 2 3], shape: [2 2])`:            "needs 4",
	} {
		_, err := Compile("seed = 1\nx = " + src + "\nout = sine(440).dup()\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}
