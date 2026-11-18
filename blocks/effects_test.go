package blocks

import (
	"math"
	"testing"

	"harp/dsl"
)

// noise is a repeatable noise for the tests
func noiseSig(n int, gain float64) []float64 {
	x := make([]float64, n)
	s := uint64(88172645463325252)
	for i := range x {
		s ^= s << 13
		s ^= s >> 7
		s ^= s << 17
		x[i] = gain * (float64(s%2000001)/1000000 - 1)
	}
	return x
}

// feed is a signal that plays the samples that it is given (a custom node that reads them from the position of the stream)
func feed(samples []float64) dsl.Sig {
	return dsl.Custom(dsl.Spec{Name: "feed", Inputs: []dsl.Sig{dsl.N(0)}, New: func(rt *dsl.Rt) dsl.Process {
		return func(in [][]float64, out []float64, n int) {
			pos := int(rt.Pos())
			for i := 0; i < n; i++ {
				if pos+i < len(samples) {
					out[i] = samples[pos+i]
				} else {
					out[i] = 0
				}
			}
		}
	}})
}

func inputsFor(info EffectInfo, n int) []dsl.Sig {
	ins := make([]dsl.Sig, len(info.Inputs))
	for k := range ins {
		ins[k] = feed(noiseSig(n, 0.3))
		if k == 0 && len(info.Inputs) == 2 && info.Inputs[0] == mono { // a vocoder: the carrier is a tone
			tone := make([]float64, n)
			for i := range tone {
				tone[i] = 0.3 * math.Sin(2*math.Pi*110*float64(i)/44100)
			}
			ins[k] = feed(tone)
		}
	}
	return ins
}

func renderOut(out dsl.Sig, n int) [][]float64 {
	_, ch := dsl.Patch{SampleRate: 44100}.Render(out, float64(n)/44100)
	return ch
}

func TestEveryEffectMakesSound(t *testing.T) {
	const n = 44100
	for _, info := range Effects() {
		t.Run(info.Name, func(t *testing.T) {
			set := info.Describe()
			if len(set.Params) == 0 {
				t.Fatal("the effect describes no settings")
			}
			ch := renderOut(Effect(info, inputsFor(info, n), nil, nil), n)
			if len(ch) != 2 {
				t.Fatalf("%d channels", len(ch))
			}
			var energy float64
			for c := range ch {
				for i, v := range ch[c] {
					if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 20 {
						t.Fatalf("channel %d sample %d is %v", c, i, v)
					}
					energy += v * v
				}
			}
			if energy < 1e-6 {
				t.Errorf("silent (energy %g)", energy)
			}
		})
	}
}

// an effect that works on as many samples as it is given gives the same sound whatever the size of the pieces
func TestEffectsDoNotDependOnThePiecesTheyAreGiven(t *testing.T) {
	const n = 8192
	for _, info := range Effects() {
		if info.Blocks {
			continue
		}
		t.Run(info.Name, func(t *testing.T) {
			whole := renderOut(Effect(info, inputsFor(info, n), nil, nil), n)
			// the same, in pieces of odd sizes: a runner that is given buffers of 37 samples
			prog := dsl.Patch{SampleRate: 44100}.Start(Effect(info, inputsFor(info, n), nil, nil))
			got := [][]float64{make([]float64, n), make([]float64, n)}
			for at := 0; at < n; {
				size := 1 + (at*7+3)%61
				size = min(size, n-at)
				prog.Process(got[0][at:at+size], got[1][at:at+size])
				at += size
			}
			for c := range whole {
				for i := range whole[c] {
					if math.Abs(whole[c][i]-got[c][i]) > 1e-5 {
						t.Fatalf("channel %d sample %d: %v in pieces of 64, %v in odd pieces: the effect works on blocks (say so in the table)", c, i, whole[c][i], got[c][i])
					}
				}
			}
		})
	}
}

func rmsOf(x []float64) float64 {
	var s float64
	for _, v := range x {
		s += v * v
	}
	return math.Sqrt(s / float64(len(x)))
}

// a setting that is a number is sent when the effect is made: a chorus that is all dry gives its input
func TestAFixedSettingIsInTheEffect(t *testing.T) {
	const n = 44100
	info := effects["chorus"]
	in := noiseSig(n, 0.3)
	dry := renderOut(Effect(info, []dsl.Sig{feed(in)}, Settings{"mix": 0}, nil), n)
	for i := 256; i < n; i++ {
		if math.Abs(dry[0][i]-in[i]) > 1e-4 {
			t.Fatalf("a chorus with mix 0 is not its input at %d: %v and %v", i, dry[0][i], in[i])
		}
	}
	wet := renderOut(Effect(info, []dsl.Sig{feed(in)}, Settings{"mix": 1}, nil), n)
	if rmsOf(wet[0][256:]) == 0 || math.Abs(wet[0][3000]-in[3000]) < 1e-3 {
		t.Error("a chorus with mix 1 is its input")
	}
}

// a setting that is a signal moves the effect: the tremolo gets deeper
func TestASettingThatIsASignalMovesTheEffect(t *testing.T) {
	const n = 44100
	info := effects["tremolo"]
	tone := make([]float64, n)
	for i := range tone {
		tone[i] = 0.5
	}
	// a depth of 0 for the first half second, then 1: the level is steady, then it goes to silence in the cycles of 8 Hz
	depth := dsl.Custom(dsl.Spec{Name: "depth", Inputs: []dsl.Sig{dsl.N(0)}, New: func(rt *dsl.Rt) dsl.Process {
		return func(in [][]float64, out []float64, k int) {
			for i := 0; i < k; i++ {
				if rt.Pos()+int64(i) < n/2 {
					out[i] = 0
				} else {
					out[i] = 1
				}
			}
		}
	}})
	ch := renderOut(Effect(info, []dsl.Sig{feed(tone)}, Settings{"rate": 8}, map[string]dsl.Sig{"depth": depth}), n)
	steady := ch[0][2000 : n/2-2000]
	moving := ch[0][n/2+2000 : n-2000]
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range steady {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	if hi-lo > 1e-3 {
		t.Errorf("with a depth of 0 the level moves from %v to %v", lo, hi)
	}
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, v := range moving {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	if hi-lo < 0.2 {
		t.Errorf("with a depth of 1 the level moves from %v to %v only", lo, hi)
	}
}

// the sidechain: with a signal in it the compressor listens to that, and without one to its own input
func TestACompressorListensToItsSidechain(t *testing.T) {
	const n = 22050
	info := effects["compressor"]
	steady := make([]float64, n)
	loud := make([]float64, n)
	for i := range steady {
		steady[i] = 0.05 * math.Sin(2*math.Pi*200*float64(i)/44100)
		loud[i] = 0.9 * math.Sin(2*math.Pi*60*float64(i)/44100)
	}
	set := Settings{"threshold": -30, "ratio": 20, "attack": 1, "release": 50, "mix": 1}
	plain := renderOut(Effect(info, []dsl.Sig{feed(steady), {}}, set, nil), n)
	ducked := renderOut(Effect(info, []dsl.Sig{feed(steady), feed(loud)}, set, nil), n)
	a, b := rmsOf(plain[0][n/2:]), rmsOf(ducked[0][n/2:])
	// a quiet tone is under the threshold by itself (-26 dB rms against -30...): the loud sidechain turns it down a lot more
	if b > a*0.5 {
		t.Errorf("a loud sidechain turned the signal from %v to %v only", a, b)
	}
}
