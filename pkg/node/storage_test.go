package node

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNode_Read_Write(t *testing.T) {
	t.Run("a8w8", func(t *testing.T) {
		nodeToWrite := make(Node[uint8, uint8])
		nodeToRead := New[uint8, uint8]()

		nodeToWrite.AddNext(1, 1)
		nodeToWrite.AddNext(2, 2)

		buf := new(bytes.Buffer)
		bytesWritten, err := nodeToWrite.WriteTo(buf)
		require.NoError(t, err)
		require.EqualValues(t, 5, bytesWritten)

		bytesRead, err := nodeToRead.ReadFrom(buf)
		require.NoError(t, err)
		require.Equal(t, bytesWritten, bytesRead)
		require.Equal(t, nodeToWrite, nodeToRead)

		require.True(t, nodeToRead.HasNext(1))
		require.True(t, nodeToRead.HasNext(2))
	})

	t.Run("a16w8", func(t *testing.T) {
		nodeToWrite := make(Node[uint16, uint8])
		nodeToRead := New[uint16, uint8]()

		nodeToWrite.AddNext(1, 1)
		nodeToWrite.AddNext(2, 2)

		buf := new(bytes.Buffer)
		bytesWritten, err := nodeToWrite.WriteTo(buf)
		require.NoError(t, err)
		require.EqualValues(t, 8, bytesWritten)

		bytesRead, err := nodeToRead.ReadFrom(buf)
		require.NoError(t, err)
		require.Equal(t, bytesWritten, bytesRead)
		require.Equal(t, nodeToWrite, nodeToRead)

		require.True(t, nodeToRead.HasNext(1))
		require.True(t, nodeToRead.HasNext(2))
	})

	t.Run("a8w16", func(t *testing.T) {
		nodeToWrite := make(Node[uint8, uint16])
		nodeToRead := New[uint8, uint16]()

		nodeToWrite.AddNext(1, 1)
		nodeToWrite.AddNext(2, 2)

		buf := new(bytes.Buffer)
		bytesWritten, err := nodeToWrite.WriteTo(buf)
		require.NoError(t, err)
		require.EqualValues(t, 7, bytesWritten)

		bytesRead, err := nodeToRead.ReadFrom(buf)
		require.NoError(t, err)
		require.Equal(t, bytesWritten, bytesRead)
		require.Equal(t, nodeToWrite, nodeToRead)

		require.True(t, nodeToRead.HasNext(1))
		require.True(t, nodeToRead.HasNext(2))
	})

	t.Run("a16w16", func(t *testing.T) {
		nodeToWrite := make(Node[uint16, uint16])
		nodeToRead := New[uint16, uint16]()

		nodeToWrite.AddNext(1, 1)
		nodeToWrite.AddNext(2, 2)

		buf := new(bytes.Buffer)
		bytesWritten, err := nodeToWrite.WriteTo(buf)
		require.NoError(t, err)
		require.EqualValues(t, 10, bytesWritten)

		bytesRead, err := nodeToRead.ReadFrom(buf)
		require.NoError(t, err)
		require.Equal(t, bytesWritten, bytesRead)
		require.Equal(t, nodeToWrite, nodeToRead)

		require.True(t, nodeToRead.HasNext(1))
		require.True(t, nodeToRead.HasNext(2))
	})
	t.Run("a8w32", func(t *testing.T) {
		nodeToWrite := make(Node[uint8, uint32])
		nodeToRead := New[uint8, uint32]()

		nodeToWrite.AddNext(1, 1)
		nodeToWrite.AddNext(2, 2)

		buf := new(bytes.Buffer)
		bytesWritten, err := nodeToWrite.WriteTo(buf)
		require.NoError(t, err)
		require.EqualValues(t, 11, bytesWritten)

		bytesRead, err := nodeToRead.ReadFrom(buf)
		require.NoError(t, err)
		require.Equal(t, bytesWritten, bytesRead)
		require.Equal(t, nodeToWrite, nodeToRead)

		require.True(t, nodeToRead.HasNext(1))
		require.True(t, nodeToRead.HasNext(2))
	})
	t.Run("a16w32", func(t *testing.T) {
		nodeToWrite := make(Node[uint16, uint32])
		nodeToRead := New[uint16, uint32]()

		nodeToWrite.AddNext(1, 1)
		nodeToWrite.AddNext(2, 2)

		buf := new(bytes.Buffer)
		bytesWritten, err := nodeToWrite.WriteTo(buf)
		require.NoError(t, err)
		require.EqualValues(t, 14, bytesWritten)

		bytesRead, err := nodeToRead.ReadFrom(buf)
		require.NoError(t, err)
		require.Equal(t, bytesWritten, bytesRead)
		require.Equal(t, nodeToWrite, nodeToRead)

		require.True(t, nodeToRead.HasNext(1))
		require.True(t, nodeToRead.HasNext(2))
	})
}
