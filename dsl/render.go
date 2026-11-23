package dsl

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"sync/atomic"
)

// Patch holds the clock and the sample rate. A step is a sixteenth by default
type Patch struct {
	SampleRate   float64    // default 44100
	BPM          float64    // default 120
	StepsPerBeat float64    // default 4
	Lookahead    int        // steps the control goroutine stays ahead of the audio when playing, default 16
	NoOptimize   bool       // compile the graph as it was built: see the optimizer in optimize.go
	Stats        *PlayStats // if set, Play fills it in when it returns
	From         float64    // Play starts this many seconds into the piece (see Runner.Skip), default 0
}

// PlayStats is what Play found out about the run
type PlayStats struct {
	// Underruns is the number of steps whose values from the generators were not there in time
	// (see Runner.Underruns): 0 means the control goroutine always kept ahead
	Underruns int
}

func (p Patch) defaults() Patch {
	if p.SampleRate == 0 {
		p.SampleRate = 44100
	}
	if p.BPM == 0 {
		p.BPM = 120
	}
	if p.StepsPerBeat == 0 {
		p.StepsPerBeat = 4
	}
	if p.Lookahead == 0 {
		p.Lookahead = 16
	}
	return p
}

// Runner runs a compiled program, a piece at a time: offline in a loop (Render) or from the
// callback of an audio device (Play). It keeps the clock between calls, so the size of the pieces
// doesn't matter: a step starts on its exact sample, because a block is cut where a step begins
type Runner struct {
	prog        *Program
	stepSamples float64
	untilStep   float64 // samples until the next step
	stepIdx     int64   // the step that starts next

	ctl         controlSide // makes the step records
	applyLanes  []lane      // the lanes whose values the audio side applies: those of the program that plays
	swapMu      sync.Mutex
	latest      *Program                // the program that was last given to Swap (or Start): the next one takes over from it
	pending     *swapReq                // a program that waits for the control goroutine to hand it to the audio side
	incoming    atomic.Pointer[swapReq] // a program that the audio side takes at its next step
	active      *swapReq                // the program that took over last, while the records made before it are played
	lastTrans   map[int64]*stepRec      // (control side) what the last program that took over was given for the steps that were made before it
	lastNewFrom int64
	wake        chan struct{} // tells the control goroutine that a program waits
	ring        *ring         // the records on their way to the audio, when the control side has a goroutine
	stop        chan struct{}
	done        chan struct{}
	err         atomic.Value
	underruns   atomic.Int64
}

// Start compiles out. Nothing plays until Process is called
func (p Patch) Start(out Sig) *Runner {
	p = p.defaults()
	stepSamples := p.SampleRate * 60 / (p.BPM * p.StepsPerBeat)
	prog := compile(out, p.SampleRate, stepSamples, !p.NoOptimize)
	return &Runner{prog: prog, latest: prog, stepSamples: stepSamples, ctl: controlSide{lanes: prog.lanes}, applyLanes: prog.lanes}
}

// Program is the compiled graph
func (r *Runner) Program() *Program { return r.prog }

// Process fills one slice per output channel of the program (one for a mono patch, left and right
// for a stereo one) with the next len samples, from one goroutine only (the audio callback). They
// must all be the same length. At the start of a step it applies the values the control side made
// for it. After Go that is a look in a queue; without Go the generators are pulled right here
func (r *Runner) Process(dst ...[]float64) {
	if len(dst) != len(r.prog.outs) {
		panic(fmt.Sprintf("dsl: Process got %d channels, the patch has %d", len(dst), len(r.prog.outs)))
	}
	size := len(dst[0])
	for _, d := range dst {
		if len(d) != size {
			panic("dsl: Process got channels of different lengths")
		}
	}
	for done := 0; done < size; {
		for r.untilStep <= 0 {
			r.startStep()
			r.untilStep += r.stepSamples
		}
		n := min(BlockSize, int(math.Ceil(r.untilStep)), size-done)
		for _, k := range r.prog.kernels {
			k(n)
		}
		for c, d := range dst {
			copy(d[done:], r.prog.outs[c][:n])
		}
		r.prog.rt.pos += int64(n)
		done += n
		r.untilStep -= float64(n)
	}
}

// Skip moves the runner on by seconds without making a sound: the generators and the clock run as they would, but the graph does not, and the notes that
// start in the part that is skipped are not played. What plays after it is what plays at that time in the piece, without what was sounding before it
// (a reverb starts empty, a note that began before it is not heard). Call it after Go and before the first Process
func (r *Runner) Skip(seconds float64) {
	rt := r.prog.rt
	size := int64(math.Round(seconds * rt.SampleRate))
	rt.skipTo = rt.pos + size
	for done := int64(0); done < size; {
		for r.untilStep <= 0 {
			r.startStep()
			r.untilStep += r.stepSamples
		}
		n := min(int64(math.Ceil(r.untilStep)), size-done)
		rt.pos += n
		done += n
		r.untilStep -= float64(n)
	}
}

func (r *Runner) startStep() {
	k := r.stepIdx
	r.stepIdx++
	if s := r.incoming.Swap(nil); s != nil { // a program takes over from this step
		r.install(s)
	}
	rec := r.fetch(k)
	if rec == nil {
		return
	}
	for i, l := range r.applyLanes {
		l.apply(rec.vals[i])
	}
}

// Render compiles out and runs it for the given seconds: one slice of samples per channel
func (p Patch) Render(out Sig, seconds float64) (*Program, [][]float64) {
	p = p.defaults()
	r := p.Start(out)
	res := make([][]float64, len(r.prog.outs))
	for i := range res {
		res[i] = make([]float64, int(seconds*p.SampleRate))
	}
	r.Process(res...)
	return r.prog, res
}

// WriteWAV writes 16 bit PCM with a channel for every slice of samples, as Render gives them
func WriteWAV(path string, sampleRate int, channels ...[]float64) error {
	return writeFile(path, sampleRate, channels...)
}

// WriteWAVFloat writes 32 bit floating point WAV, the channels interleaved: the samples as they are, with no
// clipping and no rounding to steps (what SuperCollider writes with sampleFormat "float")
func WriteWAVFloat(path string, sampleRate int, channels ...[]float64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := writeWAVFloat(f, sampleRate, channels...); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func writeWAVFloat(w io.Writer, sr int, channels ...[]float64) error {
	nch, frames := len(channels), len(channels[0])
	dataLen := uint32(frames * nch * 4)
	hdr := []any{
		[4]byte{'R', 'I', 'F', 'F'}, 36 + dataLen, [4]byte{'W', 'A', 'V', 'E'},
		[4]byte{'f', 'm', 't', ' '}, uint32(16), uint16(3), uint16(nch), uint32(sr), uint32(sr * nch * 4), uint16(nch * 4), uint16(32),
		[4]byte{'d', 'a', 't', 'a'}, dataLen,
	}
	for _, v := range hdr {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	buf := make([]float32, 0, frames*nch)
	for i := 0; i < frames; i++ {
		for _, c := range channels {
			buf = append(buf, float32(c[i]))
		}
	}
	return binary.Write(w, binary.LittleEndian, buf)
}

func writeFile(path string, sr int, channels ...[]float64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := writeWAV(f, sr, channels...); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// writeWAV writes 16 bit PCM, the channels interleaved
func writeWAV(w io.Writer, sr int, channels ...[]float64) error {
	nch, frames := len(channels), len(channels[0])
	dataLen := uint32(frames * nch * 2)
	hdr := []any{
		[4]byte{'R', 'I', 'F', 'F'}, 36 + dataLen, [4]byte{'W', 'A', 'V', 'E'},
		[4]byte{'f', 'm', 't', ' '}, uint32(16), uint16(1), uint16(nch), uint32(sr), uint32(sr * nch * 2), uint16(nch * 2), uint16(16),
		[4]byte{'d', 'a', 't', 'a'}, dataLen,
	}
	for _, v := range hdr {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	buf := make([]int16, 0, frames*nch)
	for i := 0; i < frames; i++ {
		for _, c := range channels {
			// rounded to the nearest step: truncating makes the error up to a whole step and twice the noise, which
			// is heard in a channel that is quiet (a pan to one side)
			buf = append(buf, int16(math.Round(math.Max(-1, math.Min(1, c[i]))*32767)))
		}
	}
	return binary.Write(w, binary.LittleEndian, buf)
}
