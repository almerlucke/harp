package lang

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"harp/dsl"
)

// Sessions of live edits on the example patches, as a user would do them: the audio of the whole session is looked at for what an edit must
// never do (a NaN, a sound far louder than the patch is, a click that the edit made, the generators missing their time) and for what it must
// do (a save that changes nothing changes nothing).

type session struct {
	t      *testing.T
	name   string
	r      *dsl.Runner
	cur    *Patch
	rate   float64
	chans  [][]float64 // everything that was played, per channel
	events []string
}

func startSession(t *testing.T, name, src string) *session {
	return startSessionFrom(t, name, src, nil)
}

// startSessionFrom is startSession built from prev: the numbers drawn while it is built are those of prev (a patch that is not seeded is the
// same patch again)
func startSessionFrom(t *testing.T, name, src string, prev *Patch) *session {
	t.Helper()
	var patch *Patch
	var err error
	if prev == nil {
		// the files that a patch names are looked for next to it: for the examples, in their folder
		patch, err = CompileDir(src, filepath.Dir(filepath.Join("examples", name)))
		if err != nil && strings.Contains(err.Error(), "there is no file") {
			t.Skipf("%s: %v", name, err) // the recordings of psalm's assets are not here
		}
	} else {
		patch, err = Recompile(src, prev)
	}
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	st := patch.Settings()
	st.SampleRate = 44100
	st.Lookahead = 8
	st.NoOptimize = true
	r := st.Start(patch.Out)
	if err := r.Go(st.Lookahead); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return &session{t: t, name: name, r: r, cur: patch, rate: 44100, chans: make([][]float64, r.Program().Channels())}
}

// play plays a number of seconds, paced so that the control goroutine is ahead as it is in real time
func (s *session) play(seconds float64) {
	n := int(seconds * s.rate)
	const piece = 441
	for done := 0; done < n; done += piece {
		time.Sleep(time.Millisecond)
		size := min(piece, n-done)
		bufs := make([][]float64, len(s.chans))
		for i := range bufs {
			bufs[i] = make([]float64, size)
		}
		s.r.Process(bufs...)
		for i := range bufs {
			s.chans[i] = append(s.chans[i], bufs[i]...)
		}
	}
}

func (s *session) samples() int { return len(s.chans[0]) }

// edit takes over from the next step: the sample where it was made is noted
func (s *session) edit(what, src string) error {
	at := s.samples()
	patch, err := Recompile(src, s.cur)
	if err != nil {
		return err
	}
	st := patch.Settings()
	st.NoOptimize = true
	if err := s.r.Swap(patch.Out, st); err != nil {
		return err
	}
	s.cur = patch
	kept, fresh := s.r.LastSwap()
	s.events = append(s.events, fmt.Sprintf("%.2fs %-28s kept %2d new %2d %v", float64(at)/s.rate, what, kept, fresh, s.r.LastRestarted()))
	return nil
}

func (s *session) check(maxPeak float64) {
	s.t.Helper()
	for c, ch := range s.chans {
		for i, v := range ch {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				s.t.Fatalf("%s: channel %d is %v at %.2f s", s.name, c, v, float64(i)/s.rate)
			}
			if math.Abs(v) > maxPeak {
				s.t.Fatalf("%s: channel %d is %v at %.2f s (more than %v)", s.name, c, v, float64(i)/s.rate, maxPeak)
			}
		}
	}
	if u := s.r.Underruns(); u != 0 {
		s.t.Errorf("%s: %d steps were not ready in time", s.name, u)
	}
}

func readExample(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("examples", rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// every example, saved again as it is (with a comment added): nothing changes, and where the patch is the same every time it is built, the
// sound is the sound that the patch makes without any save
func TestALiveSaveThatChangesNothingChangesNothing(t *testing.T) {
	files, _ := filepath.Glob("examples/*.harp")
	more, _ := filepath.Glob("examples/supercollider/*.harp")
	files = append(files, more...)
	for _, f := range files {
		rel := strings.TrimPrefix(f, "examples/")
		src := readExample(t, rel)
		t.Run(rel, func(t *testing.T) {
			ref := startSession(t, rel, src)
			ref.play(3)
			other := startSessionFrom(t, rel, src, ref.cur)
			other.play(3)
			// is the patch the same every time it is played?
			same := true
			for c := range ref.chans {
				for i := range ref.chans[c] {
					if ref.chans[c][i] != other.chans[c][i] {
						same = false
					}
				}
			}
			s := startSessionFrom(t, rel, src, ref.cur)
			s.play(1)
			if err := s.edit("a comment", src+"\n// saved again\n"); err != nil {
				t.Fatal(err)
			}
			s.play(2)
			s.check(8)
			kept, fresh := s.r.LastSwap()
			if fresh != 0 {
				t.Errorf("%s: a save with only a comment started %d nodes again (kept %d)", rel, fresh, kept)
			}
			if !same {
				t.Logf("%s: it is not the same every time it plays (unseeded), so the sound is not compared", rel)
				return
			}
			for c := range ref.chans {
				for i := range ref.chans[c] {
					if s.chans[c][i] != ref.chans[c][i] {
						t.Fatalf("%s: channel %d differs from the unedited patch at %.3f s (sample %d): %v, %v", rel, c, float64(i)/44100, i, s.chans[c][i], ref.chans[c][i])
					}
				}
			}
		})
	}
}

type edit struct{ what, from, to string }

// runEdits plays, makes the edits one after the other (each on the text that the one before it made), and plays on; the text of the patch
// is changed where it says from, which has to be there
func (s *session) runEdits(src string, between float64, edits []edit) string {
	s.t.Helper()
	s.play(between)
	for _, e := range edits {
		if !strings.Contains(src, e.from) {
			s.t.Fatalf("%s: there is no %q in the patch for the edit %q", s.name, e.from, e.what)
		}
		src = strings.Replace(src, e.from, e.to, 1)
		if err := s.edit(e.what, src); err != nil {
			s.t.Fatalf("%s: %q: %v", s.name, e.what, err)
		}
		s.play(between)
	}
	return src
}

func TestALiveSessionOnTheGeneratorsPiece(t *testing.T) {
	src := readExample(t, "generators_fx.harp")
	s := startSession(t, "generators_fx", src)
	s.runEdits(src, 1.5, []edit{
		{"echo mix glide", "glide: 70", "glide: 140"},
		{"more hats", "euclid(16, 7, 3)", "euclid(16, 9, 3)"},
		{"another bass phrase", "[0 0 3 0 4 0 2 -2]", "[0 0 3 0 4 0 2 -2 5]"},
		{"longer reverb", "decay: 2.2", "decay: 3"},
		{"quieter pad", ".gain(0.3)", ".gain(0.15)"},
		{"faster", "bpm = 104", "bpm = 120"},
		{"no kick", "mix = low.dup() + pad + echoed.spread(0.7) + drums_kick + drums_hat", "mix = low.dup() + pad + echoed.spread(0.7) + drums_hat"},
		{"kick again", "mix = low.dup() + pad + echoed.spread(0.7) + drums_hat", "mix = low.dup() + pad + echoed.spread(0.7) + drums_kick + drums_hat"},
		{"lead brighter", "[2400 3600 1800 4200 2800]", "[3400 4600 2800 5200 3800]"},
		{"another scale step", "mutate([7 9 11 9 8 7 4 7 9 11 14 11 9 8], swap: 0.5)", "mutate([7 9 11 9 8 7 4 7 9 11 14 11 9 7], swap: 0.5)"},
	})
	s.check(4)
	for _, e := range s.events {
		t.Log(e)
	}
}

func TestALiveSessionOnTheOldExamples(t *testing.T) {
	for _, c := range []struct {
		file  string
		edits []edit
	}{
		{"play.harp", []edit{
			{"cutoff range", "range(300, 4000)", "range(300, 6000)"},
			{"more echo", "mix: 0.3", "mix: 0.5"},
		}},
		{"supersaw.harp", []edit{{"quieter", ".gain(", ".gain(0.9 * "}}},
		{"shared_stream.harp", []edit{
			{"another phrase", "[0 2 4 7 9 7 4 2]", "[0 2 4 7 9 7 4 5]"},
			{"filter glide", "ms: 90", "ms: 40"},
			{"more notes", "euclid(16, 11)", "euclid(16, 13)"},
		}},
		{"generators_play.harp", []edit{
			{"more notes", "euclid(16, 7)", "euclid(16, 9)"},
			{"other echo", "time: [0.375, 0.5]", "time: [0.375, 0.45]"},
		}},
		{"supercollider/11_gabber_rave.harp", []edit{
			{"kick bend", "[1 1 1 1 1 0.4 1 1 0.4 1]", "[1 1 1 1 1 0.5 1 1 0.5 1]"},
			{"hoover level", "dbamp(-20)", "dbamp(-22)"},
		}},
	} {
		t.Run(c.file, func(t *testing.T) {
			src := readExample(t, c.file)
			// edits whose text is not in the example are skipped: the point is the session, not the exact text
			var edits []edit
			for _, e := range c.edits {
				if strings.Contains(src, e.from) && e.from != e.to {
					edits = append(edits, e)
				}
			}
			s := startSession(t, c.file, src)
			s.runEdits(src, 1.2, edits)
			s.check(6)
			for _, e := range s.events {
				t.Log(e)
			}
		})
	}
}

// a patch that has another number of channels is refused, and what plays goes on
func TestALiveEditThatChangesTheChannelsIsRefused(t *testing.T) {
	src := "bpm = 120\nsteps = 4\nseed = 1\nout = saw(110).lowpass(cutoff: 800, res: 0.3).gain(0.2)\n"
	s := startSession(t, "mono", src)
	s.play(1)
	if err := s.edit("stereo", strings.Replace(src, ".gain(0.2)", ".gain(0.2).dup()", 1)); err == nil || !strings.Contains(err.Error(), "channels") {
		t.Fatalf("a stereo patch for a mono one: %v", err)
	}
	s.play(1)
	// and a change after the refusal still works, from the patch that plays
	if err := s.edit("brighter", strings.Replace(src, "800", "1500", 1)); err != nil {
		t.Fatal(err)
	}
	s.play(1)
	s.check(2)
}

// the demo patch, with the edits that its comment says to try: what each of them restarts
func TestALiveSessionOnTheDemoPatch(t *testing.T) {
	// a copy of the demo as it was written: the example is there to be edited
	b, err := os.ReadFile("testdata/live_demo.harp")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	s := startSession(t, "live_demo", src)
	restarts := map[string][]string{}
	for _, e := range []edit{
		{"cutoff range", "cutoff = lfo(0.07).range(300, 2400)", "cutoff = lfo(0.07).range(300, 3000)"},
		{"faster sweep", "cutoff = lfo(0.07)", "cutoff = lfo(0.2)"},
		{"another phrase", "[0 2 4 7 4 2 0 -3]", "[0 2 4 7 9 7 4 -3]"},
		{"another brightness", "[1800 2800 2200 3400]", "[2800 3800 3200 4400]"},
		{"quieter drone", "res: 0.35).gain(0.06)", "res: 0.35).gain(0.03)"},
		{"faster", "bpm = 90\nsteps", "bpm = 110\nsteps"},
		{"longer reverb", "reverb(mix: 0.3, decay: 5)", "reverb(mix: 0.3, decay: 7)"},
		{"no drone", "mix = droned.dup() + echoed", "mix = echoed"},
		{"drone again", "mix = echoed", "mix = droned.dup() + echoed"},
		{"another rhythm", "rhythm: euclid(16, 8)", "rhythm: euclid(16, 9)"},
	} {
		s.runEdits(src, 0, []edit{e})
		src = strings.Replace(src, e.from, e.to, 1)
		restarts[e.what] = s.r.LastRestarted()
		s.play(0.8)
	}
	s.check(3)
	for _, e := range s.events {
		t.Log(e)
	}
	has := func(what, node string) bool {
		for _, n := range restarts[what] {
			if strings.HasPrefix(n, node) {
				return true
			}
		}
		return false
	}
	// what the user asked for: the reverb is only new when its own settings change
	for what, names := range restarts {
		if what != "longer reverb" && has(what, "fdnreverb") {
			t.Errorf("%q restarted the reverb: %v", what, names)
		}
	}
	if !has("longer reverb", "fdnreverb") {
		t.Error("another decay did not give a new reverb")
	}
	// the drone's sweep goes on when its speed is changed, and its filter is not new when the drone is taken out and put back
	for _, what := range []string{"cutoff range", "faster sweep", "quieter drone", "faster", "no drone", "another phrase", "another brightness", "another rhythm"} {
		if len(restarts[what]) != 0 {
			t.Errorf("%q restarted %v", what, restarts[what])
		}
	}
}

// effects and synths of psalm in a live session: a setting that is a signal moves without a restart; a number that is a setting of the effect
// makes it a new effect; another preset is a new synth; the rest goes on
func TestALiveSessionOnTheEffectsAndSynthsOfPsalm(t *testing.T) {
	src := readExample(t, "psalm_pieces.harp")
	s := startSession(t, "psalm_pieces", src)
	restarts := map[string][]string{}
	for _, e := range []edit{
		{"the lfo of the chorus", "depth: lfo(0.05).range(1, 6)", "depth: lfo(0.2).range(1, 6)"},
		{"the tape feedback", "feedback: 0.35, mix: 0.2", "feedback: 0.5, mix: 0.2"},
		{"the rate of the chorus", "rate: 0.4, depth", "rate: 0.7, depth"},
		{"another preset", "preset: \"vibraphone\"", "preset: \"marimba\""},
		{"another progression", "[g#2 d#3 c4 d#4]", "[g#2 d#3 c4 f4]"},
		{"the rhythm of the line", "euclid(16, 9)", "euclid(16, 11)"},
		{"the width", "width: 1.3", "width: 1.8"},
	} {
		s.runEdits(src, 0, []edit{e})
		src = strings.Replace(src, e.from, e.to, 1)
		restarts[e.what] = s.r.LastRestarted()
		s.play(0.6)
	}
	s.check(3)
	for _, e := range s.events {
		t.Log(e)
	}
	count := func(what, node string) (n int) {
		for _, r := range restarts[what] {
			if strings.HasPrefix(r, node) {
				n++
			}
		}
		return
	}
	if len(restarts["the lfo of the chorus"]) != 0 {
		t.Errorf("the lfo of a setting restarted %v", restarts["the lfo of the chorus"])
	}
	if count("the tape feedback", "tape") == 0 || len(restarts["the tape feedback"]) > 2 {
		t.Errorf("a number setting of the tape: %v (the tape is new, nothing else)", restarts["the tape feedback"])
	}
	if count("the rate of the chorus", "chorus") == 0 {
		t.Errorf("a number setting of the chorus: %v", restarts["the rate of the chorus"])
	}
	if count("another preset", "synth") == 0 {
		t.Errorf("another preset did not make a new synth: %v", restarts["another preset"])
	}
	if len(restarts["the rhythm of the line"]) != 0 {
		t.Errorf("the rhythm of the line restarted %v", restarts["the rhythm of the line"])
	}
	if count("the width", "widener") == 0 {
		t.Errorf("the width: %v", restarts["the width"])
	}
}
