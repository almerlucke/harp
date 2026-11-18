package blocks

import (
	"fmt"
	"sync"

	"harp/dsl"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
)

// psalm's effects and synths are modules: they ask the current environment of psalm for the sample
// rate (and a random stream) when they are made. inEnv makes them in an environment with the sample
// rate of the program, one at a time, and puts the environment that was current back. The random
// streams are seeded with a fixed number, so that what is made from the same patch sounds the same
var envMu sync.Mutex

func inEnv(sampleRate float64, build func()) {
	envMu.Lock()
	defer envMu.Unlock()
	prev := psalm.CurrentEnv()
	env := psalm.NewEnv(psalm.Config{SampleRate: sampleRate, BufferSize: dsl.BlockSize,
		BPM: psalm.DefaultBPM, PPQ: psalm.DefaultPPQ})
	env.Seed(1)
	psalm.SetCurrentEnv(env)
	defer psalm.SetCurrentEnv(prev)
	build()
}

func newFDNReverb(sampleRate float64, cfg fdnreverb.Config) (r *fdnreverb.FDNReverb) {
	inEnv(sampleRate, func() { r = fdnreverb.New(cfg) })
	return r
}

// FDNReverbWet is psalm's feedback delay network reverb, the reverberated sound only: eight delay lines
// mixed by a Hadamard matrix, with damping that makes the highs decay faster than the lows, a
// pre-delay, input diffusion and slow modulation of the lines so that it doesn't ring metallic.
// The input is stereo, as psalm's: a mono signal goes to both sides (see dsl.As), and the output
// is stereo. A signal of more than two channels is not for it (mix it down, or Spread it, first).
//
// The settings are fixed when the patch is built: Size, Decay (RT60 in seconds), Damping,
// PreDelay (ms), Diffusion, ModDepth, ModRate and Width. Mix is not used here: this is only the wet
// sound, to be blended with the dry one as you like (Crossfade, or a send level)
func FDNReverbWet(in dsl.Sig, cfg fdnreverb.Config) dsl.Sig {
	st := dsl.As(in, 2)
	return dsl.CustomN(dsl.SpecN{
		Name:    "fdnreverb",
		Key:     fmt.Sprintf("fdnreverb %+v", cfg),
		Inputs:  []dsl.Sig{st.Ch(0), st.Ch(1)},
		Outputs: 2,
		New: func(rt *dsl.Rt) dsl.ProcessN {
			r := newFDNReverb(rt.SampleRate, cfg)
			return func(b [][]float64, out [][]float64, n int) {
				l, rr := b[0], b[1]
				for i := 0; i < n; i++ {
					out[0][i], out[1][i] = r.ProcessSample(l[i], rr[i])
				}
			}
		},
	})
}

// FDNReverb is FDNReverbWet blended with the dry sound by cfg.Mix, with the equal power crossfade
// of psalm's module (0 is dry, 1 only the reverb), in stereo. For a mix that moves, or a send to
// the reverb, use FDNReverbWet and Crossfade or Mix yourself
func FDNReverb(in dsl.Sig, cfg fdnreverb.Config) dsl.Sig {
	dry := dsl.As(in, 2)
	return Crossfade(dry, FDNReverbWet(dry, cfg), cfg.Mix)
}
