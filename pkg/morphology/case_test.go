package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tagGeoxNomn = "NOUN,inan,femn,Sgtm,Geox,sing,nomn"
	tagGeoxGent = "NOUN,inan,femn,Sgtm,Geox,sing,gent"
)

// Before the fix «Москва» was stored verbatim and Parse (which lower-cases
// its input) only reached it through prediction — see the plan's
// "Planning-time findings" #2. The assertions on Predicted are the point.
func TestBuilderLowercasesWordAndLemma(t *testing.T) {
	b := morphology.NewBuilder(morphology.BuilderOptions{})
	require.NoError(t, b.AddLemma("Москва", tagGeoxNomn))
	require.NoError(t, b.AddForm("Москвы", "Москва", tagGeoxGent))
	d, err := b.Build()
	require.NoError(t, err)

	for _, q := range []string{"москва", "Москва", "МОСКВА"} {
		rs := d.Parse(q)
		require.Len(t, rs, 1, q)
		assert.Equal(t, "москва", rs[0].Word, q)
		assert.Equal(t, "москва", rs[0].Normal, q)
		assert.Equal(t, tagGeoxNomn, rs[0].Tag, "tags are stored verbatim")
		assert.False(t, rs[0].Predicted, q)
		assert.True(t, d.IsKnown(q), q)
	}

	rs := d.Parse("Москвы")
	require.Len(t, rs, 1)
	assert.Equal(t, "москва", rs[0].Normal)
	assert.False(t, rs[0].Predicted)
}

func TestBuilderCaseVariantsCollapse(t *testing.T) {
	b := morphology.NewBuilder(morphology.BuilderOptions{})
	require.NoError(t, b.AddLemma("Кот", "NOUN,anim,masc,sing,nomn"))
	require.NoError(t, b.AddLemma("кот", "NOUN,anim,masc,sing,nomn"))
	d, err := b.Build()
	require.NoError(t, err)
	assert.Len(t, d.Parse("кот"), 1, "«Кот» and «кот» are the same entry after lower-casing")
}

func TestImportTSVLowercases(t *testing.T) {
	d := importTSVString(t, "Москва\tМосквы\t"+tagGeoxGent+"\n")

	rs := d.Parse("москвы")
	require.Len(t, rs, 1)
	assert.Equal(t, "москвы", rs[0].Word)
	assert.Equal(t, "москва", rs[0].Normal)
	assert.False(t, rs[0].Predicted)
}
