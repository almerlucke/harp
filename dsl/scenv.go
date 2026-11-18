package dsl

import (
	"fmt"
	"math"
)

// SuperCollider's EnvGen, as LFUGens.cpp (3.14.1) works it out: segments with a shape each, a gate that is read once in every block
// (at the audio rate output) or a value for every block (control rate), a release node, and the leftover of a segment that is not
// a whole number of samples carried to the next.

// the shapes of a segment, as the numbers of SuperCollider
const (
	EnvStep = iota
	EnvLinear
	EnvExp
	EnvSine
	EnvWelch
	EnvCurve
	EnvSquared
	EnvCubed
	EnvHold
	envSustain
)

// EnvSeg is a segment: it goes to Level in Dur seconds, in a Shape (EnvCurve: Curve says how much it bends, 0 is straight,
// negative bends it so that it moves fast first)
type EnvSeg struct {
	Level, Dur float64
	Shape      int
	Curve      float64
}

// EnvSpec is an envelope: where it starts, its segments, the segment after which it waits for the gate to fall (-1: none), and
// the segment it goes back to there (-1: none)
type EnvSpec struct {
	Start       float64
	Segs        []EnvSeg
	ReleaseNode int
	LoopNode    int
}

// EnvPerc is Env.perc(attack, release, level, curve): up in attack seconds, and down in release seconds
func EnvPerc(attack, release, level, curve float64) EnvSpec {
	return EnvSpec{Segs: []EnvSeg{{level, attack, EnvCurve, curve}, {0, release, EnvCurve, curve}}, ReleaseNode: -1, LoopNode: -1}
}

// EnvASR is Env.asr(attack, sustain, release, curve): up in attack seconds, stays while the gate is high, and down in release seconds
func EnvASR(attack, sustain, release, curve float64) EnvSpec {
	return EnvSpec{Segs: []EnvSeg{{sustain, attack, EnvCurve, curve}, {0, release, EnvCurve, curve}}, ReleaseNode: 1, LoopNode: -1}
}

// EnvGen plays an envelope: it starts when the gate rises above 0 and, with a release node, goes on from there when the gate falls
// to 0 or below. control makes it SuperCollider's .kr: one value for every block of 64 samples. The gate is read once in a block
func EnvGen[G Param](gate G, spec EnvSpec, control bool) Sig {
	return Custom(Spec{Name: "EnvGen", Key: fmt.Sprintf("envgen %v %v", control, spec), Inputs: []Sig{Signal(gate)}, New: func(rt *Rt) Process {
		e := &envGen{rt: rt, spec: spec, control: control, level: spec.Start, endLevel: spec.Start, stage: envNotStarted, shape: EnvHold}
		return e.process
	}})
}

const envNotStarted = 1000000000

type envGen struct {
	rt      *Rt
	spec    EnvSpec
	control bool

	level, endLevel float64
	counter         int
	stage           int
	shape           int
	prevGate        float64
	released, done  bool
	residual        float64
	grow, a2, b1    float64
	y1, y2          float64
	held            float64
	gate            float64
}

func (e *envGen) rate() float64 {
	if e.control {
		return e.rt.SampleRate / scBlock
	}
	return e.rt.SampleRate
}

func (e *envGen) checkGate(gate float64) {
	switch {
	case e.prevGate <= 0 && gate > 0:
		e.stage, e.released, e.done, e.counter, e.residual = -1, false, false, 0, 0
	case gate <= -1 && e.prevGate > -1:
		dur := -gate - 1
		e.counter = max(1, int(dur*e.rate()))
		e.stage = len(e.spec.Segs) - 1
		e.released = true
		e.initSegment(dur)
		e.residual = 0
	case e.prevGate > 0 && gate <= 0 && e.spec.ReleaseNode >= 0 && !e.released:
		e.counter, e.stage, e.released, e.residual = 0, e.spec.ReleaseNode-1, true, 0
	}
	e.prevGate = gate
}

func (e *envGen) initSegment(dur float64) {
	prevEnd := e.endLevel
	if e.shape == EnvHold {
		e.level = prevEnd
	}
	seg := e.spec.Segs[e.stage]
	if dur < 0 {
		dur = seg.Dur
	}
	e.shape = seg.Shape
	curve := seg.Curve
	end := seg.Level
	e.endLevel = end
	stageDur := dur*e.rate() + e.residual
	floor := int(stageDur)
	fractional := math.Max(1, stageDur)
	e.counter = max(1, floor)
	e.residual = stageDur - float64(e.counter)
	if e.counter == 1 {
		e.shape = EnvLinear
	}
	level := e.level
	switch e.shape {
	case EnvStep:
		e.level = end
	case EnvHold:
		e.level = prevEnd
	case EnvLinear:
		e.grow = (end - level) / fractional
	case EnvExp:
		e.grow = math.Pow(end/level, 1/fractional)
	case EnvSine:
		w := math.Pi / fractional
		e.a2 = (end + level) * 0.5
		e.b1 = 2 * math.Cos(w)
		e.y1 = (end - level) * 0.5
		e.y2 = e.y1 * math.Sin(math.Pi*0.5-w)
		e.level = e.a2 - e.y1
	case EnvWelch:
		w := (math.Pi * 0.5) / fractional
		e.b1 = 2 * math.Cos(w)
		if end >= level {
			e.a2, e.y1, e.y2 = level, 0, -math.Sin(w)*(end-level)
		} else {
			e.a2, e.y1, e.y2 = end, level-end, math.Cos(w)*(level-end)
		}
		e.level = e.a2 + e.y1
	case EnvCurve:
		if math.Abs(curve) < 0.001 {
			e.shape = EnvLinear
			e.grow = (end - level) / fractional
		} else {
			a1 := (end - level) / (1 - math.Exp(curve))
			e.a2 = level + a1
			e.b1 = a1
			e.grow = math.Exp(curve / fractional)
		}
	case EnvSquared:
		e.y1, e.y2 = math.Sqrt(level), math.Sqrt(end)
		e.grow = (e.y2 - e.y1) / fractional
	case EnvCubed:
		e.y1, e.y2 = math.Cbrt(level), math.Cbrt(end)
		e.grow = (e.y2 - e.y1) / fractional
	}
}

func (e *envGen) nextSegment() {
	n := len(e.spec.Segs)
	switch {
	case e.stage+1 >= n && e.stage != envNotStarted: // the last stage is over
		e.counter, e.shape, e.done = math.MaxInt32, EnvStep, true
	case e.stage == envNotStarted:
		e.counter = math.MaxInt32
	case e.stage+1 == e.spec.ReleaseNode && !e.released:
		if ln := e.spec.LoopNode; ln >= 0 && ln < n {
			e.stage = ln
			e.initSegment(-1)
		} else {
			e.counter, e.shape, e.level = math.MaxInt32, envSustain, e.endLevel
		}
	default:
		e.stage++
		e.initSegment(-1)
	}
}

// perform gives the level and moves it on, by the shape
func (e *envGen) perform() float64 {
	out := e.level
	switch e.shape {
	case EnvLinear:
		e.level += e.grow
	case EnvExp:
		e.level *= e.grow
	case EnvSine:
		y0 := e.b1*e.y1 - e.y2
		e.level = e.a2 - y0
		e.y2, e.y1 = e.y1, y0
	case EnvWelch:
		y0 := e.b1*e.y1 - e.y2
		e.level = e.a2 + y0
		e.y2, e.y1 = e.y1, y0
	case EnvCurve:
		e.b1 *= e.grow
		e.level = e.a2 - e.b1
	case EnvSquared:
		e.y1 += e.grow
		e.level = e.y1 * e.y1
	case EnvCubed:
		e.y1 = math.Max(e.y1+e.grow, 0)
		e.level = e.y1 * e.y1 * e.y1
	}
	return out
}

// step is one sample (one step at the control rate)
func (e *envGen) step() float64 {
	if e.counter <= 0 {
		e.nextSegment()
		if e.done && !e.control {
			// the end of the last stage: the level that was ahead is given, and after it the end level
			out := e.level
			e.level = e.endLevel
			return out
		}
	}
	out := e.perform()
	if e.done {
		e.level = e.endLevel
	}
	e.counter--
	return out
}

func (e *envGen) process(in [][]float64, out []float64, n int) {
	pos := e.rt.Pos()
	for i := 0; i < n; i++ {
		if (pos+int64(i))%scBlock == 0 {
			e.checkGate(in[0][i])
			if e.control {
				e.held = e.step()
			}
		}
		if e.control {
			out[i] = e.held
		} else {
			out[i] = float64(float32(e.step()))
		}
	}
}
