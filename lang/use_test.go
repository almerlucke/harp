package lang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// files makes a folder with the files and gives the path of the first
func files(t *testing.T, fs map[string]string, main string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range fs {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, main)
}

func compileFile(t *testing.T, path string) (*Patch, error) {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return CompileFile(string(src), path)
}

func TestAPatchInSeveralFiles(t *testing.T) {
	main := files(t, map[string]string{
		"main.harp":         "use \"parts/chords.harp\"\nuse \"parts/drums.harp\"\nout = tone(220) * fade(level)\n",
		"parts/chords.harp": "bpm = 90\nlevel = 0.25\nfn fade(x) { x * 0.5 }\nfn tone(f) { sine(f) }\n",
		// a file looks for the files it uses from its own folder; chords.harp is used by both of them, and read once
		"parts/drums.harp": "use \"chords.harp\"\nhit = every(4)\n",
	}, "main.harp")
	p, err := compileFile(t, main)
	if err != nil {
		t.Fatal(err)
	}
	if p.BPM != 90 {
		t.Errorf("bpm of the file: %v", p.BPM)
	}
	if len(p.Files) != 2 || !strings.HasSuffix(p.Files[0], filepath.Join("parts", "chords.harp")) || !strings.HasSuffix(p.Files[1], filepath.Join("parts", "drums.harp")) {
		t.Errorf("files: %v", p.Files)
	}
	// the same sound as the patch written as one file
	one := "bpm = 90\nlevel = 0.25\nfn fade(x) { x * 0.5 }\nfn tone(f) { sine(f) }\nout = tone(220) * fade(level)\n"
	a, b := renderPatch(t, one, 0.05), func() [][]float64 { _, ch := p.Settings().Render(p.Out, 0.05); return ch }()
	for i := range a[0] {
		if a[0][i] != b[0][i] {
			t.Fatalf("another sound than the one file at sample %d", i)
		}
	}
}

func TestMistakesWithFilesThatAreUsed(t *testing.T) {
	for name, c := range map[string]struct {
		files map[string]string
		want  []string
	}{
		"out in a file":       {map[string]string{"main.harp": "use \"a.harp\"\nout = sine(1)\n", "a.harp": "out = sine(2)\n"}, []string{"a.harp:1:1", "out is only in the patch itself"}},
		"a name twice":        {map[string]string{"main.harp": "use \"a.harp\"\nx = 2\nout = sine(1)\n", "a.harp": "x = 1\n"}, []string{"x is already defined", "a.harp:1:1"}},
		"a name twice, other": {map[string]string{"main.harp": "x = 2\nuse \"a.harp\"\nout = sine(1)\n", "a.harp": "x = 1\n"}, []string{"a.harp:1:1: x is already defined, at the patch itself, 1:1"}},
		"a function twice":    {map[string]string{"main.harp": "use \"a.harp\"\nfn f(x) { x }\nout = sine(1)\n", "a.harp": "fn f(x) { x }\n"}, []string{"the function f is already defined, at", "a.harp:1:1"}},
		"it uses itself":      {map[string]string{"main.harp": "use \"a.harp\"\nout = sine(1)\n", "a.harp": "use \"b.harp\"\n", "b.harp": "use \"a.harp\"\n"}, []string{"is used by itself", "a.harp uses", "b.harp uses"}},
		"no such file":        {map[string]string{"main.harp": "use \"nothere.harp\"\nout = sine(1)\n"}, []string{"1:1", "nothere.harp", "can't be read"}},
		"a mistake in a file": {map[string]string{"main.harp": "use \"a.harp\"\nout = sine(1)\n", "a.harp": "\n\nx = lowpas(sine(2), 800)\n"}, []string{"a.harp:3:5", "did you mean lowPass"}},
		"syntax in a file":    {map[string]string{"main.harp": "use \"a.harp\"\nout = sine(1)\n", "a.harp": "x = = 1\n"}, []string{"a.harp:1:5", "syntax error"}},
		"no name":             {map[string]string{"main.harp": "use \"\"\nout = sine(1)\n"}, []string{"name of a file"}},
	} {
		_, err := compileFile(t, files(t, c.files, "main.harp"))
		if err == nil {
			t.Errorf("%s: no error", name)
			continue
		}
		for _, w := range c.want {
			if !strings.Contains(err.Error(), w) {
				t.Errorf("%s: %q does not say %q", name, err, w)
			}
		}
	}
	// a mistake in a file is shown with the line of that file
	main := files(t, map[string]string{"main.harp": "use \"a.harp\"\nout = sine(1)\n", "a.harp": "\nx = lowpas(sine(2), 800)\n"}, "main.harp")
	_, err := compileFile(t, main)
	if shown := err.(*Error).Show("use \"a.harp\"\nout = sine(1)\n"); !strings.Contains(shown, "2 | x = lowpas(sine(2), 800)") {
		t.Errorf("shown: %q", shown)
	}
}

// use is a word of the language, like fn and if: it can't be a name
func TestUseIsAWordOfTheLanguage(t *testing.T) {
	for _, src := range []string{"use = 440\nout = sine(1)\n", "x = use\nout = sine(1)\n", "x = sine(use)\nout = x\n", "fn use(a) { a }\nout = sine(1)\n"} {
		if _, err := Compile(src); err == nil || !strings.Contains(err.Error(), "use is a word of the language") {
			t.Errorf("%q: %v", src, err)
		}
	}
}

// a patch that is a file is built again, when a file that it uses has changed, with what is in it now
func TestAFileThatIsUsedIsReadAgainOnARecompile(t *testing.T) {
	main := files(t, map[string]string{"main.harp": "use \"a.harp\"\nout = sine(f)\n", "a.harp": "f = 220\n"}, "main.harp")
	p, err := compileFile(t, main)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(main), "a.harp"), []byte("f = 440\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src, _ := os.ReadFile(main)
	q, err := Recompile(string(src), p)
	if err != nil {
		t.Fatal(err)
	}
	_, ch := q.Settings().Render(q.Out, 0.01)
	_, want := mustBuild(t, "out = sine(440)").Settings().Render(mustBuild(t, "out = sine(440)").Out, 0.01)
	_ = want
	if len(q.Files) != 1 || len(ch[0]) == 0 {
		t.Errorf("files %v", q.Files)
	}
	one := renderPatch(t, "out = sine(440)", 0.01)[0]
	for i := range one {
		if one[i] != ch[0][i] {
			t.Fatalf("the new value of the file is not used (sample %d)", i)
		}
	}
}

// a file can be in a folder above: use "../shared.harp", from any depth
func TestUseGoesUpAFolder(t *testing.T) {
	main := files(t, map[string]string{
		"shared.harp":         "f = 330\n",
		"sub/mid.harp":        "use \"../shared.harp\"\ng = f * 2\n",
		"sub/deeper/low.harp": "use \"../../shared.harp\"\nuse \"../mid.harp\"\n", // the shared file again, by another way: read once
		"main.harp":           "use \"sub/deeper/low.harp\"\nout = sine(g)\n",
	}, "main.harp")
	p, err := compileFile(t, main)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Files) != 3 {
		t.Errorf("files: %v", p.Files)
	}
	_, got := p.Settings().Render(p.Out, 0.01)
	want := renderPatch(t, "out = sine(660)", 0.01)
	for i := range want[0] {
		if got[0][i] != want[0][i] {
			t.Fatalf("not 660 Hz at sample %d", i)
		}
	}
}
