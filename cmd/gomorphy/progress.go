package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// isInteractive reports whether stdout is a terminal - used to choose
// between newProgressReporter's redraw-in-place and one-line-per-step
// output modes.
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// newProgressReporter returns a progress callback (the shape already used
// by opencorpora.CompileFromXML's progress parameter) shaped for the
// caller's output mode: interactive redraws one line in place;
// non-interactive prints one line per call, no control characters - safe
// for a log file or a pipe.
func newProgressReporter(interactive bool) func(processed, total int) {
	return newProgressReporterTo(os.Stdout, interactive)
}

// newProgressReporterTo is newProgressReporter with an explicit writer,
// for testing without touching the real os.Stdout.
func newProgressReporterTo(w io.Writer, interactive bool) func(processed, total int) {
	if !interactive {
		return func(processed, total int) {
			_, _ = fmt.Fprintf(w, "%d/%d\n", processed, total)
		}
	}
	return func(processed, total int) {
		_, _ = fmt.Fprintf(w, "\r%d/%d", processed, total)
		if total > 0 && processed >= total {
			_, _ = fmt.Fprintln(w)
		}
	}
}
