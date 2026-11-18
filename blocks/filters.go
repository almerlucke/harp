package blocks

import (
	"fmt"

	"harp/dsl"

	"github.com/almerlucke/psalm/blocks/filters/biquad"
	"github.com/almerlucke/psalm/blocks/filters/formant"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/filters/svf"
)

// SVF is psalm's state variable filter (zero delay feedback, after Simper) in a mode: cutoff in Hz,
// res from 0 to 1 (0.5 is a Q of 1, near 1 it rings). Cutoff and res can be signals
func SVF[C, R dsl.Param](in dsl.Sig, mode svf.Mode, cutoff C, res R) dsl.Sig {
	return dsl.Custom(dsl.Spec{
		Name:   fmt.Sprintf("svf.%s", svfModes[mode]),
		Key:    fmt.Sprintf("svf %d", mode),
		Inputs: []dsl.Sig{in, dsl.Signal(cutoff), dsl.Signal(res)},
		New: func(rt *dsl.Rt) dsl.Process {
			f := svf.New(1000, 0.5, rt.SampleRate)
			c := clock{rt: rt}
			return func(b [][]float64, out []float64, n int) {
				x, fc, r := b[0], b[1], b[2]
				for i := 0; i < n; i++ {
					if c.at(i) {
						f.Set(fc[i], r[i])
					}
					out[i] = f.Process(x[i], mode)
				}
			}
		},
	})
}

var svfModes = map[svf.Mode]string{
	svf.LowPass: "lowpass", svf.BandPass: "bandpass", svf.HighPass: "highpass",
	svf.Notch: "notch", svf.Peak: "peak", svf.AllPass: "allpass",
}

// Lowpass, Highpass, Bandpass, Notch, Peak and Allpass are the modes of SVF
func Lowpass[C, R dsl.Param](in dsl.Sig, cutoff C, res R) dsl.Sig {
	return SVF(in, svf.LowPass, cutoff, res)
}
func Highpass[C, R dsl.Param](in dsl.Sig, cutoff C, res R) dsl.Sig {
	return SVF(in, svf.HighPass, cutoff, res)
}
func Bandpass[C, R dsl.Param](in dsl.Sig, cutoff C, res R) dsl.Sig {
	return SVF(in, svf.BandPass, cutoff, res)
}
func Notch[C, R dsl.Param](in dsl.Sig, cutoff C, res R) dsl.Sig {
	return SVF(in, svf.Notch, cutoff, res)
}
func Peak[C, R dsl.Param](in dsl.Sig, cutoff C, res R) dsl.Sig { return SVF(in, svf.Peak, cutoff, res) }
func Allpass[C, R dsl.Param](in dsl.Sig, cutoff C, res R) dsl.Sig {
	return SVF(in, svf.AllPass, cutoff, res)
}

// Ladder is psalm's Moog style ladder filter, four poles (24 dB per octave) or two, as a lowpass,
// bandpass or highpass: res from 0 to 1 (it sings at 1), drive saturates the input (0 is clean). Cutoff
// and res can be signals
func Ladder[C, R dsl.Param](in dsl.Sig, mode ladder.Mode, cutoff C, res R, drive float64) dsl.Sig {
	return dsl.Custom(dsl.Spec{
		Name:   "ladder",
		Key:    fmt.Sprintf("ladder %d %x", mode, drive),
		Inputs: []dsl.Sig{in, dsl.Signal(cutoff), dsl.Signal(res)},
		New: func(rt *dsl.Rt) dsl.Process {
			f := ladder.New(1000, 0.5, rt.SampleRate)
			f.SetDrive(drive)
			c := clock{rt: rt}
			return func(b [][]float64, out []float64, n int) {
				x, fc, r := b[0], b[1], b[2]
				for i := 0; i < n; i++ {
					if c.at(i) {
						f.Set(fc[i], r[i])
					}
					out[i] = f.Process(x[i], mode)
				}
			}
		},
	})
}

// Biquad is psalm's second order filter with the cookbook coefficients: lowpass, highpass,
// bandpass, notch, peaking EQ, shelves and allpass. q is the bandwidth (0.707 is flat for low and
// high pass), gainDB is the boost or cut of the peak and the shelves. Cutoff and q can be signals
func Biquad[C, Q dsl.Param](in dsl.Sig, typ biquad.Type, cutoff C, q Q, gainDB float64) dsl.Sig {
	return dsl.Custom(dsl.Spec{
		Name:   "biquad",
		Key:    fmt.Sprintf("biquad %d %x", typ, gainDB),
		Inputs: []dsl.Sig{in, dsl.Signal(cutoff), dsl.Signal(q)},
		New: func(rt *dsl.Rt) dsl.Process {
			f := biquad.New(typ, 1000, 0.707, gainDB, rt.SampleRate)
			c := clock{rt: rt}
			return func(b [][]float64, out []float64, n int) {
				x, fc, q := b[0], b[1], b[2]
				for i := 0; i < n; i++ {
					if c.at(i) {
						f.Set(typ, fc[i], q[i], gainDB)
					}
					out[i] = f.Process(x[i])
				}
			}
		},
	})
}

// Formant is psalm's vowel filter: three resonances at the formants of a vowel. vowel is a
// position from 0 to 4 (a e i o u, in between morphs), q the sharpness of the formants (5 to 20).
// Vowel and q can be signals
func Formant[V, Q dsl.Param](in dsl.Sig, vowel V, q Q) dsl.Sig {
	return dsl.Custom(dsl.Spec{
		Name:   "formant",
		Key:    "formant",
		Inputs: []dsl.Sig{in, dsl.Signal(vowel), dsl.Signal(q)},
		New: func(rt *dsl.Rt) dsl.Process {
			f := formant.New(0, 10, rt.SampleRate)
			c := clock{rt: rt}
			return func(b [][]float64, out []float64, n int) {
				x, v, q := b[0], b[1], b[2]
				for i := 0; i < n; i++ {
					if c.at(i) {
						f.Set(v[i], q[i])
					}
					out[i] = f.Process(x[i])
				}
			}
		},
	})
}
