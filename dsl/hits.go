package dsl

// Hit is a note of a rhythm that can fall anywhere in a step, not only on its start: a roll is several hits in a step, a triplet three
type Hit struct {
	Offset float64 // 0 up to 1: how far into the step it starts
	Len    float64 // the steps it lasts; 0 is until the next hit
}

// boolHits makes the hits of a rhythm of booleans: a hit on the start of every step that is true
type boolHits struct{ g Gen[bool] }

// HitsOf is a rhythm of booleans as hits (on the start of the steps that are true)
func HitsOf(g Gen[bool]) Gen[[]Hit] { return &boolHits{g} }

func (b *boolHits) Generate() []Hit {
	if b.g.Generate() {
		return []Hit{{}}
	}
	return nil
}
func (b *boolHits) Continuous() bool    { return b.g.Continuous() }
func (b *boolHits) Done() bool          { return b.g.Done() }
func (b *boolHits) Reset()              { b.g.Reset() }
func (b *boolHits) Fingerprint() string { return Fingerprint(b.g) }

// absHit is a hit with the step it is in
type absHit struct {
	step int
	Hit
}

type hitEvents struct {
	hits   Gen[[]Hit]
	hold   func(steps float64) float64
	fields []Gen[float64]
	step   int // the next step to ask the hits of
	queue  []absHit
	first  bool
}

// HitEvents makes events from hits, a generator that is pulled every step and gives the hits that start in it: an event for each, as long as the
// time to the next hit (or its Len), starting its offset into its step (Event.Shift). Hits in one step are events of 0 steps apart; the numbers of
// every field are pulled for each hit. A rhythm that starts with rests has its first event after them
func HitEvents(hits Gen[[]Hit], hold func(steps float64) float64, fields []Gen[float64]) Gen[Event] {
	return &hitEvents{hits: hits, hold: hold, fields: fields}
}

// fill asks the next steps for their hits until there are n in the queue; it gives false when there have been none for a very long time
func (g *hitEvents) fill(n int) bool {
	empty := 0
	for len(g.queue) < n {
		hs := g.hits.Generate()
		for _, h := range hs {
			g.queue = append(g.queue, absHit{g.step, h})
		}
		g.step++
		if len(hs) == 0 {
			if empty++; empty > maxRests {
				return false
			}
		} else {
			empty = 0
		}
	}
	return true
}

func (g *hitEvents) Generate() Event {
	var delay float64
	if !g.first {
		g.first = true
		if !g.fill(1) {
			g.queue = append(g.queue, absHit{step: g.step + maxRests})
		}
		delay = float64(g.queue[0].step)
	}
	if len(g.queue) == 0 && !g.fill(1) {
		g.queue = append(g.queue, absHit{step: g.step + maxRests})
	}
	cur := g.queue[0]
	next := absHit{step: cur.step + maxRests}
	if g.fill(2) {
		next = g.queue[1]
	}
	g.queue = g.queue[1:]
	steps := float64(next.step - cur.step)
	span := float64(next.step) + next.Offset - float64(cur.step) - cur.Offset
	if cur.Len > 0 {
		span = cur.Len
	}
	ev := Event{Steps: steps, Hold: g.hold(span), Span: span, Shift: cur.Offset, Delay: delay, Vals: make([]float64, len(g.fields))}
	for i, f := range g.fields {
		if f != nil {
			ev.Vals[i] = f.Generate()
		}
	}
	return ev
}

func (g *hitEvents) Fields() []Gen[float64]          { return g.fields }
func (g *hitEvents) SetField(i int, f Gen[float64])  { g.fields[i] = f }
func (g *hitEvents) Hold() func(float64) float64     { return g.hold }
func (g *hitEvents) SetHold(f func(float64) float64) { g.hold = f }
func (g *hitEvents) Continuous() bool                { return true }
func (g *hitEvents) Done() bool                      { return false }
func (g *hitEvents) Reset()                          {}
