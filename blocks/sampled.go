package blocks

import (
	"harp/dsl"

	"github.com/almerlucke/psalm/synths/sampler"
	"github.com/almerlucke/psalm/synths/streamer"
)

// The synths that play recordings. what says which instrument it is (the files, how they are cut and tuned), for a live edit: the same what is
// the same instrument, with its voices as they are.

// Sampler is psalm's sampler: an instrument of regions, each a part of a recording on a range of keys (an SFZ instrument, or a recording cut
// in pieces), played by voices that share their round robin turns and choke groups. The notes pick the regions by key and velocity
func Sampler(inst *sampler.Instrument, what string, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	cfg := sampler.DefaultConfig(inst)
	// the voices of a sampler share state (round robin, choke groups): they are made together and handed out one by one
	var made []*sampler.Voice
	next := 0
	newVoice := func() *sampler.Voice {
		if made == nil {
			sampler.NewPoly(cfg, voices).CallVoices(func(v *sampler.Voice) { made = append(made, v) })
		}
		v := made[next]
		next++
		return v
	}
	return synthWith(synthOptions{velocity: true}, "sampler "+what, newVoice, channelsOr(cfg.NumChannels), voices, notes)
}

// Streamer is psalm's streamer: long recordings played from disk while they play, so that a recording of any length costs the memory of its first
// half second. A reader of each note keeps ahead of the sound; the audio never waits for the disk
func Streamer(inst *streamer.Instrument, what string, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	cfg := streamer.DefaultConfig(inst)
	return synthWith(synthOptions{paced: true}, "streamer "+what, func() *streamer.Voice { return streamer.New(cfg) }, 2, voices, notes)
}
