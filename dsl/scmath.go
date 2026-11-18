package dsl

import "math"

func mapSig(name string, x Sig, f func(float64) float64) Sig {
	return expand(name, []Sig{x}, func(in []*node) *node {
		return &node{name: name, key: name, in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			x := b[0]
			return func(n int) {
				for i := 0; i < n; i++ {
					out[i] = f(x[i])
				}
			}
		}}
	})
}

// Tanh, Atan: the shaping functions
func Tanh(x Sig) Sig { return mapSig("tanh", x, math.Tanh) }
func Atan(x Sig) Sig { return mapSig("atan", x, math.Atan) }

// Sign is -1, 0 or 1, as the sign of the signal
func Sign(x Sig) Sig {
	return mapSig("sign", x, func(v float64) float64 {
		switch {
		case v > 0:
			return 1
		case v < 0:
			return -1
		}
		return 0
	})
}

// MidiRatio is the ratio of the frequencies a number of semitones apart: 2^(x/12). 12 is 2, -12 is a half
func MidiRatio(x Sig) Sig {
	return mapSig("midiratio", x, func(v float64) float64 { return math.Exp2(v / 12) })
}

// SinePhase is Sine that starts a part of a cycle on (0.25 is a cosine)
func SinePhase[F Param](freq F, cycles float64) Sig {
	return expand("SinePhase", []Sig{sig(freq)}, func(in []*node) *node {
		return &node{name: "sine.phase", key: "sine.phase " + formatFloats(cycles), in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			freq, phase := b[0], cycles
			return func(n int) {
				for i := 0; i < n; i++ {
					out[i] = math.Sin(2 * math.Pi * phase)
					phase += freq[i] / rt.SampleRate
					phase -= math.Floor(phase)
				}
			}
		}}
	})
}
