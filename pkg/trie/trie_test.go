package trie

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGraph_Add(t *testing.T) {
	t.Run("8w16", func(t *testing.T) {
		graph := New[uint8, uint16]()

		t.Run("add word", func(t *testing.T) {
			addIdx, err := graph.Add([]byte("dag"))
			require.NoError(t, err)
			require.Equal(t, uint16(3), addIdx)
			addIdx, err = graph.Add([]byte("daG"))
			require.NoError(t, err)
			require.Equal(t, uint16(4), addIdx)
		})
		t.Run("get existed", func(t *testing.T) {
			getIdx, err := graph.Get([]byte("dag"))
			require.NoError(t, err)
			require.Equal(t, uint16(3), getIdx)
			getIdx, err = graph.Get([]byte("daG"))
			require.NoError(t, err)
			require.Equal(t, uint16(4), getIdx)
		})
		t.Run("error not existed char in alphabet", func(t *testing.T) {
			_, err := graph.Get([]byte("dat"))
			require.ErrorIs(t, err, ErrNotFound)
		})
	})
	t.Run("a8w32", func(t *testing.T) {
		graph := New[uint8, uint32]()

		t.Run("add word", func(t *testing.T) {
			addIdx, err := graph.Add([]byte("dag"))
			require.NoError(t, err)
			require.Equal(t, uint32(3), addIdx)
			addIdx, err = graph.Add([]byte("daG"))
			require.NoError(t, err)
			require.Equal(t, uint32(4), addIdx)
		})
		t.Run("get existed", func(t *testing.T) {
			getIdx, err := graph.Get([]byte("dag"))
			require.NoError(t, err)
			require.Equal(t, uint32(3), getIdx)
			getIdx, err = graph.Get([]byte("daG"))
			require.NoError(t, err)
			require.Equal(t, uint32(4), getIdx)
		})
		t.Run("error not existed", func(t *testing.T) {
			_, err := graph.Get([]byte("dat"))
			require.ErrorIs(t, err, ErrNotFound)
		})
	})
	t.Run("alphabet uint16 words uint16", func(t *testing.T) {
		graph := New[uint16, uint16]()

		t.Run("add word", func(t *testing.T) {
			addIdx, err := graph.Add([]uint16{1, 2, 3})
			require.NoError(t, err)
			require.Equal(t, uint16(3), addIdx)
			addIdx, err = graph.Add([]uint16{1, 2, 4})
			require.NoError(t, err)
			require.Equal(t, uint16(4), addIdx)
		})
		t.Run("get existed", func(t *testing.T) {
			getIdx, err := graph.Get([]uint16{1, 2, 3})
			require.NoError(t, err)
			require.Equal(t, uint16(3), getIdx)
			getIdx, err = graph.Get([]uint16{1, 2, 4})
			require.NoError(t, err)
			require.Equal(t, uint16(4), getIdx)
		})
		t.Run("error not existed", func(t *testing.T) {
			_, err := graph.Get([]uint16{1, 2, 5})
			require.ErrorIs(t, err, ErrNotFound)
		})
	})

	t.Run("a16w32", func(t *testing.T) {
		graph := New[uint16, uint32]()

		t.Run("add word", func(t *testing.T) {
			addIdx, err := graph.Add([]uint16{1, 2, 3})
			require.NoError(t, err)
			require.Equal(t, uint32(3), addIdx)
			addIdx, err = graph.Add([]uint16{1, 2, 4})
			require.NoError(t, err)
			require.Equal(t, uint32(4), addIdx)
		})
		t.Run("get existed word", func(t *testing.T) {
			getIdx, err := graph.Get([]uint16{1, 2, 3})
			require.NoError(t, err)
			require.Equal(t, uint32(3), getIdx)
			getIdx, err = graph.Get([]uint16{1, 2, 4})
			require.NoError(t, err)
			require.Equal(t, uint32(4), getIdx)
		})
		t.Run("error not existed", func(t *testing.T) {
			_, err := graph.Get([]uint16{1, 2, 5})
			require.ErrorIs(t, err, ErrNotFound)
		})
	})

}
