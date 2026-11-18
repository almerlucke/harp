package blocks

import (
	"math"
	"testing"

	"harp/dsl"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/synths/subtractive"
)

// a patch with steps of a quarter of a second (11025 samples), so that a step is a whole number of samples
var songPatch = dsl.Patch{SampleRate: sr, BPM: 120, StepsPerBeat: 2}

const stepSamples = 11025

// pattern is a rhythm of the steps given: x plays, anything else rests
func pattern(s string) dsl.Gen[bool] {
	steps := make([]bool, len(s))
	for i, c := range s {
		steps[i] = c == 'x'
	}
	return dsl.Loop(steps...)
}

func peak(x []float64) (p float64) {
	for _, v := range x {
		p = math.Max(p, math.Abs(v))
	}
	return
}

// The notes are what psalm's voice makes of them: the sound of the voice, in the graph, one chunk later
func TestSynthIsThePsalmVoice(t *testing.T) {
	cfg := subtractive.PadPreset()
	got := render2(songPatch, Subtractive(cfg, 4, Melody(dsl.Loop(60.0), pattern("x......."), 0.8, 2)), 1.5)

	var v *subtractive.Voice
	inEnv(sr, func() { v = subtractive.New(cfg) })
	v.Clear()
	v.Note(0.5, 0.8, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, psalm.Config{})
	var ref [2][]float64
	for len(ref[0]) < len(got[0])-synthChunk {
		if v.IsActive() {
			v.Synthesize()
		} else {
			v.Output(0).Clear()
		}
		for ch := range ref {
			for _, s := range v.Output(0).Channels[ch] {
				ref[ch] = append(ref[ch], float64(s))
			}
		}
	}
	if peak(ref[0]) < 0.01 {
		t.Fatal("the reference is silent")
	}
	for ch := range ref {
		for p := 0; p < len(got[ch])-synthChunk; p++ {
			if got[ch][p+synthChunk] != ref[ch][p] {
				t.Fatalf("channel %d, sample %d: %v, the voice gives %v", ch, p, got[ch][p+synthChunk], ref[ch][p])
			}
		}
		for p := 0; p < synthChunk; p++ { // the first chunk is made while it is played: silence
			if got[ch][p] != 0 {
				t.Fatalf("channel %d: sound %v in the first chunk", ch, got[ch][p])
			}
		}
	}
}

// render2 renders and gives the channels
func render2(p dsl.Patch, out dsl.Sig, seconds float64) [][]float64 {
	_, ch := p.Render(out, seconds)
	return ch
}

// A note starts on the sample of its step: nothing before it, and the sound starts there, one chunk later
func TestSynthNoteStartsOnItsSample(t *testing.T) {
	cfg := subtractive.PadPreset()
	cfg.AmpEnv.AttackDuration = 0.1 // ms: it sounds at once
	out := render2(songPatch, Subtractive(cfg, 4, Melody(dsl.Loop(60.0), pattern(".x......"), 1, 1)), 1)[0]
	at := stepSamples + synthChunk // the note starts at sample 11025, which is sample 17 of its chunk
	first := -1
	for i, v := range out {
		if math.Abs(v) > 1e-6 {
			first = i
			break
		}
	}
	if first < at || first > at+3 {
		t.Errorf("the sound starts at sample %d, want %d (the step, 11025, and the chunk of delay, 64)", first, at)
	}
}

// the blocks the patch is cut in, and the buffers of a device, are not heard
func TestSynthDoesNotDependOnTheBlocks(t *testing.T) {
	build := func() dsl.Sig {
		return Subtractive(subtractive.LeadPreset(), 3, Chords(dsl.Loop([]float64{60, 64, 67}, []float64{62, 65, 69}), pattern("x.x.xx.."), 0.9, 1.5))
	}
	const total = 3 * stepSamples
	ref := [][]float64{make([]float64, total), make([]float64, total)}
	songPatch.Start(build()).Process(ref...)
	if peak(ref[0]) < 0.01 {
		t.Fatal("silent")
	}
	for _, piece := range []int{1, 37, 64, 100, 777} {
		got := [][]float64{make([]float64, total), make([]float64, total)}
		r := songPatch.Start(build())
		for i := 0; i < total; i += piece {
			e := min(i+piece, total)
			r.Process(got[0][i:e], got[1][i:e])
		}
		for ch := range got {
			if d := maxDiff(got[ch], ref[ch]); d != 0 {
				t.Errorf("pieces of %d, channel %d: differ by %g", piece, ch, d)
			}
		}
	}
}

// A chord is the notes together: the sum of the notes alone
func TestSynthPolyphony(t *testing.T) {
	cfg := subtractive.PluckPreset()
	// one chord, once (a chord on every step would find the voices of the one before still letting go)
	one := func(pitch float64) [][]float64 {
		return render2(songPatch, Subtractive(cfg, 3, Melody(dsl.Loop(pitch), pattern("x..."), 1, 1)), 1)
	}
	a, b, c := one(60), one(64), one(67)
	chord := render2(songPatch, Subtractive(cfg, 3, Chords(dsl.Loop([]float64{60, 64, 67}), pattern("x..."), 1, 1)), 1)
	for ch := range chord {
		sum := make([]float64, len(chord[ch]))
		for i := range sum {
			sum[i] = a[ch][i] + b[ch][i] + c[ch][i]
		}
		if d := maxDiff(chord[ch], sum); d > 1e-5 {
			t.Errorf("channel %d: the chord differs from the notes added by %g", ch, d)
		}
		if peak(chord[ch]) < 0.01 {
			t.Error("silent chord")
		}
	}
}

// More notes than voices: the voice that has sounded longest is taken over, and nothing breaks
func TestSynthStealsTheOldestVoice(t *testing.T) {
	cfg := subtractive.PadPreset() // slow release: the voices are busy when the next note comes
	two := render2(songPatch, Subtractive(cfg, 2, Melody(dsl.Loop(60.0, 64.0, 67.0, 72.0), pattern("xxxx"), 1, 4)), 1.5)[0]
	four := render2(songPatch, Subtractive(cfg, 4, Melody(dsl.Loop(60.0, 64.0, 67.0, 72.0), pattern("xxxx"), 1, 4)), 1.5)[0]
	for _, v := range two {
		if math.IsNaN(v) || math.Abs(v) > 4 {
			t.Fatalf("sample %v", v)
		}
	}
	// the first two notes have a voice each: the same until the third note comes (step 2, sample 22050, plus the chunk)
	if d := maxDiff(two[:2*stepSamples], four[:2*stepSamples]); d != 0 {
		t.Errorf("before the third note the two differ by %g", d)
	}
	if d := maxDiff(two[2*stepSamples:], four[2*stepSamples:]); d < 1e-3 {
		t.Error("with two voices the third note should take one of the first two: it sounds the same as with four")
	}
	// the fade of a stolen voice keeps the cut from being a step: the difference of two samples next to each other stays small
	var step float64
	for i := 2 * stepSamples; i < 2*stepSamples+500; i++ {
		step = math.Max(step, math.Abs(two[i+1]-two[i]))
	}
	if step > 0.2 {
		t.Errorf("a jump of %v between two samples where the voice was taken", step)
	}
}

func TestSynthLetsGo(t *testing.T) {
	cfg := subtractive.PluckPreset()
	out := render2(songPatch, Subtractive(cfg, 2, Melody(dsl.Loop(60.0), pattern("x"+"............"), 1, 1)), 3)[0]
	if peak(out[:stepSamples]) < 0.01 {
		t.Fatal("silent")
	}
	if p := peak(out[2*stepSamples:]); p > 1e-4 { // the key is let go after a step, and the release is a fraction of a second
		t.Errorf("still sounding at 0.5 s after the note, peak %v", p)
	}
}

func TestSynthIsRepeatable(t *testing.T) {
	build := func() dsl.Sig {
		return Subtractive(subtractive.StringsPreset(), 4, Chords(dsl.Loop([]float64{48, 55, 64}), pattern("x..x"), 0.7, 2))
	}
	a, b := render2(songPatch, build(), 2), render2(songPatch, build(), 2)
	if maxDiff(a[0], b[0]) != 0 || maxDiff(a[1], b[1]) != 0 {
		t.Error("the same patch sounds different")
	}
}

// Several presets of psalm play through the host
func TestSynthPresets(t *testing.T) {
	for name, cfg := range map[string]subtractive.Config{
		"bass": subtractive.BassPreset(), "lead": subtractive.LeadPreset(), "pluck": subtractive.PluckPreset(),
		"pad": subtractive.PadPreset(), "strings": subtractive.StringsPreset(), "brass": subtractive.BrassPreset(),
	} {
		out := render2(songPatch, Subtractive(cfg, 4, Melody(dsl.Loop(48.0, 55.0), pattern("x.x."), 0.8, 1.5)), 2)
		for ch := range out {
			p := peak(out[ch])
			if p < 0.01 || p > 3 || math.IsNaN(p) {
				t.Errorf("%s, channel %d: peak %v", name, ch, p)
			}
		}
	}
	// a mono voice is one channel
	mono := subtractive.LeadPreset()
	mono.NumChannels = 1
	if got := Subtractive(mono, 2, Melody(dsl.Loop(60.0), pattern("x"), 1, 1)).Channels(); got != 1 {
		t.Errorf("a mono synth has %d channels", got)
	}
	if got := Subtractive(subtractive.LeadPreset(), 2, Melody(dsl.Loop(60.0), pattern("x"), 1, 1)).Channels(); got != 2 {
		t.Errorf("a stereo synth has %d channels", got)
	}
}

func TestMelodyAndChordsGenerators(t *testing.T) {
	m := Melody(dsl.Loop(60.0, 62.0, 64.0), pattern("x.x"), 0.5, 2)
	var got []float64
	for i := 0; i < 6; i++ {
		if ns := m.Generate(); len(ns) > 0 {
			if ns[0].Velocity != 0.5 || ns[0].Length != 2 {
				t.Fatalf("note %+v", ns[0])
			}
			got = append(got, ns[0].Pitch)
		} else {
			got = append(got, -1)
		}
	}
	// the rhythm x.x repeats every 3 steps, and a pitch is taken only when it plays: 60, rest, 62, 64, rest, 60
	want := []float64{60, -1, 62, 64, -1, 60}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("melody: %v, want %v", got, want)
		}
	}
	m.Reset()
	if ns := m.Generate(); len(ns) != 1 || ns[0].Pitch != 60 {
		t.Errorf("after Reset: %v", ns)
	}
	c := Chords(dsl.Loop([]float64{60, 64}, []float64{62}), pattern("x"), 1, 1)
	if ns := c.Generate(); len(ns) != 2 || ns[1].Pitch != 64 {
		t.Errorf("chord: %v", ns)
	}
	if ns := c.Generate(); len(ns) != 1 || ns[0].Pitch != 62 {
		t.Errorf("next chord: %v", ns)
	}
}

// What the audio thread does for the synth allocates nothing: making the sound of a chunk, starting
// a note, and taking over a voice that sounds. (Starting a note used to allocate a dozen strings in
// psalm's modulation matrix, one for every key it looked for in the message: fixed there, and this
// test is what keeps it so)
func TestSynthNeverAllocatesOnTheAudioThread(t *testing.T) {
	if raceOn {
		t.Skip("the race detector allocates")
	}
	for _, c := range []struct {
		name   string
		voices int
	}{{"free voices", 8}, {"every note takes over a voice", 1}} {
		rt := &dsl.Rt{SampleRate: sr, StepSamples: stepSamples}
		h := newHost(rt, func() *subtractive.Voice { return subtractive.New(subtractive.StringsPreset()) }, 2, c.voices)
		note := []pendingNote{{msg: psalm.Message{"pitch": 60.0}, amp: 1, seconds: 5}, {msg: psalm.Message{"pitch": 67.0}, amp: 1, seconds: 5}}
		h.apply(note)
		var loudest float64
		for c := int64(0); c < 300; c++ { // 0.4 s: past the attack of the strings
			h.render(c)
			loudest = max(loudest, peak(float32sTo64(h.ready[0])))
		}
		if loudest < 0.01 {
			t.Fatalf("%s: silent", c.name)
		}
		if n := testing.AllocsPerRun(200, func() { h.render(301) }); n != 0 {
			t.Errorf("%s: making a chunk allocates %v times", c.name, n)
		}
		if n := testing.AllocsPerRun(200, func() { h.apply(note); h.render(301) }); n != 0 {
			t.Errorf("%s: starting two notes and making a chunk allocates %v times", c.name, n)
		}
	}
}

func float32sTo64(x []float32) []float64 {
	out := make([]float64, len(x))
	for i, v := range x {
		out[i] = float64(v)
	}
	return out
}
