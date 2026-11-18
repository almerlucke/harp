package lang

import (
	"fmt"
	"hash/fnv"
	"math"
	"slices"

	"harp/blocks"
	"harp/dsl"

	"github.com/almerlucke/genny/note"
)

// A score is a list of notes that have a start and a length, in steps: what a MIDI file is, and what a fugue, a canon or a phasing piece make.
// It is the other way of making music than a generator, which gives a value at every step: here the whole piece is made when the patch is built.
// A score plays on any synth (it is made into a generator of notes that goes round), and can be taken apart into a rhythm and phrases for play.

// Score is notes in steps, and how long the piece is (the steps after which it goes round)
type Score struct {
	Events []note.Event
	Len    float64 // 0: the end of the last note, to a whole step
}

func scoreV(events []note.Event, length float64) Value {
	slices.SortStableFunc(events, func(a, b note.Event) int {
		switch {
		case a.Start < b.Start:
			return -1
		case a.Start > b.Start:
			return 1
		}
		return 0
	})
	return Value{TScore, &Score{Events: events, Len: length}}
}

// span is how many steps the score goes round in
func (s *Score) span() int {
	end := s.Len
	for _, e := range s.Events {
		end = math.Max(end, e.Start+e.Duration)
	}
	return max(int(math.Ceil(end-1e-9)), 1)
}

// step is the step a note starts on (the nearest)
func stepOf(e note.Event) int { return max(int(math.Round(e.Start)), 0) }

// table is the notes that start on each step of the score
func (s *Score) table() [][]blocks.Note {
	n := s.span()
	t := make([][]blocks.Note, n)
	for _, e := range s.Events {
		k := stepOf(e) % n
		v := e.Velocity
		if v <= 0 {
			v = 0.8
		}
		t[k] = append(t[k], blocks.Note{Pitch: e.Pitch, Velocity: v, Length: math.Max(e.Duration, 0.01)})
	}
	return t
}

// gen is the score as notes for a synth: the notes of each step, round and round
func (s *Score) gen() dsl.Gen[[]blocks.Note] { return &scoreGen{t: s.table()} }

type scoreGen struct {
	t    [][]blocks.Note
	step int
}

func (g *scoreGen) Generate() []blocks.Note {
	n := g.t[g.step]
	g.step = (g.step + 1) % len(g.t)
	return slices.Clone(n)
}
func (g *scoreGen) Continuous() bool { return true }
func (g *scoreGen) Done() bool       { return false }
func (g *scoreGen) Reset()           { g.step = 0 }

// fingerprint says what the score is: the same notes are the same score
func (s *Score) fingerprint() string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%g|", s.Len)
	for _, e := range s.Events {
		fmt.Fprintf(h, "%g,%g,%g,%g;", e.Pitch, e.Velocity, e.Start, e.Duration)
	}
	return fmt.Sprintf("score(%d:%x)", len(s.Events), h.Sum64())
}

// oneVoice is the notes of the score one at a time, in order, with one note to a step (of those that start together, the highest): for the
// rhythm and the fields of play
func (s *Score) oneVoice() []note.Event {
	var out []note.Event
	for _, e := range s.Events {
		if n := len(out); n > 0 && stepOf(out[n-1]) == stepOf(e) {
			if e.Pitch > out[n-1].Pitch {
				out[n-1] = e
			}
			continue
		}
		out = append(out, e)
	}
	return out
}

// safely turns the panic of a genny function that was given what it cannot take into an error
func safely(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	f()
	return nil
}

func scoreOf(a Args, name string) *Score { return a.value(name).V.(*Score) }
