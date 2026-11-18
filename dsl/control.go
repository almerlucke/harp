package dsl

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Generators never run on the audio thread. A node that wants a value from a generator registers
// a lane: pull runs on the control side, apply on the audio side. At every step the control side
// pulls all lanes (in the order they were registered) into a stepRec, and the audio side applies
// the record of that step on the sample the step starts. A record holds plain values, so apply
// is a few assignments: it doesn't allocate, lock or call a generator.

type lane struct {
	pull  func() any
	apply func(any)
	id    *byte // what the lane is: it keeps it when a program takes over from another, and the records are translated by it (see swap.go)
	// when a program takes over and keeps the lane, but changed something in the generators behind it (a field of a pattern of notes): refresh
	// makes the values of a record that was made before it right again (it is called for every record that is not played yet, in order), and
	// finish is called after the last of them. Both run on the control goroutine
	refresh func(any) any
	finish  func()
	// prepare is called before the first of them, with the step of the first record and the values that all of them have for this lane (the
	// lane may make them again, in order, in refresh, from what it knows of them)
	prepare func(first int64, vals []any)
}

// Lane registers a pull on the control side and an apply on the audio side. The type T is what
// goes from one to the other
func Lane[T any](rt *Rt, pull func() T, apply func(T)) { laneWith(rt, pull, apply, nil, nil, nil) }

// LaneWith is Lane for a lane whose values can be made again when a program takes over from another that has a new generator behind the lane
// (see swap.go): refresh makes the values of a record that was made before it (called for each, in order, on the control goroutine), finish is
// called after the last. Either can be nil
func LaneWith[T any](rt *Rt, pull func() T, apply func(T), refresh func(T) T, finish func()) {
	laneWith(rt, pull, apply, refresh, finish, nil)
}

// LanePrepared is LaneWith for a lane that also wants to know where the records that are made again begin: prepare is called before the first
// refresh with the step of the first of them and the values the records have (how many steps of the lane are not played yet is len(vals))
func LanePrepared[T any](rt *Rt, pull func() T, apply func(T), refresh func(T) T, finish func(), prepare func(first int64, vals []T)) {
	laneWith(rt, pull, apply, refresh, finish, prepare)
}

func laneWith[T any](rt *Rt, pull func() T, apply func(T), refresh func(T) T, finish func(), prepare func(first int64, vals []T)) {
	l := lane{
		pull:  func() any { return pull() },
		apply: func(v any) { apply(v.(T)) },
		id:    new(byte),
	}
	if refresh != nil {
		l.refresh = func(v any) any { return refresh(v.(T)) }
	}
	l.finish = finish
	if prepare != nil {
		l.prepare = func(first int64, vals []any) {
			ts := make([]T, len(vals))
			for i, v := range vals {
				ts[i], _ = v.(T)
			}
			prepare(first, ts)
		}
	}
	if rt.sink != nil {
		*rt.sink = append(*rt.sink, l)
		return
	}
	rt.lanes = append(rt.lanes, l)
}

// stepRec is what the control side made for one step
type stepRec struct {
	idx  int64 // the step it is for, counting from 0
	vals []any // one per lane
}

// controlSide makes the records, in step order
type controlSide struct {
	lanes []lane
	next  int64
}

func (c *controlSide) produce() *stepRec {
	rec := &stepRec{idx: c.next, vals: make([]any, len(c.lanes))}
	for i, l := range c.lanes {
		rec.vals[i] = l.pull()
	}
	c.next++
	return rec
}

// ring is a single producer, single consumer queue of records: the control goroutine pushes, the
// audio thread peeks and drops. No locks; the audio side never waits
type ring struct {
	buf        []*stepRec
	head, tail atomic.Uint64
	// space is signalled (without waiting) when the audio side has made room: the control goroutine waits on it when the ring is
	// full, instead of waking up every millisecond to look
	space chan struct{}
}

func newRing(n int) *ring { return &ring{buf: make([]*stepRec, n), space: make(chan struct{}, 1)} }

func (r *ring) len() int { return int(r.tail.Load() - r.head.Load()) }

func (r *ring) push(rec *stepRec) bool {
	t := r.tail.Load()
	if int(t-r.head.Load()) == len(r.buf) {
		return false
	}
	r.buf[t%uint64(len(r.buf))] = rec
	r.tail.Store(t + 1)
	return true
}

func (r *ring) peek() *stepRec {
	h := r.head.Load()
	if h == r.tail.Load() {
		return nil
	}
	return r.buf[h%uint64(len(r.buf))]
}

func (r *ring) drop() {
	r.head.Add(1)
	select {
	case r.space <- struct{}{}:
	default:
	}
}

// Go moves the generators to a goroutine of their own, that stays up to lookahead steps ahead of
// the audio, and waits until it is that far. Call it before the first Process, and Close when done.
// Without Go the runner pulls the generators itself, at the start of every step (offline rendering)
func (r *Runner) Go(lookahead int) error {
	r.start(lookahead)
	for r.ring.len() < lookahead {
		select {
		case <-r.done:
			return r.Close()
		case <-time.After(200 * time.Microsecond):
		}
	}
	return nil
}

func (r *Runner) start(lookahead int) {
	if r.ring != nil {
		panic("dsl: Go called twice")
	}
	r.ring = newRing(max(lookahead, 1))
	r.prog.rt.Live = true
	r.stop = make(chan struct{})
	r.wake = make(chan struct{}, 1)
	r.done = make(chan struct{})
	go func() {
		defer close(r.done)
		defer func() {
			if p := recover(); p != nil {
				r.err.Store(fmt.Errorf("generator panicked: %v", p))
			}
		}()
		for {
			rec := r.ctl.produce()
			for !r.ring.push(rec) {
				select {
				case <-r.stop:
					return
				case <-r.ring.space:
				case <-r.wake:
					// a program waits to take over while this record waits for room: the record is made new too
					r.takeOver(&rec)
				}
			}
			if r.hasPending() {
				r.takeOver(nil)
			}
			select {
			case <-r.stop:
				return
			default:
			}
		}
	}()
}

// Close stops the goroutine of Go and returns the error if a generator panicked in it
func (r *Runner) Close() error {
	if r.ring != nil {
		select {
		case <-r.stop:
		default:
			close(r.stop)
		}
		<-r.done
	}
	if e := r.err.Load(); e != nil {
		return e.(error)
	}
	return nil
}

// Underruns counts the steps whose values were not there when the step started, because the
// control side was late (a generator that takes too long). Such a step gets no new values (a
// note is dropped) and the steps after it stay on time
func (r *Runner) Underruns() int { return int(r.underruns.Load()) }

// fetch gives the record of step k: made now, or taken from the ring (nil: not ready)
func (r *Runner) fetch(k int64) *stepRec {
	if r.ring == nil {
		return r.ctl.produce()
	}
	for {
		rec := r.ring.peek()
		if rec == nil {
			r.underruns.Add(1)
			return nil
		}
		r.ring.drop()
		if rec.idx == k {
			return r.translated(rec)
		}
		// a late record of a step that has been played without it: it is of no use any more
	}
}

// swapReq is a program that takes over from the one that plays
type swapReq struct {
	prog        *Program
	stepSamples float64
	// the records that were made for the program before it and are not played yet, in the lanes of this one (the lanes that are the same
	// have their values, the new ones values of their own), and the step from which on the records are made for it
	translated map[int64]*stepRec
	newFrom    int64
	handed     chan struct{} // closed when the control goroutine has handed the program to the audio side
}

// Swap makes the program of out take over from the one that plays, with the state of what is the same kept (see swap.go): an oscillator
// goes on from its phase, a filter from its memory, a reverb keeps its tail, a generator its place in its phrase. It takes over at the
// next step. Only for a runner that has Go. The patch must have as many channels as the one that plays; its BPM and StepsPerBeat are
// the new clock. NoOptimize is better for it: an optimized graph is another shape after a small edit. Changes can follow each other as
// fast as they come: if one has not taken over when the next is made, the next takes over from it
func (r *Runner) Swap(out Sig, p Patch) error {
	if r.ring == nil {
		return fmt.Errorf("dsl: Swap: the generators of this runner do not run on a goroutine (call Go)")
	}
	p = p.defaults()
	r.swapMu.Lock()
	rate := r.latest.rt.SampleRate
	stepSamples := rate * 60 / (p.BPM * p.StepsPerBeat)
	if len(out.ch) != len(r.latest.outs) {
		r.swapMu.Unlock()
		return fmt.Errorf("dsl: Swap: the patch has %d channels, the one that plays has %d", len(out.ch), len(r.latest.outs))
	}
	prog := compileFrom(r.latest, out, rate, stepSamples, !p.NoOptimize)
	r.latest = prog
	req := &swapReq{prog: prog, stepSamples: stepSamples, handed: make(chan struct{})}
	r.pending = req
	r.swapMu.Unlock()
	select {
	case r.wake <- struct{}{}:
	default:
	}
	// Swap returns when the control goroutine has handed the program to the audio side: from then on it takes it at its next step
	select {
	case <-req.handed:
		return nil
	case <-r.done:
		return fmt.Errorf("dsl: Swap: the control goroutine has stopped")
	case <-time.After(2 * time.Second):
		return fmt.Errorf("dsl: Swap: the control goroutine did not take the program")
	}
}

func (r *Runner) hasPending() bool {
	r.swapMu.Lock()
	defer r.swapMu.Unlock()
	return r.pending != nil
}

// takeOver is the control side of a swap: the records that are made but not played are made again for the lanes of the new program, and the
// new program is handed to the audio side, which takes it at its next step. inHand is a record that was made and does not fit in the ring
func (r *Runner) takeOver(inHand **stepRec) {
	r.swapMu.Lock()
	req := r.pending
	r.pending = nil
	r.swapMu.Unlock()
	if req == nil {
		return
	}
	for _, f := range req.prog.rebases { // generators that changed inside a lane that is kept: put in place, before the records are made right
		f()
	}
	old := map[*byte]int{} // where the lanes of the old layout are
	for i, l := range r.ctl.lanes {
		old[l.id] = i
	}
	translate := func(rec *stepRec) *stepRec {
		if t, ok := r.lastTrans[rec.idx]; ok && rec.idx < r.lastNewFrom {
			rec = t // not played yet, and made for the program before this one: its translation is what has the lanes that are the old ones now
		}
		out := &stepRec{idx: rec.idx, vals: make([]any, len(req.prog.lanes))}
		for j, l := range req.prog.lanes {
			if i, ok := old[l.id]; ok {
				out.vals[j] = rec.vals[i]
				if l.refresh != nil {
					out.vals[j] = l.refresh(out.vals[j])
				}
			} else {
				out.vals[j] = l.pull() // a lane that is new: its first values are those of the steps that are not played yet
			}
		}
		return out
	}
	req.translated = map[int64]*stepRec{}
	var pending []*stepRec
	for h := r.ring.head.Load(); h < r.ring.tail.Load(); h++ {
		pending = append(pending, r.ring.buf[h%uint64(len(r.ring.buf))])
	}
	if inHand != nil {
		pending = append(pending, *inHand)
	}
	// the lanes that make their values again are told what the records have, before they are made right one by one
	first := r.ctl.next
	if len(pending) > 0 {
		first = pending[0].idx
	}
	for _, l := range req.prog.lanes {
		i, ok := old[l.id]
		if l.prepare == nil || !ok {
			continue
		}
		vals := make([]any, len(pending))
		for k, rec := range pending {
			if t, ok := r.lastTrans[rec.idx]; ok && rec.idx < r.lastNewFrom {
				rec = t
			}
			vals[k] = rec.vals[i]
		}
		l.prepare(first, vals)
	}
	for _, rec := range pending {
		req.translated[rec.idx] = translate(rec)
	}
	if inHand != nil {
		*inHand = req.translated[(*inHand).idx]
	}
	for _, l := range req.prog.lanes {
		if l.finish != nil {
			l.finish()
		}
	}
	req.newFrom = r.ctl.next
	r.ctl.lanes = req.prog.lanes
	r.lastTrans, r.lastNewFrom = req.translated, req.newFrom
	r.incoming.Store(req)
	close(req.handed)
}

// translated gives the record as the program that plays has it: a record made before a swap is the translated one
func (r *Runner) translated(rec *stepRec) *stepRec {
	if r.active == nil {
		return rec
	}
	if rec.idx >= r.active.newFrom {
		r.active = nil
		return rec
	}
	if t, ok := r.active.translated[rec.idx]; ok {
		return t
	}
	return rec
}

// install is the audio side of a swap: from here on the program, the clock and the lanes are the new ones. The constants that kept their
// buffers are written now: the old program has run its last block
func (r *Runner) install(s *swapReq) {
	for _, f := range s.prog.fills {
		f()
	}
	s.prog.fills = nil
	r.prog = s.prog
	r.prog.rt.StepSamples = s.stepSamples
	r.stepSamples = s.stepSamples
	r.applyLanes = s.prog.lanes
	r.active = s
}
