package node

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func getSizeText(v any) string {
	switch v.(type) {
	case uint8:
		return "uint8"
	case uint16:
		return "uint16"
	case uint32:
		return "uint32"
	default:
		panic(fmt.Sprintf("unexpected type: %T", v))
	}
}

func TestNode_Hex(t *testing.T) {
	type hexProvider interface {
		io.ReaderFrom
		io.WriterTo
		Hex() (string, error)
	}

	for _, tt := range []struct {
		prev    any    // MorphemesMaxCount
		idx     any    // MorphemesMaxCount
		charIdx any    // AlphabetSize
		expect  string // HEX
	}{
		{
			prev:    uint16(1),
			idx:     uint16(2),
			charIdx: uint8(3),
			expect:  "0002000103",
		},
		{
			prev:    uint32(1),
			idx:     uint32(2),
			charIdx: uint8(3),
			expect:  "000000020000000103",
		},
		{
			prev:    uint16(1),
			idx:     uint16(2),
			charIdx: uint16(3),
			expect:  "000200010003",
		},
		{
			prev:    uint32(1),
			idx:     uint32(2),
			charIdx: uint16(3),
			expect:  "00000002000000010003",
		},
		{
			prev:    uint16(1),
			idx:     uint16(2),
			charIdx: uint32(3),
			expect:  "0002000100000003",
		},
		{
			prev:    uint32(1),
			idx:     uint32(2),
			charIdx: uint32(3),
			expect:  "000000020000000100000003",
		},
	} {
		alphabetSize := getSizeText(tt.charIdx)
		morphemesSize := getSizeText(tt.idx)

		t.Run(fmt.Sprintf("алфавит %s морфемы %s", alphabetSize, morphemesSize), func(t *testing.T) {
			var (
				a8   uint8  = 0
				a16  uint16 = 0
				i16  uint16 = 0
				i32  uint32 = 0
				p16  uint16 = 0
				p32  uint32 = 0
				node hexProvider
				read hexProvider
			)
			switch typed := tt.charIdx.(type) {
			case uint8:
				a8 = typed
			case uint16:
				a16 = typed
			}

			switch typed := tt.idx.(type) {
			case uint16:
				i16 = typed
			case uint32:
				i32 = typed
			}

			switch typed := tt.prev.(type) {
			case uint16:
				p16 = typed
			case uint32:
				p32 = typed
			}

			switch {
			case a8 != 0 && i16 != 0:
				node = New(i16, a8, p16)
				read = New(uint16(0), uint8(0), uint16(0))
			case a8 != 0 && i32 != 0:
				node = New(i32, a8, p32)
				read = New(uint32(0), uint8(0), uint32(0))
			case a16 != 0 && i16 != 0:
				node = New(i16, a16, p16)
				read = New(uint16(0), uint16(0), uint16(0))
			case a16 != 0 && i32 != 0:
				node = New(i32, a16, p32)
				read = New(uint32(0), uint16(0), uint32(0))
			default:
				require.Fail(t, "неожиданное сочетание типов")
			}
			t.Run("Hex", func(t *testing.T) {
				hex, err := node.Hex()
				require.NoError(t, err)
				require.Equal(t, tt.expect, hex)
			})
			t.Run("WriteTo-ReadFrom", func(t *testing.T) {
				buf := new(bytes.Buffer)
				writeCount, err := node.WriteTo(buf)
				require.NoError(t, err)
				readCount, err := read.ReadFrom(buf)
				require.NoError(t, err)
				require.Equal(t, writeCount, readCount)

				readHex, err := read.Hex()
				require.NoError(t, err)
				require.Equal(t, tt.expect, readHex)
			})
		})
	}
}

func TestNode_Next(t *testing.T) {
	t.Run("алфавит uint8 морфемы uint16", func(t *testing.T) {
		node := New(uint16(1), uint8(0), uint16(0))
		next := node.Next(2, 0)
		require.Equal(t, uint8(0), next.charIdx)
		require.Equal(t, uint16(1), next.prev)
		require.Equal(t, uint16(2), next.idx)
	})
	t.Run("алфавит uint8 морфемы uint32", func(t *testing.T) {
		node := New(uint32(1), uint8(0), uint32(0))
		next := node.Next(2, 0)
		require.Equal(t, uint8(0), next.charIdx)
		require.Equal(t, uint32(1), next.prev)
		require.Equal(t, uint32(2), next.idx)
	})
	t.Run("алфавит uint16 морфемы uint16", func(t *testing.T) {
		node := New(uint16(1), uint16(0), uint16(0))
		next := node.Next(2, 0)
		require.Equal(t, uint16(0), next.charIdx)
		require.Equal(t, uint16(1), next.prev)
		require.Equal(t, uint16(2), next.idx)
	})
	t.Run("алфавит uint16 морфемы uint32", func(t *testing.T) {
		node := New(uint32(1), uint16(0), uint32(0))
		next := node.Next(2, 0)
		require.Equal(t, uint16(0), next.charIdx)
		require.Equal(t, uint32(1), next.prev)
		require.Equal(t, uint32(2), next.idx)
	})
}

func TestNode_PrevIdx(t *testing.T) {
	t.Run("алфавит uint8 морфемы uint16", func(t *testing.T) {
		require.Equal(t, uint16(1), New(uint16(2), uint8(0), uint16(1)).PrevIdx())
	})
	t.Run("алфавит uint8 морфемы uint32", func(t *testing.T) {
		require.Equal(t, uint32(1), New(uint32(2), uint8(0), uint32(1)).PrevIdx())
	})
	t.Run("алфавит uint16 морфемы uint16", func(t *testing.T) {
		require.Equal(t, uint16(1), New(uint16(2), uint16(0), uint16(1)).PrevIdx())
	})
	t.Run("алфавит uint16 морфемы uint32", func(t *testing.T) {
		require.Equal(t, uint32(1), New(uint32(2), uint16(0), uint32(1)).PrevIdx())
	})
}

func TestNode_CharIdx(t *testing.T) {
	t.Run("алфавит uint8 морфемы uint16", func(t *testing.T) {
		require.Equal(t, uint8(2), New(uint16(1), uint8(2), uint16(0)).CharIdx())
	})
	t.Run("алфавит uint8 морфемы uint32", func(t *testing.T) {
		require.Equal(t, uint8(2), New(uint32(1), uint8(2), uint32(0)).CharIdx())
	})
	t.Run("алфавит uint16 морфемы uint16", func(t *testing.T) {
		require.Equal(t, uint16(2), New(uint16(1), uint16(2), uint16(0)).CharIdx())
	})
	t.Run("алфавит uint16 морфемы uint32", func(t *testing.T) {
		require.Equal(t, uint16(2), New(uint32(1), uint16(2), uint32(0)).CharIdx())
	})
}

func TestNode_HasNext(t *testing.T) {
	t.Run("алфавит uint8 морфемы uint16", func(t *testing.T) {
		node := New(uint16(1), uint8(0), uint16(0))
		_ = node.Next(2, 1)
		require.True(t, node.HasNext(1))
		require.False(t, node.HasNext(0))
	})
	t.Run("алфавит uint8 морфемы uint32", func(t *testing.T) {
		node := New(uint32(1), uint8(0), uint32(0))
		_ = node.Next(2, 1)
		require.True(t, node.HasNext(1))
		require.False(t, node.HasNext(0))
	})
	t.Run("алфавит uint16 морфемы uint16", func(t *testing.T) {
		node := New(uint16(1), uint16(0), uint16(0))
		_ = node.Next(2, 1)
		require.True(t, node.HasNext(1))
		require.False(t, node.HasNext(0))
	})
	t.Run("алфавит uint16 морфемы uint32", func(t *testing.T) {
		node := New(uint32(1), uint16(0), uint32(0))
		_ = node.Next(2, 1)
		require.True(t, node.HasNext(1))
		require.False(t, node.HasNext(0))
	})
}

func TestNode_NextIdx(t *testing.T) {
	t.Run("алфавит uint8 морфемы uint16", func(t *testing.T) {
		node := New(uint16(1), uint8(0), uint16(0))
		_ = node.Next(2, 1)
		nextIdx, exists := node.NextIdx(1)
		require.True(t, exists)
		require.Equal(t, uint16(2), nextIdx)
		nextIdx, exists = node.NextIdx(2)
		require.False(t, exists)
	})
	t.Run("алфавит uint8 морфемы uint32", func(t *testing.T) {
		node := New(uint32(1), uint8(0), uint32(0))
		_ = node.Next(2, 1)
		nextIdx, exists := node.NextIdx(1)
		require.True(t, exists)
		require.Equal(t, uint32(2), nextIdx)
		nextIdx, exists = node.NextIdx(2)
		require.False(t, exists)
	})
	t.Run("алфавит uint16 морфемы uint16", func(t *testing.T) {
		node := New(uint16(1), uint16(0), uint16(0))
		_ = node.Next(2, 1)
		nextIdx, exists := node.NextIdx(1)
		require.True(t, exists)
		require.Equal(t, uint16(2), nextIdx)
		nextIdx, exists = node.NextIdx(2)
		require.False(t, exists)
	})
	t.Run("алфавит uint16 морфемы uint32", func(t *testing.T) {
		node := New(uint32(1), uint16(0), uint32(0))
		_ = node.Next(2, 1)
		nextIdx, exists := node.NextIdx(1)
		require.True(t, exists)
		require.Equal(t, uint32(2), nextIdx)
		nextIdx, exists = node.NextIdx(2)
		require.False(t, exists)
	})
}
