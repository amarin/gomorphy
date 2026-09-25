package morphology_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReturnedStringsSurviveClose reads every string the query API returned
// after Close has unmapped the file. If any of them pointed into the
// mapping, this would crash with SIGSEGV/SIGBUS. Planning-time audit: only
// the DAWG unit/guide arrays alias the mapping; suffixes, prefixes, tags,
// info and matched keys are heap copies.
func TestReturnedStringsSurviveClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "small.dat")
	require.NoError(t, buildSmallDict(t).SaveTo(path))

	d, err := morphology.Open(path)
	require.NoError(t, err)

	readings := append(d.Parse("кота"), d.Parse("бота")...)
	lemmas := d.Lemma("мыши")
	fuzzy := d.FuzzyTop("кот", 5)
	info := d.Info()
	names := []string{d.Language(), d.TagSetName(), d.ContentHash()}
	require.NoError(t, d.Close())

	var sb strings.Builder
	for _, r := range readings {
		sb.WriteString(r.Word + r.Normal + r.Tag)
	}
	for _, l := range lemmas {
		sb.WriteString(l.Normal + l.Tag)
	}
	for _, m := range fuzzy {
		sb.WriteString(m.Word)
	}
	sb.WriteString(info.Source + info.LibraryVersion)
	for _, n := range names {
		sb.WriteString(n)
	}

	out := sb.String()
	assert.Contains(t, out, "кота")
	assert.Contains(t, out, "мышь")
	assert.Contains(t, out, "NOUN,anim")
}
