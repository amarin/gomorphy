package morphology

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImportTSVDenseSourceAndPrediction white-box check that a TSV-built
// dictionary is dense by default (Alphabet set), carries a rebuilt prediction
// DAWG (len(Prediction) == 1 for the unsharded case), and honors a
// caller-supplied Source in BuildInfo, with TagSet name "tsv" — a dict whose
// Source the caller overrides still is a TSV dict (the default Source
// behavior is locked in TestImportTSVDefaultSource).
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
	assert.Equal(t, "custom", d.Info().Source, "a caller-supplied Source is honored")
	assert.Equal(t, "tsv", d.TagSetName())
	assert.Equal(t, "ru", d.Language())
}

// TestImportTSVDefaultSource locks ImportTSV's Source default: with an empty
// BuilderOptions.Source the result reports "tsv".
func TestImportTSVDefaultSource(t *testing.T) {
	d, err := ImportTSV(strings.NewReader("кот\tкот\tNOUN,anim,masc,sing,nomn\n"), BuilderOptions{})
	require.NoError(t, err)
	require.NotNil(t, d)

	require.NotNil(t, d.Info())
	assert.Equal(t, "tsv", d.Info().Source, "Source defaults to \"tsv\" when the caller passes none")
}
