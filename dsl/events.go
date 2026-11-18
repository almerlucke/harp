package dsl

import (
	"fmt"
	"math"
)

// Events play a sequence of notes that are each a synth of their own, as the patterns of SuperCollider do (a Pbind): every event
// has a length (the steps until the next event), a time the gate stays high, and numbers (its parameters). Events go to the voices
// in turn, so that the end of one (its release) goes on while the next one starts: a voice is built once, and plays every
// n-th event.
//
// As SuperCollider does with the messages it sends, an event is put in effect at the start of the block of 64 samples that its
// time lies in. For that the control side works one step ahead: at the start of a step it already knows the event of the next one.

// Event is what the sequence gives for one event
type Event struct {
	Steps float64   // the steps until the next event (a whole number; 0 is an event that starts on the same step as the next: a chord)
	Hold  float64   // the steps the gate stays high
	Vals  []float64 // the numbers of the event
	// Delay is the steps before the very first event (a sequence that starts with rests): only the first event's counts
	Delay float64
	// Shift is how many steps later (or earlier, if it is negative) than its step the event starts: swing and humanizing. The gate is held for
	// Hold from there
	Shift float64
	// Level is how strong the event is (an accent), 1 if there is no accent
	Level float64
	// Span is the steps the event is held for, to work out how long its gate is: Steps, unless Steps is 0 (a note of a chord)
	Span float64
	// Follow is a note of a chord after the first: its numbers are those of the first, except for the note itself
	Follow bool
}

// Groove is what makes the notes of a rhythm swing and what accents them. It is a rule of the step a note starts on (counted from the
// start of the patch), so that notes that were made already can be given a new one when it is edited: Swing delays every other step by
// that many steps (1/3 is a triplet feel), Humanize moves every step by a random amount of at most that many steps (the same amount every
// time it is asked for the same step: it is made from the step and the Seed), and Accent gives the level of the notes by the step, going
// round (nil is a level of 1 everywhere)
type Groove struct {
	Swing, Humanize float64
	Seed            uint64
	Accent          []float64
}

// IsZero says that the groove does nothing
func (g Groove) IsZero() bool { return g.Swing == 0 && g.Humanize == 0 && len(g.Accent) == 0 }

// At is how many steps late a note on this step starts, and how strong it is
func (g Groove) At(step int) (shift, level float64) {
	level = 1
	if step%2 != 0 {
		shift = g.Swing
	}
	if g.Humanize != 0 {
		// a number from 0 to 1 that is always the same for a step and a seed (splitmix64)
		z := g.Seed + uint64(step)*0x9e3779b97f4a7c15
		z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
		z = (z ^ (z >> 27)) * 0x94d049bb133111eb
		z ^= z >> 31
		shift += (float64(z>>11)/float64(1<<53)*2 - 1) * g.Humanize
	}
	if n := len(g.Accent); n > 0 {
		level = g.Accent[((step%n)+n)%n]
	}
	return
}

func (e Event) span() float64 {
	if e.Span > 0 {
		return e.Span
	}
	return e.Steps
}

// EventVoice is the gate and the numbers of one voice: the numbers hold their last value
type EventVoice struct {
	Gate Sig
	Vals []Sig
	// Level is how strong each event is (the accent): only if the groove has an accent, else the zero Sig
	Level Sig
}

// scheduled is an event as the audio side has it: when it starts and ends, in samples
type scheduled struct {
	start, end int64
	at         int64 // the start of the step the event falls on: where a signal that is read for it is read
	vals       []float64
	level      float64
	voice      int
}

type pulled struct {
	offset int // steps from the step it was pulled at
	step   int // the step it starts on, counted from the start of the patch
	ev     Event
}

type eventsState struct {
	rt      *Rt
	next    Gen[Event]
	fp      EventFP     // what the generators are (the fingerprints; "" where it is not known): what changes in a live edit is put in place
	changed []bool      // fields whose generator was put in place and whose values that were made before are to be made again
	retime  bool        // the rhythm is another: the notes that were made but not played are made again from it
	rehold  bool        // how long the gates are held is another: the notes that were made have it
	queued  [][]float64 // for a rhythm that is another: the numbers that were made for the notes that are not played, to be used again, field by field
	nvoice  int
	nvals   int
	count   int // events made so far
	// the control side
	at        int // the step of the next event not made yet
	made      int
	pending   *Event
	zeros     int    // events of 0 steps in a row
	groove    Groove // the swing and the accents
	regroove  bool   // the groove is another: the notes that were made but not played are given it
	lastStart int64  // the sample the last event started at
	pulled    int    // steps pulled
	// the audio side: the cursors of the signals, each of which has its own list of what is to come
	gates  []*gateCursor
	values []*valueCursor
	step   int
}

// Events makes the voices of a sequence of events, each with vals numbers from the events
func Events(next Gen[Event], voices, vals int) []EventVoice {
	return EventsWith(next, voices, make([]Sig, vals))
}

// EventsWith is Events where a number can come from a signal instead of from the events: a field that is a signal (not the zero Sig)
// has, in a voice, the value that signal has at the start of the step of every event (a mono signal; its first channel if it has more).
// A signal is read there, and not at the start of the block of the gate, as the value of its step begins there: the number is later than
// the gate by up to the 63 samples that the gate is early
func EventsWith(next Gen[Event], voices int, fields []Sig) []EventVoice {
	return EventsWithFields(next, voices, fields, EventFP{})
}

// FieldSetter is a generator of events that has a generator for each of its numbers, and lets them be put in place of others
type FieldSetter interface {
	Fields() []Gen[float64]
	SetField(i int, g Gen[float64])
}

// TimingGen is a generator of events that has a rule for how long a gate is held, which can be given by another
type TimingGen interface {
	Hold() func(steps float64) float64
	SetHold(f func(steps float64) float64)
}

// EventFP says what the generators of events are made of, for a live edit (see Fingerprint): the rhythm (when the notes are), how long the
// gates are held, and each field. A patch that takes over from another with the same pattern of notes, differing in some of these, goes on
// with the pattern: a field that is another is put in place and the notes that were made already have its new numbers; how long a gate is
// held is another for the notes that were made; and a rhythm that is another starts at the step where the patch takes over, with the numbers
// that were made for the old notes that are not played yet (the phrases do not skip). The number of voices and of fields is what makes
// the pattern: another one is a new pattern. An empty fingerprint is not known: that part is always put in place
type EventFP struct {
	Timing, Hold string
	Groove       string // the swing and the accents
	Fields       []string
}

// EventsWithFields is EventsWith that is given the fingerprints of what the events are made of; with none (EventFP{}), the events are
// identified by the fingerprint of next, as a whole
func EventsWithFields(next Gen[Event], voices int, fields []Sig, fp EventFP) []EventVoice {
	return EventsGrooved(next, voices, fields, fp, Groove{})
}

// EventsGrooved is EventsWithFields with a groove: swing, humanizing and accents of the notes by their step
func EventsGrooved(next Gen[Event], voices int, fields []Sig, fp EventFP, groove Groove) []EventVoice {
	vals := len(fields)
	if voices < 1 {
		panic("dsl: Events: at least one voice")
	}
	st := &eventsState{next: next, nvoice: voices, nvals: vals, changed: make([]bool, vals), fp: fp, groove: groove}
	st.fp.Fields = make([]string, vals)
	copy(st.fp.Fields, fp.Fields)
	gens := Fingerprint(next)
	if fp.Timing != "" || fp.Hold != "" || fp.Groove != "" {
		gens = "events" // the nodes are the same pattern whatever it is made of: that is looked at when it takes over (RebaseFrom)
	}
	id := fmt.Sprintf("events %d %d %v", voices, vals, len(groove.Accent) > 0)
	out := make([]EventVoice, voices)
	for v := range out {
		v := v
		if len(groove.Accent) > 0 {
			out[v].Level = mono(&node{name: "events.level", id: id, lane: true, gens: gens, group: st, build: func(rt *Rt, _ [][]float64, o []float64) Kernel {
				st.attach(rt)
				c := &valueCursor{st: st, voice: v, idx: levelIdx, cur: 1}
				st.values = append(st.values, c)
				return func(n int) { c.fill(o, n) }
			}})
		}
		out[v].Gate = mono(&node{name: "events.gate", id: id, lane: true, gens: gens, group: st, build: func(rt *Rt, _ [][]float64, o []float64) Kernel {
			st.attach(rt)
			c := &gateCursor{st: st, voice: v}
			st.gates = append(st.gates, c)
			return func(n int) { c.fill(o, n) }
		}})
		out[v].Vals = make([]Sig, vals)
		for i := 0; i < vals; i++ {
			i := i
			var in []*node
			if len(fields[i].ch) > 0 {
				in = []*node{fields[i].ch[0]}
			}
			out[v].Vals[i] = mono(&node{name: "events.val", id: id, lane: true, gens: gens, group: st, in: in, build: func(rt *Rt, b [][]float64, o []float64) Kernel {
				st.attach(rt)
				c := &valueCursor{st: st, voice: v, idx: i}
				if len(b) > 0 {
					c.src = b[0]
				}
				st.values = append(st.values, c)
				return func(n int) { c.fill(o, n) }
			}})
		}
	}
	return out
}

func (st *eventsState) attach(rt *Rt) {
	if st.rt == rt {
		return
	}
	st.rt = rt
	st.gates, st.values = nil, nil
	st.at, st.pulled, st.count, st.step, st.made, st.pending, st.lastStart = 0, 0, 0, 0, 0, nil, 0
	laneWith(rt, st.pull, st.apply, st.refresh, st.finishRefresh, st.prepare)
}

// RebaseFrom: the group can take over from old if it is the same pattern of notes, made of other generators that can be put in place of
// the old ones (see EventFP)
func (st *eventsState) RebaseFrom(old any) (bool, func()) {
	o, ok := old.(*eventsState)
	if !ok || o.nvals != st.nvals {
		return false, nil
	}
	known := func(a, b string) bool { return a != "" && a == b }
	if (len(st.groove.Accent) > 0) != (len(o.groove.Accent) > 0) {
		return false, nil // an accent that is there or not is another set of nodes
	}
	sameTiming, sameHold, sameGroove := known(st.fp.Timing, o.fp.Timing), known(st.fp.Hold, o.fp.Hold), known(st.fp.Groove, o.fp.Groove)
	sameFields := true
	for i := range st.fp.Fields {
		if !known(st.fp.Fields[i], o.fp.Fields[i]) {
			sameFields = false
		}
	}
	if sameTiming && sameHold && sameFields && sameGroove {
		return true, nil
	}
	oldSet, ok1 := o.next.(FieldSetter)
	newSet, ok2 := st.next.(FieldSetter)
	oldTime, ok3 := o.next.(TimingGen)
	newTime, ok4 := st.next.(TimingGen)
	if !ok1 || !ok2 || !ok3 || !ok4 || len(oldSet.Fields()) != st.nvals || len(newSet.Fields()) != st.nvals {
		return false, nil
	}
	return true, func() {
		newFields := newSet.Fields()
		oldFields := oldSet.Fields()
		if !sameTiming {
			// another rhythm: the new generator of events takes the place of the old one, with the old generators of the fields that are
			// the same (their phrases go on); the numbers that were made for the notes that are not played are queued for them (prepare)
			for i := range newFields {
				if known(st.fp.Fields[i], o.fp.Fields[i]) {
					newSet.SetField(i, oldFields[i])
				} else {
					o.changed[i] = false // its notes are made again with the new rhythm, by the new generator
				}
			}
			o.next = st.next
			o.fp = st.fp
			o.retime = true
			return
		}
		for i := range newFields {
			if known(st.fp.Fields[i], o.fp.Fields[i]) || newFields[i] == nil { // the same, or a field that is a signal: nothing to put in place
				continue
			}
			oldSet.SetField(i, newFields[i])
			o.fp.Fields[i] = st.fp.Fields[i]
			o.changed[i] = true
		}
		if !sameHold {
			oldTime.SetHold(newTime.Hold())
			o.fp.Hold = st.fp.Hold
			o.rehold = true
		}
		if !sameGroove {
			o.groove = st.groove
			o.fp.Groove = st.fp.Groove
			o.regroove = true
		}
	}
}

// prepare is told, before the notes that were made and not played are made right, what they are: for a rhythm that is another the numbers of
// the fields that go on are queued to be used again, the new rhythm starts at the step after the first of them, and the notes are made again
// one record at a time (refresh)
func (st *eventsState) prepare(first int64, vals [][]pulled) {
	if !st.retime {
		return
	}
	set, ok := st.next.(FieldSetter)
	if !ok {
		return
	}
	fields := set.Fields()
	st.queued = make([][]float64, len(fields))
	take := func(vs []float64) {
		for i := range st.queued {
			if i < len(vs) {
				st.queued[i] = append(st.queued[i], vs[i])
			}
		}
	}
	for _, rec := range vals {
		for _, p := range rec {
			if !p.ev.Follow {
				take(p.ev.Vals)
			}
		}
	}
	if st.pending != nil && !st.pending.Follow {
		take(st.pending.Vals)
	}
	for i, f := range fields {
		if f != nil && !isChordField(f) && len(st.queued[i]) > 0 {
			set.SetField(i, &queuedGen{queue: &st.queued[i], g: f})
		}
	}
	st.pending = nil
	st.pulled = int(first)
	st.at = int(first) + 1
	st.made = 0 // the rhythm starts at its start, with its first rests
}

// refresh makes the notes that were made before a change right: where a field changed, their numbers are drawn again, from its new
// generator; where how long the gate is held changed, they have the new rule; and with a rhythm that is another they are made again
func (st *eventsState) refresh(ps []pulled) []pulled {
	if st.retime {
		return st.pull()
	}
	set, ok := st.next.(FieldSetter)
	if !ok {
		return ps
	}
	fs := set.Fields()
	out := make([]pulled, len(ps))
	for k, p := range ps {
		out[k] = p
		out[k].ev.Vals = st.redraw(p.ev.Vals, fs)
		if st.rehold {
			out[k].ev.Hold = st.holdOf(p.ev.span())
		}
		if st.regroove {
			out[k].ev.Shift, out[k].ev.Level = st.groove.At(p.step)
		}
	}
	return out
}

func (st *eventsState) holdOf(steps float64) float64 {
	if t, ok := st.next.(TimingGen); ok {
		return t.Hold()(steps)
	}
	return steps
}

func (st *eventsState) redraw(vals []float64, fs []Gen[float64]) []float64 {
	var out []float64
	for i, c := range st.changed {
		if !c || i >= len(vals) || fs[i] == nil || isChordField(fs[i]) {
			continue // (the notes of a chord cannot be made again one at a time: the ones that were made keep their chord)
		}
		if out == nil {
			out = append([]float64(nil), vals...)
		}
		out[i] = fs[i].Generate()
	}
	if out == nil {
		return vals
	}
	return out
}

// finishRefresh makes the note that was made and not yet handed to the audio right too, and is done with the change
func (st *eventsState) finishRefresh() {
	if set, ok := st.next.(FieldSetter); ok && st.pending != nil && !st.retime {
		st.pending.Vals = st.redraw(st.pending.Vals, set.Fields())
		if st.rehold {
			st.pending.Hold = st.holdOf(st.pending.span())
		}
	}
	for i := range st.changed {
		st.changed[i] = false
	}
	st.retime, st.rehold, st.regroove = false, false, false
}

// queuedGen gives the numbers that were made for notes that are no more, before it goes on with its generator
type queuedGen struct {
	queue *[]float64
	g     Gen[float64]
}

func (q *queuedGen) Generate() float64 {
	if len(*q.queue) > 0 {
		v := (*q.queue)[0]
		*q.queue = (*q.queue)[1:]
		return v
	}
	return q.g.Generate()
}
func (q *queuedGen) Continuous() bool { return q.g.Continuous() }
func (q *queuedGen) Done() bool       { return q.g.Done() }
func (q *queuedGen) Reset()           { q.g.Reset() }

// pull is the control side of a step: the events that start at the next step (and at the first, those that start at it too)
func (st *eventsState) pull() []pulled {
	k := st.pulled
	st.pulled++
	var out []pulled
	for {
		if st.pending == nil {
			ev := st.next.Generate()
			if ev.Steps < 0 || ev.Steps != math.Trunc(ev.Steps) {
				ev.Steps = 1
			}
			// an event of 0 steps starts on the same step as the next one: a chord. A generator that never gets on to the next step is not waited for
			if ev.Steps == 0 {
				if st.zeros++; st.zeros > maxChord {
					ev.Steps = 1
				}
			} else {
				st.zeros = 0
			}
			if st.made == 0 {
				st.at += int(ev.Delay)
			}
			st.made++
			st.pending = &ev
		}
		if st.at > k+1 {
			break
		}
		ev := *st.pending
		ev.Shift, ev.Level = st.groove.At(st.at)
		out = append(out, pulled{st.at - k, st.at, ev})
		st.at += int(st.pending.Steps)
		st.pending = nil
	}
	return out
}

func (st *eventsState) apply(ps []pulled) {
	k := st.step
	st.step++
	for _, p := range ps {
		step := float64(k + p.offset)
		start := int64(math.Ceil((step + p.ev.Shift) * st.rt.StepSamples))
		start = max(start, st.lastStart) // a late note does not make the next one come before it
		st.lastStart = start
		end := start + int64(math.Ceil(p.ev.Hold*st.rt.StepSamples))
		v := st.count % st.nvoice
		if st.count == 0 {
			// before its first note a voice has the numbers of the first note: a filter that is given a frequency of 0 would blow up
			for _, c := range st.values {
				if c.idx == levelIdx {
					c.cur = p.ev.Level
				} else if c.src == nil && c.idx < len(p.ev.Vals) {
					c.cur = p.ev.Vals[c.idx]
				}
			}
		}
		st.count++
		// at the start of the block the time is in, and not before now
		q := func(s int64) int64 { return max(s-s%scBlock, st.rt.pos) }
		e := scheduled{start: q(start), end: q(end), at: max(int64(math.Ceil(step*st.rt.StepSamples)), st.rt.pos), vals: p.ev.Vals, level: p.ev.Level, voice: v}
		for _, c := range st.gates {
			if c.voice == v {
				c.pending.push(e)
			}
		}
		for _, c := range st.values {
			if c.voice == v {
				c.pending.push(e)
			}
		}
	}
}

// queue is what is to come, in order: it keeps its room when it is empty
type queue struct {
	items []scheduled
	head  int
}

func (q *queue) push(e scheduled) { q.items = append(q.items, e) }
func (q *queue) ready(s int64) bool {
	return q.head < len(q.items) && q.items[q.head].start <= s
}
func (q *queue) pop() scheduled {
	e := q.items[q.head]
	q.head++
	if q.head == len(q.items) {
		q.items, q.head = q.items[:0], 0
	}
	return e
}

// the gate: high from the start of an event to its end (a new event while it is high first goes low for a block, so that it is a new trigger)
type gateCursor struct {
	st      *eventsState
	voice   int
	pending queue
	until   int64 // the gate is high up to this sample
}

func (c *gateCursor) fill(o []float64, n int) {
	pos := c.st.rt.pos
	for i := 0; i < n; i++ {
		s := pos + int64(i)
		for c.pending.ready(s) {
			c.until = max(c.until, c.pending.pop().end)
		}
		if s < c.until {
			o[i] = 1
		} else {
			o[i] = 0
		}
	}
}

// levelIdx is the index of the cursor of the accent: not one of the numbers of the events
const levelIdx = -1

// a number of the events: it changes where an event of the voice starts
type valueCursor struct {
	st      *eventsState
	voice   int
	idx     int
	pending queue
	cur     float64
	src     []float64 // the signal that the number is read from, if it is one
	started bool
}

func (c *valueCursor) fill(o []float64, n int) {
	pos := c.st.rt.pos
	for i := 0; i < n; i++ {
		s := pos + int64(i)
		if c.src != nil {
			if !c.started { // before its first note a voice has the value the signal has to begin with
				c.cur, c.started = c.src[i], true
			}
			// a signal is read at the start of the step of the event
			for c.pending.head < len(c.pending.items) && c.pending.items[c.pending.head].at <= s {
				c.cur = c.src[i]
				c.pending.pop()
			}
		} else {
			for c.pending.ready(s) {
				e := c.pending.pop()
				if c.idx == levelIdx {
					c.cur = e.level
				} else if c.idx < len(e.vals) {
					c.cur = e.vals[c.idx]
				}
			}
		}
		o[i] = c.cur
	}
}

type eventGen struct {
	dur    Gen[float64]
	hold   func(steps float64) float64
	fields []Gen[float64]
}

// EventGen makes events: the length of each (in steps) from dur, how long its gate stays high from hold(length), and one number for each field from its generator
func EventGen(dur Gen[float64], hold func(steps float64) float64, fields []Gen[float64]) Gen[Event] {
	return &eventGen{dur: dur, hold: hold, fields: fields}
}

func (g *eventGen) Generate() Event {
	steps := g.dur.Generate()
	ev := Event{Steps: steps, Hold: g.hold(steps), Vals: make([]float64, len(g.fields))}
	for i, f := range g.fields {
		if f != nil {
			ev.Vals[i] = f.Generate()
		}
	}
	return ev
}
func (g *eventGen) Fields() []Gen[float64]          { return g.fields }
func (g *eventGen) SetField(i int, f Gen[float64])  { g.fields[i] = f }
func (g *eventGen) Hold() func(float64) float64     { return g.hold }
func (g *eventGen) SetHold(f func(float64) float64) { g.hold = f }
func (g *eventGen) Continuous() bool                { return true }
func (g *eventGen) Done() bool                      { return false }
func (g *eventGen) Reset()                          {}

type rhythmGen struct {
	rhythm Gen[bool]
	hold   func(steps float64) float64
	fields []Gen[float64]
	first  bool
}

// RhythmEvents makes events from a rhythm, a generator of booleans that is pulled every step: an event starts on every true, and is as long
// as the steps until the next true (its gate stays high for hold of that); the numbers of every field are pulled when an event starts. A
// rhythm that starts with rests has its first event after them
func RhythmEvents(rhythm Gen[bool], hold func(steps float64) float64, fields []Gen[float64]) Gen[Event] {
	return &rhythmGen{rhythm: rhythm, hold: hold, fields: fields}
}

const maxRests = 100000

// maxChord is the most notes that start on one step
const maxChord = 64

func (g *rhythmGen) Generate() Event {
	var delay float64
	if !g.first {
		g.first = true
		for ; !g.rhythm.Generate() && delay < maxRests; delay++ {
		}
	}
	// at a true: count the steps up to the next
	steps := 1.0
	for steps < maxRests && !g.rhythm.Generate() {
		steps++
	}
	ev := Event{Steps: steps, Hold: g.hold(steps), Delay: delay, Vals: make([]float64, len(g.fields))}
	for i, f := range g.fields {
		if f != nil {
			ev.Vals[i] = f.Generate()
		}
	}
	return ev
}
func (g *rhythmGen) Fields() []Gen[float64]          { return g.fields }
func (g *rhythmGen) SetField(i int, f Gen[float64])  { g.fields[i] = f }
func (g *rhythmGen) Hold() func(float64) float64     { return g.hold }
func (g *rhythmGen) SetHold(f func(float64) float64) { g.hold = f }
func (g *rhythmGen) Continuous() bool                { return true }
func (g *rhythmGen) Done() bool                      { return false }
func (g *rhythmGen) Reset()                          {}

// ---- chords: a field that is a chord

// chordField is a field of the events that gives a chord: the first note is the number it gives, and the others are what ChordEvents makes more
// events of, on the same step
type chordField struct {
	g    Gen[[]float64]
	last []float64
}

// ChordField is a field that is a generator of chords: every event that the generator of events makes is made as many events as the chord has
// notes (see ChordEvents), each with one of the notes in this field
func ChordField(g Gen[[]float64]) Gen[float64] { return &chordField{g: g} }

func (c *chordField) Generate() float64 {
	c.last = c.g.Generate()
	if len(c.last) == 0 {
		return 0
	}
	return c.last[0]
}
func (c *chordField) Continuous() bool { return c.g.Continuous() }
func (c *chordField) Done() bool       { return c.g.Done() }
func (c *chordField) Reset()           { c.g.Reset() }

func isChordField(g Gen[float64]) bool {
	_, ok := g.(*chordField)
	return ok
}

type chordEvents struct {
	base  Gen[Event]
	queue []Event
}

// ChordEvents makes the events of a generator of events with a chord in a field (ChordField) a chord: one event for each note of it, all on the
// step that the event is on, and the next after the steps that the event was long. The other numbers are the same for all the notes. A chord of
// no notes is a rest, and more notes than 64 are left out. The voices of the events are used in turn, so that there have to be as many of them as
// the notes of the biggest chord, or the notes of a chord take the voices of the first ones
func ChordEvents(base Gen[Event]) Gen[Event] { return &chordEvents{base: base} }

func (g *chordEvents) field() (int, *chordField) {
	if set, ok := g.base.(FieldSetter); ok {
		for i, f := range set.Fields() {
			if c, ok := f.(*chordField); ok {
				return i, c
			}
		}
	}
	return -1, nil
}

func (g *chordEvents) Generate() Event {
	if len(g.queue) > 0 {
		ev := g.queue[0]
		g.queue = g.queue[1:]
		return ev
	}
	ev := g.base.Generate()
	i, c := g.field()
	if c == nil {
		return ev
	}
	if len(c.last) == 0 {
		ev.Hold = 0 // a rest
		return ev
	}
	notes := c.last[:min(len(c.last), maxChord)]
	if len(notes) == 1 {
		return ev
	}
	span := ev.Steps
	for k, n := range notes {
		e := ev
		e.Vals = append([]float64(nil), ev.Vals...)
		e.Vals[i] = n
		e.Span = span
		if k > 0 {
			e.Delay, e.Follow = 0, true
		}
		if k < len(notes)-1 {
			e.Steps = 0
		}
		g.queue = append(g.queue, e)
	}
	first := g.queue[0]
	g.queue = g.queue[1:]
	return first
}

func (g *chordEvents) Fields() []Gen[float64] {
	if set, ok := g.base.(FieldSetter); ok {
		return set.Fields()
	}
	return nil
}
func (g *chordEvents) SetField(i int, f Gen[float64]) {
	if set, ok := g.base.(FieldSetter); ok {
		set.SetField(i, f)
	}
}
func (g *chordEvents) Hold() func(float64) float64 {
	if t, ok := g.base.(TimingGen); ok {
		return t.Hold()
	}
	return nil
}
func (g *chordEvents) SetHold(f func(float64) float64) {
	if t, ok := g.base.(TimingGen); ok {
		t.SetHold(f)
	}
}
func (g *chordEvents) Continuous() bool { return true }
func (g *chordEvents) Done() bool       { return false }
func (g *chordEvents) Reset()           { g.queue = nil }
