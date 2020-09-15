package words

// Index nodes used to store single rune of words. Each node connected to parent node,
// containing previous rune of indexed words and set of children nodes containing next runes in words.
// If rune is a last rune in some word, such node also stores grammemes sets
// for all possible words sharing that rune.

import (
	"fmt"

	"github.com/amarin/gomorphy/internal/grammemes"
	"github.com/amarin/gomorphy/internal/text"
)

// Node implements index node routines.
type Node struct {
	children  *NodesContainer
	char      rune
	parent    *Node
	grammemes grammemes.ListList
}

// NewMappingNode creates new index node using NodeMapping container for children access.
func NewMappingNode(parent *Node, character rune) *Node {
	node := &Node{
		children:  NewNodeContainer(nil),
		char:      character,
		parent:    parent,
		grammemes: make(grammemes.ListList, 0),
	}

	node.children.parent = node

	return node
}

// Word returns full word ending in current node.
// Calls parent recursively to arrive to root node.
func (node Node) Word() text.RussianText {
	if node.parent == nil {
		return text.RussianText([]rune{node.char})
	}

	return node.parent.Word() + text.RussianText([]rune{node.char}) // Сложить текст родителя и текущий символ узла.
}

// Len returns lengths of direct children.
func (node Node) Len() int {
	return node.children.Len()
}

// String returns string representation of index node.
func (node Node) String() string {
	return "N{" + string(node.Word()) + "}"
}

// GoString returns string representation of index node.
func (node Node) GoString() string {
	return fmt.Sprintf("N(%#p){%#p,%v,\n%v}", &node, node.parent, node.Word(), node.children.GoString())
}

// Parent returns pointer to parent node of current node.
// Used to restore whole word when search.
func (node Node) Parent() *Node {
	return node.parent
}

// Root returns root ascendant node for current.
// Calls ascendants recursively.
func (node *Node) Root() *Node {
	if node.parent == nil {
		return node
	}

	return node.parent.Root()
}

// Rune returns indexed rune for current node.
func (node Node) Rune() rune {
	return node.char
}

// Forms returns set of grammemes set stored in current node.
// If current node is not final, returns empty set.
func (node Node) Forms() grammemes.ListList {
	return node.grammemes
}

// AddGrammemes adds grammemes set into current node.
// Node became known word final, but can act as intermediate node too.
func (node *Node) AddGrammemes(grammemes *grammemes.List) error {
	node.grammemes.Append(grammemes)
	return nil
}

// Child returns descend node with requested rune.
// If no such node among children found, creates and returns new empty node.
func (node *Node) Child(char rune) (child *Node) {
	return node.children.Child(char)
}

// Slice returns list of current node and all its descendants.
func (node *Node) Slice() NodeList {
	nodes := make(NodeList, 0)
	nodes = append(nodes, node)
	nodes = append(nodes, node.children.Slice()...)

	return nodes
}

// Children returns container of direct children.
func (node *Node) Children() *NodesContainer {
	return node.children
}

// IsFinalNode returns true if node contains any grammemes set i.e. is an ending node for some word.
func (node *Node) IsFinalNode() bool {
	return len(node.grammemes) > 0
}

// Endings returns final node list starting from current node.
func (node *Node) EndingsNodes() *NodeList {
	endings := NewNodeList()
	if node.IsFinalNode() {
		*endings = append(*endings, node)
	}

	childrenEndings := node.children.EndingsNodes()
	*endings = append(*endings, *childrenEndings...)

	return endings
}
