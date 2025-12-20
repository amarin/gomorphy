package node

import (
	"bytes"
	"fmt"
)

// Node defines trie node for alphabet power A and nodes set power M&
// It is not thread-safe, guard it outside if expect simultaneous reading and writing.
type Node[A alphabetSize, M wordsSize] struct {
	parent   M
	children map[A]M
}

// New creates new node.
func New[A alphabetSize, M wordsSize]() *Node[A, M] {
	return &Node[A, M]{
		parent:   0,
		children: make(map[A]M),
	}
}

// Bytes returns bytes representation of node data in same format as WriteTo produces.
// Utilizes WriteTo under the hood.
func (node *Node[A, M]) Bytes() ([]byte, error) {
	writer := new(bytes.Buffer)
	if _, err := node.WriteTo(writer); err != nil {
		return nil, err
	}

	return writer.Bytes(), nil
}

// Hex returns hex representation of node data in same format as WriteTo produces.
// Utilizes Bytes under the hood.
func (node *Node[A, M]) Hex() (string, error) {
	data, err := node.Bytes()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X", data), nil
}

// SetParent sets parent node id.
func (node *Node[A, M]) SetParent(nodeId M) {
	node.parent = nodeId
}

// Parent returns parent node id.
func (node *Node[A, M]) Parent() M {
	return node.parent
}

// SetNext register children node id.
func (node *Node[A, M]) SetNext(charId A, nodeId M) {
	node.children[charId] = nodeId
}

// GetNext returns specified character children id.
// If no such node found, returns zero value of M and false indicator.
func (node *Node[A, M]) GetNext(charId A) (nodeId M, found bool) {
	next, ok := node.children[charId]
	return next, ok
}

// HasNext returns true if node have children with specified character id registered.
func (node *Node[A, M]) HasNext(charId A) bool {
	_, ok := node.children[charId]
	return ok
}
