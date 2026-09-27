package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type wordTag struct{ Word, Tag string }

func wordTags(rs []morphology.Reading) []wordTag {
	out := make([]wordTag, len(rs))
	for i, r := range rs {
		out[i] = wordTag{r.Word, r.Tag}
	}
	return out
}

func TestFormsPyMorphy(t *testing.T) {
	d := parseDict(t)
	rs := d.Parse("кота")
	require.Len(t, rs, 1)

	forms := d.Forms(rs[0])
	assert.Equal(t, []wordTag{
		{"кот", "NOUN,anim,masc,sing,nomn"},
		{"кота", "NOUN,anim,masc,sing,gent"},
	}, wordTags(forms))
	for i, f := range forms {
		assert.Equal(t, uint16(i), f.Form)
		assert.Equal(t, "кот", f.Normal)
		assert.Equal(t, rs[0].Para, f.Para)
		assert.False(t, f.Predicted)
	}
}

// «кот» is a NOUN (paradigm 0, two forms) and a VERB (paradigm 1, one form).
func TestFormsFollowsTheGivenHomonym(t *testing.T) {
	d := parseDict(t)
	for _, r := range d.Parse("кот") {
		forms := d.Forms(r)
		switch r.Tag {
		case "VERB,impf,trans":
			assert.Equal(t, []wordTag{{"кот", "VERB,impf,trans"}}, wordTags(forms))
		default:
			assert.Len(t, forms, 2)
		}
	}
}

// prefixedFixture adds paradigm 3 = {form 0: stem, form 1: "наи"+stem}
// (prefix id 2 in paradigm-prefixes.json ["","по","наи"]).
func prefixedFixture(t *testing.T) *morphology.Dictionary {
	t.Helper()
	m := map[string]uint32{}
	stdWords(m)
	addWord(m, "лучший", 3, 0)
	addWord(m, "наилучший", 3, 1)
	dir := buildFixtureDir(t, m, nil, nil)
	writeParadigms(t, dir, [][]uint16{
		{0, 1, 0, 1, 0, 0},
		{0, 2, 0},
		{0, 1, 0, 1, 0, 0},
		{0, 0, 0, 1, 0, 2}, // suffixes 0,0 | tags 0,1 | prefixes 0,2
	})
	d, err := morphology.OpenPyMorphy(dir)
	require.NoError(t, err)
	return d
}

func TestFormsWithParadigmPrefix(t *testing.T) {
	d := prefixedFixture(t)
	rs := d.Parse("наилучший")
	require.Len(t, rs, 1)

	forms := d.Forms(rs[0])
	require.Len(t, forms, 2)
	assert.Equal(t, "лучший", forms[0].Word)
	assert.Equal(t, "наилучший", forms[1].Word)
	assert.Equal(t, "лучший", forms[1].Normal)
}

func TestFormsOfPredictedReading(t *testing.T) {
	d := predictionFixture(t)
	rs := d.Parse("котёнка")
	require.Len(t, rs, 1)
	require.True(t, rs[0].Predicted)

	forms := d.Forms(rs[0])
	assert.Equal(t, []wordTag{
		{"котёнк", "NOUN,anim,masc,sing,nomn"},
		{"котёнка", "NOUN,anim,masc,sing,gent"},
	}, wordTags(forms))
	for _, f := range forms {
		assert.True(t, f.Predicted)
	}
}

// After item J the verb lexeme of «знать» no longer contains «знати».
func TestFormsBuilderLexeme(t *testing.T) {
	d := buildFromTriples(t,
		[3]string{"знать", "знать", "NOUN,inan,femn,sing,nomn"},
		[3]string{"знати", "знать", "NOUN,inan,femn,sing,gent"},
		[3]string{"знать", "знать", "INFN,impf,tran"},
		[3]string{"знаю", "знать", "VERB,impf,tran,sing,1per,pres,indc"},
	)
	rs := d.Parse("знаю")
	require.Len(t, rs, 1)
	assert.Equal(t, []wordTag{
		{"знать", "INFN,impf,tran"},
		{"знаю", "VERB,impf,tran,sing,1per,pres,indc"},
	}, wordTags(d.Forms(rs[0])))
}

func TestFormsRejectsForeignReadings(t *testing.T) {
	d := parseDict(t)
	assert.Nil(t, d.Forms(morphology.Reading{}), "zero reading")
	assert.Nil(t, d.Forms(morphology.Reading{Word: "кот", Para: 99}), "unknown paradigm")
	assert.Nil(t, d.Forms(morphology.Reading{Word: "кот", Para: 0, Form: 5}), "form out of range")
	assert.Nil(t, d.Forms(morphology.Reading{Word: "кот", Para: 0, Form: 1}), "«кот» lacks form 1's suffix «а»")

	var nilDict *morphology.Dictionary
	assert.Nil(t, nilDict.Forms(d.Parse("кота")[0]))
}

func koshkaDict(t *testing.T) *morphology.Dictionary {
	t.Helper()
	return buildFromTriples(t,
		[3]string{"кошка", "кошка", "NOUN,anim,femn,sing,nomn"},
		[3]string{"кошки", "кошка", "NOUN,anim,femn,sing,gent"},
		[3]string{"кошке", "кошка", "NOUN,anim,femn,sing,datv"},
		[3]string{"кошки", "кошка", "NOUN,anim,femn,plur,nomn"},
		[3]string{"кошек", "кошка", "NOUN,anim,femn,plur,gent"},
	)
}

func TestInflect(t *testing.T) {
	d := koshkaDict(t)
	rs := d.Parse("кошка")
	require.Len(t, rs, 1)
	r := rs[0]

	// sing,nomn → plur: plur,nomn differs in 2 grammemes (sing/plur),
	// plur,gent in 4.
	assert.Equal(t, []wordTag{
		{"кошки", "NOUN,anim,femn,plur,nomn"},
		{"кошек", "NOUN,anim,femn,plur,gent"},
	}, wordTags(d.Inflect(r, "plur")))

	assert.Equal(t, []wordTag{
		{"кошки", "NOUN,anim,femn,sing,gent"},
		{"кошек", "NOUN,anim,femn,plur,gent"},
	}, wordTags(d.Inflect(r, "gent")))

	assert.Equal(t, []wordTag{{"кошек", "NOUN,anim,femn,plur,gent"}},
		wordTags(d.Inflect(r, "plur", "gent")))
	assert.Equal(t, "кошке", d.Inflect(r, "datv")[0].Word)
	assert.Nil(t, d.Inflect(r, "ablt"), "no such form")

	all := d.Inflect(r)
	require.Len(t, all, 5)
	assert.Equal(t, "кошка", all[0].Word, "r itself ranks first")
}

func TestInflectFromNonLemmaForm(t *testing.T) {
	d := koshkaDict(t)
	rs := d.Parse("кошек")
	require.Len(t, rs, 1)
	got := d.Inflect(rs[0], "sing", "nomn")
	require.Len(t, got, 1)
	assert.Equal(t, "кошка", got[0].Word)
	assert.Equal(t, "кошка", got[0].Normal)
}

func TestInflectForeignReading(t *testing.T) {
	d := koshkaDict(t)
	assert.Nil(t, d.Inflect(morphology.Reading{}, "gent"))
}
