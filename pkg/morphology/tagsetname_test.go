package morphology_test

import (
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/amarin/gomorphy/pkg/morphology/tagmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDictionary_TagSetName confirms Dictionary.TagSetName returns
// exactly the dictName tagmap.Map expects for each source — closing the
// "TagSet.Name is unreachable from pkg/morphology's public API" gap
// noted in docs/en/implementation/tag-mapping.md.
func TestDictionary_TagSetName(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	pm := buildFixture(t, words, nil, nil)
	assert.Equal(t, "opencorpora-int", pm.TagSetName())

	oc, err := morphology.CompileFromXML(strings.NewReader(exampleDictXML), nil)
	require.NoError(t, err)
	assert.Equal(t, "opencorpora", oc.TagSetName())

	// The names each source actually reports must be ones tagmap.Map
	// itself recognizes, not just plausible-looking strings.
	_, ok := tagmap.Map(pm.TagSetName(), "NOUN,anim,masc,sing,nomn")
	assert.True(t, ok, "tagmap.Map must recognize pymorphy2's reported TagSetName")
	_, ok = tagmap.Map(oc.TagSetName(), "NOUN,anim,masc,sing,nomn")
	assert.True(t, ok, "tagmap.Map must recognize OpenCorpora's reported TagSetName")
}

func TestDictionary_TagSetName_NilSafety(t *testing.T) {
	var d *morphology.Dictionary
	assert.Empty(t, d.TagSetName())
}

// TestMultiDictionary_DictTagSetName confirms per-index TagSet.Name
// reachability through MultiDictionary, mirroring DictInfo's existing
// pattern — the realistic case: Reading.Dict tells you which
// dictionary produced a reading, and DictTagSetName(Reading.Dict) is
// how you'd get the right dictName for tagmap.Map(dictName, reading.Tag).
func TestMultiDictionary_DictTagSetName(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	pm := buildFixture(t, words, nil, nil)
	oc, err := morphology.CompileFromXML(strings.NewReader(exampleDictXML), nil)
	require.NoError(t, err)

	m := morphology.NewMultiDictionary(pm, oc)
	assert.Equal(t, "opencorpora-int", m.DictTagSetName(0))
	assert.Equal(t, "opencorpora", m.DictTagSetName(1))
	assert.Empty(t, m.DictTagSetName(-1))
	assert.Empty(t, m.DictTagSetName(2))
}
