package lang

import (
	"fmt"
	"math"

	"harp/blocks"
	"harp/dsl"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/and"
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/choose"
	"github.com/almerlucke/genny/combine"
	"github.com/almerlucke/genny/constrain"
	"github.com/almerlucke/genny/flatten"
	"github.com/almerlucke/genny/or"
	"github.com/almerlucke/genny/pulse"
	"github.com/almerlucke/genny/sieve"
)

// Generators from genny that combine other generators, of any kind (of rhythms, of numbers, of chords, of notes), and the rhythms of genny.
// A function that takes generators of any kind gives a generator of the kind it was given.

// the five kinds of generator, and what a function does with each: a function of a generator of that type
type genKinds struct {
	B func(dsl.Gen[bool]) dsl.Gen[bool]
	I func(dsl.Gen[int]) dsl.Gen[int]
	F func(dsl.Gen[float64]) dsl.Gen[float64]
	C func(dsl.Gen[[]float64]) dsl.Gen[[]float64]
	N func(dsl.Gen[[]blocks.Note]) dsl.Gen[[]blocks.Note]
}

func kindsOf[T any](f func(dsl.Gen[T]) dsl.Gen[T]) func(dsl.Gen[T]) dsl.Gen[T] { return f }

// each applies f to a generator, whatever kind it is
func (k genKinds) each(v Value, fn string) (Value, error) {
	switch g := v.V.(type) {
	case dsl.Gen[bool]:
		if k.B != nil {
			return Value{TGenBool, k.B(g)}, nil
		}
	case dsl.Gen[int]:
		if k.I != nil {
			return Value{TGenInt, k.I(g)}, nil
		}
	case dsl.Gen[float64]:
		if k.F != nil {
			return Value{TGenFloat, k.F(g)}, nil
		}
	case dsl.Gen[[]float64]:
		if k.C != nil {
			return Value{TGenChord, k.C(g)}, nil
		}
	case dsl.Gen[[]blocks.Note]:
		if k.N != nil {
			return Value{TGenNotes, k.N(g)}, nil
		}
	default:
		return Value{}, fmt.Errorf("a generator is wanted (a rhythm, degrees, melody, ...), this is a %s", v.T)
	}
	return Value{}, fmt.Errorf("%s does not take a %s", fn, v.T)
}

// gensOf takes the items of a list as generators of one kind
func gensOf[T any](items []Value, fn string) ([]dsl.Gen[T], error) {
	out := make([]dsl.Gen[T], len(items))
	for i, it := range items {
		g, ok := it.V.(dsl.Gen[T])
		if !ok {
			return nil, fmt.Errorf("item %d is a %s, not a generator of the kind of the first", i+1, it.T)
		}
		out[i] = g
	}
	return out, nil
}

// many gives a function of a list of generators (of one kind) to a generator of that kind
type genLists struct {
	B func([]dsl.Gen[bool]) dsl.Gen[bool]
	I func([]dsl.Gen[int]) dsl.Gen[int]
	F func([]dsl.Gen[float64]) dsl.Gen[float64]
	C func([]dsl.Gen[[]float64]) dsl.Gen[[]float64]
	N func([]dsl.Gen[[]blocks.Note]) dsl.Gen[[]blocks.Note]
}

func (k genLists) of(list Value, fn string) (Value, error) {
	items := list.Items()
	if len(items) == 0 {
		return Value{}, fmt.Errorf("the list has no generators")
	}
	switch items[0].V.(type) {
	case dsl.Gen[bool]:
		g, err := gensOf[bool](items, fn)
		return Value{TGenBool, k.B(g)}, err
	case dsl.Gen[int]:
		g, err := gensOf[int](items, fn)
		return Value{TGenInt, k.I(g)}, err
	case dsl.Gen[float64]:
		g, err := gensOf[float64](items, fn)
		return Value{TGenFloat, k.F(g)}, err
	case dsl.Gen[[]float64]:
		g, err := gensOf[[]float64](items, fn)
		return Value{TGenChord, k.C(g)}, err
	case dsl.Gen[[]blocks.Note]:
		g, err := gensOf[[]blocks.Note](items, fn)
		return Value{TGenNotes, k.N(g)}, err
	}
	return Value{}, fmt.Errorf("a list of generators is wanted, item 1 is a %s", items[0].T)
}

var anyList = Type{Kind: KList}

// ---- generators of my own, for what genny has no generator for

// maybeGen lets a rhythm play each of its hits with a chance p
type maybeGen struct {
	g dsl.Gen[bool]
	p float64
}

func (m *maybeGen) Generate() bool   { return m.g.Generate() && genny.Float64() < m.p }
func (m *maybeGen) Continuous() bool { return m.g.Continuous() }
func (m *maybeGen) Done() bool       { return m.g.Done() }
func (m *maybeGen) Reset()           { m.g.Reset() }

type invertGen struct{ g dsl.Gen[bool] }

func (m *invertGen) Generate() bool   { return !m.g.Generate() }
func (m *invertGen) Continuous() bool { return m.g.Continuous() }
func (m *invertGen) Done() bool       { return m.g.Done() }
func (m *invertGen) Reset()           { m.g.Reset() }

// stutterGen gives each value of a generator n times (or lo to hi times, at random)
type stutterGen[T any] struct {
	g      dsl.Gen[T]
	lo, hi int
	left   int
	last   T
}

func (s *stutterGen[T]) Generate() T {
	if s.left == 0 {
		s.last = s.g.Generate()
		s.left = s.lo
		if s.hi > s.lo {
			s.left += genny.Intn(s.hi - s.lo + 1)
		}
	}
	s.left--
	return s.last
}
func (s *stutterGen[T]) Continuous() bool { return s.g.Continuous() }
func (s *stutterGen[T]) Done() bool       { return s.left == 0 && s.g.Done() }
func (s *stutterGen[T]) Reset()           { s.left = 0; s.g.Reset() }

func stutterOf[T any](lo, hi int) func(dsl.Gen[T]) dsl.Gen[T] {
	return func(g dsl.Gen[T]) dsl.Gen[T] { return &stutterGen[T]{g: g, lo: lo, hi: hi} }
}

// allGen plays all of the rhythms at once: a hit where both have one (and), or where one has (or)
type allGen struct {
	gs  []dsl.Gen[bool]
	and bool
}

func (m *allGen) Generate() bool {
	out := m.and
	for _, g := range m.gs { // all are advanced, so that each keeps its own place
		v := g.Generate()
		if m.and {
			out = out && v
		} else {
			out = out || v
		}
	}
	return out
}
func (m *allGen) Continuous() bool { return true }
func (m *allGen) Done() bool       { return false }
func (m *allGen) Reset() {
	for _, g := range m.gs {
		g.Reset()
	}
}

func rhythms(v Value, fn string) ([]dsl.Gen[bool], error) {
	return gensOf[bool](v.Items(), fn)
}

func init() {
	add(&Func{Name: "alternate", Result: TAny, Doc: "generators in turn: a value from the first, then one from the second, and so on, round again (a list of generators of one kind: alternate([melody(...), melody(...)]))",
		Params: []Param{{"gens", anyList, nil}},
		Call: func(a Args) (Value, error) {
			return genLists{
				B: func(g []dsl.Gen[bool]) dsl.Gen[bool] { return and.NewLoop(g...) },
				I: func(g []dsl.Gen[int]) dsl.Gen[int] { return and.NewLoop(g...) },
				F: func(g []dsl.Gen[float64]) dsl.Gen[float64] { return and.NewLoop(g...) },
				C: func(g []dsl.Gen[[]float64]) dsl.Gen[[]float64] { return and.NewLoop(g...) },
				N: func(g []dsl.Gen[[]blocks.Note]) dsl.Gen[[]blocks.Note] { return and.NewLoop(g...) },
			}.of(a.value("gens"), "alternate")
		}})
	add(&Func{Name: "anyOf", Result: TAny, Doc: "generators at random: every value is from one of the generators of the list, picked at random (a list of generators of one kind)",
		Params: []Param{{"gens", anyList, nil}},
		Call: func(a Args) (Value, error) {
			return genLists{
				B: func(g []dsl.Gen[bool]) dsl.Gen[bool] { return or.NewLoop(or.Random, g...) },
				I: func(g []dsl.Gen[int]) dsl.Gen[int] { return or.NewLoop(or.Random, g...) },
				F: func(g []dsl.Gen[float64]) dsl.Gen[float64] { return or.NewLoop(or.Random, g...) },
				C: func(g []dsl.Gen[[]float64]) dsl.Gen[[]float64] { return or.NewLoop(or.Random, g...) },
				N: func(g []dsl.Gen[[]blocks.Note]) dsl.Gen[[]blocks.Note] { return or.NewLoop(or.Random, g...) },
			}.of(a.value("gens"), "anyOf")
		}})
	add(&Func{Name: "chance", Result: TAny, Doc: "generators at random with weights: every value is from one of the generators, picked with a chance in proportion to its weight (chance([a, b], [3, 1]): a three times as often as b)",
		Params: []Param{{"gens", anyList, nil}, {"weights", ListOf(TNum), nil}},
		Call: func(a Args) (Value, error) {
			w, err := floats(a.value("weights"), "weights")
			if err != nil {
				return Value{}, err
			}
			if len(w) != len(a.value("gens").Items()) {
				return Value{}, fmt.Errorf("%d generators and %d weights", len(a.value("gens").Items()), len(w))
			}
			for _, x := range w {
				if x < 0 || math.IsNaN(x) {
					return Value{}, fmt.Errorf("a weight is not 0 or more")
				}
			}
			return genLists{
				B: func(g []dsl.Gen[bool]) dsl.Gen[bool] { return weighted(g, w) },
				I: func(g []dsl.Gen[int]) dsl.Gen[int] { return weighted(g, w) },
				F: func(g []dsl.Gen[float64]) dsl.Gen[float64] { return weighted(g, w) },
				C: func(g []dsl.Gen[[]float64]) dsl.Gen[[]float64] { return weighted(g, w) },
				N: func(g []dsl.Gen[[]blocks.Note]) dsl.Gen[[]blocks.Note] { return weighted(g, w) },
			}.of(a.value("gens"), "chance")
		}})
	add(&Func{Name: "stutter", Result: TAny, Doc: "every value of a generator n times (or between n and max times, at random): stutter(degrees(...), 2)",
		Params: []Param{{"gen", TAny, nil}, {"n", TNum, def(2)}, {"max", TNum, def(0)}},
		Call: func(a Args) (Value, error) {
			lo, hi := a.Int("n"), a.Int("max")
			if lo < 1 || lo > 64 || hi > 64 {
				return Value{}, fmt.Errorf("n is from 1 to 64")
			}
			return genKinds{B: stutterOf[bool](lo, hi), I: stutterOf[int](lo, hi), F: stutterOf[float64](lo, hi), C: stutterOf[[]float64](lo, hi), N: stutterOf[[]blocks.Note](lo, hi)}.each(a.value("gen"), "stutter")
		}})
	add(&Func{Name: "shotgun", Result: TAny, Doc: "a generator with values dropped at random: each value is dropped with the chance p and the next one taken, so that it gets thinner in what it plays, not in length",
		Params: []Param{{"gen", TAny, nil}, {"p", TNum, def(0.3)}},
		Call: func(a Args) (Value, error) {
			p := a.Num("p")
			return genKinds{
				I: func(g dsl.Gen[int]) dsl.Gen[int] { return constrain.Shotgun(g, p) },
				F: func(g dsl.Gen[float64]) dsl.Gen[float64] { return constrain.Shotgun(g, p) },
				C: func(g dsl.Gen[[]float64]) dsl.Gen[[]float64] { return constrain.Shotgun(g, p) },
			}.each(a.value("gen"), "shotgun")
		}})
	add(&Func{Name: "noRepeat", Result: TAny, Doc: "a generator of numbers that never plays the same value twice in a row",
		Params: []Param{{"gen", TAny, nil}},
		Call: func(a Args) (Value, error) {
			return genKinds{
				I: func(g dsl.Gen[int]) dsl.Gen[int] { return constrain.RemoveRepetitions(g) },
				F: func(g dsl.Gen[float64]) dsl.Gen[float64] { return constrain.RemoveRepetitions(g) },
			}.each(a.value("gen"), "noRepeat")
		}})
	add(&Func{Name: "distinct", Result: TAny, Doc: "a generator of numbers in groups of n values that are all different (a value that is in the group already is drawn again)",
		Params: []Param{{"gen", TAny, nil}, {"n", TNum, nil}},
		Call: func(a Args) (Value, error) {
			n := a.Int("n")
			if n < 1 {
				return Value{}, fmt.Errorf("n is 1 or more")
			}
			return genKinds{
				I: func(g dsl.Gen[int]) dsl.Gen[int] { return constrain.Duplicates(g, n) },
				F: func(g dsl.Gen[float64]) dsl.Gen[float64] { return constrain.Duplicates(g, n) },
			}.each(a.value("gen"), "distinct")
		}})
	add(&Func{Name: "flatten", Result: TGenFloat, Doc: "the notes of a generator of chords, one after the other (an arpeggio of the chords)",
		Params: []Param{{"chords", TGenChord, nil}},
		Call: func(a Args) (Value, error) {
			return Value{TGenFloat, flatten.New(a.value("chords").V.(dsl.Gen[[]float64]))}, nil
		}})
	add(&Func{Name: "stack", Result: TGenChord, Doc: "a chord of a value from each of the generators of numbers of the list: stack([degrees(...), degrees(...)]) is two voices that move on their own",
		Params: []Param{{"voices", ListOf(TGenFloat), nil}},
		Call: func(a Args) (Value, error) {
			g, err := gensOf[float64](a.value("voices").Items(), "stack")
			if err != nil {
				return Value{}, err
			}
			if len(g) == 0 {
				return Value{}, fmt.Errorf("no voices")
			}
			return Value{TGenChord, combine.New(g...)}, nil
		}})
	add(&Func{Name: "deal", Result: TGenFloat, Doc: "the numbers of a list in a random order, every one once, then again in another order (a shuffled deck)",
		Params: []Param{{"list", ListOf(TNum), nil}},
		Call: func(a Args) (Value, error) {
			x, err := floats(a.value("list"), "list")
			if err != nil {
				return Value{}, err
			}
			if len(x) == 0 {
				return Value{}, fmt.Errorf("the list has no numbers")
			}
			return Value{TGenFloat, bucket.NewLoop(x...).SetMode(bucket.Indexed)}, nil
		}})
	add(&Func{Name: "uniform", Result: TGenFloat, Doc: "random numbers from lo up to hi, at every step",
		Params: []Param{{"lo", TNum, nil}, {"hi", TNum, nil}},
		Call: func(a Args) (Value, error) {
			lo, hi := a.Num("lo"), a.Num("hi")
			return Value{TGenFloat, &uniformGen{lo, hi}}, nil
		}})

	// ---- rhythms
	add(&Func{Name: "maybe", Result: TGenBool, Doc: "a rhythm with each hit kept with the chance p (0 to 1): maybe(euclid(16, 5), 0.8)",
		Params: []Param{{"rhythm", TGenBool, nil}, {"p", TNum, nil}},
		Call: func(a Args) (Value, error) {
			return Value{TGenBool, &maybeGen{a.value("rhythm").V.(dsl.Gen[bool]), a.Num("p")}}, nil
		}})
	add(&Func{Name: "invert", Result: TGenBool, Doc: "a rhythm turned inside out: a hit where there was a rest and a rest where there was a hit",
		Params: []Param{{"rhythm", TGenBool, nil}},
		Call: func(a Args) (Value, error) {
			return Value{TGenBool, &invertGen{a.value("rhythm").V.(dsl.Gen[bool])}}, nil
		}})
	add(&Func{Name: "both", Result: TGenBool, Doc: "the hits that the rhythms of the list have together (and): both([euclid(16, 9), every(2)])",
		Params: []Param{{"rhythms", ListOf(TGenBool), nil}},
		Call: func(a Args) (Value, error) {
			g, err := rhythms(a.value("rhythms"), "both")
			if err != nil {
				return Value{}, err
			}
			if len(g) == 0 {
				return Value{}, fmt.Errorf("the list has no rhythms")
			}
			return Value{TGenBool, &allGen{g, true}}, nil
		}})
	add(&Func{Name: "either", Result: TGenBool, Doc: "the hits of all the rhythms of the list (or): either([euclid(16, 5), euclid(16, 3, 4)])",
		Params: []Param{{"rhythms", ListOf(TGenBool), nil}},
		Call: func(a Args) (Value, error) {
			g, err := rhythms(a.value("rhythms"), "either")
			if err != nil {
				return Value{}, err
			}
			if len(g) == 0 {
				return Value{}, fmt.Errorf("the list has no rhythms")
			}
			return Value{TGenBool, &allGen{g, false}}, nil
		}})
	add(&Func{Name: "meter", Result: TGenBool, Doc: "the hits of a meter, thinned by a level: the pulses of the divisions (meter([4, 4], level) is 16 pulses in 4 groups of 4) have an importance, the downbeat the most; the pulses less important than the level (a number, or a generator of numbers: a rising level thins the rhythm out) are rests. Level 0 is every pulse",
		Params: []Param{{"divisions", ListOf(TNum), nil}, {"level", TGenFloat, nil}},
		Call: func(a Args) (Value, error) {
			d, err := floats(a.value("divisions"), "divisions")
			if err != nil || len(d) == 0 {
				return Value{}, fmt.Errorf("the divisions are a list of whole numbers from 1: [4 4]")
			}
			div := make([]int, len(d))
			total := 1
			for i, x := range d {
				if !isWhole(x) || x < 1 || x > 64 {
					return Value{}, fmt.Errorf("a division is a whole number from 1 to 64")
				}
				if div[i] = int(x); total*div[i] > 4096 {
					return Value{}, fmt.Errorf("more than 4096 pulses")
				}
				total *= div[i]
			}
			return Value{TGenBool, pulse.NewGate(pulse.Hierarchy(div...), a.value("level").V.(dsl.Gen[float64]))}, nil
		}})
	add(&Func{Name: "sieve", Result: TGenBool, Doc: "a rhythm from a sieve of Xenakis: a hit on every step that is a member of a residue class, the classes of mods and starts joined (sieve(16, [3, 5], [0, 2]): the steps 0 3 6 .. and 2 7 12 ..); length is how many steps go round",
		Params: []Param{{"length", TNum, nil}, {"mods", ListOf(TNum), nil}, {"starts", ListOf(TNum), nil}},
		Call: func(a Args) (Value, error) {
			mods, err := floats(a.value("mods"), "mods")
			if err != nil {
				return Value{}, err
			}
			starts, err := floats(a.value("starts"), "starts")
			if err != nil {
				return Value{}, err
			}
			n := a.Int("length")
			if len(mods) == 0 || len(mods) != len(starts) || n < 1 {
				return Value{}, fmt.Errorf("a length of 1 or more, and as many starts as mods (at least one)")
			}
			var s sieve.Sieve
			for i := range mods {
				if mods[i] < 1 || !isWhole(mods[i]) || !isWhole(starts[i]) {
					return Value{}, fmt.Errorf("mods are whole numbers from 1, starts whole numbers")
				}
				c := sieve.New(int(mods[i]), int(starts[i]), 0, n-1)
				if i == 0 {
					s = c
				} else {
					s = s.Union(c)
				}
			}
			return Value{TGenBool, sieve.NewGate(s, n)}, nil
		}})
}

func weighted[T any](g []dsl.Gen[T], w []float64) dsl.Gen[T] {
	opts := make([]choose.Option[T], len(g))
	for i := range g {
		opts[i] = choose.Option[T]{Gen: g[i], Weight: w[i]}
	}
	return choose.New(opts...)
}

type uniformGen struct{ lo, hi float64 }

func (u *uniformGen) Generate() float64 { return u.lo + genny.Float64()*(u.hi-u.lo) }
func (u *uniformGen) Continuous() bool  { return true }
func (u *uniformGen) Done() bool        { return false }
func (u *uniformGen) Reset()            {}
