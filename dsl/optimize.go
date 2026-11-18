package dsl

import (
	"fmt"
	"math"
	"strings"
)

// The optimizer rewrites the graph before it is compiled. It never changes a node: it builds a
// new graph, so the Sig you made stays what it was.
//
// Pass 1, bottom up:
//   - constants: an operation on constants is a constant, and a constant is no kernel at all (its
//     buffer is filled once)
//   - identities: x*1 and x+0 are x
//   - affines: a constant in a mul or add makes it an affine (x*a+b), and an affine of an affine is
//     one affine: ((x*a1+b1)*a2+b2) = x*(a1*a2) + (b1*a2+b2)
//   - merging: nodes that compute the same from the same inputs are one node. A node that takes
//     its values from a generator is never merged
//
// Pass 2, with the number of users of every node: an affine that follows a node nobody else uses
// is moved into that node's loop (add, mul, sine, saw): one pass over the block instead of two
//
// Folding affines changes the order of the floating point operations, so the sound is the same
// but not always the same to the last bit. Patch.NoOptimize turns it all off.

// Report says what the optimizer did
type Report struct {
	NodesBefore, NodesAfter int
	Folded                  int // operations on constants that became a constant
	Identities              int // x*1 and x+0 that were taken out
	Affines                 int // a mul or add by a constant that became an affine, or two affines that became one
	Merged                  int // nodes that were the same as one already there
	Absorbed                int // affines moved into the loop of the node before them
}

func (r Report) String() string {
	if r == (Report{}) {
		return "optimizer off\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "nodes %d -> %d\n", r.NodesBefore, r.NodesAfter)
	for _, l := range []struct {
		n    int
		what string
	}{
		{r.Folded, "operations on constants folded"},
		{r.Identities, "identities taken out"},
		{r.Affines, "mul/add by a constant or affine of affine folded to an affine"},
		{r.Merged, "duplicate nodes merged"},
		{r.Absorbed, "affines moved into the loop before them"},
	} {
		if l.n > 0 {
			fmt.Fprintf(&b, "  %d %s\n", l.n, l.what)
		}
	}
	return b.String()
}

type optimizer struct {
	memo map[*node]*node // old node to its rewrite
	cse  map[string]*node
	ids  map[*node]int
	rep  Report
}

func optimize(roots []*node) ([]*node, Report) {
	o := &optimizer{memo: map[*node]*node{}, cse: map[string]*node{}, ids: map[*node]int{}}
	before := count(roots)
	r1 := make([]*node, len(roots))
	for i, r := range roots {
		r1[i] = o.rewrite(r)
	}

	uses := map[*node]int{}
	seen := map[*node]bool{}
	var walk func(n *node)
	walk = func(n *node) {
		if seen[n] {
			return
		}
		seen[n] = true
		for _, c := range n.in {
			uses[c]++
			walk(c)
		}
	}
	for _, r := range r1 {
		uses[r]++ // every output is a user too
		walk(r)
	}
	ab := &absorber{uses: uses, memo: map[*node]*node{}, rep: &o.rep}
	r2 := make([]*node, len(r1))
	for i, r := range r1 {
		r2[i] = ab.run(r)
	}

	o.rep.NodesBefore, o.rep.NodesAfter = before, count(r2)
	return r2, o.rep
}

func count(roots []*node) int {
	seen := map[*node]bool{}
	var walk func(n *node)
	walk = func(n *node) {
		if seen[n] {
			return
		}
		seen[n] = true
		for _, c := range n.in {
			walk(c)
		}
	}
	for _, r := range roots {
		walk(r)
	}
	return len(seen)
}

// add gives a new node its number, and returns the node that was there already if there is one
// with the same key (key "": never merged)
func (o *optimizer) add(n *node, key string) *node {
	if key != "" {
		if m, ok := o.cse[key]; ok {
			o.rep.Merged++
			return m
		}
		o.cse[key] = n
	}
	o.ids[n] = len(o.ids)
	return n
}

func (o *optimizer) constant(v float64) *node {
	return o.add(&node{name: fmt.Sprintf("const(%g)", v), op: opConst, a: v},
		fmt.Sprintf("const %x", math.Float64bits(v)))
}

func (o *optimizer) rewrite(n *node) *node {
	if m, ok := o.memo[n]; ok {
		return m
	}
	ins := make([]*node, len(n.in))
	same := true
	for i, c := range n.in {
		ins[i] = o.rewrite(c)
		same = same && ins[i] == c
	}
	var out *node
	switch n.op {
	case opConst:
		out = o.constant(n.a)
	case opAdd, opMul:
		out = o.binary(n.op, ins[0], ins[1])
	case opAffine:
		out = o.affine(ins[0], n.a, n.b)
	default:
		key := ""
		if n.key != "" {
			key = o.opaqueKey(n.key, ins)
		}
		if same { // nothing under it changed: it is what it was
			out = o.add(n, key)
		} else {
			cp := *n
			cp.in = ins
			out = o.add(&cp, key)
		}
	}
	o.memo[n] = out
	return out
}

func (o *optimizer) opaqueKey(key string, ins []*node) string {
	var b strings.Builder
	b.WriteString("op " + key)
	for _, c := range ins {
		fmt.Fprintf(&b, " %d", o.ids[c])
	}
	return b.String()
}

func (o *optimizer) binary(op opKind, x, y *node) *node {
	xc, yc := x.op == opConst, y.op == opConst
	switch {
	case xc && yc:
		o.rep.Folded++
		if op == opMul {
			return o.constant(x.a * y.a)
		}
		return o.constant(x.a + y.a)
	case xc || yc:
		c, other := x.a, y
		if yc {
			c, other = y.a, x
		}
		if op == opMul {
			if c != 1 {
				o.rep.Affines++
			}
			return o.affine(other, c, 0)
		}
		if c != 0 {
			o.rep.Affines++
		}
		return o.affine(other, 1, c)
	}
	name, kind := "mul", "mul"
	if op == opAdd {
		name, kind = "add", "add"
	}
	i, j := o.ids[x], o.ids[y]
	if i > j { // x*y is y*x, bit for bit
		i, j = j, i
	}
	return o.add(&node{name: name, op: op, in: []*node{x, y}}, fmt.Sprintf("%s %d %d", kind, i, j))
}

func (o *optimizer) affine(x *node, a, b float64) *node {
	switch {
	case x.op == opConst:
		o.rep.Folded++
		return o.constant(x.a*a + b)
	case a == 1 && b == 0:
		o.rep.Identities++
		return x
	case x.op == opAffine:
		o.rep.Affines++
		return o.affine(x.in[0], a*x.a, a*x.b+b)
	}
	return o.add(&node{name: "affine", op: opAffine, in: []*node{x}, a: a, b: b},
		fmt.Sprintf("affine %d %x %x", o.ids[x], math.Float64bits(a), math.Float64bits(b)))
}

type absorber struct {
	uses map[*node]int
	memo map[*node]*node
	rep  *Report
}

func (p *absorber) run(n *node) *node {
	if m, ok := p.memo[n]; ok {
		return m
	}
	ins := make([]*node, len(n.in))
	same := true
	for i, c := range n.in {
		ins[i] = p.run(c)
		same = same && ins[i] == c
	}
	out := n
	if !same {
		cp := *n
		cp.in = ins
		out = &cp
	}
	if n.op == opAffine {
		if prod := n.in[0]; p.uses[prod] == 1 && canPost(ins[0]) {
			m := *ins[0]
			if m.post { // an affine after an affine: one
				m.a, m.b = m.a*n.a, m.b*n.a+n.b
			} else {
				m.post, m.a, m.b = true, n.a, n.b
			}
			out = &m
			p.rep.Absorbed++
		}
	}
	p.memo[n] = out
	return out
}

// canPost says if a node can scale and offset its own output while it computes it
func canPost(n *node) bool {
	switch n.op {
	case opAdd, opMul:
		return true
	case opOpaque:
		return n.buildPost != nil
	}
	return false
}
