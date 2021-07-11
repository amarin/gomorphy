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
	nodeConstructor NodeConstructor
}

func (idx IndexImpl) SetNodeConstructor(constructor NodeConstructor) {
	idx.nodeConstructor = constructor
}

func (idx IndexImpl) Children() map[rune]Node {
	return idx.children
}

// addNode adds new root node unconditionally. Replaces existed node if already existed.
func (idx IndexImpl) addNode(firstRune rune, data interface{}) (node Node, err error) {
	if idx.nodeConstructor == nil {
		return nil, fmt.Errorf("%w: no constructor", ErrIndex)
	}

	node = idx.nodeConstructor(nil, firstRune, data)
	idx.children[firstRune] = node

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

// Add adds runes sequence into index. Returns final node or error if add caused error.
func (idx IndexImpl) Add(runes []rune, nodeData interface{}) (node Node, err error) {
	var firstRuneNode Node

	if len(runes) == 0 {
		return nil, fmt.Errorf("%w: add: empty runes sequence", ErrIndex)
	}

	firstRune := runes[0]
	if firstRuneNode, err = idx.getOrAddNode(firstRune); err != nil {
		return nil, err
	}

	if node, err = firstRuneNode.Add(runes[1:], nodeData); err != nil {
		return nil, fmt.Errorf("%w: add: %v", ErrIndex, err)
	}

	return node, nil
}

// Fetch lookups runes sequence in container. If found returns final node or error if not found.
func (idx IndexImpl) Fetch(runes []rune) (node Node, err error) {
	if len(runes) == 0 {
		return nil, fmt.Errorf("%w: fetch: empty runes sequence", ErrIndex)
	}

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
