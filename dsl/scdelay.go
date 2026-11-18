package dsl

import (
	"fmt"
	"math"
)

// SuperCollider's DelayN and DelayC (DelayUGens.cpp, 3.14.1): a delay line of a size that is a power of two, a delay time in samples
// (at least 1) that is cut to a whole number (DelayN) or read with cubic interpolation between four samples
// (DelayC). What the line has not had yet is silence.

type scDelay struct {
	rt     *Rt
	cubic  bool
	max    float64
	buf    []float32
	mask   int
	length float64
	wr     int
}

func newSCDelay(rt *Rt, maxTime float64, cubic bool) *scDelay {
	size := int(math.Ceil(maxTime*rt.SampleRate + 1))
	size += scBlock
	n := 1
	for n < size {
		n <<= 1
	}
	return &scDelay{rt: rt, cubic: cubic, buf: make([]float32, n), mask: n - 1, length: float64(n)}
}

func (d *scDelay) process(in [][]float64, out []float64, n int) {
	const minDelay = 1.0 // for DelayC too (SuperCollider's DelayC is made with the unit of the interpolations that need 1)
	for i := 0; i < n; i++ {
		dsamp := float32(in[1][i]) * float32(d.rt.SampleRate)
		dsamp = float32(math.Min(math.Max(float64(dsamp), minDelay), d.length))
		idsamp := int(dsamp)
		frac := dsamp - float32(idsamp)
		d.buf[d.wr&d.mask] = float32(in[0][i])
		rd1 := d.wr - idsamp
		if !d.cubic {
			if rd1 < 0 {
				out[i] = 0
			} else {
				out[i] = float64(d.buf[rd1&d.mask])
			}
		} else {
			at := func(k int) float32 {
				if k < 0 {
					return 0
				}
				return d.buf[k&d.mask]
			}
			d0, d1, d2, d3 := at(rd1+1), at(rd1), at(rd1-1), at(rd1-2)
			c0 := d1
			c1 := 0.5 * (d2 - d0)
			c2 := d0 - 2.5*d1 + 2*d2 - 0.5*d3
			c3 := 0.5*(d3-d0) + 1.5*(d1-d2)
			out[i] = float64(((c3*frac+c2)*frac+c1)*frac + c0)
		}
		d.wr++
	}
}

// SCDelayN is SuperCollider's DelayN: the delay time is cut to whole samples. max is the longest time in seconds
func SCDelayN[T Param](x Sig, max float64, time T) Sig {
	return Custom(Spec{Name: "SCDelayN", Key: fmt.Sprintf("scdelayn %g", max), Inputs: []Sig{x, Signal(time)}, New: func(rt *Rt) Process { return newSCDelay(rt, max, false).process }})
}

// SCDelayC is SuperCollider's DelayC: the delay time is any number of samples, read with cubic interpolation
func SCDelayC[T Param](x Sig, max float64, time T) Sig {
	return Custom(Spec{Name: "SCDelayC", Key: fmt.Sprintf("scdelayc %g", max), Inputs: []Sig{x, Signal(time)}, New: func(rt *Rt) Process { return newSCDelay(rt, max, true).process }})
}
