// Package testdawg builds valid dawgdic-format DAWG fixtures for tests.
//
// Build lays out keys using the double-array scheme (children:
// base^label, a node's value is a separate value unit in slot base with
// the isLeaf bit set), so fixtures are read by the same DAWG code as
// real pymorphy2 dictionaries.
package testdawg

import (
	"bytes"
	"encoding/binary"
	"sort"
)

const (
	isLeafBit  = 1 << 31
	hasLeafBit = 1 << 8
)

// node is one element of the test DAWG (trie). label is the transition
// byte from the parent.
type node struct {
	label    byte
	children map[byte]*node
	hasLeaf  bool
	value    uint32
}

// Build assembles a dictionary+guide from a set of keys with integer values.
func Build(keys map[string]uint32) ([]uint32, []byte) {
	root := &node{children: make(map[byte]*node)}
	keysSorted := make([]string, 0, len(keys))
	for k := range keys {
		keysSorted = append(keysSorted, k)
	}
	sort.Strings(keysSorted)
	for _, key := range keysSorted {
		val := keys[key]
		n := root
		for i := 0; i < len(key); i++ {
			l := key[i]
			child, ok := n.children[l]
			if !ok {
				child = &node{label: l, children: make(map[byte]*node)}
				n.children[l] = child
			}
			n = child
		}
		n.hasLeaf = true
		n.value = val
	}

	placed := map[*node]uint32{}
	used := map[uint32]bool{}
	offsetOf := map[*node]uint32{}
	valueSlot := map[*node]uint32{}
	maxIndex := uint32(0)

	placed[root] = 0
	used[0] = true
	queue := []*node{root}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		index := placed[n]

		labels := make([]byte, 0, len(n.children))
		for l := range n.children {
			labels = append(labels, l)
		}
		sort.Slice(labels, func(i, j int) bool { return labels[i] < labels[j] })

		base := firstFitBase(used, labels, n.hasLeaf)
		offsetOf[n] = index ^ base
		if n.hasLeaf {
			used[base] = true
			valueSlot[n] = base
			if base > maxIndex {
				maxIndex = base
			}
		}
		for _, l := range labels {
			child := n.children[l]
			childIndex := base ^ uint32(l)
			used[childIndex] = true
			placed[child] = childIndex
			if childIndex > maxIndex {
				maxIndex = childIndex
			}
			queue = append(queue, child)
		}
	}

	maxIndex++
	dict := make([]uint32, maxIndex)

	for n, index := range placed {
		off := offsetOf[n]
		if off >= 1<<22 {
			panic("testdawg: offset does not fit 22 bits")
		}
		unit := off<<10 | uint32(n.label)
		if n.hasLeaf {
			unit |= hasLeafBit
		}
		dict[index] = unit
	}
	for n, slot := range valueSlot {
		dict[slot] = n.value | isLeafBit
	}

	guide := make([]byte, maxIndex*2)
	for n, index := range placed {
		labels := make([]byte, 0, len(n.children))
		for l := range n.children {
			labels = append(labels, l)
		}
		sort.Slice(labels, func(i, j int) bool { return labels[i] < labels[j] })
		if len(labels) > 0 {
			guide[index*2] = labels[0]
		}
		for k, l := range labels {
			child := n.children[l]
			if k+1 < len(labels) {
				guide[placed[child]*2+1] = labels[k+1]
			}
		}
	}

	return dict, guide
}

// firstFitBase picks a base: the base^label slots (and base itself when
// hasValue) must all be free.
func firstFitBase(used map[uint32]bool, labels []byte, hasValue bool) uint32 {
	for b := uint32(0); ; b++ {
		if hasValue && used[b] {
			continue
		}
		ok := true
		for _, l := range labels {
			if used[b^uint32(l)] {
				ok = false
				break
			}
		}
		if ok {
			return b
		}
	}
}

// Marshal serializes a dictionary+guide into the words.dawg stream format:
// [uint32 count][count×uint32 units][uint32 gsize][gsize×2 bytes of guide].
func Marshal(dict []uint32, guide []byte) []byte {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(dict)))
	for _, u := range dict {
		_ = binary.Write(&buf, binary.LittleEndian, u)
	}
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(guide)/2))
	buf.Write(guide)
	return buf.Bytes()
}
