package blocks

import (
	"fmt"

	"harp/dsl"

	"github.com/almerlucke/psalm/blocks/mix"
	"github.com/almerlucke/psalm/blocks/pitchshift"
)

// Crossfade blends two signals with psalm's equal power crossfade: mix 0 is all dry, 1 all wet, and
// halfway is as loud as either (a linear blend dips 3 dB), which suits a wet signal that doesn't
// correlate with the dry one (a reverb, an echo). For an effect that works by cancelling against the dry
// signal (a flanger, a phaser) use a linear blend: Add(Mul(dry, 1-m), Mul(wet, m)). Mix can be a signal
func Crossfade[M dsl.Param](dry, wet dsl.Sig, mixAmount M) dsl.Sig {
	return dsl.Custom(dsl.Spec{
		Name:   "crossfade",
		Key:    "crossfade",
		Inputs: []dsl.Sig{dry, wet, dsl.Signal(mixAmount)},
		New: func(rt *dsl.Rt) dsl.Process {
			return func(b [][]float64, out []float64, n int) {
				d, w, m := b[0], b[1], b[2]
				for i := 0; i < n; i++ {
					gd, gw := mix.EqualPower(m[i])
					out[i] = d[i]*gd + w[i]*gw
				}
			}
		},
	})
}

// Smooth glides a signal to its changes with a one pole lowpass, taking about ms milliseconds to
// cover two thirds of a step (psalm's smoother, 30 ms is what its effects use): put it on a control
// signal that steps (a generator held for a step) so that it doesn't click or zip
func Smooth(in dsl.Sig, ms float64) dsl.Sig {
	return dsl.Custom(dsl.Spec{
		Name:   "smooth",
		Key:    fmt.Sprintf("smooth %x", ms),
		Inputs: []dsl.Sig{in},
		New: func(rt *dsl.Rt) dsl.Process {
			s := mix.NewSmoother(0, ms, rt.SampleRate)
			started := false
			return func(b [][]float64, out []float64, n int) {
				x := b[0]
				for i := 0; i < n; i++ {
					if !started { // begin at the first value, not at 0
						s.Value, started = x[i], true
					}
					out[i] = s.Next(x[i])
				}
			}
		},
	})
}

// PitchShift is psalm's delay line pitch shifter, two heads crossfading like a rotating tape head:
// semitones up (or down, negative) as a number or a signal, window the size of a grain in
// milliseconds (30 to 100: bigger is smoother on low sounds, smaller follows quick changes). It
// delays the sound by about half a window, and has the weak spot of its kind: a tone whose period
// fits the window a whole number of times and a half can thin out (see the package of psalm)
func PitchShift[S dsl.Param](in dsl.Sig, semitones S, windowMs float64) dsl.Sig {
	return dsl.Custom(dsl.Spec{
		Name:   "pitchshift",
		Key:    fmt.Sprintf("pitchshift %x", windowMs),
		Inputs: []dsl.Sig{in, dsl.Signal(semitones)},
		New: func(rt *dsl.Rt) dsl.Process {
			p := pitchshift.New(0, windowMs, rt.SampleRate)
			clk := clock{rt: rt}
			return func(b [][]float64, out []float64, n int) {
				x, semi := b[0], b[1]
				for i := 0; i < n; i++ {
					if clk.at(i) {
						p.SetSemitones(semi[i])
					}
					out[i] = p.Process(x[i])
				}
			}
		},
	})
}
