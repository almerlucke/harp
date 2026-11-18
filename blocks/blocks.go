// Package blocks makes the DSP blocks of psalm (filters, delay, oscillators) nodes of a harp
// patch. It is a module of its own, so that harp/dsl itself needs nothing of psalm.
//
// Every function here takes a Sig and gives a Sig, expands over channels like the nodes of dsl
// (a stereo input is two filters, each with a state of its own), and takes its parameters as a
// number or as a signal. A parameter that is a signal is looked at every ControlInterval samples,
// at the same samples of the stream whatever the size of the blocks: the sound doesn't depend on
// the buffer size of the device, and coefficients that are costly to compute (tan, pow) are
// computed 16 times less often.
package blocks

import "harp/dsl"

// ControlInterval is the number of samples between two looks at the parameters of a block that has
// coefficients to compute. A power of two
const ControlInterval = 16

// clock says at which samples of a block the parameters are read: the first sample the node ever
// computes, and every sample whose place in the stream is a multiple of ControlInterval
type clock struct {
	rt      *dsl.Rt
	started bool
}

// at is true if the parameters are to be read at sample i of the block. Call it in order, for
// every sample of the block, from 0
func (c *clock) at(i int) bool {
	if !c.started || (c.rt.Pos()+int64(i))&(ControlInterval-1) == 0 {
		c.started = true
		return true
	}
	return false
}
