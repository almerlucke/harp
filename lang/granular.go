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

	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/sndfile"
)

// grains and wavetable: the two synths that are made from a sound. Grains cuts a recording into short grains, a wavetable morphs through the
// frames of a table (from a file of single cycles, or one that psalm makes). Both have presets that take the recording or the table, so any of
// them works with any sound; what the preset sets can be changed by the arguments

var (
	grainSounds = map[string]sndfile.SoundFiler[float32]{}
	tables      = map[string]*wavetable.Table{}
	granularMu  sync.Mutex
)

func presetKeys[C any](m map[string]C) map[string]C {
	out := map[string]C{}
	for k, v := range m {
		out[snakeCase(k)] = v
	}
	return out
}

// snakeCase turns glassKeys into glass_keys, fmBass into fm_bass
func snakeCase(s string) string {
	var b strings.Builder
	rs := []rune(s)
	for i, r := range rs {
		upper := r >= 'A' && r <= 'Z'
		if upper && i > 0 && (rs[i-1] < 'A' || rs[i-1] > 'Z' || (i+1 < len(rs) && rs[i+1] >= 'a' && rs[i+1] <= 'z')) {
			b.WriteByte('_')
		}
		if upper {
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

var grainPresets = presetKeys(grains.Presets)
var wavetablePresets = presetKeys(wavetable.Presets)

// a table that psalm makes: the name, and what the argument `amount` is for it (its default is used when amount is 0)
var tableMakers = map[string]struct {
	amount float64
	make   func(frames int, amount float64) *wavetable.Table
}{
	"basic":      {0, func(int, float64) *wavetable.Table { return wavetable.Basic() }},
	"brightness": {0, func(n int, _ float64) *wavetable.Table { return gen.Brightness(n) }},
	"pwm":        {0.03, gen.PWM},
	"sync":       {8, gen.Sync},
	"fm":         {8, func(n int, x float64) *wavetable.Table { return gen.FM(n, 2, x) }},
	"fold":       {8, func(n int, x float64) *wavetable.Table { return gen.Fold(n, x, 0) }},
	"vowels":     {130, gen.Vowels},
	"drawbars":   {16, func(n int, x float64) *wavetable.Table { return gen.Drawbars(n, int(x)) }},
	"random":     {1, func(n int, x float64) *wavetable.Table { return gen.Random(n, 4, 32, uint64(x)) }},
	"filter":     {4, gen.Filter},
	"resonance":  {8, gen.Resonance},
	"odd_even":   {0, func(n int, _ float64) *wavetable.Table { return gen.OddEven(n) }},
	"comb":       {0.4, gen.Comb},
	"saturate":   {10, gen.Saturate},
	"quantize":   {4, func(n int, x float64) *wavetable.Table { return gen.Quantize(n, int(x)) }},
	"steps":      {4, func(n int, x float64) *wavetable.Table { return gen.Steps(n, int(x)) }},
}

func tableNames() []string {
	var n []string
	for k := range tableMakers {
		n = append(n, k)
	}
	sort.Strings(n)
	return n
}

// setIf sets what an argument gives, if it is a number (NaN is "the preset's own")
func setIf(a Args, name string, to *float64) {
	if x := a.Num(name); !math.IsNaN(x) {
		*to = x
	}
}

func init() {
	nan := def(math.NaN())
	add(&Func{Name: "grains", Result: TSig, Doc: "psalm's granular synth: every note is a stream of short grains of a recording (a sound file), played at the pitch of the note; root is the note at which the recording is at its own pitch. The preset (harp presets grains) sets the character: position (0-1, where in the recording), scan (how fast that moves: 0 holds, 1 is the original speed), spray (seconds of spread), density (grains a second), size (a grain, in ms), pitchSpread (semitones) are the preset's own unless given. Stereo out",
		Params: []Param{{"notes", TGenNotes, nil}, {"file", TStr, nil}, {"root", TNum, def(60)}, {"preset", TStr, defStr("choir_pad")}, {"voices", TNum, def(6)},
			{"position", TNum, nan}, {"scan", TNum, nan}, {"spray", TNum, nan}, {"density", TNum, nan}, {"size", TNum, nan}, {"pitchSpread", TNum, nan}},
		Call: func(a Args) (Value, error) {
			e := a.ev
			path, err := e.findFile(a.Str("file"))
			if err != nil {
				return Value{}, err
			}
			make, ok := grainPresets[a.Str("preset")]
			if !ok {
				return Value{}, fmt.Errorf("there is no grains preset %q%s", a.Str("preset"), didYouMean(a.Str("preset"), presetNames(grainPresets)))
			}
			voices := a.Int("voices")
			if voices < 1 || voices > 64 {
				return Value{}, fmt.Errorf("voices is from 1 to 64")
			}
			key := path + " | " + stamp(path)
			granularMu.Lock()
			sample, ok := grainSounds[key]
			granularMu.Unlock()
			if !ok {
				sample, err = sndfile.NewSoundFile[float32](path)
				if err != nil {
					return Value{}, fmt.Errorf("%s: %v", filepath.Base(path), err)
				}
				granularMu.Lock()
				grainSounds[key] = sample
				granularMu.Unlock()
			}
			cfg := make(sample, a.Num("root"))
			setIf(a, "position", &cfg.Position)
			setIf(a, "scan", &cfg.Scan)
			setIf(a, "spray", &cfg.Spray)
			setIf(a, "density", &cfg.Density)
			setIf(a, "size", &cfg.GrainSize)
			setIf(a, "pitchSpread", &cfg.PitchSpread)
			what := fmt.Sprintf("%s | root %g | %s | %v %v %v %v %v %v", key, a.Num("root"), a.Str("preset"), cfg.Position, cfg.Scan, cfg.Spray, cfg.Density, cfg.GrainSize, cfg.PitchSpread)
			return sigV(blocks.Grains(cfg, what, voices, a.value("notes").V.(dsl.Gen[[]blocks.Note]))), nil
		}})
	warpModes := map[string]wavetable.WarpMode{"off": wavetable.WarpOff, "bendup": wavetable.WarpBendUp, "benddown": wavetable.WarpBendDown, "sync": wavetable.WarpSync,
		"mirror": wavetable.WarpMirror, "fmb": wavetable.WarpFMB, "fmsine": wavetable.WarpFMSine}
	add(&Func{Name: "wavetable", Result: TSig, Doc: "psalm's wavetable synth: an oscillator morphing through the frames of a table, with unison, a sub oscillator and a filter. The table is a sound file of single cycles of size samples each (2048 is usual), or one that psalm makes: " + strings.Join(tableNames(), ", ") + " (frames of them, and amount: the number of their own, like the sync ratio, the fm index or the pulse width; 0 is the usual one). The preset (harp presets wavetable) sets the rest. These can be a number, or a signal that moves them while the notes sound (lfo(0.1).range(0, 1), env(...), seq(...)): position (0 to 1, where in the table: the table moves), cutoff (Hz) and resonance of the filter, and, as a signal from 0 to 1 that is the amount of the preset's own setting: warp (bends the cycle: warpMode is " + names(warpModes) + "), detune (up to 100 cents), spread (the stereo width of the unison), sub (the level of the sub oscillator); up to 4 of those four can be signals. A signal is one channel. Stereo out",
		Params: []Param{{"notes", TGenNotes, nil}, {"table", TStr, defStr("basic")}, {"preset", TStr, defStr("pad")}, {"voices", TNum, def(6)}, {"position", TAny, nan},
			{"cutoff", TAny, nan}, {"resonance", TAny, nan}, {"warp", TAny, nan}, {"warpMode", TStr, defStr("")}, {"detune", TAny, nan}, {"spread", TAny, nan}, {"sub", TAny, nan},
			{"size", TNum, def(2048)}, {"frames", TNum, def(32)}, {"amount", TNum, def(0)}},
		Call: func(a Args) (Value, error) {
			e := a.ev
			make, ok := wavetablePresets[a.Str("preset")]
			if !ok {
				return Value{}, fmt.Errorf("there is no wavetable preset %q%s", a.Str("preset"), didYouMean(a.Str("preset"), presetNames(wavetablePresets)))
			}
			voices := a.Int("voices")
			if voices < 1 || voices > 64 {
				return Value{}, fmt.Errorf("voices is from 1 to 64")
			}
			frames := a.Int("frames")
			if frames < 2 || frames > 256 {
				return Value{}, fmt.Errorf("frames is from 2 to 256")
			}
			name := a.Str("table")
			var key string
			var build func() (*wavetable.Table, error)
			if m, ok := tableMakers[name]; ok {
				amount := a.Num("amount")
				if amount == 0 {
					amount = m.amount
				}
				key = fmt.Sprintf("%s %d %g", name, frames, amount)
				build = func() (*wavetable.Table, error) { return m.make(frames, amount), nil }
			} else {
				path, err := e.findFile(name)
				if err != nil {
					return Value{}, fmt.Errorf("%v (a table is a file, or one of %s)", err, strings.Join(tableNames(), ", "))
				}
				size := a.Int("size")
				key = fmt.Sprintf("%s | %s | %d", path, stamp(path), size)
				build = func() (*wavetable.Table, error) { return wavetable.Load(path, size) }
			}
			granularMu.Lock()
			table, ok := tables[key]
			granularMu.Unlock()
			if !ok {
				var err error
				if table, err = build(); err != nil {
					return Value{}, fmt.Errorf("table %s: %v", filepath.Base(name), err)
				}
				granularMu.Lock()
				tables[key] = table
				granularMu.Unlock()
			}
			cfg := make(table)
			if m := a.Str("warpMode"); m != "" {
				mode, ok := warpModes[m]
				if !ok {
					return Value{}, fmt.Errorf("warpMode is one of %s%s", names(warpModes), didYouMean(m, presetNames(warpModes)))
				}
				cfg.WarpMode = mode
			}
			// each of the settings is a number (set in the voice) or a signal (cabled in: the position, the cutoff and the resonance have an input
			// of their own, the others go through a source of the modulation matrix that is moved by the signal)
			var mods []blocks.Mod
			var moved []string
			ext := 0
			setting := func(param string, direct int, dest string, set func(float64)) error {
				v := a.value(param)
				if v.T.Kind == KNum {
					if x := v.Float(); !math.IsNaN(x) {
						set(x)
					}
					return nil
				}
				sg, err := coerce(v, TSig)
				if err != nil {
					return fmt.Errorf("%s: %w", param, err)
				}
				if sg.Sig().Channels() != 1 {
					return fmt.Errorf("%s: a signal of one channel (this has %d)", param, sg.Sig().Channels())
				}
				index := direct
				if dest != "" {
					slot := -1
					for i, sl := range cfg.Mods {
						if sl.Dest == "" {
							slot = i
							break
						}
					}
					if ext >= modmatrix.Exts || slot < 0 {
						return fmt.Errorf("%s: no room for another signal in the modulation (the preset uses the slots, and there are 4 signals of this kind)", param)
					}
					cfg.Mods[slot] = modmatrix.Slot{Src: modmatrix.Ext1 + modmatrix.Source(ext), Dest: dest, Amount: 1}
					index = wavetable.InExt1 + ext
					ext++
				}
				mods = append(mods, blocks.Mod{Index: index, Sig: sg.Sig()})
				moved = append(moved, param)
				return nil
			}
			for _, s := range []struct {
				param  string
				direct int
				dest   string
				set    func(float64)
			}{
				{"position", wavetable.InPosition, "", func(x float64) { cfg.Position = x }},
				{"cutoff", wavetable.InCutoff, "", func(x float64) { cfg.Cutoff = x }},
				{"resonance", wavetable.InResonance, "", func(x float64) { cfg.Resonance = x }},
				{"warp", 0, "warp", func(x float64) { cfg.Warp = x }},
				{"detune", 0, "detune", func(x float64) { cfg.Detune = x }},
				{"spread", 0, "spread", func(x float64) { cfg.Spread = x }},
				{"sub", 0, "subLevel", func(x float64) { cfg.SubLevel = x }},
			} {
				if err := setting(s.param, s.direct, s.dest, s.set); err != nil {
					return Value{}, err
				}
			}
			what := fmt.Sprintf("%s | %s | %v %v %v %v %v %v %v | %s | %v", key, a.Str("preset"), cfg.Position, cfg.Cutoff, cfg.Resonance, cfg.Warp, cfg.Detune, cfg.Spread, cfg.SubLevel, a.Str("warpMode"), moved)
			return sigV(blocks.Wavetable(cfg, what, voices, a.value("notes").V.(dsl.Gen[[]blocks.Note]), mods)), nil
		}})
}
