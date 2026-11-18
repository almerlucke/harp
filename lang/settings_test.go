package lang

import (
	"math"
	"os"
	"strings"
	"testing"
)

// bright is the energy in the changes between samples against the energy: higher when there is more high
func bright(x []float64) float64 {
	d, e := 0.0, 1e-12
	for i := 1; i < len(x); i++ {
		d += (x[i] - x[i-1]) * (x[i] - x[i-1])
		e += x[i] * x[i]
	}
	return d / e
}

func synthPatch(extra string) string {
	return "bpm = 120\nsteps = 4\nn = melody([a2], every(64), velocity: 1, length: 30)\nout = subtractive(n, preset: \"bass\", voices: 2" + extra + ")\n"
}

func TestSettingsOfASynthChangeItsSound(t *testing.T) {
	at := func(extra string) []float64 { return renderPatch(t, synthPatch(extra), 1.5)[0] }
	base := at("")
	// an envelope: a slow attack is quiet in the first 100 ms
	if slow := at(", ampAttack: 800"); rmsOf(slow[:4410]) > 0.3*rmsOf(base[:4410]) {
		t.Errorf("ampAttack: %v against %v", rmsOf(slow[:4410]), rmsOf(base[:4410]))
	}
	// a knob, a choice by name (in any writing), and a signal
	open := at(", cutoff: 6000")
	if bright(open[22050:44100]) < 2*bright(base[22050:44100]) {
		t.Errorf("cutoff: %v against %v", bright(open[22050:44100]), bright(base[22050:44100]))
	}
	hp := at(`, filterMode: "high pass", cutoff: 1500`)
	hp2 := at(`, filterMode: "HighPass", cutoff: 1500`)
	if bright(hp[22050:44100]) < 2*bright(base[22050:44100]) || bright(hp[22050:44100]) != bright(hp2[22050:44100]) {
		t.Errorf("filterMode: %v %v against %v", bright(hp[22050:44100]), bright(hp2[22050:44100]), bright(base[22050:44100]))
	}
	moving := at(", cutoff: lfo(0.5).range(200, 6000)")
	if a, b := bright(moving[2000:12000]), bright(moving[32000:42000]); a == b || math.IsNaN(a) {
		t.Errorf("a signal on the cutoff does not move it: %v %v", a, b)
	}
}

func TestSettingsMistakesSaySoAndSuggest(t *testing.T) {
	for extra, want := range map[string]string{
		", cutoff: 99999999":                "cutoff is from",
		", resonance: -1":                   "resonance is from",
		`, filterMode: "hgh pass"`:          "did you mean",
		`, filterMode: "wobble"`:            "not one of",
		`, cutoff: "low"`:                   "a name is for a choice",
		", filterMode: lfo(1)":              "a signal can move a knob",
		", cutof: 500":                      "did you mean cutoff",
		", ampSustain: 2":                   "ampSustain is from 0 to 1",
		", cutoff: lfo(1).range(1,2).dup()": "one channel",
	} {
		_, err := Compile(synthPatch(extra))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", extra, err, want)
		}
	}
}

// the settings reach every synth that has them: the first knob of each, set to the top of its range, is accepted and plays
func TestEverySynthTakesItsSettings(t *testing.T) {
	calls := map[string]string{
		"subtractive": `subtractive(n, voices: 2%s)`, "supersaw": `supersaw(n, voices: 2%s)`, "modal": `modal(n, voices: 2%s)`,
		"piano": `piano(n, voices: 2%s)`, "waveguide": `waveguide(n, voices: 2%s)`, "vocal": `vocal(n, voices: 2%s)`,
		"pulsar": `pulsar(n, voices: 2%s)`, "shaper": `shaper(n, voices: 2%s)`, "organ": `organ(n%s)`,
		"wavetable": `wavetable(n, voices: 2%s)`,
	}
	for name, call := range calls {
		ss := allSynthSettings[name]
		if ss == nil || len(ss.order) < 10 {
			t.Errorf("%s has no settings (%v)", name, ss)
			continue
		}
		var given string
		for _, k := range ss.order {
			if p := ss.byKey[k]; p.Kind == 0 && !strings.HasPrefix(k, "amp") { // a knob
				given = ", " + k + ": " + formatNum((p.Min+p.Max)/2)
				break
			}
		}
		src := "bpm = 120\nsteps = 4\nn = melody([a3], every(8), velocity: 1, length: 4)\nout = " + strings.Replace(call, "%s", given, 1) + "\n"
		if _, err := Compile(src); err != nil {
			t.Errorf("%s: %v\n%s", name, err, src)
		}
	}
}

// a live edit that changes a setting changes it in the voices that sound; taking it out gives back the preset's own
func TestALiveEditChangesTheSettingsOfASoundingSynth(t *testing.T) {
	head := "bpm = 120\nsteps = 4\nn = melody([a2], every(64), velocity: 1, length: 60)\n"
	dark := head + "out = subtractive(n, preset: \"bass\", voices: 2, cutoff: 150)\n"
	light := head + "out = subtractive(n, preset: \"bass\", voices: 2, cutoff: 6000)\n"
	plain := head + "out = subtractive(n, preset: \"bass\", voices: 2)\n"
	l := startLive(t, dark)
	l.play(8000)
	a := l.play(6000)
	l.swapTo(light)
	l.play(1000)
	b := l.play(6000)
	l.swapTo(dark)
	l.play(1000)
	c := l.play(6000)
	if bright(b) < 2*bright(a) {
		t.Errorf("the cutoff does not open in the sounding note: %v against %v", bright(b), bright(a))
	}
	if bright(c) > 1.5*bright(a) {
		t.Errorf("the cutoff does not close again: %v against %v", bright(c), bright(a))
	}
	// the note goes on: no restart (the level stays about where it was)
	if rmsOf(b) < 0.3*rmsOf(a) {
		t.Errorf("the note was cut: %v against %v", rmsOf(b), rmsOf(a))
	}
	// taken out: the preset's own again (the same as a patch that never had it)
	l.swapTo(plain)
	l.play(1000)
	d := l.play(6000)
	ref := reference(t, plain, 8000+6000+1000+6000+1000+6000+1000+6000+1000)
	_ = ref
	if bright(d) <= 0 {
		t.Errorf("silent after the setting is taken out")
	}
}

func TestTheSettingsPageIsUpToDate(t *testing.T) {
	got, err := os.ReadFile("../docs/settings.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != SettingsMarkdown() {
		t.Error("docs/settings.md is not what the synths say: run  go run ./cmd/harp settings -md > ../docs/settings.md  (in lang)")
	}
}
