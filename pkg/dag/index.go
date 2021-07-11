package dag

import (
	"errors"
	"fmt"
)

// ErrIndex identifies index-related errors.
var ErrIndex = errors.New("index")

// IndexImpl implements whole node index.
type IndexImpl struct {
	children        map[rune]Node
	childrenIdx     []Node
	nodeConstructor NodeConstructor
}

// NewIndex creates new DAG words index.
func NewIndex() Index {
	return new(IndexImpl)
}

// NodeConstructor return attached node constructor.
func (idx IndexImpl) NodeConstructor() NodeConstructor {
	return idx.nodeConstructor
}

func (idx *IndexImpl) SetNodeConstructor(constructor NodeConstructor) {
	idx.nodeConstructor = constructor
}

func (idx IndexImpl) Children() map[rune]Node {
	return idx.children
}

// indexNode adds node to childrenIdx and invokes Node.SetID() with node ID in index.
func (idx *IndexImpl) indexNode(node Node) {
	nodeIdx := uint32(len(idx.childrenIdx))
	idx.childrenIdx = append(idx.childrenIdx, node)

	node.SetID(nodeIdx)
}

// addNode adds new root node unconditionally. Replaces existed node if already existed.
func (idx *IndexImpl) addNode(firstRune rune, data interface{}) (node Node, err error) {
	if idx.nodeConstructor == nil {
		return nil, fmt.Errorf("%w: no constructor", ErrIndex)
	}

	node = idx.nodeConstructor(nil, firstRune, data)
	idx.children[firstRune] = node
	idx.indexNode(node)

	return node, nil
}

// getOrAddNode get existed or add new root node.
func (idx IndexImpl) getOrAddNode(firstRune rune) (firstRuneNode Node, err error) {
	var ok bool

	if firstRuneNode, ok = idx.children[firstRune]; !ok {
		return idx.addNode(firstRune, nil)
	}

	return firstRuneNode, nil
}

func (idx *IndexImpl) prepareInternals() {
	if idx.children == nil {
		idx.children = make(map[rune]Node)
	}

	if idx.childrenIdx == nil {
		idx.childrenIdx = make([]Node, 0)
	}
}

// Add adds runes sequence into index. Returns final node or error if add caused error.
func (idx *IndexImpl) Add(runes []rune, nodeData interface{}) (node Node, err error) {
	var lastNode Node

	if len(runes) == 0 {
		return nil, fmt.Errorf("%w: add: empty runes sequence", ErrIndex)
	}

	idx.prepareInternals()

	firstRune := runes[0]
	if lastNode, err = idx.getOrAddNode(firstRune); err != nil {
		return nil, err
	}

	for _, currentRune := range runes[1:] {
		if node, err = lastNode.Add([]rune{currentRune}, nil); err != nil {
			return nil, fmt.Errorf("%w: add: %v", ErrIndex, err)
		}

		idx.indexNode(node)
		lastNode = node
	}

	lastNode.SetData(nodeData)

	return node, nil
}

// Get returns node by its index or error if no such node found.
func (idx *IndexImpl) Get(nodeIdx uint32) (node Node, err error) {
	idx.prepareInternals()

	if int(nodeIdx) >= len(idx.childrenIdx) {
		return nil, fmt.Errorf("%w: get: idx %d out of range", ErrIndex, nodeIdx)
	}

	return idx.childrenIdx[nodeIdx], nil
}

// Set directly adds node to index.
// Silently extends index if specified node ID is greater then index size.
func (idx *IndexImpl) Set(node Node) (err error) {
	idx.prepareInternals()

	if node.Parent() != nil {
		var parent Node

		parentId := node.Parent().ID()
		if parent, err = idx.Get(node.Parent().ID()); err != nil {
			return fmt.Errorf("%w: parent %v not found for node %v", ErrIndex, parentId, node.ID())
		}

		if parent.ID() != node.Parent().ID() {
			return fmt.Errorf("%w: parent %v is not indexed for node %v", ErrIndex, parentId, node.ID())
		}
	}

	if int(node.ID()) >= len(idx.childrenIdx) {
		newIdx := make([]Node, node.ID()+1)
		copy(newIdx, idx.childrenIdx)
		idx.childrenIdx = newIdx
	}

	idx.childrenIdx[node.ID()] = node

	if node.Parent() != nil {
		// add to parent children and return
		node.Parent().Set(node)

		return
	}
	// top node, add to index children
	idx.children[node.Rune()] = node

	return nil
}

// Fetch lookups runes sequence in container. If found returns final node or error if not found.
func (idx *IndexImpl) Fetch(runes []rune) (node Node, err error) {
	if len(runes) == 0 {
		return nil, fmt.Errorf("%w: fetch: empty runes sequence", ErrIndex)
	}

	idx.prepareInternals()

	firstRune := runes[0]

	firstRuneNode, ok := idx.children[firstRune]
	if !ok {
		return nil, fmt.Errorf("%w: fetch: `%v`: not found", ErrIndex, firstRune)
	}

	if node, err = firstRuneNode.Fetch(runes[1:]); err != nil {
		return nil, fmt.Errorf("%w: fetch: `%v`: not found", ErrIndex, string(runes))
	}

	return node, nil
}

// BuildNode returns new node using specified parameters or returns error.
func (idx *IndexImpl) BuildNode(parent Node, nodeRune rune, data interface{}) (Node, error) {
	if idx.nodeConstructor == nil {
		return nil, fmt.Errorf("%w: build: constructor not set", ErrIndex)
	}

	return idx.nodeConstructor(parent, nodeRune, data), nil
}
