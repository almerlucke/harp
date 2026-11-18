package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"harp/dsl"
	"harp/lang"
	"harp/lang/sc"
)

func needSC(t *testing.T) {
	t.Helper()
	if _, err := sc.Find(""); err != nil {
		t.Skip("SuperCollider is not here: " + err.Error())
	}
}

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := compareCmd(args, &out)
	return out.String(), err
}

func TestCompareTheExamples(t *testing.T) {
	needSC(t)
	// what each of the patches of examples/supercollider is, compared with SuperCollider: the words of the report
	for _, c := range []struct {
		file    string
		args    []string
		must    []string
		mustNot []string
	}{
		{"1_vibrato", []string{"-seconds", "4"}, []string{"level: same", "spectrum: same", "The same sound"}, nil},
		{"2_rlpf_saws", []string{"-seconds", "5"}, []string{"level: same", "spectrum: same", "upside down", "The same sound"}, []string{"Different"}},
		{"3_analog_bubbles", []string{"-seconds", "6"}, []string{"level: same", "not in the shape of the waves"}, []string{"Different"}},
		{"4_decay2_pings", []string{"-seconds", "6"}, []string{"level: same", "spectrum: same", "left only"}, []string{"Different"}},
		{"5_resonant_noise", []string{"-seconds", "10", "-stats"}, []string{"not compared", "level:"}, []string{"a random source: two random"}}, // random noise from both: the spectrum verdict is chance (a few dB in a band), so it is not asserted
		{"6_pan_pulse", []string{"-seconds", "10"}, []string{"level: same", "position between left and right"}, []string{"Different"}},
		{"7_dust_pings", []string{"-seconds", "10", "-stats"}, []string{"level:", "spectrum:"}, nil}, // random dust (and a random source in SuperCollider): how loud it is and its spectrum over 10 s are chance, so only that the report is made is checked
	} {
		out, err := run(t, append([]string{"../../examples/supercollider/" + c.file + ".harp"}, c.args...)...)
		if err != nil {
			t.Errorf("%s: %v", c.file, err)
			continue
		}
		for _, m := range c.must {
			if !strings.Contains(out, m) {
				t.Errorf("%s: the report does not say %q:\n%s", c.file, m, out)
			}
		}
		for _, m := range c.mustNot {
			if strings.Contains(out, m) {
				t.Errorf("%s: the report says %q", c.file, m)
			}
		}
	}
}

// two different sounds are said to be different, with the SuperCollider code from the command line or from a file
func TestCompareAWrongTranslation(t *testing.T) {
	needSC(t)
	patch := "../../examples/supercollider/1_vibrato.harp"
	for name, arg := range map[string]string{
		"code": "{ SinOsc.ar(SinOsc.kr(5, 0, 10, 880), 0, 0.2) ! 2 }", // an octave up
	} {
		out, err := run(t, patch, "-seconds", "3", "-sc", arg)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "spectrum: different") || strings.Contains(out, "The same sound") {
			t.Errorf("%s: an octave up is not said to be different:\n%s", name, out)
		}
	}
	// the code in a file
	file := filepath.Join(t.TempDir(), "other.scd")
	os.WriteFile(file, []byte("{ SinOsc.ar(SinOsc.kr(5, 0, 10, 440), 0, 0.1) ! 2 }"), 0o644) // half the level
	out, err := run(t, patch, "-seconds", "3", "-sc", file)
	if err != nil || !strings.Contains(out, "level: different") || !strings.Contains(out, "+6.0") {
		t.Errorf("from a file, half as loud: %v\n%s", err, out)
	}
}

func TestCompareSaves(t *testing.T) {
	needSC(t)
	dir := t.TempDir()
	out, err := run(t, "../../examples/supercollider/6_pan_pulse.harp", "-seconds", "2", "-save", dir)
	if err != nil || !strings.Contains(out, "6_pan_pulse_SC.wav") {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, name := range []string{"6_pan_pulse_SC.wav", "6_pan_pulse_harp.wav"} {
		rate, ch, err := dsl.ReadWAV(filepath.Join(dir, name))
		if err != nil || rate != 44100 || len(ch) != 2 || len(ch[0]) != 88200 {
			t.Errorf("%s: rate %d, %d channels, err %v", name, rate, len(ch), err)
		}
	}
}

func TestCompareErrors(t *testing.T) {
	cases := map[string][]string{
		"no patch":              {},
		"flags first":           {"-seconds", "3"},
		"no such file":          {"/no/such/patch.harp"},
		"seconds":               {"../../examples/supercollider/1_vibrato.harp", "-seconds", "0"},
		"no SuperCollider code": {"../../examples/play.harp"},
	}
	want := map[string]string{"no patch": "usage", "flags first": "usage", "no such file": "no such file", "seconds": "above 0", "no SuperCollider code": "-sc"}
	for name, args := range cases {
		_, err := run(t, args...)
		if err == nil || !strings.Contains(err.Error(), want[name]) {
			t.Errorf("%s: %v", name, err)
		}
	}
	// a mistake in the harp patch is shown with its line, as everywhere
	bad := filepath.Join(t.TempDir(), "bad.harp")
	os.WriteFile(bad, []byte("// SuperCollider\n//   { SinOsc.ar(440) }\nout = lowpas(sine(440), 800)\n"), 0o644)
	if _, err := run(t, bad); err == nil || !strings.Contains(err.Error(), "did you mean lowPass") || !strings.Contains(err.Error(), "3 | out = lowpas") {
		t.Errorf("a mistake in the patch: %v", err)
	}
}

// a mistake in the SuperCollider code is said by SuperCollider, in the lines of the code, and soon
func TestCompareSuperColliderMistake(t *testing.T) {
	needSC(t)
	start := time.Now()
	_, err := run(t, "../../examples/supercollider/1_vibrato.harp", "-seconds", "2", "-sc", "{ SinOsc.ar(440).noSuchMethod }")
	if err == nil || !strings.Contains(err.Error(), "noSuchMethod") {
		t.Errorf("%v", err)
	}
	if took := time.Since(start); took > 10*time.Second {
		t.Errorf("it took %v", took)
	}
}

func TestCompareSeeds(t *testing.T) {
	needSC(t)
	out, err := run(t, "../../examples/supercollider/7_dust_pings.harp", "-seeds", "6", "-seconds", "3")
	if err != nil || !strings.Contains(out, "6 seeds") || !strings.Contains(out, "level (dB)") || !strings.Contains(out, "sort of patch") {
		t.Fatalf("%v\n%s", err, out)
	}
	// a patch of one sort, with a level that is on the one side: not the same sort
	patch := filepath.Join(t.TempDir(), "p.harp")
	os.WriteFile(patch, []byte("// SuperCollider\n//   { SinOsc.ar(exprand(100, 1000), 0, 0.05) ! 2 }\nout = sine(expRand(100, 1000)).gain(0.2).dup()\n"), 0o644)
	out, err = run(t, patch, "-seeds", "10", "-seconds", "2")
	if err != nil || !strings.Contains(out, "Not the same sort") || !strings.Contains(out, "different") {
		t.Errorf("%v\n%s", err, out)
	}
	if _, err := run(t, patch, "-seeds", "1"); err == nil || !strings.Contains(err.Error(), "2 or more") {
		t.Errorf("one seed: %v", err)
	}
}

func TestWithSeed(t *testing.T) {
	if got := withSeed("a = 1\nseed = 8\nout = a", 3); got != "a = 1\nseed = 3\nout = a" {
		t.Errorf("%q", got)
	}
	if got := withSeed("out = 1", 3); got != "seed = 3\nout = 1" {
		t.Errorf("%q", got)
	}
}

// a WAV file instead of SuperCollider code: the file of harp itself is the same sound
func TestCompareWithAReferenceFile(t *testing.T) {
	dir := t.TempDir()
	patch := filepath.Join(dir, "p.harp")
	os.WriteFile(patch, []byte("out = sine(440).gain(0.2).dup()\n"), 0o644)
	p, _ := lang.Compile("out = sine(440).gain(0.2).dup()\n")
	_, ch := p.Settings().Render(p.Out, 2)
	ref := filepath.Join(dir, "ref.wav")
	if err := dsl.WriteWAVFloat(ref, 44100, ch...); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, patch, "-ref", ref, "-seconds", "2")
	if err != nil || !strings.Contains(out, "The same sound") || !strings.Contains(out, "waves: the same") {
		t.Errorf("%v\n%s", err, out)
	}
	// a file of another sound
	p2, _ := lang.Compile("out = sine(880).gain(0.2).dup()\n")
	_, ch2 := p2.Settings().Render(p2.Out, 2)
	other := filepath.Join(dir, "other.wav")
	dsl.WriteWAVFloat(other, 44100, ch2...)
	out, err = run(t, patch, "-ref", other, "-seconds", "2")
	if err != nil || !strings.Contains(out, "spectrum: different") {
		t.Errorf("%v\n%s", err, out)
	}
	if _, err := run(t, patch, "-ref", ref, "-seeds", "3"); err == nil || !strings.Contains(err.Error(), "-ref") {
		t.Errorf("seeds with a reference: %v", err)
	}
	if _, err := run(t, patch, "-ref", filepath.Join(dir, "nothing.wav")); err == nil {
		t.Error("a file that is not there")
	}
	// a file of another rate
	dsl.WriteWAVFloat(filepath.Join(dir, "r.wav"), 48000, ch...)
	if _, err := run(t, patch, "-ref", filepath.Join(dir, "r.wav")); err == nil || !strings.Contains(err.Error(), "48000") {
		t.Errorf("another rate: %v", err)
	}
}

// the kick of the gabber is the same wave as SuperCollider's, for 20 seconds of a pattern of 40 notes (the reference is rendered by sclang: a
// pattern is not code that compare can run)
func TestCompareTheGabberKick(t *testing.T) {
	needSC(t)
	server, _ := sc.Find("")
	dir := t.TempDir()
	cmd := exec.Command(server.Sclang, "-D", "../../examples/supercollider/gabber_compare.scd", dir)
	if b, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(b), "ALL DONE") {
		t.Fatalf("sclang: %v\n%s", err, b)
	}
	out, err := run(t, "../../examples/supercollider/9_gabber_kick.harp", "-ref", filepath.Join(dir, "kick.wav"), "-seconds", "20")
	if err != nil || !strings.Contains(out, "waves: the same") || !strings.Contains(out, "level: same") {
		t.Errorf("%v\n%s", err, out)
	}
	out, err = run(t, "../../examples/supercollider/11_gabber_rave.harp", "-ref", filepath.Join(dir, "full.wav"), "-seconds", "20", "-stats")
	if err != nil || !strings.Contains(out, "level: same") || !strings.Contains(out, "spectrum: same") {
		t.Errorf("%v\n%s", err, out)
	}
}
