package tagmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenizeOpenCorporaIntSplitsSpaceThenComma(t *testing.T) {
	// Real example from a pymorphy2 gramtab-opencorpora-int.json, quoted
	// in docs/todo.md's "Универсальный маппинг тегов между словарями"
	// section.
	assert.Equal(t,
		[]string{"NOUN", "anim", "masc", "sing", "nomn"},
		tokenizeOpenCorporaInt("NOUN,anim,masc sing,nomn"))
}

func TestTokenizeOpenCorporaIntSingleGroupNoSpace(t *testing.T) {
	assert.Equal(t,
		[]string{"NOUN", "anim", "masc"},
		tokenizeOpenCorporaInt("NOUN,anim,masc"))
}

func TestTokenizeOpenCorporaIntEmptyTagIsNil(t *testing.T) {
	assert.Nil(t, tokenizeOpenCorporaInt(""))
}

func TestOpenCorporaIntTableCoversKotNominativeSingular(t *testing.T) {
	b := buildBundle(tokenizeOpenCorporaInt("NOUN,anim,masc sing,nomn"), openCorporaIntTable)

	assert.Equal(t, []Feature{
		{DimPartOfSpeech, "N"},
		{DimAnimacy, "ANIM"},
		{DimCase, "NOM"},
		{DimNumber, "SG"},
		{DimGender, "MASC"},
	}, b.Features)
	assert.Nil(t, b.Unmapped)
}

func TestOpenCorporaAndOpenCorporaIntAgreeOnKotNominativeSingular(t *testing.T) {
	oc := buildBundle(tokenizeOpenCorpora("NOUN,anim,masc,sing,nomn"), openCorporaTable)
	pm := buildBundle(tokenizeOpenCorporaInt("NOUN,anim,masc sing,nomn"), openCorporaIntTable)

	assert.Equal(t, oc.Features, pm.Features)
}
