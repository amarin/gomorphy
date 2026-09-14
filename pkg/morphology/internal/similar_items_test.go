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

	items := d.SimilarItems("ежик", pol)
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

	items := d.SimilarItems("ежик", pol)
	require.Len(t, items, 1)
	assert.Equal(t, "ежик", items[0].Key)
}

func TestSimilarItemsNoMatch(t *testing.T) {
	pol := RussianCharPolicy()
	dict, guide := testdawg.Build(map[string]uint32{
		"кот" + string(PayloadSeparator) + b64([]byte{1}): 1,
	})
	d := NewDAWG(dict, guide)

	items := d.SimilarItems("мышь", pol)
	assert.Empty(t, items)
}
