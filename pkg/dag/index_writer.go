package dag

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// ErrWriter indicates writer related errors.
var ErrWriter = errors.New("write")

// IndexWriterImpl implements IndexWriter.
type IndexWriterImpl struct {
	nodeWriter NodeWriter
}

// SetNodeWriter sets new node writer.
func (writer IndexWriterImpl) SetNodeWriter(nodeWriter NodeWriter) {
	writer.nodeWriter = nodeWriter
}

// writeNode writes node itself. As index writes nodes ordered no node index required but parent should written.
// Child nodes will reference to its parent by its index.
func (writer IndexWriterImpl) writeNodeToIndex(w io.Writer, parentIdx uint32, node Node) (n int, err error) {
	n = 0

	if err = binary.Write(w, binary.LittleEndian, node.Rune()); err != nil {
		return n, fmt.Errorf("%w: %v", ErrWriter, err)
	}

	n += 4 // written rune 4 bytes

	if err = binary.Write(w, binary.LittleEndian, parentIdx); err != nil {
		return n, fmt.Errorf("%w: %v", ErrWriter, err)
	}

	n += 4 // written uint32 4 bytes

	return n, nil
}

// writeNode writes node data. Takes node idx in parent index and node itself.
// No node relations required to be written in node data writer.
func (writer IndexWriterImpl) writeNodeData(idx uint32, node Node) (n int, err error) {
	if n, err = writer.nodeWriter.Write(idx, node); err != nil {
		return n, fmt.Errorf("%w: `%v`: %v", ErrWriter, node.Rune(), err)
	}

	return n, nil
}

// writeNode writes node relation into index writer as well as writes node data using node writer.
// Takes index writer, node index, node parent index and node itself.
// Returns written bytes count or error if happened.
func (writer IndexWriterImpl) writeNode(w io.Writer, nodeIdx uint32, parentIdx uint32, node Node) (n int, err error) {
	var written int

	if written, err = writer.writeNodeToIndex(w, parentIdx, node); err != nil {
		return 0, err
	}

	n += written

	if written, err = writer.writeNodeData(nodeIdx, node); err != nil {
		return n, err
	}

	n += written

	return n, nil
}

// recurseWriteNode writes nodes recursive into specified writer.
// Takes writer instance, starting index and node to write itself.
// Returns last used node index after all children, written bytes sum or error if happened.
func (writer IndexWriterImpl) recurseWriteNode(
	w io.Writer, idx uint32, parentIdx uint32, node Node) (nodeIdx uint32, n int, err error) {
	bytes := 0
	n = 0

	if writer.nodeWriter == nil {
		return nodeIdx, 0, fmt.Errorf("%w: no nodes writer", ErrWriter)
	}

	if bytes, err = writer.writeNode(w, idx, parentIdx, node); err != nil {
		return nodeIdx, n, err
	}

	n += bytes

	for childRune, child := range node.Children() {
		nodeIdx++

		if nodeIdx, bytes, err = writer.recurseWriteNode(w, nodeIdx, idx, child); err != nil {
			return nodeIdx, n, fmt.Errorf("%w: `%v`: %v", ErrWriter, childRune, err)
		}

		n += bytes
	}

	return nodeIdx, n, nil
}

// WriteInto writes index data into specified writer.
func (writer IndexWriterImpl) WriteInto(idx Index, targetWriter io.Writer) (n int, err error) {
	var (
		totalBytes int
		bytes      int
	)

	if writer.nodeWriter == nil {
		return 0, fmt.Errorf("%w: no nodes writer", ErrWriter)
	}

	parentNodeIdx := uint32(math.MaxUint32) // indicate root node parent as max uint32 to have 0-based index
	nodeIdx := uint32(0)

	for rootRune, rootNode := range idx.Children() {
		if nodeIdx, bytes, err = writer.recurseWriteNode(targetWriter, nodeIdx, parentNodeIdx, rootNode); err != nil {
			return totalBytes, fmt.Errorf("%w: `%v`: %v", ErrWriter, rootRune, err)
		}

		totalBytes += bytes
	}

	return totalBytes, nil
}
