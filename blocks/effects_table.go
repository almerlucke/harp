package blocks

import (
	"sort"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/effects/chorus"
	"github.com/almerlucke/psalm/modules/effects/compressor"
	"github.com/almerlucke/psalm/modules/effects/crusher"
	"github.com/almerlucke/psalm/modules/effects/eq"
	"github.com/almerlucke/psalm/modules/effects/flanger"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/gate"
	"github.com/almerlucke/psalm/modules/effects/granulator"
	"github.com/almerlucke/psalm/modules/effects/phaser"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/effects/ringmod"
	"github.com/almerlucke/psalm/modules/effects/rotary"
	"github.com/almerlucke/psalm/modules/effects/saturate"
	"github.com/almerlucke/psalm/modules/effects/shimmer"
	"github.com/almerlucke/psalm/modules/effects/tape"
	"github.com/almerlucke/psalm/modules/effects/transient"
	"github.com/almerlucke/psalm/modules/effects/tremolo"
	"github.com/almerlucke/psalm/modules/effects/vocoder"
	"github.com/almerlucke/psalm/modules/effects/widener"
	"github.com/almerlucke/psalm/params"
)

const (
	stereo = psalm.Stereo
	mono   = psalm.Mono
)

// effects are the effects of psalm that can be hosted (Effect), by name. What each takes is what it describes of itself.
var effects = map[string]EffectInfo{
	"chorus":     {Inputs: []int{stereo}, New: func() psalm.Modular { return chorus.New(chorus.DefaultConfig()) }, Set: func() params.Set { return chorus.Describe(chorus.DefaultConfig()) }},
	"compressor": {Inputs: []int{stereo, stereo}, New: func() psalm.Modular { return compressor.New(compressor.DefaultConfig()) }, Set: func() params.Set { return compressor.Describe(compressor.DefaultConfig()) }},
	"crusher":    {Inputs: []int{stereo}, New: func() psalm.Modular { return crusher.New(crusher.DefaultConfig()) }, Set: func() params.Set { return crusher.Describe(crusher.DefaultConfig()) }},
	"eq":         {Inputs: []int{stereo}, New: func() psalm.Modular { return eq.New(eq.DefaultConfig()) }, Set: func() params.Set { return eq.Describe(eq.DefaultConfig()) }},
	"flanger":    {Inputs: []int{stereo}, New: func() psalm.Modular { return flanger.New(flanger.DefaultConfig()) }, Set: func() params.Set { return flanger.Describe(flanger.DefaultConfig()) }},
	"gate":       {Inputs: []int{stereo, stereo}, New: func() psalm.Modular { return gate.New(gate.DefaultConfig()) }, Set: func() params.Set { return gate.Describe(gate.DefaultConfig()) }},
	"granulator": {Inputs: []int{stereo}, Blocks: true, New: func() psalm.Modular { return granulator.New(granulator.DefaultConfig()) }, Set: func() params.Set { return granulator.Describe(granulator.DefaultConfig()) }},
	"phaser":     {Inputs: []int{stereo}, New: func() psalm.Modular { return phaser.New(phaser.DefaultConfig()) }, Set: func() params.Set { return phaser.Describe(phaser.DefaultConfig()) }},
	"pingpong":   {Inputs: []int{stereo}, New: func() psalm.Modular { return pingpong.New(pingpong.DefaultConfig()) }, Set: func() params.Set { return pingpong.Describe(pingpong.DefaultConfig()) }},
	"ringmod":    {Inputs: []int{stereo}, New: func() psalm.Modular { return ringmod.New(ringmod.DefaultConfig()) }, Set: func() params.Set { return ringmod.Describe(ringmod.DefaultConfig()) }},
	"rotary":     {Inputs: []int{stereo}, New: func() psalm.Modular { return rotary.New(rotary.DefaultConfig()) }, Set: func() params.Set { return rotary.Describe(rotary.DefaultConfig()) }},
	"saturate":   {Inputs: []int{stereo}, New: func() psalm.Modular { return saturate.New(saturate.DefaultConfig()) }, Set: func() params.Set { return saturate.Describe(saturate.DefaultConfig()) }},
	"shimmer":    {Inputs: []int{stereo}, Blocks: true, New: func() psalm.Modular { return shimmer.New(shimmer.DefaultConfig()) }, Set: func() params.Set { return shimmer.Describe(shimmer.DefaultConfig()) }},
	"tape":       {Inputs: []int{stereo}, New: func() psalm.Modular { return tape.New(tape.DefaultConfig()) }, Set: func() params.Set { return tape.Describe(tape.DefaultConfig()) }},
	"transient":  {Inputs: []int{stereo}, New: func() psalm.Modular { return transient.New(transient.DefaultConfig()) }, Set: func() params.Set { return transient.Describe(transient.DefaultConfig()) }},
	"tremolo":    {Inputs: []int{stereo}, New: func() psalm.Modular { return tremolo.New(tremolo.DefaultConfig()) }, Set: func() params.Set { return tremolo.Describe(tremolo.DefaultConfig()) }},
	"vocoder":    {Inputs: []int{mono, mono}, Blocks: true, New: func() psalm.Modular { return vocoder.New(vocoder.DefaultConfig()) }, Set: func() params.Set { return vocoder.Describe(vocoder.DefaultConfig()) }},
	"widener":    {Inputs: []int{stereo}, New: func() psalm.Modular { return widener.New(widener.DefaultConfig()) }, Set: func() params.Set { return widener.Describe(widener.DefaultConfig()) }},
	"freeverb": {Inputs: []int{stereo}, Blocks: true,
		New: func() psalm.Modular { return freeverb.New(0.5, 0.5, 1, 0.33, 0.7) },
		Set: func() params.Set { return freeverb.Describe(freeverb.New(0.5, 0.5, 1, 0.33, 0.7)) }},
}

// Effects lists the effects of psalm that can be hosted, by name, with their info (the name is set)
func Effects() []EffectInfo {
	names := make([]string, 0, len(effects))
	for n := range effects {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]EffectInfo, len(names))
	for i, n := range names {
		out[i] = effects[n]
		out[i].Name = n
	}
	return out
}
