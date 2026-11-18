package lang

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"harp/blocks"
	"harp/dsl"
)

// The UGens of SuperCollider as it works them out (see dsl/sc*.go), and the notes of a pattern played on an instrument that is a function of the program

var envShapes = map[string]int{"curve": dsl.EnvCurve, "lin": dsl.EnvLinear, "exp": dsl.EnvExp, "sine": dsl.EnvSine, "welch": dsl.EnvWelch,
	"step": dsl.EnvStep, "squared": dsl.EnvSquared, "cubed": dsl.EnvCubed, "hold": dsl.EnvHold}

func floats(v Value, what string) ([]float64, error) {
	if v.T.Kind == KNum {
		return []float64{v.Float()}, nil
	}
	if v.T.Kind != KList {
		return nil, fmt.Errorf("%s: a list of numbers (or one number), this is a %s", what, v.T)
	}
	out := make([]float64, len(v.Items()))
	for i, it := range v.Items() {
		if it.T.Kind != KNum {
			return nil, fmt.Errorf("%s: item %d is a %s, not a number", what, i+1, it.T)
		}
		out[i] = it.Float()
	}
	return out, nil
}

func init() {
	add(&Func{Name: "env", Result: TSig, Doc: "an envelope as SuperCollider's EnvGen: it starts when the gate rises, goes through levels in times (seconds, one fewer than the levels), each segment bent by its curve (0 is straight, negative is fast first, a list for a curve for each segment). shape: how a segment goes (curve uses the curve number; exp is XLine). With release: n it waits after segment n for the gate to fall. control: true is .kr (a value for every 64 samples)",
		Params: []Param{{"gate", TSig, nil}, {"levels", TAny, nil}, {"times", TAny, nil}, {"curve", TAny, def(0)}, {"release", TNum, def(-1)}, {"loop", TNum, def(-1)}, {"control", TBool, defBool(false)}, {"shape", TStr, defStr("curve")}},
		Call: func(a Args) (Value, error) {
			levels, err := floats(a.value("levels"), "levels")
			if err != nil {
				return Value{}, err
			}
			times, err := floats(a.value("times"), "times")
			if err != nil {
				return Value{}, err
			}
			if len(levels) < 2 || len(times) != len(levels)-1 {
				return Value{}, fmt.Errorf("with %d levels there are %d times, this has %d", len(levels), max(len(levels)-1, 1), len(times))
			}
			curves, err := floats(a.value("curve"), "curve")
			if err != nil {
				return Value{}, err
			}
			if len(curves) != 1 && len(curves) != len(times) {
				return Value{}, fmt.Errorf("curve is one number, or one for each of the %d times", len(times))
			}
			shape, ok := envShapes[a.Str("shape")]
			if !ok {
				return Value{}, fmt.Errorf("shape %q: curve, lin, exp, sine, welch, step, squared, cubed or hold", a.Str("shape"))
			}
			spec := dsl.EnvSpec{Start: levels[0], ReleaseNode: a.Int("release"), LoopNode: a.Int("loop")}
			for i, t := range times {
				c := curves[0]
				if len(curves) > 1 {
					c = curves[i]
				}
				spec.Segs = append(spec.Segs, dsl.EnvSeg{Level: levels[i+1], Dur: t, Shape: shape, Curve: c})
			}
			if spec.ReleaseNode >= len(levels) {
				return Value{}, fmt.Errorf("release is the number of a segment: 0 to %d", len(times))
			}
			return sigV(dsl.EnvGen(a.Sig("gate"), spec, a.value("control").V.(bool))), nil
		}})
	add(&Func{Name: "scsawsync", Result: TSig, Doc: "scsaw that starts again from phase 0 when sync rises above 0, as a new synth of SuperCollider does for every note",
		Params: []Param{{"freq", TSig, nil}, {"sync", TSig, nil}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.SCSawSync(a.Sig("freq"), a.Sig("sync"))), nil }})
	add(&Func{Name: "scsin", Result: TSig, Doc: "SuperCollider's SinOsc as it works it out: a table of 8192 values, a phase that is a whole number (so the frequency is a little off), the start phase in radians",
		Params: []Param{{"freq", TSig, nil}, {"phase", TNum, def(0)}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.SCSinOsc(a.Sig("freq"), a.Num("phase"))), nil }})
	add(&Func{Name: "sclpf", Result: TSig, Doc: "SuperCollider's LPF: a second order lowpass, cutoff in Hz",
		Params: []Param{{"x", TSig, nil}, {"freq", TSig, nil}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.SCLPF(a.Sig("x"), a.Sig("freq"))), nil }})
	add(&Func{Name: "schpf", Result: TSig, Doc: "SuperCollider's HPF: a second order highpass, cutoff in Hz",
		Params: []Param{{"x", TSig, nil}, {"freq", TSig, nil}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.SCHPF(a.Sig("x"), a.Sig("freq"))), nil }})
	add(&Func{Name: "scrlpf", Result: TSig, Doc: "SuperCollider's RLPF: a resonant lowpass, rq the reciprocal of Q (1 is flat, 0.5 resonates)",
		Params: []Param{{"x", TSig, nil}, {"freq", TSig, nil}, {"rq", TSig, def(1)}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.SCRLPF(a.Sig("x"), a.Sig("freq"), a.Sig("rq"))), nil }})
	add(&Func{Name: "scpeq", Result: TSig, Doc: "SuperCollider's BPeakEQ: boosts or cuts db decibel around freq, rq the bandwidth as a part of freq",
		Params: []Param{{"x", TSig, nil}, {"freq", TSig, nil}, {"rq", TSig, def(1)}, {"db", TSig, def(0)}},
		Call: func(a Args) (Value, error) {
			return sigV(dsl.SCBPeakEQ(a.Sig("x"), a.Sig("freq"), a.Sig("rq"), a.Sig("db"))), nil
		}})
	delay := func(name, doc string, f func(x dsl.Sig, max float64, t dsl.Sig) dsl.Sig) {
		add(&Func{Name: name, Result: TSig, Doc: doc,
			Params: []Param{{"x", TSig, nil}, {"max", TNum, nil}, {"time", TSig, nil}},
			Call: func(a Args) (Value, error) {
				if a.Num("max") <= 0 {
					return Value{}, fmt.Errorf("max is the longest time in seconds: above 0")
				}
				return sigV(f(a.Sig("x"), a.Num("max"), a.Sig("time"))), nil
			}})
	}
	delay("scdelayn", "SuperCollider's DelayN: a delay of time seconds (up to max), cut to whole samples",
		func(x dsl.Sig, m float64, t dsl.Sig) dsl.Sig { return dsl.SCDelayN(x, m, t) })
	delay("scdelayc", "SuperCollider's DelayC: a delay of time seconds (up to max), with cubic interpolation, for a time that moves",
		func(x dsl.Sig, m float64, t dsl.Sig) dsl.Sig { return dsl.SCDelayC(x, m, t) })
	one := func(name, doc string, f func(dsl.Sig) dsl.Sig) {
		add(&Func{Name: name, Result: TSig, Doc: doc, Params: []Param{{"x", TSig, nil}},
			Call: func(a Args) (Value, error) { return sigV(f(a.Sig("x"))), nil }})
	}
	one("tanh", "the hyperbolic tangent: a soft clip between -1 and 1", dsl.Tanh)
	one("atan", "the arc tangent: a soft clip between -pi/2 and pi/2", dsl.Atan)
	one("sign", "-1, 0 or 1: the sign", dsl.Sign)
	one("midiratio", "the ratio of frequencies that many semitones apart: 12 is 2", dsl.MidiRatio)
	add(&Func{Name: "dbamp", Result: TNum, Doc: "decibels as an amplitude: -6 is 0.5 (about), 0 is 1",
		Params: []Param{{"db", TNum, nil}},
		Call:   func(a Args) (Value, error) { return numV(math.Pow(10, a.Num("db")/20)), nil }})
	add(&Func{Name: "freeverb2", Result: TSig, Doc: "SuperCollider's FreeVerb2: the reverb of Freeverb for a signal of two channels; mix, room and damp from 0 to 1",
		Params: []Param{{"x", TSig, nil}, {"mix", TNum, def(0.33)}, {"room", TNum, def(0.5)}, {"damp", TNum, def(0.5)}},
		Call: func(a Args) (Value, error) {
			x := a.Sig("x")
			if x.Channels() != 2 {
				return Value{}, fmt.Errorf("the signal has %d channels: freeverb2 takes two (left and right)", x.Channels())
			}
			return sigV(dsl.SCFreeVerb2(x.Ch(0), x.Ch(1), a.Num("mix"), a.Num("room"), a.Num("damp"))), nil
		}})
	add(&Func{Name: "geom", Result: ListOf(TNum), Doc: "n numbers from lo to hi, each the same ratio from the one before: geom(40, 100, 4000)",
		Params: []Param{{"n", TNum, nil}, {"lo", TNum, nil}, {"hi", TNum, nil}},
		Call: func(a Args) (Value, error) {
			n, lo, hi := a.Int("n"), a.Num("lo"), a.Num("hi")
			if n < 1 || n > 100000 || lo <= 0 || hi <= 0 {
				return Value{}, fmt.Errorf("n is 1 or more, lo and hi are above 0")
			}
			items := make([]Value, n)
			for i := range items {
				p := 0.0
				if n > 1 {
					p = float64(i) / float64(n-1)
				}
				items[i] = numV(lo * math.Pow(hi/lo, p))
			}
			return listV(items), nil
		}})
	add(&Func{Name: "fill", Result: Type{Kind: KList}, Doc: "a list of what the function f gives for 0, 1, ... n-1 (and, if f takes a second argument, with): a function with random choices in it gives something else every time",
		Params: []Param{{"f", TFunc, nil}, {"n", TNum, nil}, {"with", TAny, def(0)}},
		Call: func(a Args) (Value, error) {
			f, ok := a.value("f").V.(*Func)
			if !ok || len(f.Params) < 1 || len(f.Params) > 2 {
				return Value{}, fmt.Errorf("f must be a function of the program that takes the number, and with a second argument for the value of with")
			}
			n := a.Int("n")
			if n < 1 || n > 1000 {
				return Value{}, fmt.Errorf("n is from 1 to 1000")
			}
			items := make([]Value, n)
			for i := range items {
				args := []Value{numV(float64(i))}
				if len(f.Params) == 2 {
					args = append(args, a.value("with"))
				}
				v, err := a.ev.apply(f, args...)
				if err != nil {
					return Value{}, err
				}
				items[i] = v
			}
			return listV(items), nil
		}})

	add(&Func{Name: "play", Result: TSig, Doc: "plays notes on an instrument that is a function of the program, as a Pbind plays a SynthDef: the function is given the gate and then one number for each of the fields, and every note is a new run of it. when the notes are: dur: the beats of each note in turn (a list, or a generator of beats; a whole number of steps), or rhythm: a generator of booleans that is pulled every step (euclid(16, 5)): a note on every true, as long as the steps up to the next. legato: the part of it that the gate is high (or sustain: so many beats), fields: a list with, for each number the instrument takes, a list or a generator (degrees(...), mutate(...)) that is pulled once for every note, or a signal (seq(phrase), lfo(0.1)) that has its value read at the start of every note: so one stream can drive both the notes and an effect voices: notes go to the voices in turn, each of which is a run of the function of its own (random choices in it are made again for each), so that the end of a note goes on while the next one starts. The notes start at the block of 64 samples they fall in, as SuperCollider's messages do",
		Params: []Param{{"instrument", TFunc, nil}, {"dur", TAny, def(0)}, {"rhythm", TAny, def(0)}, {"legato", TNum, def(0.8)}, {"sustain", TNum, def(0)}, {"fields", TAny, listDefault()}, {"voices", TNum, def(1)},
			{"swing", TNum, def(0)}, {"humanize", TNum, def(0)}, {"accent", TAny, listDefault()}},
		Call: func(a Args) (Value, error) {
			f, ok := a.value("instrument").V.(*Func)
			if !ok {
				return Value{}, fmt.Errorf("instrument must be a function of the program")
			}
			spb := a.ev.patch.StepsPerBeat
			var fields []dsl.Gen[float64]
			var sampled []dsl.Sig // for a field that is a signal: the signal, which is read at the start of every note
			chordAt, maxNotes := -1, 0
			voices := a.Int("voices")
			if fv := a.value("fields"); fv.T.Kind == KList {
				for i, it := range fv.Items() {
					// a field that is a chord (a generator of chords, or a list of chords): every note of it is a note of the instrument, on the same step
					if isChordValue(it) {
						if chordAt >= 0 {
							return Value{}, fmt.Errorf("fields: item %d is a chord, and so is item %d: only one field can be", chordAt+1, i+1)
						}
						cg, err := coerce(it, TGenChord)
						if err != nil {
							return Value{}, fmt.Errorf("fields: item %d: %w", i+1, err)
						}
						if it.T.Kind == KList {
							for _, c := range it.Items() {
								maxNotes = max(maxNotes, len(c.Items()))
							}
						}
						chordAt = i
						fields, sampled = append(fields, dsl.ChordField(cg.V.(dsl.Gen[[]float64]))), append(sampled, dsl.Sig{})
						continue
					}
					if it.T.Kind == KSig {
						if it.Sig().Channels() != 1 {
							return Value{}, fmt.Errorf("fields: item %d is a signal of %d channels: one channel is read", i+1, it.Sig().Channels())
						}
						fields, sampled = append(fields, nil), append(sampled, it.Sig())
						continue
					}
					g, err := floatGen(it)
					if err != nil {
						return Value{}, fmt.Errorf("fields: item %d: %w", i+1, err)
					}
					fields, sampled = append(fields, g), append(sampled, dsl.Sig{})
				}
			} else {
				return Value{}, fmt.Errorf("fields is a list of lists or generators of numbers")
			}
			if want := len(f.Params) - 1; want != len(fields) {
				return Value{}, fmt.Errorf("the instrument %s takes the gate and %d numbers, fields has %d", f.Name, want, len(fields))
			}
			legato, sustain := a.Num("legato"), a.Num("sustain")
			hold := func(n float64) float64 { return legato * n }
			if sustain > 0 {
				hold = func(float64) float64 { return sustain * spb }
			}
			dv, rv := a.value("dur"), a.value("rhythm")
			var events dsl.Gen[dsl.Event]
			switch {
			case rv.T.Kind != KNum && dv.T.Kind != KNum:
				return Value{}, fmt.Errorf("dur and rhythm are two ways to say when the notes are: give one")
			case rv.T.Kind != KNum:
				rg, err := coerce(rv, TGenBool)
				if err != nil {
					return Value{}, fmt.Errorf("rhythm: %w", err)
				}
				events = dsl.RhythmEvents(rg.V.(dsl.Gen[bool]), hold, fields)
			case dv.T.Kind != KNum || dv.Float() != 0:
				dg, err := durGen(dv, spb)
				if err != nil {
					return Value{}, err
				}
				events = dsl.EventGen(dg, hold, fields)
			default:
				return Value{}, fmt.Errorf("when are the notes? dur: [1 0.5] (beats, in turn) or rhythm: euclid(16, 5) (a step each)")
			}
			if voices < 1 || voices > 32 {
				return Value{}, fmt.Errorf("voices is from 1 to 32")
			}
			if maxNotes > voices {
				return Value{}, fmt.Errorf("a chord of %d notes needs voices: %d or more (a note of a chord is played by a voice of its own)", maxNotes, maxNotes)
			}
			// what the notes are, for a patch that takes over from this one: the rhythm, how long a gate is held, and each field are told apart,
			// so that a live edit of one of them goes on with the pattern and puts only that one in place (see dsl.EventFP)
			fp := dsl.EventFP{Fields: make([]string, len(sampled))}
			fp.Timing = joinFP(a.value("dur"), a.value("rhythm"))
			fp.Hold = joinFP(a.value("legato"), a.value("sustain"))
			for i, it := range a.value("fields").Items() {
				fp.Fields[i] = fpOf(it)
			}
			// the groove: swing (every other step is late by so many steps: 0.33 is a triplet feel), humanize (every step is early or late by
			// at most so many steps, at random) and accent (how strong the note on each step is: a list of levels that goes round, like accents(...))
			swing, humanize := a.Num("swing"), a.Num("humanize")
			if swing < 0 || swing > 0.9 || humanize < 0 || humanize > 0.5 || math.IsNaN(swing+humanize) {
				return Value{}, fmt.Errorf("swing is from 0 to 0.9 steps (0.33 is a triplet feel), humanize from 0 to 0.5")
			}
			groove := dsl.Groove{Swing: swing, Humanize: humanize}
			if humanize != 0 {
				groove.Seed = uint64(dsl.Random() * (1 << 53)) // from the random numbers of the patch: the same patch and seed are humanized the same
			}
			av := a.value("accent")
			if !(av.T.Kind == KList && len(av.Items()) == 0) {
				lv, err := floats(av, "accent")
				if err != nil {
					return Value{}, fmt.Errorf("accent is a list of levels, one for each step, that goes round (accents([3 3 2]) makes one): %w", err)
				}
				groove.Accent = lv
			}
			fp.Groove = joinFP(a.value("swing"), a.value("humanize"), av)
			if chordAt >= 0 {
				events = dsl.ChordEvents(events)
			}
			events = &fpEvents{Gen: events}
			ev := dsl.EventsGrooved(events, voices, sampled, fp, groove)
			var out dsl.Sig
			for v := range ev {
				args := []Value{sigV(ev[v].Gate)}
				for _, s := range ev[v].Vals {
					args = append(args, sigV(s))
				}
				var r Value
				var err error
				// every note is a new synth: what the instrument makes starts again from its start on the rise of the gate
				dsl.WithReset(ev[v].Gate, func() { r, err = a.ev.apply(f, args...) })
				if err != nil {
					return Value{}, err
				}
				s, err := coerce(r, TSig)
				if err != nil {
					return Value{}, fmt.Errorf("the instrument gives a %s: %w", r.T, err)
				}
				if len(groove.Accent) > 0 { // a note is as strong as the accent of its step
					s = sigV(dsl.Mul(s.Sig(), ev[v].Level))
				}
				if v == 0 {
					out = s.Sig()
				} else {
					if s.Sig().Channels() != out.Channels() {
						return Value{}, fmt.Errorf("the instrument gives %d channels, and %d for another voice", s.Sig().Channels(), out.Channels())
					}
					out = dsl.Add(out, s.Sig())
				}
			}
			return sigV(out), nil
		}})
}

// isChordValue says that a field is a chord: a generator of chords, or a list of chords (lists of notes)
func isChordValue(v Value) bool {
	if v.T.Kind == KGenChord {
		return true
	}
	return v.T.Kind == KList && len(v.Items()) > 0 && v.Items()[0].T.Kind == KList
}

func listDefault() *Value { v := listV(nil); return &v }

// floatGen is a generator of numbers from a list, a number or a generator of numbers or of whole numbers
func floatGen(v Value) (dsl.Gen[float64], error) {
	switch v.T.Kind {
	case KGenFloat:
		return v.V.(dsl.Gen[float64]), nil
	case KGenInt:
		return intAsFloat{v.V.(dsl.Gen[int])}, nil
	}
	g, err := coerce(v, TGenFloat)
	if err != nil {
		return nil, err
	}
	return g.V.(dsl.Gen[float64]), nil
}

type intAsFloat struct{ g dsl.Gen[int] }

func (i intAsFloat) Generate() float64 { return float64(i.g.Generate()) }
func (i intAsFloat) Continuous() bool  { return i.g.Continuous() }
func (i intAsFloat) Done() bool        { return i.g.Done() }
func (i intAsFloat) Reset()            { i.g.Reset() }

// durGen turns the beats of a list or a generator into steps: a whole number of them, 1 or more
func durGen(v Value, spb float64) (dsl.Gen[float64], error) {
	if v.T.Kind == KList || v.T.Kind == KNum {
		beats, err := floats(v, "dur")
		if err != nil || len(beats) == 0 {
			return nil, fmt.Errorf("dur is a list of beats (a number for every note in turn)")
		}
		steps := make([]float64, len(beats))
		for i, d := range beats {
			steps[i] = d * spb
			if steps[i] < 1 || math.Abs(steps[i]-math.Round(steps[i])) > 1e-9 {
				return nil, fmt.Errorf("dur %v is %v steps (%v a beat): a note is a whole number of steps, 1 or more (steps = ... says how many in a beat)", d, steps[i], spb)
			}
			steps[i] = math.Round(steps[i])
		}
		return dsl.Loop(steps...), nil
	}
	g, err := floatGen(v)
	if err != nil {
		return nil, fmt.Errorf("dur: %w", err)
	}
	return scaled{g, spb}, nil
}

// scaled is a generator of beats as steps, rounded to whole ones of at least 1 (a generator cannot be checked before it plays)
type scaled struct {
	g   dsl.Gen[float64]
	spb float64
}

func (s scaled) Generate() float64 { return math.Max(1, math.Round(s.g.Generate()*s.spb)) }
func (s scaled) Continuous() bool  { return s.g.Continuous() }
func (s scaled) Done() bool        { return s.g.Done() }
func (s scaled) Reset()            { s.g.Reset() }

func init() {
	add(&Func{Name: "seq", Result: TSig, Doc: "a signal that takes the next value of a list or a generator at every step, and holds it: seq([400 1200 800]). glide: so many milliseconds to move to the new value (0 steps at once). Where a signal is wanted, a generator can be given as it is (it steps). A generator with a name is one stream: used as a signal in several places it is pulled once a step and every place has the same values",
		Params: []Param{{"values", TAny, nil}, {"glide", TNum, def(0)}},
		Call: func(a Args) (Value, error) {
			g, err := floatGen(a.value("values"))
			if err != nil {
				return Value{}, fmt.Errorf("values: %w", err)
			}
			s := heldSignal(g, seqKey(a.value("values")))
			if ms := a.Num("glide"); ms > 0 {
				s = blocks.Smooth(s, ms)
			} else if ms < 0 {
				return Value{}, fmt.Errorf("glide is milliseconds: 0 or more")
			}
			return sigV(s), nil
		}})
	add(&Func{Name: "glide", Result: TSig, Doc: "makes a signal that jumps (a generator, a seq) move to its new values in about ms milliseconds, with a one pole lowpass (psalm's smoother)",
		Params: []Param{{"x", TSig, nil}, {"ms", TNum, def(30)}},
		Call: func(a Args) (Value, error) {
			if a.Num("ms") <= 0 {
				return Value{}, fmt.Errorf("ms is above 0")
			}
			return sigV(blocks.Smooth(a.Sig("x"), a.Num("ms"))), nil
		}})
	add(&Func{Name: "pulses", Result: TSig, Doc: "a gate from a rhythm: 1 for length (0 to 1) of the step on every step where the rhythm gives true, 0 otherwise. Use it with on(env, gate) or as a multiplier",
		Params: []Param{{"rhythm", TGenBool, nil}, {"length", TNum, def(0.5)}},
		Call: func(a Args) (Value, error) {
			if l := a.Num("length"); l <= 0 || l > 1 {
				return Value{}, fmt.Errorf("length is above 0 and up to 1 (a part of the step)")
			}
			// the same rhythm with the same length is the same gate wherever it is used
			key := pulseKey{a.value("rhythm").V, a.Num("length")}
			if reflect.TypeOf(key.g).Comparable() {
				if s, ok := sharedGates[key]; ok {
					return sigV(s), nil
				}
			}
			s := dsl.Gate(a.value("rhythm").V.(dsl.Gen[bool]), a.Num("length"))
			if reflect.TypeOf(key.g).Comparable() {
				sharedGates[key] = s
			}
			return sigV(s), nil
		}})
}

type pulseKey struct {
	g      any
	length float64
}

var sharedGates = map[any]dsl.Sig{}

// seqKey is the generator that a value of seq is, if it is one already (a list makes a generator of its own every time)
func seqKey(v Value) any {
	switch v.T.Kind {
	case KGenFloat, KGenInt:
		return v.V
	}
	return nil
}

// joinFP is the fingerprint of values together: empty if one of them cannot say what it is
func joinFP(vs ...Value) string {
	parts := make([]string, len(vs))
	for i, v := range vs {
		if parts[i] = fpOf(v); parts[i] == "" {
			return ""
		}
	}
	return strings.Join(parts, ", ")
}
