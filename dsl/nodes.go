package dsl

import (
	"fmt"
	"math"
)

// A node that takes signals as input is made for every channel: this is multichannel expansion.
// The signals must have the same number of channels, or one channel, which goes to every channel of
// the others. So Lowpass(stereo, 800) is two filters, each with a state of its own, on the same
// cutoff; Mul(stereo, stereo) multiplies left by left and right by right.

// width is the number of channels of a node that takes these signals: the largest, and every one
// must have that many or one
func width(who string, ins ...Sig) int {
	n := 1
	for _, s := range ins {
		if len(s.ch) == 0 {
			panic(fmt.Sprintf("dsl: %s: a signal without channels (a Sig that was never made)", who))
		}
		n = max(n, len(s.ch))
	}
	for _, s := range ins {
		if len(s.ch) != 1 && len(s.ch) != n {
			panic(fmt.Sprintf("dsl: %s: cannot combine signals of %d and %d channels (one channel combines with any number; for others use As, Fold or Spread)",
				who, len(s.ch), n))
		}
	}
	return n
}

// at is channel i of the signal, a mono signal: a signal of one channel is every channel
func (s Sig) at(i int) Sig {
	if len(s.ch) == 1 {
		i = 0
	}
	return Sig{ch: s.ch[i : i+1 : i+1]}
}

// expand makes a node for every channel of the signals: mk gets the input nodes of one channel
func expand(who string, ins []Sig, mk func(in []*node) *node) Sig {
	n := width(who, ins...)
	out := Sig{ch: make1(n)}
	for c := 0; c < n; c++ {
		in := make1(len(ins))
		for i, s := range ins {
			in[i] = s.at(c).ch[0]
		}
		out.ch[c] = mk(in)
	}
	return out
}

func make1(n int) []*node { return make([]*node, n) }

func unary(name, key string, a Sig, build func(rt *Rt, in, out []float64) Kernel) Sig {
	return expand(name, []Sig{a}, func(in []*node) *node {
		return &node{name: name, key: key, in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			return build(rt, b[0], out)
		}}
	})
}

// Mul multiplies two signals
func Mul[A, B Param](a A, b B) Sig {
	return expand("Mul", []Sig{sig(a), sig(b)}, func(in []*node) *node { return &node{name: "mul", op: opMul, in: in} })
}

// Add adds two signals
func Add[A, B Param](a A, b B) Sig {
	return expand("Add", []Sig{sig(a), sig(b)}, func(in []*node) *node { return &node{name: "add", op: opAdd, in: in} })
}

// Affine is x*a + b
func Affine(x Sig, a, b float64) Sig {
	return expand("Affine", []Sig{x}, func(in []*node) *node { return &node{name: "affine", op: opAffine, in: in, a: a, b: b} })
}

// Mul, Add and Range are also methods, for chains. Methods can't be generic, so they take a Sig: use N for a constant

func (s Sig) Mul(o Sig) Sig { return Mul(s, o) }
func (s Sig) Add(o Sig) Sig { return Add(s, o) }

// Range maps a signal in [-1, 1] (an Lfo) to [lo, hi]
func (s Sig) Range(lo, hi float64) Sig { return Affine(s, (hi-lo)/2, (hi+lo)/2) }

// Sine is a sine oscillator, freq in Hz. A frequency of several channels is several oscillators
func Sine[F Param](freq F) Sig {
	build := func(rt *Rt, in [][]float64, out []float64, a, b float64) Kernel {
		freq := in[0]
		var phase float64
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = math.Sin(2*math.Pi*phase)*a + b
				phase += freq[i] / rt.SampleRate
				phase -= math.Floor(phase)
			}
		}
	}
	return expand("Sine", []Sig{sig(freq)}, func(in []*node) *node {
		return &node{name: "sine", key: "sine", in: in,
			build:     func(rt *Rt, in [][]float64, out []float64) Kernel { return build(rt, in, out, 1, 0) },
			buildPost: build}
	})
}

// Lfo is a sine for modulation, [-1, 1], rate in Hz. Use Range to map it
func Lfo[F Param](rate F) Sig { return Sine(rate) }

// Saw is a band limited (PolyBLEP) sawtooth, freq in Hz: no aliasing from the corner. A frequency
// of several channels is several oscillators
func Saw[F Param](freq F) Sig {
	build := func(rt *Rt, in [][]float64, out []float64, a, b float64) Kernel {
		freq := in[0]
		var phase float64
		return func(n int) {
			for i := 0; i < n; i++ {
				dt := freq[i] / rt.SampleRate
				out[i] = (2*phase-1-polyBLEP(phase, dt))*a + b
				phase += dt
				phase -= math.Floor(phase)
			}
		}
	}
	return expand("Saw", []Sig{sig(freq)}, func(in []*node) *node {
		return &node{name: "saw", key: "saw", in: in,
			build:     func(rt *Rt, in [][]float64, out []float64) Kernel { return build(rt, in, out, 1, 0) },
			buildPost: build}
	})
}

func polyBLEP(t, dt float64) float64 {
	switch {
	case t < dt:
		t /= dt
		return t + t - t*t - 1
	case t > 1-dt:
		t = (t - 1) / dt
		return t*t + t + t + 1
	}
	return 0
}

// Lowpass is a resonant 12 dB lowpass (a TPT state variable filter). Cutoff in Hz, q is the
// resonance (0.707 is flat). Cutoff can be a signal: it is read every sample. Every channel has a
// filter of its own
func Lowpass[C, Q Param](in Sig, cutoff C, q Q) Sig {
	return expand("Lowpass", []Sig{in, sig(cutoff), sig(q)}, func(ins []*node) *node {
		return &node{name: "lowpass", key: "lowpass", in: ins, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			x, fc, qq := b[0], b[1], b[2]
			var ic1, ic2 float64
			maxFc := 0.45 * rt.SampleRate
			return func(n int) {
				for i := 0; i < n; i++ {
					c := math.Min(math.Max(fc[i], 20), maxFc)
					g := math.Tan(math.Pi * c / rt.SampleRate)
					k := 1 / math.Max(qq[i], 0.05)
					a1 := 1 / (1 + g*(g+k))
					a2 := g * a1
					a3 := g * a2
					v3 := x[i] - ic2
					v1 := a1*ic1 + a2*v3
					v2 := ic2 + a2*ic1 + a3*v3
					ic1 = 2*v1 - ic1
					ic2 = 2*v2 - ic2
					out[i] = v2
				}
			}
		}}
	})
}

// Lowpass as a method, with signals for cutoff and q (use N for a constant)
func (s Sig) Lowpass(cutoff, q Sig) Sig { return Lowpass(s, cutoff, q) }

// Env is an ADSR envelope, times in seconds. Give it a gate with On
type Env struct{ A, D, S, R float64 }

// Adsr describes an envelope. Sustain is a level in [0, 1]
func Adsr(a, d, s, r float64) Env { return Env{a, d, s, r} }

// On runs the envelope on a gate: rising edge starts it, falling edge releases it from where it is.
// A gate of several channels is several envelopes
func (e Env) On(gate Sig) Sig {
	return unary("adsr", fmt.Sprintf("adsr %v %v %v %v", e.A, e.D, e.S, e.R), gate, func(rt *Rt, g, out []float64) Kernel {
		const (
			idle = iota
			attack
			decay
			sustain
			release
		)
		var state int
		var level, relStep float64
		var prev float64
		sr := rt.SampleRate
		step := func(t float64) float64 { return 1 / math.Max(t*sr, 1) }
		return func(n int) {
			for i := 0; i < n; i++ {
				if g[i] > 0.5 && prev <= 0.5 {
					state = attack
				} else if g[i] <= 0.5 && prev > 0.5 && state != idle {
					state, relStep = release, level*step(e.R)
				}
				prev = g[i]
				switch state {
				case attack:
					level += step(e.A)
					if level >= 1 {
						level, state = 1, decay
					}
				case decay:
					level -= (1 - e.S) * step(e.D)
					if level <= e.S {
						level, state = e.S, sustain
					}
				case release:
					level -= relStep
					if level <= 0 {
						level, state = 0, idle
					}
				}
				out[i] = level
			}
		}
	})
}

// Pulse is a band limited (PolyBLEP) pulse wave: it is up (+1) for the first width of every period and
// down (-1) for the rest, freq in Hz and width from 0 to 1 (0.5 is a square), both can be signals. With a
// width other than 0.5 the wave has a mean of 2*width-1, so a pulse of 0.3 is on average at -0.4 (psalm's
// rectangle, in blocks, has no such offset).
//
// It is not SuperCollider's Pulse, which measured in SuperCollider 3.14.1 is half the size (a pulse of 1 from
// top to bottom), has a mean of 0, and has the part of the width at the bottom: the same patch
// is written with (Pulse + (1 - 2 width)) * -0.5. A frequency of several channels is several oscillators
func Pulse[F, W Param](freq F, width W) Sig {
	return expand("Pulse", []Sig{sig(freq), sig(width)}, func(in []*node) *node {
		return &node{name: "pulse", key: "pulse", in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			freq, width := b[0], b[1]
			var phase float64
			return func(n int) {
				for i := 0; i < n; i++ {
					dt := freq[i] / rt.SampleRate
					w := math.Min(math.Max(width[i], 0), 1)
					v := -1.0
					if phase < w {
						v = 1
					}
					// a rising edge at the start of the period and a falling one at the width, each
					// smoothed by the correction of PolyBLEP (as in Saw); with width 0 or 1 the two
					// are one place and cancel, and the wave is a constant
					v += polyBLEP(phase, dt)
					v -= polyBLEP(math.Mod(phase+1-w, 1), dt)
					out[i] = v
					phase += dt
					phase -= math.Floor(phase)
				}
			}
		}}
	})
}
