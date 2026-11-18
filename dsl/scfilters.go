package dsl

import "math"

// SuperCollider's biquad filters, as FilterUGens.cpp (3.14.1) works them out: the frequency (and the others) are read once in every
// block of 64 samples; a change makes the coefficients move in straight lines over the block, in steps every three samples (the
// loops of three of SuperCollider), and the last sample of the block has the new ones. With no change they stay as they are.

type scBiquad struct {
	rt      *Rt
	kind    int
	started bool
	y1, y2  float64
	// the coefficients now (a1 and a2 for the peaking filter), the parameters they are of, and what a block of change is
	a0, a1, a2, b1, b2 float64
	p                  [3]float32
	slope              [5]float64
	changing           bool
	cur                [5]float64
	pending            [5]float64
	j                  int // the sample of the block
	out                float64
}

const (
	scLPF = iota
	scHPF
	scRLPF
	scBPeakEQ
)

var sqrt2 = math.Sqrt2

// coefficients of a kind for parameters p, in the order a0, a1, a2, b1, b2
func (f *scBiquad) coefs(p [3]float32) (c [5]float64) {
	sr := f.rt.SampleRate
	switch f.kind {
	case scLPF:
		pfreq := float64(p[0]) * (2 * math.Pi / sr) * 0.5
		C := 1 / math.Tan(pfreq)
		C2 := C * C
		sqrt2C := C * sqrt2
		a0 := 1 / (1 + sqrt2C + C2)
		c[0], c[3], c[4] = a0, -2*(1-C2)*a0, -(1-sqrt2C+C2)*a0
	case scHPF:
		pfreq := float64(p[0]) * (2 * math.Pi / sr) * 0.5
		C := math.Tan(pfreq)
		C2 := C * C
		sqrt2C := C * sqrt2
		a0 := 1 / (1 + sqrt2C + C2)
		c[0], c[3], c[4] = a0, 2*(1-C2)*a0, -(1-sqrt2C+C2)*a0
	case scRLPF:
		qres := math.Max(0.001, float64(p[1]))
		pfreq := float64(p[0]) * (2 * math.Pi / sr)
		D := math.Tan(pfreq * qres * 0.5)
		C := (1 - D) / (1 + D)
		cosf := math.Cos(pfreq)
		b1 := (1 + C) * cosf
		c[3], c[4] = b1, -C
		c[0] = (1 + C - b1) * 0.25
	case scBPeakEQ:
		a := math.Pow(10, float64(p[2])*0.025)
		w0 := 2 * math.Pi * float64(p[0]) / sr
		alpha := math.Sin(w0) * 0.5 * float64(p[1])
		b0rz := 1 / (1 + alpha/a)
		b1 := 2 * b0rz * math.Cos(w0)
		c[0], c[1], c[2] = (1+alpha*a)*b0rz, -b1, (1-alpha*a)*b0rz
		c[3], c[4] = b1, (1-alpha/a)*-b0rz
	}
	return
}

func (f *scBiquad) load(c [5]float64) { f.a0, f.a1, f.a2, f.b1, f.b2 = c[0], c[1], c[2], c[3], c[4] }
func (f *scBiquad) store() [5]float64 { return [5]float64{f.a0, f.a1, f.a2, f.b1, f.b2} }

// step works one sample out, and keeps the state
func (f *scBiquad) step(x float64) (y float64) {
	switch f.kind {
	case scLPF:
		y0 := x + f.b1*f.y1 + f.b2*f.y2
		y = f.a0 * (y0 + 2*f.y1 + f.y2)
		f.y2, f.y1 = f.y1, y0
	case scHPF:
		y0 := x + f.b1*f.y1 + f.b2*f.y2
		y = f.a0 * (y0 - 2*f.y1 + f.y2)
		f.y2, f.y1 = f.y1, y0
	case scRLPF:
		y0 := f.a0*x + f.b1*f.y1 + f.b2*f.y2
		y = y0 + 2*f.y1 + f.y2
		f.y2, f.y1 = f.y1, y0
	case scBPeakEQ:
		y0 := x + f.b1*f.y1 + f.b2*f.y2
		y = f.a0*y0 + f.a1*f.y1 + f.a2*f.y2
		f.y2, f.y1 = f.y1, y0
	}
	return
}

func (f *scBiquad) process(in [][]float64, out []float64, n int) {
	pos := f.rt.Pos()
	for i := 0; i < n; i++ {
		if (pos+int64(i))%scBlock == 0 {
			var p [3]float32
			for k := 1; k < len(in); k++ {
				p[k-1] = float32(in[k][i])
			}
			switch {
			case !f.started:
				f.started, f.p = true, p
				f.load(f.coefs(p))
				f.changing = false
				// the constructor of the unit works one sample out of the first input: that sample is in the state before the
				// first block starts
				f.step(float64(float32(in[0][i])))
			case p != f.p:
				next := f.coefs(p)
				f.cur = f.store()
				for k := range next {
					f.slope[k] = (next[k] - f.cur[k]) * (1.0 / (scBlock / 3))
				}
				f.p, f.changing, f.j = p, true, 0
				f.pending = next
			default:
				f.changing = false
			}
			f.j = 0
		}
		y := f.step(float64(float32(in[0][i])))
		out[i] = float64(float32(y))
		f.j++
		if f.changing && f.j <= scBlock-1 && f.j%3 == 0 && f.j <= scBlock-scBlock%3 {
			// after every group of three the coefficients move on
			f.a0 += f.slope[0]
			f.a1 += f.slope[1]
			f.a2 += f.slope[2]
			f.b1 += f.slope[3]
			f.b2 += f.slope[4]
		}
		if f.changing && f.j == scBlock {
			f.load(f.pending)
			f.changing = false
		}
	}
}

func scFilter(name string, kind int, x Sig, params ...Sig) Sig {
	return Custom(Spec{Name: name, Key: name, Inputs: append([]Sig{x}, params...), New: func(rt *Rt) Process {
		return (&scBiquad{rt: rt, kind: kind}).process
	}})
}

// SCLPF is SuperCollider's LPF: a second order lowpass (Butterworth) with the cutoff in Hz
func SCLPF[F Param](x Sig, freq F) Sig { return scFilter("SCLPF", scLPF, x, Signal(freq)) }

// SCHPF is SuperCollider's HPF: a second order highpass (Butterworth)
func SCHPF[F Param](x Sig, freq F) Sig { return scFilter("SCHPF", scHPF, x, Signal(freq)) }

// SCRLPF is SuperCollider's RLPF: a resonant lowpass, rq the reciprocal of Q (1 is flat, smaller resonates)
func SCRLPF[F, R Param](x Sig, freq F, rq R) Sig {
	return scFilter("SCRLPF", scRLPF, x, Signal(freq), Signal(rq))
}

// SCBPeakEQ is SuperCollider's BPeakEQ: a peaking equalizer of db decibel at freq, rq the bandwidth as a part of freq
func SCBPeakEQ[F, R, D Param](x Sig, freq F, rq R, db D) Sig {
	return scFilter("SCBPeakEQ", scBPeakEQ, x, Signal(freq), Signal(rq), Signal(db))
}
