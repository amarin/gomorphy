package morphology

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImportTSVDenseSourceAndPrediction white-box check that a TSV-built
// dictionary is dense by default (Alphabet set), carries a rebuilt prediction
// DAWG (len(Prediction) == 1 for the unsharded case), and is stamped with
// Source "tsv" and TagSet name "tsv" — regardless of what the caller put in
// BuilderOptions (TSV import always yields a TSV-sourced dictionary, matching
// Builder's "builder" default).
func TestImportTSVDenseSourceAndPrediction(t *testing.T) {
	d, err := ImportTSV(
		strings.NewReader("кот\tкот\tNOUN,anim,masc,sing,nomn\nкот\tкота\tNOUN,anim,masc,sing,gent\n"),
		BuilderOptions{Source: "custom"},
	)
	require.NoError(t, err)
	require.NotNil(t, d)
	require.NotNil(t, d.d)

	assert.NotNil(t, d.d.Alphabet, "output must be dense by default")
	assert.Len(t, d.d.Prediction, 1, "prediction must be rebuilt by default")

	require.NotNil(t, d.Info())
	assert.Equal(t, "tsv", d.Info().Source, "Source is forced to \"tsv\", overriding the caller's BuilderOptions")
	assert.Equal(t, "tsv", d.TagSetName())
	assert.Equal(t, "ru", d.Language())
}
