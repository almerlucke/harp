package lang

import (
	"fmt"
	"sort"
	"strings"
)

// the groups of the reference (docs/reference.md): every function is in one, in the order that a person looks for them
var referenceGroups = []struct {
	title string
	names string
}{
	{"Oscillators and noise", "sine saw square triangle pulse rectangle ramp lfo noise lfnoise dust impulse"},
	{"Envelopes and ramps", "adsr on env decay decay2 line xline"},
	{"Filters", "lowpass highpass bandpass bpf"},
	{"Delays and reverb", "echo comb reverb"},
	{"Effects of psalm", "chorus flanger phaser tremolo rotary widener ringmod crusher saturate tape pingpong shimmer freeverb granulator compressor gate transient eq vocoder"},
	{"Shaping and mapping", "gain tanh atan sign clip2 midiratio hz range exprange linexp"},
	{"Channels", "dup mix spread pan product reverse"},
	{"Generators (rhythm, notes, phrases)", "euclid every mutate degrees melody chords notes"},
	{"Rhythms", "maybe invert both either meter sieve"},
	{"Combining generators", "alternate anyof chance stutter shotgun norepeat distinct flatten stack deal uniform"},
	{"Numbers that move", "wander drift wave sweep orbit chaos remap gauss expo beta cauchy weibull pink"},
	{"Rules, chance and growth", "lsystem grammar cellular life scatter markov learn stroll"},
	{"Scales, chords and lines", "scale equal just stepwise snap chord triads voicelead arp progression tonerow rowform counterpoint"},
	{"Number sequences, time and structure", "fibonacci thuemorse primes collatz divisible divide talea ratchets jump split metric sections evolve tendency"},
	{"Scores (notes with a start and a length)", "phrase midifile harmonize isorhythm chordtones fugue phasing canon transpose stretch mirror retro shift together cut tile pad tune thin pitches lengths velocities onsets chordsof"},
	{"Generators as signals", "seq glide pulses"},
	{"Notes on an instrument of your own", "play accents"},
	{"Synths", "subtractive supersaw modal piano waveguide vocal pulsar shaper grains wavetable"},
	{"Recordings", "sampler streamer"},
	{"While the patch is built (random numbers, lists, functions)", "rand exprand coin choose geom fill repeat dbamp"},
	{"SuperCollider's own algorithms", "scsaw scsawsync scsin sclpf schpf scrlpf scpeq scdelayn scdelayc freeverb2"},
}

// FunctionsMarkdown is the reference of the functions, in the groups above, as markdown (docs/reference.md is made of it:
// harp functions -md)
func FunctionsMarkdown() string {
	var b strings.Builder
	b.WriteString("# Reference: the functions\n\n")
	b.WriteString("<!-- made by `harp functions -md`: do not edit by hand (a test says when it is out of date) -->\n\n")
	b.WriteString("Every function can be called as `f(x, ...)` or as `x.f(...)`: the thing before the dot is the first argument. ")
	b.WriteString("A parameter with `= value` can be left out. Arguments can be named: `lowpass(x, cutoff: 800)`. ")
	b.WriteString("The kinds (signal, generator, ...) are explained in [types](types.md).\n")
	seen := map[string]bool{}
	for _, g := range referenceGroups {
		fmt.Fprintf(&b, "\n## %s\n", g.title)
		for _, n := range strings.Fields(g.names) {
			f, ok := funcs[n]
			if !ok {
				panic("lang: the reference has a group with a function that does not exist: " + n)
			}
			seen[n] = true
			fmt.Fprintf(&b, "\n### `%s`\n\n`%s`\n\n%s\n", n, describe(f), capital(f.Doc))
		}
	}
	var left []string
	for n := range funcs {
		if !seen[n] {
			left = append(left, n)
		}
	}
	if len(left) > 0 {
		sort.Strings(left)
		panic("lang: functions that are in no group of the reference (docs.go): " + strings.Join(left, ", "))
	}
	return b.String()
}

func capital(s string) string {
	if s == "" {
		return s
	}
	s = strings.ToUpper(s[:1]) + s[1:]
	if !strings.HasSuffix(s, ".") {
		s += "."
	}
	return s
}
