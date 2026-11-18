package lang

import (
	"fmt"
	"strings"
	"testing"

	"harp/blocks"
	"harp/dsl"
)

// evalLast works out the statements of a program and gives the value of the last
func evalLast(t *testing.T, src string) Value { return evalLastIn(t, "", src) }

// evalLastIn is evalLast for a patch in a folder (the files it names are looked for there)
func evalLastIn(t *testing.T, dir, src string) Value {
	t.Helper()
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("%v\n%s", err, src)
	}
	sharedSignals = map[any]dsl.Sig{}
	sharedGates = map[any]dsl.Sig{}
	global := &scope{vars: map[string]varDef{}}
	e := &evaluator{scope: global, global: global, fns: map[string]*Func{}, patch: &Patch{BPM: 120, StepsPerBeat: 4, Dir: dir}}
	var v Value
	for _, st := range prog.Stmts {
		if st.Fn != nil {
			if err := e.defineFn(st); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if v, err = e.expr(st.Value); err != nil {
			t.Fatalf("%v\n%s", err, src)
		}
		if st.Name == "seed" {
			dsl.Seed(uint64(v.Float()))
		}
		e.scope.vars[st.Name] = varDef{v, st.P}
	}
	return v
}

// pull takes n values from the generator that the last statement makes, as text (hits as x and ., numbers as they are, chords and notes as lists)
func pull(t *testing.T, src string, n int) string {
	t.Helper()
	dsl.Seed(1)
	if !strings.Contains(src, "\n") {
		src = "g = " + src
	}
	v := evalLast(t, "seed = 1\n"+src)
	var out []string
	for i := 0; i < n; i++ {
		switch g := v.V.(type) {
		case dsl.Gen[bool]:
			out = append(out, map[bool]string{true: "x", false: "."}[g.Generate()])
		case dsl.Gen[int]:
			out = append(out, fmt.Sprint(g.Generate()))
		case dsl.Gen[float64]:
			out = append(out, fmt.Sprint(g.Generate()))
		case dsl.Gen[[]float64]:
			out = append(out, fmt.Sprint(g.Generate()))
		case dsl.Gen[[]blocks.Note]:
			var ps []string
			for _, nt := range g.Generate() {
				ps = append(ps, fmt.Sprint(nt.Pitch))
			}
			out = append(out, "{"+strings.Join(ps, ",")+"}")
		default:
			t.Fatalf("not a generator: %s", v.T)
		}
	}
	if _, ok := v.V.(dsl.Gen[bool]); ok {
		return strings.Join(out, "")
	}
	return strings.Join(out, " ")
}

func fmtSscan(s string, v *float64) (int, error) { return fmt.Sscan(s, v) }

// listText is the numbers of a list that an expression gives
func listText(t *testing.T, expr string) string {
	t.Helper()
	var out []string
	for _, v := range evalLast(t, "x = "+expr).Items() {
		out = append(out, fmt.Sprint(v.Float()))
	}
	return strings.Join(out, " ")
}
