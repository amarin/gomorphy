package trie

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	alphabet2 "github.com/amarin/gomorphy/pkg/alphabet"
)

func TestGraph_Add(t *testing.T) {
	t.Run("alphabet uint8 words uint16", func(t *testing.T) {
		a := alphabet2.New[uint8]()
		graph := NewGraph[uint8, uint16](a)

		t.Run("add word", func(t *testing.T) {
			addIdx, err := graph.Add("dag")
			require.NoError(t, err)
			require.Equal(t, uint16(3), addIdx)
			addIdx, err = graph.Add("daG")
			require.NoError(t, err)
			require.Equal(t, uint16(4), addIdx)
		})
		t.Run("get existed word", func(t *testing.T) {
			getIdx, err := graph.Get("dag")
			require.NoError(t, err)
			require.Equal(t, uint16(3), getIdx)
			getIdx, err = graph.Get("daG")
			require.NoError(t, err)
			require.Equal(t, uint16(4), getIdx)
		})
		t.Run("error not existed char in alphabet", func(t *testing.T) {
			_, err := graph.Get("dat")
			require.ErrorIs(t, err, ErrNotFound)
		})
		t.Run("error not existed word", func(t *testing.T) {
			a.MustAdd('t')
			_, err := graph.Get("dat")
			require.ErrorIs(t, err, ErrNotFound)
		})
		t.Run("alphabet overflow", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockedAlphabet := NewMockalphabetInterface[uint8](ctrl)
			localGraph := NewGraph[uint8, uint16](mockedAlphabet)

			mockedAlphabet.EXPECT().GetOrCreate('d').Return(uint8(0), nil)
			mockedAlphabet.EXPECT().GetOrCreate('a').Return(uint8(1), nil)
			mockedAlphabet.EXPECT().GetOrCreate('g').Return(uint8(0), alphabet2.ErrOverflow)

			_, err := localGraph.Add("dag")
			require.Error(t, err)
			require.Error(t, alphabet2.ErrOverflow)
			require.Error(t, ErrAddWord)
		})
	})
	t.Run("alphabet uint8 words uint32", func(t *testing.T) {
		a := alphabet2.New[uint8]()
		graph := NewGraph[uint8, uint32](a)

		t.Run("add word", func(t *testing.T) {
			addIdx, err := graph.Add("dag")
			require.NoError(t, err)
			require.Equal(t, uint32(3), addIdx)
			addIdx, err = graph.Add("daG")
			require.NoError(t, err)
			require.Equal(t, uint32(4), addIdx)
		})
		t.Run("get existed word", func(t *testing.T) {
			getIdx, err := graph.Get("dag")
			require.NoError(t, err)
			require.Equal(t, uint32(3), getIdx)
			getIdx, err = graph.Get("daG")
			require.NoError(t, err)
			require.Equal(t, uint32(4), getIdx)
		})
		t.Run("error not existed char in alphabet", func(t *testing.T) {
			_, err := graph.Get("dat")
			require.ErrorIs(t, err, ErrNotFound)
		})
		t.Run("error not existed word", func(t *testing.T) {
			a.MustAdd('t')
			_, err := graph.Get("dat")
			require.ErrorIs(t, err, ErrNotFound)
		})
		t.Run("alphabet overflow", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockedAlphabet := NewMockalphabetInterface[uint8](ctrl)
			localGraph := NewGraph[uint8, uint32](mockedAlphabet)

			mockedAlphabet.EXPECT().GetOrCreate('d').Return(uint8(0), nil)
			mockedAlphabet.EXPECT().GetOrCreate('a').Return(uint8(1), nil)
			mockedAlphabet.EXPECT().GetOrCreate('g').Return(uint8(0), alphabet2.ErrOverflow)

			_, err := localGraph.Add("dag")
			require.Error(t, err)
			require.Error(t, alphabet2.ErrOverflow)
			require.Error(t, ErrAddWord)
		})
	})
	t.Run("alphabet uint16 words uint16", func(t *testing.T) {
		a := alphabet2.New[uint16]()
		graph := NewGraph[uint16, uint16](a)

		t.Run("add word", func(t *testing.T) {
			addIdx, err := graph.Add("dag")
			require.NoError(t, err)
			require.Equal(t, uint16(3), addIdx)
			addIdx, err = graph.Add("daG")
			require.NoError(t, err)
			require.Equal(t, uint16(4), addIdx)
		})
		t.Run("get existed word", func(t *testing.T) {
			getIdx, err := graph.Get("dag")
			require.NoError(t, err)
			require.Equal(t, uint16(3), getIdx)
			getIdx, err = graph.Get("daG")
			require.NoError(t, err)
			require.Equal(t, uint16(4), getIdx)
		})
		t.Run("error not existed char in alphabet", func(t *testing.T) {
			_, err := graph.Get("dat")
			require.ErrorIs(t, err, ErrNotFound)
		})
		t.Run("error not existed word", func(t *testing.T) {
			a.MustAdd('t')
			_, err := graph.Get("dat")
			require.ErrorIs(t, err, ErrNotFound)
		})
		t.Run("alphabet overflow", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockedAlphabet := NewMockalphabetInterface[uint8](ctrl)
			localGraph := NewGraph[uint8, uint16](mockedAlphabet)

			mockedAlphabet.EXPECT().GetOrCreate('d').Return(uint8(0), nil)
			mockedAlphabet.EXPECT().GetOrCreate('a').Return(uint8(1), nil)
			mockedAlphabet.EXPECT().GetOrCreate('g').Return(uint8(0), alphabet2.ErrOverflow)

			_, err := localGraph.Add("dag")
			require.Error(t, err)
			require.Error(t, alphabet2.ErrOverflow)
			require.Error(t, ErrAddWord)
		})
	})

	t.Run("alphabet uint16 words uint32", func(t *testing.T) {
		a := alphabet2.New[uint16]()
		graph := NewGraph[uint16, uint32](a)

		t.Run("add word", func(t *testing.T) {
			addIdx, err := graph.Add("dag")
			require.NoError(t, err)
			require.Equal(t, uint32(3), addIdx)
			addIdx, err = graph.Add("daG")
			require.NoError(t, err)
			require.Equal(t, uint32(4), addIdx)
		})
		t.Run("get existed word", func(t *testing.T) {
			getIdx, err := graph.Get("dag")
			require.NoError(t, err)
			require.Equal(t, uint32(3), getIdx)
			getIdx, err = graph.Get("daG")
			require.NoError(t, err)
			require.Equal(t, uint32(4), getIdx)
		})
		t.Run("error not existed char in alphabet", func(t *testing.T) {
			_, err := graph.Get("dat")
			require.ErrorIs(t, err, ErrNotFound)
		})
		t.Run("error not existed word", func(t *testing.T) {
			a.MustAdd('t')
			_, err := graph.Get("dat")
			require.ErrorIs(t, err, ErrNotFound)
		})
		t.Run("alphabet overflow", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockedAlphabet := NewMockalphabetInterface[uint8](ctrl)
			localGraph := NewGraph[uint8, uint32](mockedAlphabet)

			mockedAlphabet.EXPECT().GetOrCreate('d').Return(uint8(0), nil)
			mockedAlphabet.EXPECT().GetOrCreate('a').Return(uint8(1), nil)
			mockedAlphabet.EXPECT().GetOrCreate('g').Return(uint8(0), alphabet2.ErrOverflow)

			_, err := localGraph.Add("dag")
			require.Error(t, err)
			require.Error(t, alphabet2.ErrOverflow)
			require.Error(t, ErrAddWord)
		})
	})

}
