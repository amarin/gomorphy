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

// topNodesParent defines parent ID to use with top nodes to allow whole index 0 based.
const topNodesParent = uint32(math.MaxUint32)

// IndexWriterImpl implements IndexWriter.
type IndexWriterImpl struct {
	nodeWriter NodeWriter
}

// DefaultIndexWriter provides default index writer implementation.
func DefaultIndexWriter() IndexWriter {
	return new(IndexWriterImpl)
}

// SetNodeWriter sets new node writer.
func (writer IndexWriterImpl) SetNodeWriter(nodeWriter NodeWriter) {
	writer.nodeWriter = nodeWriter
}

// writeNode writes node itself. As index writes nodes ordered no node index required but parent should written.
// Child nodes will reference to its parent by its index.
func (writer IndexWriterImpl) writeNodeToIndex(w io.Writer, node Node) (n int, err error) {
	n = 0

	if err = binary.Write(w, binary.LittleEndian, node.ID()); err != nil {
		return n, fmt.Errorf("%w: %v", ErrWriter, err)
	}

	n += 4 // written node ID 4 bytes

	if err = binary.Write(w, binary.LittleEndian, node.Rune()); err != nil {
		return n, fmt.Errorf("%w: %v", ErrWriter, err)
	}

	n += 4 // written rune 4 bytes

	switch parent := node.Parent(); parent {
	case nil:
		err = binary.Write(w, binary.LittleEndian, topNodesParent)
	default:
		err = binary.Write(w, binary.LittleEndian, parent.ID())
	}

	if err != nil {
		return n, fmt.Errorf("%w: %v", ErrWriter, err)
	}

	n += 4 // written parent ID 4 bytes

	return n, nil
}

// writeNode writes node data. Takes node idx in parent index and node itself.
// No node relations required to be written in node data writer.
func (writer IndexWriterImpl) writeNodeData(dataWriter io.Writer, node Node) (n int, err error) {
	if n, err = writer.nodeWriter.Write(node, dataWriter); err != nil {
		return n, fmt.Errorf("%w: `%v`: %v", ErrWriter, node.Rune(), err)
	}

	return n, nil
}

// writeNode writes node relation into index writer as well as writes node data using node writer.
// Takes index writer, node index, node parent index and node itself.
// Returns written bytes count or error if happened.
func (writer IndexWriterImpl) writeNode(node Node, indexWriter, dataWriter io.Writer) (n int, err error) {
	var written int

	if written, err = writer.writeNodeToIndex(indexWriter, node); err != nil {
		return 0, err
	}

	n += written

	if written, err = writer.writeNodeData(dataWriter, node); err != nil {
		return n, err
	}

	n += written

	return n, nil
}

// recurseWriteNode writes nodes recursive into specified writer.
// Takes writer instance, starting index and node to write itself.
// Returns last used node index after all children, written bytes sum or error if happened.
func (writer IndexWriterImpl) recurseWriteNode(node Node, indexWriter, dataWriter io.Writer) (n int, err error) {
	bytes := 0
	n = 0

	if writer.nodeWriter == nil {
		return 0, fmt.Errorf("%w: no nodes writer", ErrWriter)
	}

	if bytes, err = writer.writeNode(node, indexWriter, dataWriter); err != nil {
		return n, err
	}

	n += bytes

	for childRune, child := range node.Children() {
		if bytes, err = writer.recurseWriteNode(child, indexWriter, dataWriter); err != nil {
			return n, fmt.Errorf("%w: `%v`: %v", ErrWriter, childRune, err)
		}

		n += bytes
	}

	return n, nil
}

// WriteInto writes index data into specified writer.
func (writer IndexWriterImpl) Write(idx Index, indexWriter, dataWriter io.Writer) (n int, err error) {
	var (
		totalBytes int
		bytes      int
	)

	if writer.nodeWriter == nil {
		return 0, fmt.Errorf("%w: no nodes writer", ErrWriter)
	}

	for rootRune, rootNode := range idx.Children() {
		if bytes, err = writer.recurseWriteNode(rootNode, indexWriter, dataWriter); err != nil {
			return totalBytes, fmt.Errorf("%w: `%v`: %v", ErrWriter, rootRune, err)
		}

		totalBytes += bytes
	}

	return totalBytes, nil
}
