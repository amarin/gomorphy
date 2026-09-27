package morphology

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// affixBuilderDict is a Builder dictionary whose paradigms have
// multi-rune suffixes: nouns кошка/кошки/кошкой/кошкам (stem «кошк»,
// form-0 suffix «а») and verbs читать/читали/читающий (stem «чита», «ть»), three
// lemmas each.
func affixBuilderDict(t *testing.T) *Dictionary {
	t.Helper()
	b := NewBuilder(BuilderOptions{})
	for _, s := range []string{"кошк", "мышк", "пешк"} {
		require.NoError(t, b.AddLemma(s+"а", "NOUN,femn,sing,nomn"))
		require.NoError(t, b.AddForm(s+"и", s+"а", "NOUN,femn,sing,gent"))
		require.NoError(t, b.AddForm(s+"ой", s+"а", "NOUN,femn,sing,ablt"))
		require.NoError(t, b.AddForm(s+"ам", s+"а", "NOUN,femn,plur,datv"))
	}
	for _, s := range []string{"чита", "ката", "мота"} {
		require.NoError(t, b.AddLemma(s+"ть", "INFN,impf"))
		require.NoError(t, b.AddForm(s+"ли", s+"ть", "VERB,impf,plur,past"))
		require.NoError(t, b.AddForm(s+"ющий", s+"ть", "PRTF,impf,pres,actv"))
	}
	d, err := b.Build()
	require.NoError(t, err)
	return d
}

// Short unknown words on a Builder dictionary (ruling R14): every
// predicted reading ends with its form's suffix, so Forms regenerates the
// lexeme with r at r.Form and the lemma ends with the form-0 suffix.
func TestPredictShortWordsKeepFormSuffix(t *testing.T) {
	d := affixBuilderDict(t)
	lemmaSuffix := map[string]string{"NOUN": "а", "INFN": "ть", "VERB": "ть", "PRTF": "ть"}
	predicted := 0
	for _, w := range []string{"и", "й", "ой", "м", "ам", "ли", "щий", "ющий", "дой", "дам", "пили", "поющий"} {
		for _, r := range d.Parse(w) {
			require.True(t, r.Predicted, "%s is not in the dictionary", w)
			predicted++
			forms := d.Forms(r)
			require.NotNil(t, forms, "%s: Forms(%+v)", w, r)
			assert.Equal(t, r.Word, forms[r.Form].Word, "%s: reading sits at its form", w)
			pos, _, _ := strings.Cut(r.Tag, ",")
			assert.True(t, strings.HasSuffix(r.Normal, lemmaSuffix[pos]), "%s: lemma %q", w, r.Normal)
		}
	}
	assert.NotZero(t, predicted)
	assert.Nil(t, d.Parse("й"), "«й» is shorter than every form suffix it could match")
	assert.Nil(t, d.Parse("м"))
}

// The runtime guard skips values whose form suffix the candidate word
// lacks, as found in Builder/merged files written before ruling R14.
func TestPredictSkipsValueWithoutFormSuffix(t *testing.T) {
	d := affixBuilderDict(t)
	good := d.Parse("дой")
	require.Len(t, good, 1)
	para, form := good[0].Para, good[0].Form // «ой», NOUN sing ablt

	value := make([]byte, 6)
	binary.BigEndian.PutUint16(value[0:2], 5)
	binary.BigEndian.PutUint16(value[2:4], para)
	binary.BigEndian.PutUint16(value[4:6], form)
	pred, err := internal.BuildDAWGWithValuesBytes([]string{"й", "ой"}, [][]byte{value, value})
	require.NoError(t, err)
	d.d.Prediction = []*internal.DAWG{pred}
	d.d.PredictionSharded = false

	assert.Nil(t, d.Parse("й"), "«й» does not end with «ой»")
	got := d.Parse("дой")
	require.Len(t, got, 1)
	assert.Equal(t, "да", got[0].Normal)
	assert.NotNil(t, d.Forms(got[0]))
}
