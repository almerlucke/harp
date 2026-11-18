// play runs the stereo lead on the sound card, through psalm's portaudio engine, until Ctrl+C
// (or for -seconds). Psalm's jack engine would do too: both are a dsl.Engine
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/synths/subtractive"

	"harp/blocks"
	. "harp/dsl"
)

func main() {
	seed := flag.Uint64("seed", 7, "seed of the generators")
	bpm := flag.Float64("bpm", 110, "tempo")
	secs := flag.Float64("seconds", 0, "stop after this many seconds (0: until Ctrl+C)")
	buf := flag.Int("buffer", 512, "samples per buffer: smaller is less latency, bigger is safer")
	reverb := flag.Float64("reverb", 0.25, "how much reverb, 0 (none) to 1 (only reverb)")
	decay := flag.Float64("decay", 2.5, "decay time of the reverb in seconds")
	pad := flag.Float64("pad", 0.4, "level of the chord pad under the lead, 0 for none")
	flag.Parse()

	Seed(*seed)
	notes := Degrees(57, Minor, Mutate([]int{7, 9, 11, 9, 8, 7, 4, 7}, 0.6))
	voice := Notes(notes, Euclid(16, 7, 0), 0.6)

	// two saws a few cents apart: one signal of two channels. The filter is two filters (one on
	// each saw) with one cutoff, the envelope is one gate for both: a mono signal goes to every channel
	saws := Saw(Chans(Mul(voice.Pitch, 0.997), Mul(voice.Pitch, 1.023)))
	// psalm's state variable filter, one for each saw, on one cutoff
	filtered := Mul(blocks.Lowpass(saws, Lfo(0.3).Range(300, 4000), 0.5), Adsr(0.005, 0.15, 0.5, 0.2).On(voice.Gate))
	dry := Gain(Spread(filtered, 0.6), 0.4) // out to the left and the right
	// an echo with feedback, a dotted eighth on the left and a quarter on the right: the time has
	// two channels, so each side has its own delay line and its own time
	beat := 60 / *bpm
	lead := blocks.Echo(dry, 2.0, Chans(N(beat*0.75), N(beat)), 0.45, 0.3)
	// psalm's subtractive synth with its pad preset, playing a chord on every bar (16 steps): a generator
	// of chords on a rhythm gives the notes, and the notes are held for 14 steps
	if *pad > 0 {
		chords := Loop([]float64{57, 60, 64}, []float64{53, 57, 60, 65}, []float64{48, 55, 60, 64}, []float64{55, 59, 62, 67})
		bar := Loop(append([]bool{true}, make([]bool, 15)...)...)
		synth := blocks.Subtractive(subtractive.PadPreset(), 8, blocks.Chords(chords, bar, 0.7, 14))
		lead = Add(lead, Gain(synth, *pad))
	}
	// psalm's feedback delay network reverb on the echoed sound, so that the echoes have a room too
	if *reverb > 0 {
		cfg := fdnreverb.DefaultConfig()
		cfg.Mix, cfg.Decay = *reverb, *decay
		lead = blocks.FDNReverb(lead, cfg)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if *secs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*secs*float64(time.Second)))
		defer cancel()
	}

	eng := paengine.New(44100, *buf)
	fmt.Printf("playing at %.0f Hz, buffer %d, %.0f bpm (Ctrl+C stops)\n", eng.GetSampleRate(), eng.GetBufferSize(), *bpm)
	if err := (Patch{BPM: *bpm}).Play(ctx, lead, eng); err != nil {
		fmt.Fprintln(os.Stderr, "audio:", err)
		os.Exit(1)
	}
}
