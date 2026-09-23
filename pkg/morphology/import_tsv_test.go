package morphology_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tsvFixture is the same small corpus builder_test.go's buildSmallDict uses
// (кот/кота, мышь/мыши), expressed as lemma-first TSV rows.
const tsvFixture = `кот	кот	NOUN,anim,masc,sing,nomn
кот	кота	NOUN,anim,masc,sing,gent
мышь	мышь	NOUN,anim,femn,sing,nomn
мышь	мыши	NOUN,anim,femn,sing,gent
`

// importTSVString runs ImportTSV over the given text and requires success.
func importTSVString(t *testing.T, s string) *morphology.Dictionary {
	t.Helper()
	d, err := morphology.ImportTSV(strings.NewReader(s), morphology.BuilderOptions{})
	require.NoError(t, err)
	require.NotNil(t, d)
	return d
}

// TestImportTSVHappyPath verifies a 3-column (lemma-first) stream parses back
// via Parse with the expected (Word, Normal, Tag) — the TSV equivalent of
// TestBuilderParse.
func TestImportTSVHappyPath(t *testing.T) {
	d := importTSVString(t, tsvFixture)

	readings := d.Parse("кот")
	require.Len(t, readings, 1)
	r := readings[0]
	assert.Equal(t, "кот", r.Word)
	assert.Equal(t, "кот", r.Normal)
	assert.Equal(t, "NOUN,anim,masc,sing,nomn", r.Tag)

	readings = d.Parse("кота")
	require.Len(t, readings, 1)
	r = readings[0]
	assert.Equal(t, "кота", r.Word)
	assert.Equal(t, "кот", r.Normal)
	assert.Equal(t, "NOUN,anim,masc,sing,gent", r.Tag)

	readings = d.Parse("мыши")
	require.Len(t, readings, 1)
	r = readings[0]
	assert.Equal(t, "мыши", r.Word)
	assert.Equal(t, "мышь", r.Normal)
	assert.Equal(t, "NOUN,anim,femn,sing,gent", r.Tag)
}

// TestImportTSVTwoColumns verifies a 2-column row (no tags) yields an empty
// tag, and a lemma row round-trips with Normal == Word.
func TestImportTSVTwoColumns(t *testing.T) {
	d := importTSVString(t, "дело\tдело\nдело\tделом\n")

	readings := d.Parse("дело")
	require.Len(t, readings, 1)
	assert.Equal(t, "дело", readings[0].Word)
	assert.Equal(t, "дело", readings[0].Normal)
	assert.Equal(t, "", readings[0].Tag)

	readings = d.Parse("делом")
	require.Len(t, readings, 1)
	assert.Equal(t, "делом", readings[0].Word)
	assert.Equal(t, "дело", readings[0].Normal)
	assert.Equal(t, "", readings[0].Tag)
}

// TestImportTSVAutoLemma verifies a missing lemma makes the wordform its own
// lemma (Normal == Word): both for a 2-column row and a 3-column row whose
// lemma column is empty.
func TestImportTSVAutoLemma(t *testing.T) {
	d := importTSVString(t, "\tкот\n\tмышь\tNOUN,anim,femn,sing,nomn\n")

	readings := d.Parse("кот")
	require.Len(t, readings, 1)
	assert.Equal(t, "кот", readings[0].Word)
	assert.Equal(t, "кот", readings[0].Normal)
	assert.Equal(t, "", readings[0].Tag)

	readings = d.Parse("мышь")
	require.Len(t, readings, 1)
	assert.Equal(t, "мышь", readings[0].Word)
	assert.Equal(t, "мышь", readings[0].Normal)
	assert.Equal(t, "NOUN,anim,femn,sing,nomn", readings[0].Tag)
}

// TestImportTSVCommentsAndBlankLines verifies blank lines and lines whose
// first non-space byte is '#' (comment lines) are skipped entirely.
func TestImportTSVCommentsAndBlankLines(t *testing.T) {
	input := `# header comment

   # indented comment
кот	кот	NOUN,anim,masc,sing,nomn

кот	кота	NOUN,anim,masc,sing,gent
`
	d := importTSVString(t, input)

	readings := d.Parse("кот")
	require.Len(t, readings, 1)
	assert.Equal(t, "NOUN,anim,masc,sing,nomn", readings[0].Tag)

	readings = d.Parse("кота")
	require.Len(t, readings, 1)
	assert.Equal(t, "кот", readings[0].Normal)
	assert.Equal(t, "NOUN,anim,masc,sing,gent", readings[0].Tag)
}

// TestImportTSVTrimsSurroundingSpaces verifies leading/trailing whitespace is
// trimmed from every field (tab is the only delimiter).
func TestImportTSVTrimsSurroundingSpaces(t *testing.T) {
	input := "  кот  \t  кот  \t  NOUN,anim,masc,sing,nomn  \n" +
		"кот\t кота \t NOUN,anim,masc,sing,gent \n"
	d := importTSVString(t, input)

	readings := d.Parse("кот")
	require.Len(t, readings, 1)
	assert.Equal(t, "кот", readings[0].Word)
	assert.Equal(t, "NOUN,anim,masc,sing,nomn", readings[0].Tag)
}

// TestImportTSVOneColumnError verifies a 1-column row is an error naming the
// offending line number.
func TestImportTSVOneColumnError(t *testing.T) {
	_, err := morphology.ImportTSV(strings.NewReader("кот\tкот\nодно\n"), morphology.BuilderOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 2", "error must name the offending line number")
}

// TestImportTSVTooManyColumnsError verifies a row with more than 3 columns is
// an error naming the offending line number.
func TestImportTSVTooManyColumnsError(t *testing.T) {
	_, err := morphology.ImportTSV(strings.NewReader("кот\tкот\tNOUN\textra\n"), morphology.BuilderOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 1", "error must name the offending line number")
}

// TestImportTSVEmptyWordError verifies a row with an empty wordform column is
// an error naming the offending line number, matching Builder.AddForm.
func TestImportTSVEmptyWordError(t *testing.T) {
	_, err := morphology.ImportTSV(strings.NewReader("кот\tкот\nкот\t \tNOUN\n"), morphology.BuilderOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 2", "error must name the offending line number")
}

// TestImportTSVPrediction verifies prediction is rebuilt and functional: an
// out-of-dictionary word with a productive ending ("ота", attested by
// "кота") gets a predicted reading with a reconstructed lemma — same shape as
// TestBuilderPrediction.
func TestImportTSVPrediction(t *testing.T) {
	d := importTSVString(t, tsvFixture)

	readings := d.Parse("пилота")
	require.NotEmpty(t, readings, "prediction must produce a reading for an OOV word")
	assert.Equal(t, "пилота", readings[0].Word)
	assert.Equal(t, "пилот", readings[0].Normal)
	assert.Equal(t, "NOUN,anim,masc,sing,gent", readings[0].Tag)
}

// TestImportTSVSaveOpenRoundTrip verifies a TSV-built dictionary round-trips
// through SaveTo/Open with identical Parse results.
func TestImportTSVSaveOpenRoundTrip(t *testing.T) {
	d := importTSVString(t, tsvFixture)

	out := filepath.Join(t.TempDir(), "tsv.dat")
	require.NoError(t, d.SaveTo(out))

	got, err := morphology.Open(out)
	require.NoError(t, err)
	defer func() { require.NoError(t, got.Close()) }()

	cmpSnapshots(t,
		readingsSnapshot(d, "кот", "кота", "мышь", "мыши"),
		readingsSnapshot(got, "кот", "кота", "мышь", "мыши"),
	)
}
