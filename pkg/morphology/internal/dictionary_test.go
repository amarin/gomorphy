package internal

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDictionaryConstruction(t *testing.T) {
	ts := NewTagSet("opencorpora")
	_, err := ts.Add("NOUN")
	require.NoError(t, err)
	_, err = ts.Add("anim")
	require.NoError(t, err)

	paradigm := NewParadigm([]uint16{5, 6}, []uint16{0, 1}, []uint16{0, 0})
	words, _ := testdawg.Build(map[string]uint32{"кот": 0})

	dict := NewDictionary(
		"ru",
		ts,
		[][]string{{"кот", "коты"}},
		[]string{"а"},
		[][]Paradigm{{paradigm}},
		[]*DAWG{NewDAWG(words, nil)},
		RussianCharPolicy(),
	)

	require.NotNil(t, dict)
	assert.Equal(t, "ru", dict.Language)
	assert.Same(t, ts, dict.TagSet)
	require.Len(t, dict.Suffixes, 1)
	assert.Equal(t, []string{"кот", "коты"}, dict.Suffixes[0])
	assert.Equal(t, []string{"а"}, dict.Prefixes)
	require.Len(t, dict.Paradigms, 1)
	require.Len(t, dict.Paradigms[0], 1)
	assert.Equal(t, uint16(5), dict.Paradigms[0][0].Suffix(0))
	require.Len(t, dict.Words, 1)
	assert.NotNil(t, dict.Words[0])
	assert.NotNil(t, dict.CharPolicy)
	assert.Empty(t, dict.Prediction)
}

func TestDictionaryEmptyComponents(t *testing.T) {
	dict := NewDictionary("en", nil, nil, nil, nil, nil, nil)
	require.NotNil(t, dict)
	assert.Empty(t, dict.Suffixes)
	assert.Len(t, dict.Paradigms, 0)
}

func TestNewDictionaryPanicsOnShardCountMismatch(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on mismatched shard slice lengths")
		}
	}()
	NewDictionary("ru", nil, [][]string{{"a"}}, nil, nil, nil, nil)
}
