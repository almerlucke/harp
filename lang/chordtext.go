package lang

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/almerlucke/psalm/tools/harmony"
)

// Chords written as text, the way psalm's harmony module reads them: symbols (Dm7 G7 Cmaj7 Bb/D F#m7b5) or roman numerals in a key (i VI V7), a
// number of bars each (*2, *0.5), - the chord before held a bar longer, . no chord

// change is a chord (nil: none) and how many bars it lasts
type change struct {
	c    *harmony.Chord
	bars float64
}

func parseChanges(text string, key string) ([]change, float64, error) {
	scale, err := harmony.Key(key)
	if err != nil {
		return nil, 0, fmt.Errorf("the key is like \"C major\" or \"A minor\" (%v)", err)
	}
	text = strings.NewReplacer("|", " ", "\n", " ", "\t", " ", ",", " ").Replace(text)
	var out []change
	total := 0.0
	for _, tok := range strings.Fields(text) {
		bars, name := 1.0, tok
		if k := strings.LastIndex(tok, "*"); k >= 0 {
			v, err := strconv.ParseFloat(tok[k+1:], 64)
			if err != nil || v <= 0 || v > 256 || math.IsNaN(v) {
				return nil, 0, fmt.Errorf("%s: how many bars the chord lasts (Dm7*2, *0.5)", tok)
			}
			name, bars = tok[:k], v
		}
		switch name {
		case "", "-":
			if len(out) == 0 {
				return nil, 0, fmt.Errorf("a - without a chord before it")
			}
			out[len(out)-1].bars += bars
		case ".":
			out = append(out, change{nil, bars})
		default:
			var c harmony.Chord
			var err error
			if strings.ContainsRune("ABCDEFG", rune(name[0])) {
				c, err = harmony.ParseChord(name)
			} else {
				c, err = scale.Roman(name)
			}
			if err != nil {
				return nil, 0, fmt.Errorf("%s: a chord (Dm7, Bb/D, F#m7b5) or a roman numeral (i, VI, V7)%s", name, didYouMean(name, nil))
			}
			out = append(out, change{&c, bars})
		}
		total += bars
	}
	if len(out) == 0 {
		return nil, 0, fmt.Errorf("no chords: \"Dm7 G7 Cmaj7\"")
	}
	return out, total, nil
}

// rootAbove is the first pitch at or above lowest that has the pitch class
func rootAbove(lowest float64, pc int) float64 {
	return lowest + float64(((pc-int(math.Mod(lowest, 12)))%12+12)%12)
}

func init() {
	add(&Func{Name: "changes", Result: ListOf(ListOf(TNum)), Doc: "chords written as text, as a list of chords: changes(\"Dm7 G7*2 Cmaj7\") or roman numerals in a key: changes(\"i VI V7\", key: \"A minor\"). A chord lasts a bar, or *2 bars (*3 and so on: whole bars), - holds the chord before for another bar, . is no chord (a rest). The chords are in root position from lowest up (48 is C3); lead: true gives each the voicing nearest the one before. Give them to chords: chords(changes(\"Am F C G\"), every(16))",
		Params: []Param{{"text", TStr, nil}, {"key", TStr, defStr("C major")}, {"lowest", TNum, def(48)}, {"lead", TBool, defBool(false)}},
		Call: func(a Args) (Value, error) {
			ch, _, err := parseChanges(a.Str("text"), a.Str("key"))
			if err != nil {
				return Value{}, err
			}
			var out []Value
			var prev []float64
			for _, c := range ch {
				n := int(math.Round(c.bars))
				if math.Abs(c.bars-float64(n)) > 1e-9 || n < 1 {
					return Value{}, fmt.Errorf("a chord of %g bars: as a list of chords a chord lasts a whole number of bars (for half bars, see part)", c.bars)
				}
				var ps []float64
				if c.c != nil {
					if a.value("lead").V.(bool) && prev != nil {
						ps = harmony.VoiceLead(prev, *c.c)
					} else {
						ps = c.c.Pitches(rootAbove(a.Num("lowest"), c.c.Root))
					}
					prev = ps
				}
				for i := 0; i < n; i++ {
					out = append(out, numList(ps))
				}
			}
			return listV(out), nil
		}})
}
