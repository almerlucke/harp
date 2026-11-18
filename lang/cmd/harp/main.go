// Command harp checks, renders and plays patches written in the harp language.
//
//	harp check patch.harp                       read and check a patch
//	harp render patch.harp -o patch.wav -seconds 10 [-float]
//	harp play patch.harp [-seconds 30] [-buffer 512] [-rate 44100] [-watch=false] [-cpuprofile file]
//	                                            plays it, and plays it again when the file is saved: what is the same keeps its state
//	harp compare patch.harp [-seconds 10] [-sc code] [-stats] [-save dir]
//	                                            render it in harp and in SuperCollider and say how alike they are
//	harp cache [clear]                          the folder of the mipmaps of recordings that are kept, and how big it is
//	harp presets [synth|-md]                    the presets of the synths, by name
//	harp settings [synth|-md]                   the settings of the synths: every parameter, with its range
//	harp functions [-md]                        the functions of the language (-md: as the markdown reference of docs/)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	paengine "github.com/almerlucke/psalm/audio/portaudio"

	"harp/dsl"
	"harp/lang"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "check":
		fs := flag.NewFlagSet("check", flag.ExitOnError)
		patch, _ := load(fs, os.Args[2:])
		fmt.Printf("ok: %d channels, %.0f bpm, %.0f steps a beat\n", patch.Out.Channels(), patch.BPM, patch.StepsPerBeat)
		for _, line := range patch.Info {
			fmt.Println("  " + line)
		}
	case "render":
		fs := flag.NewFlagSet("render", flag.ExitOnError)
		out := fs.String("o", "", "the wav file to write (default: the name of the patch with .wav)")
		secs := fs.Float64("seconds", 10, "how many seconds to render")
		rate := fs.Float64("rate", 44100, "sample rate")
		float := fs.Bool("float", false, "write 32 bit floating point samples, as they are, instead of 16 bit")
		patch, path := load(fs, os.Args[2:])
		if *out == "" {
			*out = strings.TrimSuffix(path, ".harp") + ".wav"
		}
		settings := patch.Settings()
		settings.SampleRate = *rate
		prog, channels := settings.Render(patch.Out, *secs)
		fmt.Print(prog.Opt)
		write := dsl.WriteWAV
		if *float {
			write = dsl.WriteWAVFloat
		}
		if err := write(*out, int(*rate), channels...); err != nil {
			fail(err)
		}
		fmt.Printf("wrote %s: %d channels, %.1f seconds\n", *out, len(channels), *secs)
	case "play":
		fs := flag.NewFlagSet("play", flag.ExitOnError)
		secs := fs.Float64("seconds", 0, "stop after this many seconds (0: until Ctrl+C)")
		buf := fs.Int("buffer", 512, "samples per buffer: smaller is less latency, bigger is safer")
		rate := fs.Float64("rate", 44100, "sample rate of the sound card (48000 if that is what the system runs at: then nothing is resampled)")
		watch := fs.Bool("watch", true, "play the patch again when the file changes: what is the same goes on (oscillators, filters, reverbs, generators keep their state), what changed is new. A patch with a mistake is ignored")
		prof := fs.String("cpuprofile", "", "write a profile of where the time goes to this file (go tool pprof -top harp file)")
		patch, patchPath := load(fs, os.Args[2:])
		if *prof != "" {
			f, err := os.Create(*prof)
			if err != nil {
				fail(err)
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				fail(err)
			}
			defer func() { pprof.StopCPUProfile(); f.Close() }()
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		if *secs > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(*secs*float64(time.Second)))
			defer cancel()
		}
		eng := paengine.New(*rate, *buf)
		fmt.Printf("playing at %.0f Hz, buffer %d, %.0f bpm (Ctrl+C stops)\n", eng.GetSampleRate(), eng.GetBufferSize(), patch.BPM)
		settings := patch.Settings()
		var ready func(*dsl.Runner)
		if *watch {
			settings.NoOptimize = true // an optimized graph is another shape after a small edit, and what is the same could not be told
			path := patchPath
			ready = func(r *dsl.Runner) {
				go watchFile(ctx, path, patch, func(p *lang.Patch) (string, error) {
					st := p.Settings()
					st.NoOptimize = true
					if err := r.Swap(p.Out, st); err != nil {
						return "", err
					}
					kept, fresh := r.LastSwap()
					return fmt.Sprintf(" (%d nodes go on with their state, %d start again%s)", kept, fresh, restarted(r.LastRestarted())), nil
				}, os.Stdout, 200*time.Millisecond)
			}
			fmt.Println("watching", path, "(a saved change takes over while it plays)")
		}
		if err := settings.PlayWith(ctx, patch.Out, eng, ready); err != nil {
			fail(err)
		}
	case "compare":
		if err := compareCmd(os.Args[2:], os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "harp:", err)
			os.Exit(1)
		}
	case "cache":
		dir := lang.CacheDir()
		if dir == "" {
			fmt.Println("the mipmaps are not kept (HARP_CACHE=off)")
			break
		}
		if len(os.Args) > 2 && os.Args[2] == "clear" {
			n, err := lang.ClearCache()
			if err != nil {
				fail(err)
			}
			fmt.Printf("removed %d files from %s\n", n, dir)
			break
		}
		n, size := lang.CacheSize()
		fmt.Printf("%s: %d files, %.1f MB (harp cache clear removes them)\n", dir, n, float64(size)/1e6)
	case "presets":
		synth := ""
		if len(os.Args) > 2 {
			synth = os.Args[2]
		}
		if synth == "-md" {
			fmt.Print(lang.PresetsMarkdown())
			break
		}
		out, err := lang.Presets(synth)
		if err != nil {
			fail(err)
		}
		fmt.Print(out)
	case "settings":
		synth := ""
		if len(os.Args) > 2 {
			synth = os.Args[2]
		}
		if synth == "" || synth == "-md" {
			fmt.Print(lang.SettingsMarkdown())
			break
		}
		out, err := lang.SettingsOf(synth)
		if err != nil {
			fail(err)
		}
		fmt.Print(out)
	case "functions":
		if len(os.Args) > 2 && os.Args[2] == "-md" {
			fmt.Print(lang.FunctionsMarkdown())
			break
		}
		fmt.Print(lang.Functions())
	default:
		usage()
	}
}

// load reads the patch that is the first argument and takes the flags after it, and checks the patch:
// an error is shown with the line of the source it is in
func load(fs *flag.FlagSet, args []string) (*lang.Patch, string) {
	if len(args) == 0 || args[0] == "" || args[0][0] == '-' {
		usage()
	}
	path := args[0]
	_ = fs.Parse(args[1:])
	src, err := os.ReadFile(path)
	if err != nil {
		fail(err)
	}
	patch, err := lang.CompileDir(string(src), filepath.Dir(path))
	if err != nil {
		if e, ok := err.(*lang.Error); ok {
			fmt.Fprintf(os.Stderr, "%s:%s\n", path, e.Show(string(src)))
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "%s:%v\n", path, err)
		os.Exit(1)
	}
	return patch, path
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "harp:", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  harp check patch.harp
  harp render patch.harp [-o patch.wav] [-seconds 10] [-rate 44100] [-float]
  harp play patch.harp [-seconds 30] [-buffer 512] [-rate 44100]
  harp compare patch.harp [-sc code|file] [-seconds 10] [-rate 44100] [-sclang folder] [-stats] [-save folder]
  harp functions`)
	os.Exit(2)
}

// restarted says which nodes started again, as a list with the numbers: ": reverb, events.gate x3"
func restarted(names []string) string {
	if len(names) == 0 {
		return ""
	}
	count := map[string]int{}
	var order []string
	for _, n := range names {
		if count[n] == 0 {
			order = append(order, n)
		}
		count[n]++
	}
	var parts []string
	for _, n := range order {
		if count[n] > 1 {
			parts = append(parts, fmt.Sprintf("%s x%d", n, count[n]))
		} else {
			parts = append(parts, n)
		}
	}
	return ": " + strings.Join(parts, ", ")
}
