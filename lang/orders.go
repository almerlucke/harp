package lang

import (
	"fmt"
	"strings"

	"harp/dsl"

	"github.com/almerlucke/genny/arpeggio"
	"github.com/almerlucke/genny/bucket"
)

// The orders of psalm's sequence module: the values of a list in the way that they are played

var orders = map[string]func([]float64) dsl.Gen[float64]{
	"in order":    func(v []float64) dsl.Gen[float64] { return dsl.Loop(v...) },
	"shuffled":    func(v []float64) dsl.Gen[float64] { return bucket.NewLoop(v...) },
	"random":      func(v []float64) dsl.Gen[float64] { return bucket.New(v...).SetMode(bucket.Random) },
	"up and down": func(v []float64) dsl.Gen[float64] { return arpeggio.NewLoop(v, arpeggio.Up, arpeggio.Exclusive, false) },
	"converge": func(v []float64) dsl.Gen[float64] {
		return arpeggio.NewLoop(v, arpeggio.Converge, arpeggio.None, false)
	},
	"alternate": func(v []float64) dsl.Gen[float64] {
		return arpeggio.NewLoop(v, arpeggio.Alternate, arpeggio.None, false)
	},
	"backwards": func(v []float64) dsl.Gen[float64] { return arpeggio.NewLoop(v, arpeggio.Up, arpeggio.None, true) },
}

func init() {
	names := presetNames(orders)
	add(&Func{Name: "order", Result: TGenFloat, Doc: "the numbers of a list in an order, round and round: " + strings.Join(names, ", ") + ". in order is 1 2 3 4; shuffled every value once before any again, in a new order each time; random any one every time; up and down up and back down without repeating the ends; converge from the outside in (1 4 2 3); alternate the first with each of the others in turn (1 2 1 3 1 4); backwards the list reversed",
		Params: []Param{{"values", ListOf(TNum), nil}, {"how", TStr, defStr("in order")}},
		Call: func(a Args) (Value, error) {
			x, err := numbersOf(a.value("values"), "values")
			if err != nil {
				return Value{}, err
			}
			make, ok := orders[a.Str("how")]
			if !ok {
				return Value{}, fmt.Errorf("there is no order %q: one of %s%s", a.Str("how"), strings.Join(names, ", "), didYouMean(a.Str("how"), names))
			}
			if len(x) < 2 && a.Str("how") != "in order" && a.Str("how") != "backwards" {
				return Value{}, fmt.Errorf("%s needs two values or more", a.Str("how"))
			}
			return Value{TGenFloat, make(x)}, nil
		}})
	add(&Func{Name: "pick", Result: TGenFloat, Doc: "any one number of a list, a new one every time (the same as order(values, \"random\"))",
		Params: []Param{{"values", ListOf(TNum), nil}},
		Call: func(a Args) (Value, error) {
			x, err := numbersOf(a.value("values"), "values")
			if err != nil {
				return Value{}, err
			}
			return Value{TGenFloat, orders["random"](x)}, nil
		}})
}
