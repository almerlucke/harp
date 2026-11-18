package dsl

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// render1 renders a patch of one channel
func (p Patch) render1(out Sig, seconds float64) (*Program, []float64) {
	prog, ch := p.Render(out, seconds)
	if len(ch) != 1 {
		panic("render1: not a mono patch")
	}
	return prog, ch[0]
}

func lead() Sig {
	notes := Degrees(57, Minor, Mutate([]int{7, 9, 11, 9, 8, 7, 4, 7}, 0.6))
	v := Notes(notes, Euclid(16, 7, 0), 0.6)
	return Saw(v.Pitch).
		Lowpass(Lfo(0.3).Range(300, 4000), N(0.9)).
		Mul(Adsr(0.005, 0.15, 0.5, 0.2).On(v.Gate))
}

func TestSameSeedSameSamples(t *testing.T) {
	render := func() []float64 {
		Seed(7)
		_, s := Patch{BPM: 110}.render1(lead(), 3)
		return s
	}
	a, b := render(), render()
	var peak float64
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("sample %d differs: %v vs %v", i, a[i], b[i])
		}
		if math.IsNaN(a[i]) || math.IsInf(a[i], 0) {
			t.Fatalf("sample %d is %v", i, a[i])
		}
		peak = math.Max(peak, math.Abs(a[i]))
	}
	if peak < 0.1 {
		t.Fatalf("peak %v: the patch is nearly silent", peak)
	}
}

// A step must begin on its exact sample, whatever the block size
func TestStepStartsOnItsSample(t *testing.T) {
	// 60 bpm, 1 step per beat, 100 Hz sample rate: a step is 100 samples
	p := Patch{SampleRate: 100, BPM: 60, StepsPerBeat: 1}
	g := Loop(1.0, 2.0, 3.0)
	_, s := p.render1(Hold(g), 3)
	for i, want := range map[int]float64{0: 1, 99: 1, 100: 2, 199: 2, 200: 3, 299: 3} {
		if s[i] != want {
			t.Errorf("sample %d = %v, want %v", i, s[i], want)
		}
	}
}

// An unrelated tempo gives fractional steps: the count of steps must still be right
func TestFractionalSteps(t *testing.T) {
	var pulls int
	g := countGen{&pulls}
	p := Patch{SampleRate: 44100, BPM: 117, StepsPerBeat: 4}
	p.render1(Hold(g), 10)
	want := int(math.Ceil(10 * 117 * 4 / 60)) // a step starts at 0, then every 22.6 samples...
	if pulls != want {
		t.Errorf("%d steps, want %d", pulls, want)
	}
}

type countGen struct{ n *int }

func (c countGen) Generate() float64 { *c.n++; return 0 }
func (countGen) Continuous() bool    { return true }
func (countGen) Done() bool          { return false }
func (countGen) Reset()              {}

// A band limited saw must put less energy near Nyquist than a naive one: this is the crispness
func TestSawIsBandLimited(t *testing.T) {
	const sr, f = 44100.0, 3951.0 // a high note, many harmonics fold back
	_, s := Patch{SampleRate: sr}.render1(Saw(f), 0.5)
	naive := make([]float64, len(s))
	var ph float64
	for i := range naive {
		naive[i] = 2*ph - 1
		ph += f / sr
		ph -= math.Floor(ph)
	}
	// energy of the first difference is dominated by what is close to Nyquist
	hf := func(x []float64) (e float64) {
		for i := 1; i < len(x); i++ {
			d := x[i] - x[i-1]
			e += d * d
		}
		return
	}
	if hf(s) >= hf(naive) {
		t.Errorf("polyBLEP saw is not smoother than a naive one: %v vs %v", hf(s), hf(naive))
	}
}

func TestSharedNodeBuiltOnce(t *testing.T) {
	s := Saw(220)
	prog, _ := Patch{}.render1(Add(s, s), 0.01)
	var saws int
	for _, n := range prog.names {
		if n == "saw" {
			saws++
		}
	}
	if saws != 1 {
		t.Errorf("shared saw built %d times", saws)
	}
}

// The melody moves on the steps where the rhythm plays, and only there
func TestNotesAdvanceOnPlayedSteps(t *testing.T) {
	// a step is 100 samples
	p := Patch{SampleRate: 100, BPM: 60, StepsPerBeat: 1}
	v := Notes(Loop(60.0, 62.0, 64.0), Loop(true, false, true, true), 0.5)
	// look at the pitch and the gate together
	_, s := p.render1(Add(Mul(v.Pitch, 1), Mul(v.Gate, 1e6)), 4)
	hz := func(n float64) float64 { return 440 * math.Pow(2, (n-69)/12) }
	at := func(i int, wantHz float64, gate bool) {
		t.Helper()
		g := 0.0
		if gate {
			g = 1e6
		}
		if math.Abs(s[i]-(wantHz+g)) > 1e-6 {
			t.Errorf("sample %d = %v, want pitch %.2f gate %v", i, s[i], wantHz, gate)
		}
	}
	at(0, hz(60), true)    // step 0 plays: first note, gate up
	at(49, hz(60), true)   // gate is up for half the step
	at(50, hz(60), false)  // then down
	at(100, hz(60), false) // step 1 is a rest: the pitch stays, nothing sounds
	at(200, hz(62), true)  // step 2 plays: the NEXT note, not the one after the rest
	at(300, hz(64), true)  // step 3 plays
}

// fakeEngine is a device that is driven by hand, like the audio thread of a sound card
type fakeEngine struct {
	sr      float64
	size    int
	cb      func(in, out [][]float32)
	started chan struct{} // closed by Start
	mu      sync.Mutex
	log     []string
}

func newFake(sr float64, size int) *fakeEngine {
	return &fakeEngine{sr: sr, size: size, started: make(chan struct{})}
}

func (f *fakeEngine) note(s string) { f.mu.Lock(); f.log = append(f.log, s); f.mu.Unlock() }
func (f *fakeEngine) calls() string { f.mu.Lock(); defer f.mu.Unlock(); return fmt.Sprint(f.log) }

func (f *fakeEngine) GetSampleRate() float64                   { return f.sr }
func (f *fakeEngine) GetBufferSize() int                       { return f.size }
func (f *fakeEngine) SetCallback(cb func(in, out [][]float32)) { f.cb = cb }
func (f *fakeEngine) Init() error                              { f.note("init"); return nil }
func (f *fakeEngine) Start() error                             { f.note("start"); close(f.started); return nil }
func (f *fakeEngine) Stop() error                              { f.note("stop"); return nil }
func (f *fakeEngine) Close() error                             { f.note("close"); return nil }

// What a device receives, in buffers of any size, is what Render gives
func TestPlayMatchesRender(t *testing.T) {
	Seed(7)
	_, want := Patch{SampleRate: 48000, BPM: 110}.render1(lead(), 1)

	Seed(7)
	eng := newFake(48000, 1000) // not a multiple of the block size, nor of a step
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error)
	go func() { done <- Patch{BPM: 110}.Play(ctx, lead(), eng) }()
	<-eng.started // the callback is set before the device starts
	var got []float64
	for len(got) < len(want) {
		l, r := make([]float32, eng.size), make([]float32, eng.size)
		eng.cb(nil, [][]float32{l, r})
		for i := range l {
			if l[i] != r[i] {
				t.Fatal("channels differ")
			}
			got = append(got, float64(l[i]))
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	for i := range want {
		if math.Abs(got[i]-math.Max(-1, math.Min(1, want[i]))) > 1e-6 {
			t.Fatalf("sample %d: device %v, render %v", i, got[i], want[i])
		}
	}
	if eng.calls() != "[init start stop close]" {
		t.Errorf("engine calls: %v", eng.calls())
	}
}

// goid is the number of the calling goroutine, for tests only
func goid() int {
	var b [64]byte
	n := runtime.Stack(b[:], false)
	var id int
	fmt.Sscanf(string(b[:n]), "goroutine %d ", &id)
	return id
}

// callerGen is a generator that remembers on which goroutines it was called
type callerGen struct {
	mu   sync.Mutex
	ids  map[int]bool
	next float64
}

func (c *callerGen) Generate() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ids == nil {
		c.ids = map[int]bool{}
	}
	c.ids[goid()] = true
	c.next++
	return c.next
}
func (*callerGen) Continuous() bool { return true }
func (*callerGen) Done() bool       { return false }
func (*callerGen) Reset()           {}

// After Go the generators never run on the goroutine that calls Process (the audio thread), and
// the output is what the offline render makes of the same patch
func TestGeneratorsRunOnTheControlGoroutine(t *testing.T) {
	p := Patch{SampleRate: 100, BPM: 60, StepsPerBeat: 1, Lookahead: 8}
	_, want := p.render1(Hold(&callerGen{}), 6)

	g := &callerGen{}
	r := p.Start(Hold(g))
	if err := r.Go(p.Lookahead); err != nil {
		t.Fatal(err)
	}
	got := make([]float64, 600)
	for i := 0; i < len(got); i += 37 { // odd pieces, as a device may give
		r.Process(got[i:min(i+37, len(got))])
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if g.ids[goid()] {
		t.Error("a generator ran on the goroutine of Process")
	}
	if len(g.ids) != 1 {
		t.Errorf("generators ran on %d goroutines, want 1", len(g.ids))
	}
	if r.Underruns() != 0 {
		t.Errorf("%d underruns", r.Underruns())
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sample %d: live %v, offline %v", i, got[i], want[i])
		}
	}
}

// blockedGen blocks the first call until released: a generator that takes too long
type blockedGen struct {
	release chan struct{}
	n       float64
}

func (b *blockedGen) Generate() float64 {
	if b.n == 0 {
		<-b.release
	}
	b.n++
	return b.n
}
func (*blockedGen) Continuous() bool { return true }
func (*blockedGen) Done() bool       { return false }
func (*blockedGen) Reset()           {}

// A late control side doesn't stall the audio, and doesn't move the steps after it: the step that
// had no values is played without, the late values are thrown away, and the next step is on time
func TestUnderrunDropsTheStepNotTheTiming(t *testing.T) {
	p := Patch{SampleRate: 100, BPM: 60, StepsPerBeat: 1}
	g := &blockedGen{release: make(chan struct{})}
	r := p.Start(Hold(g))
	r.start(8) // no wait for the lookahead: the control side is stuck in the first call

	s0 := make([]float64, 100)
	r.Process(s0) // step 0: nothing there yet
	if r.Underruns() != 1 {
		t.Fatalf("%d underruns, want 1", r.Underruns())
	}
	for _, v := range s0 {
		if v != 0 {
			t.Fatal("step 0 played values that were not there")
		}
	}
	close(g.release)
	for r.ring.len() < 3 { // the late record of step 0 and the next two
		time.Sleep(time.Millisecond)
	}
	s1, s2 := make([]float64, 100), make([]float64, 100)
	r.Process(s1) // step 1 gets ITS values (2), not the late ones of step 0 (1)
	r.Process(s2)
	if s1[0] != 2 || s1[99] != 2 || s2[0] != 3 {
		t.Errorf("steps 1 and 2 are %v..%v and %v, want 2 and 3", s1[0], s1[99], s2[0])
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}

type panicGen struct{ n int }

func (g *panicGen) Generate() float64 {
	if g.n++; g.n > 2 {
		panic("boom")
	}
	return 1
}
func (*panicGen) Continuous() bool { return true }
func (*panicGen) Done() bool       { return false }
func (*panicGen) Reset()           {}

// A generator that panics ends the control goroutine, not the program, and Close tells why
func TestGeneratorPanicIsAnError(t *testing.T) {
	p := Patch{SampleRate: 100, BPM: 60, StepsPerBeat: 1}
	r := p.Start(Hold(&panicGen{}))
	err := r.Go(8) // never gets 8 steps ahead
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Go returned %v", err)
	}
	if err := r.Close(); err == nil {
		t.Error("Close forgot the error")
	}
}

// A node of your own: expands over channels with a state for each, merges by key, knows the position
func TestCustomNodes(t *testing.T) {
	var positions []int64
	double := func(key string, x Sig) Sig {
		return Custom(Spec{Name: "double", Key: key, Inputs: []Sig{x}, New: func(rt *Rt) Process {
			return func(in [][]float64, out []float64, n int) {
				positions = append(positions, rt.Pos())
				for i := 0; i < n; i++ {
					out[i] = in[0][i] * 2
				}
			}
		}})
	}
	stereo := Chans(N(1), N(3))
	_, ch := Patch{}.Render(double("d", stereo), 0.01)
	if len(ch) != 2 || ch[0][7] != 2 || ch[1][7] != 6 {
		t.Errorf("channels %d: %v %v", len(ch), ch[0][7], ch[1][7])
	}
	// the same key and inputs: one node. No key: never merged
	merged := Patch{}.Start(Add(double("d", Saw(220)), double("d", Saw(220)))).Program()
	apart := Patch{}.Start(Add(double("", Saw(220)), double("", Saw(220)))).Program()
	if merged.Kernels() != apart.Kernels()-1 {
		t.Errorf("kernels: %d merged, %d not", merged.Kernels(), apart.Kernels())
	}
	// Pos is the first sample of the block: blocks follow each other
	positions = nil
	Patch{}.Render(double("p", Saw(220)), 0.01) // 441 samples in blocks of up to 64
	var at int64
	for i, p := range positions {
		if p != at {
			t.Fatalf("block %d at position %d, want %d", i, p, at)
		}
		at += BlockSize
		if at > 441 {
			break
		}
	}
	if Signal(0.5).Channels() != 1 || Signal(3).Channels() != 1 || Signal(Chans(N(1), N(2))).Channels() != 2 {
		t.Error("Signal")
	}
}

// A node with several outputs: the outputs of an instance together, an instance for every channel
func TestCustomNOutputs(t *testing.T) {
	made := 0
	split := func(x Sig) Sig {
		return CustomN(SpecN{Name: "split", Inputs: []Sig{x}, Outputs: 3, New: func(rt *Rt) ProcessN {
			made++
			return func(in [][]float64, out [][]float64, n int) {
				for i := 0; i < n; i++ {
					out[0][i], out[1][i], out[2][i] = in[0][i], in[0][i]*10, in[0][i]*100
				}
			}
		}})
	}
	_, ch := Patch{}.Render(split(N(2)), 0.01)
	if len(ch) != 3 || ch[0][9] != 2 || ch[1][9] != 20 || ch[2][9] != 200 {
		t.Fatalf("%d channels: %v %v %v", len(ch), ch[0][9], ch[1][9], ch[2][9])
	}
	// two channels in: an instance each, state of its own, the outputs of an instance together
	made = 0
	_, ch = Patch{}.Render(split(Chans(N(1), N(2))), 0.01)
	want := []float64{1, 10, 100, 2, 20, 200}
	if len(ch) != 6 || made != 2 {
		t.Fatalf("%d channels, %d instances", len(ch), made)
	}
	for i, w := range want {
		if ch[i][9] != w {
			t.Errorf("channel %d is %v, want %v", i, ch[i][9], w)
		}
	}
	// only one of the later outputs is used: the node still runs
	if _, only := (Patch{}).render1(split(N(2)).Ch(2), 0.01); only[9] != 200 {
		t.Errorf("only the third output: %v", only[9])
	}
	// the same patch built twice, and optimized or not: the same
	both := func(noOpt bool) []float64 {
		_, c := Patch{NoOptimize: noOpt}.Render(Add(split(N(2)), Chans(N(1), N(1), N(1))), 0.01)
		return []float64{c[0][5], c[1][5], c[2][5]}
	}
	if o, p := both(false), both(true); o[0] != 3 || o[1] != 21 || o[2] != 201 || o[0] != p[0] || o[1] != p[1] || o[2] != p[2] {
		t.Errorf("optimized %v, plain %v", o, p)
	}
	// two of them are never merged, even if they are the same
	x := N(1)
	if n := func() int { made = 0; Patch{}.Start(Add(split(x), split(x))); return made }(); n != 2 {
		t.Errorf("%d instances for two nodes", n)
	}
}
