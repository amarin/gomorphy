package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProgressReporter_NonInteractive_NoCarriageReturn(t *testing.T) {
	var buf strings.Builder
	report := newProgressReporterTo(&buf, false)
	report(1, 10)
	report(10, 10)

	out := buf.String()
	assert.NotContains(t, out, "\r")
	assert.Equal(t, 2, strings.Count(out, "\n"))
}

func TestNewProgressReporter_Interactive_UsesCarriageReturn(t *testing.T) {
	var buf strings.Builder
	report := newProgressReporterTo(&buf, true)
	report(1, 10)
	report(10, 10)

	out := buf.String()
	assert.Contains(t, out, "\r")
	assert.Equal(t, 1, strings.Count(out, "\n"), "interactive mode prints exactly one trailing newline, once done")
}
