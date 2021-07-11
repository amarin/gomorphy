package dag

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// ErrReader indicates reader related errors.
var ErrReader = errors.New("read")

// IndexReaderImpl implements IndexReader.
type IndexReaderImpl struct {
	nodeReader NodeReader
}

// DefaultIndexReader provides default index reader implementation.
func DefaultIndexReader() IndexReader {
	return new(IndexReaderImpl)
}

// SetNodeReader sets new node reader.
func (reader IndexReaderImpl) SetNodeReader(nodeReader NodeReader) {
	reader.nodeReader = nodeReader
}

// writeNode writes node itself. As index writes nodes ordered no node index required but parent should written.
// Child nodes will reference to its parent by its index.
func (reader IndexReaderImpl) readNodeToIndex(index Index, r io.Reader) (n int, node Node, err error) {
	var (
		nodeRune  rune
		parentIdx uint32
		parent    Node
		nodeIdx   uint32
	)

	n = 0

	if err = binary.Read(r, binary.LittleEndian, nodeIdx); err != nil {
		return n, nil, fmt.Errorf("%w: %v", ErrReader, err)
	}

	n += 4 // taken node ID 4 bytes

	if err = binary.Read(r, binary.LittleEndian, nodeRune); err != nil {
		return n, nil, fmt.Errorf("%w: %v", ErrReader, err)
	}

	n += 4 // taken rune 4 bytes

	if err = binary.Read(r, binary.LittleEndian, parentIdx); err != nil {
		return n, nil, fmt.Errorf("%w: %v", ErrReader, err)
	}

	n += 4 // taken uint32 4 bytes

	if parentIdx != topNodesParent {
		if parent, err = index.Get(parentIdx); err != nil {
			return n, nil, fmt.Errorf("%w: parent %v of %v not loaded yet", ErrReader, parentIdx, nodeIdx)
		}
	}

	if node, err = index.BuildNode(parent, nodeRune, nil); err != nil {
		return n, nil, fmt.Errorf("%w: build %v: %v", ErrReader, nodeIdx, err)
	}

	node.SetID(nodeIdx)

	if err = index.Set(node); err != nil {
		return n, node, fmt.Errorf("%w: set: %v", ErrReader, err)
	}

	return n, node, nil
}

// writeNode writes node data. Takes node idx in parent index and node itself.
// No node relations required to be written in node data writer.
func (reader IndexReaderImpl) readNodeData(node Node, nodeReader io.Reader) (n int, err error) {
	if n, err = reader.nodeReader.Read(node, nodeReader); err != nil {
		return n, fmt.Errorf("%w: `%v`: %v", ErrWriter, node.Rune(), err)
	}

	return n, nil
}

// writeNode writes node relation into index writer as well as writes node data using node writer.
// Takes index writer, node index, node parent index and node itself.
// Returns written bytes count or error if happened.
func (reader IndexReaderImpl) readNode(index Index, indexReader, nodeReader io.Reader) (n int, node Node, err error) {
	var (
		takenBytes  int
		currentNode Node
	)

	if takenBytes, currentNode, err = reader.readNodeToIndex(index, indexReader); err != nil {
		return 0, nil, err
	}

	n += takenBytes

	if takenBytes, err = reader.readNodeData(currentNode, nodeReader); err != nil {
		return n, currentNode, err
	}

	n += takenBytes

	return n, currentNode, nil
}

// Read reads index data from specified reader.
func (reader IndexReaderImpl) Read(index Index, indexReader, nodeReader io.Reader) (n int, err error) {
	var (
		bytes   int
		nodeIdx uint32
	)

	if reader.nodeReader == nil {
		return n, fmt.Errorf("%w: no nodes writer", ErrWriter)
	}

	for {
		bytes, _, err = reader.readNode(index, indexReader, nodeReader)

		switch {
		case err == nil:
			nodeIdx++

			n += bytes

			continue
		case errors.Is(err, io.EOF):
			return n, nil
		default:
			return n, err
		}
	}
}
