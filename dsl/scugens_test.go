package dsl

import (
	"fmt"
	"math"
	"testing"
)

// The ports of the UGens of SuperCollider, against what SuperCollider 3.14.1 gives (sclang and scsynth, offline, 44100 Hz). The chain is
// that of the kick of "60Hz Gabber Rave 1995": a gate of 10048 samples (SuperCollider puts the n_set at 0.2286 s in the block it
// falls in), a perc envelope for the frequency of a saw, shaped, filtered. The numbers are the samples of its renderings. The saw,
// the filters and the delay are the same as SuperCollider's to better than -70 dB: the values are compared at 1e-3 (the XLine of the equalizer is at the audio rate here and steps in SuperCollider)

func scGate() Sig {
	return Custom(Spec{Name: "gate", Inputs: []Sig{N(0)}, New: func(rt *Rt) Process {
		return func(in [][]float64, out []float64, n int) {
			pos := rt.Pos()
			for i := 0; i < n; i++ {
				out[i] = 0
				if pos+int64(i) < 10048 {
					out[i] = 1
				}
			}
		}
	}})
}

var scStages = map[string]map[int]float64{
	"env":         {0: 0, 1: 0.0354688242, 2: 0.0701424181, 100: 0.973799884, 441: 0.830539167, 1000: 0.623679101, 3000: 0.102317452, 5000: 0, 8000: 0, 12000: 0, 20000: 0},
	"saw":         {0: 0.540459991, 1: 0.538558006, 2: 0.53665626, 100: 0.142425373, 441: 0.486721098, 1000: 0.0841443464, 3000: -0.201209739, 5000: -0.117655791, 8000: -0.19349429, 12000: 0.39949277, 20000: 0.525286376},
	"shape":       {0: 1.18294621, 1: 1.18370342, 2: 1.1844604, 100: 1.34140658, 441: 1.20433998, 1000: 1.36460853, 3000: -1.31800413, 5000: -1.35126758, 8000: -1.32107568, 12000: 1.23906624, 20000: 1.1889869},
	"hpf":         {0: 1.07837105, 1: 1.01190495, 2: 0.947543085, 100: -0.0196687877, 441: 1.11102414, 1000: -0.313221365, 3000: 0.0801248625, 5000: 0.236328274, 8000: -0.0246545244, 12000: -0.222767651, 20000: 1.93547976},
	"lpf":         {0: 0.000295967009, 1: 0.000765493023, 2: 0.00146338309, 100: 0.614646435, 441: 0.583729744, 1000: 0.321315169, 3000: -0.0619427972, 5000: 0.203352109, 8000: -0.74802351, 12000: -0.226648852, 20000: -1.33353794},
	"rlpf":        {0: 0.363742799, 1: 0.602507412, 2: 0.759798467, 100: 1.32316017, 441: 1.11692917, 1000: 1.35279763, 3000: -1.32026815, 5000: -1.35274351, 8000: -1.32242465, 12000: 1.23720825, 20000: 0.306169957},
	"peq":         {0: 1.20752728, 1: 1.22461414, 2: 1.24165416, 100: 2.38089418, 441: 1.11387098, 1000: 2.0451951, 3000: -5.64102602, 5000: -3.84049773, 8000: -2.8289032, 12000: 3.40052271, 20000: 3.22059751},
	"delay":       {0: 0, 1: 0, 2: 0, 100: 0, 441: -0.107720584, 1000: 1.42392075, 3000: 0.0144080166, 5000: -0.167971581, 8000: 0.00345764263, 12000: 0.538940012, 20000: -0.000356558478},
	"delay right": {0: 0, 1: 0, 2: 0, 100: 0, 441: -0.0511157922, 1000: 0.034359511, 3000: 0.138013065, 5000: -0.288303018, 8000: -0.0194956996, 12000: -0.0242950395, 20000: -0.000436965318},
}

var scKickSamples = map[int]float64{0: 0, 1: 0.00354861259, 2: 0.00830760319, 100: 0.0513851568, 441: 0.0578956977, 1000: 0.0969041064, 3000: 0.0981435552, 5000: -0.0501347408, 8000: -0.179800451, 12000: 0.00131433248, 20000: 0, 10100: -0.0214882959, 10200: -0.123275198, 11000: -0.0417617224, 14000: 0}

func scChain() (stages map[string]Sig, kick Sig) {
	gate := scGate()
	perc := EnvGen(gate, EnvPerc(0.001, 0.08, 1, -1), false)
	freq := Mul(MidiRatio(Mul(perc, 48.0)), 60.0)
	saw := SCSawSync(freq, gate)
	shape := Add(Tanh(Mul(saw, 100.0)), Mul(Add(Sign(saw), Mul(saw, -1.0)), 0.3981071705534972))
	high := SCHPF(shape, 300.0)
	lfo := Chans(Mul(Add(SCSinOsc(8.0, 0), 1.0), 0.005), Mul(Add(SCSinOsc(8.0, math.Pi/2), 1.0), 0.005))
	delayed := SCDelayC(high, 0.01, lfo)
	stages = map[string]Sig{
		"env":         perc,
		"saw":         saw,
		"shape":       shape,
		"hpf":         high,
		"lpf":         SCLPF(shape, 100.0),
		"rlpf":        SCRLPF(shape, 7000.0, 2.0),
		"peq":         SCBPeakEQ(shape, Mul(XLine(1, 0.8, 0.3), 100.0), 0.5, 15.0),
		"delay":       delayed.Ch(0),
		"delay right": delayed.Ch(1),
	}
	hi := Add(Dup(high, 2), Mul(delayed, 0.7943282347242815))
	low := Add(Dup(SCLPF(shape, 100.0), 2), hi)
	eq := SCBPeakEQ(SCRLPF(low, 7000.0, 2.0), Mul(XLine(1, 0.8, 0.3), 100.0), 0.5, 15.0)
	kick = Mul(Mul(eq, EnvGen(gate, EnvASR(0.001, 1, 0.05, -4), false)), 0.0708)
	return
}

func closeTo(got, want float64) bool { return math.Abs(got-want) <= 1e-3*math.Max(1, math.Abs(want)) }

func TestSCPortsAgainstSuperCollider(t *testing.T) {
	stages, kick := scChain()
	for name, want := range scStages {
		_, ch := Patch{SampleRate: 44100}.Render(stages[name], 1)
		for i, w := range want {
			if !closeTo(ch[0][i], w) {
				t.Errorf("%s at %d: %v, SuperCollider gives %v", name, i, ch[0][i], w)
			}
		}
	}
	_, ch := Patch{SampleRate: 44100}.Render(kick, 1)
	for i, w := range scKickSamples {
		if !closeTo(ch[0][i], w) {
			t.Errorf("the kick at %d: %v, SuperCollider gives %v", i, ch[0][i], w)
		}
	}
}

// the gate that falls is read in the block it falls in (10048 is the start of block 157)
func TestEnvGenGateAndShapes(t *testing.T) {
	gate := Custom(Spec{Name: "g", Inputs: []Sig{N(0)}, New: func(rt *Rt) Process {
		return func(in [][]float64, out []float64, n int) {
			for i := 0; i < n; i++ {
				out[i] = 0
				if p := rt.Pos() + int64(i); p >= 100 && p < 3000 {
					out[i] = 1
				}
			}
		}
	}})
	// asr of 0.01 s up (441 samples), a sustain of 1, 0.02 s down. The gate is high from 100 to 3000: it is seen to rise at 128, the first
	// block start after it, and to fall at 3008
	_, ch := Patch{SampleRate: 44100}.Render(EnvGen(gate, EnvASR(0.01, 1, 0.02, 0), false), 0.1)
	x := ch[0]
	if x[128] != 0 || x[129] == 0 || x[100] != 0 {
		t.Errorf("the rise is seen at the start of a block: %v %v %v", x[100], x[128], x[129])
	}
	if math.Abs(x[128+220]-0.5) > 0.01 || x[1500] != 1 {
		t.Errorf("half way up %v, sustain %v", x[128+220], x[1500])
	}
	if x[3007] != 1 || x[3008] != 1 || math.Abs(x[3008+441]-0.5) > 0.01 || x[3008+883] != 0 {
		t.Errorf("down: %v %v %v %v", x[3007], x[3008], x[3008+441], x[3008+883])
	}
	// the time of a segment that is not a whole number of samples is not lost: five segments of 0.0003 s (13.23 samples) make 66.15
	spec := EnvSpec{Segs: []EnvSeg{{1, 0.0003, EnvStep, 0}, {2, 0.0003, EnvStep, 0}, {3, 0.0003, EnvStep, 0}, {4, 0.0003, EnvStep, 0}, {5, 0.0003, EnvStep, 0}}, ReleaseNode: -1, LoopNode: -1}
	_, ch = Patch{SampleRate: 44100}.Render(EnvGen(N(1), spec, false), 0.01)
	steps := []int{}
	for i := 1; i < 200; i++ {
		if ch[0][i] != ch[0][i-1] {
			steps = append(steps, i)
		}
	}
	if got := fmt.Sprint(steps); got != "[13 26 39 52]" {
		t.Errorf("the segments end at %v: 13.23 samples each, the rest carried on", got)
	}
	// the control rate makes one value in 64 samples
	_, ch = Patch{SampleRate: 44100}.Render(EnvGen(N(1), EnvSpec{Segs: []EnvSeg{{1, 0.1, EnvLinear, 0}}, ReleaseNode: -1, LoopNode: -1}, true), 0.05)
	for i := range ch[0] {
		if i%64 != 0 && ch[0][i] != ch[0][i-1] {
			t.Fatalf("a control rate envelope moves inside a block at %d", i)
		}
	}
}
