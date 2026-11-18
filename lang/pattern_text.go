package lang

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/almerlucke/psalm/tools/rhythm"
)

// The text of a pattern, as psalm's clock reads it, and of a part (notes instead of hits). A step each:
//
//	x (X, o, 1) a hit, . (_, 0, and - in a pattern) a rest      x3 a roll: three hits in the step
//	[x x x] a group that fills one step (a triplet)             *2 after a step or a group: it lasts 2 steps
//	?25 after a hit: it plays by chance, 25% (? is 50%)         %2/4 %!2/4 %fill %first: only on some passes through the pattern
//	! after a hit: it stays when the pattern mutates            spaces, | and new lines are for reading
//
// A part has notes where a pattern has hits: C3 Eb3 G3, C3+Eb3+G3 a chord, - the note before held a step longer, ^3 after a note a roll.

// textHit is a hit that is read: where it starts in the pattern (steps), how long it lasts (0: until the next), its chance, the passes it plays on,
// whether it stays when the pattern mutates, and, in a part, the note (or notes: C3+E3) it plays
type textHit struct {
	at, len float64
	slot    float64 // the width of the step (or part of one) it is in
	chance  float64
	cond    rhythm.Cond
	pin     bool
	tok     string
	holdEnd float64 // in a part: the end of the time that the note is held (a - after it)
}

// elem is a step or a group that is read
type elem struct {
	kind    byte // 'h' hit, 'r' rest, 'g' group, 'l' hold
	tok     string
	roll    int
	kids    []elem
	weight  float64
	hasLen  bool
	chance  float64
	cond    rhythm.Cond
	pin     bool
	atLevel bool
}

type textParser struct {
	s    string
	i    int
	part bool
}

func (p *textParser) skip() {
	for p.i < len(p.s) && strings.IndexByte(" \t\r\n|", p.s[p.i]) >= 0 {
		p.i++
	}
}

// parseTextPattern reads the text of a pattern (part false) or a part, and gives its hits and how many steps it has
func parseTextPattern(text string, part bool) ([]textHit, float64, error) {
	p := &textParser{s: text, part: part}
	elems, err := p.sequence(0)
	if err != nil {
		return nil, 0, err
	}
	var hits []textHit
	steps := 0.0
	last := -1 // the hit before, for a hold
	for i := range elems {
		w := elems[i].weight
		flat(&elems[i], steps, w, true, &hits, &last)
		steps += w
	}
	if steps <= 0 {
		return nil, 0, fmt.Errorf("the pattern has no steps")
	}
	for i := range hits {
		if hits[i].holdEnd > 0 && hits[i].len == 0 {
			hits[i].len = hits[i].holdEnd - hits[i].at
		}
	}
	return hits, steps, nil
}

// sequence reads elements up to the closing bracket (or the end)
func (p *textParser) sequence(depth int) ([]elem, error) {
	var out []elem
	for {
		p.skip()
		if p.i >= len(p.s) {
			if depth > 0 {
				return nil, fmt.Errorf("a [ without its ]")
			}
			return out, nil
		}
		c := p.s[p.i]
		if c == ']' {
			if depth == 0 {
				return nil, fmt.Errorf("a ] without its [")
			}
			p.i++
			return out, nil
		}
		e, err := p.element(depth)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
}

func (p *textParser) element(depth int) (elem, error) {
	e := elem{weight: 1, chance: 1}
	c := p.s[p.i]
	switch {
	case c == '[':
		p.i++
		kids, err := p.sequence(depth + 1)
		if err != nil {
			return e, err
		}
		if len(kids) == 0 {
			return e, fmt.Errorf("an empty group []")
		}
		e.kind, e.kids = 'g', kids
	case p.part:
		start := p.i
		for p.i < len(p.s) && strings.IndexByte(" \t\r\n|[]^*?%!", p.s[p.i]) < 0 {
			p.i++
		}
		e.tok = p.s[start:p.i]
		switch e.tok {
		case "":
			return e, fmt.Errorf("unexpected %q", string(c))
		case ".":
			e.kind = 'r'
		case "-":
			e.kind = 'l'
		default:
			e.kind = 'h'
		}
	case strings.IndexByte("xXo1", c) >= 0:
		p.i++
		e.kind, e.tok = 'h', string(c)
		if c != '1' { // 1 is a single step: 1010 is four steps, never a roll
			start := p.i
			for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
				p.i++
			}
			if p.i > start {
				n, _ := strconv.Atoi(p.s[start:p.i])
				if n < 1 || n > 64 {
					return e, fmt.Errorf("a roll is from 1 to 64 hits (x%d)", n)
				}
				e.roll = n
			}
		}
	case strings.IndexByte("._-0", c) >= 0:
		p.i++
		e.kind = 'r'
	default:
		return e, fmt.Errorf("unexpected %q: a step is x (a hit) or . (a rest), [x x x] a group", string(c))
	}
	// what follows a step or a group: ^3 a roll (a part), *2 how long, ?25 a chance, %2/4 a condition, ! stays
	for p.i < len(p.s) {
		switch p.s[p.i] {
		case '^':
			p.i++
			n, err := p.number()
			if err != nil || n < 1 || n > 64 || !isWhole(n) {
				return e, fmt.Errorf("^ is followed by the number of hits in the step, 1 to 64")
			}
			e.roll = int(n)
		case '*':
			p.i++
			n, err := p.number()
			if err != nil || n <= 0 || n > 4096 {
				return e, fmt.Errorf("* is followed by how many steps it lasts, above 0")
			}
			e.weight, e.hasLen = n, true
		case '?':
			p.i++
			e.chance = 0.5
			if n, err := p.number(); err == nil {
				if n < 0 || n > 100 {
					return e, fmt.Errorf("? is followed by a chance from 0 to 100 percent")
				}
				e.chance = n / 100
			}
		case '%':
			p.i++
			start := p.i
			if p.i < len(p.s) && p.s[p.i] == '!' {
				p.i++
			}
			for p.i < len(p.s) && (p.s[p.i] >= 'a' && p.s[p.i] <= 'z' || p.s[p.i] >= '0' && p.s[p.i] <= '9' || p.s[p.i] == '/') {
				p.i++
			}
			cond, err := rhythm.ParseCond(p.s[start:p.i])
			if err != nil {
				return e, err
			}
			e.cond = cond
		case '!':
			p.i++
			e.pin = true
		default:
			return e, nil
		}
	}
	return e, nil
}

func (p *textParser) number() (float64, error) {
	start := p.i
	for p.i < len(p.s) && (p.s[p.i] >= '0' && p.s[p.i] <= '9' || p.s[p.i] == '.') {
		p.i++
	}
	return strconv.ParseFloat(p.s[start:p.i], 64)
}

// flat puts the hits of an element in its slot of the pattern: a step (or as many as it lasts) at top level, a share of its group inside one
func flat(e *elem, start, width float64, top bool, out *[]textHit, last *int) {
	switch e.kind {
	case 'r':
		return
	case 'l': // a hold: the note before goes on to the end of this step
		if *last >= 0 {
			(*out)[*last].holdEnd = start + width
		}
		return
	case 'g':
		total := 0.0
		for i := range e.kids {
			total += e.kids[i].weight
		}
		at := start
		for i := range e.kids {
			w := width * e.kids[i].weight / total
			flat(&e.kids[i], at, w, false, out, last)
			at += w
		}
		// the group's own chance, condition and pin hold for the hits in it
		for j := len(*out) - 1; j >= 0 && (*out)[j].at >= start; j-- {
			(*out)[j].chance *= e.chance
			if !e.cond.Zero() {
				(*out)[j].cond = e.cond
			}
			(*out)[j].pin = (*out)[j].pin || e.pin
		}
		return
	}
	n := 1
	if e.roll > 1 {
		n = e.roll
	}
	for k := 0; k < n; k++ {
		h := textHit{at: start + float64(k)*width/float64(n), slot: width / float64(n), chance: e.chance, cond: e.cond, pin: e.pin, tok: e.tok}
		if e.hasLen && top && n == 1 {
			h.len = width
		}
		*out = append(*out, h)
		*last = len(*out) - 1
	}
}
