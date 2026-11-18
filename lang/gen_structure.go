package lang

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"harp/dsl"

	"github.com/almerlucke/genny/automaton"
	gg "github.com/almerlucke/genny/grammar"
	"github.com/almerlucke/genny/lsystem"
	"github.com/almerlucke/genny/markov"
	"github.com/almerlucke/genny/ngram"
	"github.com/almerlucke/genny/stochastic"
	"github.com/almerlucke/genny/walk"
)

// Generators that make a shape from rules: rewriting (L-systems, grammars), cellular automata, chains of chance (markov, n-grams) and walks

// pairs reads a list of "x=value" strings: "A=AB", "A=0"
func pairs(v Value, what string) ([][2]string, error) {
	var out [][2]string
	for i, it := range v.Items() {
		if it.T.Kind != KStr {
			return nil, fmt.Errorf("%s: item %d is a %s: write it as a string, like \"A=AB\"", what, i+1, it.T)
		}
		k, val, ok := strings.Cut(it.V.(string), "=")
		k = strings.TrimSpace(k)
		if !ok || k == "" {
			return nil, fmt.Errorf("%s: %q is not like \"A=AB\"", what, it.V.(string))
		}
		out = append(out, [2]string{k, strings.TrimSpace(val)})
	}
	return out, nil
}

func numbersOf(v Value, what string) ([]float64, error) {
	x, err := floats(v, what)
	if err != nil {
		return nil, err
	}
	if len(x) == 0 {
		return nil, fmt.Errorf("%s: the list has no numbers", what)
	}
	return x, nil
}

// markovGen: a chain of states with weights; every state has the value of an item of the list
func markovChain(values []float64, weights [][]float64, start int) (dsl.Gen[float64], error) {
	states := make([]*markov.ProbabilityState[float64], len(values))
	for i, v := range values {
		states[i] = markov.NewProbabilityState(v)
	}
	for i, row := range weights {
		args := make([]any, 0, 2*len(row))
		total := 0.0
		for j, w := range row {
			if w < 0 || math.IsNaN(w) {
				return nil, fmt.Errorf("a weight is 0 or more")
			}
			total += w
			args = append(args, states[j], w)
		}
		if total == 0 {
			return nil, fmt.Errorf("row %d has no weight: every state needs somewhere to go", i+1)
		}
		states[i].SetProbabilities(args...)
	}
	return markov.New[float64](starter{states[start]}, 1), nil
}

type starter struct {
	s *markov.ProbabilityState[float64]
}

func (s starter) Start([]markov.State[float64]) markov.State[float64] { return s.s }

// forever makes a generator that ends go round (a chain that has no end does not need it)
func init() {
	add(&Func{Name: "lsystem", Result: ListOf(TNum), Doc: "an L-system as a list of numbers: the axiom is rewritten depth times with the rules, and the symbols that have a value are the numbers: lsystem(\"A\", [\"A=AB\", \"B=A\"], 6, [\"A=0\", \"B=2\"]) is a phrase that is like itself at every size. A list is a phrase wherever a generator of numbers is wanted",
		Params: []Param{{"axiom", TStr, nil}, {"rules", ListOf(TStr), nil}, {"depth", TNum, nil}, {"values", ListOf(TStr), nil}},
		Call: func(a Args) (Value, error) {
			rules, err := pairs(a.value("rules"), "rules")
			if err != nil {
				return Value{}, err
			}
			vals, err := pairs(a.value("values"), "values")
			if err != nil {
				return Value{}, err
			}
			m := map[rune]string{}
			for _, r := range rules {
				if len([]rune(r[0])) != 1 {
					return Value{}, fmt.Errorf("a rule is for one symbol, not %q", r[0])
				}
				m[[]rune(r[0])[0]] = r[1]
			}
			nums := map[rune]float64{}
			for _, v := range vals {
				x, err := strconv.ParseFloat(v[1], 64)
				if err != nil || len([]rune(v[0])) != 1 {
					return Value{}, fmt.Errorf("a value is like \"A=60\", not %q", v[0]+"="+v[1])
				}
				nums[[]rune(v[0])[0]] = x
			}
			if d := a.Int("depth"); d < 0 || d > 32 {
				return Value{}, fmt.Errorf("depth is from 0 to 32")
			}
			var out []Value
			for _, r := range lsystem.Expand(a.Str("axiom"), m, a.Int("depth")) {
				if x, ok := nums[r]; ok {
					out = append(out, numV(x))
				}
			}
			if len(out) == 0 {
				return Value{}, fmt.Errorf("no symbol of the result has a value (it is %q)", lsystem.Expand(a.Str("axiom"), m, a.Int("depth")))
			}
			return listV(out), nil
		}})
	add(&Func{Name: "grammar", Result: TGenFloat, Doc: "a grammar that makes a new phrase every time round: rules are like \"S=A B\" (a name on the left, on the right the names of other rules and numbers), with \"|\" between the choices, each with a weight in front if you like (\"A=3: 60 62 | 1: 67\"). start is the rule to begin with, depth how far it is rewritten (a rule that refers to itself needs one)",
		Params: []Param{{"rules", ListOf(TStr), nil}, {"start", TStr, defStr("S")}, {"depth", TNum, def(4)}},
		Call: func(a Args) (Value, error) {
			rules, err := pairs(a.value("rules"), "rules")
			if err != nil {
				return Value{}, err
			}
			g := gg.New[float64]()
			defined := map[string]bool{}
			for _, r := range rules {
				defined[r[0]] = true
			}
			for _, r := range rules {
				var alts []gg.Alt[float64]
				for _, alt := range strings.Split(r[1], "|") {
					weight := 1.0
					if w, rest, ok := strings.Cut(alt, ":"); ok {
						x, err := strconv.ParseFloat(strings.TrimSpace(w), 64)
						if err != nil || x < 0 {
							return Value{}, fmt.Errorf("the weight in %q is a number from 0", strings.TrimSpace(alt))
						}
						weight, alt = x, rest
					}
					var syms []gg.Symbol[float64]
					for _, tok := range strings.Fields(alt) {
						if defined[tok] {
							syms = append(syms, gg.Ref[float64](tok))
						} else if x, err := strconv.ParseFloat(tok, 64); err == nil {
							syms = append(syms, gg.Term(x))
						} else {
							return Value{}, fmt.Errorf("%q in the rule %s is not a number or the name of a rule%s", tok, r[0], didYouMean(tok, keysOf(defined)))
						}
					}
					alts = append(alts, gg.Choice(weight, syms...))
				}
				if len(alts) == 0 {
					return Value{}, fmt.Errorf("the rule %s has no choices", r[0])
				}
				total := 0.0
				for _, al := range alts {
					total += al.Weight
				}
				if total == 0 {
					return Value{}, fmt.Errorf("the weights of the rule %s are all 0", r[0])
				}
				g.Rule(r[0], alts...)
			}
			if !defined[a.Str("start")] {
				return Value{}, fmt.Errorf("there is no rule %s to start with (start: \"...\")%s", a.Str("start"), didYouMean(a.Str("start"), keysOf(defined)))
			}
			if err := g.Check(a.Str("start")); err != nil {
				return Value{}, fmt.Errorf("%v", err)
			}
			if d := a.Int("depth"); d < 0 || d > 16 {
				return Value{}, fmt.Errorf("depth is from 0 to 16")
			}
			return Value{TGenFloat, gg.NewPlayer(g, a.Str("start"), a.Int("depth"))}, nil
		}})
	add(&Func{Name: "cellular", Result: TGenBool, Doc: "a rhythm from an elementary cellular automaton (rule 0 to 255, 30 is chaotic, 90 a triangle of triangles, 110 complex): the cells of a row are the steps, and the next row comes after it. start: \"one\" is a single live cell in the middle, \"random\" starts from chance",
		Params: []Param{{"rule", TNum, nil}, {"width", TNum, def(16)}, {"start", TStr, defStr("one")}},
		Call: func(a Args) (Value, error) {
			r, w := a.Int("rule"), a.Int("width")
			if r < 0 || r > 255 || w < 1 || w > 4096 {
				return Value{}, fmt.Errorf("rule is from 0 to 255, width from 1 to 4096")
			}
			var au *automaton.Automaton
			switch a.Str("start") {
			case "one":
				au = automaton.New(uint8(r), w)
			case "random":
				au = automaton.NewRandom(uint8(r), w)
			default:
				return Value{}, fmt.Errorf("start is one or random")
			}
			return Value{TGenBool, automaton.NewGate(au)}, nil
		}})
	add(&Func{Name: "life", Result: TGenBool, Doc: "a rhythm from Conway's game of life: a grid of w by h cells, grown from chance (density, 0 to 1); the cells of one row are the steps, and the grid goes on a generation after a row. rule is in B/S notation (\"B3/S23\" is Life)",
		Params: []Param{{"w", TNum, def(16)}, {"h", TNum, def(8)}, {"density", TNum, def(0.3)}, {"row", TNum, def(0)}, {"rule", TStr, defStr("B3/S23")}},
		Call: func(a Args) (Value, error) {
			w, h, row := a.Int("w"), a.Int("h"), a.Int("row")
			if w < 1 || h < 1 || w > 256 || h > 256 || row < 0 || row >= h {
				return Value{}, fmt.Errorf("w and h are from 1 to 256, row is a row of the grid (0 to h-1)")
			}
			if _, err := automaton.ParseRule(a.Str("rule")); err != nil {
				return Value{}, fmt.Errorf("%v", err)
			}
			l := automaton.NewLifeRandom(w, h, a.Str("rule"), a.Num("density"))
			return Value{TGenBool, automaton.NewRowGate(l, row)}, nil
		}})
	add(&Func{Name: "scatter", Result: TGenBool, Doc: "random hits: each step is a hit with the chance given by the generator of numbers (0 to 1): scatter(sweep(0, 0.8, 64)) gets denser. A number is the same chance every step",
		Params: []Param{{"chance", TGenFloat, nil}},
		Call: func(a Args) (Value, error) {
			return Value{TGenBool, stochastic.NewDensity(a.value("chance").V.(dsl.Gen[float64]))}, nil
		}})
	add(&Func{Name: "markov", Result: TGenFloat, Doc: "a chain of chance: the values of the list are the states, and weights is a list for each of them of how likely each state is to come next (markov([60 62 67], [[0 3 1] [2 0 2] [1 1 0]]): after 60 comes 62 three times as often as 67). It begins at start (an item of the list, counted from 0)",
		Params: []Param{{"values", ListOf(TNum), nil}, {"weights", ListOf(ListOf(TNum)), nil}, {"start", TNum, def(0)}},
		Call: func(a Args) (Value, error) {
			vals, err := numbersOf(a.value("values"), "values")
			if err != nil {
				return Value{}, err
			}
			rows := a.value("weights").Items()
			if len(rows) != len(vals) {
				return Value{}, fmt.Errorf("%d values and %d rows of weights: a row for each", len(vals), len(rows))
			}
			w := make([][]float64, len(rows))
			for i, r := range rows {
				if w[i], err = floats(r, "weights"); err != nil {
					return Value{}, err
				}
				if len(w[i]) != len(vals) {
					return Value{}, fmt.Errorf("row %d has %d weights, there are %d values", i+1, len(w[i]), len(vals))
				}
			}
			if s := a.Int("start"); s < 0 || s >= len(vals) {
				return Value{}, fmt.Errorf("start is from 0 to %d", len(vals)-1)
			}
			g, err := markovChain(vals, w, a.Int("start"))
			if err != nil {
				return Value{}, err
			}
			return Value{TGenFloat, g}, nil
		}})
	add(&Func{Name: "learn", Result: TGenFloat, Doc: "new music in the style of an example: the numbers of the list are the example (a melody, a bass line), and what comes out is made by what followed the last order values in it (2 or 3 stays close to it, 1 wanders). It goes on from the beginning of the example",
		Params: []Param{{"example", ListOf(TNum), nil}, {"order", TNum, def(2)}},
		Call: func(a Args) (Value, error) {
			x, err := numbersOf(a.value("example"), "example")
			if err != nil {
				return Value{}, err
			}
			o := a.Int("order")
			if o < 1 || o >= len(x) {
				return Value{}, fmt.Errorf("order is from 1 to %d (one less than the example is long)", len(x)-1)
			}
			return Value{TGenFloat, ngram.Learn(x, o)}, nil
		}})
	add(&Func{Name: "stroll", Result: TGenFloat, Doc: "a random walk through the numbers of a list: it starts at a place at random and each step goes to a neighbour. shape makes the list a grid (stroll(values, shape: [4 4]) walks a 4 by 4 grid, the values row by row), and it goes one step along one of the directions",
		Params: []Param{{"values", ListOf(TNum), nil}, {"shape", ListOf(TNum), listDefault()}},
		Call: func(a Args) (Value, error) {
			x, err := numbersOf(a.value("values"), "values")
			if err != nil {
				return Value{}, err
			}
			dims := []int{len(x)}
			if sh := a.value("shape").Items(); len(sh) > 0 {
				dims = nil
				n := 1
				for _, d := range sh {
					if !isWhole(d.Float()) || d.Float() < 1 || d.Float() > 4096 {
						return Value{}, fmt.Errorf("a size in shape is a whole number from 1")
					}
					dims = append(dims, int(d.Float()))
					n *= int(d.Float())
				}
				if n != len(x) {
					return Value{}, fmt.Errorf("a grid of %d cells needs %d values, there are %d", n, n, len(x))
				}
			}
			return Value{TGenFloat, walk.New[float64](walk.NewMatrix(dims, x))}, nil
		}})
}

func keysOf(m map[string]bool) []string {
	var k []string
	for s := range m {
		k = append(k, s)
	}
	return k
}
