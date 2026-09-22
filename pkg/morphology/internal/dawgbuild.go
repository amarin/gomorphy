package internal

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

// errDAWGBuild reports that the automaton cannot be laid out in a
// double-array: the range of representable offsets/slots has been exhausted.
var errDAWGBuild = errors.New("dawg: cannot place nodes in double-array")

// BuildDAWG builds a minimal DAWG (dawgdic format: dictionary uint32[] +
// guide byte[]) from a set of keys.
//
// The algorithm follows dawgdic (s-yata/dawgdic):
//  1. Keys are sorted ascending and incrementally (Daciuk's method) assembled
//     into a list-form ternary automaton. The register is keyed by the
//     signature of the WHOLE sibling chain — a node is only merged together
//     with its entire sibling list. This guarantees that a shared (merged)
//     node has exactly one entry point with all siblings in the same
//     composition, so the double-array layout can be built via DFS with
//     reuse of the first child's base.
//  2. The double-array is laid out depth-first: a node's children (other
//     than the first, if the first is a merged node) get free slots at
//     base^label; for a merged first child, the previously chosen base is
//     reused (via the link table), which gives the same slot from all parents.
//
// The value of every terminal node is 0: the payload is stored as a key
// suffix after PayloadSeparator (see SimilarItems). The order of the keys
// argument is not preserved (the slice is sorted in place).
func BuildDAWG(keys []string) (*DAWG, error) {
	return buildDAWGWithPayload(keys)
}

// BuildDAWGWithValues builds a minimal DAWG (dawgdic format) from
// (key, value) pairs and encapsulates each value as payload: every key in
// the DAWG turns into key + PayloadSeparator + base64(value), with value
// encoded as a 4-byte big-endian uint32 (the words.dawg payload shape).
//
// Thin wrapper over BuildDAWGWithValuesBytes so the payload-encapsulation
// rule lives in exactly one place. Returns (*DAWG, error).
func BuildDAWGWithValues(keys []string, values []uint32) (*DAWG, error) {
	if len(keys) != len(values) {
		return nil, fmt.Errorf("dawg: keys and values must have same length")
	}

	valuesBytes := make([][]byte, len(values))
	for i, v := range values {
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, v)
		valuesBytes[i] = b
	}
	return BuildDAWGWithValuesBytes(keys, valuesBytes)
}

// BuildDAWGWithValuesBytes builds a minimal DAWG (dawgdic format) from
// (key, value) pairs where each value is an arbitrary-length byte payload:
// every key in the DAWG turns into key + PayloadSeparator + base64(value) —
// the exact shape SimilarItems/ValuesForIndex decode.
//
// The algorithm fully follows BuildDAWG, but the keys are modified before
// assembly. Returns (*DAWG, error).
func BuildDAWGWithValuesBytes(keys []string, values [][]byte) (*DAWG, error) {
	if len(keys) != len(values) {
		return nil, fmt.Errorf("dawg: keys and values must have same length")
	}

	payloadKeys := make([]string, len(keys))
	for i, k := range keys {
		payloadKeys[i] = k + string([]byte{PayloadSeparator}) + base64.StdEncoding.EncodeToString(values[i])
	}

	return buildDAWGWithPayload(payloadKeys)
}

// BuildIntDAWG builds a dawgdic value DAWG (the IntDAWG layout pymorphy2
// uses for p_t_given_w.intdawg): every key maps to a uint32 value stored
// in its terminal node, read back with Find/WalkValues. keys must be
// unique (any order); every value must be < 1<<31 (the top bit is the
// dawgdic leaf flag).
func BuildIntDAWG(keys []string, values []uint32) (*DAWG, error) {
	if len(keys) != len(values) {
		return nil, fmt.Errorf("dawg: keys and values must have same length")
	}
	order := make([]int, len(keys))
	for i := range order {
		order[i] = i
		if values[i] >= isLeafBit {
			return nil, fmt.Errorf("dawg: value %d for key %q exceeds 31 bits", values[i], keys[i])
		}
	}
	sort.Slice(order, func(a, b int) bool { return keys[order[a]] < keys[order[b]] })
	sk := make([]string, len(keys))
	sv := make([]uint32, len(keys))
	for i, j := range order {
		sk[i], sv[i] = keys[j], values[j]
		if i > 0 && sk[i] == sk[i-1] {
			return nil, fmt.Errorf("dawg: duplicate key %q", sk[i])
		}
	}
	b := newDawgBuilder()
	b.insertKeyValues(sk, sv, nil)
	return b.compile()
}

// buildDAWGWithPayload — shared code path for BuildDAWG and BuildDAWGWithValues.
func buildDAWGWithPayload(keys []string) (*DAWG, error) {
	sort.Strings(keys)

	b := newDawgBuilder()
	b.insertKeys(keys, nil)

	return b.compile()
}

// newDawgBuilder creates a dawgBuilder with pre-allocated buffers (a shared
// entry point for BuildDAWG* and BuildDAWGWithValuesProgress).
func newDawgBuilder() *dawgBuilder {
	b := &dawgBuilder{
		register:  make(map[string]int32, 1<<20),
		merged:    make([]bool, 1),
		path:      make([]int32, 0, 32),
		sigBuf:    make([]byte, 0, 64),
		labelsBuf: make([]byte, 0, 16),
	}
	b.root = b.newNode(0)
	b.path = append(b.path, b.root)

	return b
}

// insertKeys inserts sorted keys into the builder (incrementally comparing
// the common prefix with the previous key, Daciuk-style sibling merging).
// If onInserted is not nil, it is called after every insertion with the
// 0-based index of the key just inserted — used for progress callbacks.
func (b *dawgBuilder) insertKeys(keys []string, onInserted func(i int)) {
	b.insertKeyValues(keys, nil, onInserted)
}

// insertKeyValues is like insertKeys, but additionally assigns each
// terminal node the corresponding value from values (nil for payload
// DAWGs, whose terminal values are always 0).
func (b *dawgBuilder) insertKeyValues(keys []string, values []uint32, onInserted func(i int)) {
	for i, k := range keys {
		common := 0
		for common < len(b.lastKey) && common < len(k) && b.lastKey[common] == k[common] {
			common++
		}
		b.closeSuffix(common)
		for j := common; j < len(k); j++ {
			b.appendByte(k[j])
		}
		b.nodes[b.path[len(b.path)-1]].leaf = true
		if values != nil {
			b.nodes[b.path[len(b.path)-1]].value = values[i]
		}
		b.lastKey = k

		if onInserted != nil {
			onInserted(i)
		}
	}
	b.closeSuffix(0)
}

// dawgBuilder builds a list-form DAWG (flat nodes with sibling lists) and
// then lays it out into a double-array.
type dawgBuilder struct {
	nodes []dbNode

	root     int32
	path     []int32          // nodes of the current lastKey: path[0]=root
	lastKey  string           // the last key inserted
	register map[string]int32 // signature of a closed sibling chain → first node
	merged   []bool           // first node of a chain merged with another chain

	sigBuf    []byte
	labelsBuf []byte
}

// dbNode — a list-form DAWG node. A node's children form a sibling chain via next;
// the first child is nodes[first].
type dbNode struct {
	label byte
	first int32  // first child (head of the chain); 0 — no children
	next  int32  // next sibling in the parent's chain; 0 — none
	leaf  bool   // the node is terminal (has a value)
	value uint32 // the terminal's value (BuildIntDAWG); 0 for payload DAWGs
}

func (b *dawgBuilder) newNode(label byte) int32 {
	id := int32(len(b.nodes))
	b.nodes = append(b.nodes, dbNode{label: label})
	b.merged = append(b.merged, false)
	return id
}

// appendByte adds a new node as the first child of the path's parent.
func (b *dawgBuilder) appendByte(label byte) {
	id := b.newNode(label)
	parent := b.path[len(b.path)-1]
	b.nodes[id].next = b.nodes[parent].first
	b.nodes[parent].first = id
	b.path = append(b.path, id)
}

// closeSuffix closes (minimizes) lastKey's nodes past the common part of
// length common: nodes are popped off the path bottom-up and merged/registered.
func (b *dawgBuilder) closeSuffix(common int) {
	for len(b.path) > common+1 {
		n := b.path[len(b.path)-1]
		b.path = b.path[:len(b.path)-1]
		b.replaceOrRegister(n)
	}
}

// replaceOrRegister closes node n (the parent's first child): if n's sibling
// chain is already registered, it redirects the parent's edge to it,
// otherwise it registers n.
func (b *dawgBuilder) replaceOrRegister(n int32) {
	if n == b.root {
		return
	}
	sig := b.chainSig(n)
	if twin, ok := b.register[sig]; ok {
		b.nodes[b.path[len(b.path)-1]].first = twin
		b.merged[twin] = true
		return
	}
	b.register[sig] = n
}

// chainSig computes the signature of the sibling chain starting at node n:
// for each sibling — its label, flags (terminal, has-next-sibling), and its
// child's id. The child is compared by id because children are closed
// earlier (bottom-up), and after merges identical structures share the same id.
func (b *dawgBuilder) chainSig(n int32) string {
	b.sigBuf = b.sigBuf[:0]
	for n != 0 {
		b.sigBuf = append(b.sigBuf, b.nodes[n].label)
		f := byte(0)
		if b.nodes[n].leaf {
			f |= 1
		}
		if b.nodes[n].next != 0 {
			f |= 2
		}
		v := b.nodes[n].value
		if v != 0 {
			f |= 4
		}
		b.sigBuf = append(b.sigBuf, f)
		if v != 0 {
			b.sigBuf = append(b.sigBuf, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
		}
		child := b.nodes[n].first
		b.sigBuf = append(b.sigBuf, byte(child>>24), byte(child>>16), byte(child>>8), byte(child))
		n = b.nodes[n].next
	}
	return string(b.sigBuf)
}

// compileWithProgress lays out the minimized list-form DAWG into a double-array
// (dictionary uint32[]) and builds guide. Calls the progress callback.
func (b *dawgBuilder) compileWithProgress(progress func(processed, total int)) (*DAWG, error) {
	totalNodes := int32(len(b.nodes))
	return b.compileImpl(totalNodes, progress)
}

// compileWithTotal lays out the DAWG and rescales progress to totalKeys.
// This is needed when the progress callback expects the same total as during
// the key-insertion phase.
func (b *dawgBuilder) compileWithTotal(totalKeys int, progress func(processed, total int)) (*DAWG, error) {
	totalNodes := int32(len(b.nodes))
	// Wrap progress to convert nodes->keys scale
	wrappedProgress := func(processedNodes, _ int) {
		if progress != nil {
			// Map processed nodes to keys scale
			ratio := float64(processedNodes) / float64(totalNodes)
			processedKeys := int(ratio * float64(totalKeys))
			if processedKeys > totalKeys {
				processedKeys = totalKeys
			}
			// Pass totalKeys as second arg so main.go knows we're in compile phase
			progress(processedKeys, totalKeys)
		}
	}
	return b.compileImpl(totalNodes, wrappedProgress)
}

// compileImpl — shared implementation for compileWithProgress and compileWithTotal.
func (b *dawgBuilder) compileImpl(totalNodes int32, progress func(processed, total int)) (*DAWG, error) {
	if b.nodes[b.root].first == 0 {
		dic := []uint32{1 << 10}
		dic[0] = 1 << 10
		return NewDAWG(dic, nil), nil
	}

	p := newPlacer(b, totalNodes, progress)
	if !p.place(b.root, 0) {
		return nil, errDAWGBuild
	}

	if progress != nil {
		progress(int(p.processedNodes), 0)
	}

	guide := b.buildGuide(p.dic)
	if guide == nil {
		return nil, errDAWGBuild
	}
	return NewDAWG(p.dic, guide), nil
}

// placer holds the working state for laying out a minimized list-form DAWG
// into a double-array (dic): the allocator, the link table for merged
// (shared) first children, and progress bookkeeping. Split out from
// compileImpl so node placement can be tested independently of guide
// building.
type placer struct {
	b *dawgBuilder

	dic   []uint32
	alloc *slotAllocator
	link  map[linkKey]uint32

	processedNodes int32
	progressEvery  int32
	progress       func(processed, total int)
}

// linkKey identifies a reusable base: the shared first child plus the
// value stored in the base's value unit (0 for payload DAWGs, so their
// layout is unchanged).
type linkKey struct {
	first int32
	value uint32
}

func newPlacer(b *dawgBuilder, totalNodes int32, progress func(processed, total int)) *placer {
	progressEvery := totalNodes / 10000
	if progressEvery < 10 {
		progressEvery = 10
	}

	return &placer{
		b:             b,
		dic:           []uint32{0},
		alloc:         newSlotAllocator(),
		link:          make(map[linkKey]uint32),
		progressEvery: progressEvery,
		progress:      progress,
	}
}

// place recursively lays out node n at double-array slot index, reusing a
// previously chosen base when n's first child is a merged (shared) node
// whose base is still valid at this index (Daciuk-style base reuse).
func (p *placer) place(n int32, index uint32) bool {
	node := &p.b.nodes[n]
	first := node.first
	lk := linkKey{first: first, value: node.value}

	if first != 0 && p.b.merged[first] {
		if base, ok := p.link[lk]; ok && encodable(index^base) {
			setAt(&p.dic, index, unitAt(index^base, node.label, node.leaf))
			p.tick()
			return true
		}
	}

	p.b.labelsBuf = p.b.labelsBuf[:0]
	for e := first; e != 0; e = p.b.nodes[e].next {
		p.b.labelsBuf = append(p.b.labelsBuf, p.b.nodes[e].label)
	}

	base, ok := p.alloc.alloc(index, p.b.labelsBuf)
	if !ok {
		return false
	}

	setAt(&p.dic, index, unitAt(index^base, node.label, node.leaf))
	if node.leaf {
		setAt(&p.dic, base, isLeafBit|node.value)
	}
	if first != 0 && p.b.merged[first] {
		p.link[lk] = base
	}

	for e := first; e != 0; e = p.b.nodes[e].next {
		if !p.place(e, base^uint32(p.b.nodes[e].label)) {
			return false
		}
	}

	p.tick()
	return true
}

func (p *placer) tick() {
	p.processedNodes++
	if p.progress != nil && p.processedNodes%p.progressEvery == 0 {
		p.progress(int(p.processedNodes), 0)
	}
}

// compile lays out the minimized list-form DAWG into a double-array
// (dictionary uint32[]) and builds guide.
func (b *dawgBuilder) compile() (*DAWG, error) {
	return b.compileWithProgress(nil)
}

// unitAt assembles a node's transition unit: the offset field (rel) plus
// the label and the terminal flag (presence of a value edge).
func unitAt(rel uint32, label byte, hasLeaf bool) uint32 {
	var unit uint32
	if rel < 1<<21 {
		unit = rel<<10 | uint32(label)
	} else {
		unit = rel<<2 | extensionBit | uint32(label)
	}
	if hasLeaf {
		unit |= hasLeafBit
	}
	return unit
}

// encodable checks whether the offset field is representable in a
// dictionary unit (as in dawgdic's DictionaryUnit::set_offset): values
// < 1<<21 directly, larger ones only if a multiple of 256 (extended
// format).
func encodable(rel uint32) bool {
	if rel < 1<<21 {
		return true
	}
	return rel < 1<<29 && rel&0xFF == 0
}

// buildGuide builds the guide (2 bytes per node: first child + next
// sibling) from an already laid-out double-array. Children are
// enumerated in ascending label order (as in dawgdic after the chain is
// reversed during registration), so the enumeration order matches the
// reference trie builder (testdawg).
func (b *dawgBuilder) buildGuide(dic []uint32) []byte {
	guide := make([]byte, len(dic)*2)
	fixed := make([]byte, (len(dic)+7)/8)

	var walk func(n int32, index uint32) bool
	walk = func(n int32, index uint32) bool {
		if fixed[index>>3]&(1<<(index&7)) != 0 {
			return true
		}
		fixed[index>>3] |= 1 << (index & 7)
		node := &b.nodes[n]
		if node.first == 0 {
			return true
		}
		order := make([]int32, 0, 8)
		for e := node.first; e != 0; e = b.nodes[e].next {
			order = append(order, e)
		}
		sort.Slice(order, func(i, j int) bool {
			return b.nodes[order[i]].label < b.nodes[order[j]].label
		})
		guide[index*2] = b.nodes[order[0]].label
		for k, e := range order {
			childIndex := index ^ offset(dic[index]) ^ uint32(b.nodes[e].label)
			if childIndex >= uint32(len(dic)) {
				return false
			}
			if !walk(e, childIndex) {
				return false
			}
			if k+1 < len(order) {
				guide[childIndex*2+1] = b.nodes[order[k+1]].label
			}
		}
		return true
	}

	if !walk(b.root, 0) {
		return nil
	}
	return guide
}

// setAt assigns unit into dic, growing the slice if necessary.
func setAt(dic *[]uint32, index, unit uint32) {
	if uint32(len(*dic)) <= index {
		*dic = append(*dic, make([]uint32, index-uint32(len(*dic))+1)...)
	}
	(*dic)[index] = unit
}
