package morphology

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMergeRealDictionary merges a small overlay into a real .dat
// (GOMORPHY_MERGE_BASE, e.g. .data/pymorphy/pymorphy.dat) and checks that
// every sampled base word parses identically, OOV prediction is
// unchanged, the TagSet name is kept, and the output grows ≤ 2%.
func TestMergeRealDictionary(t *testing.T) {
	path := os.Getenv("GOMORPHY_MERGE_BASE")
	if path == "" {
		t.Skip("GOMORPHY_MERGE_BASE not set")
	}
	base, err := Open(path)
	require.NoError(t, err)
	defer func() { _ = base.Close() }()

	ob := NewBuilder(BuilderOptions{Language: base.Language()})
	require.NoError(t, ob.AddForm("криптобиржа", "криптобиржа", "NOUN,inan,femn sing,nomn"))
	require.NoError(t, ob.AddForm("криптобиржи", "криптобиржа", "NOUN,inan,femn sing,gent"))
	overlay, err := ob.Build()
	require.NoError(t, err)

	start := time.Now()
	merged, err := Merge(base, []*Dictionary{overlay}, MergeAdd)
	require.NoError(t, err)
	t.Logf("merge took %v", time.Since(start))

	assert.Equal(t, base.TagSetName(), merged.TagSetName())

	var sample []string
	i := 0
	base.d.Words[0].Walk(func(key string, _ [][]byte) {
		if i++; i%250 == 0 && len(sample) < 20000 {
			w, err := base.d.Alphabet.Decode([]byte(key))
			require.NoError(t, err)
			sample = append(sample, w)
		}
	})
	sample = append(sample, "бутявкающий", "глокая", "куздра", "штеко", "будланула", "бокра")
	for _, w := range sample {
		require.Equal(t, base.Parse(w), merged.Parse(w), w)
	}
	assert.NotEmpty(t, merged.Parse("криптобиржи"))

	out := filepath.Join(t.TempDir(), "merged.dat")
	require.NoError(t, merged.SaveTo(out))
	bi, err := os.Stat(path)
	require.NoError(t, err)
	oi, err := os.Stat(out)
	require.NoError(t, err)
	t.Logf("size %d → %d (%.2f%%)", bi.Size(), oi.Size(), 100*float64(oi.Size()-bi.Size())/float64(bi.Size()))
	assert.LessOrEqual(t, float64(oi.Size()), float64(bi.Size())*1.02)
}
