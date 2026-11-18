package blocks

import (
	"fmt"
	"harp/dsl"
	"strconv"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/supersaw"
)

// Note is a note to play on a synth
type Note struct {
	Pitch    float64 // a MIDI note number, 60 is middle C; a fraction is cents
	Velocity float64 // 0 to 1, how hard: 0 is taken as 1
	Length   float64 // how long the key is held, in steps: the release of the envelopes comes after. 0 is one step
	Offset   float64 // how far into its step the note starts, 0 up to 1 (a roll, a triplet): 0 is on the step
}

// synthChunk is the number of samples a voice makes at a time. The synths of psalm make a buffer
// at a time, of the size they are made for; the chunks are on a grid of the stream (they start at
// multiples of 64), so that the sound doesn't depend on the blocks the patch is cut in
const synthChunk = dsl.BlockSize

// Synth plays notes on voices of psalm: any of its synths that can start a note (the subtractive
// and the supersaw are tried; dx, piano, organ, modal, waveguide... have the same interface). notes gives the notes to start at every step,
// none for nothing (a chord is several notes at once); a note lasts its Length, and then its envelopes
// let go. At most voices notes sound at once: a note after that takes the voice that has sounded
// longest, which fades out over the chunk it was cut in. The result has channels channels (1 or 2:
// what the voice makes).
//
// A note starts on the sample of its step, and is heard one chunk (64 samples, 1.5 ms) later: the
// chunk is made when it is complete, so that every note in it is known. Voices are made in an environment
// of psalm with the sample rate of the program and random streams seeded with the same number
// every time: the same patch sounds the same. At most 256 notes start in one chunk
func Synth[V poly.Voice](newVoice func() V, channels, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	return synth("", newVoice, channels, voices, notes)
}

// synth is Synth with a key that says which synth it is (what the voices are made from), for a program that takes over from another
// (dsl.Runner.Swap): the same key and the same notes are the same synth, with its voices as they are. Empty: it is never kept
func synth[V poly.Voice](key string, newVoice func() V, channels, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	return synthWith(synthOptions{}, key, newVoice, channels, voices, notes)
}

// synthOptions are what a kind of voice wants of the host
type synthOptions struct {
	velocity bool // the notes say their velocity in the message, as well as in the amplitude: the sampler picks its regions by it
	// paced: the voices read their sound from outside while it plays (a streamer reads files): rendering offline, a chunk is not made faster
	// than they can read, or they would have gaps. (Playing, the device sets the pace)
	paced bool
	// single: the synth is one instrument that all the notes go to (an organ: the keys share its wheels), not a set of voices that take a note each
	single bool
	// mods are signals that move a setting of every voice while it sounds: the value of each at the start of a chunk of 64 samples is given
	// to the voices as the value cabled in at the input Index (see Mod)
	mods []Mod
}

// Mod is a signal (one channel) that is cabled into an input of the voices of a synth: they take its value at the start of every chunk of
// 64 samples, 1.5 ms (the voices glide to it). The inputs are those of the synth, like the position, the cutoff and the resonance of the
// wavetable
type Mod struct {
	Index int
	Sig   dsl.Sig
}

func synthWith[V poly.Voice](opts synthOptions, key string, newVoice func() V, channels, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	if voices < 1 || channels < 1 {
		panic("blocks: Synth: wants at least one voice and one channel")
	}
	st := &synthState{notes: notes, fp: dsl.Fingerprint(notes), opts: opts}
	if key != "" {
		if st.fp != "" {
			key = fmt.Sprintf("%s %d %d", key, channels, voices)
		} else {
			key = ""
		}
	}
	var inputs []dsl.Sig
	var modIdx []int
	for _, m := range opts.mods {
		if m.Sig.Channels() != 1 {
			panic("blocks: Synth: a modulation is a signal of one channel")
		}
		inputs, modIdx = append(inputs, m.Sig), append(modIdx, m.Index)
	}
	if key != "" && len(modIdx) > 0 {
		key = fmt.Sprintf("%s mods %v", key, modIdx)
	}
	return dsl.CustomN(dsl.SpecN{
		Name:    "synth",
		Key:     key,
		State:   st,
		Inputs:  inputs,
		Outputs: channels,
		New: func(rt *dsl.Rt) dsl.ProcessN {
			h := newHost(rt, newVoice, channels, voices)
			h.paced = opts.paced
			h.single = opts.single
			h.modIdx, h.modVal = modIdx, make([]any, len(modIdx))
			st.rt = rt
			dsl.LaneWith(rt, func() []pendingNote { // on the control goroutine: the messages are made here, not on the audio thread
				return st.pending(st.notes.Generate())
			}, h.apply, st.refresh, st.finish)
			return h.process
		},
	})
}

// synthState is what the nodes of a synth share, and what a live edit can change in it: the generator of the notes. Nodes of a synth are
// the same synth when they are made of the same thing (the key); another generator of notes is put in place of the old one, the voices go on
// (what sounds goes on sounding), and the notes that were made but not played are made again from it
type synthState struct {
	opts   synthOptions
	rt     *dsl.Rt
	notes  dsl.Gen[[]Note]
	fp     string
	retime bool
}

// pending makes the messages of the notes of one step
func (st *synthState) pending(ns []Note) []pendingNote {
	if len(ns) == 0 {
		return nil
	}
	out := make([]pendingNote, len(ns))
	for i, n := range ns {
		length, vel := n.Length, n.Velocity
		if length <= 0 {
			length = 1
		}
		if vel <= 0 {
			vel = 1
		}
		msg := psalm.Message{"pitch": n.Pitch}
		if st.opts.velocity {
			msg["velocity"] = min(vel, 1)
		}
		out[i] = pendingNote{msg: msg, amp: min(vel, 1), seconds: length * st.rt.StepSamples / st.rt.SampleRate,
			offset: int64(max(0, min(n.Offset, 0.999)) * st.rt.StepSamples)}
	}
	return out
}

// RebaseFrom is for dsl: the synth can take over from old if the notes are the same, or if they are another generator that can be put in place
func (st *synthState) RebaseFrom(old any) (bool, func()) {
	o, ok := old.(*synthState)
	if !ok {
		return false, nil
	}
	if st.fp != "" && st.fp == o.fp {
		return true, nil
	}
	return true, func() {
		o.notes, o.fp, o.retime = st.notes, st.fp, true
	}
}

func (st *synthState) refresh(ps []pendingNote) []pendingNote {
	if st.retime {
		return st.pending(st.notes.Generate())
	}
	return ps
}

func (st *synthState) finish() { st.retime = false }

// Subtractive is psalm's subtractive synth (two oscillators and a sub, noise, unison, a filter
// with its own envelope, two lfos and a modulation matrix) with voices voices: cfg is one of its
// presets (subtractive.PadPreset(), LeadPreset(), BassPreset()... there are many) or your own. See Synth
func Subtractive(cfg subtractive.Config, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	channels := cfg.NumChannels
	if channels == 0 {
		channels = psalm.Stereo
	}
	return synth(fmt.Sprintf("subtractive %+v", cfg), func() *subtractive.Voice { return subtractive.New(cfg) }, channels, voices, notes)
}

// Supersaw is psalm's supersaw, the Roland JP-8000 sound: seven detuned saws around the note into
// one wide shimmering sound, with a filter, envelopes, an lfo and an ensemble. cfg is one of its
// presets (supersaw.TrancePad(), Hypersaw(), StringMachine()... there are 37) or your own. Every note
// starts its saws at random phases, so no two notes sound the same, but the same patch always
// sounds the same (the random streams are seeded). See Synth
func Supersaw(cfg supersaw.Config, voices int, notes dsl.Gen[[]Note]) dsl.Sig {
	channels := cfg.NumChannels
	if channels == 0 {
		channels = psalm.Stereo
	}
	return synth(fmt.Sprintf("supersaw %+v", cfg), func() *supersaw.Voice { return supersaw.New(cfg) }, channels, voices, notes)
}

// pendingNote is a note as the control side makes it
type pendingNote struct {
	msg     psalm.Message
	amp     float64
	seconds float64
	offset  int64 // samples after the start of the step
}

// noteEvent is a note that starts at a sample of the stream
type noteEvent struct {
	pos  int64
	note pendingNote
}

// slot is a voice, and when it last started a note
type slot[V poly.Voice] struct {
	voice V
	at    int64 // the order in which the voices were started: the lowest is the oldest
	faded int64 // the chunk in which it was last stolen and faded out
}

type host[V poly.Voice] struct {
	rt     *dsl.Rt
	cfg    psalm.Config
	nch    int
	slots  []slot[V]
	events []noteEvent // in time order
	mix    psalm.Channels
	fade   psalm.Channels // what the voices that were stolen sound while they fade
	ready  [][]float32    // the sound of the chunk that was made last
	have   bool
	paced  bool
	single bool
	chunk  int64 // the chunk that is being played
	count  int64 // notes started
	modIdx []int // the inputs of the voices that the modulations are cabled to
	modVal []any // their values at the start of the chunk that is being played
}

func newHost[V poly.Voice](rt *dsl.Rt, newVoice func() V, channels, voices int) *host[V] {
	h := &host[V]{rt: rt, nch: channels, slots: make([]slot[V], voices), events: make([]noteEvent, 0, 256),
		cfg: psalm.Config{SampleRate: rt.SampleRate, BufferSize: synthChunk, BPM: psalm.DefaultBPM, PPQ: psalm.DefaultPPQ},
		mix: psalm.NewChannels(channels, synthChunk), fade: psalm.NewChannels(channels, synthChunk),
		ready: make([][]float32, channels), chunk: -1}
	for i := range h.ready {
		h.ready[i] = make([]float32, synthChunk)
	}
	inEnv(rt.SampleRate, func() {
		for i := range h.slots {
			h.slots[i] = slot[V]{voice: newVoice(), faded: -1}
		}
	})
	return h
}

// apply is on the audio thread, on the sample that a step starts: the notes of the step start here
func (h *host[V]) apply(notes []pendingNote) {
	pos := h.rt.Pos()
	for _, n := range notes {
		if len(h.events) == cap(h.events) {
			return // no room: a note is lost, never an allocation on the audio thread
		}
		// in time order (the notes of a step can be late by different amounts: a roll)
		ev := noteEvent{pos: pos + n.offset, note: n}
		h.events = append(h.events, ev)
		for i := len(h.events) - 1; i > 0 && h.events[i-1].pos > h.events[i].pos; i-- {
			h.events[i-1], h.events[i] = h.events[i], h.events[i-1]
		}
	}
}

func (h *host[V]) process(in [][]float64, out [][]float64, n int) {
	pos := h.rt.Pos()
	for i := 0; i < n; {
		p := pos + int64(i)
		if c := p / synthChunk; c != h.chunk {
			if h.chunk >= 0 {
				h.render(h.chunk)
			}
			h.chunk = c
			for k := range h.modVal {
				h.modVal[k] = in[k][i] // (one value for every voice: boxed once)
			}
		}
		j := int(p % synthChunk)
		m := min(n-i, synthChunk-j)
		for ch := range out {
			if !h.have {
				clear(out[ch][i : i+m])
				continue
			}
			for t := 0; t < m; t++ {
				out[ch][i+t] = float64(h.ready[ch][j+t])
			}
		}
		i += m
	}
}

// render makes the sound of chunk c: the notes that start in it, then every voice that sounds
func (h *host[V]) render(c int64) {
	start := c * synthChunk
	end := start + synthChunk
	for k, v := range h.modVal {
		if v == nil {
			continue
		}
		for i := range h.slots {
			h.slots[i].voice.Receive(v, h.modIdx[k])
		}
	}
	k := 0
	for k < len(h.events) && h.events[k].pos < end {
		h.start(h.events[k], int(max(h.events[k].pos-start, 0)), c)
		k++
	}
	h.events = h.events[:copy(h.events, h.events[k:])]

	h.mix.Clear()
	h.mix.Accum(h.fade)
	h.fade.Clear()
	for i := range h.slots {
		if v := h.slots[i].voice; v.IsActive() {
			v.Synthesize()
			h.mix.Accum(v.Output(0).Channels)
		}
	}
	for ch := range h.ready {
		copy(h.ready[ch], h.mix[ch])
	}
	h.have = true
	if h.paced && !h.rt.Live {
		time.Sleep(300 * time.Microsecond)
	}
}

// pick is the voice for a new note: a free one, or if there is none the one that has sounded longest
func (h *host[V]) pick() *slot[V] {
	var oldest *slot[V]
	for i := range h.slots {
		s := &h.slots[i]
		if !s.voice.IsActive() {
			return s
		}
		if oldest == nil || s.at < oldest.at {
			oldest = s
		}
	}
	return oldest
}

// start gives a note a voice; a voice that still sounds is stolen, and fades out
func (h *host[V]) start(ev noteEvent, offset int, chunk int64) {
	if h.single { // the one instrument takes every note as it is: nothing is stolen, nothing is cleared
		h.slots[0].voice.Note(ev.note.seconds, ev.note.amp, ev.note.msg, psalm.CtrlTick{SampleOffset: offset}, h.cfg)
		h.count++
		return
	}
	s := h.pick()
	if s.voice.IsActive() && s.faded != chunk { // what it sounds is not cut: it fades over this chunk, as psalm's poly does
		s.faded = chunk
		s.voice.Synthesize()
		inv := 1 / float32(synthChunk)
		h.fade.CopyWithFunc(s.voice.Output(0).Channels, func(buf psalm.Buffer, samp float32, index int) {
			buf[index] += samp * (1 - inv*float32(index))
		})
	}
	s.voice.Clear()
	s.voice.Note(ev.note.seconds, ev.note.amp, ev.note.msg, psalm.CtrlTick{SampleOffset: offset}, h.cfg)
	h.count++
	s.at = h.count
}

// noteGen is a generator of notes made of functions
// fp is the fingerprint of the generators it is made of (empty if they do not have one), so that a synth that plays it can be kept
type noteGen struct {
	fp    string
	next  func() []Note
	reset func()
}

func (g *noteGen) Generate() []Note { return g.next() }

// Fingerprint makes the notes a generator that can be taken over (dsl.Fingerprinter)
func (g *noteGen) Fingerprint() string { return g.fp }
func (g *noteGen) Continuous() bool    { return true }
func (g *noteGen) Done() bool          { return false }
func (g *noteGen) Reset()              { g.reset() }

// constGen is a number that is the same for every note (it says what it is, so that a live edit knows the notes are the same)
type constGen struct{ v float64 }

func (c constGen) Generate() float64   { return c.v }
func (c constGen) Continuous() bool    { return true }
func (c constGen) Done() bool          { return false }
func (c constGen) Reset()              {}
func (c constGen) Fingerprint() string { return strconv.FormatFloat(c.v, 'g', -1, 64) }

// Melody plays a melody on a rhythm, as Notes does for the oscillators: the rhythm is pulled every
// step, the pitches only on the steps where it plays, so the next pitch is the next note. Every note has the same velocity and length
func Melody(pitches dsl.Gen[float64], rhythm dsl.Gen[bool], velocity, length float64) dsl.Gen[[]Note] {
	return MelodyOf(pitches, rhythm, constGen{velocity}, constGen{length})
}

// MelodyOf is Melody where the velocity and the length of each note are values of generators too, taken with the pitch when the rhythm plays: a
// sequence of velocities, a sequence of lengths. A note of velocity 0 (or below) is not played: a rest
func MelodyOf(pitches dsl.Gen[float64], rhythm dsl.Gen[bool], velocity, length dsl.Gen[float64]) dsl.Gen[[]Note] {
	return MelodyHits(pitches, dsl.HitsOf(rhythm), velocity, length)
}

// MelodyHits is MelodyOf with a rhythm of hits, that can fall anywhere in a step (a roll, a triplet) and last as long as they say: a note for every
// hit, as long as its length times the length of the hit (when it has one)
func MelodyHits(pitches dsl.Gen[float64], hits dsl.Gen[[]dsl.Hit], velocity, length dsl.Gen[float64]) dsl.Gen[[]Note] {
	return &noteGen{
		fp: noteFP("melody", "", pitches, hits, velocity, length),
		next: func() []Note {
			var notes []Note
			for _, h := range hits.Generate() {
				p, v, l := pitches.Generate(), velocity.Generate(), length.Generate()
				if !(v > 0) {
					continue
				}
				if h.Len > 0 {
					l *= h.Len
				}
				notes = append(notes, Note{Pitch: p, Velocity: v, Length: l, Offset: h.Offset})
			}
			return notes
		},
		reset: func() { pitches.Reset(); hits.Reset(); velocity.Reset(); length.Reset() },
	}
}

// Chords plays chords on a rhythm: on the steps where the rhythm plays, all the notes of the next
// chord start together
func Chords(chords dsl.Gen[[]float64], rhythm dsl.Gen[bool], velocity, length float64) dsl.Gen[[]Note] {
	return ChordsOf(chords, rhythm, constGen{velocity}, constGen{length})
}

// ChordsOf is Chords where the velocity and the length of each chord (the same for all its notes) are values of generators, taken with the chord when
// the rhythm plays. A chord of velocity 0 (or below) is not played
func ChordsOf(chords dsl.Gen[[]float64], rhythm dsl.Gen[bool], velocity, length dsl.Gen[float64]) dsl.Gen[[]Note] {
	return ChordsHits(chords, dsl.HitsOf(rhythm), velocity, length)
}

// ChordsHits is ChordsOf with a rhythm of hits (see MelodyHits)
func ChordsHits(chords dsl.Gen[[]float64], hits dsl.Gen[[]dsl.Hit], velocity, length dsl.Gen[float64]) dsl.Gen[[]Note] {
	return &noteGen{
		fp: noteFP("chords", "", chords, hits, velocity, length),
		next: func() []Note {
			var notes []Note
			for _, h := range hits.Generate() {
				pitches, v, l := chords.Generate(), velocity.Generate(), length.Generate()
				if !(v > 0) {
					continue
				}
				if h.Len > 0 {
					l *= h.Len
				}
				for _, p := range pitches {
					notes = append(notes, Note{Pitch: p, Velocity: v, Length: l, Offset: h.Offset})
				}
			}
			return notes
		},
		reset: func() { chords.Reset(); hits.Reset(); velocity.Reset(); length.Reset() },
	}
}

// noteFP is the fingerprint of notes made of generators: empty if one of them cannot say what it is
func noteFP(kind, settings string, gens ...any) string {
	fp := dsl.Fingerprint(gens...)
	if fp == "" {
		return ""
	}
	return kind + "(" + settings + ";" + fp + ")"
}
