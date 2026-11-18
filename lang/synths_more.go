package lang

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"harp/blocks"
	"harp/dsl"

	"github.com/almerlucke/psalm/synths/drumsynth"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/organ"
)

// organ, dx and drumSynth: the last synths of psalm. The organ is one instrument (every note is a key of it), the FM synth can read the patches of
// a DX7 (a SysEx file), and the drum synth plays drums that are made: a note is a drum, by its General MIDI key

var (
	sysexBanks = map[string][]dx.Patch{}
	sysexMu    sync.Mutex
)

// drumNames are the drums by name, as a key of General MIDI: the lowest key of each (drum("kick") is 35)
var drumKeys = func() map[string]int {
	m := map[string]int{}
	for key, name := range drumsynth.GeneralMIDI {
		if k, ok := m[name]; !ok || key < k {
			m[name] = key
		}
	}
	return m
}()

func init() {
	nan := def(math.NaN())
	add(&Func{Name: "organ", Result: TSig, Doc: "psalm's tonewheel organ, one instrument: every note is a key of it (a chord is several keys, a note is held for its length), with its drawbars, percussion, vibrato and Leslie. The preset (harp presets organ) sets the registration; drawbars is the registration written as organists do, a digit from 0 to 8 for each of the nine drawbars (\"888000000\" is a jazz sound), leslie is how much of the rotary speaker there is (0 to 1) and speed how fast it turns (0 stopped, 0.5 slow, 1 fast). Stereo out",
		Params: []Param{{"notes", TGenNotes, nil}, {"preset", TStr, defStr("jazz")}, {"drawbars", TStr, defStr("")}, {"leslie", TNum, nan}, {"speed", TNum, nan}},
		Call: func(a Args) (Value, error) {
			make, ok := organPresets[a.Str("preset")]
			if !ok {
				return Value{}, fmt.Errorf("there is no organ preset %q%s", a.Str("preset"), didYouMean(a.Str("preset"), presetNames(organPresets)))
			}
			cfg := make()
			if d := a.Str("drawbars"); d != "" {
				digits := 0
				for _, r := range d {
					switch {
					case r >= '0' && r <= '8':
						digits++
					case r == ' ' || r == '-':
					default:
						return Value{}, fmt.Errorf("drawbars are digits from 0 to 8, like \"888000000\" (not %q)", r)
					}
				}
				if digits != 9 {
					return Value{}, fmt.Errorf("drawbars are nine digits, one for each drawbar (here %d)", digits)
				}
				cfg.Drawbars = organ.Bars(d)
			}
			for _, s := range []struct {
				param string
				to    *float64
			}{{"leslie", &cfg.Leslie}, {"speed", &cfg.LeslieSpeed}} {
				if x := a.Num(s.param); !math.IsNaN(x) {
					if x < 0 || x > 1 {
						return Value{}, fmt.Errorf("%s is from 0 to 1", s.param)
					}
					*s.to = x
				}
			}
			what := fmt.Sprintf("%s | %s | %v | %v", a.Str("preset"), a.Str("drawbars"), cfg.Leslie, cfg.LeslieSpeed)
			return sigV(blocks.Organ(cfg, what, a.value("notes").V.(dsl.Gen[[]blocks.Note]))), nil
		}})

	add(&Func{Name: "dx", Result: TSig, Doc: "psalm's FM synth in the way of the Yamaha DX7 and TX81Z: operators that modulate each other (electric pianos, bells, brass, basses, pads, ...), playing notes with voices notes at once. The preset (harp presets dx) is one of its own patches; file is a DX7 SysEx bank (a .syx file of 32 patches) and patch the number of the patch in it, counted from 0, which takes the place of the preset. Stereo out",
		Params: []Param{{"notes", TGenNotes, nil}, {"preset", TStr, defStr("e_piano")}, {"voices", TNum, def(8)}, {"file", TStr, defStr("")}, {"patch", TNum, def(0)}},
		Call: func(a Args) (Value, error) {
			voices := a.Int("voices")
			if voices < 1 || voices > 64 {
				return Value{}, fmt.Errorf("voices is from 1 to 64")
			}
			var patch dx.Patch
			var what string
			if f := a.Str("file"); f != "" {
				path, err := a.ev.findFile(f)
				if err != nil {
					return Value{}, err
				}
				key := path + " | " + stamp(path)
				sysexMu.Lock()
				bank, ok := sysexBanks[key]
				sysexMu.Unlock()
				if !ok {
					if bank, err = dx.LoadSysEx(path); err != nil {
						return Value{}, fmt.Errorf("%s: %v", filepath.Base(path), err)
					}
					sysexMu.Lock()
					sysexBanks[key] = bank
					sysexMu.Unlock()
				}
				n := a.Int("patch")
				if n < 0 || n >= len(bank) {
					return Value{}, fmt.Errorf("%s has %d patches: patch is from 0 to %d", filepath.Base(path), len(bank), len(bank)-1)
				}
				patch = bank[n]
				what = fmt.Sprintf("%s | %d", key, n)
				a.ev.patch.Info = append(a.ev.patch.Info, fmt.Sprintf("  %s: %d patches, playing %d: %s", filepath.Base(path), len(bank), n, patch.Name))
			} else {
				make, ok := dxPresets[a.Str("preset")]
				if !ok {
					return Value{}, fmt.Errorf("there is no dx preset %q%s", a.Str("preset"), didYouMean(a.Str("preset"), presetNames(dxPresets)))
				}
				patch = make()
				what = "preset " + a.Str("preset")
			}
			return sigV(blocks.DX(patch, what, voices, a.value("notes").V.(dsl.Gen[[]blocks.Note]))), nil
		}})

	kitNames := presetNames(drumsynth.Kits)
	add(&Func{Name: "drumSynth", Result: TSig, Doc: "psalm's drum synthesizer: drums that are made, not played from recordings. kit is " + strings.Join(kitNames, ", ") + ". A note is a drum, by its General MIDI key: drum(\"kick\") is the number for a kick, drum(\"snare\"), drum(\"hat\"), drum(\"open\"), drum(\"clap\"), drum(\"tomLow\") and so on (the names are in docs/synths.md); melody([drum(\"kick\")], every(4)) plays it. The voices are made together, so that a closed hat stops an open one. Stereo out",
		Params: []Param{{"notes", TGenNotes, nil}, {"kit", TStr, defStr("808")}, {"voices", TNum, def(8)}},
		Call: func(a Args) (Value, error) {
			kit, ok := drumsynth.Kits[a.Str("kit")]
			if !ok {
				return Value{}, fmt.Errorf("there is no kit %q: one of %s%s", a.Str("kit"), strings.Join(kitNames, ", "), didYouMean(a.Str("kit"), kitNames))
			}
			voices := a.Int("voices")
			if voices < 1 || voices > 64 {
				return Value{}, fmt.Errorf("voices is from 1 to 64")
			}
			cfg := drumsynth.DefaultConfig()
			cfg.Kit = kit()
			return sigV(blocks.DrumSynth(cfg, "kit "+a.Str("kit"), voices, a.value("notes").V.(dsl.Gen[[]blocks.Note]))), nil
		}})
	var names []string
	for n := range drumKeys {
		names = append(names, n)
	}
	sort.Strings(names)
	add(&Func{Name: "drum", Result: TNum, Doc: "the General MIDI key of a drum, by its name, for drumSynth: " + strings.Join(names, ", "),
		Params: []Param{{"name", TStr, nil}},
		Call: func(a Args) (Value, error) {
			k, ok := drumKeys[a.Str("name")]
			if !ok {
				return Value{}, fmt.Errorf("there is no drum %q: one of %s%s", a.Str("name"), strings.Join(names, ", "), didYouMean(a.Str("name"), names))
			}
			return numV(float64(k)), nil
		}})
}
