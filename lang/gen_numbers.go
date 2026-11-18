package lang

import (
	"fmt"
	"math"
	"strings"

	"harp/dsl"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/float/dist"
	"github.com/almerlucke/genny/float/iter"
	"github.com/almerlucke/genny/float/iter/updaters/chaos"
	"github.com/almerlucke/genny/float/lfo"
	"github.com/almerlucke/genny/float/lissajous"
	"github.com/almerlucke/genny/float/noise"
	"github.com/almerlucke/genny/float/ramp"
	"github.com/almerlucke/genny/float/rwalk"
)

// Generators of numbers that move: random walks, smooth noise, waves and ramps in steps, chaotic systems, orbits and the distributions of
// chance. They give a number at every step: to degrees (through quantizing), to the fields of play, to the parameters of effects (as seq)

// axisGen takes one value of the frames of a generator of frames
type axisGen struct {
	g    dsl.Gen[[]float64]
	axis int
}

func (a *axisGen) Generate() float64 {
	f := a.g.Generate()
	if a.axis < len(f) {
		return f[a.axis]
	}
	return 0
}
func (a *axisGen) Continuous() bool { return a.g.Continuous() }
func (a *axisGen) Done() bool       { return a.g.Done() }
func (a *axisGen) Reset()           { a.g.Reset() }

// remapGen maps a range of values to another
type remapGen struct {
	g                  dsl.Gen[float64]
	inLo, inHi, lo, hi float64
}

func (r *remapGen) Generate() float64 {
	x := (r.g.Generate() - r.inLo) / (r.inHi - r.inLo)
	return r.lo + x*(r.hi-r.lo)
}
func (r *remapGen) Continuous() bool { return r.g.Continuous() }
func (r *remapGen) Done() bool       { return r.g.Done() }
func (r *remapGen) Reset()           { r.g.Reset() }

func finite(a Args, names ...string) error {
	for _, n := range names {
		if x := a.Num(n); math.IsNaN(x) || math.IsInf(x, 0) {
			return fmt.Errorf("%s is not a number", n)
		}
	}
	return nil
}

var lfoShapes = map[string]lfo.Shape{"sine": lfo.Sine, "triangle": lfo.Triangle, "saw": lfo.Saw, "square": lfo.Square, "hold": lfo.SampleHold, "smooth": lfo.SmoothRandom}
var walkModes = map[string]rwalk.Mode{"clip": rwalk.Clip, "bounce": rwalk.Bounce, "jump": rwalk.Jump}

func names[T any](m map[string]T) string { return strings.Join(presetNames(m), ", ") }

func init() {
	add(&Func{Name: "wander", Result: TGenFloat, Doc: "a random walk: the number takes a step at random from where it is, of at most step (a part of the range, 0 to 1), and stays between lo and hi: at the edge it clips, bounces or jumps back (mode: " + names(walkModes) + ")",
		Params: []Param{{"lo", TNum, nil}, {"hi", TNum, nil}, {"step", TNum, def(0.1)}, {"start", TNum, def(math.NaN())}, {"mode", TStr, defStr("bounce")}},
		Call: func(a Args) (Value, error) {
			if err := finite(a, "lo", "hi", "step"); err != nil {
				return Value{}, err
			}
			mode, ok := walkModes[a.Str("mode")]
			if !ok {
				return Value{}, fmt.Errorf("mode is one of %s%s", names(walkModes), didYouMean(a.Str("mode"), presetNames(walkModes)))
			}
			lo, hi := a.Num("lo"), a.Num("hi")
			start := a.Num("start")
			if math.IsNaN(start) {
				start = (lo + hi) / 2
			}
			return Value{TGenFloat, rwalk.New(start, lo, hi, a.Num("step"), mode)}, nil
		}})
	add(&Func{Name: "drift", Result: TGenFloat, Doc: "smooth noise from -1 to 1: it wanders without jumps (speed is how many wiggles a step, 0.1 is slow; octaves add finer detail)",
		Params: []Param{{"speed", TNum, def(0.1)}, {"octaves", TNum, def(1)}},
		Call: func(a Args) (Value, error) {
			if err := finite(a, "speed"); err != nil {
				return Value{}, err
			}
			o := a.Int("octaves")
			if o < 1 || o > 8 {
				return Value{}, fmt.Errorf("octaves is from 1 to 8")
			}
			return Value{TGenFloat, noise.New(a.Num("speed"), o)}, nil
		}})
	add(&Func{Name: "wave", Result: TGenFloat, Doc: "a slow wave in steps: one cycle takes period steps, between lo and hi (shape: " + names(lfoShapes) + "; hold is a new random value each cycle, smooth a smooth random one)",
		Params: []Param{{"period", TNum, nil}, {"lo", TNum, def(-1)}, {"hi", TNum, def(1)}, {"shape", TStr, defStr("sine")}},
		Call: func(a Args) (Value, error) {
			if err := finite(a, "period", "lo", "hi"); err != nil {
				return Value{}, err
			}
			shape, ok := lfoShapes[a.Str("shape")]
			if !ok {
				return Value{}, fmt.Errorf("shape is one of %s%s", names(lfoShapes), didYouMean(a.Str("shape"), presetNames(lfoShapes)))
			}
			if a.Num("period") <= 0 {
				return Value{}, fmt.Errorf("the period is above 0 steps")
			}
			l := lfo.New(shape, 1, a.Num("period"))
			return Value{TGenFloat, &remapGen{l, -1, 1, a.Num("lo"), a.Num("hi")}}, nil
		}})
	add(&Func{Name: "sweep", Result: TGenFloat, Doc: "a ramp in steps: from from to to in steps steps, then it stays at to (curve 1 is straight, above 1 slow at first, below 1 fast at first)",
		Params: []Param{{"from", TNum, nil}, {"to", TNum, nil}, {"steps", TNum, nil}, {"curve", TNum, def(1)}},
		Call: func(a Args) (Value, error) {
			if err := finite(a, "from", "to", "curve"); err != nil {
				return Value{}, err
			}
			n := a.Int("steps")
			if n < 2 || n > 1<<24 {
				return Value{}, fmt.Errorf("steps is 2 or more")
			}
			return Value{TGenFloat, ramp.NewForever(n, a.Num("from"), a.Num("to"), a.Num("curve"))}, nil
		}})
	add(&Func{Name: "orbit", Result: TGenFloat, Doc: "a Lissajous curve as a number: two sines at the ratio a to b with a phase difference of delta (radians), one round in steps steps; axis 0 is x, 1 is y (use both for two sources that belong together). From -1 to 1",
		Params: []Param{{"a", TNum, def(3)}, {"b", TNum, def(2)}, {"steps", TNum, def(32)}, {"axis", TNum, def(0)}, {"delta", TNum, def(math.Pi / 2)}},
		Call: func(a Args) (Value, error) {
			if err := finite(a, "a", "b", "delta"); err != nil {
				return Value{}, err
			}
			if a.Int("steps") < 1 || (a.Int("axis") != 0 && a.Int("axis") != 1) {
				return Value{}, fmt.Errorf("steps is 1 or more, axis is 0 or 1")
			}
			return Value{TGenFloat, &axisGen{lissajous.New(a.Num("a"), a.Num("b"), a.Num("delta"), a.Int("steps")), a.Int("axis")}}, nil
		}})
	add(&Func{Name: "chaos", Result: TGenFloat, Doc: "a chaotic system as a number (system: logistic, henon, lorenz, rossler, duffing, pendulum, standardmap): it never repeats, and is not random. p and q set the system (logistic: p is the rate, 3.7 is chaotic; henon: p and q, 1.4 and 0.3; standardmap: p the kick, 1; lorenz and rossler: p is the speed of time, 0.01); axis picks which of its values (0 is x, 1 is y, 2 is z). The ranges differ per system: use remap",
		Params: []Param{{"system", TStr, nil}, {"p", TNum, def(math.NaN())}, {"q", TNum, def(math.NaN())}, {"axis", TNum, def(0)}},
		Call: func(a Args) (Value, error) {
			or := func(name string, d float64) float64 {
				if x := a.Num(name); !math.IsNaN(x) {
					return x
				}
				return d
			}
			var it *iter.Iterator
			dims := 2
			switch a.Str("system") {
			case "logistic":
				it, dims = iter.New([]float64{0.5}, chaos.NewVerhulst(or("p", 3.7))), 1
			case "henon":
				it = iter.New([]float64{0.1, 0.1}, chaos.NewHenon(or("p", 1.4), or("q", 0.3)))
			case "lorenz":
				it, dims = iter.New([]float64{1, 1, 1}, chaos.NewLorenz(10, 28, 8.0/3, or("p", 0.01))), 3
			case "rossler":
				it, dims = iter.New([]float64{1, 1, 1}, chaos.NewRossler(0.2, 0.2, 5.7, or("p", 0.05))), 3
			case "duffing":
				it = iter.New([]float64{0.1, 0}, chaos.NewDuffing())
			case "pendulum":
				it = iter.New([]float64{0.2, 0}, chaos.NewPendulum())
			case "standardmap":
				it = iter.New([]float64{0.5, 0.5}, chaos.NewStandardMap(or("p", 1)))
			default:
				return Value{}, fmt.Errorf("system is one of logistic, henon, lorenz, rossler, duffing, pendulum, standardmap%s", didYouMean(a.Str("system"), []string{"logistic", "henon", "lorenz", "rossler", "duffing", "pendulum", "standardmap"}))
			}
			if ax := a.Int("axis"); ax < 0 || ax >= dims {
				return Value{}, fmt.Errorf("%s has %d values: axis is from 0 to %d", a.Str("system"), dims, dims-1)
			}
			return Value{TGenFloat, &axisGen{it, a.Int("axis")}}, nil
		}})
	add(&Func{Name: "remap", Result: TGenFloat, Doc: "a generator of numbers moved from one range to another: remap(chaos(\"lorenz\"), -20, 20, 48, 72)",
		Params: []Param{{"gen", TGenFloat, nil}, {"inLo", TNum, nil}, {"inHi", TNum, nil}, {"lo", TNum, nil}, {"hi", TNum, nil}},
		Call: func(a Args) (Value, error) {
			if err := finite(a, "inLo", "inHi", "lo", "hi"); err != nil {
				return Value{}, err
			}
			if a.Num("inLo") == a.Num("inHi") {
				return Value{}, fmt.Errorf("inLo and inHi are the same")
			}
			return Value{TGenFloat, &remapGen{a.value("gen").V.(dsl.Gen[float64]), a.Num("inLo"), a.Num("inHi"), a.Num("lo"), a.Num("hi")}}, nil
		}})

	// ---- chance
	positive := func(a Args, names ...string) error {
		for _, n := range names {
			if x := a.Num(n); !(x > 0) || math.IsInf(x, 0) {
				return fmt.Errorf("%s is above 0", n)
			}
		}
		return nil
	}
	add(&Func{Name: "gauss", Result: TGenFloat, Doc: "random numbers in a bell curve around mean, most of them within sd of it",
		Params: []Param{{"mean", TNum, nil}, {"sd", TNum, def(1)}},
		Call: func(a Args) (Value, error) {
			if err := positive(a, "sd"); err != nil {
				return Value{}, fmt.Errorf("%v", err)
			}
			return Value{TGenFloat, dist.NewGaussian(a.Num("mean"), a.Num("sd"))}, nil
		}})
	add(&Func{Name: "expo", Result: TGenFloat, Doc: "random numbers from an exponential curve with this mean: many small ones, now and then a big one",
		Params: []Param{{"mean", TNum, nil}},
		Call: func(a Args) (Value, error) {
			if err := positive(a, "mean"); err != nil {
				return Value{}, fmt.Errorf("%v", err)
			}
			return Value{TGenFloat, dist.NewExponential(a.Num("mean"))}, nil
		}})
	add(&Func{Name: "beta", Result: TGenFloat, Doc: "random numbers between lo and hi shaped by a and b: both below 1 gives mostly the edges, both above 1 the middle, a above b gives the high end",
		Params: []Param{{"a", TNum, nil}, {"b", TNum, nil}, {"lo", TNum, def(0)}, {"hi", TNum, def(1)}},
		Call: func(a Args) (Value, error) {
			if err := positive(a, "a", "b"); err != nil {
				return Value{}, fmt.Errorf("%v", err)
			}
			return Value{TGenFloat, dist.NewBeta(a.Num("a"), a.Num("b")).Scale(a.Num("lo"), a.Num("hi"))}, nil
		}})
	add(&Func{Name: "cauchy", Result: TGenFloat, Doc: "random numbers around location, half of them within scale of it, with now and then one very far away",
		Params: []Param{{"location", TNum, nil}, {"scale", TNum, def(1)}},
		Call: func(a Args) (Value, error) {
			if err := positive(a, "scale"); err != nil {
				return Value{}, fmt.Errorf("%v", err)
			}
			return Value{TGenFloat, dist.NewCauchy(a.Num("location"), a.Num("scale"))}, nil
		}})
	add(&Func{Name: "weibull", Result: TGenFloat, Doc: "random numbers from threshold up: shape below 1 gives many near the threshold and a long tail, 1 is exponential, above 1 a hump",
		Params: []Param{{"threshold", TNum, nil}, {"spread", TNum, def(1)}, {"shape", TNum, def(1)}},
		Call: func(a Args) (Value, error) {
			if err := positive(a, "spread", "shape"); err != nil {
				return Value{}, fmt.Errorf("%v", err)
			}
			return Value{TGenFloat, dist.NewWeibull(a.Num("threshold"), a.Num("spread"), a.Num("shape"))}, nil
		}})
	add(&Func{Name: "pink", Result: TGenFloat, Doc: "random numbers from 0 to 1 with slow drifts and fast detail (1/f noise): between white noise and a random walk; rows is the number of octaves of detail",
		Params: []Param{{"rows", TNum, def(6)}},
		Call: func(a Args) (Value, error) {
			if r := a.Int("rows"); r < 1 || r > 32 {
				return Value{}, fmt.Errorf("rows is from 1 to 32")
			}
			return Value{TGenFloat, dist.NewPink(a.Int("rows"))}, nil
		}})
}

var _ genny.Generator[float64] = (*remapGen)(nil)
