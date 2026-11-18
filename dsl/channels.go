package dsl

import (
	"fmt"
	"math"
)

// Channels. A Sig has one channel (mono), two (stereo) or more. The rules, in one place:
//
//   - Nodes expand: a node that takes signals is made for every channel (see width in nodes.go).
//     One channel goes to all channels of the others; otherwise the counts must be equal, and a
//     patch that has 2 and 3 panics when it is built.
//   - Where a fixed number of channels is wanted (a device, a node with a shape of its own), As
//     connects the way psalm's sockets do: mono into stereo is copied to both sides, stereo into
//     mono is the average of the two.
//   - Pan, Spread, Sum, Fold and Mono are the ways to go from one number of channels to another,
//     on purpose.

// Chans puts the channels of the signals one after the other: Chans(a, b) is a stereo signal of two
// mono signals
func Chans(sigs ...Sig) Sig {
	var out Sig
	for _, s := range sigs {
		if len(s.ch) == 0 {
			panic("dsl: Chans: a signal without channels (a Sig that was never made)")
		}
		out.ch = append(out.ch, s.ch...)
	}
	if len(out.ch) == 0 {
		panic("dsl: Chans: nothing to put together")
	}
	return out
}

// Stereo makes a stereo signal of a left and a right mono signal
func Stereo(left, right Sig) Sig {
	if len(left.ch) != 1 || len(right.ch) != 1 {
		panic(fmt.Sprintf("dsl: Stereo: wants two mono signals, got %d and %d channels", len(left.ch), len(right.ch)))
	}
	return Chans(left, right)
}

// Dup gives a mono signal n channels, all the same (the signal at full level on each)
func Dup(s Sig, n int) Sig {
	if len(s.ch) != 1 || n < 1 {
		panic(fmt.Sprintf("dsl: Dup(%d channels, %d): wants a mono signal and n >= 1", len(s.ch), n))
	}
	out := Sig{ch: make1(n)}
	for i := range out.ch {
		out.ch[i] = s.ch[0]
	}
	return out
}

// Sum adds all channels into one: a mono signal
func Sum(s Sig) Sig {
	if len(s.ch) == 0 {
		panic("dsl: Sum: a signal without channels (a Sig that was never made)")
	}
	m := s.Ch(0)
	for i := 1; i < len(s.ch); i++ {
		m = Add(m, s.Ch(i))
	}
	return m
}

// Mono mixes down to one channel with the average of the channels, as psalm does connecting
// stereo to mono
func Mono(s Sig) Sig {
	if len(s.ch) == 1 {
		return s
	}
	return Mul(Sum(s), 1/float64(len(s.ch)))
}

// As connects a signal to n channels, as psalm's sockets do: mono into n channels is copied to
// all of them, n channels into mono is their average, an equal number is as it is. Other counts
// have no rule (what should 3 channels into 2 do?): use Spread or Fold, or choose the channels
func As(s Sig, n int) Sig {
	switch c := len(s.ch); {
	case c == n:
		return s
	case c == 1:
		return Dup(s, n)
	case n == 1:
		return Mono(s)
	default:
		panic(fmt.Sprintf("dsl: As: cannot connect %d channels to %d: use Spread, Fold or Ch", c, n))
	}
}

// Fold mixes down to n channels: channel i is added to channel i mod n. After Pan, whose channels
// are left, right, left, right..., Fold(x, 2) is the stereo mix
func Fold(s Sig, n int) Sig {
	if n < 1 || len(s.ch) < n {
		panic(fmt.Sprintf("dsl: Fold(%d channels, %d): wants 1 <= n <= channels", len(s.ch), n))
	}
	out := make([]Sig, n)
	for i := range out {
		out[i] = s.Ch(i)
	}
	for i := n; i < len(s.ch); i++ {
		out[i%n] = Add(out[i%n], s.Ch(i))
	}
	return Chans(out...)
}

// Mix adds signals together, channel by channel (see Add). No signals is silence
func Mix(sigs ...Sig) Sig {
	if len(sigs) == 0 {
		return N(0)
	}
	m := sigs[0]
	for _, s := range sigs[1:] {
		m = Add(m, s)
	}
	return m
}

// Gain scales a signal, a number or a signal, every channel the same
func Gain[G Param](s Sig, g G) Sig { return Mul(s, g) }

// Pan puts mono signals in the stereo field: pos is -1 (left) to 1 (right), 0 in the middle, as a
// number or a signal. Every channel of in is panned (at the position of the same channel of pos,
// or at pos if that is mono), and the result has the channels of every one after the other: left,
// right, left, right... (Fold(x, 2) mixes them). The law is equal power, so the loudness stays the
// same wherever it is:
//
//	left = in * cos((pos+1) * pi/4)     right = in * sin((pos+1) * pi/4)
//
// In the middle both sides are 0.707 (-3 dB). A position that is a number costs nothing per
// sample: it is two constant gains, which the optimizer folds into whatever comes before. A
// position that is a signal computes its gains every sample
func Pan[P Param](in Sig, pos P) Sig {
	p := sig(pos)
	n := width("Pan", in, p)
	var out []*node
	for i := 0; i < n; i++ {
		x, pi := in.at(i), p.at(i)
		var l, r Sig
		if c := pi.ch[0]; c.op == opConst {
			gl, gr := panGains(c.a)
			l, r = Mul(x, gl), Mul(x, gr)
		} else {
			l, r = Mul(x, panGain("pan.left", pi, math.Cos)), Mul(x, panGain("pan.right", pi, math.Sin))
		}
		out = append(out, l.ch[0], r.ch[0])
	}
	return Sig{ch: out}
}

func panGain(name string, pos Sig, f func(theta float64) float64) Sig {
	return unary(name, name, pos, func(_ *Rt, pos, out []float64) Kernel {
		return func(n int) {
			for i := 0; i < n; i++ {
				out[i] = f(panAngle(pos[i]))
			}
		}
	})
}

func panAngle(pos float64) float64 { return (math.Min(math.Max(pos, -1), 1) + 1) * math.Pi / 4 }

func panGains(pos float64) (l, r float64) {
	theta := panAngle(pos)
	return math.Cos(theta), math.Sin(theta)
}

// Spread puts the channels of a signal in the stereo field, evenly from -width (the first) to width
// (the last), 0..1 where 1 is the whole field. One channel is in the middle. The channels are
// scaled by 1/sqrt(n), so that a number of independent sounds is as loud as one (what SuperCollider's
// Splay does)
func Spread(s Sig, width float64) Sig {
	n := len(s.ch)
	if n == 0 {
		panic("dsl: Spread: a signal without channels (a Sig that was never made)")
	}
	panned := make([]Sig, n)
	for i := range panned {
		pos := 0.0
		if n > 1 {
			pos = width * (2*float64(i)/float64(n-1) - 1)
		}
		panned[i] = Pan(s.Ch(i), pos)
	}
	return Mul(Fold(Chans(panned...), 2), 1/math.Sqrt(float64(n)))
}
