package blocks

import (
	"context"
	"flag"
	"math"
	"runtime"
	"runtime/debug"
	"slices"
	"testing"
	"time"

	"harp/dsl"

	"github.com/almerlucke/psalm/synths/subtractive"
)

// The soak test: does the live path ever miss its deadline? It runs a heavy patch through Patch.Play,
// on a device that asks for a buffer on a real clock, and times every callback. A short test
// can pass while a rare event (a garbage collection at the wrong moment, a control goroutine that
// fell behind) goes unseen, so it is made to run for minutes:
//
//	go test -run Soak -soak 2m -timeout 0 -v ./blocks
//	go test -run Soak -soak 2m -timeout 0 -v -soak-garbage 50 ./blocks     # with a part of the program making garbage
//	go test -run Soak -soak 2m -timeout 0 -v -soak-gcpercent 400 ./blocks  # with the collector tuned
//
// Without -soak a short run of the same code is a smoke test of the test itself.

var (
	soakFor       = flag.Duration("soak", 0, "run the soak test for this long (0: skip it); needs -timeout 0 or longer")
	soakVoices    = flag.Int("soak-voices", 64, "voices in the patch of the soak test")
	soakSynth     = flag.Int("soak-synth", 8, "voices of each of the two synths (a pad and an arpeggio) in the soak test, 0 for none")
	soakBuffer    = flag.Int("soak-buffer", 512, "samples per buffer of the device of the soak test")
	soakGarbage   = flag.Float64("soak-garbage", 0, "MB per second of garbage made by another part of the program, to stress the collector")
	soakGCPercent = flag.Int("soak-gcpercent", -2, "debug.SetGCPercent for the run (-2: leave it as it is, -1: collector off)")
	soakMemLimit  = flag.Int("soak-memlimit", 0, "debug.SetMemoryLimit for the run, in MiB (0: leave it)")
)

// soakPatch is a bank of voices: a saw each, from a chord generator, each with its own filter on a
// moving cutoff, a gate and envelope from a rhythm generator, spread over the field, and a stereo
// echo. With synthVoices it has two synths of psalm as well: a pad of six note chords (the voices of
// a chord are still letting go when the next comes, so voices are taken over now and then) and a
// plucked arpeggio with a note on most steps (20 note starts a second), both through the host
func soakPatch(voices, synthVoices int) dsl.Sig {
	chords := make([][]float64, 8)
	for c := range chords {
		chords[c] = make([]float64, voices)
		for v := range chords[c] {
			chords[c][v] = float64(36 + (v*7+c*5)%48)
		}
	}
	pitch := dsl.PitchN(dsl.Loop(chords...), voices)
	amp := dsl.Adsr(0.005, 0.1, 0.6, 0.3).On(dsl.Gate(dsl.Euclid(16, 9, 0), 0.6))
	filtered := Lowpass(dsl.Saw(pitch), dsl.Lfo(0.2).Range(200, 5000), 0.5)
	mix := dsl.Gain(dsl.Spread(dsl.Mul(filtered, amp), 0.8), 1/math.Sqrt(float64(voices)))
	if synthVoices > 0 {
		progression := dsl.Loop(
			[]float64{45, 52, 57, 60, 64, 67}, []float64{41, 48, 53, 57, 60, 65},
			[]float64{36, 43, 48, 52, 55, 60}, []float64{43, 50, 55, 59, 62, 67})
		everyEighth := dsl.Loop(true, false, false, false, false, false, false, false) // a chord every 8 steps: 0.4 s
		pad := Subtractive(subtractive.PadPreset(), synthVoices, Chords(progression, everyEighth, 0.7, 6))
		arpeggio := dsl.Loop(60.0, 64.0, 67.0, 72.0, 67.0, 64.0)
		mostSteps := dsl.Loop(true, true, false, true, true, true, false, true)
		arp := Subtractive(subtractive.PluckPreset(), synthVoices, Melody(arpeggio, mostSteps, 0.8, 1))
		mix = dsl.Add(mix, dsl.Add(dsl.Gain(pad, 0.3), dsl.Gain(arp, 0.3)))
	}
	return Echo(mix, 1.0, dsl.Chans(dsl.N(0.31), dsl.N(0.42)), 0.4, 0.3)
}

// clockEngine is a device: it asks for a buffer every period of real time, on a thread of its
// own (a sound card does the same), and times what the callback took
type clockEngine struct {
	sr   float64
	size int
	cb   func(in, out [][]float32)

	stop, done chan struct{}
	took       []time.Duration // how long each callback ran
	late       []time.Duration // how long after its time the callback was called (the scheduler, not us)
	badSamples int
	peak       float64
}

func newClock(sr float64, size int, expected int) *clockEngine {
	return &clockEngine{sr: sr, size: size,
		took: make([]time.Duration, 0, expected+1024), late: make([]time.Duration, 0, expected+1024)}
}

func (e *clockEngine) GetSampleRate() float64                   { return e.sr }
func (e *clockEngine) GetBufferSize() int                       { return e.size }
func (e *clockEngine) SetCallback(cb func(in, out [][]float32)) { e.cb = cb }
func (e *clockEngine) Init() error                              { return nil }
func (e *clockEngine) Close() error                             { return nil }
func (e *clockEngine) Start() error {
	e.stop, e.done = make(chan struct{}), make(chan struct{})
	go e.run()
	return nil
}
func (e *clockEngine) Stop() error { close(e.stop); <-e.done; return nil }
func (e *clockEngine) period() time.Duration {
	return time.Duration(float64(e.size) / e.sr * float64(time.Second))
}

func (e *clockEngine) run() {
	defer close(e.done)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	period := e.period()
	bufs := [][]float32{make([]float32, e.size), make([]float32, e.size)}
	next := time.Now().Add(period)
	for {
		select {
		case <-e.stop:
			return
		default:
		}
		if wait := time.Until(next); wait > 0 {
			time.Sleep(wait)
		}
		start := time.Now()
		e.cb(nil, bufs)
		took := time.Since(start)
		e.took = append(e.took, took)
		e.late = append(e.late, max(start.Sub(next), 0))
		next = next.Add(period)
		for _, ch := range bufs { // what came out is sound, checked after the clock stopped
			for _, v := range ch {
				if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
					e.badSamples++
				}
				e.peak = math.Max(e.peak, math.Abs(float64(v)))
			}
		}
	}
}

// garbage allocates mbps MB a second and keeps some of it alive, like another part of a program
func garbage(ctx context.Context, mbps float64) {
	if mbps <= 0 {
		return
	}
	go func() {
		keep := make([][]byte, 512)
		tick := time.NewTicker(10 * time.Millisecond)
		defer tick.Stop()
		perTick := int(mbps * 1e6 * 0.010)
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				for made := 0; made < perTick; made += 32 << 10 {
					keep[i%len(keep)] = make([]byte, 32<<10)
					i++
				}
			}
		}
	}()
}

type soakResult struct {
	callbacks            int
	period               time.Duration
	p50, p99, p999, max  time.Duration
	missed               int // callbacks that took longer than the buffer lasts
	maxLate              time.Duration
	underruns            int
	gcs                  uint32
	gcPauseMax, gcPauses time.Duration
	garbageMBps          float64 // what the patch itself allocates
	heapGrowth           int64
	bad                  int
	peak                 float64
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	return sorted[min(len(sorted)-1, int(math.Ceil(p*float64(len(sorted))))-1)]
}

func runSoak(t *testing.T, voices, synthVoices, buffer int, d time.Duration, garbageMBps float64) soakResult {
	t.Helper()
	const sr = 44100.0
	eng := newClock(sr, buffer, int(d.Seconds()*sr/float64(buffer)))
	var stats dsl.PlayStats
	patch := dsl.Patch{BPM: 150, StepsPerBeat: 8, Stats: &stats} // a step every 50 ms
	out := soakPatch(voices, synthVoices)

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	garbage(ctx, garbageMBps)
	if err := patch.Play(ctx, out, eng); err != nil {
		t.Fatal(err)
	}

	runtime.ReadMemStats(&after)
	runtime.GC()
	var settled runtime.MemStats
	runtime.ReadMemStats(&settled)

	r := soakResult{callbacks: len(eng.took), period: eng.period(), underruns: stats.Underruns,
		gcs: after.NumGC - before.NumGC, gcPauses: time.Duration(after.PauseTotalNs - before.PauseTotalNs),
		garbageMBps: float64(after.TotalAlloc-before.TotalAlloc) / 1e6 / d.Seconds(),
		heapGrowth:  int64(settled.HeapAlloc) - int64(before.HeapAlloc), bad: eng.badSamples, peak: eng.peak}
	for i := uint32(0); i < min(r.gcs, 256); i++ { // the pause of each collection since the start
		r.gcPauseMax = max(r.gcPauseMax, time.Duration(after.PauseNs[(after.NumGC-1-i)%256]))
	}
	sorted := slices.Clone(eng.took)
	slices.Sort(sorted)
	r.p50, r.p99, r.p999, r.max = percentile(sorted, 0.5), percentile(sorted, 0.99), percentile(sorted, 0.999), sorted[len(sorted)-1]
	for _, took := range eng.took {
		if took > r.period {
			r.missed++
		}
	}
	for _, late := range eng.late {
		r.maxLate = max(r.maxLate, late)
	}
	return r
}

func (r soakResult) log(t *testing.T, voices, synthVoices int) {
	pct := func(d time.Duration) float64 { return 100 * float64(d) / float64(r.period) }
	t.Logf("%d voices and 2 synths of %d voices, %d callbacks of %v", voices, synthVoices, r.callbacks, r.period.Round(10*time.Microsecond))
	t.Logf("  callback time: median %v (%.2f%% of the buffer), 99%% %v, 99.9%% %v, worst %v (%.1f%%)",
		r.p50.Round(time.Microsecond), pct(r.p50), r.p99.Round(time.Microsecond), r.p999.Round(time.Microsecond),
		r.max.Round(time.Microsecond), pct(r.max))
	t.Logf("  missed deadlines (took longer than the buffer lasts): %d", r.missed)
	t.Logf("  worst lateness of the clock itself (scheduler of this machine, not the engine): %v", r.maxLate.Round(time.Microsecond))
	t.Logf("  control goroutine underruns: %d", r.underruns)
	t.Logf("  garbage collections: %d, worst pause %v, total pause %v; the patch allocates %.2f MB/s",
		r.gcs, r.gcPauseMax.Round(time.Microsecond), r.gcPauses.Round(time.Microsecond), r.garbageMBps)
	t.Logf("  heap after the run, compared with before: %+.1f MiB; peak of the sound %.3f; bad samples %d",
		float64(r.heapGrowth)/(1<<20), r.peak, r.bad)
}

func TestSoak(t *testing.T) {
	if *soakFor == 0 {
		t.Skip("a long run: use -soak 2m -timeout 0 -v (see soak_test.go)")
	}
	if *soakGCPercent != -2 {
		defer debug.SetGCPercent(debug.SetGCPercent(*soakGCPercent))
	}
	if *soakMemLimit > 0 {
		defer debug.SetMemoryLimit(debug.SetMemoryLimit(int64(*soakMemLimit) << 20))
	}
	r := runSoak(t, *soakVoices, *soakSynth, *soakBuffer, *soakFor, *soakGarbage)
	r.log(t, *soakVoices, *soakSynth)

	if r.bad > 0 || r.peak < 0.01 {
		t.Errorf("the sound is wrong: %d bad samples, peak %v", r.bad, r.peak)
	}
	if raceOn {
		t.Log("race detector on: deadlines and memory are not judged")
		return
	}
	if r.missed > 0 {
		t.Errorf("%d callbacks took longer than the buffer lasts (worst %v of %v)", r.missed, r.max, r.period)
	}
	if r.underruns > 0 {
		t.Errorf("%d steps had no values from the generators in time", r.underruns)
	}
	if r.heapGrowth > 32<<20 {
		t.Errorf("the heap grew %.1f MiB: a leak?", float64(r.heapGrowth)/(1<<20))
	}
}

// The test itself, short: the driver keeps time, the numbers are filled in, nothing is wrong with the sound
func TestSoakSmoke(t *testing.T) {
	const d = 1500 * time.Millisecond
	r := runSoak(t, 16, 4, 512, d, 0)
	r.log(t, 16, 4)
	want := int(d / r.period)
	if r.callbacks < want-8 || r.callbacks > want+2 {
		t.Errorf("%d callbacks in %v, want about %d: the clock of the device is wrong", r.callbacks, d, want)
	}
	if r.bad > 0 || r.peak < 0.01 {
		t.Errorf("the sound is wrong: %d bad samples, peak %v", r.bad, r.peak)
	}
	if r.p50 <= 0 || r.max < r.p50 {
		t.Errorf("times: median %v worst %v", r.p50, r.max)
	}
}

// The kernels allocate nothing: what the audio thread runs once the patch is built makes no garbage for
// the collector. (A patch without generators: those allocate on the control side, which is another goroutine)
func TestAudioPathDoesNotAllocate(t *testing.T) {
	if raceOn {
		t.Skip("the race detector allocates")
	}
	voices := dsl.Chans(dsl.N(110), dsl.N(165), dsl.N(220), dsl.N(330))
	filtered := Lowpass(dsl.Saw(voices), dsl.Lfo(0.2).Range(200, 5000), 0.5)
	out := Echo(dsl.Spread(filtered, 0.8), 1.0, dsl.Chans(dsl.N(0.31), dsl.N(0.42)), 0.4, 0.3)
	r := dsl.Patch{}.Start(out)
	l, rr := make([]float64, 512), make([]float64, 512)
	r.Process(l, rr) // the first call makes whatever it needs
	if n := testing.AllocsPerRun(200, func() { r.Process(l, rr) }); n != 0 {
		t.Errorf("Process allocates %v times per call", n)
	}
}
