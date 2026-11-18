// Package lang is the text language of harp: patches written as text, checked for rates and channels,
// and built into the signal graph of harp/dsl.
//
// This file is the syntax: a grammar of exbana patterns that makes a syntax tree. A program is a list
// of statements, each a name and an expression:
//
//	melody = degrees(a3, minor, mutate([7 9 11 9 8 7 4 7], swap: 0.6))
//	voice  = notes(melody, euclid(16, 7), length: 0.6)
//	lead   = saw([voice.pitch * 0.997, voice.pitch * 1.023])
//	  .lowpass(cutoff: lfo(0.3).range(300, 4000), res: 0.5)
//	  .gain(adsr(0.005, 0.15, 0.5, 0.2).on(voice.gate))
//	out = lead * 0.4
//
// A function of your own is fn name(a, b) { statements and then the value }, and if cond { ... } else { ... } is
// an expression: only the branch that is taken is worked out.
//
// x.f(a, b) is f(x, a, b): the thing before the dot is the first argument. A name after a dot without
// parentheses is a field (voice.pitch). Arguments can be named (res: 0.5). Newlines are white space, so
// a line that starts with a dot goes on with the line before it. A // starts a comment.
package lang

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	ebnf "github.com/almerlucke/exbana/v3"
	"github.com/almerlucke/exbana/v3/patterns/alternation"
	"github.com/almerlucke/exbana/v3/patterns/concatenation"
	"github.com/almerlucke/exbana/v3/patterns/end"
	ent "github.com/almerlucke/exbana/v3/patterns/entity"
	"github.com/almerlucke/exbana/v3/patterns/repetition"
	"github.com/almerlucke/exbana/v3/patterns/rule"
	"github.com/almerlucke/exbana/v3/patterns/vector"
	"github.com/almerlucke/exbana/v3/readers/runes"
)

// Pos is a place in the source, from 1
type Pos struct{ Line, Col int }

func (p Pos) String() string { return fmt.Sprintf("%d:%d", p.Line, p.Col) }

// Expr is an expression of the syntax tree
type Expr interface {
	At() Pos
	String() string
}

type (
	// Num is a number
	Num struct {
		P Pos
		V float64
	}
	// NoteName is a note written as a name (c#4), a number of the scale of MIDI notes
	NoteName struct {
		P    Pos
		V    float64
		Text string
	}
	// Str is a string
	Str struct {
		P Pos
		V string
	}
	// Name is a name: a variable, or a constant like minor
	Name struct {
		P    Pos
		Name string
	}
	// List is [a b c] or [a, b, c]
	List struct {
		P     Pos
		Items []Expr
	}
	// Call is f(args), or recv.f(args): Recv is the first argument if there is one
	Call struct {
		P    Pos
		Recv Expr // nil for f(args)
		Name string
		Args []Arg
	}
	// Field is recv.name, without parentheses
	Field struct {
		P    Pos
		Recv Expr
		Name string
	}
	// Neg is -x
	Neg struct {
		P Pos
		X Expr
	}
	// If is if cond { ... } else { ... }
	If struct {
		P          Pos
		Cond       Expr
		Then, Else *Block
	}
	// Bin is l op r, op one of + - * /
	Bin struct {
		P    Pos
		Op   byte
		L, R Expr
	}
)

// Arg is an argument of a call, with a name or without
type Arg struct {
	P     Pos
	Name  string // "" for a positional argument
	Value Expr
}

// Stmt is name = expr, or with Fn set a function: fn name(params) { body }
type Stmt struct {
	P     Pos
	Name  string
	Value Expr
	Fn    *FnDef
}

// FnDef is a function of the program: its parameters, and the block that is its value
type FnDef struct {
	Params []string
	Body   *Block
}

// Block is { name = expr ... result }: statements of its own, and the expression that is its value
type Block struct {
	P      Pos
	Stmts  []Stmt
	Result Expr
}

func (b *Block) String() string {
	parts := make([]string, 0, len(b.Stmts)+1)
	for _, st := range b.Stmts {
		parts = append(parts, st.Name+" = "+st.Value.String())
	}
	return "{ " + strings.Join(append(parts, b.Result.String()), "; ") + " }"
}

// Program is the statements of a source, in order
type Program struct{ Stmts []Stmt }

func (e *Num) At() Pos      { return e.P }
func (e *NoteName) At() Pos { return e.P }
func (e *Str) At() Pos      { return e.P }
func (e *Name) At() Pos     { return e.P }
func (e *List) At() Pos     { return e.P }
func (e *Call) At() Pos     { return e.P }
func (e *Field) At() Pos    { return e.P }
func (e *Neg) At() Pos      { return e.P }
func (e *Bin) At() Pos      { return e.P }
func (e *If) At() Pos       { return e.P }

func (e *Num) String() string      { return strconv.FormatFloat(e.V, 'g', -1, 64) }
func (e *NoteName) String() string { return e.Text }
func (e *Str) String() string      { return strconv.Quote(e.V) }
func (e *Name) String() string     { return e.Name }
func (e *Neg) String() string      { return "(-" + e.X.String() + ")" }
func (e *Bin) String() string {
	return "(" + e.L.String() + " " + string(e.Op) + " " + e.R.String() + ")"
}
func (e *Field) String() string { return e.Recv.String() + "." + e.Name }
func (e *List) String() string {
	parts := make([]string, len(e.Items))
	for i, it := range e.Items {
		parts[i] = it.String()
	}
	return "[" + strings.Join(parts, " ") + "]"
}
func (e *Call) String() string {
	parts := make([]string, len(e.Args))
	for i, a := range e.Args {
		parts[i] = a.String()
	}
	s := e.Name + "(" + strings.Join(parts, ", ") + ")"
	if e.Recv != nil {
		s = e.Recv.String() + "." + s
	}
	return s
}
func (a Arg) String() string {
	if a.Name != "" {
		return a.Name + ": " + a.Value.String()
	}
	return a.Value.String()
}

// String prints the program as the tree it is: one statement a line, every operator in parentheses
func (e *If) String() string {
	return "if " + e.Cond.String() + " " + e.Then.String() + " else " + e.Else.String()
}

func (p *Program) String() string {
	var b strings.Builder
	for _, s := range p.Stmts {
		if s.Fn != nil {
			fmt.Fprintf(&b, "fn %s(%s) %s\n", s.Name, strings.Join(s.Fn.Params, ", "), s.Fn.Body)
			continue
		}
		fmt.Fprintf(&b, "%s = %s\n", s.Name, s.Value)
	}
	return b.String()
}

type (
	pattern = ebnf.Pattern[rune, runes.Pos]
	match   = ebnf.Match[rune, runes.Pos]
	reader  = ebnf.Reader[rune, runes.Pos]
)

// reserved are the words of the language that are not names
var reserved = map[string]bool{"fn": true, "if": true, "else": true}

func reservedError(p Pos, s string) error {
	return &Error{Pos: p, Msg: fmt.Sprintf("%s is a word of the language, it can't be used as a name", s)}
}

var noteName = regexp.MustCompile(`^([a-gA-G])([#b]?)([0-9])$`)

var noteSemitone = map[byte]int{'c': 0, 'd': 2, 'e': 4, 'f': 5, 'g': 7, 'a': 9, 'b': 11}

// parseNote reads c#4, Bb2, a3 as a MIDI note, with C3 as 48 (the numbering of psalm)
func parseNote(s string) (float64, bool) {
	m := noteName.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	semis := noteSemitone[byte(unicode.ToLower(rune(m[1][0])))]
	switch m[2] {
	case "#":
		semis++
	case "b":
		semis--
	}
	octave, _ := strconv.Atoi(m[3])
	return float64(12*(octave+1) + semis), true
}

// at is where a match begins, counted from 1 (the positions of exbana start at 0)
func at(m *match) Pos { return Pos{Line: m.Begin.Line + 1, Col: m.Begin.Col + 1} }

func textOf(m *match, r reader) (string, error) {
	rs, err := r.Range(m.Begin, m.End)
	return string(rs), err
}

func exprOf(m *match, r reader) (Expr, error) {
	v, err := m.Eval(r)
	if err != nil {
		return nil, err
	}
	return v.(Expr), nil
}

func delegate(m *match, r reader) (any, error) { return m.Components[0].Eval(r) }

// grammar is the grammar of the language, in white space insensitive form: every rule leaves the white
// space after it to the rule that holds it
func grammar() pattern {
	conc := func(ps ...pattern) pattern { return concatenation.New(ps...) }
	alt := func(ps ...pattern) pattern { return alternation.New(ps...) }
	rep := func(p pattern, min, max int) *repetition.Repetition[rune, runes.Pos] {
		return repetition.New(p, min, max)
	}
	any0 := func(p pattern) pattern { return rep(p, 0, 0) }
	// a repetition that never gives back what it matched: where the next thing cannot start the same way as the repeated one. Without
	// it a mistake at the end of a long program makes the parser try every other way to cut up what came before (it takes minutes)
	every := func(p pattern) pattern { return rep(p, 0, 0).SetPossessive(true) }
	opt := func(p pattern) pattern { return rep(p, 0, 1) }
	more1 := func(p pattern) pattern { return rep(p, 1, 0) }
	// the longest run, never given back: 18 is a number, not 1 and then 8 (the white space between the items of a list can be empty, so
	// without it every number of two digits could be cut in two, and a mistake after a long list made the parser try all of the ways)
	run := func(p pattern) pattern { return rep(p, 1, 0).SetPossessive(true) }
	char := func(r rune) pattern {
		return ent.New[rune, runes.Pos](func(x rune) bool { return x == r }).Printed(strconv.Quote(string(r)))
	}
	class := func(f func(rune) bool, name string) pattern { return ent.New[rune, runes.Pos](f).Named(name) }

	space := class(unicode.IsSpace, "space")
	// white space and comments are taken whole and never given back: a comment that is cut short would leave its text to be read as
	// program (a "." in a comment is the dot of a call)
	comment := conc(char('/'), char('/'), every(class(func(r rune) bool { return r != '\n' }, "comment")))
	ws := rule.New("white space", rep(alt(space, comment), 0, 0).SetPossessive(true)).SetAtomic(true)
	digit := class(func(r rune) bool { return r >= '0' && r <= '9' }, "digit")
	letter := class(func(r rune) bool { return unicode.IsLetter(r) || r == '_' }, "letter")

	number := rule.New("number", conc(
		run(digit),
		opt(conc(char('.'), run(digit))),
		opt(conc(class(func(r rune) bool { return r == 'e' || r == 'E' }, "exponent"),
			opt(class(func(r rune) bool { return r == '+' || r == '-' }, "sign")), run(digit))),
	)).SetAtomic(true).SetEval(func(m *match, r reader) (any, error) {
		s, err := textOf(m, r)
		if err != nil {
			return nil, err
		}
		v, err := strconv.ParseFloat(s, 64)
		return &Num{P: at(m), V: v}, err
	})

	str := rule.New("string", conc(char('"'), any0(class(func(r rune) bool { return r != '"' && r != '\n' }, "character")), char('"'))).
		SetAtomic(true).SetEval(func(m *match, r reader) (any, error) {
		s, err := textOf(m, r)
		return &Str{P: at(m), V: s[1 : len(s)-1]}, err
	})

	identifier := func() pattern { return conc(letter, every(alt(letter, digit))) }
	// a word is a name, or a note if it is written as one (a3, bb2, c#4: the sharp is part of the word)
	word := rule.New("name", conc(identifier(), opt(conc(char('#'), digit)))).SetAtomic(true).SetEval(func(m *match, r reader) (any, error) {
		s, err := textOf(m, r)
		if err != nil {
			return nil, err
		}
		if reserved[s] {
			return nil, reservedError(at(m), s)
		}
		if v, ok := parseNote(s); ok {
			return &NoteName{P: at(m), V: v, Text: s}, nil
		}
		if strings.Contains(s, "#") {
			return nil, &Error{Pos: at(m), Msg: fmt.Sprintf("%s is not a note (a note is a letter from a to g, an optional # or b, and an octave: c#4)", s)}
		}
		return &Name{P: at(m), Name: s}, nil
	})
	ident := rule.New("name", identifier()).SetAtomic(true).SetEval(func(m *match, r reader) (any, error) {
		s, err := textOf(m, r)
		if err == nil && reserved[s] {
			return nil, reservedError(at(m), s)
		}
		return s, err
	})
	keyword := func(w string) pattern { return vector.Text[runes.Pos](w) }
	spaces1 := more1(space)
	block := rule.Declare[rune, runes.Pos]("block")

	expr := rule.Declare[rune, runes.Pos]("expression").SetEval(foldBinary)
	unary := rule.Declare[rune, runes.Pos]("unary").SetEval(delegate)
	item := rule.Declare[rune, runes.Pos]("list item").SetEval(delegate)

	named := rule.New("named argument", conc(ident, ws, char(':'), ws, expr)).SetEval(func(m *match, r reader) (any, error) {
		name, err := m.Components[0].Eval(r)
		if err != nil {
			return nil, err
		}
		v, err := exprOf(m.Components[4], r)
		return Arg{P: at(m), Name: name.(string), Value: v}, err
	})
	arg := rule.New("argument", alt(named, expr)).SetEval(func(m *match, r reader) (any, error) {
		v, err := m.Components[0].Eval(r)
		if err != nil {
			return nil, err
		}
		if a, ok := v.(Arg); ok {
			return a, nil
		}
		return Arg{P: at(m), Value: v.(Expr)}, nil
	})
	args := rule.New("arguments", conc(opt(conc(arg, any0(conc(ws, char(','), ws, arg)), opt(conc(ws, char(','))))))).
		SetEval(func(m *match, r reader) (any, error) {
			var out []Arg
			if list := m.Components[0].Components; len(list) == 1 {
				first, err := list[0].Components[0].Eval(r)
				if err != nil {
					return nil, err
				}
				out = append(out, first.(Arg))
				for _, more := range list[0].Components[1].Components {
					a, err := more.Components[3].Eval(r)
					if err != nil {
						return nil, err
					}
					out = append(out, a.(Arg))
				}
			}
			return out, nil
		})

	// f(args): no white space before the parenthesis, so that a line that starts with ( after a line
	// that ends with a name is not a call
	call := rule.New("call", conc(ident, char('('), ws, args, ws, char(')'))).SetEval(func(m *match, r reader) (any, error) {
		name, _ := m.Components[0].Eval(r)
		a, err := m.Components[3].Eval(r)
		if err != nil {
			return nil, err
		}
		return &Call{P: at(m), Name: name.(string), Args: a.([]Arg)}, nil
	})

	// [a * 2, b] with commas between any expressions, or [7 9 11] with white space between plain items
	// (a number, a name, a call, a negative number)
	commaList := rule.New("comma list", conc(expr, more1(conc(ws, char(','), ws, opt(expr))))).SetEval(func(m *match, r reader) (any, error) {
		first, err := exprOf(m.Components[0], r)
		items := []Expr{first}
		for _, more := range m.Components[1].Components {
			if e := more.Components[3].Components; len(e) == 1 { // after the last comma there may be nothing
				x, err := exprOf(e[0], r)
				if err != nil {
					return nil, err
				}
				items = append(items, x)
			}
		}
		return items, err
	})
	spaceList := rule.New("space list", conc(item, any0(conc(ws, item)))).SetEval(func(m *match, r reader) (any, error) {
		first, err := exprOf(m.Components[0], r)
		items := []Expr{first}
		for _, more := range m.Components[1].Components {
			x, err := exprOf(more.Components[1], r)
			if err != nil {
				return nil, err
			}
			items = append(items, x)
		}
		return items, err
	})
	list := rule.New("list", conc(char('['), ws, opt(alt(commaList, spaceList)), ws, char(']'))).SetEval(func(m *match, r reader) (any, error) {
		l := &List{P: at(m)}
		if body := m.Components[2].Components; len(body) == 1 {
			items, err := body[0].Components[0].Eval(r)
			if err != nil {
				return nil, err
			}
			l.Items = items.([]Expr)
		}
		return l, nil
	})

	paren := rule.New("parenthesis", conc(char('('), ws, expr, ws, char(')'))).SetEval(func(m *match, r reader) (any, error) {
		return exprOf(m.Components[2], r)
	})
	// if cond { ... } else { ... }
	ifExpr := rule.New("if", conc(keyword("if"), spaces1, expr, ws, block, ws, keyword("else"), ws, block)).SetEval(func(m *match, r reader) (any, error) {
		cond, err := exprOf(m.Components[2], r)
		if err != nil {
			return nil, err
		}
		then, err := m.Components[4].Eval(r)
		if err != nil {
			return nil, err
		}
		els, err := m.Components[8].Eval(r)
		if err != nil {
			return nil, err
		}
		return &If{P: at(m), Cond: cond, Then: then.(*Block), Else: els.(*Block)}, nil
	})
	primary := rule.New("primary", alt(number, str, list, paren, ifExpr, call, word)).SetEval(delegate)

	// primary { . name [ (args) ] }
	member := conc(ws, char('.'), ws, ident, opt(conc(char('('), ws, args, ws, char(')'))))
	postfix := rule.New("postfix", conc(primary, any0(member))).SetEval(func(m *match, r reader) (any, error) {
		acc, err := exprOf(m.Components[0], r)
		if err != nil {
			return nil, err
		}
		for _, mem := range m.Components[1].Components {
			name, _ := mem.Components[3].Eval(r)
			p := at(mem.Components[1])
			if parens := mem.Components[4].Components; len(parens) == 1 {
				a, err := parens[0].Components[2].Eval(r)
				if err != nil {
					return nil, err
				}
				acc = &Call{P: p, Recv: acc, Name: name.(string), Args: a.([]Arg)}
			} else {
				acc = &Field{P: p, Recv: acc, Name: name.(string)}
			}
		}
		return acc, nil
	})

	negation := rule.New("negation", conc(char('-'), ws, unary)).SetEval(func(m *match, r reader) (any, error) {
		x, err := exprOf(m.Components[2], r)
		if n, ok := x.(*Num); ok { // -2 is a number
			return &Num{P: at(m), V: -n.V}, err
		}
		return &Neg{P: at(m), X: x}, err
	})
	unary.Define(alt(negation, postfix))
	// a plain item of a list: what can stand between two spaces without an operator
	item.Define(alt(negation, postfix))

	term := rule.New("term", conc(unary, any0(conc(ws, class(func(r rune) bool { return r == '*' || r == '/' }, "operator"), ws, unary)))).
		SetEval(foldBinary)
	expr.Define(conc(term, any0(conc(ws, class(func(r rune) bool { return r == '+' || r == '-' }, "operator"), ws, term))))

	assignment := rule.New("statement", conc(ident, ws, char('='), ws, expr)).SetEval(func(m *match, r reader) (any, error) {
		name, err := m.Components[0].Eval(r)
		if err != nil {
			return nil, err
		}
		v, err := exprOf(m.Components[4], r)
		return Stmt{P: at(m), Name: name.(string), Value: v}, err
	})
	// { name = expr ... result }
	block.Define(conc(char('{'), ws, any0(conc(assignment, ws)), expr, ws, char('}')))
	block.SetEval(func(m *match, r reader) (any, error) {
		b := &Block{P: at(m)}
		for _, s := range m.Components[2].Components {
			v, err := s.Components[0].Eval(r)
			if err != nil {
				return nil, err
			}
			b.Stmts = append(b.Stmts, v.(Stmt))
		}
		res, err := exprOf(m.Components[3], r)
		b.Result = res
		return b, err
	})
	// fn name(a, b) { ... }
	params := conc(ident, any0(conc(ws, char(','), ws, ident)))
	fnDef := rule.New("function", conc(keyword("fn"), spaces1, ident, char('('), ws, opt(params), ws, char(')'), ws, block)).SetEval(func(m *match, r reader) (any, error) {
		name, err := m.Components[2].Eval(r)
		if err != nil {
			return nil, err
		}
		def := &FnDef{}
		if ps := m.Components[5].Components; len(ps) == 1 {
			first, err := ps[0].Components[0].Eval(r)
			if err != nil {
				return nil, err
			}
			def.Params = append(def.Params, first.(string))
			for _, more := range ps[0].Components[1].Components {
				p, err := more.Components[3].Eval(r)
				if err != nil {
					return nil, err
				}
				def.Params = append(def.Params, p.(string))
			}
		}
		body, err := m.Components[9].Eval(r)
		if err != nil {
			return nil, err
		}
		def.Body = body.(*Block)
		return Stmt{P: at(m), Name: name.(string), Fn: def}, nil
	})
	topItem := rule.New("item", alt(fnDef, assignment)).SetEval(delegate)
	return rule.New("program", conc(ws, every(conc(topItem, ws)), end.New[rune, runes.Pos]().Printed("end of the program"))).
		SetEval(func(m *match, r reader) (any, error) {
			p := &Program{}
			for _, s := range m.Components[1].Components {
				v, err := s.Components[0].Eval(r)
				if err != nil {
					return nil, err
				}
				p.Stmts = append(p.Stmts, v.(Stmt))
			}
			return p, nil
		})
}

// foldBinary folds `operand { operator operand }` to the left, for the rules of terms and expressions
func foldBinary(m *match, r reader) (any, error) {
	acc, err := exprOf(m.Components[0], r)
	if err != nil {
		return nil, err
	}
	for _, item := range m.Components[1].Components {
		op, err := textOf(item.Components[1], r)
		if err != nil {
			return nil, err
		}
		right, err := exprOf(item.Components[3], r)
		if err != nil {
			return nil, err
		}
		acc = &Bin{P: at(item.Components[1]), Op: op[0], L: acc, R: right}
	}
	return acc, nil
}

// Parse reads a source into a syntax tree. A syntax error says where it is and what was expected there
func Parse(src string) (*Program, error) {
	rd, err := runes.New(strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	log := runes.NewFarthestLog(rd)
	ctx := ebnf.NewParseContext[rune, runes.Pos](rd).WithLogger(log)
	matched, m, err := ebnf.Parse(grammar(), ebnf.Reader[rune, runes.Pos](ctx))
	if err != nil {
		return nil, err
	}
	if !matched {
		// the farthest place the parser got to, and what would have been right there
		if se := log.Last(); se != nil {
			msg := se.Error()
			if i := strings.Index(msg, ": "); i >= 0 {
				msg = msg[i+2:]
			}
			return nil, &Error{Pos: Pos{Line: se.Pos.Line + 1, Col: se.Pos.Col + 1}, Msg: "syntax error: " + msg}
		}
		return nil, &Error{Pos: Pos{1, 1}, Msg: "syntax error"}
	}
	v, err := m.Eval(ebnf.Reader[rune, runes.Pos](ctx))
	if err != nil {
		return nil, err
	}
	return v.(*Program), nil
}
