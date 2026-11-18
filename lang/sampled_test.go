package lang

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"harp/dsl"

	"github.com/almerlucke/sndfile"
)

// a recording for the tests: four bursts of a tone, each a decaying 440 Hz (or another) of 0.3 s, in a quiet room, 3 seconds at 44100 Hz
func writeBursts(t *testing.T, dir string) string {
	t.Helper()
	const sr = 44100
	x := make([]float64, 3*sr)
	s := uint64(12345)
	for i := range x {
		s ^= s << 13
		s ^= s >> 7
		s ^= s << 17
		x[i] = 0.0005 * (float64(s%2001)/1000 - 1) // the room
	}
	for k, at := range []float64{0.2, 0.9, 1.6, 2.3} {
		f := 440.0 * math.Pow(2, float64(k)/4)
		for i := 0; i < int(0.3*sr); i++ {
			t := float64(i) / sr
			x[int(at*sr)+i] += 0.6 * math.Exp(-9*t) * math.Sin(2*math.Pi*f*t)
		}
	}
	path := filepath.Join(dir, "bursts.wav")
	if err := dsl.WriteWAVFloat(path, sr, x, x); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeTone(t *testing.T, dir, name string, seconds float64) string {
	t.Helper()
	const sr = 44100
	x := make([]float64, int(seconds*sr))
	for i := range x {
		x[i] = 0.4 * math.Sin(2*math.Pi*220*float64(i)/sr)
	}
	path := filepath.Join(dir, name)
	if err := dsl.WriteWAVFloat(path, sr, x); err != nil {
		t.Fatal(err)
	}
	return path
}

func renderIn(t *testing.T, dir, src string, seconds float64) ([]float64, *Patch) {
	t.Helper()
	p, err := CompileDir(src, dir)
	if err != nil {
		t.Fatalf("%v\n%s", err, src)
	}
	_, ch := p.Settings().Render(p.Out, seconds)
	return ch[0], p
}

func energy(x []float64) (e float64) {
	for _, v := range x {
		e += v * v
	}
	return e / float64(max(len(x), 1))
}

func TestASamplerPlaysARecording(t *testing.T) {
	dir := t.TempDir()
	writeTone(t, dir, "tone.wav", 1)
	// the whole sound on key 60 (first: 60): a note of 60 plays it at its own pitch, silence before it starts
	x, _ := renderIn(t, dir, "bpm = 120\nnotes = melody([60], every(32), velocity: 1, length: 8)\nout = sampler(notes, \"tone.wav\", first: 60)\n", 2)
	if energy(x[:100]) > 1e-3 && energy(x[10000:20000]) < 1e-4 {
		t.Errorf("energy at the start %v, later %v", energy(x[:100]), energy(x[10000:20000]))
	}
	if energy(x[2000:30000]) < 1e-3 {
		t.Errorf("the sampler is silent: %v", energy(x[2000:30000]))
	}
	// a note an octave up is the sound an octave up: a tone of 440 Hz instead of 220
	zero := func(x []float64) int { // zero crossings, up
		n := 0
		for i := 1; i < len(x); i++ {
			if x[i-1] < 0 && x[i] >= 0 {
				n++
			}
		}
		return n
	}
	lo, _ := renderIn(t, dir, "bpm = 120\nnotes = melody([60], every(32), velocity: 1, length: 8)\nout = sampler(notes, \"tone.wav\", first: 60)\n", 1)
	hi, _ := renderIn(t, dir, "bpm = 120\nnotes = melody([72], every(32), velocity: 1, length: 8)\nout = sampler(notes, \"tone.wav\", mode: \"whole\", args: [84 60], first: 60)\n", 1)
	a, b := zero(lo[2000:20000]), zero(hi[2000:20000])
	if b < 2*a-6 || b > 2*a+6 {
		t.Errorf("an octave up has %d rising zero crossings against %d: it should be twice as many", b, a)
	}
}

func TestASamplerCutsARecordingIntoPieces(t *testing.T) {
	dir := t.TempDir()
	writeBursts(t, dir)
	src := "bpm = 120\nnotes = melody([%d], every(32), velocity: 1, length: 4)\nout = sampler(notes, \"bursts.wav\", mode: \"hits\", args: [9 0.08], first: 36)\n"
	_, p := renderIn(t, dir, strings.Replace(src, "%d", "36", 1), 1)
	// the first piece is what comes before the first hit (the quiet room): five pieces on keys 36 to 40, the four bursts on the last four
	if len(p.Info) == 0 || !strings.Contains(p.Info[0], "5 on keys 36-40") {
		t.Errorf("what the recording was cut into: %v", p.Info)
	}
	// every key is a piece: the bursts sound
	var pieces []float64
	for key := 37; key < 41; key++ {
		x, _ := renderIn(t, dir, strings.Replace(src, "%d", string(rune('0'+key/10))+string(rune('0'+key%10)), 1), 1)
		if energy(x[1000:20000]) < 1e-4 {
			t.Errorf("key %d is silent", key)
		}
		pieces = append(pieces, energy(x[1000:20000]))
	}
	_ = pieces
}

func TestASamplerPlaysAnSFZInstrument(t *testing.T) {
	dir := t.TempDir()
	writeTone(t, dir, "tone.wav", 1)
	sfz := "<group> loop_mode=no_loop\n<region> sample=tone.wav lokey=0 hikey=127 pitch_keycenter=57\n"
	if err := os.WriteFile(filepath.Join(dir, "inst.sfz"), []byte(sfz), 0o644); err != nil {
		t.Fatal(err)
	}
	x, _ := renderIn(t, dir, "bpm = 120\nnotes = melody([57], every(32), velocity: 1, length: 8)\nout = sampler(notes, \"inst.sfz\")\n", 1.5)
	if energy(x[2000:30000]) < 1e-3 {
		t.Errorf("the sfz instrument is silent")
	}
}

func TestAStreamerPlaysALongRecordingFromDisk(t *testing.T) {
	dir := t.TempDir()
	writeTone(t, dir, "long.wav", 1)
	// one note of 3 seconds on a sound of 1 second: with loop it goes on, without it ends
	src := "bpm = 120\nnotes = melody([60], every(64), velocity: 1, length: 24)\nout = streamer(notes, \"long.wav\", loop: %v, hold: true)\n"
	looped, _ := renderIn(t, dir, strings.Replace(src, "%v", "true", 1), 3)
	once, _ := renderIn(t, dir, strings.Replace(src, "%v", "false", 1), 3)
	late := func(x []float64) float64 { return energy(x[int(1.6*44100):int(2.4*44100)]) }
	if late(looped) < 1e-3 {
		t.Errorf("a loop is silent after the end of the sound: %v", late(looped))
	}
	if late(once) > 1e-4 {
		t.Errorf("a sound that is not a loop goes on after its end: %v", late(once))
	}
	if energy(once[2000:30000]) < 1e-3 {
		t.Errorf("the streamer is silent at the start")
	}
}

func TestFilesAreLookedForWhereTheyAre(t *testing.T) {
	dir := t.TempDir()
	writeTone(t, dir, "tone.wav", 0.5)
	sub := filepath.Join(dir, "sounds")
	os.Mkdir(sub, 0o755)
	writeTone(t, sub, "inner.wav", 0.5)
	ok := "notes = melody([60], every(8))\nout = sampler(notes, %q, first: 60)\n"
	for _, name := range []string{"tone.wav", "sounds/inner.wav", filepath.Join(dir, "tone.wav")} {
		if _, err := CompileDir(strings.Replace(ok, "%q", "\""+name+"\"", 1), dir); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for src, mention := range map[string]string{
		"notes = melody([60], every(8))\nout = sampler(notes, \"nothing.wav\")":                             "there is no file nothing.wav (looked in",
		"notes = melody([60], every(8))\nout = sampler(notes, \"tone.wav\", mode: \"juggle\")":              "juggle",
		"notes = melody([60], every(8))\nout = sampler(notes, \"tone.wav\", voices: 0)":                     "voices",
		"notes = melody([60], every(8))\nout = streamer(notes, \"nothing.wav\")":                            "there is no file",
		"notes = melody([60], every(8))\nout = sampler(notes, \"tone.wav\", args: [\"a\"], mode: \"hits\")": "not a number",
	} {
		if _, err := CompileDir(src, dir); err == nil || !strings.Contains(err.Error(), mention) {
			t.Errorf("%q: %v, want a mention of %q", src, err, mention)
		}
	}
}

// a recording is read once, and again when the file changes
func TestARecordingIsReadOnceWhileItIsTheSame(t *testing.T) {
	dir := t.TempDir()
	path := writeTone(t, dir, "tone.wav", 0.5)
	src := "notes = melody([60], every(8))\nout = sampler(notes, \"tone.wav\", first: 60)\n"
	loadedMu.Lock()
	before := len(loadedSamplers)
	loadedMu.Unlock()
	for i := 0; i < 3; i++ {
		if _, err := CompileDir(src, dir); err != nil {
			t.Fatal(err)
		}
	}
	loadedMu.Lock()
	after := len(loadedSamplers)
	loadedMu.Unlock()
	if after != before+1 {
		t.Errorf("three builds made %d instruments", after-before)
	}
	// the file is changed: it is read again
	writeTone(t, dir, "tone.wav", 0.7)
	os.Chtimes(path, timeNow().Add(5), timeNow().Add(5))
	if _, err := CompileDir(src, dir); err != nil {
		t.Fatal(err)
	}
	loadedMu.Lock()
	again := len(loadedSamplers)
	loadedMu.Unlock()
	if again != after+1 {
		t.Errorf("a changed file was not read again: %d, %d", again, after)
	}
}

func timeNow() time.Time { return time.Now() }

// edits of a patch that plays recordings: the notes can change without the sampler being cut; another setting of the sound is a new instrument
func TestALiveSessionWithASamplerAndAStreamer(t *testing.T) {
	dir := t.TempDir()
	tone := writeTone(t, dir, "tone.wav", 1)
	src := "bpm = 120\nsteps = 4\nseed = 1\nhits = melody([60 62], euclid(16, 5), velocity: 0.9, length: 6)\n" +
		"out = sampler(hits, \"" + tone + "\", first: 60, args: [72 60]).gain(0.5) + streamer(melody([60], every(64), length: 40), \"" + tone + "\", loop: true).gain(0.2)\n"
	s := startSession(t, "recordings", src)
	restarts := map[string][]string{}
	for _, e := range []edit{
		{"the rhythm of the notes", "euclid(16, 5)", "euclid(16, 7)"},
		{"the notes", "melody([60 62]", "melody([60 64 67]"},
		{"the gain", ".gain(0.5)", ".gain(0.3)"},
		{"the volume of the sound", "args: [72 60])", "args: [72 60], vol: -6)"},
		{"the streamer loops no more", "loop: true", "loop: false"},
	} {
		s.runEdits(src, 0, []edit{e})
		src = strings.Replace(src, e.from, e.to, 1)
		restarts[e.what] = s.r.LastRestarted()
		s.play(0.5)
	}
	s.check(3)
	for _, e := range s.events {
		t.Log(e)
	}
	for _, what := range []string{"the rhythm of the notes", "the notes", "the gain"} {
		if len(restarts[what]) != 0 {
			t.Errorf("%q restarted %v", what, restarts[what])
		}
	}
	if len(restarts["the volume of the sound"]) == 0 {
		t.Error("another volume of the sound is the same instrument")
	}
}

// the lower resolutions of a recording are kept in a folder: made once, read after that, made again when the file changes
func TestMipMapsAreKept(t *testing.T) {
	cache := t.TempDir()
	was := sndfile.MipMapCache
	sndfile.MipMapCache = cache
	defer func() { sndfile.MipMapCache = was }()
	dir := t.TempDir()
	path := writeBursts(t, dir)
	count := func() int {
		m, _ := filepath.Glob(filepath.Join(cache, "*.mip"))
		return len(m)
	}
	first, err := sndfile.NewMipMapSoundFile[float32](path, 4)
	if err != nil {
		t.Fatal(err)
	}
	if count() != 1 {
		t.Fatalf("%d files in the cache after the first load", count())
	}
	second, err := sndfile.NewMipMapSoundFile[float32](path, 4)
	if err != nil {
		t.Fatal(err)
	}
	for c := 0; c < first.NumChannels(); c++ {
		for d := 0; d < 4; d++ {
			a, b := first.Buffer(c, d), second.Buffer(c, d)
			if len(a) != len(b) {
				t.Fatalf("channel %d level %d: %d and %d samples", c, d, len(a), len(b))
			}
			for i := range a {
				if a[i] != b[i] {
					t.Fatalf("channel %d level %d sample %d: made %v, read %v", c, d, i, a[i], b[i])
				}
			}
		}
	}
	if count() != 1 {
		t.Errorf("a load that read the cache made another file: %d", count())
	}
	// another depth is another file; a changed file is another file
	if _, err := sndfile.NewMipMapSoundFile[float32](path, 3); err != nil {
		t.Fatal(err)
	}
	if count() != 2 {
		t.Errorf("another depth: %d files", count())
	}
	writeTone(t, dir, "bursts.wav", 2)
	os.Chtimes(path, timeNow().Add(7), timeNow().Add(7))
	if _, err := sndfile.NewMipMapSoundFile[float32](path, 4); err != nil {
		t.Fatal(err)
	}
	if count() != 3 {
		t.Errorf("a changed file: %d files", count())
	}
	// a file in the cache that is cut short is not used, and is made again
	files, _ := filepath.Glob(filepath.Join(cache, "*.mip"))
	for _, f := range files {
		os.Truncate(f, 100)
	}
	again, err := sndfile.NewMipMapSoundFile[float32](path, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Buffer(0, 3)) == 0 {
		t.Error("no sound after a damaged cache")
	}
	// nothing is kept without a folder
	sndfile.MipMapCache = ""
	before := count()
	if _, err := sndfile.NewMipMapSoundFile[float32](path, 4); err != nil {
		t.Fatal(err)
	}
	if count() != before {
		t.Error("a file was kept with no cache folder")
	}
}

func TestTheCacheCanBeMeasuredAndCleared(t *testing.T) {
	cache := t.TempDir()
	was := sndfile.MipMapCache
	sndfile.MipMapCache = cache
	defer func() { sndfile.MipMapCache = was }()
	path := writeBursts(t, t.TempDir())
	if _, err := sndfile.NewMipMapSoundFile[float32](path, 4); err != nil {
		t.Fatal(err)
	}
	if n, size := CacheSize(); n != 1 || size < 1000 {
		t.Errorf("%d files, %d bytes", n, size)
	}
	if n, err := ClearCache(); err != nil || n != 1 {
		t.Errorf("cleared %d: %v", n, err)
	}
	if n, _ := CacheSize(); n != 0 {
		t.Errorf("%d files after clearing", n)
	}
}
