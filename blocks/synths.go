package blocks

import (
	"fmt"

	"harp/dsl"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/psalm/synths/modal"
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
func Modal(cfg modal.Config, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	return synth(fmt.Sprintf("modal %+v", cfg), func() *modal.Voice { return modal.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes)
}

// Piano is psalm's piano: hammers, strings and a soundboard
func Piano(cfg piano.Config, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	return synth(fmt.Sprintf("piano %+v", cfg), func() *piano.Voice { return piano.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes)
}

// Waveguide is psalm's physical models: plucked and bowed strings, pipes and membranes as waves going up and down a line
func Waveguide(cfg waveguide.Config, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	return synth(fmt.Sprintf("waveguide %+v", cfg), func() *waveguide.Voice { return waveguide.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes)
}

// Vocal is psalm's voice: a glottis and a vocal tract (solo, choir, ooh, aah, ...)
func Vocal(cfg vocal.Config, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	return synth(fmt.Sprintf("vocal %+v", cfg), func() *vocal.Voice { return vocal.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes)
}

// Pulsar is psalm's pulsar synthesis: trains of short pulses, with formants
func Pulsar(cfg pulsar.Config, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	return synth(fmt.Sprintf("pulsar %+v", cfg), func() *pulsar.Voice { return pulsar.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes)
}

// Shaper is psalm's waveshaping synth, phase distortion and the like
func Shaper(cfg shaper.Config, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	return synth(fmt.Sprintf("shaper %+v", cfg), func() *shaper.Voice { return shaper.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes)
}

// Grains is psalm's granular synth: every note is a stream of short grains of a recording, at the pitch of the note. what names the recording and
// the settings (it tells a live edit whether the voices are the same)
func Grains(cfg grains.Config, what string, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	return synth("grains "+what, func() *grains.Voice { return grains.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes)
}

// Wavetable is psalm's wavetable synth: an oscillator morphing through the frames of a table, in unison, with a filter. mods are signals that
// move the position, the cutoff, the resonance or the values cabled to the modulation matrix while the notes sound (wavetable.InPosition...)
func Wavetable(cfg wavetable.Config, what string, voices int, notes dsl.Gen[[]Note], mods []Mod) dsl.Sig {
	return synthWith(synthOptions{mods: mods}, "wavetable "+what, func() *wavetable.Voice { return wavetable.New(cfg) }, channelsOr(cfg.NumChannels), voices, notes)
}
