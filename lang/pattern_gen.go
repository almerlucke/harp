package lang

import (
	"fmt"
	"math"
	"sort"

	"harp/dsl"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/psalm/tools/rhythm"
)

// patternGen plays the text of a pattern, pass after pass: a generator that is pulled once for every step and gives the hits that start in it. A pass
// has its own functions (E(5,16,+1) turn every pass), its mutations, the hits that its conditions and chances let through
type patternGen struct {
	src       string
	live      bool // the text has functions that change with every pass
	mutate    float64
	seed      uint64
	freeze    int // the pass after which it stops changing (0: never)
	drift     float64
	song      *Song
	pass      int64
	passStart float64
	steps     float64
	hits      []dsl.Hit // of the pass, with their start in the pattern in Offset (steps): an offset here is not within a step
	step      int
}

func newPatternGen(src string, mutate float64, seed uint64, freeze int, drift float64, song *Song) (*patternGen, error) {
	g := &patternGen{src: src, live: rhythm.IsLive(src), mutate: mutate, seed: seed, freeze: freeze, drift: drift, song: song}
	if err := g.build(0); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *patternGen) build(pass int64) error {
	text := g.src
	if g.live {
		var err error
		if text, err = rhythm.ExpandPass(g.src, pass); err != nil {
			return err
		}
	} else if t, err := rhythm.Expand(g.src); err != nil {
		return err
	} else {
		text = t
	}
	hits, steps, err := parseTextPattern(text, false)
	if err != nil {
		return err
	}
	g.steps, g.pass = steps, pass
	hits = g.mutated(hits, steps, pass)
	g.hits = g.hits[:0]
	for _, h := range hits {
		at := g.passStart + h.at
		if !h.cond.Zero() && !h.cond.Holds(pass, g.song != nil && g.song.inFill(at)) {
			continue
		}
		if h.chance < 1 && genny.Float64() >= h.chance {
			continue
		}
		off := h.at
		if g.drift > 0 {
			off += g.drift * g.wander(at)
		}
		g.hits = append(g.hits, dsl.Hit{Offset: off, Len: h.len})
	}
	return nil
}

// mutated is the hits of a pass that changes: some hits that are not kept (!) fall out, some rests become hits, as many as a quarter of the steps at
// most (at mutate 1). The same seed makes the same changes; after the freeze pass it makes no more
func (g *patternGen) mutated(hits []textHit, steps float64, pass int64) []textHit {
	if g.mutate <= 0 {
		return hits
	}
	if g.freeze > 0 && pass > int64(g.freeze) {
		pass = int64(g.freeze)
	}
	p := g.mutate / 4
	var out []textHit
	has := map[int]bool{}
	for _, h := range hits {
		s := int(math.Floor(h.at))
		has[s] = true
		if !h.pin && dsl.Unit(g.seed, int(pass)*1000+s, 3) < p {
			continue
		}
		out = append(out, h)
	}
	for s := 0; s < int(steps); s++ {
		if !has[s] && dsl.Unit(g.seed, int(pass)*1000+s, 4) < p {
			out = append(out, textHit{at: float64(s), chance: 1})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].at < out[j].at })
	return out
}

// wander is a smooth curve from 0 to 1 of the time in steps (not jitter: it moves slowly and the same way every time)
func (g *patternGen) wander(t float64) float64 {
	a, b := float64(g.seed%628)/100, float64(g.seed%377)/60
	return (math.Sin(t*0.37+a) + math.Sin(t*0.91+b) + 2) / 4
}

func (g *patternGen) Generate() []dsl.Hit {
	k := float64(g.step)
	g.step++
	var out []dsl.Hit
	for {
		for _, h := range g.hits {
			if t := g.passStart + h.Offset; t >= k && t < k+1 {
				out = append(out, dsl.Hit{Offset: t - k, Len: h.Len})
			}
		}
		if g.passStart+g.steps >= k+1 {
			break
		}
		g.passStart += g.steps
		if err := g.build(g.pass + 1); err != nil {
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Offset < out[j].Offset })
	return out
}

func (g *patternGen) Continuous() bool { return true }
func (g *patternGen) Done() bool       { return false }
func (g *patternGen) Reset() {
	g.step, g.passStart = 0, 0
	g.build(0)
}

// inFill says that a step is in the last bar of its section
func (s *Song) inFill(step float64) bool {
	i, p := s.at(step)
	return (1-p)*s.Len[i] <= float64(s.BarSteps)+1e-9
}

// plain says that the text of a pattern is steps and rests, and nothing else: then it is a rhythm of booleans
func plainPattern(hits []textHit, steps float64) ([]bool, bool) {
	if steps != math.Trunc(steps) {
		return nil, false
	}
	out := make([]bool, int(steps))
	for _, h := range hits {
		s := int(h.at)
		if float64(s) != h.at || h.len != 0 || h.chance != 1 || !h.cond.Zero() || out[s] {
			return nil, false
		}
		out[s] = true
	}
	return out, true
}

func init() {
	add(&Func{Name: "pattern", Result: TAny, Doc: "a rhythm written as text, as the clock of psalm reads it: x is a hit and . a rest, a step each (\"x..x ..x. x...\", spaces and | are for reading). x3 is a roll, three hits in a step; [x x x] a group that fills one step (a triplet; [x . x x] with rests); *2 after a step or a group: it lasts 2 steps; ?25 after a hit: 25% chance (? is 50%); %2/4 only the second time through in every four (%!2/4 the other three), %first only the first time, %fill only in the last bar of a section of song; ! after a hit: it stays when the pattern mutates. The functions E(5,16) (euclid), A(3+3+2) (additive), CA(30,16) (a cellular automaton) and M(x . x x;16;3) (a Markov chain on an example) give their steps, and +1 in them (E(5,16,+1)) goes on every pass: the rhythm turns round. mutate (0 to 1) changes the pattern from pass to pass (hits fall out, rests become hits: as many as a quarter of the steps at 1) with seed, freeze is the pass after which it changes no more, drift (0 to 0.3) moves the timing along a smooth curve. Plain steps and rests give a rhythm of booleans; with more it gives hits, that melody, chords and play take as their rhythm",
		Params: []Param{{"steps", TStr, nil}, {"mutate", TNum, def(0)}, {"seed", TNum, def(0)}, {"freeze", TNum, def(0)}, {"drift", TNum, def(0)}, {"song", TAny, def(0)}},
		Call: func(a Args) (Value, error) {
			mut, drift := a.Num("mutate"), a.Num("drift")
			if mut < 0 || mut > 1 || drift < 0 || drift > 0.3 || a.Int("seed") < 0 || a.Int("freeze") < 0 {
				return Value{}, fmt.Errorf("mutate is from 0 to 1, drift from 0 to 0.3, seed and freeze 0 or more")
			}
			var song *Song
			if sv := a.value("song"); sv.T.Kind == KSong {
				song = sv.V.(*Song)
			} else if sv.T.Kind != KNum {
				return Value{}, fmt.Errorf("song is a song(...)")
			}
			g, err := newPatternGen(a.Str("steps"), mut, uint64(a.Int("seed")), a.Int("freeze"), drift, song)
			if err != nil {
				return Value{}, err
			}
			if !g.live && mut == 0 && drift == 0 {
				text, _ := rhythm.Expand(a.Str("steps"))
				hits, steps, _ := parseTextPattern(text, false)
				if bools, ok := plainPattern(hits, steps); ok {
					return Value{TGenBool, dsl.Loop(bools...)}, nil
				}
			}
			return Value{TGenHits, g}, nil
		}})
}
