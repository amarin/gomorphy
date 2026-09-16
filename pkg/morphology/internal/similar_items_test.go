package internal

import (
	"encoding/base64"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func b64(p []byte) string {
	return base64.StdEncoding.EncodeToString(p)
}

func TestSimilarItemsFindsEAndYoVariants(t *testing.T) {
	pol := RussianCharPolicy()
	dict, guide := testdawg.Build(map[string]uint32{
		"ежик" + string(PayloadSeparator) + b64([]byte{0, 1, 0, 2}): 1,
		"ёжик" + string(PayloadSeparator) + b64([]byte{3, 4, 5, 6}): 2,
		"кот" + string(PayloadSeparator) + b64([]byte{9, 9}):        3,
	})
	d := NewDAWG(dict, guide)

	items := d.SimilarItems("ежик", pol, nil)
	require.Len(t, items, 2)

	assert.Equal(t, "ежик", items[0].Key)
	require.Len(t, items[0].Values, 1)
	assert.Equal(t, []byte{0, 1, 0, 2}, items[0].Values[0])

	assert.Equal(t, "ёжик", items[1].Key)
	require.Len(t, items[1].Values, 1)
	assert.Equal(t, []byte{3, 4, 5, 6}, items[1].Values[0])
}

func TestSimilarItemsExactOnlyWithoutSubstitutes(t *testing.T) {
	pol := NewCharPolicy()
	dict, guide := testdawg.Build(map[string]uint32{
		"ежик" + string(PayloadSeparator) + b64([]byte{1}): 1,
		"ёжик" + string(PayloadSeparator) + b64([]byte{2}): 2,
	})
	d := NewDAWG(dict, guide)

	items := d.SimilarItems("ежик", pol, nil)
	require.Len(t, items, 1)
	assert.Equal(t, "ежик", items[0].Key)
}

func TestSimilarItemsNoMatch(t *testing.T) {
	pol := RussianCharPolicy()
	dict, guide := testdawg.Build(map[string]uint32{
		"кот" + string(PayloadSeparator) + b64([]byte{1}): 1,
	})
	d := NewDAWG(dict, guide)

	items := d.SimilarItems("мышь", pol, nil)
	assert.Empty(t, items)
}

func TestSimilarItemsWithDenseAlphabet(t *testing.T) {
	corpus := []string{"кот", "кота", "мышь"}
	alphabet, err := NewDenseAlphabet(1, corpus)
	require.NoError(t, err)

	encode := func(s string) string {
		b, err := alphabet.Encode(s)
		require.NoError(t, err)
		return string(b)
	}

	dict, guide := testdawg.Build(map[string]uint32{
		encode("кот") + string(PayloadSeparator) + b64([]byte{1}):  0,
		encode("кота") + string(PayloadSeparator) + b64([]byte{2}): 0,
	})
	d := NewDAWG(dict, guide)

	items := d.SimilarItems("кот", NewCharPolicy(), alphabet)
	require.Len(t, items, 1)
	assert.Equal(t, "кот", items[0].Key)
	require.Len(t, items[0].Values, 1)
	assert.Equal(t, []byte{1}, items[0].Values[0])

	items = d.SimilarItems("кота", NewCharPolicy(), alphabet)
	require.Len(t, items, 1)
	assert.Equal(t, "кота", items[0].Key)

	items = d.SimilarItems("мышь", NewCharPolicy(), alphabet)
	assert.Empty(t, items, "мышь не было закодировано в этот DAWG")
}

func TestSimilarItemsWithDenseAlphabetAndCharPolicy(t *testing.T) {
	corpus := []string{"ежик", "ёжик"}
	alphabet, err := NewDenseAlphabet(1, corpus)
	require.NoError(t, err)

	encode := func(s string) string {
		b, err := alphabet.Encode(s)
		require.NoError(t, err)
		return string(b)
	}

	dict, guide := testdawg.Build(map[string]uint32{
		encode("ёжик") + string(PayloadSeparator) + b64([]byte{9}): 0,
	})
	d := NewDAWG(dict, guide)

	// "ежик" не в DAWG буквально; подмена е→ё в CharPolicy, применённая
	// через alphabet.Encode, должна найти "ёжик".
	items := d.SimilarItems("ежик", RussianCharPolicy(), alphabet)
	require.Len(t, items, 1)
	assert.Equal(t, "ёжик", items[0].Key)
	require.Len(t, items[0].Values, 1)
	assert.Equal(t, []byte{9}, items[0].Values[0])
}

func TestSimilarItemsNilAlphabetUnchanged(t *testing.T) {
	// Same fixture and assertions as TestSimilarItemsFindsEAndYoVariants,
	// but passing alphabet explicitly as nil — guards the "nil means
	// identical to before" contract at the call-site level, not just by
	// inspection of followRuneVia.
	pol := RussianCharPolicy()
	dict, guide := testdawg.Build(map[string]uint32{
		"ежик" + string(PayloadSeparator) + b64([]byte{0, 1, 0, 2}): 1,
		"ёжик" + string(PayloadSeparator) + b64([]byte{3, 4, 5, 6}): 2,
	})
	d := NewDAWG(dict, guide)

	items := d.SimilarItems("ежик", pol, nil)
	require.Len(t, items, 2)
	assert.Equal(t, "ежик", items[0].Key)
	assert.Equal(t, "ёжик", items[1].Key)
}
