package lang

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"harp/dsl"
)

// Patch is a program that has been checked and built: the signal to play, and the settings it asked for
type Patch struct {
	Out          dsl.Sig // what is heard: a signal of one or more channels
	BPM          float64
	StepsPerBeat float64
	Seed         uint64
	HasSeed      bool
	// StartSeed seeds the numbers that are drawn while the patch is built, before a seed statement says otherwise (rand, choose, the seeds of
	// the noises): a patch that is built again for a live edit starts from the same one, and draws the same numbers
	StartSeed uint64
	// Info says what the patch made that is good to know (what a sound file was cut into, and on which keys)
	Info []string
	// Dir is the folder of the patch: the files that it names (sounds, SFZ instruments) are looked for there first
	Dir string
	// File is the path of the patch when it is a file (CompileFile), and Files the files that it uses (use "drums.harp"), by their full path: a
	// change in one of them is a change of the patch
	File  string
	Files []string
}

// Settings is the clock of the patch, as dsl wants it
func (p *Patch) Settings() dsl.Patch { return dsl.Patch{BPM: p.BPM, StepsPerBeat: p.StepsPerBeat} }

// Compile reads a program, checks it, and builds the signal graph of it. The statements are done in
// order; seed = 7 seeds the generators that come after it, so put it first. The names with a meaning
// are bpm (default 120), steps (steps in a beat, default 4), seed and out (what is heard)
func Compile(src string) (*Patch, error) { return CompileDir(src, "") }

// CompileDir is Compile for a patch in a folder: the files that it names are looked for there (and in the folder that harp runs in)
func CompileDir(src, dir string) (*Patch, error) {
	prog, err := Parse(src)
	if err != nil {
		return nil, err
	}
	return buildPatch(prog, nil, dir)
}

// CompileFile is Compile for a patch that is a file: the folder of the file is where the files that it names are looked for, and the files that
// it uses (use "drums.harp") are read from there; Patch.Files are all of them, for what watches them
func CompileFile(src, path string) (*Patch, error) {
	prog, err := Parse(src)
	if err != nil {
		return nil, err
	}
	return buildFile(prog, nil, path)
}

// Recompile is Compile for a patch that is edited while it plays (prev is the one that plays): the numbers that are drawn while it is
// built are the same as for prev, so that a save that changes nothing changes nothing, and the generators that are running do not have
// their random stream seeded again (a seed statement seeds only what is drawn while the patch is built)
func Recompile(src string, prev *Patch) (*Patch, error) {
	prog, err := Parse(src)
	if err != nil {
		return nil, err
	}
	if prev.File != "" {
		return buildFile(prog, prev, prev.File)
	}
	return buildPatch(prog, prev, prev.Dir)
}

func buildFile(prog *Program, prev *Patch, path string) (*Patch, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return buildPatchFile(prog, prev, filepath.Dir(abs), abs)
}

// Build checks and builds a program that was parsed
func Build(prog *Program) (*Patch, error) { return buildPatch(prog, nil, "") }

func buildPatch(prog *Program, prev *Patch, dir string) (*Patch, error) {
	return buildPatchFile(prog, prev, dir, "")
}

func buildPatchFile(prog *Program, prev *Patch, dir, file string) (*Patch, error) {
	start := uint64(time.Now().UnixNano())
	if prev != nil {
		start = prev.StartSeed
	}
	dsl.SeedBuild(start)
	sharedSignals = map[any]dsl.Sig{} // a program starts with no generators made into signals
	sharedGates = map[any]dsl.Sig{}
	global := &scope{vars: map[string]varDef{}}
	e := &evaluator{scope: global, global: global, fns: map[string]*Func{}, patch: &Patch{BPM: 120, StepsPerBeat: 4, StartSeed: start, Dir: dir, File: file}, live: prev != nil}
	e.uses = &useState{loaded: map[string]bool{}, fnAt: map[string]Pos{}}
	if file != "" {
		e.uses.loaded[file], e.uses.stack = true, []string{file}
	}
	if err := e.run(prog, dir, false); err != nil {
		return nil, err
	}
	if e.uses.out == nil {
		return nil, &Error{Pos: Pos{Line: 1, Col: 1}, Msg: "the patch has no out: write out = ... with what is to be heard"}
	}
	e.patch.Out = e.uses.out.Sig()
	return e.patch, nil
}

// run does the statements of a program, in order: those of the patch, and those of a file that it uses (used: then there is no out)
func (e *evaluator) run(prog *Program, dir string, used bool) error {
	for _, st := range prog.Stmts {
		if st.Use != "" {
			if err := e.use(st, dir); err != nil {
				return err
			}
			continue
		}
		if st.Fn != nil { // fn name(...) { ... }: known from here on
			if err := e.defineFn(st); err != nil {
				return err
			}
			continue
		}
		v, err := e.expr(st.Value)
		if err != nil {
			return err
		}
		switch st.Name {
		case "bpm", "steps", "seed":
			if v.T.Kind != KNum {
				return &Error{Pos: st.Value.At(), Msg: fmt.Sprintf("%s is a number, not a %s", st.Name, v.T)}
			}
			f := v.Float()
			switch {
			case st.Name == "bpm" && f > 0:
				e.patch.BPM = f
			case st.Name == "steps" && f > 0:
				e.patch.StepsPerBeat = f
			case st.Name == "seed" && f >= 0 && isWhole(f):
				e.patch.Seed, e.patch.HasSeed = uint64(f), true
				if e.live {
					dsl.SeedBuild(e.patch.Seed) // the generators that run go on with their stream
				} else {
					dsl.Seed(e.patch.Seed) // what is made after this draws from the seeded stream
				}
			default:
				return &Error{Pos: st.Value.At(), Msg: fmt.Sprintf("%s = %v is not a possible value", st.Name, f)}
			}
			e.scope.vars[st.Name] = varDef{v, st.P}
		case "out":
			if used {
				return &Error{Pos: st.P, Msg: "out is only in the patch itself, not in a file that is used"}
			}
			if e.uses.out != nil {
				return &Error{Pos: st.P, Msg: fmt.Sprintf("out is already defined, at %s", e.uses.outAt)}
			}
			c, err := coerce(v, TSig)
			if err != nil {
				return &Error{Pos: st.Value.At(), Msg: "out is what is heard: " + err.Error()}
			}
			e.uses.out, e.uses.outAt = &c, st.P
		default:
			if _, ok := consts[st.Name]; ok {
				return &Error{Pos: st.P, Msg: fmt.Sprintf("%s is a name of the language, it can't be changed", st.Name)}
			}
			if err := e.define(st.Name, v, st.P); err != nil {
				return err
			}
		}
	}
	return nil
}

type varDef struct {
	v  Value
	at Pos
}

// scope is the names of a function or a block, and the scope around it
type scope struct {
	vars   map[string]varDef
	parent *scope
}

type evaluator struct {
	uses   *useState // the files that the patch uses, and what the statements of all of them share (only the evaluator of the patch has it)
	live   bool      // the patch is built again while it plays (Recompile)
	scope  *scope    // where names are looked up and defined now
	global *scope    // the names of the program: what the functions can see besides their own
	fns    map[string]*Func
	patch  *Patch
	depth  int // how many functions of the program are being worked out one in the other
}

// maxDepth is how many calls of functions of the program may be one in the other: a function that calls itself would not end
const maxDepth = 200

func (e *evaluator) lookup(name string) (varDef, bool) {
	for s := e.scope; s != nil; s = s.parent {
		if d, ok := s.vars[name]; ok {
			return d, true
		}
	}
	return varDef{}, false
}

// define makes a name in the scope that is now: a name that is there already in that same scope is not made again (one in a scope
// around it is hidden)
func (e *evaluator) define(name string, v Value, at Pos) error {
	if _, ok := consts[name]; ok {
		return &Error{Pos: at, Msg: fmt.Sprintf("%s is a name of the language, it can't be changed", name)}
	}
	if prev, ok := e.scope.vars[name]; ok {
		return &Error{Pos: at, Msg: fmt.Sprintf("%s is already defined, at %s", name, e.where(prev.at))}
	}
	e.scope.vars[name] = varDef{v, at}
	return nil
}

func (e *evaluator) names() []string {
	var names []string
	for s := e.scope; s != nil; s = s.parent {
		for n := range s.vars {
			names = append(names, n)
		}
	}
	for n := range consts {
		names = append(names, n)
	}
	for n := range e.fns {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// defineFn makes a function of the program known. It sees the names of the program, and its own, and nothing else: a function
// does not see the names of the place it is called from
func (e *evaluator) defineFn(st Stmt) error {
	if _, ok := e.fns[st.Name]; ok {
		return &Error{Pos: st.P, Msg: fmt.Sprintf("the function %s is already defined, at %s", st.Name, e.where(e.fnWhere(st.Name)))}
	}
	if e.uses != nil {
		e.uses.fnAt[st.Name] = st.P
	}
	seen := map[string]bool{}
	params := make([]Param, len(st.Fn.Params))
	for i, p := range st.Fn.Params {
		if seen[p] {
			return &Error{Pos: st.P, Msg: fmt.Sprintf("%s is a parameter of %s twice", p, st.Name)}
		}
		seen[p] = true
		params[i] = Param{Name: p, Type: TAny}
	}
	def, name := st.Fn, st.Name
	f := &Func{Name: name, Params: params, Result: TAny, Doc: "a function of the program"}
	f.Call = func(a Args) (Value, error) {
		if a.ev.depth >= maxDepth {
			return Value{}, fmt.Errorf("%s is called from %s, %d deep: a function that calls itself does not end", name, name, maxDepth)
		}
		inner := &evaluator{global: a.ev.global, fns: a.ev.fns, patch: a.ev.patch, depth: a.ev.depth + 1}
		inner.scope = &scope{vars: map[string]varDef{}, parent: inner.global}
		for i, p := range def.Params {
			inner.scope.vars[p] = varDef{a.vals[i], st.P}
		}
		return inner.block(def.Body)
	}
	e.fns[name] = f
	return nil
}

// block works out the statements of a block in a scope of their own, and then the expression that is its value
func (e *evaluator) block(b *Block) (Value, error) {
	inner := &evaluator{scope: &scope{vars: map[string]varDef{}, parent: e.scope}, global: e.global, fns: e.fns, patch: e.patch, depth: e.depth}
	for _, st := range b.Stmts {
		v, err := inner.expr(st.Value)
		if err != nil {
			return Value{}, err
		}
		if err := inner.define(st.Name, v, st.P); err != nil {
			return Value{}, err
		}
	}
	return inner.expr(b.Result)
}

// ifExpr works out the condition, and then only the block that is taken: what is in the other is never made (nothing of it is built, no random number is drawn)
func (e *evaluator) ifExpr(x *If) (Value, error) {
	c, err := e.expr(x.Cond)
	if err != nil {
		return Value{}, err
	}
	if c.T.Kind != KBool {
		return Value{}, &Error{Pos: x.Cond.At(), Msg: fmt.Sprintf("if needs true or false, this is a %s", c.T), Hint: "coin(0.5) gives true half of the time"}
	}
	if c.V.(bool) {
		return e.block(x.Then)
	}
	return e.block(x.Else)
}

// expr works out what an expression is, from the inside out: the types of its parts say what it is
func (e *evaluator) expr(x Expr) (Value, error) {
	switch x := x.(type) {
	case *Num:
		return numV(x.V), nil
	case *NoteName:
		return numV(x.V), nil
	case *Str:
		return strV(x.V), nil
	case *If:
		return e.ifExpr(x)
	case *Name:
		if d, ok := e.lookup(x.Name); ok {
			return d.v, nil
		}
		if c, ok := consts[x.Name]; ok {
			return c, nil
		}
		if f, ok := e.fns[x.Name]; ok { // a function of the program is a value: it can be given to repeat
			return Value{TFunc, f}, nil
		}
		if _, ok := funcs[x.Name]; ok {
			return Value{}, &Error{Pos: x.P, Msg: fmt.Sprintf("%s is a function: write %s(...) to call it", x.Name, x.Name)}
		}
		return Value{}, &Error{Pos: x.P, Msg: fmt.Sprintf("%s is not defined%s", x.Name, didYouMean(x.Name, e.names()))}
	case *List:
		items := make([]Value, len(x.Items))
		for i, it := range x.Items {
			v, err := e.expr(it)
			if err != nil {
				return Value{}, err
			}
			items[i] = v
		}
		return listV(items), nil
	case *Neg:
		v, err := e.expr(x.X)
		if err != nil {
			return Value{}, err
		}
		if v.T.Kind == KNum {
			return numV(-v.Float()), nil
		}
		s, err := coerce(v, TSig)
		if err != nil {
			return Value{}, &Error{Pos: x.P, Msg: "can't negate: " + err.Error()}
		}
		return sigV(dsl.Mul(s.Sig(), -1.0)), nil
	case *Bin:
		return e.binary(x)
	case *Field:
		recv, err := e.expr(x.Recv)
		if err != nil {
			return Value{}, err
		}
		return field(recv, x)
	case *Call:
		return e.call(x)
	}
	return Value{}, fmt.Errorf("unknown expression %T", x)
}

// field is recv.name: the parts of a voice
func field(recv Value, x *Field) (Value, error) {
	if recv.T.Kind == KVoice {
		switch x.Name {
		case "pitch":
			return sigV(recv.V.(dsl.Voice).Pitch), nil
		case "gate":
			return sigV(recv.V.(dsl.Voice).Gate), nil
		}
		return Value{}, &Error{Pos: x.P, Msg: fmt.Sprintf("a voice has no %s, it has pitch and gate%s", x.Name, didYouMean(x.Name, []string{"pitch", "gate"}))}
	}
	return Value{}, &Error{Pos: x.P, Msg: fmt.Sprintf("a %s has no fields (.%s): did you mean a call, .%s(...)?", recv.T, x.Name, x.Name)}
}

// binary is + - * /: on numbers it is arithmetic, done now; otherwise on signals (a number or a list
// stands for a signal), where it is a node of the graph
func (e *evaluator) binary(x *Bin) (Value, error) {
	l, err := e.expr(x.L)
	if err != nil {
		return Value{}, err
	}
	r, err := e.expr(x.R)
	if err != nil {
		return Value{}, err
	}
	if l.T.Kind == KNum && r.T.Kind == KNum {
		a, b := l.Float(), r.Float()
		switch x.Op {
		case '+':
			return numV(a + b), nil
		case '-':
			return numV(a - b), nil
		case '*':
			return numV(a * b), nil
		case '/':
			if b == 0 {
				return Value{}, &Error{Pos: x.P, Msg: "division by zero"}
			}
			return numV(a / b), nil
		}
	}
	ls, errL := coerce(l, TSig)
	rs, errR := coerce(r, TSig)
	if errL != nil || errR != nil {
		bad, err := l, errL
		if errL == nil {
			bad, err = r, errR
		}
		return Value{}, &Error{Pos: x.P, Msg: fmt.Sprintf("%c works on numbers and signals, and one side is a %s", x.Op, bad.T), Hint: err.Error()}
	}
	a, b := ls.Sig(), rs.Sig()
	if (a.Channels() != b.Channels()) && a.Channels() != 1 && b.Channels() != 1 {
		return Value{}, &Error{Pos: x.P, Msg: fmt.Sprintf("can't combine signals of %d and %d channels (one channel goes with any number)", a.Channels(), b.Channels())}
	}
	switch x.Op {
	case '+':
		return sigV(dsl.Add(a, b)), nil
	case '*':
		return sigV(dsl.Mul(a, b)), nil
	case '-':
		return sigV(dsl.Add(a, dsl.Mul(b, -1.0))), nil
	case '/':
		if r.T.Kind != KNum {
			return Value{}, &Error{Pos: x.P, Msg: "a signal can be divided by a number, not by another signal"}
		}
		if r.Float() == 0 {
			return Value{}, &Error{Pos: x.P, Msg: "division by zero"}
		}
		return sigV(dsl.Mul(a, 1/r.Float())), nil
	}
	return Value{}, fmt.Errorf("unknown operator %c", x.Op)
}

// call is f(args) and recv.f(args): the receiver is the first argument. Arguments are worked out
// from left to right, so the generators that are made in them are made in the order they are written
func (e *evaluator) call(c *Call) (Value, error) {
	type given struct {
		at   Pos
		name string
		v    Value
	}
	var args []given
	// the receiver is first in the text, so it is checked first; for f(args) the name of the function is
	if c.Recv != nil {
		v, err := e.expr(c.Recv)
		if err != nil {
			return Value{}, err
		}
		args = append(args, given{c.Recv.At(), "", v})
	}
	f, ok := e.fns[c.Name]
	if !ok {
		f, ok = funcs[c.Name]
	}
	if !ok { // a name that was written in one word: it is the function that it is now called
		if now, renamed := renamedFuncs[c.Name]; renamed {
			if f, ok = funcs[now]; ok {
				e.renamedNote(c.Name, now)
			}
		}
	}
	if !ok {
		names := make([]string, 0, len(funcs)+len(e.fns))
		for n := range funcs {
			names = append(names, n)
		}
		for n := range e.fns {
			names = append(names, n)
		}
		return Value{}, &Error{Pos: c.P, Msg: fmt.Sprintf("there is no function %s%s", c.Name, didYouMean(c.Name, names))}
	}
	sawNamed := false
	for _, a := range c.Args {
		if a.Name != "" {
			sawNamed = true
		} else if sawNamed {
			return Value{}, &Error{Pos: a.P, Msg: "a positional argument can't come after a named one"}
		}
		v, err := e.expr(a.Value)
		if err != nil {
			return Value{}, err
		}
		args = append(args, given{a.P, a.Name, v})
	}

	vals := make([]Value, len(f.Params))
	set := make([]bool, len(f.Params))
	next := 0
	for _, g := range args {
		var i int
		if g.name == "" {
			if next >= len(f.Params) {
				return Value{}, &Error{Pos: g.at, Msg: fmt.Sprintf("%s takes %d arguments, this is one more", f.Name, len(f.Params)), Hint: describe(f)}
			}
			i = next
			next++
		} else {
			i = -1
			names := make([]string, len(f.Params))
			for k, p := range f.Params {
				names[k] = p.Name
				if p.Name == g.name {
					i = k
				}
			}
			if now, renamed := renamedParams[g.name]; i < 0 && renamed { // the old spelling of a parameter
				for k, p := range f.Params {
					if p.Name == now {
						i = k
						e.renamedNote(g.name, now)
					}
				}
			}
			if i < 0 {
				return Value{}, &Error{Pos: g.at, Msg: fmt.Sprintf("%s has no parameter %s%s", f.Name, g.name, didYouMean(g.name, names)), Hint: describe(f)}
			}
		}
		if set[i] {
			return Value{}, &Error{Pos: g.at, Msg: fmt.Sprintf("%s is given twice to %s", f.Params[i].Name, f.Name)}
		}
		v, err := coerce(g.v, f.Params[i].Type)
		if err != nil {
			return Value{}, &Error{Pos: g.at, Msg: fmt.Sprintf("%s: %s: %s", f.Name, f.Params[i].Name, err.Error())}
		}
		vals[i], set[i] = v, true
	}
	for i, p := range f.Params {
		if set[i] {
			continue
		}
		if p.Default == nil {
			return Value{}, &Error{Pos: c.P, Msg: fmt.Sprintf("%s needs %s", f.Name, p.Name), Hint: describe(f)}
		}
		v, err := coerce(*p.Default, p.Type)
		if err != nil {
			return Value{}, err
		}
		vals[i] = v
	}
	out, err := f.Call(Args{f: f, vals: vals, ev: e})
	if err == nil {
		out = fingerprinted(out, f.Name, vals)
	}
	if pe, ok := err.(*Error); ok { // a mistake inside a function of the program keeps its own place
		return Value{}, pe
	}
	if err != nil {
		return Value{}, &Error{Pos: c.P, Msg: fmt.Sprintf("%s: %s", f.Name, err.Error())}
	}
	return out, nil
}

// apply calls a function that is a value (one of the program) with arguments that are already worked out
func (e *evaluator) apply(f *Func, args ...Value) (Value, error) {
	if len(args) != len(f.Params) {
		return Value{}, fmt.Errorf("%s takes %d arguments, it is called with %d", f.Name, len(f.Params), len(args))
	}
	vals := make([]Value, len(args))
	for i, v := range args {
		c, err := coerce(v, f.Params[i].Type)
		if err != nil {
			return Value{}, fmt.Errorf("%s: %s: %w", f.Name, f.Params[i].Name, err)
		}
		vals[i] = c
	}
	return f.Call(Args{f: f, vals: vals, ev: e})
}
