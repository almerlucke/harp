package lang

import (
	"strings"
	"testing"
)

func TestCombiningGenerators(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		// in turn, round
		{"alternate([degrees(60, major, [0 1 2]), degrees(70, major, [0 1])])", "60 70 62 72 64 70"},
		{"alternate([euclid(4, 1), euclid(4, 2)])", "xx.."},
		{"stutter(degrees(60, major, [0 1 2]), 2)", "60 60 62 62 64 64 60 60"},
		{"maybe(every(2), 1)", "x.x.x.x."},
		{`pattern("x..x ..x. | x---")`, "x..x..x.x..."},
		{"maybe(every(2), 0)", "........"},
		{"invert(every(4))", ".xxx.xxx"},
		{"both([euclid(8, 4), euclid(8, 2)])", "x...x..."},
		{"either([every(4), every(3)])", "x..xx.x.xx.."},
		{"meter([4], 0)", "xxxx"},
		{"meter([4], 3)", "x..."},
		{"meter([2 2], 2)", "x.x."},
		{"sieve(8, [4], [1])", ".x...x.."},
		{"sieve(8, [4 3], [0 1])", "xx..x..x"},
		{"stack([degrees(60, major, [0 2]), degrees(67, major, [0 1])])", "[60 67] [64 69] [60 67] [64 69]"},
	} {
		if c.want == "" {
			continue
		}
		n := len(c.want) // a rhythm: a character for each step
		if strings.Contains(c.want, " ") {
			n = len(strings.Split(c.want, " "))
			if strings.Contains(c.want, "[") { // chords: a space inside each
				n = strings.Count(c.want, "[")
			}
		}
		if got := pull(t, c.src, n); got != c.want {
			t.Errorf("%s:\n got  %s\n want %s", c.src, got, c.want)
		}
	}
}

func TestRandomGeneratorsAreSeededAndInRange(t *testing.T) {
	a := pull(t, "anyOf([degrees(60, major, [0]), degrees(72, major, [0])])", 16)
	b := pull(t, "anyOf([degrees(60, major, [0]), degrees(72, major, [0])])", 16)
	if a != b {
		t.Errorf("the same seed gives another stream: %s and %s", a, b)
	}
	if !strings.Contains(a, "60") || !strings.Contains(a, "72") {
		t.Errorf("anyOf picks one of them: %s", a)
	}
	// weights: a is three times as likely as b, and a weight of 0 is never chosen
	w := pull(t, "chance([degrees(60, major, [0]), degrees(72, major, [0])], [1 0])", 20)
	if strings.Contains(w, "72") {
		t.Errorf("a weight of 0: %s", w)
	}
	// thinning leaves a rhythm with fewer hits
	if h := strings.Count(pull(t, "maybe(every(1), 0.5)", 200), "x"); h < 60 || h > 140 {
		t.Errorf("half of 200: %d", h)
	}
	for _, src := range []string{"uniform(3, 4)", "deal([1 2 3 4])", "shotgun(degrees(60, major, [0 1 2 3]), 0.5)", "noRepeat(uniform(1, 1.0001))", "distinct(degrees(60, major, mutate([0 1 2 3], 1)), 4)"} {
		pull(t, src, 12) // no panic, no hang
	}
	if d := pull(t, "deal([1 2 3 4])", 4); len(strings.Fields(d)) != 4 || strings.Count(d, "1") != 1 || strings.Count(d, "4") != 1 {
		t.Errorf("a deck has each card once: %s", d)
	}
}

func TestCombiningMistakesAreSaid(t *testing.T) {
	for src, want := range map[string]string{
		"out = alternate([1 2 3]).dup()":                             "generator",
		"out = chance([every(2), every(3)], [1]).dup()":              "2 generators and 1 weights",
		"out = stutter(every(2), 0).dup()":                           "1 to 64",
		`out = pattern("x.q").dup()`:                                 "unexpected",
		`out = pattern("  ").dup()`:                                  "no steps",
		"out = both([]).dup()":                                       "no rhythms",
		"out = meter([0], 0).dup()":                                  "1 to 64",
		"out = sieve(8, [3, 4], [1]).dup()":                          "as many starts",
		"out = noRepeat(every(2)).dup()":                             "does not take",
		"out = alternate([every(2), degrees(60, major, [0])]).dup()": "kind of the first",
	} {
		_, err := Compile(src)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}
