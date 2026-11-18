package lang

import "fmt"

// Names are camelCase: a function or a parameter of two words has the second one with a capital (voiceLead, barSteps). These are the names that
// were written in one word before: a patch that has them still works, and harp check says what they are now

// renamedFuncs are the functions, by their old name
var renamedFuncs = map[string]string{
	"bysection": "bySection", "norepeat": "noRepeat", "voicelead": "voiceLead", "exprange": "expRange", "linexp": "linExp", "midiratio": "midiRatio",
	"dbamp": "dbAmp", "tonerow": "toneRow", "rowform": "rowForm", "chordtones": "chordTones", "chordsof": "chordsOf", "thuemorse": "thueMorse",
	"midifile": "midiFile", "drumsynth": "drumSynth", "pingpong": "pingPong", "ringmod": "ringMod", "lfnoise": "lfNoise", "lowpass": "lowPass",
	"highpass": "highPass", "bandpass": "bandPass", "scsawsync": "scSawSync", "scsaw": "scSaw", "scsin": "scSin", "sclpf": "scLpf", "schpf": "scHpf",
	"scrlpf": "scRlpf", "scpeq": "scPeq", "scdelayn": "scDelayN", "scdelayc": "scDelayC", "exprand": "expRand", "anyof": "anyOf",
}

// renamedParams are the parameters, by their old name
var renamedParams = map[string]string{
	"barsteps": "barSteps", "beatsteps": "beatSteps", "chordsteps": "chordSteps", "loopstart": "loopStart", "loopend": "loopEnd",
	"pitchspread": "pitchSpread", "warpmode": "warpMode", "maxleap": "maxLeap", "inlo": "inLo", "inhi": "inHi",
}

// oldNames are the functions by their new name, to say what they were
func (e *evaluator) renamedNote(old, now string) {
	note := fmt.Sprintf("  note: %s is now called %s", old, now)
	for _, l := range e.patch.Info {
		if l == note {
			return
		}
	}
	e.patch.Info = append(e.patch.Info, note)
}
