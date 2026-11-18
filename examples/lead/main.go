// lead renders a short melody in stereo: a mutating phrase on a euclidean rhythm, two detuned
// saws through a sweeping filter
package main

import (
	"fmt"

	. "harp/dsl"
)

func main() {
	Seed(7)

	notes := Degrees(57, Minor, Mutate([]int{7, 9, 11, 9, 8, 7, 4, 7}, 0.6)) // Gen[float64]: event rate
	rhythm := Euclid(16, 7, 0)                                               // Gen[bool]: event rate

	voice := Notes(notes, rhythm, 0.6) // the next note is taken when the rhythm plays: with the envelope

	// two saws a few cents apart: one signal of two channels. The filter is two filters (one on
	// each saw) with one cutoff, the envelope is one gate for both: a mono signal goes to every channel
	saws := Saw(Chans(Mul(voice.Pitch, 0.997), Mul(voice.Pitch, 1.003)))
	filtered := saws.Lowpass(Lfo(0.3).Range(300, 4000), N(0.9)).
		Mul(Adsr(0.005, 0.15, 0.5, 0.2).On(voice.Gate))
	lead := Gain(Spread(filtered, 0.6), 0.4) // and out to the left and the right

	p := Patch{BPM: 110}
	prog, channels := p.Render(lead, 8)
	fmt.Print(prog, prog.Opt)
	if err := WriteWAV("lead.wav", 44100, channels...); err != nil {
		panic(err)
	}
}
