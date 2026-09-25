package morphology_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentHashShape(t *testing.T) {
	h := buildSmallDict(t).ContentHash()
	assert.Len(t, h, 32)
	assert.Regexp(t, `^[0-9a-f]{32}$`, h)

	var nilDict *morphology.Dictionary
	assert.Equal(t, "", nilDict.ContentHash())
}

func TestContentHashStableAcrossSaveOpen(t *testing.T) {
	d := buildSmallDict(t)
	want := d.ContentHash()

	path := filepath.Join(t.TempDir(), "a.dat")
	require.NoError(t, d.SaveTo(path))

	opened, err := morphology.Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, opened.Close()) }()
	assert.Equal(t, want, opened.ContentHash())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	fromBytes, err := morphology.OpenBytes(data)
	require.NoError(t, err)
	assert.Equal(t, want, fromBytes.ContentHash())
}

// Re-saving writes a new BuiltAt into "info"; ContentHash ignores "info",
// including Source.
func TestContentHashIgnoresInfo(t *testing.T) {
	a := morphology.NewBuilder(morphology.BuilderOptions{Source: "a"})
	b := morphology.NewBuilder(morphology.BuilderOptions{Source: "b"})
	for _, bl := range []*morphology.Builder{a, b} {
		require.NoError(t, bl.AddLemma("кот", "NOUN,anim,masc,sing,nomn"))
		require.NoError(t, bl.AddForm("кота", "кот", "NOUN,anim,masc,sing,gent"))
	}
	da, err := a.Build()
	require.NoError(t, err)
	db, err := b.Build()
	require.NoError(t, err)

	require.NotEqual(t, da.Info().Source, db.Info().Source)
	assert.Equal(t, da.ContentHash(), db.ContentHash())

	dir := t.TempDir()
	p1, p2 := filepath.Join(dir, "1.dat"), filepath.Join(dir, "2.dat")
	require.NoError(t, da.SaveTo(p1))
	require.NoError(t, da.SaveTo(p2))
	o1, err := morphology.Open(p1)
	require.NoError(t, err)
	defer func() { _ = o1.Close() }()
	o2, err := morphology.Open(p2)
	require.NoError(t, err)
	defer func() { _ = o2.Close() }()
	assert.Equal(t, o1.ContentHash(), o2.ContentHash())
}

func TestContentHashDeterministicRebuild(t *testing.T) {
	assert.Equal(t, buildSmallDict(t).ContentHash(), buildSmallDict(t).ContentHash())
}

func TestContentHashChangesWithContent(t *testing.T) {
	base := buildFromTriples(t,
		[3]string{"кот", "кот", mergeTagMascNomn},
		[3]string{"кота", "кот", mergeTagMascGent},
	)
	changed := buildFromTriples(t,
		[3]string{"кот", "кот", mergeTagMascNomn},
		[3]string{"коту", "кот", mergeTagMascGent},
	)
	assert.NotEqual(t, base.ContentHash(), changed.ContentHash())

	policy := morphology.NewBuilder(morphology.BuilderOptions{CharPolicy: morphology.NoCharPolicy()})
	require.NoError(t, policy.AddForm("кот", "кот", mergeTagMascNomn))
	require.NoError(t, policy.AddForm("кота", "кот", mergeTagMascGent))
	withoutPolicy, err := policy.Build()
	require.NoError(t, err)
	assert.NotEqual(t, base.ContentHash(), withoutPolicy.ContentHash(), "meta (CharPolicy) is content")
}

func TestContentHashPyMorphy(t *testing.T) {
	d := parseDict(t)
	assert.Len(t, d.ContentHash(), 32)
	assert.Equal(t, d.ContentHash(), d.ContentHash(), "cached value is stable")
}
