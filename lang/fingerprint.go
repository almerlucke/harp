package lang

import (
	"fmt"
	"strings"

	"harp/blocks"
	"harp/dsl"
)

// A running patch that is edited (harp play watches the file) takes over what it can from the old one: a generator that is the same keeps
// its place in its phrase. For that a generator has to say what it is: its fingerprint is the call that made it, with the fingerprints of
// its arguments: degrees(a3, minor, mutate([7 9 11], 0.5)). The same text is the same generator, from the same seed (the numbers drawn when
// the patch is built are in the text).

// fpGen is a generator with a fingerprint
type fpGen[T any] struct {
	dsl.Gen[T]
	fp string
}

func (g *fpGen[T]) Fingerprint() string { return g.fp }

// Fingerprint of intAsFloat is that of the generator it adapts
func (i intAsFloat) Fingerprint() string {
	if f, ok := i.g.(dsl.Fingerprinter); ok {
		return f.Fingerprint()
	}
	return ""
}

func (s scaled) Fingerprint() string {
	if f, ok := s.g.(dsl.Fingerprinter); ok && f.Fingerprint() != "" {
		return fmt.Sprintf("beats(%s;%g)", f.Fingerprint(), s.spb)
	}
	return ""
}

// fpOf is the fingerprint of a value, empty if it cannot be told what it is. A signal is "~": it is not a generator, and the nodes that are
// made from it are matched by where they are
func fpOf(v Value) string {
	switch v.T.Kind {
	case KNum:
		return fmt.Sprintf("%g", v.V.(float64))
	case KBool:
		return fmt.Sprint(v.V.(bool))
	case KStr:
		return fmt.Sprintf("%q", v.V.(string))
	case KSig:
		return "~"
	case KScale:
		return fmt.Sprint(v.V)
	case KList:
		items := v.Items()
		parts := make([]string, len(items))
		for i, it := range items {
			if parts[i] = fpOf(it); parts[i] == "" {
				return ""
			}
		}
		return "[" + strings.Join(parts, " ") + "]"
	case KScore:
		return v.V.(*Score).fingerprint()
	case KSong:
		return v.V.(*Song).fingerprint()
	case KFunc:
		if f, ok := v.V.(*Func); ok {
			return "fn:" + f.Name
		}
		return ""
	}
	if v.T.isGen() {
		if f, ok := v.V.(dsl.Fingerprinter); ok {
			return f.Fingerprint()
		}
	}
	return ""
}

// fingerprinted gives a generator the fingerprint call(args), if all of the arguments have one
func fingerprinted(out Value, call string, args []Value) Value {
	if !out.T.isGen() {
		return out
	}
	if f, ok := out.V.(dsl.Fingerprinter); ok && f.Fingerprint() != "" {
		return out
	}
	parts := make([]string, len(args))
	for i, a := range args {
		if parts[i] = fpOf(a); parts[i] == "" {
			return out
		}
	}
	return wrapFP(out, call+"("+strings.Join(parts, ", ")+")")
}

// wrapFP puts the fingerprint on the generator of the value
func wrapFP(v Value, fp string) Value {
	switch g := v.V.(type) {
	case dsl.Gen[float64]:
		v.V = &fpGen[float64]{g, fp}
	case dsl.Gen[int]:
		v.V = &fpGen[int]{g, fp}
	case dsl.Gen[bool]:
		v.V = &fpGen[bool]{g, fp}
	case dsl.Gen[[]float64]:
		v.V = &fpGen[[]float64]{g, fp}
	case dsl.Gen[[]blocks.Note]:
		v.V = &fpGen[[]blocks.Note]{g, fp}
	case dsl.Gen[[]dsl.Hit]:
		v.V = &fpGen[[]dsl.Hit]{g, fp}
	}
	return v
}

// fpEvents is the generator of the notes of a play, with its fields and how long its gates are held that can be put in place
type fpEvents struct {
	dsl.Gen[dsl.Event]
}

func (g *fpEvents) Hold() func(float64) float64 {
	if t, ok := g.Gen.(dsl.TimingGen); ok {
		return t.Hold()
	}
	return nil
}

func (g *fpEvents) SetHold(f func(float64) float64) {
	if t, ok := g.Gen.(dsl.TimingGen); ok {
		t.SetHold(f)
	}
}

func (g *fpEvents) Fields() []dsl.Gen[float64] {
	if f, ok := g.Gen.(dsl.FieldSetter); ok {
		return f.Fields()
	}
	return nil
}

func (g *fpEvents) SetField(i int, f dsl.Gen[float64]) {
	if s, ok := g.Gen.(dsl.FieldSetter); ok {
		s.SetField(i, f)
	}
}
