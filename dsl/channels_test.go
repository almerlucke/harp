package dsl

import (
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"os"
	"strings"
	"testing"
)

func near(t *testing.T, what string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// panics runs f and returns what it panicked with ("" if it didn't)
func panics(f func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = r.(string)
		}
	}()
	f()
	return ""
}

// dc renders constant channels and gives the value of every channel
func dc(t *testing.T, out Sig) []float64 {
	t.Helper()
	_, ch := Patch{}.Render(out, 0.01)
	v := make([]float64, len(ch))
	for i, c := range ch {
		v[i] = c[5]
	}
	return v
}

// leadStereo is the lead as two detuned saws, spread over the field
func leadStereo() Sig {
	notes := Degrees(57, Minor, Mutate([]int{7, 9, 11, 9, 8, 7, 4, 7}, 0.6))
	v := Notes(notes, Euclid(16, 7, 0), 0.6)
	saws := Saw(Chans(Mul(v.Pitch, 0.997), Mul(v.Pitch, 1.003))) // two channels
	filtered := saws.Lowpass(Lfo(0.3).Range(300, 4000), N(0.9)). // a filter on each, one cutoff
									Mul(Adsr(0.005, 0.15, 0.5, 0.2).On(v.Gate)) // one envelope on both
	return Gain(Spread(filtered, 0.6), 0.4)
}

func TestChannelCounts(t *testing.T) {
	s := Chans(N(1), N(2), N(3))
	if s.Channels() != 3 || N(1).Channels() != 1 {
		t.Fatal("Channels")
	}
	if v := dc(t, s.Ch(1)); len(v) != 1 || v[0] != 2 {
		t.Errorf("Ch(1) = %v", v)
	}
	if v := dc(t, Chans(Chans(N(1), N(2)), N(3))); len(v) != 3 || v[2] != 3 {
		t.Errorf("Chans of Chans = %v", v)
	}
	if !strings.Contains(panics(func() { s.Ch(3) }), "Ch(3)") {
		t.Error("Ch out of range does not panic")
	}
	if !strings.Contains(panics(func() { Stereo(s, N(1)) }), "Stereo") {
		t.Error("Stereo of three channels does not panic")
	}
	if !strings.Contains(panics(func() { Add(Sig{}, N(1)) }), "never made") {
		t.Error("an empty Sig does not panic")
	}
}

func TestExpansion(t *testing.T) {
	stereo := Chans(N(1), N(2))
	// a node on stereo is a node on each channel; a mono signal goes to both
	for _, c := range []struct {
		name string
		got  Sig
		want []float64
	}{
		{"mul by a number", Mul(stereo, 0.5), []float64{0.5, 1}},
		{"mul by a mono signal", Mul(stereo, N(3)), []float64{3, 6}},
		{"mono times stereo", Mul(N(3), stereo), []float64{3, 6}},
		{"mul pairs the channels", Mul(stereo, Chans(N(10), N(100))), []float64{10, 200}},
		{"add", Add(stereo, Chans(N(10), N(100))), []float64{11, 102}},
		{"affine", Affine(stereo, 2, 1), []float64{3, 5}},
		{"gain", Gain(stereo, 2), []float64{2, 4}},
		{"mix", Mix(stereo, N(10), stereo), []float64{12, 14}},
		{"empty mix", Mix(), []float64{0}},
	} {
		got := dc(t, c.got)
		if len(got) != len(c.want) {
			t.Errorf("%s: %d channels, want %d", c.name, len(got), len(c.want))
			continue
		}
		for i := range got {
			near(t, c.name, got[i], c.want[i], 1e-12)
		}
	}
	// channels that don't match are a mistake, said when the patch is made
	msg := panics(func() { Mul(Chans(N(1), N(2)), Chans(N(1), N(2), N(3))) })
	if !strings.Contains(msg, "cannot combine signals of 2 and 3 channels") || !strings.Contains(msg, "Mul") {
		t.Errorf("mismatch: %q", msg)
	}
}

// Each channel of an expanded node has a state of its own: what a filter on stereo gives on a side
// is what the same filter gives alone on that signal
func TestExpandedNodesHaveTheirOwnState(t *testing.T) {
	build := func(f1, f2 float64) Sig {
		return Saw(Chans(N(f1), N(f2))).Lowpass(Lfo(2.0).Range(300, 3000), N(2.0))
	}
	_, both := Patch{}.Render(build(220, 331), 0.3)
	_, a := Patch{}.render1(Saw(220).Lowpass(Lfo(2.0).Range(300, 3000), N(2.0)), 0.3)
	_, b := Patch{}.render1(Saw(331).Lowpass(Lfo(2.0).Range(300, 3000), N(2.0)), 0.3)
	if maxDiff(both[0], a) != 0 || maxDiff(both[1], b) != 0 {
		t.Errorf("channels differ from the filters alone by %g and %g", maxDiff(both[0], a), maxDiff(both[1], b))
	}
	// two channels that are the same signal are one filter, computed once
	same := Patch{}.Start(build(220, 220)).Program()
	if same.Kernels() != 3 { // one saw, one sine for the cutoff, one lowpass
		t.Errorf("%d kernels for two equal channels: %s", same.Kernels(), same)
	}
	apart := Patch{}.Start(build(220, 331)).Program()
	if apart.Kernels() != 5 { // two saws and two lowpasses, and one sine (with its scale) for the cutoff of both
		t.Errorf("%d kernels for two different channels: %s", apart.Kernels(), apart)
	}
}

func TestAConnectsLikePsalm(t *testing.T) {
	for _, c := range []struct {
		name string
		got  Sig
		want []float64
	}{
		{"mono into stereo: both sides", As(N(0.8), 2), []float64{0.8, 0.8}},
		{"mono into 3", As(N(0.8), 3), []float64{0.8, 0.8, 0.8}},
		{"stereo into mono: the average", As(Chans(N(1), N(0.5)), 1), []float64{0.75}},
		{"3 into mono", As(Chans(N(3), N(6), N(9)), 1), []float64{6}},
		{"the same number: as it is", As(Chans(N(1), N(2)), 2), []float64{1, 2}},
		{"Mono", Mono(Chans(N(1), N(0.5))), []float64{0.75}},
		{"Dup", Dup(N(2), 2), []float64{2, 2}},
		{"Sum", Sum(Chans(N(1), N(2), N(3), N(4))), []float64{10}},
		{"Fold to 2", Fold(Chans(N(1), N(2), N(3), N(4)), 2), []float64{4, 6}},
		{"Fold to 3 of 4", Fold(Chans(N(1), N(2), N(3), N(4)), 3), []float64{5, 2, 3}},
	} {
		got := dc(t, c.got)
		if len(got) != len(c.want) {
			t.Errorf("%s: %d channels, want %d", c.name, len(got), len(c.want))
			continue
		}
		for i := range got {
			near(t, c.name, got[i], c.want[i], 1e-12)
		}
	}
	// no rule for 3 into 2: it says so
	if msg := panics(func() { As(Chans(N(1), N(2), N(3)), 2) }); !strings.Contains(msg, "3 channels to 2") {
		t.Errorf("3 into 2: %q", msg)
	}
	if panics(func() { Fold(N(1), 2) }) == "" || panics(func() { Dup(Chans(N(1), N(2)), 2) }) == "" {
		t.Error("Fold to more channels, or Dup of stereo, does not panic")
	}
}

func TestPanLawIsEqualPower(t *testing.T) {
	for _, pos := range []float64{-1, -0.5, 0, 0.3, 1, 2.5, -7} { // beyond the sides is the side
		v := dc(t, Pan(N(1), pos))
		c := math.Min(math.Max(pos, -1), 1)
		near(t, "left", v[0], math.Cos((c+1)*math.Pi/4), 1e-12)
		near(t, "right", v[1], math.Sin((c+1)*math.Pi/4), 1e-12)
		near(t, "power", v[0]*v[0]+v[1]*v[1], 1, 1e-12)
	}
	v := dc(t, Pan(N(1), -1))
	near(t, "hard left: left", v[0], 1, 1e-12)
	near(t, "hard left: right", v[1], 0, 1e-12)
	v = dc(t, Pan(N(1), 0))
	near(t, "middle", v[0], 1/math.Sqrt2, 1e-12) // -3 dB
	near(t, "middle", v[1], 1/math.Sqrt2, 1e-12)
}

// Pan of several channels: left, right, left, right...
func TestPanExpands(t *testing.T) {
	v := dc(t, Pan(Chans(N(1), N(2)), Chans(N(-1), N(1))))
	want := []float64{1, 0, 0, 2}
	if len(v) != 4 {
		t.Fatalf("%d channels, want 4", len(v))
	}
	for i := range v {
		near(t, "channel", v[i], want[i], 1e-12)
	}
	if got := Pan(Chans(N(1), N(2), N(3)), 0.1).Channels(); got != 6 {
		t.Errorf("3 channels panned: %d", got)
	}
	if msg := panics(func() { Pan(Chans(N(1), N(2)), Chans(N(0), N(0), N(0))) }); !strings.Contains(msg, "Pan") {
		t.Errorf("mismatch in Pan: %q", msg)
	}
}

// A position that is a signal follows the same law, every sample
func TestPanFollowsASignal(t *testing.T) {
	_, pos := Patch{}.render1(Lfo(30.0), 0.2)
	_, ch := Patch{}.Render(Pan(N(1), Lfo(30.0)), 0.2)
	for i := range pos {
		near(t, "left", ch[0][i], math.Cos((pos[i]+1)*math.Pi/4), 1e-12)
		near(t, "right", ch[1][i], math.Sin((pos[i]+1)*math.Pi/4), 1e-12)
		near(t, "power", ch[0][i]*ch[0][i]+ch[1][i]*ch[1][i], 1, 1e-12)
	}
}

// A fixed position is a pair of constant gains, not trigonometry in the audio loop
func TestFixedPanCostsNoTrigonometry(t *testing.T) {
	still := Patch{}.Start(Pan(Saw(220), -0.5)).Program()
	for _, n := range still.names {
		if strings.HasPrefix(n, "pan.") {
			t.Errorf("a fixed pan runs a kernel: %s", still)
		}
	}
	moving := Patch{}.Start(Pan(Saw(220), Lfo(0.5))).Program()
	if !strings.Contains(moving.String(), "pan.left") || !strings.Contains(moving.String(), "pan.right") {
		t.Errorf("a moving pan has no gain kernels: %s", moving)
	}
	// and a pan on a node nobody else uses moves into it: no extra kernels for a scale
	gained := Patch{}.Start(Pan(Saw(220), 0.2).Ch(0)).Program()
	if gained.Kernels() != 1 {
		t.Errorf("%d kernels for a saw with a gain", gained.Kernels())
	}
}

func TestSpread(t *testing.T) {
	// three channels at -w, 0, w, equal power each, and 1/sqrt(3) for the number of them
	w := 0.5
	v := dc(t, Spread(Chans(N(1), N(1), N(1)), w))
	var l, r float64
	for _, pos := range []float64{-w, 0, w} {
		gl, gr := panGains(pos)
		l, r = l+gl, r+gr
	}
	near(t, "left", v[0], l/math.Sqrt(3), 1e-12)
	near(t, "right", v[1], r/math.Sqrt(3), 1e-12)
	near(t, "symmetric", v[0], v[1], 1e-12)
	// the order is the order of the channels: the first is on the left
	v = dc(t, Spread(Chans(N(1), N(0)), 1))
	near(t, "first is left", v[0], 1/math.Sqrt2, 1e-12)
	near(t, "first is not right", v[1], 0, 1e-12)
	// one channel: the middle, at full level
	v = dc(t, Spread(N(1), 1))
	near(t, "one channel", v[0], 1/math.Sqrt2, 1e-12)
	// width 0: all in the middle
	v = dc(t, Spread(Chans(N(1), N(1)), 0))
	near(t, "width 0", v[0], 2*(1/math.Sqrt2)/math.Sqrt2, 1e-12)
	if Spread(Chans(N(1), N(2), N(3), N(4), N(5)), 1).Channels() != 2 {
		t.Error("Spread is not stereo")
	}
}

// A generator of chords is a multichannel pitch: one channel for every note
func TestPitchNIsOneChannelForEveryNote(t *testing.T) {
	hz := func(n float64) float64 { return 440 * math.Pow(2, (n-69)/12) }
	chords := Loop([]float64{60, 64, 67}, []float64{62, 65}, nil, []float64{72, 76, 79, 83})
	p := Patch{SampleRate: 100, BPM: 60, StepsPerBeat: 1}
	prog, ch := p.Render(PitchN(chords, 3), 4)
	if len(ch) != 3 {
		t.Fatalf("%d channels", len(ch))
	}
	for step, want := range [][]float64{
		{hz(60), hz(64), hz(67)},
		{hz(62), hz(65), hz(65)}, // two notes in three channels: the last is repeated
		{hz(62), hz(65), hz(65)}, // no notes: nothing changes
		{hz(72), hz(76), hz(79)}, // four notes in three channels: cut off
	} {
		for c := range ch {
			near(t, "pitch", ch[c][step*100+50], want[c], 1e-9)
		}
	}
	_ = prog
	if panics(func() { PitchN(chords, 0) }) == "" {
		t.Error("width 0 does not panic")
	}
	// a chord is the pitch of a bank of saws, in the field, as loud as one
	out := Spread(Saw(PitchN(chords, 3)), 0.8)
	if out.Channels() != 2 {
		t.Errorf("a chord of saws spread is %d channels", out.Channels())
	}
}

func TestStereoOptimizedSoundsTheSame(t *testing.T) {
	run := func(noOpt bool) (*Program, [][]float64) {
		Seed(3)
		return Patch{BPM: 110, NoOptimize: noOpt}.Render(leadStereo(), 4)
	}
	po, o := run(false)
	pp, p := run(true)
	if d := math.Max(maxDiff(o[0], p[0]), maxDiff(o[1], p[1])); d > 1e-9 {
		t.Errorf("optimized differs by %g", d)
	}
	if po.Kernels() >= pp.Kernels() {
		t.Errorf("kernels %d (optimized) vs %d", po.Kernels(), pp.Kernels())
	}
	var peakL, peakR, diff float64
	for i := range o[0] {
		peakL, peakR = math.Max(peakL, math.Abs(o[0][i])), math.Max(peakR, math.Abs(o[1][i]))
		diff = math.Max(diff, math.Abs(o[0][i]-o[1][i]))
	}
	if peakL < 0.05 || peakR < 0.05 || diff < 0.01 {
		t.Errorf("peaks %v %v, left and right differ by %v: expected two different sounds", peakL, peakR, diff)
	}
}

func TestProcessWantsTheChannelsOfThePatch(t *testing.T) {
	for _, c := range []struct {
		name string
		out  Sig
		dst  [][]float64
	}{
		{"mono patch, stereo buffers", Saw(220), [][]float64{make([]float64, 8), make([]float64, 8)}},
		{"stereo patch, mono buffer", Dup(Saw(220), 2), [][]float64{make([]float64, 8)}},
		{"different lengths", Dup(Saw(220), 2), [][]float64{make([]float64, 8), make([]float64, 4)}},
	} {
		if panics(func() { Patch{}.Start(c.out).Process(c.dst...) }) == "" {
			t.Errorf("%s: no panic", c.name)
		}
	}
}

// A stereo patch on a device with three channels: left, right, and the third is silent. A mono patch is on all
func TestPlayChannels(t *testing.T) {
	Seed(7)
	_, want := Patch{SampleRate: 48000, BPM: 110}.Render(leadStereo(), 1)

	Seed(7)
	eng := newFake(48000, 1000)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error)
	go func() { done <- Patch{BPM: 110}.Play(ctx, leadStereo(), eng) }()
	<-eng.started
	got := [3][]float64{}
	for len(got[0]) < len(want[0]) {
		c := [][]float32{make([]float32, eng.size), make([]float32, eng.size), make([]float32, eng.size)}
		for i := range c[2] {
			c[2][i] = 9 // must be overwritten
		}
		eng.cb(nil, c)
		for ch := range c {
			for _, v := range c[ch] {
				got[ch] = append(got[ch], float64(v))
			}
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	clamp := func(v float64) float64 { return math.Max(-1, math.Min(1, v)) }
	var differ bool
	for i := range want[0] {
		near(t, "left", got[0][i], clamp(want[0][i]), 1e-6)
		near(t, "right", got[1][i], clamp(want[1][i]), 1e-6)
		if got[2][i] != 0 {
			t.Fatal("the third channel is not silent")
		}
		differ = differ || got[0][i] != got[1][i]
	}
	if !differ {
		t.Error("left and right are the same")
	}
}

func TestWAVChannels(t *testing.T) {
	var b bytes.Buffer
	if err := writeWAV(&b, 44100, []float64{0.5, -0.5}, []float64{1, -1}); err != nil {
		t.Fatal(err)
	}
	h := b.Bytes()
	u16 := func(o int) int { return int(binary.LittleEndian.Uint16(h[o:])) }
	u32 := func(o int) int { return int(binary.LittleEndian.Uint32(h[o:])) }
	i16 := func(o int) int { return int(int16(binary.LittleEndian.Uint16(h[o:]))) }
	if u16(22) != 2 || u32(24) != 44100 || u32(28) != 44100*4 || u16(32) != 4 || u32(40) != 8 || len(h) != 44+8 {
		t.Errorf("header: channels %d rate %d byte rate %d align %d data %d, %d bytes", u16(22), u32(24), u32(28), u16(32), u32(40), len(h))
	}
	if i16(44) != 16384 || i16(46) != 32767 || i16(48) != -16384 || i16(50) != -32767 { // L R L R: 0.5 of 32767 is 16383.5, rounded to 16384
		t.Errorf("samples: %d %d %d %d", i16(44), i16(46), i16(48), i16(50))
	}
}

// The samples are rounded to the nearest of the 65535 steps, not cut: the error is at most half a step, and
// it has no direction (cutting always errs towards zero, which is a distortion of its own)
func TestWAVRoundsToTheNearestStep(t *testing.T) {
	const n = 20000
	x := make([]float64, n)
	for i := range x {
		x[i] = math.Sin(float64(i)*0.0371) * 0.9 * math.Sin(float64(i)*0.00011)
	}
	var b bytes.Buffer
	if err := writeWAV(&b, 44100, x); err != nil {
		t.Fatal(err)
	}
	h := b.Bytes()
	var worst, bias float64
	for i := range x {
		q := float64(int16(binary.LittleEndian.Uint16(h[44+2*i:]))) / 32767
		e := q - x[i]
		worst = math.Max(worst, math.Abs(e))
		bias += e
	}
	if worst > 0.5/32767+1e-12 {
		t.Errorf("the error is %v steps, at most half a step is possible", worst*32767)
	}
	if math.Abs(bias/n) > 0.02/32767 { // cutting gives a mean error of a quarter of a step in the direction of zero
		t.Errorf("the error has a direction: mean %v steps", bias/n*32767)
	}
	// and it is the same for a negative as for a positive value
	var c bytes.Buffer
	if err := writeWAV(&c, 44100, []float64{0.3, -0.3, 1.5, -1.5, 0}); err != nil {
		t.Fatal(err)
	}
	g := c.Bytes()
	at := func(i int) int { return int(int16(binary.LittleEndian.Uint16(g[44+2*i:]))) }
	if at(0) != -at(1) || at(2) != 32767 || at(3) != -32767 || at(4) != 0 {
		t.Errorf("%d %d %d %d %d", at(0), at(1), at(2), at(3), at(4))
	}
}

func TestWAVFloat(t *testing.T) {
	var b bytes.Buffer
	if err := writeWAVFloat(&b, 48000, []float64{0.5, -1.5, 0.123456789}, []float64{0, 2, -0.25}); err != nil {
		t.Fatal(err)
	}
	h := b.Bytes()
	u16 := func(o int) int { return int(binary.LittleEndian.Uint16(h[o:])) }
	u32 := func(o int) int { return int(binary.LittleEndian.Uint32(h[o:])) }
	f32 := func(i int) float32 { return math.Float32frombits(binary.LittleEndian.Uint32(h[44+4*i:])) }
	if u16(20) != 3 || u16(22) != 2 || u32(24) != 48000 || u32(28) != 48000*8 || u16(32) != 8 || u16(34) != 32 || u32(40) != 24 || len(h) != 44+24 {
		t.Errorf("header: format %d channels %d rate %d byte rate %d align %d bits %d data %d, %d bytes", u16(20), u16(22), u32(24), u32(28), u16(32), u16(34), u32(40), len(h))
	}
	// the values are as they are: beyond full scale too, as float32, L R L R
	if f32(0) != 0.5 || f32(1) != 0 || f32(2) != -1.5 || f32(3) != 2 || f32(4) != float32(0.123456789) || f32(5) != -0.25 {
		t.Errorf("samples: %v %v %v %v %v %v", f32(0), f32(1), f32(2), f32(3), f32(4), f32(5))
	}
}

func TestReadWAV(t *testing.T) {
	dir := t.TempDir()
	x := []float64{0, 0.5, -0.5, 0.25, 1.5, -0.123456789}
	y := []float64{1, -1, 0.1, 0, 0.75, 0.3}
	// float: exactly as float32, beyond full scale too
	if err := WriteWAVFloat(dir+"/f.wav", 48000, x, y); err != nil {
		t.Fatal(err)
	}
	rate, ch, err := ReadWAV(dir + "/f.wav")
	if err != nil || rate != 48000 || len(ch) != 2 || len(ch[0]) != len(x) {
		t.Fatalf("float: rate %d, %d channels, err %v", rate, len(ch), err)
	}
	for i := range x {
		if ch[0][i] != float64(float32(x[i])) || ch[1][i] != float64(float32(y[i])) {
			t.Errorf("float sample %d: %v %v", i, ch[0][i], ch[1][i])
		}
	}
	// 16 bit: what WriteWAV wrote, within a step, clipped at full scale
	if err := WriteWAV(dir+"/p.wav", 44100, x, y); err != nil {
		t.Fatal(err)
	}
	rate, ch, err = ReadWAV(dir + "/p.wav")
	if err != nil || rate != 44100 || len(ch) != 2 {
		t.Fatalf("pcm: rate %d, %d channels, err %v", rate, len(ch), err)
	}
	for i := range x {
		want := math.Max(-1, math.Min(1, x[i]))
		if math.Abs(ch[0][i]-want) > 1.0/32767 {
			t.Errorf("pcm sample %d: %v, want %v", i, ch[0][i], want)
		}
	}
	// the rest: not a wav, cut off, a chunk we do not know in front of the data, a format we do not read
	if _, _, err := parseWAV([]byte("not a wav file at all")); err == nil {
		t.Error("garbage is read")
	}
	good, _ := os.ReadFile(dir + "/f.wav")
	if _, c, err := parseWAV(good[:len(good)-6]); err != nil || len(c[0]) < len(x)-1 { // the last frame is cut: what is there is read
		t.Errorf("a cut off file: %v", err)
	}
	var extra bytes.Buffer
	extra.Write(good[:12])
	extra.WriteString("LIST")
	binary.Write(&extra, binary.LittleEndian, uint32(5)) // an odd size: one byte of padding follows
	extra.Write([]byte{1, 2, 3, 4, 5, 0})
	extra.Write(good[12:])
	if _, c, err := parseWAV(extra.Bytes()); err != nil || len(c) != 2 || c[0][1] != 0.5 {
		t.Errorf("an unknown chunk: %v", err)
	}
	bad := append([]byte{}, good...)
	binary.LittleEndian.PutUint16(bad[20:], 99)
	if _, _, err := parseWAV(bad); err == nil {
		t.Error("format 99 is read")
	}
}
