package morphology_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLemmaKota(t *testing.T) {
	d := parseDict(t)

	refs := d.Lemma("кота")
	require.Len(t, refs, 1)

	assert.Equal(t, "кот", refs[0].Normal)
	assert.Equal(t, "NOUN,anim,masc,sing,nomn", refs[0].Tag)
	assert.Equal(t, uint16(0), refs[0].Para)
}

func TestLemmaHomonyms(t *testing.T) {
	d := parseDict(t)

	refs := d.Lemma("кот")
	require.Len(t, refs, 2)
	assert.Equal(t, "кот", refs[0].Normal)
	assert.Equal(t, "NOUN,anim,masc,sing,nomn", refs[0].Tag)
	assert.Equal(t, "кот", refs[1].Normal)
	assert.Equal(t, "VERB,impf,trans", refs[1].Tag)
}

func TestLemmaUnknown(t *testing.T) {
	d := parseDict(t)
	assert.Nil(t, d.Lemma("неттакогослова"))
}
