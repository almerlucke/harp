package lang

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// the reference of the functions is made from the code: it is checked here, so that it does not go out of date
func TestTheReferenceIsUpToDate(t *testing.T) {
	got, err := os.ReadFile("../docs/reference.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != FunctionsMarkdown() {
		t.Error("docs/reference.md is not what the functions say: run  go run ./cmd/harp functions -md > ../docs/reference.md  (in lang)")
	}
}

var link = regexp.MustCompile(`\]\(([^)#]+\.md)(#[^)]*)?\)`)

// every link between the pages goes to a page that is there, and every page is reachable from the index
func TestTheDocsLinkToPagesThatExist(t *testing.T) {
	pages, _ := filepath.Glob("../docs/*.md")
	if len(pages) < 8 {
		t.Fatalf("only %d pages", len(pages))
	}
	linked := map[string]bool{"README.md": true}
	for _, p := range pages {
		b, _ := os.ReadFile(p)
		for _, m := range link.FindAllStringSubmatch(string(b), -1) {
			target := filepath.Join(filepath.Dir(p), m[1])
			if _, err := os.Stat(target); err != nil {
				t.Errorf("%s links to %s, which is not there", filepath.Base(p), m[1])
			}
			if filepath.Base(p) == "README.md" {
				linked[filepath.Base(m[1])] = true
			}
		}
	}
	for _, p := range pages {
		if !linked[filepath.Base(p)] {
			t.Errorf("%s is not linked from the index (docs/README.md)", filepath.Base(p))
		}
	}
}

var harpBlock = regexp.MustCompile("(?s)```harp\n(.*?)```")

// the patches in the docs (the blocks marked harp) are patches: they compile
func TestThePatchesInTheDocsCompile(t *testing.T) {
	pages, _ := filepath.Glob("../docs/*.md")
	n := 0
	for _, p := range pages {
		b, _ := os.ReadFile(p)
		for _, m := range harpBlock.FindAllStringSubmatch(string(b), -1) {
			n++
			if _, err := Compile(m[1]); err != nil {
				t.Errorf("%s: a patch in it does not compile: %v\n%s", filepath.Base(p), err, strings.TrimSpace(m[1]))
			}
		}
	}
	if n < 3 {
		t.Errorf("only %d patches are marked in the docs", n)
	}
}

// the list of the presets is made from the code too
func TestThePresetsPageIsUpToDate(t *testing.T) {
	got, err := os.ReadFile("../docs/presets.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != PresetsMarkdown() {
		t.Error("docs/presets.md is not what the synths say: run  go run ./cmd/harp presets -md > ../docs/presets.md  (in lang)")
	}
}
