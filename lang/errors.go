package lang

import (
	"fmt"
	"strings"
)

// Error is a mistake in a program, at a place in it
type Error struct {
	Pos  Pos
	Msg  string
	Hint string // what could be meant, if there is a guess
}

func (e *Error) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Pos, e.Msg, e.Hint)
	}
	return fmt.Sprintf("%s: %s", e.Pos, e.Msg)
}

// Show prints the error with the line of the source it is in, and a mark under the place
func (e *Error) Show(src string) string {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	if e.Pos.Line < 1 || e.Pos.Line > len(lines) {
		return e.Error()
	}
	line := lines[e.Pos.Line-1]
	prefix := fmt.Sprintf("%d | ", e.Pos.Line)
	mark := strings.Repeat(" ", len(prefix)+max(e.Pos.Col-1, 0)) + "^"
	return e.Error() + "\n" + prefix + line + "\n" + mark
}
