package morphology_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tagFemnNomn = "NOUN,inan,femn,sing,nomn"

func yolkaDict(t *testing.T, opts morphology.BuilderOptions) *morphology.Dictionary {
	t.Helper()
	b := morphology.NewBuilder(opts)
	require.NoError(t, b.AddLemma("ёлка", tagFemnNomn))
	d, err := b.Build()
	require.NoError(t, err)
	return d
}

func TestBuilderDefaultCharPolicyIsRussian(t *testing.T) {
	d := yolkaDict(t, morphology.BuilderOptions{})

	assert.True(t, d.IsKnown("елка"))
	rs := d.Parse("елка")
	require.Len(t, rs, 1)
	assert.Equal(t, "ёлка", rs[0].Word)
	assert.False(t, rs[0].Predicted)
}

func TestBuilderNoCharPolicy(t *testing.T) {
	d := yolkaDict(t, morphology.BuilderOptions{CharPolicy: morphology.NoCharPolicy()})

	assert.True(t, d.IsKnown("ёлка"))
	assert.False(t, d.IsKnown("елка"))
	for _, r := range d.Parse("елка") {
		assert.True(t, r.Predicted, "only a prediction may answer «елка»: %+v", r)
	}
}

func TestBuilderCharPolicySurvivesSaveOpen(t *testing.T) {
	d := yolkaDict(t, morphology.BuilderOptions{CharPolicy: morphology.NoCharPolicy()})
	path := filepath.Join(t.TempDir(), "nopolicy.dat")
	require.NoError(t, d.SaveTo(path))

	got, err := morphology.Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, got.Close()) }()
	assert.False(t, got.IsKnown("елка"))
	assert.True(t, got.IsKnown("ёлка"))
}

func TestBuilderCustomCharPolicy(t *testing.T) {
	pol := morphology.NewCharPolicy(morphology.Substitution{From: 'и', To: 'і'})
	b := morphology.NewBuilder(morphology.BuilderOptions{CharPolicy: pol})
	require.NoError(t, b.AddLemma("міръ", "NOUN,inan,masc,sing,nomn"))
	d, err := b.Build()
	require.NoError(t, err)

	assert.True(t, d.IsKnown("миръ"))
	assert.False(t, d.IsKnown("елка"))
}

// The default policy is chosen by language: е→ё for "ru" and for an empty
// Language (which means "ru"), no substitutions for any other language.
func TestDefaultCharPolicyByLanguage(t *testing.T) {
	const tsv = "ёлка\tёлка\t" + tagFemnNomn + "\n"
	cases := []struct {
		language string
		yo       bool
	}{{"", true}, {"ru", true}, {"en", false}, {"uk", false}}
	for _, c := range cases {
		t.Run("language="+c.language, func(t *testing.T) {
			b := yolkaDict(t, morphology.BuilderOptions{Language: c.language})
			assert.Equal(t, c.yo, b.IsKnown("елка"), "Builder")
			assert.True(t, b.IsKnown("ёлка"), "Builder")

			d, err := morphology.ImportTSV(strings.NewReader(tsv), morphology.BuilderOptions{Language: c.language})
			require.NoError(t, err)
			assert.Equal(t, c.yo, d.IsKnown("елка"), "ImportTSV")
			assert.True(t, d.IsKnown("ёлка"), "ImportTSV")
		})
	}
}

// An explicit policy wins over the language default.
func TestExplicitCharPolicyOverridesLanguage(t *testing.T) {
	d := yolkaDict(t, morphology.BuilderOptions{Language: "en", CharPolicy: morphology.RussianCharPolicy()})
	assert.True(t, d.IsKnown("елка"))
}

func TestUniMorphOptionsCharPolicyFromOutside(t *testing.T) {
	d, err := morphology.CompileFromUniMorph(strings.NewReader("ёж\tёж\tN;NOM;SG\n"),
		morphology.UniMorphOptions{Language: "ru", CharPolicy: morphology.NoCharPolicy()})
	require.NoError(t, err)
	assert.True(t, d.IsKnown("ёж"))
	assert.False(t, d.IsKnown("еж"))
}

func TestFuzzyAppliesCharPolicy(t *testing.T) {
	ru := yolkaDict(t, morphology.BuilderOptions{})
	got := ru.Fuzzy("елка", 0)
	require.Len(t, got, 1)
	assert.Equal(t, morphology.FuzzyMatch{Word: "ёлка", Distance: 0}, got[0])
	assert.Equal(t, got, ru.FuzzyTop("елка", 1))

	none := yolkaDict(t, morphology.BuilderOptions{CharPolicy: morphology.NoCharPolicy()})
	assert.Empty(t, none.Fuzzy("елка", 0))
	got = none.Fuzzy("елка", 1)
	require.Len(t, got, 1)
	assert.Equal(t, 1, got[0].Distance)
}

func TestCharPolicyConstructors(t *testing.T) {
	to, ok := morphology.RussianCharPolicy().Substitute('е')
	assert.True(t, ok)
	assert.Equal(t, 'ё', to)
	_, ok = morphology.NoCharPolicy().Substitute('е')
	assert.False(t, ok)
	assert.NotNil(t, morphology.NoCharPolicy(), "an explicit empty policy, not nil")
}
