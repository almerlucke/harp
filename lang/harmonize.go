package lang

import (
	"fmt"
	"strings"

	"github.com/almerlucke/genny/harmony"
	"github.com/almerlucke/genny/note"
)

var cadences = map[string]harmony.Cadence{"authentic": harmony.Authentic, "half": harmony.Half, "plagal": harmony.Plagal, "deceptive": harmony.Deceptive}

func init() {
	add(&Func{Name: "harmonize", Result: TScore, Doc: "chords written for four voices (soprano, alto, tenor, bass) by the rules of four part writing: no parallel fifths or octaves, no crossing voices, a leading tone that rises and a seventh that falls, small leaps. The chords are the scale degrees of degrees (0 is I, 3 is IV, 4 is V, counted from 0: [0 3 4 0]), or, when there are none, n chords that are made by the grammar of tonic, subdominant and dominant, in phrases that end in a cadence. root is the tonic (a note, any octave) and mode is major or minor. With explicit degrees seventh: true makes them seventh chords; for the chords that are made, sevenths, secondary (dominants of the next chord) and borrowed (iv in a major key) are chances from 0 to 1, phrase the chords to a phrase, cadence the way phrases end (authentic, half, plagal, deceptive; else half and authentic in turn). A chord lasts chordsteps steps. voice picks one voice (0 is the soprano, 3 the bass) to give a synth of its own, -1 takes all together. harp check says the chords, and the rules that had to be broken, if any",
		Params: []Param{{"root", TNum, nil}, {"mode", TStr, defStr("major")}, {"degrees", ListOf(TNum), listDefault()}, {"n", TNum, def(8)}, {"chordsteps", TNum, def(16)}, {"voice", TNum, def(-1)},
			{"seventh", TBool, defBool(false)}, {"sevenths", TNum, def(0)}, {"secondary", TNum, def(0)}, {"borrowed", TNum, def(0)}, {"phrase", TNum, def(4)}, {"cadence", TStr, defStr("")},
			{"variety", TNum, def(0)}, {"legato", TNum, def(1)}, {"velocity", TNum, def(0.8)}},
		Call: func(a Args) (Value, error) {
			var minor bool
			switch a.Str("mode") {
			case "major":
			case "minor":
				minor = true
			default:
				return Value{}, fmt.Errorf("mode is major or minor")
			}
			key := harmony.Key{Tonic: a.Int("root"), Minor: minor}
			steps := a.Num("chordsteps")
			if !(steps >= 1) || steps > 4096 || !(a.Num("legato") > 0) || a.Num("variety") < 0 {
				return Value{}, fmt.Errorf("chordsteps is from 1 to 4096, legato above 0, variety 0 or more")
			}
			for _, p := range []string{"sevenths", "secondary", "borrowed"} {
				if x := a.Num(p); x < 0 || x > 1 {
					return Value{}, fmt.Errorf("%s is a chance from 0 to 1", p)
				}
			}
			var chords []harmony.Chord
			if ds := a.value("degrees").Items(); len(ds) > 0 {
				for i, d := range ds {
					if d.T.Kind != KNum || !isWhole(d.Float()) || d.Float() < 0 || d.Float() > 6 {
						return Value{}, fmt.Errorf("degrees are whole numbers from 0 (I) to 6 (vii): item %d is %v", i+1, d.V)
					}
					chords = append(chords, harmony.Diatonic(key, int(d.Float()), a.value("seventh").V.(bool)))
				}
			} else {
				n := a.Int("n")
				if n < 1 || n > 256 || a.Int("phrase") < 2 || a.Int("phrase") > 64 {
					return Value{}, fmt.Errorf("n is from 1 to 256 chords, phrase from 2 to 64")
				}
				cfg := harmony.Config{Key: key, PhraseLength: a.Int("phrase"), SeventhProb: a.Num("sevenths"), SecondaryProb: a.Num("secondary"), BorrowedProb: a.Num("borrowed")}
				if c := a.Str("cadence"); c != "" {
					cad, ok := cadences[c]
					if !ok {
						return Value{}, fmt.Errorf("cadence is one of %s%s", names(cadences), didYouMean(c, presetNames(cadences)))
					}
					cfg.Cadences = map[harmony.Cadence]float64{cad: 1}
				}
				var gen *harmony.Generator
				if err := safely(func() { gen = harmony.New(cfg) }); err != nil {
					return Value{}, err
				}
				for i := 0; i < n; i++ {
					chords = append(chords, gen.Generate())
				}
			}
			var voicings []harmony.Voicing
			var broken []harmony.Violation
			if err := safely(func() { voicings, broken = harmony.Realize(chords, harmony.VoiceConfig{Variety: a.Num("variety")}) }); err != nil {
				return Value{}, err
			}
			voices := make([][]note.Event, 4)
			var numerals []string
			for i, v := range voicings {
				numerals = append(numerals, chords[i].Numeral)
				for k := 0; k < 4; k++ {
					voices[k] = append(voices[k], note.Event{Pitch: float64(v.Notes[k]), Velocity: a.Num("velocity"), Start: float64(i) * steps, Duration: steps * a.Num("legato")})
				}
			}
			info := fmt.Sprintf("  harmonize: %s in %s", strings.Join(numerals, " "), key)
			if len(broken) > 0 {
				var bs []string
				for _, b := range broken {
					bs = append(bs, b.String())
				}
				info += "; rules that had to be broken: " + strings.Join(bs, ", ")
			}
			a.ev.patch.Info = append(a.ev.patch.Info, info)
			v, err := voiceOf(voices, a.Int("voice"))
			if err == nil {
				v.V.(*Score).Len = float64(len(chords)) * steps
			}
			return v, err
		}})
}
