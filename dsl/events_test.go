package dsl

import (
	"testing"
	"time"
)

// fixedEvents makes events of the lengths that are given, one after the other, with a gate that stays high for half of the length and
// a number that counts the events (1, 2, 3...)
type fixedEvents struct {
	steps []float64
	n     int
}

func (f *fixedEvents) Generate() Event {
	s := f.steps[f.n%len(f.steps)]
	f.n++
	return Event{Steps: s, Hold: s / 2, Vals: []float64{float64(f.n)}}
}
func (*fixedEvents) Continuous() bool { return true }
func (*fixedEvents) Done() bool       { return false }
func (*fixedEvents) Reset()           {}

func TestEventsStartInTheBlockTheyFallIn(t *testing.T) {
	// 210 beats a minute, 4 steps a beat, 44100 Hz: a step of 3150 samples
	p := Patch{SampleRate: 44100, BPM: 210, StepsPerBeat: 4}
	ev := Events(&fixedEvents{steps: []float64{4, 3, 1}}, 1, 1)
	r := p.Start(Chans(ev[0].Gate, ev[0].Vals[0]))
	n := 8 * 3150
	g, v := make([]float64, n), make([]float64, n)
	r.Process(g, v)
	// the events start at steps 0, 4, 7, 8 and have gates of half their length: 2, 1.5 and 0.5 steps
	type want struct{ start, hold float64 }
	for k, w := range []want{{0, 2}, {4, 1.5}, {7, 0.5}} {
		start := int(w.start * 3150)
		start -= start % 64
		end := int((w.start + w.hold) * 3150)
		end -= end % 64
		if g[start] != 1 || (start > 0 && g[start-1] != 0) {
			t.Errorf("event %d: the gate does not rise at %d", k, start)
		}
		if g[end-1] != 1 || g[end] != 0 {
			t.Errorf("event %d: the gate does not fall at %d: %v %v", k, end, g[end-1], g[end])
		}
		if v[start] != float64(k+1) || (start > 0 && v[start-1] != float64(k)) && k > 0 {
			t.Errorf("event %d: the number is %v at its start, %v before", k, v[start], v[max(start-1, 0)])
		}
	}
}

func TestEventsGoToTheVoicesInTurn(t *testing.T) {
	p := Patch{SampleRate: 44100, BPM: 210, StepsPerBeat: 4}
	ev := Events(&fixedEvents{steps: []float64{2}}, 2, 1)
	r := p.Start(Chans(ev[0].Vals[0], ev[1].Vals[0]))
	n := 6 * 3150
	a, b := make([]float64, n), make([]float64, n)
	r.Process(a, b)
	at := func(x []float64, step int) float64 { return x[step*3150+100] }
	// events 1, 2, 3 start at steps 0, 2, 4: voice 0 has 1 and 3, voice 1 has 2. Before its first event a voice has the number of the first
	if at(a, 0) != 1 || at(b, 0) != 1 || at(a, 2) != 1 || at(b, 2) != 2 || at(a, 4) != 3 || at(b, 4) != 2 {
		t.Errorf("voice 0: %v %v %v, voice 1: %v %v %v", at(a, 0), at(a, 2), at(a, 4), at(b, 0), at(b, 2), at(b, 4))
	}
}

func TestEventsWithAReset(t *testing.T) {
	// what is made inside WithReset starts again at the rise of the gate: a counter (a custom node) counts from 0 again at every note
	p := Patch{SampleRate: 44100, BPM: 210, StepsPerBeat: 4}
	ev := Events(&fixedEvents{steps: []float64{2}}, 1, 0)
	var counter Sig
	WithReset(ev[0].Gate, func() {
		counter = Custom(Spec{Name: "count", Inputs: []Sig{N(0)}, New: func(rt *Rt) Process {
			c := 0.0
			return func(in [][]float64, out []float64, n int) {
				for i := 0; i < n; i++ {
					out[i] = c
					c++
				}
			}
		}})
	})
	r := p.Start(counter)
	x := make([]float64, 6*3150)
	r.Process(x)
	// the notes start at 0, 6300, 12600 (in blocks: 6272, 12544): the counter is 0 there and goes on
	for _, s := range []int{0, 6272, 12544} {
		if x[s] != 0 || x[s+10] != 10 {
			t.Errorf("at %d the counter is %v and %v ten later", s, x[s], x[s+10])
		}
	}
	if x[6272-1] != 6272-1 {
		t.Errorf("before the second note it counts on: %v", x[6272-1])
	}
}

type boolSeq struct {
	v []bool
	n int
}

func (b *boolSeq) Generate() bool { x := b.v[b.n%len(b.v)]; b.n++; return x }
func (*boolSeq) Continuous() bool { return true }
func (*boolSeq) Done() bool       { return false }
func (*boolSeq) Reset()           {}

type floatSeq struct{ n float64 }

func (f *floatSeq) Generate() float64 { f.n++; return f.n }
func (*floatSeq) Continuous() bool    { return true }
func (*floatSeq) Done() bool          { return false }
func (*floatSeq) Reset()              {}

// a rhythm gives the notes: one on every true, as long as the steps up to the next true; a number from the field for every note; a
// rhythm that starts with rests has its first note after them
func TestRhythmEvents(t *testing.T) {
	r := RhythmEvents(&boolSeq{v: []bool{false, false, true, false, false, true, true}}, func(s float64) float64 { return s / 2 }, []Gen[float64]{&floatSeq{}})
	e1 := r.Generate()
	if e1.Delay != 2 || e1.Steps != 3 || e1.Hold != 1.5 || e1.Vals[0] != 1 {
		t.Errorf("first: %+v", e1)
	}
	e2 := r.Generate()
	if e2.Delay != 0 || e2.Steps != 1 || e2.Vals[0] != 2 {
		t.Errorf("second: %+v", e2)
	}
	// the rhythm is 7 steps: false false true false false true true, then false false true...: the third note is 3 steps again
	e3 := r.Generate()
	if e3.Steps != 3 || e3.Vals[0] != 3 {
		t.Errorf("third: %+v", e3)
	}
	// played: the first note is at step 2
	p := Patch{SampleRate: 44100, BPM: 210, StepsPerBeat: 4}
	ev := Events(RhythmEvents(&boolSeq{v: []bool{false, false, true, false, true}}, func(s float64) float64 { return s }, nil), 1, 0)
	rr := p.Start(ev[0].Gate)
	g := make([]float64, 6*3150)
	rr.Process(g)
	at := 2 * 3150
	at -= at % 64
	if g[at-1] != 0 || g[at] != 1 {
		t.Errorf("the first note does not start at step 2: %v %v", g[at-1], g[at])
	}
}

// an event generator that has a fingerprint, for the tests of live edits of the fields
type fpEventsTest struct {
	*eventGen
	fp string
}

func (g fpEventsTest) Fingerprint() string { return g.fp }

func counting(from float64) *fpCounter {
	c := &fpCounter{fp: "c"}
	c.calls.Store(int64(from))
	return c
}

func TestTheFieldsOfAPatternAreRebasedByTheirFingerprints(t *testing.T) {
	mk := func(f0, f1 Gen[float64]) Gen[Event] {
		return fpEventsTest{EventGen(Loop(2.0), func(s float64) float64 { return s }, []Gen[float64]{f0, f1}).(*eventGen), "pattern"}
	}
	a0, a1 := counting(0), counting(100)
	r := started(t, Chans(EventsWithFields(mk(a0, a1), 1, []Sig{{}, {}}, EventFP{Timing: "t", Hold: "h", Fields: []string{"f0", "f1"}})[0].Vals...))
	run(r, 5000)
	// the second field is another generator, the first is the same: the first goes on, the second is the new one from its start
	b0, b1 := counting(0), counting(500)
	if err := r.Swap(Chans(EventsWithFields(mk(b0, b1), 1, []Sig{{}, {}}, EventFP{Timing: "t", Hold: "h", Fields: []string{"f0", "g1"}})[0].Vals...), swapPatch); err != nil {
		t.Fatal(err)
	}
	if kept, fresh := r.LastSwap(); fresh != 0 {
		t.Errorf("kept %d, new %d: %v", kept, fresh, r.LastRestarted())
	}
	x := run(r, 30000)
	if b0.pulls() != 0 {
		t.Errorf("the first field was pulled from the new generator %d times: it should be the old one", b0.pulls())
	}
	if b1.pulls() == 0 {
		t.Error("the second field never was pulled from its new generator")
	}
	// the second field is 501, 502, ... from the first note after the swap on: no note has a number of the old one (101...) after it
	last := x[1][len(x[1])-1]
	if last < 501 || last > 520 {
		t.Errorf("the second field ends at %v", last)
	}
	// the first goes on counting where it was: a note every 2 steps of 1000 samples: at 35000 samples it is at about 18
	if x[0][len(x[0])-1] < 15 {
		t.Errorf("the first field went back: %v", x[0][len(x[0])-1])
	}
	// fingerprints that cannot be told: the fields are replaced (the pattern is kept), and a pattern with another number of fields is new
	c0, c1 := counting(0), counting(900)
	if err := r.Swap(Chans(EventsWithFields(mk(c0, c1), 1, []Sig{{}, {}}, EventFP{Hold: "h"})[0].Vals...), swapPatch); err != nil {
		t.Fatal(err)
	}
	if _, fresh := r.LastSwap(); fresh != 0 {
		t.Errorf("fields without fingerprints started the pattern again: %v", r.LastRestarted())
	}
	run(r, 4000)
	if c0.pulls() == 0 || c1.pulls() == 0 {
		t.Error("fields without fingerprints are not put in place")
	}
	only := counting(0)
	one := EventsWithFields(fpEventsTest{EventGen(Loop(2.0), func(s float64) float64 { return s }, []Gen[float64]{only}).(*eventGen), "pattern"}, 1, []Sig{{}}, EventFP{Timing: "t", Hold: "h", Fields: []string{"x"}})[0].Vals[0]
	if err := r.Swap(Chans(one, one), swapPatch); err != nil {
		t.Fatal(err)
	}
	if _, fresh := r.LastSwap(); fresh == 0 {
		t.Error("a pattern with another number of fields was kept")
	}
}

// events of no steps start on the step of the next one (a chord); a generator that never gets on to the next step is cut short
func TestEventsOfZeroStepsAreOnTheSameStep(t *testing.T) {
	p := Patch{SampleRate: 44100, BPM: 210, StepsPerBeat: 4}
	ev := Events(&holdEvents{steps: []float64{0, 0, 3}}, 3, 1)
	r := p.Start(Chans(ev[0].Gate, ev[1].Gate, ev[2].Gate))
	n := 8 * 3150
	a, b, c := make([]float64, n), make([]float64, n), make([]float64, n)
	r.Process(a, b, c)
	// the three events of the first group start at step 0, the three of the next at step 3, on a voice each
	for _, g := range [][]float64{a, b, c} {
		if g[100] != 1 || g[3*3150+100] != 1 {
			t.Fatalf("a voice is not on in both groups: %v %v", g[100], g[3*3150+100])
		}
	}
	loop := Events(&holdEvents{steps: []float64{0}}, 1, 1)
	q := p.Start(Chans(loop[0].Gate))
	done := make(chan struct{})
	go func() {
		q.Process(make([]float64, 8*3150))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("events that never get to the next step hang the control side")
	}
}

// holdEvents are events of the lengths that are given, each held for a step
type holdEvents struct {
	steps []float64
	n     int
}

func (f *holdEvents) Generate() Event {
	s := f.steps[f.n%len(f.steps)]
	f.n++
	return Event{Steps: s, Hold: 1, Span: 3, Vals: []float64{float64(f.n)}}
}
func (*holdEvents) Continuous() bool { return true }
func (*holdEvents) Done() bool       { return false }
func (*holdEvents) Reset()           {}
