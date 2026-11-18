// Package sc runs SuperCollider: a piece of its code is rendered offline (in non-realtime mode: no sound card,
// faster than the sound lasts) into samples, to compare with what harp makes of the same patch.
package sc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"harp/dsl"
)

// Server is a SuperCollider that was found: the language (sclang) and the sound server (scsynth)
type Server struct {
	Sclang  string
	Scsynth string
}

// Find looks for SuperCollider: in dir if that is given, then in $HARP_SC and $SC_HOME/bin, then on the PATH, then in
// ~/.local/supercollider/bin (where building it from source into the home folder puts it), then in /usr/local/bin and /usr/bin
func Find(dir string) (*Server, error) {
	var dirs []string
	if dir != "" {
		dirs = append(dirs, dir)
	}
	if d := os.Getenv("HARP_SC"); d != "" {
		dirs = append(dirs, d)
	}
	if d := os.Getenv("SC_HOME"); d != "" {
		dirs = append(dirs, filepath.Join(d, "bin"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "supercollider", "bin"))
	}
	dirs = append(dirs, "/usr/local/bin", "/usr/bin")
	if lang, err := exec.LookPath("sclang"); err == nil {
		if synth, err := exec.LookPath("scsynth"); err == nil {
			return &Server{lang, synth}, nil
		}
	}
	for _, d := range dirs {
		lang, synth := filepath.Join(d, "sclang"), filepath.Join(d, "scsynth")
		if isExecutable(lang) && isExecutable(synth) {
			return &Server{lang, synth}, nil
		}
	}
	return nil, errors.New("SuperCollider was not found (sclang and scsynth): give the folder with -sclang, or set $HARP_SC to it")
}

func isExecutable(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir() && st.Mode()&0o111 != 0
}

// Render plays the code, a function in the way of { SinOsc.ar(440, 0, 0.2) ! 2 } (as the argument of .play), for seconds at
// the sample rate, and gives the samples of the two output channels. A function of one channel is heard
// on the left only, as in SuperCollider: the right channel is silent. The second result is what sclang printed.
// codeStartsAtLine is the line of the script that the code of the user starts at: SuperCollider says where a mistake is in
// the lines of the script, and the message is turned into the lines of the code (see errorLines)
const codeStartsAtLine = 6

func (s *Server) Render(code string, seconds, rate float64) ([][]float64, string, error) {
	return s.render(code, seconds, rate, -1)
}

// RenderSeed is Render with the random numbers of the code (ExpRand, coin, choose, the noises of the patch) started from a seed, so
// that a run can be repeated
func (s *Server) RenderSeed(code string, seconds, rate float64, seed int) ([][]float64, string, error) {
	if seed < 0 {
		return nil, "", errors.New("sc: a seed is 0 or more")
	}
	return s.render(code, seconds, rate, seed)
}

func (s *Server) render(code string, seconds, rate float64, seed int) ([][]float64, string, error) {
	if seconds <= 0 || rate <= 0 {
		return nil, "", errors.New("sc: seconds and the sample rate have to be above 0")
	}
	code = strings.TrimSpace(code)
	if !strings.HasPrefix(code, "{") {
		code = "{ " + code + " }"
	}
	dir, err := os.MkdirTemp("", "harp-sc-")
	if err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "out.wav")
	for _, p := range []string{s.Scsynth, out} {
		if strings.ContainsAny(p, `"\`) {
			return nil, "", fmt.Errorf("sc: the path %q has a quote or a backslash in it", p)
		}
	}
	// the work is in a try: a mistake in the code of the user (a message that is not understood, a wrong argument) is
	// printed with where it is, and sclang goes out at once, so that what it printed is not lost in a process that waits
	script := fmt.Sprintf(`(
Server.program = "%s";
Score.program = Server.program;
try { %s
~def = SynthDef(\harp_compare, { %s Out.ar(0, SynthDef.wrap(
%s
)) });
~score = Score([
    [0.0, ['/d_recv', ~def.asBytes]],
    [0.0, ['/s_new', \harp_compare, 1000, 0, 0]],
    [%g, ['/n_free', 1000]]
]);
~score.recordNRT(
    oscFilePath: "%s",
    outputFilePath: "%s",
    sampleRate: %g, headerFormat: "WAV", sampleFormat: "float",
    options: ServerOptions.new.numOutputBusChannels_(2).numInputBusChannels_(0),
    duration: %g,
    action: { "HARP_SC_DONE".postln; 0.exit }
);
} { |error| error.reportError; "HARP_SC_FAILED".postln; 0.exit };
)
`, s.Scsynth, seedStatement(seed), serverSeed(seed), code, seconds+1, filepath.Join(dir, "score.osc"), out, rate, seconds)
	scriptPath := filepath.Join(dir, "render.scd")
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		return nil, "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second+time.Duration(seconds*float64(time.Second)/2))
	defer cancel()
	// sclang writes to a pipe in large pieces, and what it says about a mistake would come too late (or not at all, as
	// it then waits for ever): stdbuf, where there is one, makes it write every line when it has it
	args := []string{s.Sclang, "-D", scriptPath}
	if sb, err := exec.LookPath("stdbuf"); err == nil {
		args = append([]string{sb, "-oL", "-eL"}, args...)
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) } // so that it can say what it was doing
	cmd.WaitDelay = 3 * time.Second                                          // and if it does not, it is killed
	cmd.Dir = dir
	buf := &lockedBuffer{}
	cmd.Stdout, cmd.Stderr = buf, buf
	if err := cmd.Start(); err != nil {
		return nil, "", fmt.Errorf("sc: starting sclang: %w", err)
	}
	finished := make(chan error, 1)
	go func() { finished <- cmd.Wait() }()
	// SuperCollider that finds an error says so and then waits for ever: when ERROR shows up, and the work
	// is not done, it is given a second to finish what it prints, and then stopped
	var runErr error
	var errorSeen time.Time
wait:
	for {
		select {
		case runErr = <-finished:
			break wait
		case <-time.After(50 * time.Millisecond):
			txt := buf.String()
			if strings.Contains(txt, "HARP_SC_DONE") {
				continue
			}
			if strings.Contains(txt, "ERROR") {
				if errorSeen.IsZero() {
					errorSeen = time.Now()
				} else if time.Since(errorSeen) > time.Second {
					cancel()
				}
			}
		}
	}
	text := buf.String()
	if !strings.Contains(text, "HARP_SC_DONE") || strings.Contains(text, "ERROR:") {
		// the code did not run: SuperCollider says why on lines that start with ERROR (and goes on waiting, so it was stopped)
		return nil, text, fmt.Errorf("SuperCollider did not render the code:\n%s", errorLines(text, runErr, ctx.Err()))
	}
	got, channels, err := dsl.ReadWAV(out)
	if err != nil {
		return nil, text, fmt.Errorf("sc: reading what SuperCollider rendered: %w", err)
	}
	if got != int(rate) {
		return nil, text, fmt.Errorf("sc: SuperCollider rendered at %d Hz, not %g", got, rate)
	}
	n := int(seconds * rate)
	for c := range channels {
		if len(channels[c]) > n {
			channels[c] = channels[c][:n]
		}
	}
	return channels, text, nil
}

var lineRef = regexp.MustCompile(`line \d+ char \d+`)

// errorLines picks what SuperCollider says is wrong out of everything it prints
func errorLines(text string, runErr, ctxErr error) string {
	var lines []string
	all := strings.Split(text, "\n")
	for i, l := range all {
		if strings.HasPrefix(l, "ERROR") || strings.Contains(l, "ERROR:") {
			end := min(i+8, len(all))
			lines = append(lines, all[i:end]...)
			break
		}
	}
	if len(lines) == 0 {
		tail := all
		if len(tail) > 10 {
			tail = tail[len(tail)-10:]
		}
		lines = tail
	}
	msg := strings.TrimSpace(strings.Join(lines, "\n"))
	msg = lineRef.ReplaceAllStringFunc(msg, func(m string) string {
		var l, c int
		fmt.Sscanf(m, "line %d char %d", &l, &c)
		if l >= codeStartsAtLine {
			return fmt.Sprintf("line %d char %d of your code", l-codeStartsAtLine+1, c)
		}
		return m
	})
	switch {
	case ctxErr != nil && !strings.Contains(text, "ERROR"):
		msg += "\n(stopped: it took too long)"
	case runErr != nil && msg == "":
		msg = runErr.Error()
	}
	return msg
}

// FromComments finds the SuperCollider code in the comment at the top of a patch: the lines of the first block of comment
// lines indented by three spaces or more that follows a comment line that starts with "SuperCollider". The
// .play at the end is left off. This is how the patches of examples/supercollider say what they are the
// translation of
func FromComments(src string) (string, bool) {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	i := 0
	for ; i < len(lines); i++ {
		l := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(l, "//") {
			if l == "" {
				continue
			}
			return "", false // the comment at the top is over
		}
		if strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(l, "//")), "SuperCollider") {
			i++
			break
		}
	}
	var code []string
	for ; i < len(lines); i++ {
		l := lines[i]
		t := strings.TrimSpace(l)
		if !strings.HasPrefix(t, "//") {
			break
		}
		body := strings.TrimPrefix(t, "//")
		switch {
		case strings.TrimSpace(body) == "":
			if len(code) > 0 {
				i = len(lines) // a blank comment line ends the block
			}
		case strings.HasPrefix(body, "   "):
			code = append(code, strings.TrimRight(body[3:], " \t"))
		default:
			if len(code) > 0 {
				i = len(lines)
			}
		}
	}
	text := strings.TrimSpace(strings.Join(code, "\n"))
	text = strings.TrimSuffix(strings.TrimSpace(strings.TrimSuffix(text, ";")), ".play")
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "{") || !strings.HasSuffix(text, "}") {
		return "", false
	}
	return text, true
}

// lockedBuffer is what the output of sclang is collected in, while it is also read
type lockedBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// serverSeed seeds the random numbers of the server too (the noises and the LFNoise are drawn there, not in sclang), on the line
// of the SynthDef
func serverSeed(seed int) string {
	if seed < 0 {
		return ""
	}
	return fmt.Sprintf("RandSeed.ir(1, %d);", seed)
}

// seedStatement is on the line of the try, so that the lines of the code of the user stay where they were
func seedStatement(seed int) string {
	if seed < 0 {
		return ""
	}
	return fmt.Sprintf("thisThread.randSeed = %d;", seed)
}
