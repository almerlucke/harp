package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"harp/lang"
)

// watchFile compiles the patch again whenever the file changes, and gives the result to swap, which makes it take over from the patch that
// plays. A patch that does not compile is told about (with the place of the mistake) and ignored: what plays goes on. It returns when ctx
// is done
func watchFile(ctx context.Context, path string, current *lang.Patch, swap func(*lang.Patch) (string, error), out io.Writer, every time.Duration) {
	last, _ := stamp(path)
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		now, err := stamp(path)
		if err != nil || now == last {
			continue // gone for a moment (an editor that saves by replacing the file) or the same
		}
		// the file may be half written: it is read again a moment later if it does not compile and has changed since
		time.Sleep(30 * time.Millisecond)
		src, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		last = now
		patch, err := lang.Recompile(string(src), current)
		if err != nil {
			if e, ok := err.(*lang.Error); ok {
				fmt.Fprintf(out, "%s:%s\n(not changed: the old patch plays on)\n", path, e.Show(string(src)))
			} else {
				fmt.Fprintf(out, "%s: %v\n(not changed: the old patch plays on)\n", path, err)
			}
			continue
		}
		what, err := swap(patch)
		if err != nil {
			fmt.Fprintf(out, "%s: %v\n(not changed: the old patch plays on)\n", path, err)
			continue
		}
		current = patch // the next change takes over from this one
		fmt.Fprintf(out, "%s: changed, it takes over at the next step%s\n", path, what)
	}
}

// stamp says whether a file has changed: its time and size
func stamp(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d %d", fi.ModTime().UnixNano(), fi.Size()), nil
}
