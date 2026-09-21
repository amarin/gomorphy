package internal

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file is the end-to-end cross-check joining Task 1
// (BuildDictionaryFromEntries) and Task 2 (BuildPrediction): it proves the
// (para, form) ids written into the prediction DAWG line up with the
// paradigms/affixes the engine's readingForm uses to reconstruct a
// wordform. The test lives in package internal, which must not import
// pkg/morphology, so both readingForm and predictForPrefix are mirrored
// verbatim from pkg/morphology/parse.go. Duplicating the formula is the
// point: a drift between the two packages is exactly what this test
// detects.

// predIntegrationReading mirrors the engine's morphology.Reading for the
// fields this test checks.
type predIntegrationReading struct {
	Word   string
	Normal string
	Tag    string
	Para   uint16
	Form   uint16
}

// predictionIntegrationCorpus builds a raw dictionary with two lemmas that
// share the suffix "а" so prediction keys are multi-valent. Paradigm ids
// assigned by first appearance:
//
//	0 — кошка: suffixes (а, и, ой)
//	1 — дело:  suffixes (о, а, ом)
//
// The two paradigms differ, so "а" (which both lemmas hit) carries
// payloads for two distinct (para, form) pairs.
func predictionIntegrationCorpus(t *testing.T) *Dictionary {
	t.Helper()
	entries := []BuildEntry{
		{Word: "кошка", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,nomn"},
		{Word: "кошки", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,gent"},
		{Word: "кошкой", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,ablt"},
		{Word: "дело", Lemma: "дело", Tag: "NOUN,inan,neut,sing,nomn"},
		{Word: "дела", Lemma: "дело", Tag: "NOUN,inan,neut,sing,gent"},
		{Word: "делом", Lemma: "дело", Tag: "NOUN,inan,neut,sing,ablt"},
	}
	d, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)
	require.Len(t, d.Words, 1, "prediction is only built for unsharded dictionaries")
	return d
}

// predictionIntegrationProductive is a locally-defined stand-in for the
// engine's productive predicate (which lives in pkg/morphology and cannot
// be imported here). Every corpus tag is productive.
func predictionIntegrationProductive(tag string) bool { return tag != "" }

// predIntegrationReadingForm duplicates the engine's
// (*Dictionary).readingForm formula (pkg/morphology/parse.go): normal form
// is prefix₀ + stem + suffix₀ for form≠0, otherwise the word itself.
func predIntegrationReadingForm(t *testing.T, d *Dictionary, word string, paraNum, form uint16) predIntegrationReading {
	t.Helper()
	require.Less(t, int(paraNum), len(d.Paradigms[0]), "para %d out of range", paraNum)
	para := d.Paradigms[0][paraNum]
	require.Less(t, int(form), para.Len(), "form %d out of range for para %d", form, paraNum)

	prefix := d.Prefixes[para.Prefix(int(form))]
	suffix := d.Suffixes[0][para.Suffix(int(form))]
	norm := word
	if form != 0 {
		stem := strings.TrimPrefix(word, prefix)
		stem = strings.TrimSuffix(stem, suffix)
		p0 := d.Prefixes[para.Prefix(0)]
		s0 := d.Suffixes[0][para.Suffix(0)]
		norm = p0 + stem + s0
	}

	return predIntegrationReading{
		Word:   word,
		Normal: norm,
		Tag:    d.TagSet.TagName(para.Tag(int(form))),
		Para:   paraNum,
		Form:   form,
	}
}

// predIntegrationSuffixSplits duplicates suffixSplits from parse.go:
// longest suffix first, wordStart/wordEnd pairs.
func predIntegrationSuffixSplits(word string, max int) [][2]string {
	rr := []rune(word)
	n := len(rr)
	if n == 0 {
		return nil
	}
	if max > n {
		max = n
	}
	out := make([][2]string, 0, max)
	for i := 1; i <= max; i++ {
		out = append(out, [2]string{string(rr[:n-i]), string(rr[n-i:])})
	}
	return out
}

// predIntegrationPredict mirrors the engine's predictForPrefix
// (pkg/morphology/parse.go:146-190): widen from the longest suffix split
// toward shorter ones, decode count/para/form payloads, stop once more than
// one attested reading has accumulated. It uses the same
// DAWG.SimilarItems lookup and the same raw (non-dense) prediction keys
// that the engine does.
func predIntegrationPredict(t *testing.T, d *Dictionary, word string) []predIntegrationReading {
	t.Helper()
	require.NotEmpty(t, d.Prediction, "no prediction DAWG built")
	splits := predIntegrationSuffixSplits(word, 5)
	if len(splits) == 0 {
		return nil
	}

	var readings []predIntegrationReading
	seen := make(map[string]bool)
	totalCount := 0

	for i := len(splits) - 1; i >= 0; i-- {
		wordStart, wordEnd := splits[i][0], splits[i][1]
		for _, it := range d.Prediction[0].SimilarItems(wordEnd, d.CharPolicy, nil) {
			for _, v := range it.Values {
				if len(v) < 6 {
					continue
				}
				count := int(binary.BigEndian.Uint16(v[:2]))
				para := binary.BigEndian.Uint16(v[2:4])
				form := binary.BigEndian.Uint16(v[4:6])

				if int(para) >= len(d.Paradigms[0]) || int(form) >= d.Paradigms[0][para].Len() {
					continue
				}
				tag := d.TagSet.TagName(d.Paradigms[0][para].Tag(int(form)))
				if !predictionIntegrationProductive(tag) {
					continue
				}
				totalCount += count

				r := predIntegrationReadingForm(t, d, wordStart+it.Key, para, form)
				key := r.Word + "\x00" + r.Normal + "\x00" + r.Tag
				if seen[key] {
					continue
				}
				seen[key] = true
				readings = append(readings, r)
			}
		}
		if totalCount > 1 {
			break
		}
	}
	return readings
}

// TestPredictionReadingFormIntegration is the cross-check: a predicted
// (para, form) triple, fed through the engine's readingForm math, recovers
// the lemma. The dense recompile is applied between BuildPrediction and the
// assertions, so the test also pins the ordering invariant
// BuildDictionaryFromEntries → BuildPrediction → RecompileDense.
func TestPredictionReadingFormIntegration(t *testing.T) {
	d := predictionIntegrationCorpus(t)

	require.NoError(t, BuildPrediction(d, predictionIntegrationProductive))
	require.Len(t, d.Prediction, 1, "one prediction DAWG for the single prefix (\"\")")

	require.NoError(t, RecompileDense(d))

	// RecompileDense densifies Words only; prediction keys stay raw UTF-8
	// text, which is why the engine looks them up with alphabet=nil. A
	// densified prediction would make Walk yield encoded bytes instead of
	// the suffix text.
	require.NotNil(t, d.Alphabet, "Words must have been densified")
	foundRawKey := false
	d.Prediction[0].Walk(func(k string, _ [][]byte) {
		if k == "елом" {
			foundRawKey = true
		}
	})
	assert.True(t, foundRawKey,
		"prediction keys must remain raw text after RecompileDense")

	tests := []struct {
		query  string
		normal string
		tag    string
		para   uint16
		form   uint16
		inDict bool
	}{
		// In-dictionary ablative forms, reached through the prediction DAWG.
		{query: "кошкой", normal: "кошка", tag: "NOUN,anim,femn,sing,ablt", para: 0, form: 2, inDict: true},
		{query: "делом", normal: "дело", tag: "NOUN,inan,neut,sing,ablt", para: 1, form: 2, inDict: true},
		// Out-of-dictionary forms: the same paradigm/suffix is predicted for
		// a novel stem, and readingForm still recovers the right lemma.
		{query: "ложкой", normal: "ложка", tag: "NOUN,anim,femn,sing,ablt", para: 0, form: 2, inDict: false},
		{query: "селом", normal: "село", tag: "NOUN,inan,neut,sing,ablt", para: 1, form: 2, inDict: false},
	}

	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			if !tc.inDict {
				assert.Empty(t, d.Words[0].SimilarItems(tc.query, d.CharPolicy, d.Alphabet),
					"%q must be absent from the dense Words DAWG", tc.query)
			}

			readings := predIntegrationPredict(t, d, tc.query)
			require.NotEmpty(t, readings, "prediction must yield a reading for %q", tc.query)

			want := predIntegrationReading{
				Word:   tc.query,
				Normal: tc.normal,
				Tag:    tc.tag,
				Para:   tc.para,
				Form:   tc.form,
			}
			assert.Contains(t, readings, want,
				"predicted (para=%d, form=%d) must decode to %q via readingForm math",
				tc.para, tc.form, tc.normal)
		})
	}
}

// TestPredictionReadingFormDiscriminates proves the integration assertions
// above are not vacuously satisfiable: feeding a wrong form id or a wrong
// paradigm id through the same readingForm math does NOT recover the lemma.
func TestPredictionReadingFormDiscriminates(t *testing.T) {
	d := predictionIntegrationCorpus(t)
	require.NoError(t, BuildPrediction(d, predictionIntegrationProductive))
	require.NoError(t, RecompileDense(d))

	// Form 0 of the "дело" paradigm is the lemma form, so its normal is the
	// wordform itself — not the lemma "дело". The integration test asserts
	// the predicted form 2, which it must distinguish from form 0.
	wrongForm := predIntegrationReadingForm(t, d, "делом", 1, 0)
	assert.NotEqual(t, "дело", wrongForm.Normal,
		"form 0 must not masquerade as the predicted form's normal")
	assert.Equal(t, "делом", wrongForm.Normal)

	// The "кошка" paradigm's form 2 suffix is "ой", which does not fit
	// "делом" — the reconstruction must not accidentally yield "дело".
	wrongPara := predIntegrationReadingForm(t, d, "делом", 0, 2)
	assert.NotEqual(t, "дело", wrongPara.Normal,
		"a wrong paradigm id must not recover the lemma")
}
