package lang

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"harp/blocks"
	"harp/dsl"

	"github.com/almerlucke/psalm/synths/sampler"
	"github.com/almerlucke/psalm/synths/streamer"
)

// The synths that play recordings: the files are named in the patch, looked for in the folder of the patch (and in the folder harp runs in),
// read when the patch is built, and kept by their name and what was asked of them, so that a live edit that does not change them does not read
// them again

var (
	loadedMu       sync.Mutex
	loadedSamplers = map[string]*sampler.Instrument{}
	loadedStreams  = map[string]*streamer.Instrument{}
	loadedReports  = map[string][]string{}
)

// findFile gives the path of a file that a patch names: as it is if it is absolute, else in the folder of the patch, else in the folder harp
// runs in
func (e *evaluator) findFile(name string) (string, error) {
	if filepath.IsAbs(name) {
		if _, err := os.Stat(name); err != nil {
			return "", fmt.Errorf("there is no file %s", name)
		}
		return name, nil
	}
	tried := []string{}
	for _, dir := range []string{e.patch.Dir, ""} {
		p := filepath.Join(dir, name)
		if dir == "" {
			p = name
		}
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		tried = append(tried, p)
	}
	return "", fmt.Errorf("there is no file %s (looked in %s)", name, strings.Join(tried, " and "))
}

// stamp says what a file is now: its size and time, so that a changed file is read again
func stamp(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d %d", fi.Size(), fi.ModTime().UnixNano())
}

func init() {
	add(&Func{Name: "sampler", Result: TSig, Doc: "psalm's sampler playing notes with a recording: an SFZ instrument (file: \"harp.sfz\", as it is), or a sound file (wav, aiff, flac, ogg) in the way of mode: whole (the sound on one key, or the keys first to args[0] in tune at args[1]), hits (cut at the onsets: args: rise in dB, least gap in seconds, then from and to in seconds), change, pauses, verses, even (args: the number of pieces), chops, notes. The pieces are on the keys from first up (a note of that pitch plays the first piece). loop: seconds of crossfade makes it go round while the key is down. mipmaps: the number of lower resolutions made of each sound (4 is good), for cleaner high notes; the load takes much longer. Stereo out. See docs/recordings.md",
		Params: []Param{{"notes", TGenNotes, nil}, {"file", TStr, nil}, {"mode", TStr, defStr("")}, {"args", TAny, listDefault()}, {"first", TNum, def(36)},
			{"voices", TNum, def(8)}, {"vol", TNum, def(0)}, {"pan", TNum, def(0)}, {"loop", TNum, def(-1)}, {"mipmaps", TNum, def(0)}},
		Call: func(a Args) (Value, error) {
			e := a.ev
			path, err := e.findFile(a.Str("file"))
			if err != nil {
				return Value{}, err
			}
			mode := a.Str("mode")
			if mode == "" {
				mode = "whole"
				if strings.EqualFold(filepath.Ext(path), ".sfz") {
					mode = "sfz"
				}
			}
			args, err := floats(a.value("args"), "args")
			if err != nil {
				return Value{}, err
			}
			parts := []string{fmt.Sprint(a.Int("first")), fmt.Sprintf("%q", path), mode}
			for _, x := range args {
				parts = append(parts, fmt.Sprint(x))
			}
			parts = append(parts, fmt.Sprintf("vol=%g", a.Num("vol")), fmt.Sprintf("pan=%g", a.Num("pan")))
			if l := a.Num("loop"); l >= 0 {
				parts = append(parts, fmt.Sprintf("loop=%g", l))
			}
			line := strings.Join(parts, " ")
			depth := a.Int("mipmaps")
			if depth < 0 || depth > 8 {
				return Value{}, fmt.Errorf("mipmaps is from 0 (none) to 8")
			}
			what := fmt.Sprintf("%s | mipmaps %d | %s", line, depth, stamp(path))
			voices := a.Int("voices")
			if voices < 1 || voices > 64 {
				return Value{}, fmt.Errorf("voices is from 1 to 64")
			}
			load := sampler.Loader(sampler.LoadFile)
			if depth > 0 {
				load = sampler.MipMaps(depth)
			}
			loadedMu.Lock()
			inst, ok := loadedSamplers[what]
			report := loadedReports[what]
			loadedMu.Unlock()
			if !ok {
				var rep []string
				inst, rep, err = sampler.ParseBanks(line, load)
				if err != nil {
					return Value{}, fmt.Errorf("%s: %v", filepath.Base(path), strings.TrimPrefix(err.Error(), "line 1: "))
				}
				report = rep
				loadedMu.Lock()
				loadedSamplers[what], loadedReports[what] = inst, rep
				loadedMu.Unlock()
			}
			e.patch.Info = append(e.patch.Info, report...)
			return sigV(blocks.Sampler(inst, what, voices, a.value("notes").V.(dsl.Gen[[]blocks.Note]))), nil
		}})
	add(&Func{Name: "streamer", Result: TSig, Doc: "psalm's streamer playing notes with a long recording that is read from disk while it plays (the memory it takes is its first half second): file is a sound file; key is the note at which it plays at its own pitch (a note above plays it faster); loop makes it go round (between loopstart and loopend, in seconds) while the key is down; hold lets the end of a note release it; start and end take a part of the file (seconds). Stereo out",
		Params: []Param{{"notes", TGenNotes, nil}, {"file", TStr, nil}, {"key", TNum, def(60)}, {"loop", TBool, defBool(false)}, {"hold", TBool, defBool(false)},
			{"speed", TNum, def(1)}, {"start", TNum, def(0)}, {"end", TNum, def(0)}, {"loopstart", TNum, def(0)}, {"loopend", TNum, def(0)},
			{"release", TNum, def(0.1)}, {"vol", TNum, def(0)}, {"pan", TNum, def(0)}, {"voices", TNum, def(4)}},
		Call: func(a Args) (Value, error) {
			e := a.ev
			path, err := e.findFile(a.Str("file"))
			if err != nil {
				return Value{}, err
			}
			parts := []string{fmt.Sprint(a.Int("key")), fmt.Sprintf("%q", path)}
			if a.value("loop").V.(bool) {
				parts = append(parts, "loop")
			}
			if a.value("hold").V.(bool) {
				parts = append(parts, "hold")
			}
			for _, kv := range []struct {
				name, param string
			}{{"speed", "speed"}, {"start", "start"}, {"end", "end"}, {"loopstart", "loopstart"}, {"loopend", "loopend"}, {"release", "release"}, {"vol", "vol"}, {"pan", "pan"}} {
				x := a.Num(kv.param)
				if math.IsNaN(x) || math.IsInf(x, 0) {
					return Value{}, fmt.Errorf("%s is not a number", kv.param)
				}
				parts = append(parts, fmt.Sprintf("%s=%g", kv.name, x))
			}
			line := strings.Join(parts, " ")
			what := line + " | " + stamp(path)
			voices := a.Int("voices")
			if voices < 1 || voices > 64 {
				return Value{}, fmt.Errorf("voices is from 1 to 64")
			}
			loadedMu.Lock()
			inst, ok := loadedStreams[what]
			report := loadedReports[what]
			loadedMu.Unlock()
			if !ok {
				var rep []string
				inst, rep, err = streamer.ParseInstrument(line)
				if err != nil {
					return Value{}, fmt.Errorf("%s: %v", filepath.Base(path), strings.TrimPrefix(err.Error(), "line 1: "))
				}
				report = rep
				loadedMu.Lock()
				loadedStreams[what], loadedReports[what] = inst, rep
				loadedMu.Unlock()
			}
			e.patch.Info = append(e.patch.Info, report...)
			return sigV(blocks.Streamer(inst, what, voices, a.value("notes").V.(dsl.Gen[[]blocks.Note]))), nil
		}})
}
