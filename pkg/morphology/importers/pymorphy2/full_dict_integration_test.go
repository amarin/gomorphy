//go:build integration

package pymorphy2_test

import (
	"os"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/importers/pymorphy2"
	"github.com/amarin/gomorphy/pkg/morphology/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func importFullDict(t *testing.T) *internal.Dictionary {
	t.Helper()
	dir := os.Getenv("GOMORPHY_PYMORPHY2_DIR")
	if dir == "" {
		t.Skip("GOMORPHY_PYMORPHY2_DIR not set")
	}
	d, err := pymorphy2.ImportFromDir(dir)
	require.NoError(t, err)
	require.NotNil(t, d)
	return d
}

func TestFullDictStructure(t *testing.T) {
	d := importFullDict(t)

	require.Len(t, d.Paradigms, 1, "pymorphy2 import is never sharded")
	assert.Greater(t, len(d.Paradigms[0]), 1000, "полный словарь: ~3000 парадигм")
	require.Len(t, d.Suffixes, 1)
	assert.Greater(t, len(d.Suffixes[0]), 1000, "полный словарь: ~5K суффиксов")
	assert.Greater(t, len(d.TagSet.Tags), 500, "полный словарь: ~1K тегов")
	assert.Equal(t, []string{"", "по", "наи"}, d.Prefixes)

	require.Len(t, d.Words, 1)
	require.NotNil(t, d.Words[0])
	require.Len(t, d.Prediction, 3, "prediction-suffixes по числу префиксов")
	require.NotNil(t, d.Probability)
}

func TestFullDictVseReadings(t *testing.T) {
	d := importFullDict(t)

	items := d.Words[0].SimilarItems("все", d.CharPolicy)
	total := 0
	for _, it := range items {
		total += len(it.Values)
	}
	t.Logf("все: items=%d readings=%d", len(items), total)
	assert.GreaterOrEqual(t, total, 4, "Parse(\"все\") даёт ≥4 разбора")
}
