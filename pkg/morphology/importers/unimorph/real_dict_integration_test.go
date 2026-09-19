//go:build integration

// Integration tests here require the real UniMorph rus TSV at
// .data/unimorph/ru/data (see docs/en/todo.md / Makefile's
// test-integration target, and pkg/unimorph.Loader for how to obtain
// one — e.g. `gomorphy download unimorph`). Run explicitly:
//
//	go test -tags=integration ./pkg/morphology/importers/unimorph/... -v
package unimorph_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/importers/unimorph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	realTSVEnvVar      = "GOMORPHY_UNIMORPH_TSV"
	realTSVDefaultPath = ".data/unimorph/ru/data"
)

func realTSVPath(t *testing.T) string {
	t.Helper()

	path := os.Getenv(realTSVEnvVar)
	if path == "" {
		path = findRealTSV()
		if path == "" {
			t.Skipf("real UniMorph rus TSV not found (%s to override)", realTSVEnvVar)
		}
	}
	return path
}

// findRealTSV walks up from the working dir looking for
// .data/unimorph/ru/data, so this test passes both from the package dir
// and from the repo root (same approach as opencorpora's
// real_dict_integration_test.go).
func findRealTSV() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, realTSVDefaultPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func TestImportFromTSV_RealDict(t *testing.T) {
	path := realTSVPath(t)

	d, err := unimorph.CompileFromTSVFile(path, unimorph.Options{Language: "ru"})
	require.NoError(t, err)

	totalParadigms := 0
	for _, shard := range d.Paradigms {
		totalParadigms += len(shard)
	}
	t.Logf("shards=%d paradigms=%d tags=%d", len(d.Words), totalParadigms, len(d.TagSet.Tags))
	assert.Greater(t, totalParadigms, 1000, "real rus: expected on the order of thousands of unique paradigms")
	assert.Equal(t, "unimorph", d.TagSet.Name)
	assert.Greater(t, len(d.TagSet.Tags), 10, "real rus: expected more than a handful of unique bundles")

	// A word known to exist in the real rus data, with a specific, stable
	// reading (verified against the actual downloaded file — see
	// docs/en/implementation/stage-16-import-unimorph.md).
	items := d.Words[0].SimilarItems("кота", d.CharPolicy, d.Alphabet)
	require.NotEmpty(t, items, `"кота" must be found in the real rus dictionary`)
}
