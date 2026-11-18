package blocks

import (
	"fmt"
	"math"

	"harp/dsl"

	"github.com/almerlucke/psalm/blocks/oversample"
	"github.com/almerlucke/psalm/blocks/shapechain"
	"github.com/almerlucke/psalm/blocks/sine"
	"github.com/almerlucke/psalm/blocks/tanh"
)

// Shape runs a function over every sample: a waveshaper. With factor 2 or more it runs the
// function at that many times the sample rate, with a lowpass on the way up and on the way down
// (psalm's oversampler), so the harmonics a sharp function makes above the Nyquist frequency
// don't fold back as aliasing. The lowpass delays the sound by Latency samples (about a
// millisecond at 4 times: see oversample.New(factor, oversample.TapsPerPhase).Latency()). Factor 1
// is the function as it is, no delay. A genny shaper or a shapechain is f when you pass its Shape method
//
// Shapes made of a function are never merged: there is no telling what a function is
func Shape(in dsl.Sig, f func(float64) float64, factor int) dsl.Sig {
	return dsl.Custom(dsl.Spec{
		Name:   fmt.Sprintf("shape x%d", max(factor, 1)),
		Inputs: []dsl.Sig{in},
		New: func(rt *dsl.Rt) dsl.Process {
			var o *oversample.Oversampler
			if factor > 1 {
				o = oversample.New(factor, oversample.TapsPerPhase)
			}
			return func(b [][]float64, out []float64, n int) {
				x := b[0]
				if o == nil {
					for i := 0; i < n; i++ {
						out[i] = f(x[i])
					}
					return
				}
				for i := 0; i < n; i++ {
					out[i] = o.Process(x[i], f)
				}
			}
		},
	})
}

// Saturate is soft clipping: tanh(drive * x), with psalm's fast tanh. The drive can be a signal
// (the harder, the more harmonics). With factor 2 or more it is oversampled (see Shape, and its delay)
func Saturate[D dsl.Param](in dsl.Sig, drive D, factor int) dsl.Sig {
	return dsl.Custom(dsl.Spec{
		Name:   fmt.Sprintf("saturate x%d", max(factor, 1)),
		Key:    fmt.Sprintf("saturate %d", max(factor, 1)),
		Inputs: []dsl.Sig{in, dsl.Signal(drive)},
		New: func(rt *dsl.Rt) dsl.Process {
			var o *oversample.Oversampler
			if factor > 1 {
				o = oversample.New(factor, oversample.TapsPerPhase)
			}
			var d float64 // the drive of the sample that is being done, read by f
			f := func(v float64) float64 { return tanh.Tanh(d * v) }
			return func(b [][]float64, out []float64, n int) {
				x, drive := b[0], b[1]
				for i := 0; i < n; i++ {
					d = drive[i]
					if o == nil {
						out[i] = f(x[i])
					} else {
						out[i] = o.Process(x[i], f)
					}
				}
			}
		},
	})
}

// FastSine is a sine oscillator that reads psalm's table of one cycle with linear interpolation
// (what the effects of psalm use for their lfos): cheaper than a sine, and the error is small
// but not zero. Freq in Hz, a number or a signal
func FastSine[F dsl.Param](freq F) dsl.Sig {
	return dsl.Custom(dsl.Spec{
		Name:   "fastsine",
		Key:    "fastsine",
		Inputs: []dsl.Sig{dsl.Signal(freq)},
		New: func(rt *dsl.Rt) dsl.Process {
			var phase float64
			return func(b [][]float64, out []float64, n int) {
				f := b[0]
				for i := 0; i < n; i++ {
					out[i] = sine.Cycle(phase)
					phase += f[i] / rt.SampleRate
					phase -= math.Floor(phase)
				}
			}
		},
	})
}

// ChainOsc is an oscillator whose wave is a shapechain, psalm's small language for waves: the
// phase of the oscillator (0 to 1) goes into the chain and what comes out is the sound.
//
//	ChainOsc(shapechain.MustParse("bipolar | mirror -0.5 0.5"), 220)
//	ChainOsc(shapechain.MustParse("mult m1[1,5] | sin"), 110, Lfo(0.2).Range(0, 1))
//
// The macros m1 to m4 of the chain are the signals that follow the frequency, from 0 to 1 (the
// ones that are left out are 0.5). They are read every ControlInterval samples. The chain makes the
// wave as it is written: a sharp edge in it aliases (see Chain.Bandwidth)
func ChainOsc[F dsl.Param](c *shapechain.Chain, freq F, macros ...dsl.Sig) dsl.Sig {
	if len(macros) > shapechain.NumMacros {
		panic(fmt.Sprintf("blocks: ChainOsc: a chain has %d macros, got %d signals", shapechain.NumMacros, len(macros)))
	}
	inputs := []dsl.Sig{dsl.Signal(freq)}
	for i := 0; i < shapechain.NumMacros; i++ {
		if i < len(macros) {
			inputs = append(inputs, macros[i])
		} else {
			inputs = append(inputs, dsl.N(0.5))
		}
	}
	return dsl.Custom(dsl.Spec{
		Name:   "chainosc",
		Key:    "chainosc " + c.Text(),
		Inputs: inputs,
		New: func(rt *dsl.Rt) dsl.Process {
			var phase float64
			var m shapechain.Macros
			clk := clock{rt: rt}
			return func(b [][]float64, out []float64, n int) {
				f := b[0]
				for i := 0; i < n; i++ {
					if clk.at(i) {
						for k := range m {
							m[k] = b[1+k][i]
						}
					}
					out[i] = c.Eval(phase, &m)
					phase += f[i] / rt.SampleRate
					phase -= math.Floor(phase)
				}
			}
		},
	})
}
