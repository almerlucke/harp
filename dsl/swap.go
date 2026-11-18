package dsl

import (
	"fmt"
	"strings"
)

// Taking over from a running program (Runner.Swap): the graph of an edited patch is matched with the graph that plays, node by node,
// and what is the same keeps its state: the phase of an oscillator, the memory of a filter, the tail of a reverb, the place of a
// generator in its phrase. Nothing is copied: a node that is the same is not built again, its kernel (with its state in it) is used
// in the new program, with the same buffers.
//
// What counts as the same:
//   - the place in the graph: a node is matched with a node of the old graph that has the same name and the same inputs, matched in
//     their turn (the values of constants are not looked at: a changed cutoff does not make a new filter, and the constant that holds
//     it is written again in its old buffer);
//   - for the state: the same id (what the node is: its key, and the settings that are in it), and, for a node that takes its values
//     from generators, the same fingerprint of them. A node that differs in what it is gets a new kernel (the old state is gone),
//     but it keeps its buffer, so that the nodes after it keep theirs;
//   - nodes that share state (the two outputs of a stereo reverb, the gate and the numbers of a pattern) are kept or not all together;
//   - a pattern of notes whose fields (the generators of its numbers) are not all the same is kept all the same: the generators that changed
//     are put in the old pattern in place of the old ones, and the notes that were made but are not played yet get their new numbers
//     (EventsWithFields). Only the fields; another rhythm, length of a gate or number of voices is another pattern.

// Fingerprinter is implemented by a generator that can tell what it is: the same fingerprint is the same generator, with the same
// settings, so that a generator in the new patch can be the generator of the old one, at its place in its phrase.
type Fingerprinter interface{ Fingerprint() string }

// Fingerprint is the fingerprint of generators together; empty if one of them does not have one
func Fingerprint(gs ...any) string {
	var parts []string
	for _, g := range gs {
		f, ok := g.(Fingerprinter)
		if !ok || f.Fingerprint() == "" {
			return ""
		}
		parts = append(parts, f.Fingerprint())
	}
	return strings.Join(parts, ";")
}

// Rebaser is the state that several nodes share, when it can take over from an old one that differs from it in some of the generators that it
// is made from: RebaseFrom says if it can, and gives what puts the new generators in the old one (it runs on the control goroutine, which is
// the only one that uses them)
type Rebaser interface {
	RebaseFrom(old any) (ok bool, apply func())
}

// entry is a node of a program as it was built
type entry struct {
	n     *node
	sig   string
	kids  []*entry
	buf   []float64
	k     Kernel
	name  string
	lanes []lane
	// matching with the previous program
	grp   any         // the state that its nodes share, as it is alive: that of the old node if it was taken over
	in    [][]float64 // the buffers that its kernel reads (those of its inputs, or, for a kernel that was taken over, the ones it has always read)
	old   *entry      // the entry of the previous program that this one takes over from
	reuse bool        // its kernel (and lanes) are the old ones
}

// identity of the node for its state: empty means it cannot be told, so it is never kept
func (n *node) identity() string {
	if n.op != opOpaque {
		return ""
	}
	id := n.id
	if id == "" {
		id = n.key
	}
	if id == "" || (n.lane && n.gens == "") {
		return ""
	}
	if n.name != "" {
		id = n.name + ":" + id
	}
	if n.post {
		id += fmt.Sprintf(" post %g %g", n.a, n.b)
	}
	return id + "|" + n.gens
}

func label(n *node) string {
	switch n.op {
	case opConst:
		return "c"
	case opAdd:
		return "add"
	case opMul:
		return "mul"
	case opAffine:
		return "affine"
	}
	if n.post {
		return n.name + "+"
	}
	return n.name
}

// compileFrom is compile, taking over what it can from prev (nil: from nothing)
func compileFrom(prev *Program, out Sig, sampleRate, stepSamples float64, optimized bool) *Program {
	p := &Program{}
	if prev != nil {
		p.rt = prev.rt
	} else {
		p.rt = &Rt{SampleRate: sampleRate, StepSamples: stepSamples}
	}
	if len(out.ch) == 0 {
		panic("dsl: a signal without channels cannot be compiled")
	}
	roots := append([]*node(nil), out.ch...)
	if optimized {
		roots, p.Opt = optimize(roots)
	}
	// the nodes in run order, with their signatures
	var order []*entry
	byNode := map[*node]*entry{}
	var visit func(n *node) *entry
	visit = func(n *node) *entry {
		if e, ok := byNode[n]; ok {
			return e
		}
		e := &entry{n: n}
		byNode[n] = e
		var sb strings.Builder
		sb.WriteString(label(n))
		if id := n.identity(); id != "" {
			sb.WriteString("{" + id + "}") // what it is: two generators of patterns are not the same, wherever they are
		}
		sb.WriteByte('(')
		for _, c := range n.in {
			ce := visit(c)
			e.kids = append(e.kids, ce)
			sb.WriteString(ce.sig)
			sb.WriteByte(',')
		}
		sb.WriteByte(')')
		e.sig = sb.String()
		order = append(order, e)
		return e
	}
	var rootEntries []*entry
	for _, r := range roots {
		rootEntries = append(rootEntries, visit(r))
	}
	if prev != nil {
		taken := matchEntries(prev, order)
		softMatch(rootEntries, prev.roots, taken)
	}
	// what is kept. A kernel that is taken over reads the buffers it has always read: where its inputs are other nodes than before, those
	// are copied into them (a copy of a block, before the kernel), unless the buffer is another node's of the new program
	claimed := map[*float64]bool{}
	for _, e := range order {
		if e.old != nil {
			claimed[&e.old.buf[0]] = true
		}
	}
	for _, e := range order {
		e.reuse = e.old != nil && e.n.op == opOpaque && e.n.identity() != "" && e.n.identity() == e.old.n.identity() && bridgeable(e, claimed)
	}
	keepGroupsTogether(p, prev, order)
	// the kernels
	for _, e := range order {
		n := e.n
		switch {
		case n.op == opConst && e.old != nil:
			e.buf = e.old.buf
			v := n.a
			buf := e.buf
			p.fills = append(p.fills, func() {
				for i := range buf {
					buf[i] = v
				}
			})
		default:
			if e.old != nil {
				e.buf = e.old.buf
			} else {
				e.buf = make([]float64, BlockSize)
			}
			if n.op == opConst {
				p.consts++
				for i := range e.buf {
					e.buf[i] = n.a
				}
				break
			}
			ins := make([][]float64, len(e.kids))
			for i, k := range e.kids {
				ins[i] = k.buf
			}
			if e.reuse {
				e.k, e.name, e.lanes, e.in = e.old.k, e.old.name, e.old.lanes, e.old.in
				for i, in := range ins {
					if &in[0] != &e.in[i][0] { // another node feeds it than before: its blocks are copied where the kernel reads
						dst, src := e.in[i], in
						p.kernels = append(p.kernels, func(n int) { copy(dst[:n], src[:n]) })
						p.names = append(p.names, "bridge")
					}
				}
				break
			}
			e.in = ins
			p.rt.sink = &e.lanes
			e.k, e.name = p.build(n, ins, e.buf)
			p.rt.sink = nil
		}
		if e.k != nil {
			p.kernels = append(p.kernels, e.k)
			p.names = append(p.names, e.name)
		}
		p.lanes = append(p.lanes, e.lanes...)
	}
	p.entries = order
	p.roots = rootEntries
	p.groupSize = map[any]int{}
	for _, e := range order {
		e.grp = e.n.group
		if e.reuse {
			e.grp = e.old.grp // the state that is alive is the old one's
		}
		if e.grp != nil {
			p.groupSize[e.grp]++
		}
	}
	for _, r := range rootEntries {
		p.outs = append(p.outs, r.buf)
	}
	return p
}

// matchEntries gives every entry of a new graph the entry of the old one that it takes over from: the first of the same signature that
// is not taken yet and has the same inputs. A constant is matched by what uses it: whichever constant the old node had there
func matchEntries(prev *Program, order []*entry) map[*entry]bool {
	bySig := map[string][]*entry{}
	for _, o := range prev.entries {
		bySig[o.sig] = append(bySig[o.sig], o)
	}
	taken := map[*entry]bool{}
	for _, e := range order {
		if e.n.op == opConst {
			continue // matched by its first user
		}
		for _, o := range bySig[e.sig] {
			if taken[o] || len(o.kids) != len(e.kids) {
				continue
			}
			ok := true
			for i, k := range e.kids {
				switch {
				case k.n.op == opConst:
					// the old input has to be a constant too, and not claimed by another
					if o.kids[i].n.op != opConst || (k.old != nil && k.old != o.kids[i]) || (k.old == nil && taken[o.kids[i]]) {
						ok = false
					}
				default:
					if k.old != o.kids[i] {
						ok = false
					}
				}
			}
			if !ok {
				continue
			}
			taken[o] = true
			e.old = o
			for i, k := range e.kids {
				if k.n.op == opConst && k.old == nil {
					k.old = o.kids[i]
					taken[o.kids[i]] = true
				}
			}
			break
		}
	}
	return taken
}

// softMatch pairs what is left, from the outputs down: where a node of the new graph and a node of the old one are in the same place (the
// same output, or the same input of nodes that are paired) and are the same kind of node, they are paired, whatever is under them. So a
// reverb whose input is another sum (a layer was added or taken away before it) is still that reverb, with its tail, and a filter that is
// given another oscillator is still that filter. Nodes that are matched already (the same, all the way down) stay as they are
func softMatch(newRoots, oldRoots []*entry, taken map[*entry]bool) {
	var pair func(n, o *entry)
	pair = func(n, o *entry) {
		if n.old != nil || taken[o] || label(n.n) != label(o.n) || len(n.kids) != len(o.kids) {
			return
		}
		n.old, taken[o] = o, true
		for i := range n.kids {
			pair(n.kids[i], o.kids[i])
		}
	}
	for i := range newRoots {
		if i < len(oldRoots) {
			pair(newRoots[i], oldRoots[i])
		}
	}
	// the kids of nodes that are matched, but whose kids are not (an exact match has the same kids, so only a node that was paired above has them unpaired)
}

// bridgeable says that the kernel of the old node can be used: the same number of inputs, and where an input is not the buffer that the
// kernel reads, that buffer is not in use by a node of the new program (a copy of the input is put in it)
func bridgeable(e *entry, claimed map[*float64]bool) bool {
	if len(e.kids) != len(e.old.in) {
		return false
	}
	for i, k := range e.kids {
		dst := e.old.in[i]
		if k.old != nil && &dst[0] == &k.old.buf[0] { // the input is the node that it was (its buffer is the old one's)
			continue
		}
		if claimed[&dst[0]] {
			return false
		}
	}
	return true
}

// keepGroupsTogether: the nodes that share state are kept all together or not at all: all of them kept, all from one group of the old
// program, and as many as that group had
func keepGroupsTogether(p *Program, prev *Program, order []*entry) {
	groups := map[any][]*entry{}
	for _, e := range order {
		if e.n.group != nil {
			groups[e.n.group] = append(groups[e.n.group], e)
		}
	}
	for _, members := range groups {
		var oldGroup any
		ok := prev != nil
		for _, e := range members {
			if !ok {
				break
			}
			if !e.reuse || e.old.grp == nil || (oldGroup != nil && e.old.grp != oldGroup) {
				ok = false
				break
			}
			oldGroup = e.old.grp
		}
		if ok && prev.groupSize[oldGroup] != len(members) {
			ok = false
		}
		if rb, isRebaser := members[0].n.group.(Rebaser); ok && isRebaser {
			// the group may differ from the old one in generators that can be put in place of the old ones (a field of a pattern of notes)
			can, apply := rb.RebaseFrom(oldGroup)
			if !can {
				ok = false
			} else if apply != nil {
				p.rebases = append(p.rebases, apply)
			}
		}
		if !ok {
			for _, e := range members {
				e.reuse = false
			}
		}
	}
}

// Taken over says how many of the nodes of the program that has state were taken over from the program before it, and how many are new
// (or changed in themselves, or without anything to tell them by): what a live edit kept and what it started again
func (p *Program) Taken() (kept, fresh int) {
	for _, e := range p.entries {
		if e.n.op != opOpaque {
			continue
		}
		if e.reuse {
			kept++
		} else {
			fresh++
		}
	}
	return
}

// LastSwap is Taken for the program that Swap made last
func (r *Runner) LastSwap() (kept, fresh int) {
	r.swapMu.Lock()
	defer r.swapMu.Unlock()
	return r.latest.Taken()
}

// Channels is the number of output channels of the program
func (p *Program) Channels() int { return len(p.outs) }

// Restarted names the nodes of the program that are not taken over from the one before it: what started again
func (p *Program) Restarted() []string {
	var names []string
	for _, e := range p.entries {
		if e.n.op == opOpaque && !e.reuse {
			names = append(names, e.n.name)
		}
	}
	return names
}

// LastRestarted is Restarted for the program that Swap made last
func (r *Runner) LastRestarted() []string {
	r.swapMu.Lock()
	defer r.swapMu.Unlock()
	return r.latest.Restarted()
}
