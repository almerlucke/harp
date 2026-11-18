package lang

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/almerlucke/psalm/synths/dx"
)

func TestTheOrganIsOneInstrumentWithKeys(t *testing.T) {
	one := renderPatch(t, "bpm = 120\nout = organ(melody([60], every(64), velocity: 0.8, length: 40), preset: \"jazz\")\n", 2)[0]
	chord := renderPatch(t, "bpm = 120\nout = organ(chords([[60 64 67 72 76 79]], every(64), velocity: 0.8, length: 40), preset: \"jazz\")\n", 2)[0]
	if energy(one[8000:60000]) < 1e-6 {
		t.Fatal("the organ is silent")
	}
	// six keys together are all there (a voice a key would have stolen them from each other with one voice): louder than one
	if energy(chord[8000:60000]) < 2*energy(one[8000:60000]) {
		t.Errorf("six keys are not louder than one: %v against %v", energy(chord[8000:60000]), energy(one[8000:60000]))
	}
	// a key that is let go stops: after the note (40 steps = 5 s at 120 bpm... the note is 2.5 s) it is quiet
	short := renderPatch(t, "bpm = 240\nout = organ(melody([60], every(64), velocity: 0.8, length: 4), preset: \"jazz\", leslie: 0)\n", 3)[0]
	if energy(short[int(2.5*44100):]) > 0.05*energy(short[4000:30000]) {
		t.Errorf("the key is let go and the organ goes on: %v against %v", energy(short[int(2.5*44100):]), energy(short[4000:30000]))
	}
	// the drawbars: the 16' and 8' alone are lower than the 2' and 1' alone
	low := renderPatch(t, "bpm = 120\nout = organ(melody([60], every(64), length: 40), drawbars: \"800000000\", leslie: 0)\n", 1.5)[0]
	high := renderPatch(t, "bpm = 120\nout = organ(melody([60], every(64), length: 40), drawbars: \"000008888\", leslie: 0)\n", 1.5)[0]
	crossings := func(x []float64) int {
		n := 0
		for i := 1; i < len(x); i++ {
			if x[i-1] < 0 && x[i] >= 0 {
				n++
			}
		}
		return n
	}
	if crossings(high[10000:40000]) < 3*crossings(low[10000:40000]) {
		t.Errorf("drawbars of high footages are not higher: %d crossings against %d", crossings(high[10000:40000]), crossings(low[10000:40000]))
	}
	for src, want := range map[string]string{
		`organ(notes, drawbars: "88800")`:     "nine digits",
		`organ(notes, drawbars: "88800000x")`: "digits from 0 to 8",
		`organ(notes, leslie: 2)`:             "leslie is from 0 to 1",
		`organ(notes, preset: "jaz")`:         "did you mean jazz",
	} {
		_, err := Compile("notes = melody([60], every(4))\nout = " + src + "\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}

func TestTheFMSynthPlaysPresetsAndDX7Banks(t *testing.T) {
	for _, preset := range []string{"e_piano", "bell", "bass"} {
		x := renderPatch(t, "bpm = 120\nout = dx(melody([57 60 64], every(8), velocity: 0.8, length: 6), preset: \""+preset+"\", voices: 3)\n", 2)[0]
		if energy(x[2000:60000]) < 1e-7 {
			t.Errorf("%s is silent", preset)
		}
	}
	// a bank of the DX7: two patches, the second is the one that is asked for
	dir := t.TempDir()
	bell, bass := dx.Presets["bellPad"](), dx.Presets["woodBass"]()
	bell.Name, bass.Name = "BELLPAD", "WOODBASS"
	if err := os.WriteFile(filepath.Join(dir, "bank.syx"), dx.EncodeBank([]dx.Patch{bell, bass}), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "bpm = 120\nnotes = melody([48 55], every(8), velocity: 0.8, length: 6)\nout = dx(notes, file: \"bank.syx\", patch: 1)\n"
	prog, _ := Parse(src)
	p, err := buildPatch(prog, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if info := strings.Join(p.Info, "\n"); !strings.Contains(info, "playing 1") {
		t.Errorf("harp check says: %q", info)
	}
	_, ch := p.Settings().Render(p.Out, 2)
	if energy(ch[0][2000:60000]) < 1e-7 {
		t.Error("the patch of the bank is silent")
	}
	for s, want := range map[string]string{
		`dx(notes, file: "bank.syx", patch: 40)`: "from 0 to",
		`dx(notes, file: "nothing.syx")`:         "there is no file",
		`dx(notes, preset: "e_pian")`:            "did you mean e_piano",
	} {
		_, err := CompileDir("notes = melody([60], every(4))\nout = "+s+"\n", dir)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", s, err, want)
		}
	}
}

func TestTheDrumSynthPlaysDrumsByKey(t *testing.T) {
	for name, want := range map[string]float64{"kick": 35, "snare": 38, "clap": 39, "hat": 42, "open": 46} {
		if got := evalLast(t, "x = drum(\""+name+"\")").Float(); got != want {
			t.Errorf("drum(%q) is %v, wanted %v", name, got, want)
		}
	}
	// a kick is low, a hat is high: the zero crossings of the first 80 ms
	zero := func(x []float64) int {
		n := 0
		for i := 1; i < len(x); i++ {
			if x[i-1] < 0 && x[i] >= 0 {
				n++
			}
		}
		return n
	}
	hit := func(name string) []float64 {
		return renderPatch(t, "bpm = 120\nout = drumSynth(melody([drum(\""+name+"\")], every(64), velocity: 1, length: 1), kit: \"808\", voices: 4)\n", 1)[0]
	}
	kick, hat := hit("kick"), hit("hat")
	if energy(kick[100:4000]) < 1e-6 || energy(hat[100:4000]) < 1e-8 {
		t.Fatalf("a drum is silent: kick %v, hat %v", energy(kick[100:4000]), energy(hat[100:4000]))
	}
	if zero(hat[200:3700]) < 4*zero(kick[200:3700]) {
		t.Errorf("a hat is not higher than a kick: %d and %d crossings", zero(hat[200:3700]), zero(kick[200:3700]))
	}
	// a closed hat stops an open one: an open hat on step 0 and a closed one on step 1 (a step is 0.125 s at 120 bpm and 4 steps a beat)
	open := renderPatch(t, "bpm = 120\nout = drumSynth(melody([drum(\"open\")], every(64), velocity: 1, length: 1), voices: 4)\n", 1.5)[0]
	stopped := renderPatch(t, "bpm = 120\nout = drumSynth(melody([drum(\"open\") drum(\"hat\")], pattern(\"xx..............\"), velocity: 1, length: 1), voices: 4)\n", 1.5)[0]
	tail := func(x []float64) float64 { return energy(x[int(0.22*44100):int(0.4*44100)]) } // after the closed hat has died away, where the open one still sounds
	if tail(open) < 1e-7 {
		t.Fatalf("the open hat does not sound in the window: %v", tail(open))
	}
	if tail(stopped) > 0.05*tail(open) {
		t.Errorf("the closed hat did not stop the open one: %v against %v", tail(stopped), tail(open))
	}
	for src, want := range map[string]string{
		`drumSynth(notes, kit: "909x")`:  "no kit",
		`drumSynth(notes, kit: "tr808")`: "no kit",
		`drum("kik")`:                    "did you mean kick",
	} {
		_, err := Compile("notes = melody([36], every(4))\nx = " + src + "\nout = sine(1).dup()\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
	_ = math.Pi
}
