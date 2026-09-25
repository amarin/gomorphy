package morphology_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// savedSmallDict saves buildSmallDict to a temp file and returns the path
// and the file's bytes.
func savedSmallDict(t *testing.T) (string, []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "small.dat")
	require.NoError(t, buildSmallDict(t).SaveTo(path))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return path, data
}

func TestOpenBytesMatchesOpen(t *testing.T) {
	path, data := savedSmallDict(t)

	fromFile, err := morphology.Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, fromFile.Close()) }()

	fromBytes, err := morphology.OpenBytes(data)
	require.NoError(t, err)

	words := []string{"кот", "кота", "мышь", "мыши", "бота"}
	cmpSnapshots(t, readingsSnapshot(fromFile, words...), readingsSnapshot(fromBytes, words...))
	assert.Equal(t, fromFile.Fuzzy("кот", 1), fromBytes.Fuzzy("кот", 1))
	assert.Equal(t, fromFile.Info(), fromBytes.Info())
	assert.Equal(t, fromFile.TagSetName(), fromBytes.TagSetName())
	assert.NoError(t, fromBytes.Close(), "Close is a no-op for OpenBytes")
}

func TestOpenBytesChecksumMismatch(t *testing.T) {
	_, data := savedSmallDict(t)
	bad := slices.Clone(data)
	bad[len(bad)-1] ^= 0xFF

	_, err := morphology.OpenBytes(bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestOpenBytesRejectsGarbage(t *testing.T) {
	_, err := morphology.OpenBytes(nil)
	require.Error(t, err)
	_, err = morphology.OpenBytes([]byte("not a dictionary at all"))
	require.Error(t, err)
}

// //go:embed data has no alignment guarantee. A buffer shifted by one byte
// makes every section misaligned, so ParseDAWG takes its copying path.
func TestOpenBytesMisaligned(t *testing.T) {
	_, data := savedSmallDict(t)
	buf := make([]byte, len(data)+1)
	copy(buf[1:], data)

	d, err := morphology.OpenBytes(buf[1:])
	require.NoError(t, err)
	rs := d.Parse("кота")
	require.Len(t, rs, 1)
	assert.Equal(t, "кот", rs[0].Normal)
	assert.False(t, rs[0].Predicted)
}
