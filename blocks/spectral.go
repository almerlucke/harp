package blocks

import (
	"fmt"

	"harp/dsl"

	"github.com/almerlucke/psalm/blocks/fft"
)

// Spectral processes a signal in the frequency domain, frame by frame (a short time Fourier
// transform with overlap add): every hop = size/4 samples, the last size samples are windowed with
// a Hann window, transformed with psalm's real FFT, handed to a function that may change the
// bins, transformed back, windowed again and added to the output. With a function that does
// nothing the sound comes out as it went in, delayed by size-1 samples.
//
// newProcess is called once for every channel (so a function with a state, like a frozen spectrum
// or the last frame, has one for each channel) and returns the function. It gets the bins 0 to
// size/2 of the frame: re and im, bin k is at k*sampleRate/size Hz. It may change them in place, and
// must not keep them. Size is a power of 2, at least 16 (1024 is 23 ms at 44.1 kHz: a bigger
// frame is finer in frequency, and more delay). It runs on the audio thread: it must not allocate or wait
//
// Spectral nodes are never merged: there is no telling what a function is
func Spectral(in dsl.Sig, size int, newProcess func() func(re, im []float64)) dsl.Sig {
	if size < 16 || size&(size-1) != 0 {
		panic(fmt.Sprintf("blocks: Spectral: size %d must be a power of 2, at least 16", size))
	}
	return dsl.Custom(dsl.Spec{
		Name:   fmt.Sprintf("spectral %d", size),
		Inputs: []dsl.Sig{in},
		New: func(rt *dsl.Rt) dsl.Process {
			const overlap = 4
			hop := size / overlap
			scale := 1 / 1.5 // the Hann window squared, added at 4 times overlap, is 1.5
			r := fft.NewReal(size)
			win := fft.HannWindow(size)
			f := newProcess()
			hist, acc, frame := make([]float64, size), make([]float64, size), make([]float64, size)
			re, im := make([]float64, size/2+1), make([]float64, size/2+1)
			t := 0 // samples taken in
			return func(b [][]float64, out []float64, n int) {
				x := b[0]
				for i := 0; i < n; i++ {
					hist[t%size] = x[i]
					t++
					if t%hop == 0 { // a frame ends at the sample just taken in
						for k := range frame { // the oldest sample is at t%size
							frame[k] = hist[(t+k)%size] * win[k]
						}
						r.Forward(frame, re, im)
						f(re, im)
						r.Inverse(re, im, frame)
						for k := range frame { // frame[k] is the sample of time t-size+k
							acc[(t+k)%size] += frame[k] * win[k] * scale
						}
					}
					// the sample of time t-size has had all of its frames: that is what comes out
					out[i] = acc[t%size]
					acc[t%size] = 0
				}
			}
		},
	})
}
