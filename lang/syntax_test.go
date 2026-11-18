package lang

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func parse(t *testing.T, src string) string {
	t.Helper()
	p, err := Parse(src)
	if err != nil {
		t.Fatalf("%q: %v", src, err)
	}
	return strings.TrimSpace(p.String())
}

func TestParseExpressions(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		// numbers, notes, names, strings
		{"x = 12", "x = 12"},
		{"x = 0.5e2", "x = 50"},
		{"x = -3", "x = -3"},
		{"x = a3", "x = a3"},
		{"x = c#4", "x = c#4"},
		{"x = Bb2", "x = Bb2"},
		{"x = abc", "x = abc"},
		{"x = a3b", "x = a3b"}, // a name that only starts like a note
		{`x = "pad"`, `x = "pad"`},
		// operators: precedence and left to right
		{"x = 1 + 2 * 3", "x = (1 + (2 * 3))"},
		{"x = 1 * 2 + 3", "x = ((1 * 2) + 3)"},
		{"x = 10 - 3 - 2", "x = ((10 - 3) - 2)"},
		{"x = 8 / 4 / 2", "x = ((8 / 4) / 2)"},
		{"x = (1 + 2) * 3", "x = ((1 + 2) * 3)"},
		{"x = -a * 2", "x = ((-a) * 2)"},
		{"x = a - -2", "x = (a - -2)"},
		// calls, named arguments, methods, fields
		{"x = f()", "x = f()"},
		{"x = f(1, 2)", "x = f(1, 2)"},
		{"x = f(1, res: 0.5, mix: a + 1)", "x = f(1, res: 0.5, mix: (a + 1))"},
		{"x = f(1,)", "x = f(1)"},
		{"x = a.f(1)", "x = a.f(1)"},
		{"x = a.f()", "x = a.f()"},
		{"x = a.b", "x = a.b"},
		{"x = a.b.c(1).d", "x = a.b.c(1).d"},
		{"x = f(a).g(b: 2).h", "x = f(a).g(b: 2).h"},
		{"x = voice.pitch * 0.997", "x = (voice.pitch * 0.997)"},
		// lists: spaces between plain items, commas between any expressions
		{"x = []", "x = []"},
		{"x = [1 2 3]", "x = [1 2 3]"},
		{"x = [7 9 11 -2]", "x = [7 9 11 -2]"},
		{"x = [1, 2, 3]", "x = [1 2 3]"},
		{"x = [a * 2, b + 1]", "x = [(a * 2) (b + 1)]"},
		{"x = [a3 c4 e4]", "x = [a3 c4 e4]"},
		{"x = [[1 2] [3 4]]", "x = [[1 2] [3 4]]"},
		{"x = [f(1) g.h 3]", "x = [f(1) g.h 3]"},
		{"x = [1,]", "x = [1]"},
		{"x = [1 2\n  3]", "x = [1 2 3]"},
		// white space, newlines, comments
		{"x = 1 // one\ny = 2", "x = 1\ny = 2"},
		{"// a comment\nx = 1", "x = 1"},
		{"x = 1\n\n\ny = 2\n", "x = 1\ny = 2"},
		{"x = 1 +\n  2", "x = (1 + 2)"},
		{"x = 1\n  + 2", "x = (1 + 2)"},
		{"", ""},
		{"  \n // nothing\n", ""},
		{"x = 8 / 4 // half", "x = (8 / 4)"},
		{"x = 8 //2", "x = 8"},
	} {
		if got := parse(t, c.src); got != c.want {
			t.Errorf("%q\n  got  %s\n  want %s", c.src, got, c.want)
		}
	}
}

// a chain goes on in the next line when that starts with a dot, and a new statement starts at a name and =
func TestParseChainsAndStatements(t *testing.T) {
	src := `
melody = degrees(a3, minor, mutate([7 9 11 9 8 7 4 7], swap: 0.6))
lead = saw([voice.pitch * 0.997, voice.pitch * 1.023])
  .lowpass(cutoff: lfo(0.3).range(300, 4000), res: 0.5)   // a filter
  .gain(adsr(0.005, 0.15, 0.5, 0.2).on(voice.gate))
out = lead * 0.4
`
	want := `melody = degrees(a3, minor, mutate([7 9 11 9 8 7 4 7], swap: 0.6))
lead = saw([(voice.pitch * 0.997) (voice.pitch * 1.023)]).lowpass(cutoff: lfo(0.3).range(300, 4000), res: 0.5).gain(adsr(0.005, 0.15, 0.5, 0.2).on(voice.gate))
out = (lead * 0.4)`
	if got := parse(t, src); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

// a line that starts with a parenthesis after a line that ends with a name is a new thing, not a call
func TestParseParenthesisAtTheStartOfALine(t *testing.T) {
	src := "a = b\nc = (1 + 2) * 3"
	if got, want := parse(t, src), "a = b\nc = ((1 + 2) * 3)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParsePositions(t *testing.T) {
	p, err := Parse("x = 1\ny = f(a,\n  b: 2)\n")
	if err != nil {
		t.Fatal(err)
	}
	if p.Stmts[0].P != (Pos{1, 1}) || p.Stmts[1].P != (Pos{2, 1}) {
		t.Errorf("statements at %v and %v", p.Stmts[0].P, p.Stmts[1].P)
	}
	call := p.Stmts[1].Value.(*Call)
	if call.P != (Pos{2, 5}) || call.Args[0].P != (Pos{2, 7}) || call.Args[1].P != (Pos{3, 3}) {
		t.Errorf("call at %v, arguments at %v and %v", call.P, call.Args[0].P, call.Args[1].P)
	}
	if n := p.Stmts[0].Value.(*Num); n.P != (Pos{1, 5}) {
		t.Errorf("number at %v", n.P)
	}
}

func TestNoteNames(t *testing.T) {
	for name, want := range map[string]float64{"c3": 48, "a3": 57, "a2": 45, "c4": 60, "c#4": 61, "Bb2": 46, "G5": 79, "b3": 59, "c0": 12} {
		if got, ok := parseNote(name); !ok || got != want {
			t.Errorf("%s: %v %v, want %v", name, got, ok, want)
		}
	}
	for _, name := range []string{"abc", "a", "h3", "a10", "cc4", "a3x", "x3", "e-1"} {
		if _, ok := parseNote(name); ok {
			t.Errorf("%s is not a note", name)
		}
	}
}

func TestSyntaxErrors(t *testing.T) {
	for _, c := range []struct{ src, where, mentions string }{
		{"x = ", "1:5", "expected"},
		{"x = (1 + 2", "1:11", "expected"},
		{"x = f(1, 2", "1:11", "expected"},
		{"x = [1 2", "1:9", "expected"},
		{"x = 1 +", "1:8", "expected"},
		{"x == 1", "1:4", "expected"},
		{"x = 1\ny = ]", "2:5", "expected"},
		{"x = f(res: )", "1:12", "expected"},
		{"1 = x", "1:1", "expected"},
		{"x = a..b", "1:7", "expected"},
	} {
		_, err := Parse(c.src)
		if err == nil {
			t.Errorf("%q: no error", c.src)
			continue
		}
		if !strings.Contains(err.Error(), c.where) || !strings.Contains(err.Error(), c.mentions) {
			t.Errorf("%q: %q does not mention %s and %q", c.src, err, c.where, c.mentions)
		}
	}
}

func TestSharpsAndComments(t *testing.T) {
	// # is a sharp in a note, and is not a comment; // is a comment
	if got, want := parse(t, "x = [c#4 f#3 a3]"), "x = [c#4 f#3 a3]"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	_, err := Parse("x = abc#4")
	if err == nil || !strings.Contains(err.Error(), "not a note") {
		t.Errorf("abc#4: %v", err)
	}
	if _, err := Parse("x = 1 # not a comment"); err == nil {
		t.Error("a # after a value is not a comment")
	}
}

func TestErrorShowsTheLine(t *testing.T) {
	e := &Error{Pos: Pos{2, 5}, Msg: "unknown function lowpas", Hint: "did you mean lowpass?"}
	got := e.Show("a = 1\nb = lowpas(a)\nc = 3")
	want := "2:5: unknown function lowpas (did you mean lowpass?)\n2 | b = lowpas(a)\n        ^"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestParseFunctionsAndIf(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"fn f(a) { a }", "fn f(a) { a }"},
		{"fn f() { 1 }", "fn f() { 1 }"},
		{"fn f(a, b, c) { a + b * c }", "fn f(a, b, c) { (a + (b * c)) }"},
		{"fn f(a) {\n  x = a * 2\n  y = x + 1\n  y.g(2)\n}", "fn f(a) { x = (a * 2); y = (x + 1); y.g(2) }"},
		{"fn f(a) { x = 1\n x }", "fn f(a) { x = 1; x }"},
		{"x = if c { 1 } else { 2 }", "x = if c { 1 } else { 2 }"},
		{"x = if coin(0.5) {\n  a = 1\n  a + 2\n} else {\n  3\n}", "x = if coin(0.5) { a = 1; (a + 2) } else { 3 }"},
		{"x = if a { 1 } else { 2 } * 3", "x = (if a { 1 } else { 2 } * 3)"},
		{"x = (if a { 1 } else { if b { 2 } else { 3 } })", "x = if a { 1 } else { if b { 2 } else { 3 } }"},
		{"fn f(a) { if a { 1 } else { 2 } }", "fn f(a) { if a { 1 } else { 2 } }"},
		{"x = if a.b(1) { c } else { d }.e", "x = if a.b(1) { c } else { d }.e"},
		{"fn f(a) { a }\nout = f(1)", "fn f(a) { a }\nout = f(1)"},
		{"fn f(a) { a } // a comment\nx = 1", "fn f(a) { a }\nx = 1"},
		{"iffy = 1\nelsewhere = 2\nfnord = 3", "iffy = 1\nelsewhere = 2\nfnord = 3"}, // names that start with a word of the language
	} {
		if got := parse(t, c.src); got != c.want {
			t.Errorf("%q\n  got  %s\n  want %s", c.src, got, c.want)
		}
	}
}

func TestParseFunctionErrors(t *testing.T) {
	for _, c := range []struct{ src, mentions string }{
		{"fn f(a) { }", "expected"},                // no value
		{"fn f(a) { x = 1 }", "expected"},          // only a statement
		{"fn f(a { a }", "expected"},               // no )
		{"fn (a) { a }", "expected"},               // no name
		{"fn f(a, ) { a }", "expected"},            // a comma too many
		{"x = if a { 1 }", "expected"},             // no else
		{"x = if { 1 } else { 2 }", "expected"},    // no condition
		{"fn if(a) { a }", "word of the language"}, // reserved
		{"else = 1", "word of the language"},
		{"x = fn", "word of the language"},
		{"fn f(if) { 1 }", "word of the language"},
	} {
		_, err := Parse(c.src)
		if err == nil || !strings.Contains(err.Error(), c.mentions) {
			t.Errorf("%q: %v, want a mention of %q", c.src, err, c.mentions)
		}
	}
}

// a mistake at the end of a long program is reported at once: the parser does not try the other ways to cut up what came before
func TestAMistakeAtTheEndOfALongProgramIsQuick(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		b.WriteString("x" + strings.Repeat("a", i%5) + string(rune('a'+i%26)) + string(rune('a'+i/26)) + " = [1 2 3 4].mix() * (2 + 3) - [4 5].mix()\n")
	}
	b.WriteString("fn f(a, b) { c = a + b\n c * 2 }\nout = f(1, 2)\na")
	start := time.Now()
	_, err := Parse(b.String())
	if err == nil || !strings.Contains(err.Error(), "expected") {
		t.Fatalf("no error: %v", err)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("it took %v", took)
	}
}

// the text of a comment is never program: a dot in it is not a call that goes on from the line before
func TestCommentsAreWholeAndNeverRead(t *testing.T) {
	for _, c := range []string{".d}", "{ SinOsc.ar(exprand(100, 1000), 0, 0.05) ! 2 }", "a.b(c)", ".", "x = 1", "fn f(a) { a }"} {
		for _, src := range []string{
			"seed = 1\n// " + c + "\nout = sine(1).dup()\n",
			"seed = 1 // " + c + "\nout = sine(1).dup()\n",
			"fn f(x) { y = x // " + c + "\n y }\nout = f(1)\n",
		} {
			if _, err := Parse(src); err != nil {
				t.Errorf("%q: %v", src, err)
			}
		}
	}
}

// numbers of two digits were cut in two to try another way to read a long list (the white space between items can be empty): the time to
// parse doubled with each number, so that a list of 30 inside a list never finished
func TestALongListInAListParsesAtOnce(t *testing.T) {
	var nums []string
	for i := 1; i <= 120; i++ {
		nums = append(nums, fmt.Sprint(i*7))
	}
	l := strings.Join(nums, " ")
	for _, src := range []string{
		"x = [[" + l + "]]\nout = sine(1).dup()\n",
		"x = [[[" + l + "]]]\nout = sine(1).dup()\n",
		"x = [[" + l + "]] +\nout = sine(1).dup()\n", // a mistake after it: said, and soon
		"fn abcdefghij1234567890(a) { a }\nx = [abcdefghij1234567890(1) abcdefghij1234567890(2)]\n",
	} {
		start := time.Now()
		Parse(src)
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("it took %v: %.40q", d, src)
		}
	}
	// the numbers are what they were
	p, err := Compile("x = [[10 11 120]]\nout = sine(1).dup()\n")
	if err != nil || p == nil {
		t.Fatal(err)
	}
}
