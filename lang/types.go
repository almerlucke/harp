package lang

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"harp/blocks"
	"harp/dsl"
)

// Kind is what a value is. The two rates of harp are kinds: a signal is at audio rate, a generator
// makes a value for every step of the clock (event rate), and the checker keeps them apart.
type Kind int

const (
	KNum      Kind = iota // a number (a note is one too)
	KBool                 // true or false
	KStr                  // a string
	KSig                  // a signal of one or more channels: audio rate
	KList                 // a list of values of one kind
	KGenInt               // a generator of whole numbers (scale degrees): event rate
	KGenFloat             // a generator of numbers (pitches): event rate
	KGenBool              // a generator of booleans (a rhythm): event rate
	KGenChord             // a generator of chords (lists of pitches): event rate
	KGenNotes             // a generator of notes to play on a synth: event rate
	KEnv                  // an envelope, waiting for a gate
	KVoice                // the pitch and the gate of a melody: two signals
	KScale                // the steps of a scale
	KFunc                 // a function (of the program), to give to repeat or compose
	KAny                  // only for the parameters of a function of the program: any value
	KScore                // a score: notes with a start and a length, in steps (a MIDI file, a fugue, a canon)
)

// Type is the kind of a value, and for a list the kind of its items (nil if it is empty)
type Type struct {
	Kind Kind
	Elem *Type
}

var (
	TNum      = Type{Kind: KNum}
	TBool     = Type{Kind: KBool}
	TStr      = Type{Kind: KStr}
	TSig      = Type{Kind: KSig}
	TGenInt   = Type{Kind: KGenInt}
	TGenFloat = Type{Kind: KGenFloat}
	TGenBool  = Type{Kind: KGenBool}
	TGenChord = Type{Kind: KGenChord}
	TGenNotes = Type{Kind: KGenNotes}
	TEnv      = Type{Kind: KEnv}
	TVoice    = Type{Kind: KVoice}
	TScale    = Type{Kind: KScale}
	TFunc     = Type{Kind: KFunc}
	TAny      = Type{Kind: KAny}
	TScore    = Type{Kind: KScore}
)

// ListOf is the type of a list of items of a type
func ListOf(t Type) Type { return Type{Kind: KList, Elem: &t} }

func (t Type) String() string {
	switch t.Kind {
	case KNum:
		return "number"
	case KBool:
		return "boolean"
	case KStr:
		return "string"
	case KSig:
		return "signal"
	case KList:
		if t.Elem == nil {
			return "empty list"
		}
		return "list of " + t.Elem.String()
	case KGenInt:
		return "generator of whole numbers"
	case KGenFloat:
		return "generator of numbers"
	case KGenBool:
		return "generator of booleans"
	case KGenChord:
		return "generator of chords"
	case KGenNotes:
		return "generator of notes"
	case KEnv:
		return "envelope"
	case KVoice:
		return "voice"
	case KScale:
		return "scale"
	case KFunc:
		return "function"
	case KAny:
		return "anything"
	case KScore:
		return "score"
	}
	return "?"
}

// isGen says if the type is at event rate
func (t Type) isGen() bool { return t.Kind >= KGenInt && t.Kind <= KGenNotes }

// Value is something a name can have: its type and the Go value that is in the graph of harp
type Value struct {
	T Type
	V any
}

func numV(f float64) Value     { return Value{TNum, f} }
func sigV(s dsl.Sig) Value     { return Value{TSig, s} }
func strV(s string) Value      { return Value{TStr, s} }
func (v Value) Float() float64 { return v.V.(float64) }
func (v Value) Sig() dsl.Sig   { return v.V.(dsl.Sig) }
func (v Value) Items() []Value { return v.V.([]Value) }

func listV(items []Value) Value {
	t := Type{Kind: KList}
	if len(items) > 0 {
		e := items[0].T
		t.Elem = &e
	}
	return Value{t, items}
}

func isWhole(f float64) bool { return f == math.Trunc(f) && math.Abs(f) < 1<<31 }

// coerce turns a value into the type that is wanted, where the language says a value of one kind
// may stand for another:
//
//	number         -> signal (a constant), generator (the same value every step)
//	generator of numbers -> signal (a value for every step, held until the next: a generator that is used as a signal in
//	                two places is the same signal in both: one stream, moved on once a step)
//	list           -> signal (the items are the channels), generator (the items, again and again)
//
// It says no with the reason when there is none
func coerce(v Value, want Type) (Value, error) {
	r, err := coerceOnly(v, want)
	if err == nil && r.T.isGen() && r.T.Kind != v.T.Kind {
		// a list or a number made into a generator: the same list is the same generator
		r = fingerprinted(r, "as", []Value{v})
	}
	return r, err
}

func coerceOnly(v Value, want Type) (Value, error) {
	if want.Kind == KAny {
		return v, nil
	}
	if v.T.Kind == want.Kind && want.Kind != KList { // the same kind (a list is looked at item by item, below)
		return v, nil
	}
	switch want.Kind {
	case KSig:
		switch v.T.Kind {
		case KNum:
			return sigV(dsl.N(v.Float())), nil
		case KGenFloat:
			return sigV(heldSignal(v.V.(dsl.Gen[float64]), v.V)), nil
		case KGenInt:
			return sigV(heldSignal(intAsFloat{v.V.(dsl.Gen[int])}, v.V)), nil
		case KList:
			items := v.Items()
			if len(items) == 0 {
				return Value{}, fmt.Errorf("an empty list has no channels")
			}
			sigs := make([]dsl.Sig, len(items))
			for i, it := range items {
				c, err := coerce(it, TSig)
				if err != nil {
					return Value{}, fmt.Errorf("item %d of the list: %w", i+1, err)
				}
				sigs[i] = c.Sig()
			}
			return sigV(dsl.Chans(sigs...)), nil
		}
	case KGenInt:
		switch v.T.Kind {
		case KNum:
			return coerce(listV([]Value{v}), want)
		case KList:
			ints := make([]int, len(v.Items()))
			for i, it := range v.Items() {
				if it.T.Kind != KNum || !isWhole(it.Float()) {
					return Value{}, fmt.Errorf("item %d of the list is not a whole number", i+1)
				}
				ints[i] = int(it.Float())
			}
			if len(ints) == 0 {
				return Value{}, fmt.Errorf("an empty list has nothing to play")
			}
			return Value{TGenInt, dsl.Loop(ints...)}, nil
		}
	case KGenFloat:
		switch v.T.Kind {
		case KNum:
			return coerce(listV([]Value{v}), want)
		case KList:
			fs := make([]float64, len(v.Items()))
			for i, it := range v.Items() {
				if it.T.Kind != KNum {
					return Value{}, fmt.Errorf("item %d of the list is a %s, not a number", i+1, it.T)
				}
				fs[i] = it.Float()
			}
			if len(fs) == 0 {
				return Value{}, fmt.Errorf("an empty list has nothing to play")
			}
			return Value{TGenFloat, dsl.Loop(fs...)}, nil
		}
	case KGenBool:
		if v.T.Kind == KList {
			bs := make([]bool, len(v.Items()))
			for i, it := range v.Items() {
				if it.T.Kind != KBool {
					return Value{}, fmt.Errorf("item %d of the list is a %s, not a boolean", i+1, it.T)
				}
				bs[i] = it.V.(bool)
			}
			if len(bs) == 0 {
				return Value{}, fmt.Errorf("an empty list has nothing to play")
			}
			return Value{TGenBool, dsl.Loop(bs...)}, nil
		}
	case KGenNotes:
		if v.T.Kind == KScore {
			return Value{TGenNotes, v.V.(*Score).gen()}, nil
		}
	case KGenChord:
		if v.T.Kind == KList {
			items := v.Items()
			if len(items) == 0 {
				return Value{}, fmt.Errorf("an empty list has nothing to play")
			}
			if items[0].T.Kind == KNum { // one chord, played again and again
				items = []Value{v}
			}
			chords := make([][]float64, len(items))
			for i, it := range items {
				if it.T.Kind != KList {
					return Value{}, fmt.Errorf("item %d of the list is a %s, not a chord (a list of notes)", i+1, it.T)
				}
				for _, n := range it.Items() {
					if n.T.Kind != KNum {
						return Value{}, fmt.Errorf("a chord is a list of notes, but item %d has a %s in it", i+1, n.T)
					}
					chords[i] = append(chords[i], n.Float())
				}
			}
			return Value{TGenChord, dsl.Loop(chords...)}, nil
		}
	case KList:
		if v.T.Kind == KList && want.Elem == nil {
			return v, nil
		}
		if v.T.Kind == KList {
			out := make([]Value, len(v.Items()))
			for i, it := range v.Items() {
				c, err := coerce(it, *want.Elem)
				if err != nil {
					return Value{}, fmt.Errorf("item %d of the list: %w", i+1, err)
				}
				out[i] = c
			}
			return listV(out), nil
		}
	}
	return Value{}, mismatch(v, want)
}

// mismatch says what was wanted and what there is, and for the two rates what is the matter
func mismatch(v Value, want Type) error {
	msg := fmt.Sprintf("expected a %s, got a %s", want, v.T)
	switch {
	case want.Kind == KSig && v.T.isGen():
		msg += ": a generator makes a value for every step (event rate) and a signal is audio rate; a generator is turned into a signal by the function that takes it (notes gives a voice with a pitch and a gate)"
	case want.isGen() && v.T.Kind == KSig:
		msg += ": a signal is audio rate, and a generator is at event rate, one value for every step"
	case want.isGen() && v.T.isGen():
		msg += ": they make different things"
	}
	return fmt.Errorf("%s", msg)
}

// describe is the signature of a function, for messages and for the listing
func describe(f *Func) string {
	parts := make([]string, len(f.Params))
	for i, p := range f.Params {
		s := p.Name + ": " + p.Type.String()
		if p.Default != nil {
			if x, ok := p.Default.V.(float64); ok && math.IsNaN(x) {
				s += " = the preset's own" // a setting that a preset sets, unless it is given
			} else {
				s += " = " + fmt.Sprint(p.Default.V)
			}
		}
		parts[i] = s
	}
	return strings.ReplaceAll(f.Name+"("+strings.Join(parts, ", ")+") -> "+f.Result.String(), "empty list", "list")
}

var _ = blocks.Note{} // the kinds of generators are those of blocks and dsl

// sharedSignals are the signals that generators have been made into, so that a generator that is used as a signal in two places (a
// phrase that drives a cutoff and a pan) is one stream: it is pulled once a step, and both have the same value. It is for the
// program that is being built (Build starts it empty)
var sharedSignals = map[any]dsl.Sig{}

// heldSignal is the signal of a generator: held for a step. key says which generator it is (the value that was given), and a key that
// cannot be compared makes a signal of its own
func heldSignal(g dsl.Gen[float64], key any) dsl.Sig {
	if key == nil || !reflect.TypeOf(key).Comparable() {
		return dsl.Hold(g)
	}
	if s, ok := sharedSignals[key]; ok {
		return s
	}
	s := dsl.Hold(g)
	sharedSignals[key] = s
	return s
}
