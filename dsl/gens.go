package dsl

import (
	"github.com/almerlucke/genny/euclid"
	"github.com/almerlucke/genny/mutate"
	"github.com/almerlucke/genny/scale"
	"github.com/almerlucke/genny/sequence"
)

// Steps of common scales, in semitones above the root (genny's)
var (
	Major           = scale.Major
	Minor           = scale.Minor
	Dorian          = scale.Dorian
	MinorPentatonic = scale.MinorPentatonic
)

// Euclid is a euclidean rhythm: pulses spread over steps, rotated
func Euclid(steps, pulses, rotation int) Gen[bool] { return euclid.New(steps, pulses, rotation) }

// Loop plays the values in a loop
func Loop[T any](values ...T) Gen[T] { return sequence.NewLoop(values...) }

// Mutate plays a phrase that swaps notes around now and then, swapProb (0..1) after every pass
func Mutate(phrase []int, swapProb float64) Gen[int] {
	return mutate.New(phrase, mutate.Config[int]{SwapProb: swapProb})
}

// Degrees turns scale degrees (0 is the root, 7 an octave up in a 7 note scale) into MIDI notes
func Degrees(root float64, steps []float64, degrees Gen[int]) Gen[float64] {
	return scale.NewDegrees(scale.New(root, steps), degrees)
}
