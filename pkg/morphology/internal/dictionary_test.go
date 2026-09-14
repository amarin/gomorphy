package internal

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDictionaryConstruction(t *testing.T) {
	ts := NewTagSet("opencorpora")
	ts.Add("NOUN")
	ts.Add("anim")

	paradigm := NewParadigm([]uint16{5, 6}, []uint16{0, 1}, []uint16{0, 0})
	words, _ := testdawg.Build(map[string]uint32{"кот": 0})

	dict := NewDictionary(
		"ru",
		ts,
		[]string{"кот", "коты"},
		[]string{"а"},
		[]Paradigm{paradigm},
		NewDAWG(words, nil),
		RussianCharPolicy(),
	)

	require.NotNil(t, dict)
	assert.Equal(t, "ru", dict.Language)
	assert.Same(t, ts, dict.TagSet)
	assert.Equal(t, []string{"кот", "коты"}, dict.Suffixes)
	assert.Equal(t, []string{"а"}, dict.Prefixes)
	require.Len(t, dict.Paradigms, 1)
	assert.Equal(t, uint16(5), dict.Paradigms[0].Suffix(0))
	assert.NotNil(t, dict.Words)
	assert.NotNil(t, dict.CharPolicy)
	assert.Empty(t, dict.Prediction)
}

func TestDictionaryEmptyComponents(t *testing.T) {
	dict := NewDictionary("en", nil, nil, nil, nil, nil, nil)
	require.NotNil(t, dict)
	assert.Empty(t, dict.Suffixes)
	assert.Len(t, dict.Paradigms, 0)
}
