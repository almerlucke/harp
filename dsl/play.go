package dsl

import (
	"context"
	"errors"
	"math"
)

// Engine is an audio device. It has the shape of psalm's audio.Engine (portaudio, jack), so those
// can be used as they are
type Engine interface {
	GetSampleRate() float64
	GetBufferSize() int
	SetCallback(func(in, out [][]float32))
	Init() error
	Start() error
	Stop() error
	Close() error
}

// Play runs out on the engine until ctx is done. A mono patch goes to every output channel; a
// patch of more channels to the first channels of the device, one each, and the other channels of
// the device are silent (use As to make the patch fit the device). Channels of the patch the device
// does not have are not heard. The sample rate is the
// engine's, whatever Patch.SampleRate says. The generators run on their own goroutine, up to
// Patch.Lookahead steps ahead of the audio. The engine is closed when Play returns
func (p Patch) Play(ctx context.Context, out Sig, eng Engine) (err error) {
	return p.PlayWith(ctx, out, eng, nil)
}

// PlayWith is Play that gives the runner to ready (when it plays, before the engine starts), for Swap: a patch that is changed while it
// plays. ready must not block: start a goroutine for what it does over time
func (p Patch) PlayWith(ctx context.Context, out Sig, eng Engine, ready func(r *Runner)) (err error) {
	p.SampleRate = eng.GetSampleRate()
	p = p.defaults()
	r := p.Start(out)
	// the generators run on a goroutine of their own, ahead of the audio
	if err := r.Go(p.Lookahead); err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, r.Close())
		if p.Stats != nil {
			p.Stats.Underruns = r.Underruns()
		}
	}()
	if p.From > 0 {
		r.Skip(p.From)
	}
	if ready != nil {
		ready(r)
	}
	nch := len(r.prog.outs)
	bufs := make([][]float64, nch)
	for i := range bufs {
		bufs[i] = make([]float64, 0, eng.GetBufferSize())
	}
	eng.SetCallback(func(_, dst [][]float32) {
		if len(dst) == 0 {
			return
		}
		n := len(dst[0])
		for i := range bufs {
			if cap(bufs[i]) < n { // the device gave a bigger buffer than it said: grow once, here
				bufs[i] = make([]float64, n)
			}
			bufs[i] = bufs[i][:n]
		}
		r.Process(bufs...)
		for c, ch := range dst {
			switch {
			case nch == 1: // mono: the same on every channel
				fill(ch, bufs[0])
			case c < nch:
				fill(ch, bufs[c])
			default: // a stereo patch on more than two channels
				clear(ch)
			}
		}
	})
	if err := eng.Init(); err != nil {
		return err
	}
	defer func() { err = errors.Join(err, eng.Close()) }()
	if err := eng.Start(); err != nil {
		return err
	}
	<-ctx.Done()
	return eng.Stop()
}

// fill converts to the device's format, hard clamped to [-1, 1]
func fill(dst []float32, src []float64) {
	for i, v := range src {
		dst[i] = float32(math.Max(-1, math.Min(1, v)))
	}
}
