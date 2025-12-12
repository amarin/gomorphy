package indexer

import (
	"bytes"
	"testing"

	"github.com/amarin/logging"
	"github.com/stretchr/testify/require"
)

func TestIndexOf_WriteTo(t *testing.T) {
	require.NoError(t,
		logging.Init(
			logging.WithLevel(logging.LevelDebug),
			logging.WithFormat(logging.FormatText),
		))
	idx := New[uint8, fake]("any")
	t.Run("пустой индекс", func(t *testing.T) {
		buf := new(bytes.Buffer)
		n, err := idx.WriteTo(buf)
		require.NoError(t, err)
		require.Equal(t, int64(2), n)
		require.Equal(t, []byte{1, 0}, buf.Bytes())
	})
	t.Run("непустой индекс", func(t *testing.T) {
		require.NotPanics(t, func() {
			idx.MustAdd('a')
			idx.MustAdd('b')
			idx.MustAdd('c')
		})
		buf := new(bytes.Buffer)
		n, err := idx.WriteTo(buf)
		require.NoError(t, err)
		require.Equal(t, int64(14), n)
		require.Equal(t,
			[]byte{0x1, 0x3, 0x0, 0x0, 0x0, 0x61, 0x0, 0x0, 0x0, 0x62, 0x0, 0x0, 0x0, 0x63},
			buf.Bytes(),
		)
	})
}

func TestIndexOf_ReadFrom_uint8(t *testing.T) {
	require.NoError(t,
		logging.Init(
			logging.WithLevel(logging.LevelDebug),
			logging.WithFormat(logging.FormatText),
		))
	idx := New[uint8, fake]("any")
	t.Run("пустой индекс", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte{1, 0})
		n, err := idx.ReadFrom(buf)
		require.NoError(t, err)
		require.Equal(t, int64(2), n)
		require.Equal(t, 0, idx.Len())
	})
	t.Run("непустой индекс", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte{0x1, 0x3, 0x0, 0x0, 0x0, 0x61, 0x0, 0x0, 0x0, 0x62, 0x0, 0x0, 0x0, 0x63})
		n, err := idx.ReadFrom(buf)
		require.NoError(t, err)
		require.Equal(t, int64(14), n)
		require.Equal(t, 3, idx.Len())
		require.Equal(t, fake('a'), idx.MustGet(0))
		require.Equal(t, fake('b'), idx.MustGet(1))
		require.Equal(t, fake('c'), idx.MustGet(2))
	})
}

func TestIndexOf_ReadFrom_uint16(t *testing.T) {
	require.NoError(t,
		logging.Init(
			logging.WithLevel(logging.LevelDebug),
			logging.WithFormat(logging.FormatText),
		))
	idx := New[uint16, fake]("any")
	t.Run("пустой индекс", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte{0x2, 0, 0})
		n, err := idx.ReadFrom(buf)
		require.NoError(t, err)
		require.Equal(t, int64(3), n)
		require.Equal(t, 0, idx.Len())
	})
	t.Run("непустой индекс", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte{0x2, 0x3, 0x0, 0x0, 0x0, 0x61, 0x0, 0x0, 0x0, 0x62, 0x0, 0x0, 0x0, 0x63})
		n, err := idx.ReadFrom(buf)
		require.NoError(t, err)
		require.Equal(t, int64(14), n)
		require.Equal(t, 3, idx.Len())
		require.Equal(t, fake('a'), idx.MustGet(0))
		require.Equal(t, fake('b'), idx.MustGet(1))
		require.Equal(t, fake('c'), idx.MustGet(2))
	})
}
