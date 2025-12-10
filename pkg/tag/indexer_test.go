package tag

import (
	"bytes"
	"testing"

	"github.com/amarin/logging"
	"github.com/stretchr/testify/require"
)

func TestNewIndexer(t *testing.T) {
	require.NoError(t,
		logging.Init(
			logging.WithLevel(logging.LevelDebug),
			logging.WithFormat(logging.FormatText),
		))

	i := NewIndexer()
	i.MustAdd(Tag{
		Parent: "PART",
		Name:   "NAME",
	})
	require.Equal(t, 1, i.Len())

	expectBytes := []byte{0x1, 0x1, 0x50, 0x41, 0x52, 0x54, 0x4e, 0x41, 0x4d, 0x45}

	buf := bytes.NewBuffer(nil)
	n, err := i.WriteTo(buf)
	require.NoError(t, err)
	require.Equal(t, int64(10), n)
	require.Equal(t, expectBytes, buf.Bytes())

	anotherIndex := NewIndexer()
	n, err = anotherIndex.ReadFrom(buf)
	require.NoError(t, err)
	require.Equal(t, int64(10), n)
	require.Equal(t, Name("NAME"), anotherIndex.MustGet(0).Name)
	require.Equal(t, Name("PART"), anotherIndex.MustGet(0).Parent)
}
