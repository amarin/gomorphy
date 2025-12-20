package node

import (
	"bytes"
	"fmt"
	"math"
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
	t.Run("a8w8", func(t *testing.T) {
		nodeToWrite := New[uint8, uint8]()

		nodeToWrite.SetNext(1, 1)
		nodeToWrite.SetNext(2, 2)

		buf := new(bytes.Buffer)
		bytesWritten, err := nodeToWrite.WriteTo(buf)
		require.NoError(t, err)
		require.EqualValues(t, 6, bytesWritten)

		hexValue, err := nodeToWrite.Hex()
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%X", buf.Bytes()), hexValue)
		require.Equal(t, "000201010202", hexValue)
	})

	t.Run("a16w8", func(t *testing.T) {
		nodeToWrite := New[uint16, uint8]()

		nodeToWrite.SetNext(1, 1)
		nodeToWrite.SetNext(2, 2)

		buf := new(bytes.Buffer)
		bytesWritten, err := nodeToWrite.WriteTo(buf)
		require.NoError(t, err)
		require.EqualValues(t, 9, bytesWritten)

		hexValue, err := nodeToWrite.Hex()
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%X", buf.Bytes()), hexValue)
		require.Equal(t, "000002000101000202", hexValue)
	})

	t.Run("a8w16", func(t *testing.T) {
		nodeToWrite := New[uint8, uint16]()
		nodeToWrite.SetParent(13)

		nodeToWrite.SetNext(1, 1)
		nodeToWrite.SetNext(2, 2)

		buf := new(bytes.Buffer)
		bytesWritten, err := nodeToWrite.WriteTo(buf)
		require.NoError(t, err)
		require.EqualValues(t, 9, bytesWritten)

		hexValue, err := nodeToWrite.Hex()
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%X", buf.Bytes()), hexValue)
		require.Equal(t, "000D02010001020002", hexValue)
	})

	t.Run("a16w16", func(t *testing.T) {
		nodeToWrite := New[uint16, uint16]()
		nodeToWrite.SetParent(255)

		nodeToWrite.SetNext(1, 1)
		nodeToWrite.SetNext(2, 2)

		buf := new(bytes.Buffer)
		bytesWritten, err := nodeToWrite.WriteTo(buf)
		require.NoError(t, err)
		require.EqualValues(t, 12, bytesWritten)

		hexValue, err := nodeToWrite.Hex()
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%X", buf.Bytes()), hexValue)
		require.Equal(t, "00FF00020001000100020002", hexValue)
	})
	t.Run("a8w32", func(t *testing.T) {
		nodeToWrite := New[uint8, uint32]()
		nodeToWrite.SetParent(0xABCDEF01)

		nodeToWrite.SetNext(1, 1)
		nodeToWrite.SetNext(2, 2)

		buf := new(bytes.Buffer)
		bytesWritten, err := nodeToWrite.WriteTo(buf)
		require.NoError(t, err)
		require.EqualValues(t, 15, bytesWritten)

		hexValue, err := nodeToWrite.Hex()
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%X", buf.Bytes()), hexValue)
		require.Equal(t, "ABCDEF010201000000010200000002", hexValue)
	})
	t.Run("a16w32", func(t *testing.T) {
		nodeToWrite := New[uint16, uint32]()
		nodeToWrite.SetParent(0xA0A0A0A0)

		nodeToWrite.SetNext(1, 1)
		nodeToWrite.SetNext(2, 2)

		buf := new(bytes.Buffer)
		bytesWritten, err := nodeToWrite.WriteTo(buf)
		require.NoError(t, err)
		require.EqualValues(t, 18, bytesWritten)

		hexValue, err := nodeToWrite.Hex()
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%X", buf.Bytes()), hexValue)
		require.Equal(t, "A0A0A0A00002000100000001000200000002", hexValue)
	})
}

func TestNode_HasNext(t *testing.T) {
	t.Run("a8w8", func(t *testing.T) {
		nodeToRead := New[uint8, uint8]()

		nodeToRead.SetNext(1, 1)
		require.True(t, nodeToRead.HasNext(1))
		require.False(t, nodeToRead.HasNext(2))
		nodeToRead.SetNext(2, 2)
		require.True(t, nodeToRead.HasNext(2))
	})

	t.Run("a16w8", func(t *testing.T) {
		nodeToRead := New[uint16, uint8]()

		nodeToRead.SetNext(1, 1)
		require.True(t, nodeToRead.HasNext(1))
		require.False(t, nodeToRead.HasNext(2))
		nodeToRead.SetNext(2, 2)
		require.True(t, nodeToRead.HasNext(2))
	})

	t.Run("a8w16", func(t *testing.T) {
		nodeToRead := New[uint8, uint16]()

		nodeToRead.SetNext(1, 1)
		require.True(t, nodeToRead.HasNext(1))
		require.False(t, nodeToRead.HasNext(2))
		nodeToRead.SetNext(2, 2)
		require.True(t, nodeToRead.HasNext(2))
	})

	t.Run("a16w16", func(t *testing.T) {
		nodeToRead := New[uint16, uint16]()

		nodeToRead.SetNext(1, 1)
		require.True(t, nodeToRead.HasNext(1))
		require.False(t, nodeToRead.HasNext(2))
		nodeToRead.SetNext(2, 2)
		require.True(t, nodeToRead.HasNext(2))
	})
	t.Run("a8w32", func(t *testing.T) {
		nodeToRead := New[uint8, uint32]()

		nodeToRead.SetNext(1, 1)
		require.True(t, nodeToRead.HasNext(1))
		require.False(t, nodeToRead.HasNext(2))
		nodeToRead.SetNext(2, 2)
		require.True(t, nodeToRead.HasNext(2))
	})
	t.Run("a16w32", func(t *testing.T) {
		nodeToRead := New[uint16, uint32]()

		nodeToRead.SetNext(1, 1)
		require.True(t, nodeToRead.HasNext(1))
		require.False(t, nodeToRead.HasNext(2))
		nodeToRead.SetNext(2, 2)
		require.True(t, nodeToRead.HasNext(2))
	})
}

func TestNode_NextIdx(t *testing.T) {
	t.Run("a8w8", func(t *testing.T) {
		c, w := uint8(math.MaxUint8), uint8(math.MaxUint8)
		nodeToRead := New[uint8, uint8]()
		next, found := nodeToRead.GetNext(c)

		require.False(t, found)
		require.EqualValues(t, 0, next)

		nodeToRead.SetNext(c, w)
		next, found = nodeToRead.GetNext(c)
		require.True(t, found)
		require.EqualValues(t, w, next)
	})

	t.Run("a16w8", func(t *testing.T) {
		c, w := uint16(math.MaxUint16), uint8(math.MaxUint8)
		nodeToRead := New[uint16, uint8]()
		next, found := nodeToRead.GetNext(c)

		require.False(t, found)
		require.EqualValues(t, 0, next)

		nodeToRead.SetNext(c, w)
		next, found = nodeToRead.GetNext(c)
		require.True(t, found)
		require.EqualValues(t, w, next)
	})

	t.Run("a8w16", func(t *testing.T) {
		c, w := uint8(math.MaxUint8), uint16(math.MaxUint16)
		nodeToRead := New[uint8, uint16]()
		next, found := nodeToRead.GetNext(c)

		require.False(t, found)
		require.EqualValues(t, 0, next)

		nodeToRead.SetNext(c, w)
		next, found = nodeToRead.GetNext(c)
		require.True(t, found)
		require.EqualValues(t, w, next)
	})

	t.Run("a16w16", func(t *testing.T) {
		c, w := uint16(math.MaxUint16), uint16(math.MaxUint16)
		nodeToRead := New[uint16, uint16]()
		next, found := nodeToRead.GetNext(c)

		require.False(t, found)
		require.EqualValues(t, 0, next)

		nodeToRead.SetNext(c, w)
		next, found = nodeToRead.GetNext(c)
		require.True(t, found)
		require.EqualValues(t, w, next)
	})
	t.Run("a8w32", func(t *testing.T) {
		c, w := uint8(math.MaxUint8), uint32(math.MaxUint32)
		nodeToRead := New[uint8, uint32]()
		next, found := nodeToRead.GetNext(c)

		require.False(t, found)
		require.EqualValues(t, 0, next)

		nodeToRead.SetNext(c, w)
		next, found = nodeToRead.GetNext(c)
		require.True(t, found)
		require.EqualValues(t, w, next)
	})
	t.Run("a16w32", func(t *testing.T) {
		c, w := uint16(math.MaxUint16), uint32(math.MaxUint32)
		nodeToRead := New[uint16, uint32]()
		next, found := nodeToRead.GetNext(c)

		require.False(t, found)
		require.EqualValues(t, 0, next)

		nodeToRead.SetNext(c, w)
		next, found = nodeToRead.GetNext(c)
		require.True(t, found)
		require.EqualValues(t, w, next)
	})
}
