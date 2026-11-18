package lang

import (
	"fmt"
	"strings"

	"harp/blocks"
	"harp/dsl"

	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/shaper"
	"github.com/almerlucke/psalm/synths/vocal"
	"github.com/almerlucke/psalm/synths/waveguide"
)

// the synths of psalm that play notes: one function each, with the presets of the synth by name (presets_gen.go, made by internal/genpresets)

func registerSynth[C any](name, doc, preset string, presets map[string]func() C, play func(C, int, dsl.Gen[[]blocks.Note]) dsl.Sig) {
	add(&Func{Name: name, Result: TSig, Doc: doc + ". Stereo out. The presets are in docs/presets.md (harp presets " + name + ")",
		Params: []Param{{"notes", TGenNotes, nil}, {"preset", TStr, defStr(preset)}, {"voices", TNum, def(8)}},
		Call: func(a Args) (Value, error) {
			make, ok := presets[a.Str("preset")]
			if !ok {
				return Value{}, fmt.Errorf("there is no %s preset %q%s", name, a.Str("preset"), didYouMean(a.Str("preset"), presetNames(presets)))
			}
			voices := a.Int("voices")
			if voices < 1 || voices > 64 {
				return Value{}, fmt.Errorf("voices is from 1 to 64")
			}
			return sigV(play(make(), voices, a.value("notes").V.(dsl.Gen[[]blocks.Note]))), nil
		}})
}

func init() {
	registerSynth("modal", "psalm's modal synth: struck and bowed bars, bells, bottles, cymbals and gongs as sets of resonances (marimba, vibraphone, church bell, ...), playing notes with voices notes at once", "marimba", modalPresets, blocks.Modal)
	registerSynth("piano", "psalm's piano: hammers, strings and a soundboard, playing notes with voices notes at once", "grand", pianoPresets, blocks.Piano)
	registerSynth("waveguide", "psalm's physical models: plucked and bowed strings, flutes, drums and more as waves going up and down a line, playing notes with voices notes at once", "guitar", waveguidePresets, blocks.Waveguide)
	registerSynth("vocal", "psalm's singing voice: a glottis and a vocal tract (solo, choir, humming, chants, ...), playing notes with voices notes at once", "solo", vocalPresets, blocks.Vocal)
	registerSynth("pulsar", "psalm's pulsar synthesis: trains of short pulses with formants (vowel leads, buzz bass, insects, ...), playing notes with voices notes at once", "vowel_lead", pulsarPresets, blocks.Pulsar)
	registerSynth("shaper", "psalm's waveshaping and phase distortion synth (cz leads, fm basses, pads, ...), playing notes with voices notes at once", "acid", shaperPresets, blocks.Shaper)
}

// presetTables are all the presets by the name of the synth: what harp presets prints
var presetTables = map[string][]string{}

func init() {
	for name, names := range map[string][]string{
		"subtractive": presetNames(subtractivePresets),
		"supersaw":    presetNames(supersawPresets),
		"modal":       presetNames(modalPresets),
		"piano":       presetNames(pianoPresets),
		"waveguide":   presetNames(waveguidePresets),
		"vocal":       presetNames(vocalPresets),
		"pulsar":      presetNames(pulsarPresets),
		"shaper":      presetNames(shaperPresets),
		"grains":      presetNames(grainPresets),
		"wavetable":   presetNames(wavetablePresets),
	} {
		presetTables[name] = names
	}
}

var _ = modal.Config{}
var _ = piano.Config{}
var _ = pulsar.Config{}
var _ = shaper.Config{}
var _ = vocal.Config{}
var _ = waveguide.Config{}

// PresetsMarkdown lists the presets of every synth (docs/presets.md)
func PresetsMarkdown() string {
	var b strings.Builder
	b.WriteString("# Presets of the synths\n\n<!-- made by `harp presets -md`: do not edit by hand (a test says when it is out of date) -->\n\n")
	b.WriteString("A synth takes `preset: \"name\"`: `modal(notes, preset: \"church_bell\")`. The names are those of the presets of psalm, in snake case.\n")
	for _, synth := range []string{"subtractive", "supersaw", "modal", "piano", "waveguide", "vocal", "pulsar", "shaper", "grains", "wavetable"} {
		names := presetTables[synth]
		fmt.Fprintf(&b, "\n## %s (%d)\n\n%s\n", synth, len(names), strings.Join(names, ", "))
	}
	return b.String()
}

// Presets lists the presets of one synth (all of them if the name is empty)
func Presets(synth string) (string, error) {
	if synth == "" {
		return PresetsMarkdown(), nil
	}
	names, ok := presetTables[synth]
	if !ok {
		return "", fmt.Errorf("there is no synth %q%s", synth, didYouMean(synth, presetNames(presetTables)))
	}
	return strings.Join(names, "\n") + "\n", nil
}
