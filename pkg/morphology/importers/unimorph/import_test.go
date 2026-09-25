package unimorph_test

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/importers/unimorph"
	"github.com/amarin/gomorphy/pkg/morphology/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// miniTSV is a small, hand-checkable UniMorph-shaped fixture:
//   - "кот" has a row with form == lemma (nomn-like reading) plus one
//     inflected form ("кота") — form 0 comes from the real row.
//   - "мышь" has NO row with form == lemma at all — form 0 must be
//     synthesized.
//   - "я"/"меня" is a suppletive pair sharing one lemma, whose forms
//     share no literal prefix at all — exercises Q6's "lemma text always
//     feeds lcp()" requirement.
const miniTSV = "кот\tкот\tN;NOM;SG\n" +
	"кот\tкота\tN;ACC;SG\n" +
	"мышь\tмыши\tN;GEN;SG\n" +
	"я\tя\tPRO;NOM;SG\n" +
	"я\tменя\tPRO;ACC;SG\n"

func ruOptions() unimorph.Options {
	return unimorph.Options{Language: "ru"}
}

func TestImportFromTSV_BasicLemmasAndForms(t *testing.T) {
	d, err := unimorph.CompileFromTSV(strings.NewReader(miniTSV), ruOptions())
	require.NoError(t, err)
	require.Len(t, d.Words, 1, "small fixture must fit in one shard")

	for _, word := range []string{"кот", "кота", "мыши", "я", "меня"} {
		items := d.Words[0].SimilarItems(word, d.CharPolicy, d.Alphabet)
		assert.Greater(t, len(items), 0, "word %q must be found", word)
	}

	assert.Equal(t, "ru", d.Language)
	assert.Equal(t, "unimorph", d.TagSet.Name)
}

func TestImportFromTSV_SyntheticForm0(t *testing.T) {
	// "мышь" never appears as its own form in miniTSV — form 0 must be
	// synthesized with an empty tag, and "мыши" must still resolve to
	// the lemma "мышь" as its normal form.
	d, err := unimorph.CompileFromTSV(strings.NewReader(miniTSV), ruOptions())
	require.NoError(t, err)

	items := d.Words[0].SimilarItems("мыши", d.CharPolicy, d.Alphabet)
	require.Len(t, items, 1)
	require.Len(t, items[0].Values, 1)

	para, ok := paradigmFor(t, d, 0, items[0].Values[0])
	require.True(t, ok)
	require.Greater(t, para.Len(), 1, "мышь's paradigm must have more than the synthesized form 0")
}

func TestImportFromTSV_RealFormAtLemmaBecomesForm0(t *testing.T) {
	// "кот" has a real row with form == lemma ("кот\tкот\tN;NOM;SG") —
	// form 0 must carry that row's own tag, not an empty one.
	d, err := unimorph.CompileFromTSV(strings.NewReader(miniTSV), ruOptions())
	require.NoError(t, err)

	items := d.Words[0].SimilarItems("кот", d.CharPolicy, d.Alphabet)
	require.Len(t, items, 1)
	require.Len(t, items[0].Values, 1)

	para, ok := paradigmFor(t, d, 0, items[0].Values[0])
	require.True(t, ok)
	tagID := para.Tag(0)
	assert.Equal(t, "N;NOM;SG", d.TagSet.TagName(tagID))
}

func TestImportFromTSV_Suppletion(t *testing.T) {
	// "я"/"меня" share no literal prefix. lcp() over [lemma, "я", "меня"]
	// must still yield stem="" so both forms round-trip through the DAWG
	// key = the literal wordform text.
	d, err := unimorph.CompileFromTSV(strings.NewReader(miniTSV), ruOptions())
	require.NoError(t, err)

	for _, word := range []string{"я", "меня"} {
		items := d.Words[0].SimilarItems(word, d.CharPolicy, d.Alphabet)
		require.Len(t, items, 1, "word %q", word)
		assert.Equal(t, word, items[0].Key)
	}
}

func TestImportFromTSV_EmptyLemma(t *testing.T) {
	tsv := "\tбыстро\tADV\n"
	d, err := unimorph.CompileFromTSV(strings.NewReader(tsv), ruOptions())
	require.NoError(t, err)

	items := d.Words[0].SimilarItems("быстро", d.CharPolicy, d.Alphabet)
	require.Len(t, items, 1)
}

func TestImportFromTSV_MalformedLinesSkippedNotFatal(t *testing.T) {
	tsv := "кот\tкот\tN;NOM;SG\n" +
		"broken line with no tabs\n" +
		"too\tmany\tfields\there\n" +
		"дом\tдом\tN;NOM;SG\n"

	var malformed []int
	opts := ruOptions()
	opts.OnMalformed = func(lineNumber int, text string) {
		malformed = append(malformed, lineNumber)
	}

	d, err := unimorph.CompileFromTSV(strings.NewReader(tsv), opts)
	require.NoError(t, err)
	assert.Equal(t, []int{2, 3}, malformed)

	items := d.Words[0].SimilarItems("кот", d.CharPolicy, d.Alphabet)
	assert.Len(t, items, 1)
	items = d.Words[0].SimilarItems("дом", d.CharPolicy, d.Alphabet)
	assert.Len(t, items, 1)
}

func TestImportFromTSV_EmptyBundleAllowed(t *testing.T) {
	tsv := "кот\tкот\t\n"
	d, err := unimorph.CompileFromTSV(strings.NewReader(tsv), ruOptions())
	require.NoError(t, err)

	items := d.Words[0].SimilarItems("кот", d.CharPolicy, d.Alphabet)
	require.Len(t, items, 1)
	para, ok := paradigmFor(t, d, 0, items[0].Values[0])
	require.True(t, ok)
	assert.Equal(t, "", d.TagSet.TagName(para.Tag(0)))
}

func TestImportFromTSV_DuplicateRowWithinLemmaCollapses(t *testing.T) {
	tsv := "кот\tкота\tN;ACC;SG\n" +
		"кот\tкота\tN;ACC;SG\n" + // exact duplicate row
		"кот\tкот\tN;NOM;SG\n"

	d, err := unimorph.CompileFromTSV(strings.NewReader(tsv), ruOptions())
	require.NoError(t, err)

	items := d.Words[0].SimilarItems("кота", d.CharPolicy, d.Alphabet)
	require.Len(t, items, 1)
	assert.Len(t, items[0].Values, 1, "the duplicate row must not produce a second reading")
}

func TestImportFromTSV_UnsupportedLanguageErrors(t *testing.T) {
	_, err := unimorph.CompileFromTSV(strings.NewReader(miniTSV), unimorph.Options{Language: "en"})
	assert.Error(t, err)

	_, err = unimorph.CompileFromTSV(strings.NewReader(miniTSV), unimorph.Options{})
	assert.Error(t, err, "an empty Language must also be rejected, not silently default")
}

func TestImportFromTSV_EmptyInput(t *testing.T) {
	d, err := unimorph.CompileFromTSV(strings.NewReader(""), ruOptions())
	require.NoError(t, err)
	require.Len(t, d.Words, 1)
	assert.Nil(t, d.Words[0].SimilarItems("кот", d.CharPolicy, d.Alphabet))
}

// TestImportFromTSV_OversizedCharPolicyRejected verifies a caller-supplied
// Options.CharPolicy with more than 255 substitutions is rejected with a
// wrapped error at import time, instead of panicking later inside
// EncodeMeta (SaveTo/ContentHash) — see internal.ValidateCharPolicy.
func TestImportFromTSV_OversizedCharPolicyRejected(t *testing.T) {
	subs := make([]internal.Substitution, 256)
	for i := range subs {
		subs[i] = internal.Substitution{From: rune('a' + i), To: rune('A' + i)}
	}
	opts := unimorph.Options{Language: "ru", CharPolicy: internal.NewCharPolicy(subs...)}

	_, err := unimorph.CompileFromTSV(strings.NewReader(miniTSV), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "256")
}

// paradigmFor decodes value (a words.dawg payload: 2 bytes BE paraID +
// 2 bytes BE formIdx) into the shard's paradigm.
func paradigmFor(t *testing.T, d *internal.Dictionary, shard int, value []byte) (internal.Paradigm, bool) {
	t.Helper()
	require.GreaterOrEqual(t, len(value), 4)
	paraID := binary.BigEndian.Uint16(value[:2])
	if shard < 0 || shard >= len(d.Paradigms) || int(paraID) >= len(d.Paradigms[shard]) {
		return internal.Paradigm{}, false
	}
	return d.Paradigms[shard][paraID], true
}
