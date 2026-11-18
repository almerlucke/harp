package lang

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/note"
	"github.com/almerlucke/psalm/tools/harmony"
)

// A part is a melody and its rhythm in one text, as psalm's part module reads it: a step each, C3 Eb3 G3 - Bb3 . G3 Eb3, as a score.
//
//	C3 Eb3 48    a note (a name, or a number), . a rest, - the note before held a step longer
//	C3+Eb3+G3    a chord          [C3 D3 Eb3] a group that fills a step          C3^3 a roll of 3        *2 ?25 %2/4 ! as in a pattern
//	@1 @3 @5     chord tones of the chords that are given (chords: "Am F C G", a bar each): @1 the root, @2 the next tone, @3 the fifth, @4 the
//	             seventh (in a triad the root an octave up), @-1 the tone below the root, @b the bass, @c the whole chord; ' after one an octave up,
//	             , an octave down

// partNote reads C3, Eb3, f#4, bb-1, or a number: the pitch it is (C4 is 60)
func partNote(s string) (float64, bool) {
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v, true
	}
	if s == "" || strings.IndexByte("abcdefgABCDEFG", s[0]) < 0 {
		return 0, false
	}
	pc := map[byte]int{'c': 0, 'd': 2, 'e': 4, 'f': 5, 'g': 7, 'a': 9, 'b': 11}[strings.ToLower(s[:1])[0]]
	i := 1
	for i < len(s) && (s[i] == '#' || s[i] == 'b') {
		if s[i] == '#' {
			pc++
		} else {
			pc--
		}
		i++
	}
	oct, err := strconv.Atoi(s[i:])
	if err != nil || oct < -2 || oct > 9 {
		return 0, false
	}
	return float64(12*(oct+1) + pc), true
}

type partContext struct {
	chords   []change
	total    float64 // in bars
	barSteps float64
	register float64
	bass     float64
}

func (pc *partContext) chordAt(step float64) *harmony.Chord {
	if len(pc.chords) == 0 {
		return nil
	}
	x := math.Mod(step/pc.barSteps, pc.total)
	if x < 0 {
		x += pc.total
	}
	at := 0.0
	for _, c := range pc.chords {
		if x < at+c.bars {
			return c.c
		}
		at += c.bars
	}
	return pc.chords[len(pc.chords)-1].c
}

// pitches is what a token plays at a step: a note, or notes (C3+E3), or the chord tones that @ asks of the chord of that moment (none: a rest)
func (pc *partContext) pitches(tok string, step float64) ([]float64, error) {
	var out []float64
	for _, atom := range strings.Split(tok, "+") {
		shift := 0.0
		for strings.HasSuffix(atom, "'") || strings.HasSuffix(atom, ",") {
			if strings.HasSuffix(atom, "'") {
				shift += 12
			} else {
				shift -= 12
			}
			atom = atom[:len(atom)-1]
		}
		if !strings.HasPrefix(atom, "@") {
			p, ok := partNote(atom)
			if !ok {
				return nil, fmt.Errorf("%q is not a note (C3, Eb3, f#4, or a number) or a chord tone (@1, @3, @b, @c)", atom)
			}
			out = append(out, p+shift)
			continue
		}
		c := pc.chordAt(step)
		if len(pc.chords) == 0 {
			return nil, fmt.Errorf("%s asks for a chord tone, and there are no chords (chords: \"Am F C G\")", atom)
		}
		if c == nil {
			continue
		}
		root := rootAbove(pc.register, c.Root)
		tone := func(i int) float64 {
			n := len(c.Intervals)
			return root + float64(c.Intervals[((i%n)+n)%n]) + 12*float64(floorDiv(i, n))
		}
		switch w := atom[1:]; w {
		case "b":
			out = append(out, c.BassPitch(pc.bass)+shift)
		case "c":
			for i := range c.Intervals {
				out = append(out, tone(i)+shift)
			}
		default:
			n, err := strconv.Atoi(w)
			if err != nil || n == 0 {
				return nil, fmt.Errorf("%s: @1 is the root, @2 the next tone, @3 the fifth, @4 the seventh, @-1 the tone below the root, @b the bass, @c the whole chord", atom)
			}
			i := n - 1
			if n < 0 {
				i = n
			}
			out = append(out, tone(i)+shift)
		}
	}
	return out, nil
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func lcm(a, b int) int {
	x, y := a, b
	for y != 0 {
		x, y = y, x%y
	}
	return a / x * b
}

func init() {
	add(&Func{Name: "part", Result: TScore, Doc: "a melody and its rhythm in one text, as a score: part(\"C3 Eb3 G3 - Bb3 . G3 Eb3\"), a step each. A note is a name (C3, Eb3, f#4; C4 is 60) or a number, . a rest, - the note before held a step longer, C3+Eb3+G3 a chord, [C3 D3 Eb3] a group that fills a step, C3^3 a roll of three, *2 how many steps, ?25 a chance, %2/4 only some passes through it (the text is played passes times: 0 gives as many as the conditions need), ! as in a pattern. chords (\"Am F C G\": symbols, or roman numerals in key, a bar each of barSteps) gives chord tones to write: @1 the root, @2 the next tone, @3 the fifth, @4 the seventh, @-1 the tone below the root, @b the bass, @c the whole chord, ' after one an octave up and , down: \"@1 @3 @5 @3\" follows the chords. register is the lowest note of @1 (48), bass of @b (36). legato is how much of its step a note is held (0.9)",
		Params: []Param{{"text", TStr, nil}, {"chords", TStr, defStr("")}, {"key", TStr, defStr("C major")}, {"barSteps", TNum, def(16)}, {"register", TNum, def(48)}, {"bass", TNum, def(36)},
			{"velocity", TNum, def(0.8)}, {"legato", TNum, def(0.9)}, {"passes", TNum, def(0)}},
		Call: func(a Args) (Value, error) {
			hits, steps, err := parseTextPattern(a.Str("text"), true)
			if err != nil {
				return Value{}, err
			}
			ctx := &partContext{barSteps: a.Num("barSteps"), register: a.Num("register"), bass: a.Num("bass")}
			if ctx.barSteps < 1 {
				return Value{}, fmt.Errorf("barSteps is 1 or more")
			}
			if c := a.Str("chords"); c != "" {
				if ctx.chords, ctx.total, err = parseChanges(c, a.Str("key")); err != nil {
					return Value{}, fmt.Errorf("chords: %w", err)
				}
			}
			if !(a.Num("legato") > 0) || a.Num("velocity") < 0 || a.Num("velocity") > 1 {
				return Value{}, fmt.Errorf("legato is above 0, velocity from 0 to 1")
			}
			passes := a.Int("passes")
			if passes == 0 {
				passes = 1
				for _, h := range hits {
					if h.cond.Of > 0 {
						passes = min(lcm(passes, h.cond.Of), 64)
					}
				}
			}
			if passes < 1 || passes > 64 {
				return Value{}, fmt.Errorf("passes is from 1 to 64")
			}
			var events []note.Event
			for pass := 0; pass < passes; pass++ {
				base := float64(pass) * steps
				var kept []textHit
				for _, h := range hits {
					if !h.cond.Zero() && !h.cond.Holds(int64(pass), false) {
						continue
					}
					if h.chance < 1 && genny.Float64() >= h.chance {
						continue
					}
					kept = append(kept, h)
				}
				for _, h := range kept {
					ps, err := ctx.pitches(h.tok, base+h.at)
					if err != nil {
						return Value{}, err
					}
					dur := h.len
					if dur == 0 {
						dur = h.slot * a.Num("legato") // of its own step: a rest after it does not make it longer
					}
					for _, p := range ps {
						events = append(events, note.Event{Pitch: p, Velocity: a.Num("velocity"), Start: base + h.at, Duration: dur})
					}
				}
			}
			return scoreV(events, float64(passes)*steps), nil
		}})
}
