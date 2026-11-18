package lang

import (
	"strings"
	"testing"
)

func TestGrainsPlayARecording(t *testing.T) {
	dir := t.TempDir()
	writeTone(t, dir, "tone.wav", 2)
	src := "bpm = 120\nnotes = melody([60], every(64), velocity: 1, length: 12)\nout = grains(notes, \"tone.wav\", preset: \"%s\"%s)\n"
	for _, p := range []string{"choir_pad", "pluck", "texture"} {
		x, _ := renderIn(t, dir, strings.Replace(strings.Replace(src, "%s", p, 1), "%s", "", 1), 2)
		if energy(x[4000:40000]) < 1e-4 {
			t.Errorf("preset %s is silent: %v", p, energy(x[4000:40000]))
		}
	}
	// the arguments of the preset are taken, and a wrong name says what there is
	if x, _ := renderIn(t, dir, strings.Replace(strings.Replace(src, "%s", "choir_pad", 1), "%s", ", position: 0.2, scan: 0, density: 80, size: 60", 1), 1); energy(x[4000:40000]) < 1e-4 {
		t.Error("with arguments it is silent")
	}
	_, err := CompileDir(strings.Replace(strings.Replace(src, "%s", "choir_pat", 1), "%s", "", 1), dir)
	if err == nil || !strings.Contains(err.Error(), "did you mean choir_pad") {
		t.Errorf("a wrong preset: %v", err)
	}
	if _, err := CompileDir(strings.Replace(strings.Replace(src, "%s", "pluck", 1), "%s", "", 1), t.TempDir()); err == nil || !strings.Contains(err.Error(), "there is no file") {
		t.Errorf("no file: %v", err)
	}
}

func TestAWavetablePlaysATable(t *testing.T) {
	dir := t.TempDir()
	src := "bpm = 120\nnotes = melody([57], every(64), velocity: 1, length: 12)\nout = wavetable(notes, %s)\n"
	for _, args := range []string{`table: "basic"`, `table: "sync", preset: "reese", position: 0.6`, `table: "fm", frames: 16, amount: 4, preset: "bass"`, `table: "vowels"`} {
		x, _ := renderIn(t, dir, strings.Replace(src, "%s", args, 1), 2)
		if energy(x[4000:40000]) < 1e-4 {
			t.Errorf("%s is silent", args)
		}
	}
	// a table from a file: a recording of single cycles (a sine of 2048 samples a cycle, four cycles) is a table of four frames
	writeTone(t, dir, "cycles.wav", 1)
	if x, _ := renderIn(t, dir, strings.Replace(src, "%s", `table: "cycles.wav", size: 1024`, 1), 2); energy(x[4000:40000]) < 1e-4 {
		t.Error("a table from a file is silent")
	}
	if _, err := CompileDir(strings.Replace(src, "%s", `table: "nothing.wav"`, 1), dir); err == nil || !strings.Contains(err.Error(), "or one of") {
		t.Errorf("a table that is not there: %v", err)
	}
	if _, err := CompileDir(strings.Replace(src, "%s", `preset: "pdd"`, 1), dir); err == nil || !strings.Contains(err.Error(), "wavetable preset") {
		t.Errorf("a wrong preset: %v", err)
	}
}

func TestSnakeCase(t *testing.T) {
	for in, want := range map[string]string{"glassKeys": "glass_keys", "fmBass": "fm_bass", "pad": "pad", "subBass": "sub_bass", "wobbleBass": "wobble_bass"} {
		if got := snakeCase(in); got != want {
			t.Errorf("%s: %s", in, got)
		}
	}
}

// the position of a wavetable that is a signal moves the table while a note sounds: from a sine to a bright saw
func TestAWavetableMovesWithASignal(t *testing.T) {
	// a long note; brightness is the energy of the difference of the samples against the energy: a sine has little, a saw a lot
	brightness := func(x []float64) float64 {
		var d, e float64
		for i := 1; i < len(x); i++ {
			d += (x[i] - x[i-1]) * (x[i] - x[i-1])
			e += x[i] * x[i]
		}
		return d / e
	}
	note := "notes = melody([a3], every(64), velocity: 1, length: 60)\n"
	play := func(args string) []float64 {
		return renderPatch(t, "bpm = 120\n"+note+"out = wavetable(notes, table: \"brightness\", preset: \"pad\", cutoff: 12000, "+args+", voices: 2)\n", 3.5)[0]
	}
	early := func(x []float64) float64 { return brightness(x[20000:40000]) }  // the position is about 0.2 here
	late := func(x []float64) float64 { return brightness(x[120000:140000]) } // and about 0.9 here
	moving, at0, at1 := play("position: line(0, 1, 3)"), play("position: 0"), play("position: 1")
	if early(moving) < 3*early(at0) {
		t.Errorf("early, the table is already a little open: %v against %v at position 0", early(moving), early(at0))
	}
	if d := late(moving) / late(at1); d < 0.8 || d > 1.2 {
		t.Errorf("late, the table is open: %v against %v at position 1", late(moving), late(at1))
	}
	if early(moving) > 0.9*early(at1) {
		t.Errorf("early, it is not yet as open as position 1: %v against %v", early(moving), early(at1))
	}
	// the cutoff moves: closed at first, open later
	cut := renderPatch(t, "bpm = 120\n"+note+"out = wavetable(notes, table: \"brightness\", preset: \"pad\", position: 1, cutoff: line(200, 12000, 3), voices: 2)\n", 3.5)[0]
	if brightness(cut[120000:140000]) < 2*brightness(cut[20000:40000]) {
		t.Errorf("the cutoff opens: %v then %v", brightness(cut[20000:40000]), brightness(cut[120000:140000]))
	}
	// warp, detune, spread and sub are signals from 0 to 1 as well, and a warp mode can be asked for
	for _, args := range []string{`warp: lfo(0.5).range(0, 1), warpmode: "sync"`, `detune: lfo(0.3).range(0, 1)`, `spread: lfo(0.3).range(0, 1), sub: lfo(0.2).range(0, 1)`} {
		x := renderPatch(t, "bpm = 120\n"+note+"out = wavetable(notes, table: \"sync\", preset: \"pad\", "+args+", voices: 2)\n", 1.5)[0]
		if energy(x[10000:60000]) < 1e-5 {
			t.Errorf("%s is silent", args)
		}
	}
	for src, want := range map[string]string{
		`wavetable(notes, position: [lfo(1), lfo(2)])`: "one channel",
		`wavetable(notes, warpmode: "twist")`:          "one of",
		`wavetable(notes, position: "high")`:           "position",
	} {
		_, err := Compile("notes = melody([60], every(4))\nout = " + src + "\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}
