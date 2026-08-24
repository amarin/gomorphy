package build

import "sort"

// tnode is a temporary mutable trie node used only during the build phase.
// Build() compiles the whole tree into flat CSR arrays and drops these.
type tnode struct {
	kids     map[byte]*tnode
	final    bool
	state    uint32 // assigned during CSR compilation
	postings []uint32
	lastPair uint32
	hasLast  bool
}

// insert walks/creates the path for text and returns the terminal node.
func (b *Builder) insert(text []byte) *tnode {
	cur := b.root

	for _, c := range text {
		next, ok := cur.kids[c]
		if !ok {
			next = &tnode{kids: make(map[byte]*tnode)}
			cur.kids[c] = next
		}

		cur = next
	}

	cur.final = true

	return cur
}

// attachPair registers a unique (textID, ancodeID) pair, logs the (pair,
// lemma) attachment and appends the pair to the terminal trie node postings.
func (b *Builder) attachPair(textID, ancodeID, lemma uint32) error {
	key := uint64(textID)<<32 | uint64(ancodeID)

	pair, ok := b.pairIdx[key]
	if !ok {
		pair = uint32(len(b.pairTexts))
		b.pairIdx[key] = pair
		b.pairTexts = append(b.pairTexts, textID)
		b.pairAncodes = append(b.pairAncodes, ancodeID)
	}

	b.attPair = append(b.attPair, pair)
	b.attLemma = append(b.attLemma, lemma)

	node := b.nodes[textID]
	if node == nil {
		node = b.insert(b.textIdx.Get(textID))
		b.nodes[textID] = node
	}

	if !node.hasLast || node.lastPair != pair {
		node.hasLast = true
		node.lastPair = pair
		node.postings = append(node.postings, pair)
	}

	return nil
}

// csr holds compiled compressed-sparse-row arrays of the trie.
type csr struct {
	stateOff    []uint32 // state -> [start,end) of its transitions
	transLabel  []byte   // transition label, grouped by source state, sorted
	transTarget []uint32 // transition target state
	finals      []uint64 // bit i set = state i is final
	postOff     []uint32 // state -> [start,end) of postings
	posts       []uint32 // sorted unique pair ids
}

// compile assigns BFS state numbers (children visited in ascending label
// order, so layout depends only on the word set) and emits CSR arrays.
func compile(root *tnode) csr {
	states := []*tnode{root}
	root.state = 0

	for i := 0; i < len(states); i++ {
		cur := states[i]

		labels := make([]byte, 0, len(cur.kids))
		for l := range cur.kids {
			labels = append(labels, l)
		}

		sort.Slice(labels, func(a, z int) bool { return labels[a] < labels[z] })

		for _, l := range labels {
			kid := cur.kids[l]
			kid.state = uint32(len(states))
			states = append(states, kid)
		}
	}

	out := csr{
		stateOff:    make([]uint32, 0, len(states)+1),
		transLabel:  make([]byte, 0),
		transTarget: make([]uint32, 0),
		finals:      make([]uint64, (len(states)+63)/64),
		postOff:     make([]uint32, 0, len(states)+1),
	}

	for _, st := range states {
		out.stateOff = append(out.stateOff, uint32(len(out.transLabel)))

		labels := make([]byte, 0, len(st.kids))
		for l := range st.kids {
			labels = append(labels, l)
		}

		sort.Slice(labels, func(a, z int) bool { return labels[a] < labels[z] })

		for _, l := range labels {
			out.transLabel = append(out.transLabel, l)
			out.transTarget = append(out.transTarget, st.kids[l].state)
		}

		if st.final {
			out.finals[st.state>>6] |= 1 << (st.state & 63)

			out.postOff = append(out.postOff, uint32(len(out.posts)))
			out.posts = appendSortedUnique(out.posts, st.postings)
		} else {
			out.postOff = append(out.postOff, uint32(len(out.posts)))
		}
	}

	out.stateOff = append(out.stateOff, uint32(len(out.transLabel)))
	out.postOff = append(out.postOff, uint32(len(out.posts)))

	return out
}

func appendSortedUnique(dst, src []uint32) []uint32 {
	if len(src) == 0 {
		return dst
	}

	tmp := make([]uint32, len(src))
	copy(tmp, src)
	sort.Slice(tmp, func(a, b int) bool { return tmp[a] < tmp[b] })

	for i, v := range tmp {
		if i > 0 && v == tmp[i-1] {
			continue
		}

		dst = append(dst, v)
	}

	return dst
}
