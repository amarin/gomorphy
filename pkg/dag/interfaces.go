package dag

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
