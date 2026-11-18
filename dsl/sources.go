package dsl

import (
	"fmt"
	"math"
	"strconv"
)

// Sources and shapers of signals that are not oscillators of a fixed wave: noise, impulses, decays, a ramp.
// They follow the definitions of SuperCollider's UGens of the same kind (WhiteNoise, LFNoise0/1,
// Impulse, Dust, Decay, Decay2, LFSaw, midicps), so that a patch written for it means the same here.
//
// A source of random numbers draws its seed from genny when it is made, so seed = n (Seed) makes a
// patch sound the same every time, and two sources made one after the other are not the same noise.
// They are never merged for that reason.

// rng is a small fast random number generator (xorshift64*), one for every source
type rng struct{ s uint64 }

func seed64() uint64 {
	return uint64(buildIntn(1<<31))<<32 | uint64(buildIntn(1<<31)) | 1
}

func (r *rng) next() uint64 {
	r.s ^= r.s >> 12
	r.s ^= r.s << 25
	r.s ^= r.s >> 27
	return r.s * 2685821657736338717
}

func (r *rng) float() float64   { return float64(r.next()>>11) / (1 << 53) } // [0, 1)
func (r *rng) bipolar() float64 { return r.float()*2 - 1 }                   // [-1, 1)

// Noise is white noise, evenly spread from -1 to 1
func Noise() Sig {
	seed := seed64()
	return mono(&node{name: "noise", id: "noise", build: func(rt *Rt, _ [][]float64, out []float64) Kernel {
		r := rng{seed}
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = r.bipolar()
			}
		}
	}})
}

// LFNoise is low frequency noise: a new random value from -1 to 1 freq times a second. With interpolate
// it moves in a straight line from one to the next (LFNoise1), without it steps (LFNoise0). Every
// channel of freq has noise of its own
func LFNoise[F Param](freq F, interpolate bool) Sig {
	return expand("LFNoise", []Sig{sig(freq)}, func(in []*node) *node {
		seed := seed64()
		return &node{name: "lfnoise", id: fmt.Sprintf("lfnoise %v", interpolate), in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			r, f := rng{seed}, b[0]
			prev, next := r.bipolar(), r.bipolar()
			var phase float64
			return func(n int) {
				for i := 0; i < n; i++ {
					phase += f[i] / rt.SampleRate
					for phase >= 1 {
						phase--
						prev, next = next, r.bipolar()
					}
					if interpolate {
						out[i] = prev + (next-prev)*phase
					} else {
						out[i] = prev
					}
				}
			}
		}}
	})
}

// Impulse is a 1 at the start and then every 1/freq seconds, and 0 in between
func Impulse[F Param](freq F) Sig {
	return expand("Impulse", []Sig{sig(freq)}, func(in []*node) *node {
		return &node{name: "impulse", key: "impulse", in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			f, phase := b[0], 1.0 // the first sample is an impulse
			return func(n int) {
				for i := 0; i < n; i++ {
					// as SuperCollider's: look at the phase, and then move it, so that the period is
					// exactly sr/freq samples
					if phase >= 1 {
						phase--
						out[i] = 1
					} else {
						out[i] = 0
					}
					phase += f[i] / rt.SampleRate
				}
			}
		}}
	})
}

// Dust is random impulses, density of them a second on average, each of a value from 0 to 1
func Dust[D Param](density D) Sig {
	return expand("Dust", []Sig{sig(density)}, func(in []*node) *node {
		seed := seed64()
		return &node{name: "dust", id: "dust", in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			r, d := rng{seed}, b[0]
			return func(n int) {
				for i := 0; i < n; i++ {
					out[i] = 0
					if r.float() < d[i]/rt.SampleRate {
						out[i] = r.float()
					}
				}
			}
		}}
	})
}

// decayCoef is the factor per sample that makes a level fall 60 dB (to a thousandth) in seconds
func decayCoef(seconds, sampleRate float64) float64 {
	if seconds <= 0 {
		return 0
	}
	return math.Exp(math.Log(0.001) / (seconds * sampleRate))
}

// Decay makes every input an exponential decay that falls 60 dB in seconds: an impulse gives the
// decaying level, and a train of them gives a level that is hit again and again
func Decay[T Param](x Sig, seconds T) Sig {
	return expand("Decay", []Sig{x, sig(seconds)}, func(in []*node) *node {
		return &node{name: "decay", key: "decay", in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			in, secs := b[0], b[1]
			var y, last, coef float64
			last = -1
			return func(n int) {
				for i := 0; i < n; i++ {
					if secs[i] != last {
						last, coef = secs[i], decayCoef(secs[i], rt.SampleRate)
					}
					y = in[i] + coef*y
					out[i] = y
				}
			}
		}}
	})
}

// Decay2 is Decay with an attack: the difference of two decays, one that falls in decay seconds and
// one that falls in attack seconds. An impulse gives a level that rises in about attack and then falls
func Decay2[A, D Param](x Sig, attack A, decay D) Sig {
	return expand("Decay2", []Sig{x, sig(attack), sig(decay)}, func(in []*node) *node {
		return &node{name: "decay2", key: "decay2", in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			in, att, dec := b[0], b[1], b[2]
			var yd, ya, lastA, lastD, coefA, coefD float64
			lastA, lastD = -1, -1
			return func(n int) {
				for i := 0; i < n; i++ {
					if att[i] != lastA {
						lastA, coefA = att[i], decayCoef(att[i], rt.SampleRate)
					}
					if dec[i] != lastD {
						lastD, coefD = dec[i], decayCoef(dec[i], rt.SampleRate)
					}
					yd = in[i] + coefD*yd
					ya = in[i] + coefA*ya
					out[i] = yd - ya
				}
			}
		}}
	})
}

// Ramp is a rising ramp from -1 to 1, freq times a second, that jumps back to -1 (SuperCollider's LFSaw:
// it is not band limited, which is what is wanted for a modulation, and starts at 0)
func Ramp[F Param](freq F) Sig {
	return expand("Ramp", []Sig{sig(freq)}, func(in []*node) *node {
		return &node{name: "ramp", key: "ramp", in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			f := b[0]
			var phase float64
			return func(n int) {
				for i := 0; i < n; i++ {
					out[i] = phase
					phase += 2 * f[i] / rt.SampleRate
					if phase >= 1 {
						phase -= 2
					} else if phase < -1 {
						phase += 2
					}
				}
			}
		}}
	})
}

// Hz turns MIDI notes into frequencies (midicps): 69 is 440 Hz, and 12 more is twice as much. A
// signal of notes (a sweep) is a signal of frequencies
func Hz[N Param](note N) Sig {
	return unary("hz", "hz", sig(note), func(_ *Rt, in, out []float64) Kernel {
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = 440 * math.Exp2((in[i]-69)/12)
			}
		}
	})
}

// Line goes from start to end in seconds, in a straight line, and then stays at end (SuperCollider's Line)
func Line(start, end, seconds float64) Sig {
	return mono(&node{name: "line", key: lineKey("line", start, end, seconds), build: func(rt *Rt, _ [][]float64, out []float64) Kernel {
		var t float64
		total := math.Max(seconds*rt.SampleRate, 1)
		return func(n int) {
			for i := 0; i < n; i++ {
				if t >= total {
					out[i] = end // and it stays exactly there
				} else {
					out[i] = start + (end-start)*(t/total)
				}
				t++
			}
		}
	}})
}

// XLine goes from start to end in seconds by the same ratio every second (an exponential line,
// the way pitch is heard to move), and then stays at end. Start and end can't be 0 or have different signs
func XLine(start, end, seconds float64) Sig {
	if start == 0 || end == 0 || (start < 0) != (end < 0) {
		panic("dsl: XLine: start and end can't be 0 and must have the same sign")
	}
	return mono(&node{name: "xline", key: lineKey("xline", start, end, seconds), build: func(rt *Rt, _ [][]float64, out []float64) Kernel {
		var t float64
		total := math.Max(seconds*rt.SampleRate, 1)
		ratio := math.Log2(end / start)
		return func(n int) {
			for i := 0; i < n; i++ {
				if t >= total {
					out[i] = end
				} else {
					out[i] = start * math.Exp2(ratio*(t/total))
				}
				t++
			}
		}
	}})
}

func lineKey(kind string, a, b, c float64) string {
	return kind + " " + formatFloats(a, b, c)
}

// ExpRange maps a signal from -1..1 to lo..hi along an exponential curve: the middle is the geometric mean,
// so a sweep of an lfo is even in the way pitch is heard (SuperCollider's exprange). lo and hi
// can't be 0 or have different signs
func (s Sig) ExpRange(lo, hi float64) Sig {
	if lo == 0 || hi == 0 || (lo < 0) != (hi < 0) {
		panic("dsl: ExpRange: lo and hi can't be 0 and must have the same sign")
	}
	ratio := math.Log2(hi / lo)
	return unary("exprange", "exprange "+formatFloats(lo, hi), s, func(_ *Rt, in, out []float64) Kernel {
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = lo * math.Exp2(ratio*(in[i]+1)/2)
			}
		}
	})
}

func formatFloats(xs ...float64) string {
	s := ""
	for _, x := range xs {
		s += strconv.FormatFloat(x, 'g', -1, 64) + " "
	}
	return s
}

// LinExp maps a signal from the linear range inLo..inHi to the exponential range lo..hi, which can be signals: the output is
// lo * (hi/lo)^((x - inLo) / (inHi - inLo)). The input is not clipped (a value beyond the range goes on along the curve), as in
// SuperCollider's LinExp; lo and hi can't be 0 or have different signs: the output is 0 there. A reversed output range
// (lo above hi) goes down
func LinExp[L, H Param](x Sig, inLo, inHi float64, lo L, hi H) Sig {
	span := inHi - inLo
	return expand("LinExp", []Sig{x, sig(lo), sig(hi)}, func(in []*node) *node {
		return &node{name: "linexp", key: "linexp " + formatFloats(inLo, inHi), in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			x, lo, hi := b[0], b[1], b[2]
			return func(n int) {
				for i := 0; i < n; i++ {
					l, h := lo[i], hi[i]
					if l == 0 || h == 0 || (l < 0) != (h < 0) || span == 0 {
						out[i] = 0
						continue
					}
					out[i] = l * math.Exp2(math.Log2(h/l)*(x[i]-inLo)/span)
				}
			}
		}}
	})
}

// Clip2 limits a signal to the range -level..level
func Clip2[L Param](x Sig, level L) Sig {
	return expand("Clip2", []Sig{x, sig(level)}, func(in []*node) *node {
		return &node{name: "clip2", key: "clip2", in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			x, lv := b[0], b[1]
			return func(n int) {
				for i := 0; i < n; i++ {
					l := math.Abs(lv[i])
					out[i] = math.Max(-l, math.Min(l, x[i]))
				}
			}
		}}
	})
}

// Product multiplies all the channels of a signal into one
func Product(s Sig) Sig {
	if len(s.ch) == 0 {
		panic("dsl: Product: a signal without channels (a Sig that was never made)")
	}
	p := s.Ch(0)
	for i := 1; i < len(s.ch); i++ {
		p = Mul(p, s.Ch(i))
	}
	return p
}

// Reverse puts the channels of a signal in the other order
func Reverse(s Sig) Sig {
	if len(s.ch) == 0 {
		panic("dsl: Reverse: a signal without channels (a Sig that was never made)")
	}
	out := make([]*node, len(s.ch))
	for i := range out {
		out[i] = s.ch[len(s.ch)-1-i]
	}
	return Sig{ch: out}
}

// LFNoiseControl is LFNoise as SuperCollider's LFNoise0.kr and LFNoise1.kr work it: at the control rate, one value for every block
// of 64 samples (held over the block), the frequency read once in the block, and a new random value every sr/64/freq of them. A
// patch of SuperCollider that modulates with these has jumps of a block at a time, which LFNoise does not have
func LFNoiseControl[F Param](freq F, interpolate bool) Sig {
	return expand("LFNoiseControl", []Sig{sig(freq)}, func(in []*node) *node {
		seed := seed64()
		return &node{name: "lfnoise.kr", id: fmt.Sprintf("lfnoise.kr %v", interpolate), in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			r, f := rng{seed}, b[0]
			level := r.bipolar()
			var slope float64
			counter := 0
			held := 0.0
			return func(n int) {
				pos := rt.Pos()
				for i := 0; i < n; i++ {
					if (pos+int64(i))%scBlock == 0 {
						if counter <= 0 {
							counter = max(1, int(rt.SampleRate/scBlock/math.Max(f[i], 0.001)))
							next := r.bipolar()
							slope = (next - level) / float64(counter)
							if !interpolate {
								level = next
							}
						}
						held = level
						if interpolate {
							level += slope
						}
						counter--
					}
					out[i] = held
				}
			}
		}}
	})
}

// RampControl is Ramp as SuperCollider's LFSaw.kr works it out: one value for every block of 64 samples, held over the block, the
// frequency read once in the block
func RampControl[F Param](freq F) Sig {
	return expand("RampControl", []Sig{sig(freq)}, func(in []*node) *node {
		return &node{name: "ramp.kr", key: "ramp.kr", in: in, build: func(rt *Rt, b [][]float64, out []float64) Kernel {
			f := b[0]
			var phase, held float64
			return func(n int) {
				pos := rt.Pos()
				for i := 0; i < n; i++ {
					if (pos+int64(i))%scBlock == 0 {
						held = phase
						phase += 2 * f[i] * scBlock / rt.SampleRate
						if phase >= 1 {
							phase -= 2
						} else if phase < -1 {
							phase += 2
						}
					}
					out[i] = held
				}
			}
		}}
	})
}
