package lang

import (
	"fmt"
	"math"

	"harp/blocks"
	"harp/dsl"

	"github.com/almerlucke/genny/clock"
	"github.com/almerlucke/genny/evolve"
	"github.com/almerlucke/genny/float/tendency"
	groovegen "github.com/almerlucke/genny/groove"
	"github.com/almerlucke/genny/listtools"
	"github.com/almerlucke/genny/numseq"
	"github.com/almerlucke/genny/ratchet"
	"github.com/almerlucke/genny/section"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/split"
)

// Number sequences, rhythms of time and of structure, durations that are cut up, sections, and phrases that evolve

// gateOf turns a generator of whole numbers into a rhythm: a hit where the number can be divided by n
type gateOf struct {
	g dsl.Gen[int]
	n int
}

func (d *gateOf) Generate() bool   { return d.g.Generate()%d.n == 0 }
func (d *gateOf) Continuous() bool { return d.g.Continuous() }
func (d *gateOf) Done() bool       { return d.g.Done() }
func (d *gateOf) Reset()           { d.g.Reset() }

func sectionOf[T any](g []dsl.Gen[T], steps []int, restart bool) dsl.Gen[T] {
	parts := make([]section.Part[T], len(g))
	for i := range g {
		parts[i] = section.Part[T]{Gen: g[i], Steps: steps[i], Restart: restart}
	}
	return section.NewLoop(parts...)
}

func init() {
	add(&Func{Name: "accents", Result: ListOf(TNum), Doc: "how strong each step of a bar is, by the groups of a meter, as a list for the accent of play: the first step of the bar is the strongest (downbeat), the first of every other group medium (group), the rest weak (other). accents([3 3 2]) is 8 steps in groups of 3, 3 and 2; accents([4 4 4 4]) a bar of 4 beats of 4 steps",
		Params: []Param{{"groups", ListOf(TNum), nil}, {"downbeat", TNum, def(1)}, {"group", TNum, def(0.8)}, {"other", TNum, def(0.55)}},
		Call: func(a Args) (Value, error) {
			g, err := numbersOf(a.value("groups"), "groups")
			if err != nil {
				return Value{}, err
			}
			groups := make([]int, len(g))
			total := 0
			for i, x := range g {
				if !isWhole(x) || x < 1 || x > 256 {
					return Value{}, fmt.Errorf("a group is a whole number of steps from 1 to 256")
				}
				groups[i] = int(x)
				total += groups[i]
			}
			if total > 4096 {
				return Value{}, fmt.Errorf("a bar of at most 4096 steps")
			}
			acc := groovegen.NewAccent(groups...).Levels(a.Num("downbeat"), a.Num("group"), a.Num("other"))
			out := make([]float64, total)
			for i := range out {
				out[i] = acc.Generate()
			}
			return numList(out), nil
		}})
	add(&Func{Name: "fibonacci", Result: TGenInt, Doc: "the Fibonacci numbers 0 1 1 2 3 5 8 ... each taken modulo mod (so they go round in a pattern): fibonacci(8) is 0 1 1 2 3 5 0 5 5 2 7 1 0 ...",
		Params: []Param{{"mod", TNum, def(12)}},
		Call: func(a Args) (Value, error) {
			if a.Int("mod") < 2 || a.Int("mod") > 4096 {
				return Value{}, fmt.Errorf("mod is from 2 to 4096")
			}
			return Value{TGenInt, numseq.NewFibonacci(a.Int("mod"))}, nil
		}})
	add(&Func{Name: "thueMorse", Result: TGenInt, Doc: "the Thue-Morse sequence 0 1 1 0 1 0 0 1 ...: it never repeats and has no long runs. As degrees it is a fair melody of two notes, with divisible a rhythm",
		Params: nil,
		Call:   func(a Args) (Value, error) { return Value{TGenInt, numseq.NewThueMorse()}, nil }})
	add(&Func{Name: "primes", Result: TGenInt, Doc: "the prime numbers 2 3 5 7 11 13 ...",
		Params: nil,
		Call:   func(a Args) (Value, error) { return Value{TGenInt, numseq.NewPrimes()}, nil }})
	add(&Func{Name: "collatz", Result: TGenInt, Doc: "the hailstone numbers from start: halve it when even, else three times it plus one, until it reaches 1, then again (27 climbs to 9232 before it falls)",
		Params: []Param{{"start", TNum, nil}},
		Call: func(a Args) (Value, error) {
			if a.Int("start") < 1 || a.Int("start") > 1<<20 {
				return Value{}, fmt.Errorf("start is a whole number from 1")
			}
			return Value{TGenInt, numseq.NewCollatzLoop(a.Int("start"))}, nil
		}})
	add(&Func{Name: "divisible", Result: TGenBool, Doc: "a rhythm from a generator of whole numbers: a hit where the number can be divided by n (divisible(fibonacci(12), 2): a hit on the even numbers)",
		Params: []Param{{"numbers", TGenInt, nil}, {"n", TNum, def(2)}},
		Call: func(a Args) (Value, error) {
			if a.Int("n") < 1 {
				return Value{}, fmt.Errorf("n is 1 or more")
			}
			return Value{TGenBool, &gateOf{a.value("numbers").V.(dsl.Gen[int]), a.Int("n")}}, nil
		}})
	add(&Func{Name: "divide", Result: TGenBool, Doc: "a hit on every nth step, from the step offset: divide(3) is x..x..x.., divide(4, 2) is ..x...x.",
		Params: []Param{{"n", TNum, nil}, {"offset", TNum, def(0)}},
		Call: func(a Args) (Value, error) {
			if a.Int("n") < 1 {
				return Value{}, fmt.Errorf("n is 1 or more")
			}
			return Value{TGenBool, clock.NewDivider(a.Int("n"), a.Int("offset"))}, nil
		}})
	add(&Func{Name: "talea", Result: TGenBool, Doc: "a rhythm from the lengths of its notes, in steps: talea([3 2 3]) is x..x.x.. (a note of 3 steps, one of 2, one of 3). With a list of notes of another length (the color) that goes round alongside it, the two come back together only after both have gone round whole times: isorhythm",
		Params: []Param{{"lengths", ListOf(TNum), nil}},
		Call: func(a Args) (Value, error) {
			x, err := numbersOf(a.value("lengths"), "lengths")
			if err != nil {
				return Value{}, err
			}
			var pat []bool
			for _, l := range x {
				if !isWhole(l) || l < 1 || l > 256 {
					return Value{}, fmt.Errorf("a length is a whole number of steps from 1 to 256")
				}
				pat = append(pat, true)
				for i := 1; i < int(l); i++ {
					pat = append(pat, false)
				}
			}
			return Value{TGenBool, dsl.Loop(pat...)}, nil
		}})
	add(&Func{Name: "ratchets", Result: TGenBool, Doc: "a rhythm of rolls: every beat of size steps has as many hits, equally spaced, as the next number of counts says (ratchets([1 2 1 4], 4): one hit, two, one, then four, in beats of 4 steps; a count above size is cut to size)",
		Params: []Param{{"counts", ListOf(TNum), nil}, {"size", TNum, def(4)}},
		Call: func(a Args) (Value, error) {
			c, err := numbersOf(a.value("counts"), "counts")
			if err != nil {
				return Value{}, err
			}
			size := a.Int("size")
			if size < 1 || size > 64 {
				return Value{}, fmt.Errorf("size is from 1 to 64 steps")
			}
			var pat []bool
			for _, n := range c {
				if !isWhole(n) || n < 0 || n > 64 {
					return Value{}, fmt.Errorf("a count is a whole number from 0 to 64")
				}
				beat := make([]bool, size)
				if n > 0 {
					for _, off := range ratchet.Offsets(int(n)) {
						beat[min(int(off*float64(size)), size-1)] = true
					}
				}
				pat = append(pat, beat...)
			}
			return Value{TGenBool, dsl.Loop(pat...)}, nil
		}})
	add(&Func{Name: "jump", Result: TGenFloat, Doc: "a way through a list: chunk values in a row, each step further along, then the start jumps leap places and another chunk begins; jump([0 1 2 3 4 5 6 7], 3, 1, 4) is 0 1 2, 4 5 6, 0 1 2 ... (the places go round the list)",
		Params: []Param{{"values", ListOf(TNum), nil}, {"chunk", TNum, def(3)}, {"step", TNum, def(1)}, {"leap", TNum, def(1)}},
		Call: func(a Args) (Value, error) {
			x, err := numbersOf(a.value("values"), "values")
			if err != nil {
				return Value{}, err
			}
			if a.Int("chunk") < 1 {
				return Value{}, fmt.Errorf("chunk is 1 or more")
			}
			return Value{TGenFloat, listtools.NewJump(x, a.Int("chunk"), a.Int("step"), a.Int("leap"))}, nil
		}})
	add(&Func{Name: "split", Result: ListOf(TNum), Doc: "a length cut into related parts, as a list (for the beats of play): the value is divided into branches equal parts, and each part may be divided again with the chance prob, until a part would be shorter than lower. split(4, 0.5, 0.5, 2) is a half, then two quarters, one of which is two eighths, ... It is made when the patch is built, by chance: seed it",
		Params: []Param{{"value", TNum, nil}, {"lower", TNum, def(0.25)}, {"prob", TNum, def(0.5)}, {"branches", TNum, def(2)}},
		Call: func(a Args) (Value, error) {
			v, lo := a.Num("value"), a.Num("lower")
			if !(v > 0) || !(lo > 0) || a.Int("branches") < 2 || a.Int("branches") > 16 || v/lo > 1<<14 {
				return Value{}, fmt.Errorf("value and lower are above 0 (and lower is not tiny against value), branches is from 2 to 16")
			}
			return numList(split.Split(v, lo, a.Num("prob"), a.Int("branches"))), nil
		}})
	add(&Func{Name: "metric", Result: ListOf(TNum), Doc: "a length in n equal parts, as a list: metric(4, 3) is three parts of 1.333",
		Params: []Param{{"value", TNum, nil}, {"n", TNum, nil}},
		Call: func(a Args) (Value, error) {
			if !(a.Num("value") > 0) || a.Int("n") < 1 || a.Int("n") > 4096 {
				return Value{}, fmt.Errorf("value is above 0, n is from 1 to 4096")
			}
			return numList(split.Metric(a.Num("value"), a.Int("n"))), nil
		}})
	add(&Func{Name: "sections", Result: TAny, Doc: "generators one after the other, each for a number of steps (a step is a value taken from it), then round again: sections([a, b], [64, 32]) is a for 64, b for 32. With restart: true each starts from the beginning when its turn comes, else it goes on where it was",
		Params: []Param{{"gens", anyList, nil}, {"steps", ListOf(TNum), nil}, {"restart", TBool, defBool(false)}},
		Call: func(a Args) (Value, error) {
			st, err := numbersOf(a.value("steps"), "steps")
			if err != nil {
				return Value{}, err
			}
			if len(st) != len(a.value("gens").Items()) {
				return Value{}, fmt.Errorf("%d generators and %d lengths: a length for each", len(a.value("gens").Items()), len(st))
			}
			n := make([]int, len(st))
			for i, s := range st {
				if !isWhole(s) || s < 1 {
					return Value{}, fmt.Errorf("a length is a whole number of steps from 1")
				}
				n[i] = int(s)
			}
			r := a.value("restart").V.(bool)
			return genLists{
				B: func(g []dsl.Gen[bool]) dsl.Gen[bool] { return sectionOf(g, n, r) },
				I: func(g []dsl.Gen[int]) dsl.Gen[int] { return sectionOf(g, n, r) },
				F: func(g []dsl.Gen[float64]) dsl.Gen[float64] { return sectionOf(g, n, r) },
				C: func(g []dsl.Gen[[]float64]) dsl.Gen[[]float64] { return sectionOf(g, n, r) },
				N: func(g []dsl.Gen[[]blocks.Note]) dsl.Gen[[]blocks.Note] { return sectionOf(g, n, r) },
			}.of(a.value("gens"), "sections")
		}})
	add(&Func{Name: "evolve", Result: TGenInt, Doc: "a phrase of scale degrees that gets better as it plays: a population of random phrases of length notes between lo and hi is bred, and the best is played; after every round of the phrase it evolves generations more. What is better is what you weigh: smooth (small steps), variety (many different intervals), arch (a rise to a peak and a fall)",
		Params: []Param{{"length", TNum, def(8)}, {"lo", TNum, def(-7)}, {"hi", TNum, def(7)}, {"smooth", TNum, def(1)}, {"variety", TNum, def(0)}, {"arch", TNum, def(0)}, {"generations", TNum, def(1)}, {"population", TNum, def(40)}},
		Call: func(a Args) (Value, error) {
			n, lo, hi := a.Int("length"), a.Int("lo"), a.Int("hi")
			if n < 2 || n > 256 || lo >= hi || a.Int("population") < 2 || a.Int("population") > 1000 {
				return Value{}, fmt.Errorf("length is from 2 to 256, lo is below hi, population from 2 to 1000")
			}
			var parts []evolve.Part
			for _, p := range []struct {
				w float64
				f evolve.Fitness
			}{{a.Num("smooth"), evolve.Smoothness()}, {a.Num("variety"), evolve.IntervalVariety()}, {a.Num("arch"), evolve.Contour(evolve.Arch(n))}} {
				if p.w < 0 || math.IsNaN(p.w) {
					return Value{}, fmt.Errorf("a weight is 0 or more")
				}
				if p.w > 0 {
					parts = append(parts, evolve.Weight(p.f, p.w))
				}
			}
			if len(parts) == 0 {
				return Value{}, fmt.Errorf("give smooth, variety or arch a weight above 0: what is a good phrase?")
			}
			ga := evolve.New(evolve.Config{Population: a.Int("population"), Length: n, Min: lo, Max: hi, Fitness: evolve.Weighted(parts...), MutationRate: 0.08, Elitism: 1})
			return Value{TGenInt, evolve.NewPlayer(ga, a.Int("generations"))}, nil
		}})
	add(&Func{Name: "tendency", Result: TGenFloat, Doc: "random numbers inside a band that moves, a tendency mask of Koenig: segments is a list of [steps, lo1, hi1, lo2, hi2] (over so many steps the lower edge goes from lo1 to lo2 and the upper from hi1 to hi2): tendency([[16, 60, 62, 48, 72]]) opens from nearly one note to two octaves. loop: true goes round, else the last band stays",
		Params: []Param{{"segments", ListOf(ListOf(TNum)), nil}, {"loop", TBool, defBool(true)}},
		Call: func(a Args) (Value, error) {
			var segs []tendency.Segment
			for i, s := range a.value("segments").Items() {
				x, err := floats(s, "segments")
				if err != nil {
					return Value{}, err
				}
				if len(x) != 5 || !isWhole(x[0]) || x[0] < 0 {
					return Value{}, fmt.Errorf("segment %d is [steps, lo1, hi1, lo2, hi2]", i+1)
				}
				segs = append(segs, tendency.Segment{N: int(x[0]), A1: x[1], A2: x[2], Z1: x[3], Z2: x[4]})
			}
			if len(segs) == 0 {
				return Value{}, fmt.Errorf("segments has no segments")
			}
			return Value{TGenFloat, tendency.NewValue(tendency.NewMask(a.value("loop").V.(bool), segs...))}, nil
		}})
}

var _ = sequence.New[int]
