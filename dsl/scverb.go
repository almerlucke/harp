package dsl

import "fmt"

// SCFreeVerb2 is SuperCollider's FreeVerb2 (ReverbUGens.cpp, 3.14.1): the Freeverb of Jezar, for two channels: eight lowpass feedback
// combs and four allpass filters for each, the right channel with delay lines 23 samples longer. The input of both is the sum of
// the two channels. mix, room and damp are from 0 to 1, read once in a block. It works in 32 bit floating point, as SuperCollider's does

var fvLeft = [12]int{225, 341, 441, 556, 1617, 1557, 1491, 1422, 1277, 1116, 1188, 1356}
var fvRight = [12]int{248, 364, 464, 579, 1640, 1580, 1514, 1445, 1300, 1139, 1211, 1379}

type fvChannel struct {
	line [12][]float32
	iota [12]int
	// the allpass states (R*_0 of the allpass lines 0..3) and the comb output and lowpass states
	ap   [4]float32
	comb [8]float32
	lp   [8]float32
}

func newFvChannel(sizes [12]int) *fvChannel {
	c := &fvChannel{}
	for i, n := range sizes {
		c.line[i] = make([]float32, n)
	}
	return c
}

// step takes the summed and scaled input, and gives the wet output
func (c *fvChannel) step(in4, room, damp, damp1 float32) float32 {
	var t [12]float32
	for i := range c.line {
		c.iota[i]++
		if c.iota[i] == len(c.line[i]) {
			c.iota[i] = 0
		}
		t[i] = c.line[i][c.iota[i]]
	}
	for j := 0; j < 8; j++ {
		l := 4 + j
		c.lp[j] = damp1*c.comb[j] + damp*c.lp[j]
		c.line[l][c.iota[l]] = in4 + room*c.lp[j]
		c.comb[j] = t[l]
	}
	sum := (c.comb[0] + c.comb[1]) + (c.comb[2] + c.comb[3]) + ((c.comb[4] + c.comb[5]) + (c.comb[6] + c.comb[7]))
	// the order of the additions as in the source: ((0.5*R3 + R4) + (R6 + R8)) + ((R10 + R12) + (R14 + (R16 + R18)))
	feed := ((0.5*c.ap[3] + c.comb[0]) + (c.comb[1] + c.comb[2])) + ((c.comb[3] + c.comb[4]) + (c.comb[5] + (c.comb[6] + c.comb[7])))
	c.line[3][c.iota[3]] = feed
	c.ap[3] = t[3]
	r := c.ap[3] - sum
	for k := 2; k >= 0; k-- {
		c.line[k][c.iota[k]] = 0.5*c.ap[k] + r
		c.ap[k] = t[k]
		r = c.ap[k] - r
	}
	return r
}

// SCFreeVerb2 makes the reverb of two channels (left and right): the result has two channels
func SCFreeVerb2(left, right Sig, mix, room, damp float64) Sig {
	return CustomN(SpecN{Name: "SCFreeVerb2", Key: fmt.Sprintf("freeverb2 %g %g %g", mix, room, damp), Inputs: []Sig{left, right}, Outputs: 2, New: func(rt *Rt) ProcessN {
		l, r := newFvChannel(fvLeft), newFvChannel(fvRight)
		clamp := func(v float64) float32 { return float32(min(max(v, 0), 1)) }
		m, ro, d := clamp(mix), clamp(room), clamp(damp)
		wet, dry := m, 1-m
		room5 := 0.7 + 0.28*ro
		damp6 := 0.4 * d
		damp7 := 1 - damp6
		return func(in [][]float64, out [][]float64, n int) {
			for i := 0; i < n; i++ {
				a, b := float32(in[0][i]), float32(in[1][i])
				in4 := 1.5e-02 * (a + b)
				out[0][i] = float64(dry*a + wet*l.step(in4, room5, damp6, damp7))
				out[1][i] = float64(dry*b + wet*r.step(in4, room5, damp6, damp7))
			}
		}
	}})
}
