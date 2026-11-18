package blocks

import (
	"fmt"
	"math"

	"harp/dsl"

	"github.com/almerlucke/psalm/blocks/delay"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
)

// delaySamples is the time in samples, with the shortest (one sample) and the longest the line
// can hold (it reads three samples around the place, for the interpolation)
func delaySamples(seconds, sr float64, length int) float64 {
	return math.Min(math.Max(seconds*sr, 1), float64(length-3))
}

// Delay is a delay line with the delay of psalm (Hermite interpolation): time in seconds can be a
// signal, so a delay with a slow lfo on the time is a chorus or a flanger. The line holds
// maxSeconds, the longest time it can have; the shortest is one sample
func Delay[T dsl.Param](in dsl.Sig, maxSeconds float64, time T) dsl.Sig {
	return dsl.Custom(dsl.Spec{
		Name:   "delay",
		Key:    fmt.Sprintf("delay %x", maxSeconds),
		Inputs: []dsl.Sig{in, dsl.Signal(time)},
		New: func(rt *dsl.Rt) dsl.Process {
			d := delay.New(int(maxSeconds*rt.SampleRate) + 4)
			return func(b [][]float64, out []float64, n int) {
				x, t := b[0], b[1]
				for i := 0; i < n; i++ {
					out[i] = d.ReadHermite(delaySamples(t[i], rt.SampleRate, d.Length))
					d.Write(x[i])
				}
			}
		},
	})
}

// Echo is a delay with feedback, in the one node (a patch can't loop back on itself): the signal,
// and after time seconds the signal again, feedback of it again, and so on. mix is how much of
// the echoes is heard: 0 is the dry signal, 1 only the echoes. Time, feedback and mix can be
// signals. Keep the feedback under 1
func Echo[T, F, M dsl.Param](in dsl.Sig, maxSeconds float64, time T, feedback F, mix M) dsl.Sig {
	return dsl.Custom(dsl.Spec{
		Name:   "echo",
		Key:    fmt.Sprintf("echo %x", maxSeconds),
		Inputs: []dsl.Sig{in, dsl.Signal(time), dsl.Signal(feedback), dsl.Signal(mix)},
		New: func(rt *dsl.Rt) dsl.Process {
			d := delay.New(int(maxSeconds*rt.SampleRate) + 4)
			return func(b [][]float64, out []float64, n int) {
				x, t, fb, mix := b[0], b[1], b[2], b[3]
				for i := 0; i < n; i++ {
					echo := d.ReadHermite(delaySamples(t[i], rt.SampleRate, d.Length))
					d.Write(x[i] + echo*fb[i])
					out[i] = x[i] + (echo-x[i])*mix[i]
				}
			}
		},
	})
}

// Osc is psalm's band limited oscillator (sine, triangle, square, pulse, saw, ramp and the
// others of blosc): freq in Hz, pulseWidth from 0 to 1 for the shapes that have one (0.5 is a
// square). Both can be signals; the frequency is read every sample, so it can be modulated at
// audio rate (FM)
func Osc[F, W dsl.Param](wave blosc.Waveform, freq F, pulseWidth W) dsl.Sig {
	return dsl.Custom(dsl.Spec{
		Name:   "osc",
		Key:    fmt.Sprintf("osc %d", wave),
		Inputs: []dsl.Sig{dsl.Signal(freq), dsl.Signal(pulseWidth)},
		New: func(rt *dsl.Rt) dsl.Process {
			o := blosc.New(440, rt.SampleRate, 0, 0.5, 1, wave)
			c := clock{rt: rt}
			return func(b [][]float64, out []float64, n int) {
				f, pw := b[0], b[1]
				for i := 0; i < n; i++ {
					o.SetFrequency(f[i], rt.SampleRate)
					if c.at(i) {
						o.SetPulseWidth(pw[i])
					}
					out[i] = o.Tick(0, false, 0)
				}
			}
		},
	})
}
