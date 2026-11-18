package dsl

import "math"

// SCSaw is SuperCollider's Saw, worked out the way SuperCollider works it out (OscUGens.cpp, Saw_next, 3.14.1): a band limited
// pulse train, the sum of 2N+1 harmonics found as sin((2N+1)x)/sin(x) from a table, that a leaky integrator (0.999) makes a saw.
//
// It is not the same as Saw (PolyBLEP): it is half the size and it falls, and what matters, the frequency is read once in every
// block of 64 samples, and a change of it that changes N (the number of harmonics that fit below half the sample rate) is a
// crossfade between the two sums over that block, and the integrator keeps what it had: a patch that moves the frequency fast
// (an exponential sweep over 6 to 11000 Hz) makes it ring with peaks that are many times the size of the saw. Use it to mimic a
// patch of SuperCollider, and Saw for a saw that is the same under every modulation
func SCSaw[F Param](freq F) Sig {
	return Custom(Spec{Name: "SCSaw", Key: "scsaw", Inputs: []Sig{Signal(freq)}, New: func(rt *Rt) Process {
		return newSCSaw(rt).process
	}})
}

// SCSawSync is SCSaw that starts again, from phase 0 and the start of the integrator, when the sync signal rises above 0: as a new
// synth of SuperCollider does for every note. The sync is read once in a block, as everything is there
func SCSawSync[F Param](freq F, sync Sig) Sig {
	return Custom(Spec{Name: "SCSawSync", Key: "scsaw.sync", Inputs: []Sig{Signal(freq), sync}, New: func(rt *Rt) Process {
		s := newSCSaw(rt)
		var prev float64
		return func(in [][]float64, out []float64, n int) {
			pos := rt.Pos()
			for i := 0; i < n; i++ {
				if (pos+int64(i))%scBlock == 0 {
					if g := in[1][i]; prev <= 0 && g > 0 {
						s.reset(float32(in[0][i]))
					}
					prev = in[1][i]
				}
				s.processOne(in[0][i:i+1], out[i:i+1], pos+int64(i))
			}
		}
	}})
}

const (
	scSineSize = 8192
	scBad      = float32(1e20)
	scBlock    = 64 // SuperCollider's block size: the frequency is read once in each
)

// the tables of SuperCollider (common/Samp.cpp): a sine, and its inverse with the bad values near the zeros
var scSine, scInvSine = func() (s, inv [scSineSize + 1]float32) {
	for i := 0; i <= scSineSize; i++ {
		d := math.Sin(float64(i) * 2 * math.Pi / scSineSize)
		s[i] = float32(d)
		inv[i] = float32(1 / d)
	}
	inv[0], inv[scSineSize/2], inv[scSineSize] = scBad, scBad, scBad
	for i := 1; i <= 8; i++ {
		inv[i], inv[scSineSize-i] = scBad, scBad
		inv[scSineSize/2-i], inv[scSineSize/2+i] = scBad, scBad
	}
	return
}()

type scSaw struct {
	rt       *Rt
	started  bool
	phase    int32
	n        int32
	freqin   float32
	scale    float32
	y1       float32
	cpstoinc float64
	// what holds for the block that is going on
	freq             int32
	crossfade        bool
	prevN2, nn2      int32
	prevScale, cur   float32
	xfade, xfadeStep float32
}

func newSCSaw(rt *Rt) *scSaw {
	return &scSaw{rt: rt, cpstoinc: scSineSize * (1 / rt.SampleRate) * 65536 * 0.5}
}

// phaseFrac is PhaseFrac of SC_SndBuf.h: the fraction between two entries of a table
func scPhaseFrac(phase uint32) float32 {
	return math.Float32frombits(0x3F800000|(0x007FFF80&(phase<<7))) - 1
}

// at reads the table at the phase, as (phase >> xlobits) & xlomask13 does: two entries and the fraction between them
func scAt(tbl *[scSineSize + 1]float32, phase int32) (t0, t1 float32) {
	i := (uint32(phase) >> 16) & 0x1FFF
	return tbl[i], tbl[i+1]
}

func scLinterp(frac, a, b float32) float32 { return a + frac*(b-a) }

func (s *scSaw) startBlock(freqin float32) {
	sr := float32(s.rt.SampleRate)
	s.crossfade = false
	if freqin != s.freqin {
		n := int32((sr * 0.5) / freqin)
		if n != s.n {
			maxf := s.freqin
			if freqin > maxf {
				maxf = freqin
			}
			s.freq = int32(s.cpstoinc * float64(maxf))
			s.crossfade = true
		} else {
			s.freq = int32(s.cpstoinc * float64(freqin))
		}
		s.prevN2, s.prevScale = 2*s.n+1, s.scale
		s.n, s.scale = n, 0.5/float32(n)
	} else {
		s.freq = int32(s.cpstoinc * float64(freqin))
	}
	s.nn2 = 2*s.n + 1
	s.xfade, s.xfadeStep = 0, 1.0/scBlock
	s.freqin = freqin
}

// reset starts the oscillator again, as a new unit does: its state as at the start, for the frequency it has now
func (s *scSaw) reset(freq float32) {
	s.started = true
	s.freqin = freq
	s.n = int32((float32(s.rt.SampleRate) * 0.5) / freq)
	s.scale = 0.5 / float32(s.n)
	s.y1 = -0.46
	s.phase = 0
	s.crossfade = false
}

func (s *scSaw) process(in [][]float64, out []float64, n int) {
	pos := s.rt.Pos()
	for i := 0; i < n; i++ {
		s.processOne(in[0][i:i+1], out[i:i+1], pos+int64(i))
	}
}

// processOne is one sample, at the position of the stream
func (s *scSaw) processOne(f, outs []float64, at int64) {
	const i = 0
	if !s.started {
		s.reset(float32(f[0]))
	}
	for once := true; once; once = false {
		if at%scBlock == 0 {
			s.startBlock(float32(f[i]))
		}
		t0, t1 := scAt(&scInvSine, s.phase)
		pfrac := scPhaseFrac(uint32(s.phase))
		var denom float32
		bad := t0 == scBad || t1 == scBad
		if bad {
			t0, t1 = scAt(&scSine, s.phase)
		}
		denom = t0 + (t1-t0)*pfrac
		if bad && float32(math.Abs(float64(denom))) < 0.0005 {
			s.y1 = 1 + 0.999*s.y1
		} else {
			// the sum of harmonics for a number: the sine of (2N+1) times the angle
			sum := func(n2 int32, scale float32) float32 {
				rphase := s.phase * n2
				a, b := scAt(&scSine, rphase)
				numer := scLinterp(scPhaseFrac(uint32(rphase)), a, b)
				if bad {
					return (numer/denom - 1) * scale
				}
				return (numer*denom - 1) * scale
			}
			v := sum(s.nn2, s.scale)
			if s.crossfade {
				n1 := sum(s.prevN2, s.prevScale)
				v = n1 + s.xfade*(v-n1)
			}
			s.y1 = v + 0.999*s.y1
		}
		outs[i] = float64(s.y1)
		s.phase += s.freq
		if s.crossfade {
			s.xfade += s.xfadeStep
		}
	}
}

// SCSinOsc is SuperCollider's SinOsc as it works it out (OscUGens.cpp): a table of 8192 values read with linear interpolation, and a
// phase that is a 32 bit whole number: a step of cps*2^29/sr cut to a whole number, so the frequency is a little off (for 8 Hz at
// 44100: 9 parts in a million), and the phase offset (in radians) is the starting phase. The frequency is read once in a block
func SCSinOsc[F Param](freq F, phase float64) Sig {
	return Custom(Spec{Name: "SCSinOsc", Key: "scsinosc " + formatFloats(phase), Inputs: []Sig{Signal(freq)}, New: func(rt *Rt) Process {
		cpstoinc := scSineSize * (1 / rt.SampleRate) * 65536
		ph := int32(phase * (scSineSize * (1 / (2 * math.Pi)) * 65536))
		var inc int32
		return func(in [][]float64, out []float64, n int) {
			pos := rt.Pos()
			for i := 0; i < n; i++ {
				if (pos+int64(i))%scBlock == 0 {
					inc = int32(cpstoinc * float64(float32(in[0][i])))
				}
				idx := (uint32(ph) >> 16) & 0x1FFF
				frac := float32(uint32(ph)&0xFFFF) / 65536
				out[i] = float64(scSine[idx] + frac*(scSine[idx+1]-scSine[idx]))
				ph += inc
			}
		}
	}})
}
