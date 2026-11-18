package lang

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"harp/blocks"
	"harp/dsl"

	"github.com/almerlucke/psalm/params"
)

// The settings of the synths of psalm, as parameters of the functions: every synth describes its settings (psalm's Describe: the key of the message
// that changes each, its range, its unit and its choices), and each is an optional parameter with that name. A setting is a number (set in
// every voice), a name for a choice (filterMode: "band pass"), true or false for a switch, or, for a knob, a signal that moves it while the notes
// sound (cutoff: lfo(0.1).range(300, 3000)). The envelopes are ampAttack, ampDecay, ampSustain and ampRelease (milliseconds, sustain 0 to 1), and
// the same for the other envelopes of the synth (filterAttack, positionDecay...). What is not given is the preset's own.
//
// The modulation matrix (mod1Src...) and the very large sets (the operators of dx, the drums of drumSynth) are not here.

var matrixKey = regexp.MustCompile(`^mod[0-9]+(Src|Dest|Amount)$`)

// synthSettings is what a synth function knows of its settings
type synthSettings struct {
	base     params.Set
	name     string
	order    []string
	byKey    map[string]params.Param
	matrix   bool                            // the synth has a modulation matrix: the parameter mods
	describe func(a Args) (params.Set, bool) // the settings as the preset of this call has them; false when the preset is not known (the call says so)
}

var allSynthSettings = map[string]*synthSettings{}

var envStages = map[byte]string{'A': "Attack", 'D': "Decay", 'S': "Sustain", 'R': "Release"}

// envParam is a setting of an envelope as a setting of a knob
func envParam(prefix, group string, stage byte) params.Param {
	p := params.Param{Key: prefix + envStages[stage], Group: group, Kind: params.Knob, Min: 0.1, Max: 60000, Unit: "ms", Exp: true}
	if stage == 'S' {
		p.Min, p.Max, p.Unit, p.Exp = 0, 1, "", false
	}
	return p
}

func envValue(e params.Envelope, stage byte) float64 {
	return map[byte]float64{'A': e.Attack, 'D': e.Decay, 'S': e.Sustain, 'R': e.Release}[stage]
}

// attachSettings gives a synth function the settings that base describes of the synth as parameters; describe gives them as the preset of a call has them.
// taken are the settings that the function has as parameters of its own
func attachSettings(name string, base params.Set, describe func(a Args) (params.Set, bool), taken ...string) {
	f := funcs[name]
	if f == nil {
		panic("lang: settings for a function that is not there: " + name)
	}
	have := map[string]bool{}
	for _, p := range f.Params {
		have[p.Name] = true
	}
	for _, k := range taken {
		have[k] = true
	}
	ss := &synthSettings{base: base, name: name, byKey: map[string]params.Param{}, describe: describe}
	add := func(p params.Param) {
		if matrixKey.MatchString(p.Key) || have[p.Key] {
			return
		}
		ss.order = append(ss.order, p.Key)
		ss.byKey[p.Key] = p
		f.Params = append(f.Params, Param{Name: p.Key, Type: TAny, Default: def(math.NaN())})
	}
	nbase := len(f.Params)
	for _, p := range base.Params {
		if p.Key == "mod1Src" {
			ss.matrix = true
		}
		add(p)
	}
	if ss.matrix && !have["mods"] {
		ss.order = append(ss.order, "mods")
		f.Params = append(f.Params, Param{Name: "mods", Type: TStr, Default: defStr("")})
	}
	for _, e := range base.Envelopes {
		for i := 0; i < len(e.Stages); i++ {
			add(envParam(e.Prefix, e.Group, e.Stages[i]))
		}
	}
	f.Settings, f.NBase = name, nbase
	f.Doc += fmt.Sprintf(". Every setting of the synth is a parameter (%d of them, with their ranges: harp settings %s, docs/settings.md), a number, a name or true/false, or for a knob a signal that moves it; the envelopes are ampAttack, ampDecay, ampSustain, ampRelease and the others of the synth; what is not given is the preset's own", len(ss.order), name)
	allSynthSettings[name] = ss
	inner := f.Call
	f.Call = func(a Args) (Value, error) {
		set, ok := describe(a)
		if !ok {
			return inner(a) // the preset is not there: the call says so
		}
		opt, err := ss.read(a, set)
		if err != nil {
			return Value{}, err
		}
		a.synthOpts = append(a.synthOpts, opt)
		return inner(a)
	}
}

// norm is a name without the spaces, the hyphens and the capitals: "low pass", "lowPass" and "Low-Pass" are the same
func norm(s string) string {
	return strings.ToLower(strings.NewReplacer(" ", "", "-", "", "_", "").Replace(s))
}

// read makes the options of the host from the settings that are given in a call
func (ss *synthSettings) read(a Args, set params.Set) (blocks.SynthOption, error) {
	given, defaults := blocks.Settings{}, blocks.Settings{}
	for _, p := range set.Params {
		defaults[p.Key] = p.Value
	}
	for _, e := range set.Envelopes {
		for i := 0; i < len(e.Stages); i++ {
			defaults[e.Prefix+envStages[e.Stages[i]]] = envValue(e, e.Stages[i])
		}
	}
	var keyed []blocks.KeyedMod
	if ss.matrix {
		if text := a.Str("mods"); strings.TrimSpace(text) != "" {
			msg, err := readMods(text, set)
			if err != nil {
				return nil, fmt.Errorf("mods: %w", err)
			}
			for k, v := range msg {
				given[k] = v
			}
		}
	}
	for _, key := range ss.order {
		if key == "mods" {
			continue
		}
		v := a.value(key)
		if v.T.Kind == KNum && math.IsNaN(v.Float()) {
			continue // not given: the preset's own
		}
		p := ss.byKey[key]
		switch {
		case v.T.Kind == KNum:
			x := v.Float()
			if p.Kind == params.Switch && x != 0 && x != 1 {
				return nil, fmt.Errorf("%s is a switch: true or false", key)
			}
			if x < p.Min-1e-9 || x > p.Max+1e-9 || math.IsInf(x, 0) {
				if p.Kind == params.Choice {
					return nil, fmt.Errorf("%s is one of %s (a name, or a number from 0 to %d: %g is not)", key, strings.Join(p.Options, ", "), len(p.Options)-1, x)
				}
				return nil, fmt.Errorf("%s is from %g to %g%s: %g is not", key, p.Min, p.Max, unitText(p), x)
			}
			given[key] = x
		case v.T.Kind == KBool:
			if p.Kind != params.Switch {
				return nil, fmt.Errorf("%s: true or false is for a switch", key)
			}
			given[key] = map[bool]float64{false: 0, true: 1}[v.V.(bool)]
		case v.T.Kind == KStr:
			if p.Kind != params.Choice {
				return nil, fmt.Errorf("%s: a name is for a choice, and this is a number from %g to %g", key, p.Min, p.Max)
			}
			idx := -1
			for i, o := range p.Options {
				if norm(o) == norm(v.V.(string)) {
					idx = i
				}
			}
			if idx < 0 {
				return nil, fmt.Errorf("%s: %q is not one of %s%s", key, v.V.(string), strings.Join(p.Options, ", "), didYouMean(v.V.(string), p.Options))
			}
			given[key] = float64(idx)
		default:
			if p.Kind != params.Knob {
				return nil, fmt.Errorf("%s: a signal can move a knob, not a switch or a choice", key)
			}
			c, err := coerce(v, TSig)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			if c.Sig().Channels() != 1 {
				return nil, fmt.Errorf("%s: a signal of one channel (this has %d)", key, c.Sig().Channels())
			}
			keyed = append(keyed, blocks.KeyedMod{Key: key, Sig: c.Sig()})
		}
	}
	return blocks.WithSettings(given, defaults, keyed), nil
}

func unitText(p params.Param) string {
	if p.Unit == "" {
		return ""
	}
	return " " + p.Unit
}

// SettingsMarkdown lists the settings of every synth, by group (docs/settings.md)
func SettingsMarkdown() string {
	var b strings.Builder
	b.WriteString("# Settings of the synths\n\n<!-- made by `harp settings -md`: do not edit by hand (a test says when it is out of date) -->\n\n")
	b.WriteString("Every setting of a synth is an optional parameter of its function, with the name psalm gives it: `subtractive(notes, preset: \"pad\", cutoff: 900, filterMode: \"band pass\", ampAttack: 400)`. A knob is a number, or a signal that moves it while the notes sound (`cutoff: lfo(0.1).range(300, 3000)`); a choice is a name (or its number); a switch is `true` or `false`. What is not given is the preset's own. The envelopes are a setting for each stage, in milliseconds (`ampAttack`, `ampDecay`, `ampSustain` from 0 to 1, `ampRelease`).\n\n")
	for _, name := range synthSettingNames() {
		b.WriteString(synthSettingsMarkdown(name))
	}
	return b.String()
}

func synthSettingNames() []string {
	var names []string
	for _, n := range []string{"subtractive", "supersaw", "wavetable", "modal", "piano", "waveguide", "vocal", "pulsar", "shaper", "organ", "grains"} {
		if _, ok := allSynthSettings[n]; ok {
			names = append(names, n)
		}
	}
	return names
}

// synthSettingsMarkdown is the settings of one synth, group by group
func synthSettingsMarkdown(name string) string {
	ss := allSynthSettings[name]
	var b strings.Builder
	fmt.Fprintf(&b, "\n## %s (%d)\n", name, len(ss.order))
	if ss.matrix {
		src, dest := matrixOptions(ss)
		fmt.Fprintf(&b, "\n**Modulation matrix** (`mods`): sources %s; destinations %s\n", strings.Join(src, ", "), strings.Join(dest, ", "))
	}
	var groups []string
	byGroup := map[string][]string{}
	for _, k := range ss.order {
		g := ss.byKey[k].Group
		if _, ok := byGroup[g]; !ok {
			groups = append(groups, g)
		}
		byGroup[g] = append(byGroup[g], k)
	}
	for _, g := range groups {
		fmt.Fprintf(&b, "\n**%s**: ", g)
		var parts []string
		for _, k := range byGroup[g] {
			parts = append(parts, "`"+k+"` "+settingRange(ss.byKey[k]))
		}
		b.WriteString(strings.Join(parts, "; ") + "\n")
	}
	return b.String()
}

func settingRange(p params.Param) string {
	switch p.Kind {
	case params.Switch:
		return "(true or false)"
	case params.Choice:
		return "(" + strings.Join(p.Options, ", ") + ")"
	}
	return fmt.Sprintf("(%g to %g%s)", p.Min, p.Max, unitText(p))
}

var _ dsl.Sig

// SettingsOf lists the settings of one synth
func SettingsOf(synth string) (string, error) {
	if now, ok := renamedFuncs[synth]; ok {
		synth = now
	}
	if _, ok := allSynthSettings[synth]; !ok {
		return "", fmt.Errorf("there is no synth %q with settings%s", synth, didYouMean(synth, synthSettingNames()))
	}
	return synthSettingsMarkdown(synth), nil
}

// readMods reads the modulation matrix as text: slots separated by ; or a line, each "source -> destination amount" (the amount from -1 to 1, 1 when
// it is left out): "lfo 1 -> cutoff 0.5; velocity -> level 0.3". The slots of the preset are replaced; starting with + the slots are added
// to the ones that the preset has, in the places that are free. It gives the messages of the slots (the numbers of the choices) and clears
// the ones that are not used
func readMods(text string, set params.Set) (map[string]float64, error) {
	byKey := map[string]params.Param{}
	for _, p := range set.Params {
		byKey[p.Key] = p
	}
	srcOpts, destOpts := byKey["mod1Src"].Options, byKey["mod1Dest"].Options
	slots := 0
	for i := 1; byKey[fmt.Sprintf("mod%dSrc", i)].Key != ""; i++ {
		slots = i
	}
	text = strings.TrimSpace(text)
	add := strings.HasPrefix(text, "+")
	text = strings.TrimPrefix(text, "+")
	out := map[string]float64{}
	next := 1
	if add {
		for next <= slots && byKey[fmt.Sprintf("mod%dSrc", next)].Value != 0 {
			next++
		}
	} else {
		for i := 1; i <= slots; i++ { // the slots of the preset are cleared
			out[fmt.Sprintf("mod%dSrc", i)] = 0
			out[fmt.Sprintf("mod%dDest", i)] = 0
			out[fmt.Sprintf("mod%dAmount", i)] = 0
		}
	}
	find := func(what, name string, opts []string) (float64, error) {
		for i, o := range opts {
			if i > 0 && norm(o) == norm(name) {
				return float64(i), nil
			}
		}
		return 0, fmt.Errorf("%s %q is not one of %s%s", what, name, strings.Join(opts[1:], ", "), didYouMean(name, opts[1:]))
	}
	for _, item := range strings.FieldsFunc(text, func(r rune) bool { return r == ';' || r == '\n' }) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		arrow := strings.Index(item, ">")
		if arrow < 0 {
			return nil, fmt.Errorf("%q: a slot is source -> destination amount, like \"lfo 1 -> cutoff 0.5\"", item)
		}
		src := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(item[:arrow]), "-"))
		rest := strings.Fields(item[arrow+1:])
		amount := 1.0
		if n := len(rest); n > 1 {
			if x, err := strconv.ParseFloat(rest[n-1], 64); err == nil {
				amount, rest = x, rest[:n-1]
			}
		}
		dest := strings.Join(rest, " ")
		if amount < -1 || amount > 1 {
			return nil, fmt.Errorf("%q: the amount is from -1 to 1", item)
		}
		if next > slots {
			return nil, fmt.Errorf("a synth has %d slots in its matrix", slots)
		}
		si, err := find("source", src, srcOpts)
		if err != nil {
			return nil, err
		}
		di, err := find("destination", dest, destOpts)
		if err != nil {
			return nil, err
		}
		out[fmt.Sprintf("mod%dSrc", next)] = si
		out[fmt.Sprintf("mod%dDest", next)] = di
		out[fmt.Sprintf("mod%dAmount", next)] = amount
		next++
	}
	return out, nil
}

// matrixOptions are the sources and the destinations of the matrix of a synth, from its first preset
func matrixOptions(ss *synthSettings) (src, dest []string) {
	for _, p := range ss.base.Params {
		switch p.Key {
		case "mod1Src":
			src = p.Options[1:]
		case "mod1Dest":
			dest = p.Options[1:]
		}
	}
	return
}
