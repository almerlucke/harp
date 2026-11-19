package lang

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// A patch can be written in several files: use "drums.harp" puts the statements of that file at that place, as if they were written there. The
// names and the functions of the file are the patch's from there on, and the file can use others (their names are looked for from the folder of
// the file that uses them). A file that is used twice, by two files, is read once; a file that uses itself, through others, is a mistake. Only
// the patch itself has out.

// useState is what the statements of the patch and of the files that it uses share while it is built
type useState struct {
	loaded map[string]bool // the files that were read (by their full path)
	stack  []string        // the files that are being read: the one that is being read is the last
	fnAt   map[string]Pos  // where the functions are defined
	out    *Value
	outAt  Pos
}

// where says a place for a message: with several files the patch itself is named too
func (e *evaluator) where(p Pos) string {
	if p.File == "" && e.uses != nil && len(e.uses.loaded) > 1 {
		return "the patch itself, " + p.String()
	}
	return p.String()
}

func (e *evaluator) fnWhere(name string) Pos {
	if e.uses == nil {
		return Pos{}
	}
	return e.uses.fnAt[name]
}

// use reads a file that the patch uses, and does its statements here. dir is the folder of the file that has the statement
func (e *evaluator) use(st Stmt, dir string) error {
	path := st.Use
	if !filepath.IsAbs(path) {
		if dir == "" {
			dir = "."
		}
		path = filepath.Join(dir, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	u := e.uses
	for i, f := range u.stack {
		if f == abs {
			chain := make([]string, 0, len(u.stack)-i+1)
			for _, c := range u.stack[i:] {
				chain = append(chain, shortPath(c))
			}
			chain = append(chain, shortPath(abs))
			return &Error{Pos: st.P, Msg: fmt.Sprintf("%s is used by itself: %s", shortPath(abs), strings.Join(chain, " uses "))}
		}
	}
	if u.loaded[abs] {
		return nil // another file used it before: its names are there already
	}
	src, err := os.ReadFile(abs)
	if err != nil {
		hint := ""
		if _, serr := os.Stat(abs); serr != nil {
			hint = "looked for " + abs
		}
		return &Error{Pos: st.P, Msg: fmt.Sprintf("use %q: the file can't be read", st.Use), Hint: hint}
	}
	prog, err := ParseFile(string(src), abs)
	if err != nil {
		return err
	}
	u.loaded[abs] = true
	u.stack = append(u.stack, abs)
	e.patch.Files = append(e.patch.Files, abs)
	e.patch.Info = append(e.patch.Info, fmt.Sprintf("  uses %s", shortPath(abs)))
	err = e.run(prog, filepath.Dir(abs), true)
	u.stack = u.stack[:len(u.stack)-1]
	return err
}
