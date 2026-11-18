package lang

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func infoOf(t *testing.T, src string) string {
	t.Helper()
	prog, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	p, err := buildPatch(prog, nil, "")
	if err != nil {
		t.Fatalf("%v\n%s", err, src)
	}
	return strings.Join(p.Info, "\n")
}

func TestOldNamesStillWorkAndSayWhatTheyAreNow(t *testing.T) {
	// a function in one word: the same sound, and a note
	old := "out = sine(440).lowpass(cutoff: 800).gain(0.2).dup()\n"
	now := "out = sine(440).lowPass(cutoff: 800).gain(0.2).dup()\n"
	a, b := renderPatch(t, old, 0.05), renderPatch(t, now, 0.05)
	for i := range a[0] {
		if a[0][i] != b[0][i] {
			t.Fatalf("the old name is another sound at sample %d", i)
		}
	}
	if info := infoOf(t, old); !strings.Contains(info, "lowpass is now called lowPass") {
		t.Errorf("harp check says: %q", info)
	}
	if info := infoOf(t, now); strings.Contains(info, "now called") {
		t.Errorf("the new name has a note: %q", info)
	}
	// the effects (their names are psalm's), the names of the SuperCollider ones, the generators
	for _, c := range []struct{ old, now string }{
		{"pingpong", "pingPong"}, {"ringmod", "ringMod"}, {"scsaw", "scSaw"}, {"dbamp", "dbAmp"}, {"lfnoise", "lfNoise"},
	} {
		var src string
		switch c.old {
		case "pingpong", "ringmod":
			src = "out = " + c.old + "(sine(440))\n"
		case "scsaw":
			src = "out = scsaw(440).dup()\n"
		case "dbamp":
			src = "out = sine(440).gain(dbamp(-12)).dup()\n"
		default:
			src = "out = lfnoise(4).dup()\n"
		}
		if info := infoOf(t, src); !strings.Contains(info, c.old+" is now called "+c.now) {
			t.Errorf("%s: harp check says %q", c.old, info)
		}
	}
	// a parameter in one word
	if info := infoOf(t, "form = song(\"A 1\", barsteps: 4)\nout = sine(1).dup()\n"); !strings.Contains(info, "barsteps is now called barSteps") {
		t.Errorf("harp check says: %q", info)
	}
	if info := infoOf(t, "form = song(\"A 1\", barSteps: 4)\nout = sine(1).dup()\n"); strings.Contains(info, "now called") {
		t.Errorf("the new spelling has a note: %q", info)
	}
	// a name that is neither is still said to be wrong, with what is near it (the new spelling)
	if _, err := Compile("out = lowpas(sine(440), 800)\n"); err == nil || !strings.Contains(err.Error(), "did you mean lowPass") {
		t.Errorf("%v", err)
	}
	if _, err := Compile("out = song(\"A 1\", barstep: 4)\n"); err == nil || !strings.Contains(err.Error(), "did you mean barSteps") {
		t.Errorf("%v", err)
	}
	// the preset list of a synth by its old name
	if s, err := Presets("drumsynth"); err != nil || !strings.Contains(s, "909") {
		t.Errorf("%v %q", err, s)
	}
}

// the docs and the examples are written with the names as they are now: no call of a function or an argument that has its old spelling
// (the SuperCollider code in comments and in docs/supercollider.md is SuperCollider's)
func TestTheDocsAndTheExamplesHaveTheNewNames(t *testing.T) {
	var files []string
	for _, pat := range []string{"../docs/*.md", "examples/*.harp", "../README.md"} {
		m, _ := filepath.Glob(pat)
		files = append(files, m...)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "supercollider.md") {
			continue
		}
		b, _ := os.ReadFile(f)
		for n, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") || strings.Contains(line, "were written in one word before") {
				continue // comments: prose, and SuperCollider code; and the sentence that says what the old names were
			}
			for old, now := range renamedFuncs {
				if regexp.MustCompile(`\b` + old + `\(`).MatchString(line) {
					t.Errorf("%s:%d: %s( is %s(: %s", f, n+1, old, now, strings.TrimSpace(line))
				}
			}
			for old, now := range renamedParams {
				if regexp.MustCompile(`\b` + old + `\b`).MatchString(line) {
					t.Errorf("%s:%d: %s is %s: %s", f, n+1, old, now, strings.TrimSpace(line))
				}
			}
		}
	}
}
