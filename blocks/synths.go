package blocks

import (
	"fmt"

	"harp/dsl"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/synths/drumsynth"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/shaper"
	"github.com/almerlucke/psalm/synths/vocal"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/synths/wavetable"
)

// The other synths of psalm, played like Subtractive and Supersaw: voices of one kind, notes from a generator of notes. cfg is a preset of
// the synth or your own. A voice that gets a note while all are busy takes over from the oldest one

func channelsOr(n int) int {
	if n == 0 {
		return psalm.Stereo
	}
	return n
}

// Modal is psalm's modal synth: struck and bowed bars, bells, strings and plates as sets of resonances (marimba, vibraphone, bells, ...)
func Modal(cfg modal.Config, voices int, notes dsl.Gen[[]Note], opts ...SynthOption) dsl.Sig {
	return synth(fmt.Sprintf("modal %+v", cfg), func() *modal.Voice { return modal.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes, opts...)
}

// Piano is psalm's piano: hammers, strings and a soundboard
func Piano(cfg piano.Config, voices int, notes dsl.Gen[[]Note], opts ...SynthOption) dsl.Sig {
	return synth(fmt.Sprintf("piano %+v", cfg), func() *piano.Voice { return piano.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes, opts...)
}

// Waveguide is psalm's physical models: plucked and bowed strings, pipes and membranes as waves going up and down a line
func Waveguide(cfg waveguide.Config, voices int, notes dsl.Gen[[]Note], opts ...SynthOption) dsl.Sig {
	return synth(fmt.Sprintf("waveguide %+v", cfg), func() *waveguide.Voice { return waveguide.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes, opts...)
}

// Vocal is psalm's voice: a glottis and a vocal tract (solo, choir, ooh, aah, ...)
func Vocal(cfg vocal.Config, voices int, notes dsl.Gen[[]Note], opts ...SynthOption) dsl.Sig {
	return synth(fmt.Sprintf("vocal %+v", cfg), func() *vocal.Voice { return vocal.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes, opts...)
}

// Pulsar is psalm's pulsar synthesis: trains of short pulses, with formants
func Pulsar(cfg pulsar.Config, voices int, notes dsl.Gen[[]Note], opts ...SynthOption) dsl.Sig {
	return synth(fmt.Sprintf("pulsar %+v", cfg), func() *pulsar.Voice { return pulsar.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes, opts...)
}

// Shaper is psalm's waveshaping synth, phase distortion and the like
func Shaper(cfg shaper.Config, voices int, notes dsl.Gen[[]Note], opts ...SynthOption) dsl.Sig {
	return synth(fmt.Sprintf("shaper %+v", cfg), func() *shaper.Voice { return shaper.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes, opts...)
}

// Grains is psalm's granular synth: every note is a stream of short grains of a recording, at the pitch of the note. what names the recording and
// the settings (it tells a live edit whether the voices are the same)
func Grains(cfg grains.Config, what string, voices int, notes dsl.Gen[[]Note], opts ...SynthOption) dsl.Sig {
	return synth("grains "+what, func() *grains.Voice { return grains.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes, opts...)
}

// Wavetable is psalm's wavetable synth: an oscillator morphing through the frames of a table, in unison, with a filter. mods are signals that
// move the position, the cutoff, the resonance or the values cabled to the modulation matrix while the notes sound (wavetable.InPosition...)
func Wavetable(cfg wavetable.Config, what string, voices int, notes dsl.Gen[[]Note], mods []Mod, opts ...SynthOption) dsl.Sig {
	return synthWith(synthOptions{mods: mods}, "wavetable "+what, func() *wavetable.Voice { return wavetable.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes, opts...)
}

// organVoice lets the organ be driven like a voice: it is one instrument that every note goes to (the keys share the wheels), always sounding
// (its wheels spin, its Leslie rings on)
type organVoice struct{ *organ.Organ }

func (o organVoice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, cfg psalm.Config) {
	o.Note(3600, amplitude, message, tick, cfg)
}
func (o organVoice) NoteOff()       { o.AllNotesOff() }
func (o organVoice) Clear()         {}
func (o organVoice) IsActive() bool { return true }

// Organ is psalm's tonewheel organ, with its drawbars, percussion, vibrato and Leslie: one instrument, the notes are keys of it (a chord is
// several keys, a note is held for its Length). what names the settings (it tells a live edit whether it is the same organ)
func Organ(cfg organ.Config, what string, notes dsl.Gen[[]Note], opts ...SynthOption) dsl.Sig {
	return synthWith(synthOptions{single: true}, "organ "+what, func() organVoice { return organVoice{organ.New(cfg)} }, channelsOr(cfg.NumChannels), 1, notes, opts...)
}

// DX is psalm's FM synth, in the way of the Yamaha DX7 and TX81Z: operators and algorithms, with the patches of the DX7 (SysEx) or its own
func DX(patch dx.Patch, what string, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	return synth("dx "+what, func() *dx.Voice { return dx.New(patch, psalm.Stereo) }, psalm.Stereo, voices, notes)
}

// DrumSynth is psalm's drum synthesizer: a kit of drums that are made, not played from recordings (kick, snare, hats, toms, cymbals, claps, ...).
// A note is a drum by its General MIDI key (36 kick, 38 snare, 42 closed hat, 46 open hat, ...); the voices are made together, so that an open hat
// is stopped by a closed one
func DrumSynth(cfg drumsynth.Config, what string, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	var made []*drumsynth.Voice
	next := 0
	newVoice := func() *drumsynth.Voice {
		if made == nil {
			drumsynth.NewPoly(cfg, voices).CallVoices(func(v *drumsynth.Voice) { made = append(made, v) })
		}
		v := made[next]
		next++
		return v
	}
	return synth("drumsynth "+what, newVoice, channelsOr(cfg.NumChannels), voices, notes)
}
