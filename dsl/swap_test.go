package dsl

import (
	"math"
	"sync/atomic"
	"testing"
	"time"
)

// run plays n samples from the runner, in blocks of the size of a device buffer, and gives the channels
func run(r *Runner, n int) [][]float64 {
	ch := len(r.prog.outs)
	out := make([][]float64, ch)
	for i := range out {
		out[i] = make([]float64, n)
	}
	const buf = 128
	for at := 0; at < n; at += buf {
		// the test runs much faster than the clock: wait for the control goroutine to be ahead, as it is in time
		for deadline := time.Now().Add(2 * time.Second); r.ring.len() < 2 && time.Now().Before(deadline); {
			time.Sleep(50 * time.Microsecond)
		}
		end := min(at+buf, n)
		piece := make([][]float64, ch)
		for i := range piece {
			piece[i] = out[i][at:end]
		}
		r.Process(piece...)
	}
	return out
}

// 120 bpm, 4 steps a beat, 8000 Hz: a step is 1000 samples
var swapPatch = Patch{SampleRate: 8000, BPM: 120, StepsPerBeat: 4, Lookahead: 4, NoOptimize: true}

func started(t *testing.T, out Sig) *Runner {
	t.Helper()
	r := swapPatch.Start(out)
	if err := r.Go(swapPatch.Lookahead); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

// the swap is in effect at a step: find the first sample where the output is not what the old program gave (the signal that is
// compared with is given by the test)
func TestSwapKeepsTheStateOfWhatIsTheSame(t *testing.T) {
	// a sine of 100 Hz at 0.5, and then at 0.25 and 130 Hz in the next version: the constants differ, the graph is the same: the phase goes on
	a := Mul(Sine(100.0), 0.5)
	r := started(t, a)
	first := run(r, 3000)
	if err := r.Swap(Mul(Sine(130.0), 0.25), swapPatch); err != nil {
		t.Fatal(err)
	}
	rest := run(r, 40000)
	// the frequency changes at the swap, the phase is what it was then: find the swap (where the frequency changes) and check that the
	// phase is continuous there: the sine of the new version at the sample is sin(phase0 + 2 pi 130 (t - t0)) for the phase that the old one had
	all := append(first[0], rest[0]...)
	swapAt := -1
	for i := 3000; i < len(all); i++ {
		// the old version never gets above 0.5, the new one never above 0.25: the first sample of the new has the new amplitude
		// relation to its neighbours: look at the second difference being what a 130 Hz sine of 0.25 has instead of 100 Hz of 0.5
		old := 0.5 * math.Sin(2*math.Pi*100*float64(i)/8000)
		if math.Abs(all[i]-old) > 1e-9 {
			swapAt = i
			break
		}
	}
	if swapAt < 0 || swapAt > 3000+1000+1 { // at the next step after the call
		t.Fatalf("the swap did not happen in time: %d", swapAt)
	}
	// the first sample that differs may be one after the step (a sine of 0 is 0 for both): the swap is at the step before it
	swapAt -= swapAt % 1000
	// the phase at the swap is the phase of the old sine at that sample, and 130 Hz goes on from it
	phase0 := 2 * math.Pi * 100 * float64(swapAt) / 8000
	for i := swapAt; i < swapAt+2000; i++ {
		want := 0.25 * math.Sin(phase0+2*math.Pi*130*float64(i-swapAt)/8000)
		if math.Abs(all[i]-want) > 1e-6 {
			t.Fatalf("sample %d: %v, want %v: the oscillator did not keep its phase", i, all[i], want)
		}
	}
}

func TestSwapRestartsWhatChangedInItself(t *testing.T) {
	// the delay time is in the node: a delay with another maximum is another node, and its memory is gone; the sine before it keeps its phase
	r := started(t, SCDelayN(Sine(100.0), 0.02, N(0.005)))
	run(r, 3000)
	if err := r.Swap(SCDelayN(Sine(100.0), 0.03, N(0.005)), swapPatch); err != nil {
		t.Fatal(err)
	}
	// it is built new: the program lists the delay as built (not taken over)
	if got := r.latest.entries; len(got) == 0 {
		t.Fatal("no entries")
	}
	var kept, fresh int
	for _, e := range r.latest.entries {
		if e.n.op != opOpaque {
			continue
		}
		if e.reuse {
			kept++
		} else {
			fresh++
		}
	}
	if kept != 1 || fresh != 1 { // the sine is kept, the delay is not
		t.Errorf("kept %d, new %d: want the sine kept and the delay new", kept, fresh)
	}
}

func TestSwapWithAnotherOscillatorKeepsTheFilterAfterIt(t *testing.T) {
	r := started(t, Lowpass(Saw(100.0), 800.0, 0.5))
	run(r, 1000)
	// a pulse for a saw: the oscillator is another node, the lowpass after it is the same filter, with its memory, given another input
	if err := r.Swap(Lowpass(Pulse(100.0, 0.5), 800.0, 0.5), swapPatch); err != nil {
		t.Fatal(err)
	}
	var kept, fresh []string
	for _, e := range r.latest.entries {
		if e.n.op != opOpaque {
			continue
		}
		if e.reuse {
			kept = append(kept, e.n.name)
		} else {
			fresh = append(fresh, e.n.name)
		}
	}
	if len(kept) != 1 || kept[0] != "lowpass" || len(fresh) != 1 || fresh[0] != "pulse" {
		t.Errorf("kept %v, new %v: the filter should be kept and the oscillator new", kept, fresh)
	}
	x := run(r, 4000)[0]
	for i, v := range x {
		if math.IsNaN(v) || math.Abs(v) > 5 {
			t.Fatalf("sample %d is %v", i, v)
		}
	}
	run(r, 3000) // the first change takes over before the next
	if err := r.Swap(Lowpass(Saw(100.0), 800.0, 0.5), swapPatch); err != nil {
		t.Fatal(err)
	}
}

// the input of a node that goes on is another one than before (a layer was taken out of a sum): the node keeps its state, with the new
// input copied to where it reads
func TestSwapWhereTheInputOfAKeptNodeIsAnotherNode(t *testing.T) {
	a, b := Mul(Sine(100.0), 0.3), Mul(Sine(230.0), 0.3)
	r := started(t, SCDelayN(Add(a, b), 0.05, N(0.01)))
	run(r, 4000)
	// the sum is gone: the delay is given one of the sines. It is the same delay, its line holds what it heard
	if err := r.Swap(SCDelayN(Mul(Sine(100.0), 0.3), 0.05, N(0.01)), swapPatch); err != nil {
		t.Fatal(err)
	}
	if _, fresh := r.LastSwap(); fresh != 0 {
		t.Errorf("%d nodes started again: %v", fresh, r.LastRestarted())
	}
	x := run(r, 6000)[0]
	// the delay time is 0.01 s = 80 samples, and the program takes over at sample 4000, where the audio is: the first 80 samples that it gives are
	// what the line held (the old sum), after that the delayed new input (the 100 Hz sine alone)
	sum := func(t int) float64 {
		return 0.3*math.Sin(2*math.Pi*100*float64(t)/8000) + 0.3*math.Sin(2*math.Pi*230*float64(t)/8000)
	}
	if math.Abs(x[10]-sum(4000+10-80)) > 1e-6 {
		t.Errorf("the line does not hold the sum that came before the swap: %v, want %v", x[10], sum(4000+10-80))
	}
	if want := 0.3 * math.Sin(2*math.Pi*100*float64(4000+200-80)/8000); math.Abs(x[200]-want) > 1e-6 {
		t.Errorf("after the swap the delay does not give the new input: %v, want %v", x[200], want)
	}
	// and a buffer that another node uses is not written into: the sum is still in the program, as the input of another node
	r2 := started(t, Chans(SCDelayN(Add(a, b), 0.05, N(0.01)), Add(a, b)))
	run(r2, 2000)
	if err := r2.Swap(Chans(SCDelayN(a, 0.05, N(0.01)), Add(a, b)), swapPatch); err != nil {
		t.Fatal(err)
	}
	y := run(r2, 4000)
	for i := range y[1] {
		want := 0.3*math.Sin(2*math.Pi*100*float64(2000+i)/8000) + 0.3*math.Sin(2*math.Pi*230*float64(2000+i)/8000)
		if math.Abs(y[1][i]-want) > 1e-6 {
			t.Fatalf("the sum that stays in the program is wrong at %d: %v, want %v", i, y[1][i], want)
		}
	}
}

func TestSwapKeepsGeneratorsInTheirPhrase(t *testing.T) {
	g1 := &fpCounter{fp: "count"}
	r := started(t, Hold(g1))
	run(r, 5000)
	// the same generator (the same fingerprint) in the new patch: the old one goes on, the new one is never pulled
	g2 := &fpCounter{fp: "count"}
	if err := r.Swap(Mul(Hold(g2), 2.0), swapPatch); err != nil {
		t.Fatal(err)
	}
	x := run(r, 30000)[0]
	if g2.pulls() != 0 {
		t.Errorf("the new generator was pulled %d times", g2.pulls())
	}
	// the new program doubles what the old generator makes: the values count on, without going back to 1
	last := x[len(x)-1] / 2
	if last < 9 {
		t.Errorf("the generator went back to the start: its last value is %v after 35 steps", last)
	}
	// another fingerprint is another generator: it starts at its start
	g3 := &fpCounter{fp: "other"}
	if err := r.Swap(Mul(Hold(g3), 2.0), swapPatch); err != nil {
		t.Fatal(err)
	}
	y := run(r, 30000)[0]
	// the old generator would be at about 40 by now, doubled 80: the new one has counted from 1
	if g3.pulls() < 20 || y[len(y)-1] > 2*float64(g3.pulls()) {
		t.Errorf("a generator with another fingerprint: pulled %d times, last %v", g3.pulls(), y[len(y)-1])
	}
	// without a fingerprint the generator can not be told: it is new every time
	g4 := &plainCounter{}
	r2 := started(t, Hold(g4))
	run(r2, 2000)
	g5 := &plainCounter{}
	if err := r2.Swap(Hold(g5), swapPatch); err != nil {
		t.Fatal(err)
	}
	run(r2, 20000)
	if g5.pulls() == 0 {
		t.Error("a generator without a fingerprint was taken over")
	}
}

type plainCounter struct{ calls atomic.Int64 }

func (c *plainCounter) pulls() int        { return int(c.calls.Load()) }
func (c *plainCounter) Generate() float64 { return float64(c.calls.Add(1)) }
func (*plainCounter) Continuous() bool    { return true }
func (*plainCounter) Done() bool          { return false }
func (*plainCounter) Reset()              {}

type fpCounter struct {
	plainCounter
	fp string
}

func (c *fpCounter) Fingerprint() string { return c.fp }

func TestSwapErrors(t *testing.T) {
	r := swapPatch.Start(Sine(100.0))
	if err := r.Swap(Sine(200.0), swapPatch); err == nil {
		t.Error("a swap without Go")
	}
	r = started(t, Sine(100.0))
	if err := r.Swap(Chans(Sine(100.0), Sine(200.0)), swapPatch); err == nil {
		t.Error("another number of channels")
	}
}

func TestSwapGroupsAreKeptTogether(t *testing.T) {
	// the gate and the pitch of a voice share state: kept both or neither
	mk := func() Voice { return Notes(Loop(60.0, 62.0), Loop(true), 0.5) }
	v := mk()
	r := started(t, Chans(v.Pitch, v.Gate))
	run(r, 2000)
	if err := r.Swap(Chans(mk().Pitch, mk().Gate), swapPatch); err != nil {
		t.Fatal(err)
	}
	for _, e := range r.latest.entries {
		if e.n.group != nil && e.reuse {
			t.Error("a group of nodes was kept without fingerprints")
		}
	}
}

// a change takes over at the next step, whatever the lookahead of the control side is
func TestSwapTakesOverAtTheNextStepWhateverTheLookahead(t *testing.T) {
	p := swapPatch
	p.Lookahead = 16
	r := p.Start(Mul(Sine(100.0), 0.5))
	if err := r.Go(p.Lookahead); err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	run(r, 3300) // the clock is 300 samples into step 3
	if err := r.Swap(Mul(Sine(100.0), 0.25), p); err != nil {
		t.Fatal(err)
	}
	x := run(r, 3000)[0]
	// the new gain (0.25) is there from the start of step 4, at 4000: the 700 samples that are left of step 3 are the old program's
	for i := 3300; i < 4000; i++ {
		if want := 0.5 * math.Sin(2*math.Pi*100*float64(i)/8000); math.Abs(x[i-3300]-want) > 1e-9 {
			t.Fatalf("sample %d is %v, the old program gives %v", i, x[i-3300], want)
		}
	}
	for i := 4000; i < 6000; i++ {
		if want := 0.25 * math.Sin(2*math.Pi*100*float64(i)/8000); math.Abs(x[i-3300]-want) > 1e-9 {
			t.Fatalf("sample %d is %v, the new program gives %v: it took over late", i, x[i-3300], want)
		}
	}
}

// the control goroutine is waiting for room in a ring that the audio does not empty: a change reaches it all the same
func TestSwapReachesAControlGoroutineThatWaits(t *testing.T) {
	r := started(t, Mul(Sine(100.0), 0.5))
	time.Sleep(30 * time.Millisecond) // the ring is full, the control goroutine waits for the audio
	if err := r.Swap(Mul(Sine(100.0), 0.25), swapPatch); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	x := run(r, 2000)[0]
	if math.Abs(x[1500]-0.25*math.Sin(2*math.Pi*100*1500/8000)) > 1e-9 {
		t.Errorf("the change did not arrive: %v", x[1500])
	}
}

// changes that follow each other before the first has taken over: the last one plays, and what was kept goes on being kept
func TestSwapsThatFollowEachOtherQuickly(t *testing.T) {
	g := &fpCounter{fp: "count"}
	r := started(t, Hold(g))
	run(r, 2000)
	for i := 1; i <= 4; i++ {
		if err := r.Swap(Mul(Hold(&fpCounter{fp: "count"}), float64(i)), swapPatch); err != nil {
			t.Fatal(err)
		}
	}
	x := run(r, 20000)[0]
	// the last program multiplies by 4, and the values count on from the first generator: 4, 8, 12... each 4 times the one before
	last := x[len(x)-1] / 4
	if last < 15 {
		t.Errorf("the generator did not go on: %v", last)
	}
	for step := 6; step < 20; step++ {
		if x[step*1000+10] != 4*math.Trunc(x[step*1000+10]/4) || x[step*1000+10] <= x[(step-1)*1000+10] {
			t.Fatalf("step %d: %v after %v: not a steady count of 4s", step, x[step*1000+10], x[(step-1)*1000+10])
		}
	}
	// and with a new generator in the middle of the chain, the later ones are that one's
	g2 := &fpCounter{fp: "count"}
	r2 := started(t, Hold(g2))
	run(r2, 2000)
	n1 := &fpCounter{fp: "other"}
	if err := r2.Swap(Mul(Hold(n1), 2.0), swapPatch); err != nil {
		t.Fatal(err)
	}
	if err := r2.Swap(Mul(Hold(&fpCounter{fp: "other"}), 3.0), swapPatch); err != nil {
		t.Fatal(err)
	}
	y := run(r2, 20000)[0]
	if n1.pulls() == 0 || y[len(y)-1]/3 > 2*float64(n1.pulls()) {
		t.Errorf("the generator of the first change: %d pulls, last %v", n1.pulls(), y[len(y)-1])
	}
}
