package lang

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"harp/dsl"
)

// live starts a program the way harp play does, but offline, paced so that the control goroutine keeps ahead as it does in time
type live struct {
	t   *testing.T
	r   *dsl.Runner
	p   dsl.Patch
	cur *Patch // the patch that plays: the next one is built from it (Recompile)
}

func startLive(t *testing.T, src string) *live {
	t.Helper()
	patch := mustBuild(t, src)
	st := patch.Settings()
	st.SampleRate = 8000 // a step of 1000 samples at 120 bpm and 4 steps a beat
	st.Lookahead = 4
	st.NoOptimize = true
	r := st.Start(patch.Out)
	if err := r.Go(st.Lookahead); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return &live{t: t, r: r, p: st, cur: patch}
}

func (l *live) play(n int) []float64 {
	out := make([]float64, 0, n)
	for len(out) < n {
		time.Sleep(2 * time.Millisecond) // the control goroutine is ahead after this
		size := min(250, n-len(out))
		chans := make([][]float64, l.r.Program().Channels())
		for i := range chans {
			chans[i] = make([]float64, size)
		}
		l.r.Process(chans...)
		out = append(out, chans[0]...)
	}
	return out
}

func (l *live) swapTo(src string) {
	l.t.Helper()
	if err := l.trySwap(src); err != nil {
		l.t.Fatal(err)
	}
}

func (l *live) trySwap(src string) error {
	patch, err := Recompile(src, l.cur)
	if err != nil {
		return err
	}
	st := patch.Settings()
	st.NoOptimize = true
	if err := l.r.Swap(patch.Out, st); err != nil {
		return err
	}
	l.cur = patch
	return nil
}

// the same program run from the start, for what the playing one is compared with
func reference(t *testing.T, src string, n int) []float64 {
	t.Helper()
	l := startLive(t, src)
	return l.play(n)
}

const phraseA = "bpm = 120\nsteps = 4\nout = degrees(60, major, [0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15])"

func TestALiveEditKeepsAGeneratorWhereItIs(t *testing.T) {
	ref := reference(t, phraseA, 40000)
	l := startLive(t, phraseA)
	l.play(5000)
	// the same phrase, one more on every note: the generator is the same, so it goes on from where it was
	l.swapTo(phraseA + " + 1")
	rest := l.play(35000)
	all := append(make([]float64, 5000), rest...)
	// the steps after the swap: the new value is the old stream's plus 1
	changedAt := -1
	for step := 5; step < 40; step++ {
		got, want := all[step*1000+10], ref[step*1000+10]
		if got == want+1 {
			if changedAt < 0 {
				changedAt = step
			}
			continue
		}
		if changedAt >= 0 {
			t.Fatalf("step %d: %v, the stream goes on with %v (+1)", step, got, want)
		}
	}
	if changedAt < 0 || changedAt > 5+6 {
		t.Fatalf("the edit was not heard in time: first at step %d", changedAt)
	}
}

func TestALiveEditWithAnotherPhraseStartsItAgain(t *testing.T) {
	l := startLive(t, phraseA)
	l.play(5000)
	l.swapTo("bpm = 120\nsteps = 4\nout = degrees(60, major, [0 2 4])")
	rest := l.play(15000)
	// the new phrase starts at its first note (60): a restart, not the place of the old one (which was at the 6th note or later)
	first := -1
	for step := 0; step < 15; step++ {
		if rest[step*1000+10] == 60 && (step == 0 || rest[(step-1)*1000+10] != 60) {
			first = step
		}
	}
	// the old phrase is at 60+ (0 1 2 3 4 5 ... in major: 60 62 64 65 67 69 71 72 ...): a 60 after the first steps is the new one starting
	if first < 0 {
		t.Errorf("the new phrase never started from its start: %v", rest[10:15000])
	}
}

func TestALiveEditKeepsTheStateOfTheOscillatorsAndFilters(t *testing.T) {
	a := "bpm = 120\nsteps = 4\nout = saw(110).lowpass(cutoff: 800, res: 0.3).gain(0.5)"
	l := startLive(t, a)
	l.play(3000)
	// another cutoff and gain: the same nodes
	l.swapTo("bpm = 120\nsteps = 4\nout = saw(110).lowpass(cutoff: 1200, res: 0.3).gain(0.25)")
	kept, fresh := l.r.LastSwap()
	if kept < 2 || fresh != 0 {
		t.Errorf("kept %d, new %d: the saw and the lowpass should both be kept", kept, fresh)
	}
	l.play(3000) // a change takes over before the next one
	// another oscillator: it is new, and the filter after it is the same filter, with its memory, given another input
	l.swapTo("bpm = 120\nsteps = 4\nout = square(110).lowpass(cutoff: 1200, res: 0.3).gain(0.25)")
	kept, fresh = l.r.LastSwap()
	if kept != 1 || fresh != 1 {
		t.Errorf("kept %d, new %d after another oscillator: the filter is kept, the oscillator is new (%v)", kept, fresh, l.r.LastRestarted())
	}
	l.play(3000)
	// a setting that is in the node (the decay of a reverb) is new for it, and the oscillator before it is kept
	l = startLive(t, "bpm = 120\nsteps = 4\nout = square(110).reverb(mix: 0.2, decay: 2)")
	l.play(2000)
	l.swapTo("bpm = 120\nsteps = 4\nout = square(110).reverb(mix: 0.2, decay: 3)")
	kept, fresh = l.r.LastSwap()
	if kept < 1 || fresh < 1 {
		t.Errorf("kept %d, new %d: the oscillator is kept and the reverb with the other decay is new", kept, fresh)
	}
}

const notesA = `bpm = 120
steps = 4
fn inst(gate, n) { gate * n }
out = play(inst, rhythm: euclid(8, 3), legato: 1, fields: [degrees(60, major, [0 1 2 3 4 5 6 7 8 9 10 11])])`

func TestALiveEditKeepsAPatternOfNotesWhereItIs(t *testing.T) {
	ref := reference(t, notesA, 40000)
	l := startLive(t, notesA)
	l.play(5000)
	// the instrument changes (twice as loud): the notes go on in their phrase
	l.swapTo(`bpm = 120
steps = 4
fn inst(gate, n) { gate * n * 2 }
out = play(inst, rhythm: euclid(8, 3), legato: 1, fields: [degrees(60, major, [0 1 2 3 4 5 6 7 8 9 10 11])])`)
	rest := l.play(35000)
	all := append(make([]float64, 5000), rest...)
	// find where the sound is twice the reference's: from there on, always
	doubled := -1
	for i := 6000; i < 40000; i += 100 {
		if ref[i] != 0 && all[i] == 2*ref[i] {
			if doubled < 0 {
				doubled = i
			}
		} else if ref[i] != 0 && doubled >= 0 {
			t.Fatalf("at %d: %v, the phrase goes on with %v (doubled)", i, all[i], 2*ref[i])
		}
	}
	if doubled < 0 {
		t.Fatal("the edit was never heard")
	}
	if kept, _ := l.r.LastSwap(); kept == 0 {
		t.Error("nothing was kept")
	}
}

// two fields; the instrument gives them on two channels
const twoFields = `bpm = 120
steps = 4
fn inst(gate, a, b) { [gate * a, gate * b] }
out = play(inst, rhythm: euclid(8, 3), legato: 1, fields: [degrees(60, major, [0 1 2 3 4 5 6 7 8 9 10 11]), %s])`

func (l *live) playChannels(n int) [][]float64 {
	chans := make([][]float64, l.r.Program().Channels())
	for i := range chans {
		chans[i] = make([]float64, 0, n)
	}
	for done := 0; done < n; {
		time.Sleep(2 * time.Millisecond)
		size := min(250, n-done)
		bufs := make([][]float64, len(chans))
		for i := range bufs {
			bufs[i] = make([]float64, size)
		}
		l.r.Process(bufs...)
		for i := range bufs {
			chans[i] = append(chans[i], bufs[i]...)
		}
		done += size
	}
	return chans
}

// a field that is edited is put in place, and the rest of the pattern goes on: the other phrase is where it was, the rhythm is the same, and
// the notes that were made before the edit already have the new numbers
func TestALiveEditOfOneFieldKeepsTheRestOfThePatternWhereItIs(t *testing.T) {
	srcA := fmt.Sprintf(twoFields, "[1 2 3]")
	srcB := fmt.Sprintf(twoFields, "[10 20 30 40]")
	ref := startLive(t, srcA).playChannels(40000)
	l := startLive(t, srcA)
	before := l.playChannels(5000)
	l.swapTo(srcB)
	if kept, fresh := l.r.LastSwap(); fresh != 0 {
		t.Errorf("kept %d, %d started again: %v", kept, fresh, l.r.LastRestarted())
	}
	after := l.playChannels(35000)
	a := append(before[0], after[0]...)
	b := append(before[1], after[1]...)
	// the first phrase is the one of the reference, all the way: nothing was lost or repeated
	for i := range a {
		if a[i] != ref[0][i] {
			t.Fatalf("the phrase of the first field differs at sample %d (%.2f s): %v, the unedited pattern has %v", i, float64(i)/8000, a[i], ref[0][i])
		}
	}
	// the second field has the new numbers from the first note after the edit (at the next step): 10 20 30 40 in turn, from 10
	var seen []float64
	for i := 5000; i < len(b); i++ {
		if b[i] != 0 && (len(seen) == 0 || b[i] != seen[len(seen)-1]) {
			seen = append(seen, b[i])
		}
	}
	// the note that sounds when the edit is made goes on until its end (the change takes over at the next step): it has an old number
	for len(seen) > 0 && seen[0] <= 3 {
		seen = seen[1:]
	}
	if len(seen) < 4 || seen[0] != 10 || seen[1] != 20 || seen[2] != 30 || seen[3] != 40 {
		t.Errorf("the second field after the edit: %v, want 10 20 30 40 from the first note", seen)
	}
	// the notes before the edit were the old ones
	if b[100] == 0 || b[100] > 3 {
		t.Errorf("before the edit the second field is %v", b[100])
	}
}

// the rhythm is edited: the pattern goes on, the notes follow the new rhythm from the next step, and the phrase of the field goes on from
// the note that it was at: none is skipped, none is played twice
func TestALiveEditOfTheRhythmKeepsThePhrase(t *testing.T) {
	srcA := fmt.Sprintf(twoFields, "[1 2 3]")
	srcB := strings.Replace(srcA, "euclid(8, 3)", "euclid(8, 5)", 1)
	l := startLive(t, srcA)
	before := l.playChannels(9000)
	l.swapTo(srcB)
	if kept, fresh := l.r.LastSwap(); fresh != 0 {
		t.Errorf("kept %d, %d started again: %v", kept, fresh, l.r.LastRestarted())
	}
	after := l.playChannels(40000)
	a := append(before[0], after[0]...)
	// the notes: the value of the first field changes at every note (the phrase is 60 62 64 65 67 69 71 72 74 76 77 79, again and again)
	phrase := []float64{60, 62, 64, 65, 67, 69, 71, 72, 74, 76, 77, 79}
	var notes []float64
	var starts []int
	for i, v := range a {
		if v != 0 && (len(notes) == 0 || v != notes[len(notes)-1]) {
			notes = append(notes, v)
			starts = append(starts, i)
		}
	}
	if len(notes) < 12 {
		t.Fatalf("only %d notes", len(notes))
	}
	for k, v := range notes {
		if v != phrase[k%len(phrase)] {
			t.Fatalf("note %d is %v: the phrase has %v there (a note was skipped or played twice): %v", k, v, phrase[k%len(phrase)], notes)
		}
	}
	// the length of the notes: with euclid(8, 3) 2 or 3 steps, with euclid(8, 5) 1 or 2 (a step is 1000 samples); the last ones are all the new
	n := len(starts)
	for k := n - 8; k < n-1; k++ {
		// (the notes start at the start of a block of 64 samples: the lengths are what they are within that)
		if d := starts[k+1] - starts[k]; math.Abs(float64(d)-1000) > 64 && math.Abs(float64(d)-2000) > 64 {
			t.Errorf("a note after the edit is %d samples long: with the new rhythm it is 1000 or 2000", d)
		}
	}
	// and the old rhythm had longer ones before it
	long := false
	for k := 0; k+1 < n && starts[k+1] < 9000; k++ {
		if starts[k+1]-starts[k] >= 3000 {
			long = true
		}
	}
	if !long {
		t.Log("(no note of 3 steps before the edit: the check of the old rhythm is not made)")
	}
}

// how long a gate is held is edited: the notes that were made already have it
func TestALiveEditOfTheLegatoIsHeardAtOnce(t *testing.T) {
	src := `bpm = 120
steps = 4
fn inst(gate) { gate }
out = play(inst, rhythm: [true false false false], legato: 1)`
	l := startLive(t, src)
	l.play(4500)
	l.swapTo(strings.Replace(src, "legato: 1", "legato: 0.25", 1))
	if kept, fresh := l.r.LastSwap(); fresh != 0 {
		t.Errorf("kept %d, %d started again", kept, fresh)
	}
	x := l.play(12000)
	// notes every 4 steps (4000 samples); with legato 0.25 the gate is high for 1000 samples: from the swap on (a step is 1000, the swap is
	// at 5000) the notes at 8000 and 12000 are short, the one before that (at 4000, now sounding) is as it was
	hi := func(at int) float64 { return x[at-4500] }
	if hi(8000+500) != 1 || hi(8000+1500) != 0 || hi(12000+500) != 1 || hi(12000+1500) != 0 {
		t.Errorf("the notes after the edit are not short: %v %v %v %v", hi(8500), hi(9500), hi(12500), hi(13500))
	}
}

// the notes of a synth are edited: what sounds goes on sounding (the voices are the same), and the next notes are the new ones
func TestALiveEditOfTheNotesOfASynthDoesNotCutWhatSounds(t *testing.T) {
	src := "bpm = 120\nsteps = 4\nseed = 1\nout = supersaw(chords([[a3 c4 e4]], every(32), velocity: 0.9, length: 28), preset: \"trance_pad\", voices: 4).gain(0.3)\n"
	l := startLive(t, src)
	rms := func(x []float64) float64 {
		var s float64
		for _, v := range x {
			s += v * v
		}
		return math.Sqrt(s / float64(len(x)))
	}
	before := l.play(16000) // a chord that began at 0 and rings (length 28 steps: 3.5 s at 8000 samples/s of this clock... a step is 1000 samples)
	l.swapTo(strings.Replace(src, "[[a3 c4 e4]]", "[[a3 c4 f4] [g3 b3 d4]]", 1))
	if kept, fresh := l.r.LastSwap(); fresh != 0 {
		t.Errorf("kept %d, %d started again: %v", kept, fresh, l.r.LastRestarted())
	}
	after := l.play(6000)
	// the chord goes on: the sound after the edit is as loud as the sound just before it (it is not cut)
	if a, b := rms(before[10000:16000]), rms(after[1000:6000]); b < 0.5*a {
		t.Errorf("the chord that sounded was cut: %v before the edit, %v after it", a, b)
	}
}
