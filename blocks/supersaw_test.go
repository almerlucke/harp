package blocks

import (
	"math"
	"testing"

	"harp/dsl"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/synths/supersaw"
)

// The supersaw through the host: the same voice interface as the subtractive, the same checks

func TestSupersawIsThePsalmVoice(t *testing.T) {
	cfg := supersaw.TrancePad()
	got := render2(songPatch, Supersaw(cfg, 4, Melody(dsl.Loop(60.0), pattern("x......."), 0.8, 2)), 1.5)

	var v *supersaw.Voice
	inEnv(sr, func() { v = supersaw.New(cfg) })
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
	}
	if maxDiff(got[0], got[1]) < 1e-3 {
		t.Error("left and right are the same: no stereo spread")
	}
}

func TestSupersawNoteStartsOnItsSample(t *testing.T) {
	cfg := supersaw.TrancePad()
	cfg.AmpEnv.AttackDuration = 0.1
	out := render2(songPatch, Supersaw(cfg, 4, Melody(dsl.Loop(60.0), pattern(".x......"), 1, 1)), 1)[0]
	at := stepSamples + synthChunk
	first := -1
	for i, v := range out {
		if math.Abs(v) > 1e-6 {
			first = i
			break
		}
	}
	if first < at || first > at+3 {
		t.Errorf("the sound starts at sample %d, want %d", first, at)
	}
}

func TestSupersawDoesNotDependOnTheBlocks(t *testing.T) {
	build := func() dsl.Sig {
		return Supersaw(supersaw.Hypersaw(), 3, Chords(dsl.Loop([]float64{60, 64, 67}, []float64{62, 65, 69}), pattern("x.x.xx.."), 0.9, 1.5))
	}
	const total = 3 * stepSamples
	ref := [][]float64{make([]float64, total), make([]float64, total)}
	songPatch.Start(build()).Process(ref...)
	if peak(ref[0]) < 0.01 {
		t.Fatal("silent")
	}
	for _, piece := range []int{1, 37, 100, 777} {
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

// Every note starts its saws at random phases: a chord is not the sum of its notes played alone,
// but it is louder than one note, and the same patch sounds the same every time
func TestSupersawPolyphonyAndRepeatability(t *testing.T) {
	cfg := supersaw.Pluck()
	chord := func() [][]float64 {
		return render2(songPatch, Supersaw(cfg, 3, Chords(dsl.Loop([]float64{60, 64, 67}), pattern("x..."), 1, 1)), 1)
	}
	a, b := chord(), chord()
	if maxDiff(a[0], b[0]) != 0 || maxDiff(a[1], b[1]) != 0 {
		t.Error("the same patch sounds different")
	}
	single := render2(songPatch, Supersaw(cfg, 3, Melody(dsl.Loop(60.0), pattern("x..."), 1, 1)), 1)
	if r1, r3 := rms(single[0][:stepSamples]), rms(a[0][:stepSamples]); r3 < 1.3*r1 {
		t.Errorf("rms of a chord %v is not clearly above that of one note %v", r3, r1)
	}
	// two notes in a row on one voice start from other phases: the second is not the first again
	two := render2(songPatch, Supersaw(cfg, 1, Melody(dsl.Loop(60.0), pattern("x.x."), 1, 1)), 1)[0]
	first, second := two[synthChunk:synthChunk+2000], two[2*stepSamples+synthChunk:2*stepSamples+synthChunk+2000]
	if maxDiff(first, second) < 1e-3 {
		t.Error("two notes sound sample for sample the same: the saws do not start at random phases")
	}
}

// every preset of psalm's supersaw plays through the host
func TestSupersawPresets(t *testing.T) {
	presets := map[string]supersaw.Config{
		"TrancePad": supersaw.TrancePad(), "Lead": supersaw.Lead(), "Pluck": supersaw.Pluck(), "Hoover": supersaw.Hoover(),
		"Stab": supersaw.Stab(), "UnisonBass": supersaw.UnisonBass(), "Hypersaw": supersaw.Hypersaw(), "WobbleBass": supersaw.WobbleBass(),
		"Breathing": supersaw.Breathing(), "Swarm": supersaw.Swarm(), "GlideLead": supersaw.GlideLead(), "StringMachine": supersaw.StringMachine(),
		"ChoirWash": supersaw.ChoirWash(), "EpicPad": supersaw.EpicPad(), "DarkSwarm": supersaw.DarkSwarm(), "ShimmerPad": supersaw.ShimmerPad(),
		"AnthemLead": supersaw.AnthemLead(), "ScreamingSaw": supersaw.ScreamingSaw(), "HardstyleLead": supersaw.HardstyleLead(), "ArpSaw": supersaw.ArpSaw(),
		"GatedPad": supersaw.GatedPad(), "PumpPad": supersaw.PumpPad(), "DubstepGrowl": supersaw.DubstepGrowl(), "HouseStab": supersaw.HouseStab(),
		"Dive": supersaw.Dive(), "AirPad": supersaw.AirPad(), "BrassSection": supersaw.BrassSection(), "TrioLead": supersaw.TrioLead(),
		"DropBass": supersaw.DropBass(), "ReeseBass": supersaw.ReeseBass(), "RavePluck": supersaw.RavePluck(), "AnalogPad": supersaw.AnalogPad(),
		"ExpressiveLead": supersaw.ExpressiveLead(), "MorphPad": supersaw.MorphPad(), "RiseUp": supersaw.RiseUp(), "SoftPluck": supersaw.SoftPluck(),
		"GliderBass": supersaw.GliderBass(),
	}
	for name, cfg := range presets {
		out := render2(songPatch, Supersaw(cfg, 4, Melody(dsl.Loop(48.0, 55.0), pattern("x.x."), 0.8, 2)), 3)
		for ch := range out {
			p := peak(out[ch])
			if p < 0.005 || p > 4 || math.IsNaN(p) {
				t.Errorf("%s, channel %d: peak %v", name, ch, p)
			}
		}
	}
	if len(presets) != 37 {
		t.Errorf("%d presets listed", len(presets))
	}
}

// what the audio thread does for the supersaw allocates nothing, as for the subtractive
func TestSupersawNeverAllocatesOnTheAudioThread(t *testing.T) {
	if raceOn {
		t.Skip("the race detector allocates")
	}
	for _, c := range []struct {
		name   string
		voices int
	}{{"free voices", 8}, {"every note takes over a voice", 1}} {
		rt := &dsl.Rt{SampleRate: sr, StepSamples: stepSamples}
		h := newHost(rt, func() *supersaw.Voice { return supersaw.New(supersaw.TrancePad()) }, 2, c.voices)
		note := []pendingNote{{msg: psalm.Message{"pitch": 60.0}, amp: 1, seconds: 5}, {msg: psalm.Message{"pitch": 67.0}, amp: 1, seconds: 5}}
		h.apply(note)
		var loudest float64
		for c := int64(0); c < 300; c++ {
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
