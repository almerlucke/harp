package lang

import (
	"fmt"
	"strings"

	"harp/blocks"
	"harp/dsl"

	"github.com/almerlucke/psalm/params"
)

// The effects of psalm, as functions: one for each of blocks.Effects(), made from what the effect describes of itself (params.Set): its
// settings are its named parameters, with the range and the default that psalm gives, and each can be a number (set when the patch is built)
// or a signal (it moves the setting while it plays; read once in every 64 samples or less).

var effectDocs = map[string]string{
	"chorus":     "a chorus: copies of the sound whose delay moves, so that it thickens and widens",
	"compressor": "a compressor with a soft knee and a sidechain: give a signal as sidechain to duck by it (a kick that makes the bass go down)",
	"crusher":    "a bit crusher and sample rate reducer",
	"eq":         "an equalizer: low and high cuts, a low and a high shelf, two bells",
	"flanger":    "a flanger: a short delay that sweeps, with feedback",
	"freeverb":   "the Freeverb reverb (comb and allpass filters), stereo",
	"gate":       "a gate or expander with a sidechain: it turns the sound down when it is quiet",
	"granulator": "a granular effect: the sound cut into small grains that are played again, scattered, shifted or frozen",
	"phaser":     "a phaser: allpass stages that sweep, with feedback",
	"pingpong":   "a delay that bounces from one side to the other",
	"ringmod":    "a ring modulator, or a frequency shifter (mode: shift)",
	"rotary":     "a rotary speaker (Leslie): a horn and a drum that turn, slow or fast",
	"saturate":   "saturation: soft, tube, hard clipping or folding, with drive and tone",
	"shimmer":    "a reverb whose tail goes up an octave (or another interval) again and again",
	"tape":       "a tape delay with wow, flutter and saturation",
	"transient":  "a transient shaper: more or less attack and sustain",
	"tremolo":    "a tremolo (the level moves) or an auto pan (the place moves)",
	"vocoder":    "a vocoder: the spectrum of the modulator (a voice) on the carrier (a synth)",
	"widener":    "a stereo widener, that keeps the lows in the middle",
}

func init() {
	for _, info := range blocks.Effects() {
		registerEffect(info)
	}
}

func registerEffect(info blocks.EffectInfo) {
	set := info.Describe()
	var ps []Param
	var inputNames []string
	switch {
	case info.Name == "vocoder":
		inputNames = []string{"carrier", "modulator"}
		ps = []Param{{"carrier", TSig, nil}, {"modulator", TSig, nil}}
	case len(info.Inputs) == 2:
		inputNames = []string{"x", "sidechain"}
		ps = []Param{{"x", TSig, nil}, {"sidechain", TAny, def(0)}}
	default:
		inputNames = []string{"x"}
		ps = []Param{{"x", TSig, nil}}
	}
	byKey := map[string]params.Param{}
	var doc []string
	for _, p := range set.Params {
		byKey[p.Key] = p
		d := def(p.Value)
		ps = append(ps, Param{Name: p.Key, Type: TAny, Default: d})
		doc = append(doc, describeSetting(p))
	}
	text := effectDocs[info.Name]
	if text == "" {
		text = "an effect of psalm"
	}
	text += ". The sound comes out in stereo (a mono signal goes to both sides). Settings: " + strings.Join(doc, "; ") + ". A setting can be a number or a signal that moves it"
	if info.Blocks {
		text += ". It works in blocks of 64 samples, so the sound is 1.5 ms late"
	}
	name := info.Name
	if now, ok := renamedFuncs[name]; ok { // pingpong is pingPong: the effect keeps psalm's name, the function is camelCase
		name = now
	}
	add(&Func{Name: name, Result: TSig, Doc: text, Params: ps, Call: func(a Args) (Value, error) {
		ins := make([]dsl.Sig, len(info.Inputs))
		for k, name := range inputNames {
			v := a.value(name)
			if k > 0 && v.T.Kind == KNum { // a sidechain that is not given
				continue
			}
			c, err := coerce(v, TSig)
			if err != nil {
				return Value{}, fmt.Errorf("%s: %w", name, err)
			}
			ins[k] = c.Sig()
		}
		fixed := blocks.Settings{}
		mods := map[string]dsl.Sig{}
		for _, p := range set.Params {
			v := a.value(p.Key)
			if v.T.Kind == KNum && v.Float() == p.Value {
				continue // not given, or given as it is
			}
			switch {
			case v.T.Kind == KNum:
				fixed[p.Key] = v.Float()
			case v.T.Kind == KBool:
				if p.Kind != params.Switch {
					return Value{}, fmt.Errorf("%s: true or false is for a switch", p.Key)
				}
				fixed[p.Key] = map[bool]float64{false: 0, true: 1}[v.V.(bool)]
			case v.T.Kind == KStr:
				if p.Kind != params.Choice {
					return Value{}, fmt.Errorf("%s: a name is for a choice (%s is a number)", p.Key, p.Key)
				}
				idx := -1
				for i, o := range p.Options {
					if strings.EqualFold(o, v.V.(string)) {
						idx = i
					}
				}
				if idx < 0 {
					return Value{}, fmt.Errorf("%s: %q is not one of %s", p.Key, v.V.(string), strings.Join(p.Options, ", "))
				}
				fixed[p.Key] = float64(idx)
			default:
				if p.Kind != params.Knob {
					return Value{}, fmt.Errorf("%s: a signal can move a knob, not a switch or a choice", p.Key)
				}
				c, err := coerce(v, TSig)
				if err != nil {
					return Value{}, fmt.Errorf("%s: %w", p.Key, err)
				}
				mods[p.Key] = dsl.Mono(c.Sig())
			}
		}
		return sigV(blocks.Effect(info, ins, fixed, mods)), nil
	}})
}

// describeSetting is a setting for the documentation: key, range, unit and default
func describeSetting(p params.Param) string {
	switch p.Kind {
	case params.Switch:
		return fmt.Sprintf("%s (true or false, %v)", p.Key, p.Value != 0)
	case params.Choice:
		def := ""
		if int(p.Value) < len(p.Options) {
			def = p.Options[int(p.Value)]
		}
		return fmt.Sprintf("%s (%s; %s)", p.Key, strings.Join(p.Options, ", "), def)
	}
	unit := ""
	if p.Unit != "" {
		unit = " " + p.Unit
	}
	return fmt.Sprintf("%s (%g to %g%s, %g)", p.Key, p.Min, p.Max, unit, p.Value)
}
