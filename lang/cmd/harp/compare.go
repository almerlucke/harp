package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"harp/dsl"
	"harp/lang"
	"harp/lang/compare"
	"harp/lang/sc"
)

// compareCmd renders a patch in harp, renders the SuperCollider code that it is the translation of in SuperCollider (offline),
// and says how much the two are the same sound
func compareCmd(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	scArg := fs.String("sc", "", "the SuperCollider code, a function like { SinOsc.ar(440, 0, 0.2) ! 2 }, or the path of a file with it (default: the code in the comment at the top of the patch, after the line that starts with // SuperCollider)")
	secs := fs.Float64("seconds", 10, "how many seconds to render and compare")
	rate := fs.Float64("rate", 44100, "sample rate")
	dir := fs.String("sclang", "", "the folder with sclang and scsynth (default: $HARP_SC, the PATH, ~/.local/supercollider/bin)")
	stats := fs.Bool("stats", false, "compare only what holds for the sound in the long run (the level and the spectrum), not the waves: for patches with noise or dust")
	ref := fs.String("ref", "", "a WAV file that SuperCollider made (of a pattern, for example) to compare with, instead of rendering code: it is compared as it is, from the start")
	seeds := fs.Int("seeds", 0, "for a patch that is made by chance when it is built: render this many seeds (1, 2, ...) in both and compare the sort of patch they make (the distributions of the level, the brightness and the bands), not one patch with another")
	save := fs.String("save", "", "a folder to write both renderings in, as 16 bit WAV files to listen to")
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return errors.New("usage: harp compare patch.harp [-sc code|file] [-seconds 10] [-rate 44100] [-sclang folder] [-stats] [-seeds n] [-ref file.wav] [-save folder]")
	}
	path := args[0]
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *secs <= 0 || *rate <= 0 {
		return errors.New("-seconds and -rate have to be above 0")
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	patch, err := lang.CompileDir(string(src), filepath.Dir(path))
	if err != nil {
		if e, ok := err.(*lang.Error); ok {
			return fmt.Errorf("%s:%s", path, e.Show(string(src)))
		}
		return fmt.Errorf("%s: %w", path, err)
	}

	code := *scArg
	switch {
	case *ref != "":
		code = "(the file " + *ref + ")"
	case code == "":
		var ok bool
		if code, ok = sc.FromComments(string(src)); !ok {
			return fmt.Errorf("%s has no SuperCollider code in its comment (a comment line that starts with // SuperCollider, and below it the code, indented by three spaces): give it with -sc", path)
		}
	default:
		if b, err := os.ReadFile(code); err == nil { // a file
			code = string(b)
		}
	}

	var server *sc.Server
	if *ref == "" {
		var err error
		if server, err = sc.Find(*dir); err != nil {
			return err
		}
	}
	if *seeds < 0 || *seeds == 1 {
		return errors.New("-seeds is 2 or more (10 or more to say something)")
	}
	if *seeds > 0 && *ref != "" {
		return errors.New("-seeds renders SuperCollider for every seed: it can't be used with -ref")
	}
	if *seeds > 0 {
		return compareSeeds(out, path, string(src), code, server, *seeds, *secs, *rate)
	}
	settings := patch.Settings()
	settings.SampleRate = *rate
	_, harpOut := settings.Render(patch.Out, *secs)
	var scOut [][]float64
	if *ref != "" {
		r, ch, err := dsl.ReadWAV(*ref)
		if err != nil {
			return err
		}
		if float64(r) != *rate {
			return fmt.Errorf("%s is at %d Hz, -rate is %g", *ref, r, *rate)
		}
		for i := range ch {
			ch[i] = ch[i][:min(len(ch[i]), len(harpOut[0]))]
		}
		scOut = ch
	} else if scOut, _, err = server.Render(code, *secs, *rate); err != nil {
		return err
	}

	fmt.Fprintf(out, "%s\nSuperCollider: %s\n", path, oneLine(code))
	if strings.Contains(code, "Noise") || strings.Contains(code, "Dust") || strings.Contains(code, "Rand") {
		if !*stats {
			fmt.Fprintln(out, "(this patch has a random source: two random sources are never the same wave; -stats compares the long run only)")
		}
	}
	rep := compare.Compare(scOut, harpOut, compare.Options{Rate: *rate, Statistical: *stats, NameA: "SuperCollider", NameB: "harp"})
	fmt.Fprint(out, "\n", rep)

	if *save != "" {
		if err := os.MkdirAll(*save, 0o755); err != nil {
			return err
		}
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		files := map[string][][]float64{base + "_SC.wav": scOut, base + "_harp.wav": harpOut}
		for name, ch := range files {
			if err := dsl.WriteWAV(filepath.Join(*save, name), int(*rate), ch...); err != nil {
				return err
			}
		}
		fmt.Fprintf(out, "\nwritten, to listen to: %s and %s in %s\n", base+"_SC.wav", base+"_harp.wav", *save)
	}
	return nil
}

// oneLine is the code on one line, for the head of the report
func oneLine(code string) string {
	f := strings.Fields(code)
	s := strings.Join(f, " ")
	if len(s) > 150 {
		s = s[:147] + "..."
	}
	return s
}

var seedLine = regexp.MustCompile(`(?m)^seed\s*=.*$`)

// withSeed is the source with its seed set to n (or a seed added, if it had none)
func withSeed(src string, n int) string {
	line := fmt.Sprintf("seed = %d", n)
	if seedLine.MatchString(src) {
		return seedLine.ReplaceAllLiteralString(src, line)
	}
	return line + "\n" + src
}

// compareSeeds renders the seeds 1 to n, in harp and in SuperCollider, a few at a time, and compares the two sets
func compareSeeds(out io.Writer, path, src, code string, server *sc.Server, n int, secs, rate float64) error {
	harpRuns := make([][][]float64, n)
	scRuns := make([][][]float64, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4) // sclang and scsynth are heavy to start: a few at a time
	var mu sync.Mutex             // the seed of harp is one for the whole process: a harp patch is built one at a time
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			seed := i + 1
			mu.Lock()
			p, err := lang.CompileDir(withSeed(src, seed), filepath.Dir(path))
			mu.Unlock()
			if err != nil {
				errs[i] = fmt.Errorf("seed %d: %w", seed, err)
				return
			}
			st := p.Settings()
			st.SampleRate = rate
			_, harpRuns[i] = st.Render(p.Out, secs)
			scRuns[i], _, errs[i] = server.RenderSeed(code, secs, rate, seed)
			if errs[i] != nil {
				errs[i] = fmt.Errorf("seed %d: %w", seed, errs[i])
			}
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "%s\nSuperCollider: %s\n\n", path, oneLine(code))
	fmt.Fprint(out, compare.Seeds(scRuns, harpRuns, compare.Options{Rate: rate, NameA: "SuperCollider", NameB: "harp"}))
	return nil
}
