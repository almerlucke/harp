package sc

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

func server(t *testing.T) *Server {
	t.Helper()
	s, err := Find("")
	if err != nil {
		t.Skip("SuperCollider is not here: " + err.Error())
	}
	return s
}

func peak(x []float64) (p float64) {
	for _, v := range x {
		p = math.Max(p, math.Abs(v))
	}
	return
}

func TestFromComments(t *testing.T) {
	for file, want := range map[string]string{
		"1_vibrato.harp":        "{ SinOsc.ar(SinOsc.kr(5, 0, 10, 440), 0, 0.2) ! 2 }",
		"2_rlpf_saws.harp":      "{ RLPF.ar(Saw.ar([100, 101], 0.1), SinOsc.kr(0.2).range(300, 3000), 0.2) }",
		"6_pan_pulse.harp":      "{ Pan2.ar(Pulse.ar(100, 0.3, 0.1), LFTri.kr(0.2)) }",
		"7_dust_pings.harp":     "{ (SinOsc.ar(1200) * Decay2.ar(Dust.ar(4), 0.005, 0.15) * 0.3) ! 2 }",
		"5_resonant_noise.harp": "{ Resonz.ar(WhiteNoise.ar(0.5), LFNoise1.kr(1).exprange(200, 5000), 0.05) }",
	} {
		src, err := os.ReadFile("../examples/supercollider/" + file)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := FromComments(string(src))
		if !ok || got != want {
			t.Errorf("%s: %q %v, want %q", file, got, ok, want)
		}
	}
	// the code that goes over several lines keeps its lines
	src, _ := os.ReadFile("../examples/supercollider/3_analog_bubbles.harp")
	got, ok := FromComments(string(src))
	if !ok || !strings.HasPrefix(got, "{\n") || !strings.Contains(got, "var f = LFSaw.kr(0.4, 0, 24, LFSaw.kr([8, 7.23], 0, 3, 80)).midicps;") || !strings.HasSuffix(got, "}") || strings.Contains(got, ".play") {
		t.Errorf("bubbles: %q %v", got, ok)
	}
	// every patch of the examples says which SuperCollider code it is
	files, _ := os.ReadDir("../examples/supercollider")
	for _, f := range files {
		// the gabber is a pattern of two SynthDefs: its reference is the file that gabber_compare.scd makes (harp compare -ref)
		if !strings.HasSuffix(f.Name(), ".harp") || strings.HasPrefix(f.Name(), "11_") {
			continue
		}
		src, _ := os.ReadFile("../examples/supercollider/" + f.Name())
		if _, ok := FromComments(string(src)); !ok {
			t.Errorf("%s has no SuperCollider code in its comment", f.Name())
		}
	}
	// and those that do not say it are not guessed at: indented comment lines that are not after the word
	for _, src := range []string{
		"out = 1",
		"// the patch\n//   harp play it\nout = 1",
		"// SuperCollider:\n// no indented lines here\nout = 1",
		"// SuperCollider:\n//\n//   not a function\nout = 1",
		"out = 1\n// SuperCollider:\n//   { SinOsc.ar }.play",
	} {
		if got, ok := FromComments(src); ok {
			t.Errorf("%q gave %q", src, got)
		}
	}
	// the .play and a ; are taken off, and the first block only
	got, _ = FromComments("// SuperCollider\n//   { Saw.ar(100) }.play;\n//\n//   { Pulse.ar(100) }.play\nout = 1")
	if got != "{ Saw.ar(100) }" {
		t.Errorf("got %q", got)
	}
}

func TestRender(t *testing.T) {
	s := server(t)
	ch, _, err := s.Render("{ SinOsc.ar(440, 0, 0.2) ! 2 }", 1, 44100)
	if err != nil {
		t.Fatal(err)
	}
	if len(ch) != 2 || len(ch[0]) != 44100 || math.Abs(peak(ch[0])-0.2) > 1e-6 {
		t.Fatalf("%d channels of %d samples, peak %v", len(ch), len(ch[0]), peak(ch[0]))
	}
	for i := range ch[0] {
		if ch[0][i] != ch[1][i] {
			t.Fatal("! 2 gives two channels that are not the same")
		}
	}
	// a function of one channel is heard on the left only, as with .play
	ch, _, err = s.Render("{ SinOsc.ar(440, 0, 0.2) }", 0.5, 48000)
	if err != nil || len(ch[0]) != 24000 || peak(ch[0]) < 0.19 || peak(ch[1]) != 0 {
		t.Errorf("mono: err %v, %d samples, peaks %v %v", err, len(ch[0]), peak(ch[0]), peak(ch[1]))
	}
	// code that is not a function in braces is one: the braces are put around it
	ch, _, err = s.Render("DC.ar(0.25) ! 2", 0.2, 44100)
	if err != nil || math.Abs(ch[0][5000]-0.25) > 1e-6 {
		t.Errorf("without braces: %v", err)
	}
}

// a mistake in the SuperCollider code is said, with the words of SuperCollider, and soon: it does not wait for a timeout
func TestRenderErrors(t *testing.T) {
	s := server(t)
	for name, code := range map[string]string{
		"syntax":        "{ SinOsc.ar( }",
		"not a class":   "{ NoSuchUGen.ar(440) }",
		"wrong message": "{ SinOsc.ar(440).noSuchMethod }",
	} {
		start := time.Now()
		_, text, err := s.Render(code, 1, 44100)
		if err == nil || !strings.Contains(err.Error(), "ERROR") {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if took := time.Since(start); took > 8*time.Second {
			t.Errorf("%s: it took %v to find out", name, took)
		}
		_ = text
	}
	// SuperCollider says where in the code, and it is in the lines of the code, not of the script that was made of it
	_, _, err := s.Render("{\n  SinOsc.ar(440, 0, 0.2) +\n  SinOsc.ar( \n}", 1, 44100)
	if err == nil || !strings.Contains(err.Error(), "of your code") {
		t.Errorf("the place of the mistake: %v", err)
	}
}

func TestFindsNothingWhereThereIsNothing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("HARP_SC", "")
	t.Setenv("SC_HOME", "")
	if _, err := Find(t.TempDir()); err == nil {
		// /usr/bin and /usr/local/bin may have it, which is fine
		t.Log("found in a system folder")
	} else if !strings.Contains(err.Error(), "-sclang") {
		t.Errorf("the message does not say what to do: %v", err)
	}
}

// a seed makes the random choices of the code (those of sclang, when the patch is built) repeat
func TestRenderSeed(t *testing.T) {
	s, err := Find("")
	if err != nil {
		t.Skip(err)
	}
	const code = "{ SinOsc.ar(exprand(100, 1000), 0, 0.1) ! 2 }"
	a, _, err := s.RenderSeed(code, 0.5, 44100, 3)
	if err != nil {
		t.Fatal(err)
	}
	b, _, _ := s.RenderSeed(code, 0.5, 44100, 3)
	c, _, _ := s.RenderSeed(code, 0.5, 44100, 4)
	same := func(x, y [][]float64) bool {
		for i := range x[0] {
			if x[0][i] != y[0][i] {
				return false
			}
		}
		return true
	}
	if !same(a, b) {
		t.Error("the same seed is not the same patch")
	}
	if same(a, c) {
		t.Error("another seed is the same patch")
	}
	// the noises are drawn in the server: they repeat with the seed as well
	const noisy = "{ (LFNoise1.ar(300) + WhiteNoise.ar(0.1)) ! 2 }"
	n1, _, _ := s.RenderSeed(noisy, 0.5, 44100, 5)
	n2, _, _ := s.RenderSeed(noisy, 0.5, 44100, 5)
	n3, _, _ := s.RenderSeed(noisy, 0.5, 44100, 6)
	if !same(n1, n2) || same(n1, n3) {
		t.Errorf("the noises of the server with a seed: repeat %v, other seed the same %v", same(n1, n2), same(n1, n3))
	}
	if _, _, err := s.RenderSeed(code, 0.5, 44100, -1); err == nil {
		t.Error("a negative seed")
	}
}
