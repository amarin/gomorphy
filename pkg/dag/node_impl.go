package dag

import (
	"errors"
	"fmt"
)

// ErrNode indicates some node related errors.
var ErrNode = errors.New("node")

// NodeImpl implements Node routines.
type NodeImpl struct {
	parent   Node
	id       uint32
	rune     rune
	children map[rune]Node
	data     interface{}
}

func (node *NodeImpl) prepareInternals() {
	if node.children == nil {
		node.children = make(map[rune]Node)
	}
}

// Children returns node children mapping.
func (node NodeImpl) Children() map[rune]Node {
	node.prepareInternals()

	return node.children
}

// ID returns node ID.
func (node NodeImpl) ID() uint32 {
	return node.id
}

// SetID sets new node ID.
func (node *NodeImpl) SetID(newID uint32) {
	node.id = newID
}

// Rune returns node rune.
func (node NodeImpl) Rune() rune {
	return node.rune
}

// Parent returns parent node. If node is 1st level node parent returns nil.
func (node NodeImpl) Parent() Node {
	return node.parent
}

// SetParent sets new node parent.
func (node *NodeImpl) SetParent(parent Node) {
	node.parent = parent
}

// Data returns node related data.
func (node NodeImpl) Data() interface{} {
	return node.data
}

// SetData sets new node data.
func (node *NodeImpl) SetData(data interface{}) {
	node.data = data
}

// Add adds runes sequence into container. Returns final node filled with node data or error if add caused error.
func (node *NodeImpl) Add(runes []rune, nodeData interface{}) (newNode Node, err error) {
	node.prepareInternals()

	switch len(runes) {
	case 0:
		return nil, fmt.Errorf("%w: empty runes", ErrNode)
	case 1:
		newNode = DefaultNodeConstructor(node, runes[0], nodeData)
		node.Set(newNode)

		return newNode, nil
	default:
		mediator := DefaultNodeConstructor(node, runes[0], nil)

		node.Set(mediator)

		if newNode, err = mediator.Add(runes[1:], nodeData); err != nil {
			return newNode, fmt.Errorf("%w: create: %v", ErrNode, err)
		}

		return newNode, nil
	}
}

// Fetch lookups runes sequence in container. If found returns final node or error if not found.
func (node *NodeImpl) Fetch(runes []rune) (child Node, err error) {
	var ok bool

	node.prepareInternals()

	if len(runes) == 0 { // fetch reached final node
		return node, nil
	}

	firstChar := runes[0]
	if child, ok = node.children[firstChar]; ok {
		if child, err = child.Fetch(runes[1:]); err != nil {
			return nil, fmt.Errorf("%w: `%v` error: %v", ErrNode, firstChar, err)
		}

		return child, nil
	}

	return nil, fmt.Errorf("%w: `%v` not found", ErrNode, firstChar)
}

// Set directly adds child node.
func (node *NodeImpl) Set(n Node) {
	node.children[n.Rune()] = n
	if n.Parent() == nil || n.Parent().ID() != node.ID() {
		n.SetParent(node)
	}
}

// DefaultNodeConstructor builds Node using default NodeImpl type.
func DefaultNodeConstructor(parent Node, nodeRune rune, data interface{}) Node {
	node := &NodeImpl{
		parent:   parent,
		id:       0,
		rune:     nodeRune,
		children: nil,
		data:     data,
	}
	if parent != nil {
		parent.Set(node)
	}

	return node
}
