package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"harp/lang"
)

type swaps struct {
	mu   sync.Mutex
	outs []*lang.Patch
	fail error
}

func (s *swaps) swap(p *lang.Patch) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail != nil {
		return "", s.fail
	}
	s.outs = append(s.outs, p)
	return " (test)", nil
}

func (s *swaps) count() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.outs) }

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatalf("timed out waiting for %s", what)
}

// write changes the file so that its stamp is different, whatever the resolution of the clock of the file system
func write(t *testing.T, path, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Add(time.Duration(len(src)) * time.Millisecond)
	os.Chtimes(path, now, now)
}

func TestWatchTakesOverGoodChangesAndIgnoresBadOnes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.harp")
	write(t, path, "out = sine(220)\n")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first, err := lang.Compile("out = sine(220)\n")
	if err != nil {
		t.Fatal(err)
	}
	var s swaps
	var log bytes.Buffer
	var logMu sync.Mutex
	done := make(chan struct{})
	go func() {
		watchFile(ctx, path, first, s.swap, &syncWriter{&log, &logMu}, 10*time.Millisecond)
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	if s.count() != 0 {
		t.Fatal("a swap without a change")
	}
	write(t, path, "out = sine(330)\n")
	waitFor(t, "the first change", func() bool { return s.count() == 1 })
	// a mistake: it is told, with its place, and nothing is swapped
	write(t, path, "out = sine(330\n")
	waitFor(t, "the message about the mistake", func() bool {
		logMu.Lock()
		defer logMu.Unlock()
		return strings.Contains(log.String(), "the old patch plays on")
	})
	if s.count() != 1 {
		t.Errorf("a patch with a mistake was swapped in: %d swaps", s.count())
	}
	logMu.Lock()
	if !strings.Contains(log.String(), "p.harp:2:1") {
		t.Errorf("the message has no place: %q", log.String())
	}
	logMu.Unlock()
	// and a good one again
	write(t, path, "out = sine(440).gain(0.5)\n")
	waitFor(t, "the change after the mistake", func() bool { return s.count() == 2 })
	// a patch that the runner refuses (another number of channels) is told about, and the old one plays on
	s.mu.Lock()
	s.fail = os.ErrInvalid
	s.mu.Unlock()
	write(t, path, "out = sine(550).dup()\n")
	waitFor(t, "the message about the refusal", func() bool {
		logMu.Lock()
		defer logMu.Unlock()
		return strings.Count(log.String(), "the old patch plays on") >= 2
	})
	cancel()
	<-done
}

type syncWriter struct {
	b  *bytes.Buffer
	mu *sync.Mutex
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}
