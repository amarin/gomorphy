package dag

import (
	"io"
)

// Container defines nodes container interface. It applicable both for nodes index (aka root node) and node itself.
type Container interface {
	// Children returns node children mapping.
	Children() map[rune]Node
	// Add adds runes sequence into container. Returns final node filled with node data or error if add caused error.
	Add([]rune, interface{}) (Node, error)
	// Fetch lookups runes sequence in container. If found returns final node or error if not found.
	Fetch([]rune) (Node, error)
}

// Node implements container methods as well as node specific rune fetcher and parent resolver.
type Node interface {
	Container
	// Rune returns node rune.
	Rune() rune
	// Parent returns parent node. If node is 1st level node parent returns nil.
	Parent() Node
	// Data returns node related data.
	Data() interface{}
	// SetData sets new node data.
	SetData(data interface{})
}

// NodeConstructor defines node constructor function interface.
// Used by Index implementations to create new nodes.
type NodeConstructor func(parent Node, nodeRune rune, data interface{}) Node

// Index defines nodes index interface.
type Index interface {
	Container
	// SetNodeConstructor sets new node constructor.
	SetNodeConstructor(constructor NodeConstructor)
}

// NodeWriter specifies node writer interface. It required to use with IndexWriter.
type NodeWriter interface {
	// Write writes node with specified index into specified writer.
	Write(idx uint32, node Node) (n int, err error)
}

// IndexWriter specifies index writer interface.
// It writes only nodes relations requiring separate NodeWriter to write nodes data.
type IndexWriter interface {
	SetNodeWriter(nodeWriter NodeWriter)
	// WriteInto writes index into specified writer.
	WriteInto(idx Index, writer io.Writer) (n int, err error)
}

// NodeReader specifies node reader interface. It required to use with IndexReader.
type NodeReader interface {
	// Read reads node data with specified index. Returns taken bytes, node data or read error.
	Read(idx uint32) (n int, nodeData interface{}, err error)
}

// IndexReader specifies index reader interface.
// It reads only nodes relations itself requiring separate NodeReader to load nodes data.
type IndexReader interface {
	SetNodeReader(nodeWriter NodeReader)
	// ReadFrom reads index from specified reader. Returns taken bytes, index
	ReadFrom(index Index, reader io.Reader) (n int, err error)
}
