package lang

import (
	"fmt"
	"math"
	"path/filepath"

	"harp/dsl"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/chord"
	"github.com/almerlucke/genny/fugue"
	"github.com/almerlucke/genny/isorhythm"
	"github.com/almerlucke/genny/line"
	"github.com/almerlucke/genny/melody"
	"github.com/almerlucke/genny/midi"
	"github.com/almerlucke/genny/note"
	"github.com/almerlucke/genny/phase"
	"github.com/almerlucke/genny/scale"
)

func scaleFrom(root float64, steps []float64) (s *scale.Scale, err error) {
	err = safely(func() { s = scale.New(root, steps) })
	return
}

func (e *evaluator) midiScore(path string, track int) (Value, error) {
	f, err := midi.ReadFileWith(path, e.patch.StepsPerBeat)
	if err != nil {
		return Value{}, fmt.Errorf("%s: %v", filepath.Base(path), err)
	}
	var events []note.Event
	for i, t := range f.Tracks {
		if track < 0 || track == i {
			events = append(events, t.Events...)
		}
	}
	if track >= len(f.Tracks) {
		return Value{}, fmt.Errorf("the file has %d tracks: track is from 0 to %d (or -1 for all)", len(f.Tracks), len(f.Tracks)-1)
	}
	if len(events) == 0 {
		return Value{}, fmt.Errorf("there are no notes in %s", filepath.Base(path))
	}
	e.patch.Info = append(e.patch.Info, fmt.Sprintf("  %s: %d tracks, %d notes (steps are %g a beat)", filepath.Base(path), len(f.Tracks), len(events), e.patch.StepsPerBeat))
	return scoreV(events, 0), nil
}

// voiceOf picks one voice of several lines (-1 is all of them together)
func voiceOf(voices [][]note.Event, voice int) (Value, error) {
	if voice >= len(voices) || voice < -1 {
		return Value{}, fmt.Errorf("there are %d voices: voice is from 0 to %d (or -1 for all)", len(voices), len(voices)-1)
	}
	if voice >= 0 {
		return scoreV(voices[voice], 0), nil
	}
	return scoreV(line.Merge(voices...), 0), nil
}

func init() {
	add(&Func{Name: "phrase", Result: TScore, Doc: "a score from a list of pitches that follow each other, each as long as the next of durations (in steps, going round): phrase([60 62 64 -1 67], [2 1 1 2 4]). A pitch below 0 is a rest. velocity and legato (the part of its duration a note is held) are a number, or a list with a value for each pitch (a rest has one too), going round: velocity: [1 0.5 0.8], legato: [1 0.5]. A score plays on any synth: a synth takes it where it takes notes",
		Params: []Param{{"pitches", ListOf(TNum), nil}, {"durations", ListOf(TNum), listOf(1)}, {"velocity", TAny, def(0.8)}, {"legato", TAny, def(1)}},
		Call: func(a Args) (Value, error) {
			p, err := numbersOf(a.value("pitches"), "pitches")
			if err != nil {
				return Value{}, err
			}
			d, err := numbersOf(a.value("durations"), "durations")
			if err != nil {
				return Value{}, err
			}
			vel, err := numbersOf(a.value("velocity"), "velocity")
			if err != nil {
				return Value{}, err
			}
			leg, err := numbersOf(a.value("legato"), "legato")
			if err != nil {
				return Value{}, err
			}
			var events []note.Event
			t := 0.0
			for i, pitch := range p {
				dur := d[i%len(d)]
				if !(dur > 0) || dur > 4096 {
					return Value{}, fmt.Errorf("a duration is above 0 steps (%g)", dur)
				}
				v, l := vel[i%len(vel)], leg[i%len(leg)]
				if pitch >= 0 && v > 0 { // a velocity of 0 is a rest as well
					if !(l > 0) {
						return Value{}, fmt.Errorf("legato is above 0 (%g)", l)
					}
					events = append(events, note.Event{Pitch: pitch, Velocity: v, Start: t, Duration: dur * l})
				}
				t += dur
			}
			return scoreV(events, t), nil
		}})
	add(&Func{Name: "midiFile", Result: TScore, Doc: "a score from a MIDI file: its notes, in the steps of this patch (steps a beat; the tempo is that of the patch). track picks one track (counted from 0), -1 takes all of them together. harp check says how many tracks and notes there are. Look at the tracks one at a time to give each its own synth",
		Params: []Param{{"file", TStr, nil}, {"track", TNum, def(-1)}},
		Call: func(a Args) (Value, error) {
			path, err := a.ev.findFile(a.Str("file"))
			if err != nil {
				return Value{}, err
			}
			return a.ev.midiScore(path, a.Int("track"))
		}})
	add(&Func{Name: "isorhythm", Result: TScore, Doc: "an isorhythmic piece: the talea is a list of durations in steps (a negative one is a rest) and the color a list of pitches; the two go round at their own lengths, and come back together only after their lowest common multiple. length is how many steps it lasts",
		Params: []Param{{"talea", ListOf(TNum), nil}, {"color", ListOf(TNum), nil}, {"length", TNum, nil}},
		Call: func(a Args) (Value, error) {
			t, err := numbersOf(a.value("talea"), "talea")
			if err != nil {
				return Value{}, err
			}
			c, err := numbersOf(a.value("color"), "color")
			if err != nil {
				return Value{}, err
			}
			if l := a.Num("length"); !(l > 0) || l > 1<<16 {
				return Value{}, fmt.Errorf("length is from 1 to 65536 steps")
			}
			ev := isorhythm.Events(t, c, a.Num("length"))
			if len(ev) == 0 {
				return Value{}, fmt.Errorf("the talea has no notes (a duration above 0)")
			}
			return scoreV(ev, a.Num("length")), nil
		}})
	add(&Func{Name: "chordTones", Result: TScore, Doc: "a melody over chords, one chord to a bar: on the first step of each beat a note of the chord (the first of the bar strongest), between them steps along the scale, and the last step of a bar a chord tone again. chords is a list of chords (lists of notes), repeated if there are more bars. By chance: seed it",
		Params: []Param{{"chords", ListOf(ListOf(TNum)), nil}, {"root", TNum, nil}, {"scale", TScale, nil}, {"bars", TNum, def(0)}, {"barSteps", TNum, def(16)}, {"beatSteps", TNum, def(4)},
			{"maxLeap", TNum, def(3)}, {"low", TNum, def(60)}, {"high", TNum, def(84)}},
		Call: func(a Args) (Value, error) {
			var cs []chord.Chord
			for i, c := range a.value("chords").Items() {
				x, err := floats(c, "chords")
				if err != nil {
					return Value{}, fmt.Errorf("chord %d: %w", i+1, err)
				}
				cs = append(cs, x)
			}
			if len(cs) == 0 {
				return Value{}, fmt.Errorf("the list has no chords")
			}
			s, err := scaleFrom(a.Num("root"), scaleSteps(a, "scale"))
			if err != nil {
				return Value{}, err
			}
			bars := a.Int("bars")
			if bars == 0 {
				bars = len(cs)
			}
			if bars < 1 || bars > 1024 || a.Int("barSteps") < 1 || a.Int("beatSteps") < 1 || a.Int("barSteps")%a.Int("beatSteps") != 0 || a.Num("high")-a.Num("low") < 12 {
				return Value{}, fmt.Errorf("bars is from 1 to 1024, barSteps a multiple of beatSteps, and high at least an octave above low")
			}
			var ev []note.Event
			if err := safely(func() {
				ev = melody.ChordTonesList(cs, bars, melody.Config{Scale: s, BarSteps: a.Int("barSteps"), BeatSteps: a.Int("beatSteps"), MaxLeap: a.Int("maxLeap"), Low: a.Num("low"), High: a.Num("high")})
			}); err != nil {
				return Value{}, err
			}
			return scoreV(ev, float64(bars*a.Int("barSteps"))), nil
		}})
	add(&Func{Name: "fugue", Result: TScore, Doc: "the exposition of a fugue: a subject (pitches that are all notes of the key, and a duration in steps for each) enters in each of the voices in turn, as the subject and as the answer in the fifth, against a countersubject that is written for it by the rules of counterpoint. voice picks one voice (0 is the highest) to give its own synth, -1 takes all of them together. root and scale are the key; barSteps is the length of a bar. By chance: seed it",
		Params: []Param{{"subject", ListOf(TNum), nil}, {"durations", ListOf(TNum), nil}, {"voices", TNum, def(3)}, {"root", TNum, nil}, {"scale", TScale, nil}, {"voice", TNum, def(-1)}, {"barSteps", TNum, def(16)}},
		Call: func(a Args) (Value, error) {
			p, err := numbersOf(a.value("subject"), "subject")
			if err != nil {
				return Value{}, err
			}
			d, err := numbersOf(a.value("durations"), "durations")
			if err != nil {
				return Value{}, err
			}
			if len(p) != len(d) {
				return Value{}, fmt.Errorf("%d pitches and %d durations: a duration for each", len(p), len(d))
			}
			if a.Int("voices") < 1 || a.Int("voices") > 8 || a.Int("barSteps") < 1 {
				return Value{}, fmt.Errorf("voices is from 1 to 8")
			}
			key, err := scaleFrom(a.Num("root"), scaleSteps(a, "scale"))
			if err != nil {
				return Value{}, err
			}
			var f *fugue.Fugue
			if err := safely(func() {
				sub := fugue.SubjectFromPitches(key, p, d)
				f, _ = fugue.Exposition(sub, a.Int("voices"), key, fugue.Config{StepsPerBar: float64(a.Int("barSteps"))})
			}); err != nil {
				return Value{}, fmt.Errorf("the subject: %v (every pitch of it is a note of the key)", err)
			}
			v, err := voiceOf(f.Voices, a.Int("voice"))
			if err != nil {
				return Value{}, err
			}
			v.V.(*Score).Len = f.End()
			return v, nil
		}})
	add(&Func{Name: "phasing", Result: TScore, Doc: "phasing music as Steve Reich made it: a pattern (a score) is played by two voices, one of which is a little faster by drift steps every round, so that it slowly moves away from the other and after some rounds is in unison again. voice 0 is the steady one, 1 the one that moves, -1 both. cycles is how many rounds (0: until they are together again)",
		Params: []Param{{"pattern", TScore, nil}, {"drift", TNum, nil}, {"cycles", TNum, def(0)}, {"voice", TNum, def(-1)}},
		Call: func(a Args) (Value, error) {
			p := scoreOf(a, "pattern")
			length := phase.Length(p.Events)
			if !(a.Num("drift") > 0) || !(a.Num("drift") < length) {
				return Value{}, fmt.Errorf("drift is above 0 and below the length of the pattern (%g steps)", length)
			}
			n := a.Int("cycles")
			if n == 0 {
				n = phase.Cycles(p.Events, a.Num("drift"))
			}
			if n < 1 || n > 4096 {
				return Value{}, fmt.Errorf("cycles is from 1 to 4096 (here it would be %d: a drift that divides the pattern in a whole number of steps comes back sooner)", n)
			}
			first, second := phase.Events(p.Events, n, a.Num("drift"))
			v, err := voiceOf([][]note.Event{first, second}, a.Int("voice"))
			if err == nil {
				v.V.(*Score).Len = float64(n) * length
			}
			return v, err
		}})
	add(&Func{Name: "canon", Result: TScore, Doc: "a canon: voices that all play the score, the next one delay steps later and interval semitones higher (a canon at the fifth after two bars of 16 steps is delay 32, interval 7). voice picks one (0 is the score itself), -1 takes all of them together",
		Params: []Param{{"score", TScore, nil}, {"delay", TNum, nil}, {"interval", TNum, def(0)}, {"voices", TNum, def(2)}, {"voice", TNum, def(-1)}},
		Call: func(a Args) (Value, error) {
			if a.Int("voices") < 1 || a.Int("voices") > 16 || a.Num("delay") < 0 {
				return Value{}, fmt.Errorf("voices is from 1 to 16, delay is 0 or more")
			}
			s := scoreOf(a, "score")
			v, err := voiceOf(line.Canon(s.Events, a.Num("delay"), a.Num("interval"), a.Int("voices")), a.Int("voice"))
			if err == nil {
				v.V.(*Score).Len = float64(s.span()) + a.Num("delay")*float64(a.Int("voices")-1)
			}
			return v, err
		}})

	// ---- changes of a score
	add(&Func{Name: "transpose", Result: TScore, Doc: "a score moved up by semitones",
		Params: []Param{{"score", TScore, nil}, {"semitones", TNum, nil}},
		Call: func(a Args) (Value, error) {
			s := scoreOf(a, "score")
			return scoreV(line.Transpose(s.Events, a.Num("semitones")), s.Len), nil
		}})
	add(&Func{Name: "stretch", Result: TScore, Doc: "a score longer (factor above 1, augmentation) or shorter (below 1, diminution): starts and lengths are multiplied",
		Params: []Param{{"score", TScore, nil}, {"factor", TNum, nil}},
		Call: func(a Args) (Value, error) {
			if f := a.Num("factor"); !(f > 0) || f > 64 {
				return Value{}, fmt.Errorf("factor is above 0 (and at most 64)")
			}
			s := scoreOf(a, "score")
			return scoreV(line.Stretch(s.Events, a.Num("factor")), s.Len*a.Num("factor")), nil
		}})
	add(&Func{Name: "mirror", Result: TScore, Doc: "a score turned upside down around the pitch axis: a note as far under it as it was over it. With a scale (and root) the mirror is in the scale, else every semitone counts",
		Params: []Param{{"score", TScore, nil}, {"axis", TNum, nil}, {"root", TNum, def(math.NaN())}, {"scale", TScale, &Value{TScale, scale.Chromatic}}},
		Call: func(a Args) (Value, error) {
			s := scoreOf(a, "score")
			var sc *scale.Scale
			if r := a.Num("root"); !math.IsNaN(r) {
				var err error
				if sc, err = scaleFrom(r, scaleSteps(a, "scale")); err != nil {
					return Value{}, err
				}
			}
			return scoreV(line.Invert(s.Events, a.Num("axis"), sc), s.Len), nil
		}})
	add(&Func{Name: "retro", Result: TScore, Doc: "a score played backwards (the crab canon): the note that ended last starts first",
		Params: []Param{{"score", TScore, nil}},
		Call: func(a Args) (Value, error) {
			s := scoreOf(a, "score")
			return scoreV(line.Retrograde(s.Events), s.Len), nil
		}})
	add(&Func{Name: "shift", Result: TScore, Doc: "a score that starts later by steps (or earlier, with a negative number: what starts before the start is cut)",
		Params: []Param{{"score", TScore, nil}, {"steps", TNum, nil}},
		Call: func(a Args) (Value, error) {
			s := scoreOf(a, "score")
			return scoreV(line.Shift(s.Events, a.Num("steps")), math.Max(s.Len+a.Num("steps"), 0)), nil
		}})
	add(&Func{Name: "together", Result: TScore, Doc: "scores that play at once, as one: together([bass, melody])",
		Params: []Param{{"scores", ListOf(TScore), nil}},
		Call: func(a Args) (Value, error) {
			var all [][]note.Event
			length := 0.0
			for _, s := range a.value("scores").Items() {
				all = append(all, s.V.(*Score).Events)
				length = math.Max(length, float64(s.V.(*Score).span()))
			}
			if len(all) == 0 {
				return Value{}, fmt.Errorf("the list has no scores")
			}
			return scoreV(line.Merge(all...), length), nil
		}})
	add(&Func{Name: "cut", Result: TScore, Doc: "the part of a score from step from up to step to, moved to start at 0",
		Params: []Param{{"score", TScore, nil}, {"from", TNum, def(0)}, {"to", TNum, nil}},
		Call: func(a Args) (Value, error) {
			if !(a.Num("to") > a.Num("from")) {
				return Value{}, fmt.Errorf("to is after from")
			}
			return scoreV(line.Slice(scoreOf(a, "score").Events, a.Num("from"), a.Num("to")), a.Num("to")-a.Num("from")), nil
		}})
	add(&Func{Name: "tile", Result: TScore, Doc: "a score repeated n times, each pass length steps after the one before (the length of the score if left out)",
		Params: []Param{{"score", TScore, nil}, {"n", TNum, nil}, {"length", TNum, def(0)}},
		Call: func(a Args) (Value, error) {
			s := scoreOf(a, "score")
			l := a.Num("length")
			if l == 0 {
				l = float64(s.span())
			}
			if a.Int("n") < 1 || a.Int("n") > 4096 || !(l > 0) {
				return Value{}, fmt.Errorf("n is from 1 to 4096, length above 0")
			}
			return scoreV(line.Loop(s.Events, a.Int("n"), l), l*float64(a.Int("n"))), nil
		}})
	add(&Func{Name: "pad", Result: TScore, Doc: "a score that goes round after steps steps: to leave silence at the end (or to cut the end short: the notes that start after it do not play)",
		Params: []Param{{"score", TScore, nil}, {"steps", TNum, nil}},
		Call: func(a Args) (Value, error) {
			if !(a.Num("steps") >= 1) || a.Num("steps") > 1<<20 {
				return Value{}, fmt.Errorf("steps is from 1")
			}
			s := scoreOf(a, "score")
			return scoreV(line.Slice(s.Events, 0, a.Num("steps")), a.Num("steps")), nil
		}})
	add(&Func{Name: "tune", Result: TScore, Doc: "a score with every pitch moved to the nearest note of a scale on a root",
		Params: []Param{{"score", TScore, nil}, {"root", TNum, nil}, {"scale", TScale, nil}},
		Call: func(a Args) (Value, error) {
			sc, err := scaleFrom(a.Num("root"), scaleSteps(a, "scale"))
			if err != nil {
				return Value{}, err
			}
			s := scoreOf(a, "score")
			return scoreV(line.Quantize(s.Events, sc), s.Len), nil
		}})
	add(&Func{Name: "thin", Result: TScore, Doc: "a score with notes left out: a value of the rhythm for each note in turn, a note stays where it is a hit (thin(score, euclid(7, 5)))",
		Params: []Param{{"score", TScore, nil}, {"keep", TGenBool, nil}},
		Call: func(a Args) (Value, error) {
			s := scoreOf(a, "score")
			return scoreV(line.Thin(s.Events, a.value("keep").V.(dsl.Gen[bool])), s.Len), nil
		}})

	// ---- a score for play: the rhythm and the numbers of its notes
	one := func(name, doc string, val func(note.Event) float64) {
		add(&Func{Name: name, Result: TGenFloat, Doc: doc,
			Params: []Param{{"score", TScore, nil}},
			Call: func(a Args) (Value, error) {
				ev := scoreOf(a, "score").oneVoice()
				if len(ev) == 0 {
					return Value{}, fmt.Errorf("the score has no notes")
				}
				x := make([]float64, len(ev))
				for i, e := range ev {
					x[i] = val(e)
				}
				return Value{TGenFloat, dsl.Loop(x...)}, nil
			}})
	}
	one("pitches", "the pitches of a score, one for each note in turn, going round: a field for play. When notes start together, the highest is taken (play is one voice a note)", func(e note.Event) float64 { return e.Pitch })
	one("lengths", "how long each note of a score is, in steps, one for each note in turn: a field for play (sustain)", func(e note.Event) float64 { return e.Duration })
	one("velocities", "how hard each note of a score is, 0 to 1, one for each note in turn: a field for play", func(e note.Event) float64 {
		if e.Velocity <= 0 {
			return 0.8
		}
		return e.Velocity
	})
	add(&Func{Name: "chordsOf", Result: TGenChord, Doc: "the notes of a score that start together, as chords, one for each step that a note starts on, going round: the chord field of play, with onsets as its rhythm: play(inst, rhythm: onsets(s), fields: [chordsOf(s)], voices: 4). A step with one note is a chord of one",
		Params: []Param{{"score", TScore, nil}},
		Call: func(a Args) (Value, error) {
			s := scoreOf(a, "score")
			var chords [][]float64
			for k, notes := range s.table() {
				if len(notes) == 0 {
					continue
				}
				_ = k
				c := make([]float64, len(notes))
				for i, n := range notes {
					c[i] = n.Pitch
				}
				chords = append(chords, c)
			}
			if len(chords) == 0 {
				return Value{}, fmt.Errorf("the score has no notes")
			}
			return Value{TGenChord, dsl.Loop(chords...)}, nil
		}})
	add(&Func{Name: "onsets", Result: TGenBool, Doc: "the rhythm of a score: a hit on each step that a note starts on, going round: the rhythm of play, with the pitches of the score as a field: play(inst, rhythm: onsets(s), fields: [pitches(s)])",
		Params: []Param{{"score", TScore, nil}},
		Call: func(a Args) (Value, error) {
			s := scoreOf(a, "score")
			hits := make([]bool, s.span())
			for _, e := range s.Events {
				hits[stepOf(e)%len(hits)] = true
			}
			return Value{TGenBool, dsl.Loop(hits...)}, nil
		}})
}

func listOf(x ...float64) *Value {
	items := make([]Value, len(x))
	for i, v := range x {
		items[i] = numV(v)
	}
	v := listV(items)
	return &v
}

var _ genny.Generator[bool]
