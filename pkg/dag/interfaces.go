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
	// ID returns node ID.
	ID() uint32
	// SetID sets new node ID.
	SetID(newID uint32)
	// Rune returns node rune.
	Rune() rune
	// SetRune sets new node rune. Use before adding to index, rune changes is not propagated onto parent and index.
	SetRune(nodeRune rune)
	// Parent returns parent node. If node is 1st level node parent returns nil.
	Parent() Node
	// SetParent sets new node parent.
	SetParent(Node)
	// Data returns node related data.
	Data() interface{}
	// Set directly adds child node.
	Set(node Node)
	// SetData sets new node data.
	SetData(data interface{})
}

// NodeConstructor defines node constructor function interface.
// Used by Index implementations to create new nodes.
type NodeConstructor func(parent Node, nodeRune rune, data interface{}) Node

// Index defines nodes index interface.
type Index interface {
	Container
	NodeConstructor() NodeConstructor
	// SetNodeConstructor sets new node constructor.
	SetNodeConstructor(constructor NodeConstructor)
	// Get returns node by its index or error if no such node found.
	Get(nodeIdx uint32) (node Node, err error)
	// Set directly adds node to index.
	// Silently extends index if specified node ID is greater then index size.
	Set(node Node) (err error)
	// BuildNode returns new node using specified parameters or returns error.
	BuildNode(parent Node, nodeRune rune, data interface{}) (Node, error)
}

// NodeWriter specifies node writer interface. It required to use with IndexWriter.
type NodeWriter interface {
	// Write writes node data into specified io.Writer.
	Write(node Node, w io.Writer) (n int, err error)
}

// IndexWriter specifies index writer interface.
// It writes only nodes relations requiring separate NodeWriter to write nodes data.
type IndexWriter interface {
	SetNodeWriter(nodeWriter NodeWriter)
	// Write writes index into specified writer.
	Write(idx Index, indexWriter, dataWriter io.Writer) (n int, err error)
}

// NodeReader specifies node reader interface. It required to use with IndexReader.
type NodeReader interface {
	// Read reads specified node data from io.Reader.
	Read(node Node, reader io.Reader) (n int, err error)
}

// IndexReader specifies index reader interface.
// It reads only nodes relations itself requiring separate NodeReader to load nodes data.
type IndexReader interface {
	SetNodeReader(nodeWriter NodeReader)
	// Read reads index and nodes data from readers into index. Returns taken bytes or error if happened.
	Read(index Index, indexReader, dataReader io.Reader) (n int, err error)
}
