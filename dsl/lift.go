package dsl

import (
	"fmt"
	"math"
)

// Lifts turn event rate generators into signals. A generator is pulled once per step, on the
// control side, and the value is applied on the sample the step starts: the audio kernels only
// ever hold a value. A generator should be lifted once: lifting it twice pulls it twice.

func mtof(note float64) float64 { return 440 * math.Pow(2, (note-69)/12) }

// Hold is a signal that holds the value of a generator until the next step
func Hold(g Gen[float64]) Sig {
	return mono(&node{name: "hold", id: "hold", lane: true, gens: Fingerprint(g), build: func(rt *Rt, _ [][]float64, out []float64) Kernel {
		var cur float64
		Lane(rt, g.Generate, func(v float64) { cur = v })
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = cur
			}
		}
	}})
}

// Pitch turns a generator of MIDI notes into a signal in Hz, held for a step. A4 (69) is 440 Hz
func Pitch(g Gen[float64]) Sig {
	return mono(&node{name: "pitch", id: "pitch", lane: true, gens: Fingerprint(g), build: func(rt *Rt, _ [][]float64, out []float64) Kernel {
		cur := 440.0
		Lane(rt, func() float64 { return mtof(g.Generate()) }, func(v float64) { cur = v })
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = cur
			}
		}
	}})
}

// Gate turns a generator of bools into a gate signal: high for length (0..1) of the step when the
// generator gives true, low otherwise. A rising edge starts an envelope, so a gate that is low
// for a moment between two steps is what retriggers it
func Gate(g Gen[bool], length float64) Sig {
	return mono(&node{name: "gate", id: fmt.Sprintf("gate %g", length), lane: true, gens: Fingerprint(g), build: func(rt *Rt, _ [][]float64, out []float64) Kernel {
		var left int
		Lane(rt, g.Generate, func(play bool) {
			if play {
				left = int(length * rt.StepSamples)
			} else {
				left = 0
			}
		})
		return func(n int) {
			for i := 0; i < n; i++ {
				if left > 0 {
					out[i] = 1
					left--
				} else {
					out[i] = 0
				}
			}
		}
	}})
}

// Voice is the pitch and the gate of a melody that only moves when a note is played
type Voice struct {
	Pitch Sig // Hz, changes on the sample the gate rises, and holds through the release
	Gate  Sig // high for length of the step on every step the rhythm plays
}

// Notes plays a melody on a rhythm. The rhythm is pulled every step; the melody only on the steps
// where the rhythm is true, so the next note starts together with the envelope that the gate
// triggers, and a rest keeps the last pitch (the release of the note before it doesn't jump).
//
// A length of 1 keeps the gate high between two played steps: no new trigger, only a pitch change
// (legato). Lift a generator once: Notes pulls both of them
func Notes(melody Gen[float64], rhythm Gen[bool], length float64) Voice {
	v := &voice{melody: melody, rhythm: rhythm, length: length, freq: 440}
	gens := Fingerprint(melody, rhythm)
	pitch := mono(&node{name: "notes.pitch", id: fmt.Sprintf("notes %g", length), lane: true, gens: gens, group: v, build: func(rt *Rt, _ [][]float64, out []float64) Kernel {
		v.attach(rt)
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = v.freq
			}
		}
	}})
	gate := mono(&node{name: "notes.gate", id: fmt.Sprintf("notes %g", length), lane: true, gens: gens, group: v, build: func(rt *Rt, _ [][]float64, out []float64) Kernel {
		v.attach(rt)
		return func(n int) {
			for i := 0; i < n; i++ {
				if v.left > 0 {
					out[i] = 1
					v.left--
				} else {
					out[i] = 0
				}
			}
		}
	}})
	return Voice{Pitch: pitch, Gate: gate}
}

type voice struct {
	melody Gen[float64]
	rhythm Gen[bool]
	length float64
	freq   float64
	left   int // samples the gate stays high
	rt     *Rt // the compile this voice is attached to
}

// voiceStep is what the control side makes of a step of a voice
type voiceStep struct {
	play bool
	freq float64
}

// attach registers the lane once per compile, however many of its signals are built
func (v *voice) attach(rt *Rt) {
	if v.rt == rt {
		return
	}
	v.rt = rt
	Lane(rt, func() (s voiceStep) {
		if v.rhythm.Generate() { // the melody only moves when the rhythm plays
			s.play, s.freq = true, mtof(v.melody.Generate())
		}
		return
	}, func(s voiceStep) {
		if s.play {
			v.freq = s.freq
			v.left = int(v.length * rt.StepSamples)
		} else {
			v.left = 0
		}
	})
}

// PitchN is Pitch for a generator of chords (or any list of notes): a signal in Hz of width
// channels, one for every note, held for a step. A chord of fewer notes than width repeats its last
// note in the channels that are left, a longer one is cut off, and an empty one changes nothing.
// The width is fixed when the patch is made, as the number of channels cannot change while it
// plays. Saw(PitchN(chords, 3)) is three saws: Spread or Fold them
func PitchN(g Gen[[]float64], width int) Sig {
	if width < 1 {
		panic("dsl: PitchN: width must be at least 1")
	}
	p := &pitchN{g: g, hz: make([]float64, width)}
	for i := range p.hz {
		p.hz[i] = 440
	}
	out := Sig{ch: make1(width)}
	for i := range out.ch {
		out.ch[i] = &node{name: "pitchN", id: fmt.Sprintf("pitchN %d", width), lane: true, gens: Fingerprint(g), group: p, build: func(rt *Rt, _ [][]float64, out []float64) Kernel {
			p.attach(rt)
			return func(n int) {
				f := p.hz[i]
				for j := 0; j < n; j++ {
					out[j] = f
				}
			}
		}}
	}
	return out
}

type pitchN struct {
	g  Gen[[]float64]
	hz []float64
	rt *Rt
}

func (p *pitchN) attach(rt *Rt) {
	if p.rt == rt {
		return
	}
	p.rt = rt
	Lane(rt, func() []float64 {
		notes := p.g.Generate()
		if len(notes) == 0 {
			return nil
		}
		hz := make([]float64, len(p.hz))
		for i := range hz {
			hz[i] = mtof(notes[min(i, len(notes)-1)])
		}
		return hz
	}, func(hz []float64) {
		if hz != nil {
			copy(p.hz, hz)
		}
	})
}
