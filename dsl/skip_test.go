package dsl

import (
	"context"
	"testing"
)

const skipStep = 3150 // 210 beats a minute, 4 steps a beat, 44100 Hz

func skipPatch() (Patch, func() Sig) {
	p := Patch{SampleRate: 44100, BPM: 210, StepsPerBeat: 4}
	return p, func() Sig {
		ev := Events(&fixedEvents{steps: []float64{4, 3, 1}}, 1, 1)
		return Chans(ev[0].Gate, ev[0].Vals[0])
	}
}

// A runner that skips ahead makes what the piece makes from there on: the same notes at the same times, the same numbers (the generators ran), and
// nothing of a note that began in what was skipped, even when it is still sounding. The events start at the steps 0, 4, 7, 8, 12, 15, 16; the one of
// step 8 lasts 2 steps, and a skip of 9 steps comes in the middle of it
func TestSkipPlaysTheRestOfThePiece(t *testing.T) {
	p, build := skipPatch()
	r := p.Start(build())
	fg, fv := make([]float64, 24*skipStep), make([]float64, 24*skipStep)
	r.Process(fg, fv)

	r = p.Start(build())
	r.Skip(9 * skipStep / 44100.0)
	n := 15 * skipStep
	g, v := make([]float64, n), make([]float64, n)
	r.Process(g, v)
	for i := 0; i < 3*skipStep-64; i++ { // steps 9 to 12 (an event starts with the block it falls in): the event of step 8 is not played
		if g[i] != 0 {
			t.Fatalf("sample %d after the skip: a note of the skipped part sounds", i)
		}
	}
	// the block that the event of step 12 starts in begins a little before it in a render from 0, and so does not in a run that skipped: leave it out
	for i := 3*skipStep + 64; i < n; i++ {
		if g[i] != fg[9*skipStep+i] {
			t.Fatalf("sample %d after the skip: the gate is %v, in the whole piece %v", i, g[i], fg[9*skipStep+i])
		}
		if g[i] != 0 && v[i] != fv[9*skipStep+i] {
			t.Fatalf("sample %d after the skip: the number is %v, in the whole piece %v", i, v[i], fv[9*skipStep+i])
		}
	}
	if v[3*skipStep+100] != 5 {
		t.Errorf("the first event that plays is number %v, it is the fifth", v[3*skipStep+100])
	}
}

// Play with From starts that far in: a device is given the sound of the piece from there
func TestPlayFromStartsInThePiece(t *testing.T) {
	p, build := skipPatch()
	p.From = 9 * skipStep / 44100.0
	eng := newFake(44100, skipStep)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error)
	go func() { done <- p.Play(ctx, build(), eng) }()
	<-eng.started
	var gate []float32
	for k := 0; k < 4; k++ {
		l, r := make([]float32, eng.size), make([]float32, eng.size)
		eng.cb(nil, [][]float32{l, r})
		gate = append(gate, l[100])
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if gate[0] != 0 || gate[1] != 0 || gate[2] != 0 || gate[3] != 1 {
		t.Errorf("the gates of the first four steps are %v: three of silence (the note of step 8 is not played), and the note of step 12", gate)
	}
}
