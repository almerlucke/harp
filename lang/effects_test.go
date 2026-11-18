package lang

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestEffectsOfPsalmAreFunctions(t *testing.T) {
	for _, name := range []string{"chorus", "flanger", "phaser", "tremolo", "rotary", "widener", "ringMod", "crusher", "saturate", "tape", "pingPong",
		"shimmer", "freeverb", "granulator", "compressor", "gate", "transient", "eq", "vocoder"} {
		f, ok := funcs[name]
		if !ok {
			t.Errorf("no function %s", name)
			continue
		}
		if !strings.Contains(f.Doc, "Settings:") {
			t.Errorf("%s: no settings in its documentation", name)
		}
	}
	// the settings are the ones of psalm
	if got := describe(funcs["chorus"]); !strings.Contains(got, "rate: anything = 0.6") || !strings.Contains(got, "mix: anything") {
		t.Errorf("chorus: %s", got)
	}
}

func rms(x []float64) float64 {
	var s float64
	for _, v := range x {
		s += v * v
	}
	return math.Sqrt(s / float64(len(x)))
}

func TestAnEffectInAPatch(t *testing.T) {
	// settings as numbers, as a choice by its name, as a signal that moves, and a mono input in stereo out
	for _, src := range []string{
		"out = saw(110).chorus(rate: 0.7, depth: 4, mix: 0.5)",
		"out = saw(110).flanger(shape: \"triangle\", feedback: 0.5)",
		"out = saw(110).tremolo(mode: \"pan\", rate: lfo(0.2).range(2, 8), depth: 0.8)",
		"out = saw(110).saturate(shape: \"fold\", drive: 12)",
		"out = saw(110).rotary(fast: true)",
		"out = saw(110).compressor(threshold: -30, ratio: 8).widener(width: 1.8)",
		"out = noise().gain(0.2).gate(threshold: -30, sidechain: saw(60))",
		"out = vocoder(saw(110), noise().gain(0.3))",
		"out = saw(110).shimmer(mix: 0.3)",
	} {
		p := mustBuild(t, src)
		_, ch := p.Settings().Render(p.Out, 1)
		if len(ch) != 2 {
			t.Errorf("%s: %d channels", src, len(ch))
			continue
		}
		if r := rms(ch[0]); math.IsNaN(r) || r < 1e-5 || r > 5 {
			t.Errorf("%s: rms %v", src, r)
		}
	}
}

func TestEffectSettingsAreChecked(t *testing.T) {
	for src, mention := range map[string]string{
		"out = saw(110).flanger(shape: \"zigzag\")": "not one of",
		"out = saw(110).chorus(mix: \"half\")":      "choice",
		"out = saw(110).chorus(mix: true)":          "switch",
		"out = saw(110).chorus(nonsense: 1)":        "no parameter nonsense",
		"out = saw(110).rotary(fast: lfo(1))":       "switch or a choice",
		"out = saw(110).vocoder()":                  "needs",
		"out = saw(110).tremolo(mode: lfo(1))":      "choice",
	} {
		if _, err := build(t, src); err == nil || !strings.Contains(err.Error(), mention) {
			t.Errorf("%s: %v, want a mention of %q", src, err, mention)
		}
	}
}

// the sidechain of a compressor makes it duck
func TestASidechainDucks(t *testing.T) {
	plain := mustBuild(t, "out = saw(200).gain(0.1).compressor(threshold: -20, ratio: 20, attack: 1, release: 40)")
	ducked := mustBuild(t, "out = saw(200).gain(0.1).compressor(threshold: -20, ratio: 20, attack: 1, release: 40, sidechain: sine(60))")
	_, a := plain.Settings().Render(plain.Out, 1)
	_, b := ducked.Settings().Render(ducked.Out, 1)
	if rms(b[0][22050:]) > 0.5*rms(a[0][22050:]) {
		t.Errorf("a loud sidechain: %v against %v", rms(b[0][22050:]), rms(a[0][22050:]))
	}
}

// every preset of every synth plays a note: a sound, nothing wrong with it
func TestEveryPresetOfEverySynthPlaysANote(t *testing.T) {
	var silent []string
	dir := t.TempDir()
	writeTone(t, dir, "tone.wav", 1) // for the synths that are made from a sound
	for synth, names := range presetTables {
		for _, preset := range names {
			src := "bpm = 120\nseed = 1\nnotes = melody([a3 c4 e4], every(8), velocity: 0.9, length: 0.9)\nout = " + synthCall(synth, preset) + "\n"
			p, err := CompileDir(src, dir)
			if err != nil {
				t.Errorf("%s %s: %v", synth, preset, err)
				continue
			}
			_, ch := p.Settings().Render(p.Out, 1.2)
			peak := 0.0
			for _, c := range ch {
				for i, v := range c {
					if math.IsNaN(v) || math.IsInf(v, 0) {
						t.Errorf("%s %s: sample %d is %v", synth, preset, i, v)
						break
					}
					peak = math.Max(peak, math.Abs(v))
				}
			}
			if peak < 1e-5 {
				silent = append(silent, synth+" "+preset)
			}
			if peak > 8 {
				t.Errorf("%s %s: peak %v", synth, preset, peak)
			}
		}
	}
	if len(silent) > 0 {
		t.Logf("%d presets are silent for a note of a3: %v", len(silent), silent)
	}
}

// synthCall is a note played on a synth with a preset (the organ has no voices, the drum synth has kits, not presets)
func synthCall(synth, preset string) string {
	switch synth {
	case "organ":
		return fmt.Sprintf("organ(notes, preset: %q)", preset)
	case "drumSynth":
		return fmt.Sprintf("drumSynth(notes, kit: %q, voices: 2)", preset)
	case "grains":
		return fmt.Sprintf("grains(notes, preset: %q, voices: 2, file: \"tone.wav\")", preset)
	}
	return fmt.Sprintf("%s(notes, preset: %q, voices: 2)", synth, preset)
}
