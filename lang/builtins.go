package lang

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"harp/blocks"
	"harp/dsl"

	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
)

// Param is a parameter of a function of the language. A parameter with a Default can be left out.
type Param struct {
	Name    string
	Type    Type
	Default *Value
}

// Func is a function of the language: what it takes, what it gives, and what it does. The thing
// before the dot in x.f(...) is the first parameter, so every function can be written both ways
type Func struct {
	Name   string
	Params []Param
	Result Type
	Doc    string
	Call   func(a Args) (Value, error)
}

// Args are the arguments of a call, by the name of the parameter, already of the type it asked for
type Args struct {
	f    *Func
	vals []Value
	ev   *evaluator // where the call is made: a function of the program is worked out from here
}

func (a Args) value(name string) Value {
	for i, p := range a.f.Params {
		if p.Name == name {
			return a.vals[i]
		}
	}
	panic("lang: " + a.f.Name + " has no parameter " + name)
}
func (a Args) Num(name string) float64 { return a.value(name).Float() }
func (a Args) Int(name string) int     { return int(math.Round(a.value(name).Float())) }
func (a Args) Sig(name string) dsl.Sig { return a.value(name).Sig() }
func (a Args) Str(name string) string  { return a.value(name).V.(string) }

func def(v float64) *Value  { x := numV(v); return &x }
func defBool(b bool) *Value { x := Value{TBool, b}; return &x }
func defStr(s string) *Value {
	x := strV(s)
	return &x
}

// funcs is every function of the language, by name
var funcs = map[string]*Func{}

// consts are the names that are always there
var consts = map[string]Value{
	"true":       {TBool, true},
	"false":      {TBool, false},
	"major":      {TScale, dsl.Major},
	"minor":      {TScale, dsl.Minor},
	"dorian":     {TScale, dsl.Dorian},
	"pentatonic": {TScale, dsl.MinorPentatonic},
	"pi":         numV(math.Pi),
}

func add(f *Func) { funcs[f.Name] = f }

func init() {
	// ---- generators: event rate, a value for every step
	add(&Func{Name: "euclid", Result: TGenBool, Doc: "a euclidean rhythm: pulses spread evenly over steps, turned by rotation",
		Params: []Param{{"steps", TNum, nil}, {"pulses", TNum, nil}, {"rotation", TNum, def(0)}},
		Call: func(a Args) (Value, error) {
			steps, pulses := a.Int("steps"), a.Int("pulses")
			if steps < 1 {
				return Value{}, fmt.Errorf("steps must be at least 1")
			}
			return Value{TGenBool, dsl.Euclid(steps, pulses, a.Int("rotation"))}, nil
		}})
	add(&Func{Name: "every", Result: TGenBool, Doc: "plays on the first of every n steps",
		Params: []Param{{"n", TNum, nil}},
		Call: func(a Args) (Value, error) {
			n := a.Int("n")
			if n < 1 {
				return Value{}, fmt.Errorf("n must be at least 1")
			}
			steps := make([]bool, n)
			steps[0] = true
			return Value{TGenBool, dsl.Loop(steps...)}, nil
		}})
	add(&Func{Name: "mutate", Result: TGenInt, Doc: "plays a phrase of scale degrees, and swaps notes around now and then (swap is the chance, 0 to 1)",
		Params: []Param{{"phrase", ListOf(TNum), nil}, {"swap", TNum, def(0.5)}},
		Call: func(a Args) (Value, error) {
			items := a.value("phrase").Items()
			if len(items) == 0 {
				return Value{}, fmt.Errorf("the phrase has no notes")
			}
			phrase := make([]int, len(items))
			for i, it := range items {
				if !isWhole(it.Float()) {
					return Value{}, fmt.Errorf("item %d of the phrase is not a whole number (a scale degree)", i+1)
				}
				phrase[i] = int(it.Float())
			}
			return Value{TGenInt, dsl.Mutate(phrase, a.Num("swap"))}, nil
		}})
	add(&Func{Name: "degrees", Result: TGenFloat, Doc: "scale degrees (0 is the root, 7 an octave up in a scale of 7 notes) as pitches, in a scale on a root note",
		Params: []Param{{"root", TNum, nil}, {"scale", TScale, nil}, {"degrees", TGenInt, nil}},
		Call: func(a Args) (Value, error) {
			return Value{TGenFloat, dsl.Degrees(a.Num("root"), a.value("scale").V.([]float64), a.value("degrees").V.(dsl.Gen[int]))}, nil
		}})
	add(&Func{Name: "notes", Result: TVoice, Doc: "a melody on a rhythm: the next pitch is taken when the rhythm plays, together with the gate that starts an envelope. Gives a voice with the signals pitch (in Hz) and gate",
		Params: []Param{{"melody", TGenFloat, nil}, {"rhythm", TGenBool, nil}, {"length", TNum, def(0.6)}},
		Call: func(a Args) (Value, error) {
			return Value{TVoice, dsl.Notes(a.value("melody").V.(dsl.Gen[float64]), a.value("rhythm").V.(dsl.Gen[bool]), a.Num("length"))}, nil
		}})
	add(&Func{Name: "chords", Result: TGenNotes, Doc: "chords on a rhythm, as notes for a synth: when the rhythm plays, all the notes of the next chord start, held for length steps. velocity (0 to 1) and length can each be a number, or a list or a generator of numbers: a sequence, one value for each chord, in turn (velocity: [1 0.5 0.8 0.5], length: [2 6 4]); a velocity of 0 is a rest",
		Params: []Param{{"chords", TGenChord, nil}, {"rhythm", TAny, nil}, {"velocity", TAny, def(0.7)}, {"length", TAny, def(1)}},
		Call: func(a Args) (Value, error) {
			vel, length, err := noteSequences(a)
			if err != nil {
				return Value{}, err
			}
			hits, err := hitsOf(a.value("rhythm"))
			if err != nil {
				return Value{}, fmt.Errorf("rhythm: %w", err)
			}
			return Value{TGenNotes, blocks.ChordsHits(a.value("chords").V.(dsl.Gen[[]float64]), hits, vel, length)}, nil
		}})
	add(&Func{Name: "melody", Result: TGenNotes, Doc: "a melody on a rhythm, as notes for a synth: the next pitch is taken when the rhythm plays. velocity (0 to 1) and length (in steps) can each be a number, or a list or a generator of numbers: a sequence, one value for each note, in turn, that goes round (velocity: [1 0.5 0.8 0.5], length: [1 1 3], velocity: wander(0.4, 1)); a velocity of 0 is a rest",
		Params: []Param{{"pitches", TGenFloat, nil}, {"rhythm", TAny, nil}, {"velocity", TAny, def(0.8)}, {"length", TAny, def(1)}},
		Call: func(a Args) (Value, error) {
			vel, length, err := noteSequences(a)
			if err != nil {
				return Value{}, err
			}
			hits, err := hitsOf(a.value("rhythm"))
			if err != nil {
				return Value{}, fmt.Errorf("rhythm: %w", err)
			}
			return Value{TGenNotes, blocks.MelodyHits(a.value("pitches").V.(dsl.Gen[float64]), hits, vel, length)}, nil
		}})

	// ---- signals: audio rate
	for _, osc := range []struct {
		name, doc string
		f         func(dsl.Sig) dsl.Sig
	}{
		{"saw", "a rising sawtooth between -1 and 1, band limited (SuperCollider's Saw is half the size and falls): freq in Hz (a list of frequencies is a signal of several channels, one oscillator each)", func(f dsl.Sig) dsl.Sig { return dsl.Saw(f) }},
		{"sine", "a sine: freq in Hz", func(f dsl.Sig) dsl.Sig { return dsl.Sine(f) }},
		{"lfo", "a sine for modulation, from -1 to 1: rate in Hz. Use range to map it", func(f dsl.Sig) dsl.Sig { return dsl.Lfo(f) }},
	} {
		osc := osc
		pname := "freq"
		if osc.name == "lfo" {
			pname = "rate"
		}
		add(&Func{Name: osc.name, Result: TSig, Doc: osc.doc, Params: []Param{{pname, TSig, nil}},
			Call: func(a Args) (Value, error) { return sigV(osc.f(a.Sig(pname))), nil }})
	}
	add(&Func{Name: "range", Result: TSig, Doc: "maps a signal from -1..1 (an lfo) to lo..hi",
		Params: []Param{{"x", TSig, nil}, {"lo", TNum, nil}, {"hi", TNum, nil}},
		Call:   func(a Args) (Value, error) { return sigV(a.Sig("x").Range(a.Num("lo"), a.Num("hi"))), nil }})
	add(&Func{Name: "adsr", Result: TEnv, Doc: "an envelope: attack, decay and release in seconds, sustain a level from 0 to 1. Give it a gate with on",
		Params: []Param{{"attack", TNum, nil}, {"decay", TNum, nil}, {"sustain", TNum, nil}, {"release", TNum, nil}},
		Call: func(a Args) (Value, error) {
			return Value{TEnv, dsl.Adsr(a.Num("attack"), a.Num("decay"), a.Num("sustain"), a.Num("release"))}, nil
		}})
	add(&Func{Name: "on", Result: TSig, Doc: "runs an envelope on a gate: a rising edge starts it, a falling edge lets it go",
		Params: []Param{{"env", TEnv, nil}, {"gate", TSig, nil}},
		Call:   func(a Args) (Value, error) { return sigV(a.value("env").V.(dsl.Env).On(a.Sig("gate"))), nil }})
	add(&Func{Name: "gain", Result: TSig, Doc: "scales a signal by a number or another signal (x * amount)",
		Params: []Param{{"x", TSig, nil}, {"amount", TSig, nil}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.Mul(a.Sig("x"), a.Sig("amount"))), nil }})
	add(&Func{Name: "spread", Result: TSig, Doc: "puts the channels of a signal in the stereo field, evenly from left to right (width 0 to 1); one channel is in the middle",
		Params: []Param{{"x", TSig, nil}, {"width", TNum, def(1)}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.Spread(a.Sig("x"), a.Num("width"))), nil }})

	// ---- psalm's blocks
	add(&Func{Name: "lowpass", Result: TSig, Doc: "psalm's state variable lowpass: cutoff in Hz (a signal moves it), res from 0 to 1. A filter for every channel",
		Params: []Param{{"x", TSig, nil}, {"cutoff", TSig, nil}, {"res", TSig, def(0.5)}},
		Call: func(a Args) (Value, error) {
			return sigV(blocks.Lowpass(a.Sig("x"), a.Sig("cutoff"), a.Sig("res"))), nil
		}})
	add(&Func{Name: "highpass", Result: TSig, Doc: "psalm's state variable highpass: cutoff in Hz, res from 0 to 1",
		Params: []Param{{"x", TSig, nil}, {"cutoff", TSig, nil}, {"res", TSig, def(0.5)}},
		Call: func(a Args) (Value, error) {
			return sigV(blocks.Highpass(a.Sig("x"), a.Sig("cutoff"), a.Sig("res"))), nil
		}})
	add(&Func{Name: "echo", Result: TSig, Doc: "a delay with feedback: max is the longest time in seconds, time can be a signal (a list gives every channel its own), mix is how much of the echoes is heard. A time that moves bends the pitch of what is in the line (the echoes squeak): keep it fixed, or move it slowly",
		Params: []Param{{"x", TSig, nil}, {"max", TNum, nil}, {"time", TSig, nil}, {"feedback", TSig, def(0.4)}, {"mix", TSig, def(0.3)}},
		Call: func(a Args) (Value, error) {
			if a.Num("max") <= 0 {
				return Value{}, fmt.Errorf("max is the longest delay in seconds, and has to be above 0")
			}
			return sigV(blocks.Echo(a.Sig("x"), a.Num("max"), a.Sig("time"), a.Sig("feedback"), a.Sig("mix"))), nil
		}})
	add(&Func{Name: "reverb", Result: TSig, Doc: "psalm's feedback delay network reverb: mix from 0 (dry) to 1, decay is the reverb time in seconds. Stereo out",
		Params: []Param{{"x", TSig, nil}, {"mix", TNum, def(0.25)}, {"decay", TNum, def(2.5)}},
		Call: func(a Args) (Value, error) {
			x := a.Sig("x")
			if n := x.Channels(); n > 2 {
				return Value{}, fmt.Errorf("a reverb takes one or two channels, this signal has %d: spread it into stereo first", n)
			}
			cfg := fdnreverb.DefaultConfig()
			cfg.Mix, cfg.Decay = a.Num("mix"), a.Num("decay")
			return sigV(blocks.FDNReverb(x, cfg)), nil
		}})
	add(&Func{Name: "subtractive", Result: TSig, Doc: "psalm's subtractive synth playing notes, with one of its presets (see presets) and voices notes at once. Stereo out",
		Params: []Param{{"notes", TGenNotes, nil}, {"preset", TStr, defStr("pad")}, {"voices", TNum, def(8)}},
		Call: func(a Args) (Value, error) {
			make, ok := subtractivePresets[a.Str("preset")]
			if !ok {
				return Value{}, fmt.Errorf("there is no subtractive preset %q%s", a.Str("preset"), didYouMean(a.Str("preset"), presetNames(subtractivePresets)))
			}
			voices := a.Int("voices")
			if voices < 1 {
				return Value{}, fmt.Errorf("a synth needs at least one voice")
			}
			return sigV(blocks.Subtractive(make(), voices, a.value("notes").V.(dsl.Gen[[]blocks.Note]))), nil
		}})
	add(&Func{Name: "supersaw", Result: TSig, Doc: "psalm's supersaw, the sound of the JP-8000, playing notes, with one of its presets and voices notes at once. Stereo out",
		Params: []Param{{"notes", TGenNotes, nil}, {"preset", TStr, defStr("trance_pad")}, {"voices", TNum, def(8)}},
		Call: func(a Args) (Value, error) {
			make, ok := supersawPresets[a.Str("preset")]
			if !ok {
				return Value{}, fmt.Errorf("there is no supersaw preset %q%s", a.Str("preset"), didYouMean(a.Str("preset"), presetNames(supersawPresets)))
			}
			voices := a.Int("voices")
			if voices < 1 {
				return Value{}, fmt.Errorf("a synth needs at least one voice")
			}
			return sigV(blocks.Supersaw(make(), voices, a.value("notes").V.(dsl.Gen[[]blocks.Note]))), nil
		}})
}

func init() {
	// ---- sources and shapers in the way of SuperCollider's UGens of the same kind (see dsl/sources.go)
	add(&Func{Name: "hz", Result: TSig, Doc: "MIDI notes as frequencies in Hz (midicps): 69 is 440. A signal of notes, a sweep, is a signal of frequencies",
		Params: []Param{{"note", TSig, nil}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.Hz(a.Sig("note"))), nil }})
	add(&Func{Name: "ramp", Result: TSig, Doc: "a rising ramp from -1 to 1, freq times a second, that starts at 0 and jumps back to -1 (LFSaw). Not band limited, for modulation. With control: true as SuperCollider's .kr: one value for every block of 64 samples",
		Params: []Param{{"freq", TSig, nil}, {"control", TBool, defBool(false)}},
		Call: func(a Args) (Value, error) {
			if a.value("control").V.(bool) {
				return sigV(dsl.RampControl(a.Sig("freq"))), nil
			}
			return sigV(dsl.Ramp(a.Sig("freq"))), nil
		}})
	add(&Func{Name: "noise", Result: TSig, Doc: "white noise from -1 to 1 (WhiteNoise); every noise is a noise of its own",
		Call: func(a Args) (Value, error) { return sigV(dsl.Noise()), nil }})
	add(&Func{Name: "lfnoise", Result: TSig, Doc: "a new random value from -1 to 1, freq times a second: in straight lines between them (LFNoise1), or in steps with interp: false (LFNoise0). With control: true as SuperCollider's .kr: one value for every block of 64 samples, so that it moves in steps of 1.5 ms",
		Params: []Param{{"freq", TSig, nil}, {"interp", TBool, defBool(true)}, {"control", TBool, defBool(false)}},
		Call: func(a Args) (Value, error) {
			if a.value("control").V.(bool) {
				return sigV(dsl.LFNoiseControl(a.Sig("freq"), a.value("interp").V.(bool))), nil
			}
			return sigV(dsl.LFNoise(a.Sig("freq"), a.value("interp").V.(bool))), nil
		}})
	add(&Func{Name: "impulse", Result: TSig, Doc: "a 1 at the start and then freq times a second, and 0 between",
		Params: []Param{{"freq", TSig, nil}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.Impulse(a.Sig("freq"))), nil }})
	add(&Func{Name: "dust", Result: TSig, Doc: "random impulses of a value from 0 to 1, density of them a second on average",
		Params: []Param{{"density", TSig, nil}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.Dust(a.Sig("density"))), nil }})
	add(&Func{Name: "decay", Result: TSig, Doc: "makes impulses an exponential decay that falls 60 dB in time seconds",
		Params: []Param{{"x", TSig, nil}, {"time", TSig, nil}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.Decay(a.Sig("x"), a.Sig("time"))), nil }})
	add(&Func{Name: "decay2", Result: TSig, Doc: "makes impulses a swell that rises in about attack seconds and falls 60 dB in decay seconds (Decay2)",
		Params: []Param{{"x", TSig, nil}, {"attack", TSig, nil}, {"decay", TSig, nil}},
		Call: func(a Args) (Value, error) {
			return sigV(dsl.Decay2(a.Sig("x"), a.Sig("attack"), a.Sig("decay"))), nil
		}})
	add(&Func{Name: "line", Result: TSig, Doc: "goes from start to end in seconds, in a straight line, and stays at end",
		Params: []Param{{"start", TNum, nil}, {"end", TNum, nil}, {"seconds", TNum, nil}},
		Call: func(a Args) (Value, error) {
			return sigV(dsl.Line(a.Num("start"), a.Num("end"), a.Num("seconds"))), nil
		}})
	add(&Func{Name: "xline", Result: TSig, Doc: "goes from start to end in seconds by the same ratio every second, and stays at end. Neither can be 0",
		Params: []Param{{"start", TNum, nil}, {"end", TNum, nil}, {"seconds", TNum, nil}},
		Call: func(a Args) (Value, error) {
			if a.Num("start") == 0 || a.Num("end") == 0 || (a.Num("start") < 0) != (a.Num("end") < 0) {
				return Value{}, fmt.Errorf("start and end can't be 0, and must be both above or both below 0")
			}
			return sigV(dsl.XLine(a.Num("start"), a.Num("end"), a.Num("seconds"))), nil
		}})
	add(&Func{Name: "exprange", Result: TSig, Doc: "maps a signal from -1..1 to lo..hi along an exponential curve, the way pitch is heard (exprange). Neither can be 0",
		Params: []Param{{"x", TSig, nil}, {"lo", TNum, nil}, {"hi", TNum, nil}},
		Call: func(a Args) (Value, error) {
			if a.Num("lo") == 0 || a.Num("hi") == 0 || (a.Num("lo") < 0) != (a.Num("hi") < 0) {
				return Value{}, fmt.Errorf("lo and hi can't be 0, and must be both above or both below 0")
			}
			return sigV(a.Sig("x").ExpRange(a.Num("lo"), a.Num("hi"))), nil
		}})
	add(&Func{Name: "pan", Result: TSig, Doc: "puts a signal in the stereo field: pos from -1 (left) to 1 (right), equal power as Pan2. A signal of several channels gives left and right for every channel",
		Params: []Param{{"x", TSig, nil}, {"pos", TSig, def(0)}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.Pan(a.Sig("x"), a.Sig("pos"))), nil }})
	add(&Func{Name: "mix", Result: TSig, Doc: "adds all the channels of a signal into one (Mix)",
		Params: []Param{{"x", TSig, nil}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.Sum(a.Sig("x"))), nil }})
	add(&Func{Name: "dup", Result: TSig, Doc: "gives a signal of one channel n channels, all the same (x ! n)",
		Params: []Param{{"x", TSig, nil}, {"n", TNum, def(2)}},
		Call: func(a Args) (Value, error) {
			if c := a.Sig("x").Channels(); c != 1 {
				return Value{}, fmt.Errorf("dup takes a signal of one channel, this has %d", c)
			}
			if a.Int("n") < 1 {
				return Value{}, fmt.Errorf("n must be at least 1")
			}
			return sigV(dsl.Dup(a.Sig("x"), a.Int("n"))), nil
		}})
	add(&Func{Name: "bandpass", Result: TSig, Doc: "psalm's state variable bandpass: cutoff in Hz, res from 0 to 1 (near 1 it is narrow). Its gain at the cutoff is the Q, which is higher than a unity bandpass (Resonz)",
		Params: []Param{{"x", TSig, nil}, {"cutoff", TSig, nil}, {"res", TSig, def(0.5)}},
		Call: func(a Args) (Value, error) {
			return sigV(blocks.Bandpass(a.Sig("x"), a.Sig("cutoff"), a.Sig("res"))), nil
		}})
	add(&Func{Name: "comb", Result: TSig, Doc: "a comb filter with a feedback delay of delay seconds that rings for decay seconds (60 dB), as CombN: only the delayed sound is heard",
		Params: []Param{{"x", TSig, nil}, {"delay", TNum, nil}, {"decay", TNum, nil}},
		Call: func(a Args) (Value, error) {
			delay, decay := a.Num("delay"), a.Num("decay")
			if delay <= 0 || decay <= 0 {
				return Value{}, fmt.Errorf("delay and decay have to be above 0")
			}
			return sigV(blocks.Echo(a.Sig("x"), delay, delay, math.Pow(0.001, delay/decay), 1.0)), nil
		}})
	for _, w := range []struct {
		name, doc string
		wave      blosc.Waveform
	}{
		{"triangle", "a triangle wave from -1 to 1: freq in Hz (psalm's band limited oscillator)", blosc.Triangle},
		{"square", "a square wave from -1 to 1: freq in Hz (psalm's band limited oscillator)", blosc.Square},
	} {
		w := w
		add(&Func{Name: w.name, Result: TSig, Doc: w.doc, Params: []Param{{"freq", TSig, nil}},
			Call: func(a Args) (Value, error) { return sigV(blocks.Osc(w.wave, a.Sig("freq"), 0.5)), nil }})
	}
	add(&Func{Name: "pulse", Result: TSig, Doc: "a band limited pulse wave between -1 and 1: +1 for the first width of the period and -1 for the rest, freq in Hz, width from 0 to 1 (0.5 is a square). With a width other than 0.5 it has a mean of 2*width-1 (a DC offset; rectangle has none). SuperCollider's Pulse is not this: it is half the size, has a mean of 0 and is turned over (see examples/supercollider/6_pan_pulse.harp)",
		Params: []Param{{"freq", TSig, nil}, {"width", TSig, def(0.5)}},
		Call: func(a Args) (Value, error) {
			return sigV(dsl.Pulse(a.Sig("freq"), a.Sig("width"))), nil
		}})
	add(&Func{Name: "rectangle", Result: TSig, Doc: "psalm's band limited pulse wave, with no DC offset: its mean is 0 for every width (so it swings between about +1.6 and -0.8 at a width of 0.3), freq in Hz, width from 0 to 1",
		Params: []Param{{"freq", TSig, nil}, {"width", TSig, def(0.5)}},
		Call: func(a Args) (Value, error) {
			return sigV(blocks.Osc(blosc.Rectangle, a.Sig("freq"), a.Sig("width"))), nil
		}})
}

func presetNames[T any](m map[string]T) []string {
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// didYouMean is a hint for a name that is not there: the nearest of the names, if one is near
func didYouMean(name string, names []string) string {
	best, bestD := "", 1<<30
	for _, n := range names {
		if d := distance(name, n); d < bestD {
			best, bestD = n, d
		}
	}
	if best != "" && bestD <= max(2, len(name)/3) {
		return fmt.Sprintf(" (did you mean %s?)", best)
	}
	return ""
}

// distance is the edit distance of two strings
func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// Functions lists the functions of the language, with what they take and do
func Functions() string {
	names := make([]string, 0, len(funcs))
	for n := range funcs {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		fmt.Fprintf(&b, "%s\n    %s\n", describe(funcs[n]), funcs[n].Doc)
	}
	return b.String()
}

func init() {
	// ---- what is decided while the patch is built (not while it plays): random numbers and choices, from the seeded stream
	add(&Func{Name: "coin", Result: TBool, Doc: "true with the chance p (0 to 1), false otherwise: decided when the patch is built, for if",
		Params: []Param{{"p", TNum, def(0.5)}},
		Call:   func(a Args) (Value, error) { return Value{TBool, dsl.Random() < a.Num("p")}, nil }})
	add(&Func{Name: "rand", Result: TNum, Doc: "a random number from lo up to hi, decided when the patch is built",
		Params: []Param{{"lo", TNum, nil}, {"hi", TNum, nil}},
		Call: func(a Args) (Value, error) {
			return numV(a.Num("lo") + (a.Num("hi")-a.Num("lo"))*dsl.Random()), nil
		}})
	add(&Func{Name: "exprand", Result: TNum, Doc: "a random number from lo up to hi, with the same chance for every ratio (ExpRand), decided when the patch is built. Neither can be 0",
		Params: []Param{{"lo", TNum, nil}, {"hi", TNum, nil}},
		Call: func(a Args) (Value, error) {
			lo, hi := a.Num("lo"), a.Num("hi")
			if lo <= 0 || hi <= 0 {
				return Value{}, fmt.Errorf("lo and hi must be above 0 (this is %v and %v)", lo, hi)
			}
			return numV(lo * math.Pow(hi/lo, dsl.Random())), nil
		}})
	add(&Func{Name: "choose", Result: TAny, Doc: "one item of a list, picked at random when the patch is built",
		Params: []Param{{"items", Type{Kind: KList}, nil}},
		Call: func(a Args) (Value, error) {
			items := a.value("items").Items()
			if len(items) == 0 {
				return Value{}, fmt.Errorf("an empty list has nothing to choose")
			}
			return items[int(dsl.Random()*float64(len(items)))], nil
		}})
	add(&Func{Name: "repeat", Result: TAny, Doc: "gives x to the function f, gives what comes out to f again, and so on, n times: repeat(f, 16, x). A function with random choices in it makes new ones every time",
		Params: []Param{{"f", TFunc, nil}, {"n", TNum, nil}, {"x", TAny, nil}},
		Call: func(a Args) (Value, error) {
			f, ok := a.value("f").V.(*Func)
			if !ok || len(f.Params) != 1 {
				return Value{}, fmt.Errorf("f must be a function of the program that takes one argument")
			}
			n := a.Int("n")
			if n < 0 || n > 1000 {
				return Value{}, fmt.Errorf("n is %d: from 0 to 1000", n)
			}
			x := a.value("x")
			for i := 0; i < n; i++ {
				var err error
				if x, err = a.ev.apply(f, x); err != nil {
					return Value{}, err
				}
			}
			return x, nil
		}})

	// ---- more of SuperCollider's UGens
	add(&Func{Name: "linexp", Result: TSig, Doc: "maps a signal from the range inlo..inhi to lo..hi along an exponential curve (LinExp); lo and hi can be signals; the input is not clipped",
		Params: []Param{{"x", TSig, nil}, {"inlo", TNum, nil}, {"inhi", TNum, nil}, {"lo", TSig, nil}, {"hi", TSig, nil}},
		Call: func(a Args) (Value, error) {
			return sigV(dsl.LinExp(a.Sig("x"), a.Num("inlo"), a.Num("inhi"), a.Sig("lo"), a.Sig("hi"))), nil
		}})
	add(&Func{Name: "clip2", Result: TSig, Doc: "limits a signal to -level..level (clip2)",
		Params: []Param{{"x", TSig, nil}, {"level", TSig, def(1)}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.Clip2(a.Sig("x"), a.Sig("level"))), nil }})
	add(&Func{Name: "product", Result: TSig, Doc: "multiplies all the channels of a signal into one",
		Params: []Param{{"x", TSig, nil}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.Product(a.Sig("x"))), nil }})
	add(&Func{Name: "reverse", Result: TSig, Doc: "the channels of a signal in the other order",
		Params: []Param{{"x", TSig, nil}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.Reverse(a.Sig("x"))), nil }})
	add(&Func{Name: "bpf", Result: TSig, Doc: "a bandpass that lets the cutoff through at its own level, as SuperCollider's BPF: freq in Hz, rq is the bandwidth as a part of freq (0.2 is a Q of 5)",
		Params: []Param{{"x", TSig, nil}, {"freq", TSig, nil}, {"rq", TNum, def(1)}},
		Call: func(a Args) (Value, error) {
			rq := a.Num("rq")
			if rq <= 0 || rq > 2 {
				return Value{}, fmt.Errorf("rq is %v: above 0 and up to 2", rq)
			}
			return sigV(dsl.Mul(blocks.Bandpass(a.Sig("x"), a.Sig("freq"), 1-rq/2), rq)), nil
		}})
}

func init() {
	add(&Func{Name: "scsaw", Result: TSig, Doc: "SuperCollider's Saw as SuperCollider works it out: half the size, falling, the frequency read once in 64 samples, and it rings when the frequency jumps (saw does not). To mimic a patch of SuperCollider",
		Params: []Param{{"freq", TSig, nil}},
		Call:   func(a Args) (Value, error) { return sigV(dsl.SCSaw(a.Sig("freq"))), nil }}) //nolint
}

// noteSequences are the velocity and the length of the notes of melody and chords: each a number, or a list or a generator of numbers (a value
// for each note, in turn)
func noteSequences(a Args) (vel, length dsl.Gen[float64], err error) {
	if vel, err = floatGen(a.value("velocity")); err != nil {
		return nil, nil, fmt.Errorf("velocity: %w", err)
	}
	if length, err = floatGen(a.value("length")); err != nil {
		return nil, nil, fmt.Errorf("length: %w", err)
	}
	return vel, length, nil
}

// hitsOf is a rhythm as hits: the hits of a pattern, or a rhythm of booleans (a hit on the start of every step that is true; a list of booleans too)
func hitsOf(v Value) (dsl.Gen[[]dsl.Hit], error) {
	if v.T.Kind == KGenHits {
		return v.V.(dsl.Gen[[]dsl.Hit]), nil
	}
	b, err := coerce(v, TGenBool)
	if err != nil {
		return nil, err
	}
	return dsl.HitsOf(b.V.(dsl.Gen[bool])), nil
}
