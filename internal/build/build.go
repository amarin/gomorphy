package build

import "slices"

// Build compiles all accumulated data into an immutable Snapshot.
// The Builder remains usable; subsequent additions require a new Build call.
func (b *Builder) Build() *Snapshot {
	c := compile(b.root)

	snap := &Snapshot{
		TextData:      slices.Clone(b.texts.Data()),
		TextOffs:      slices.Clone(b.texts.Offsets()),
		GrammemeNames: slices.Clone(b.grammemes),
		StateOff:      c.stateOff,
		TransLabel:    c.transLabel,
		TransTarget:   c.transTarget,
		Finals:        c.finals,
		PostingsOff:   c.postOff,
		Postings:      c.posts,
	}

	snap.AncodeOff, snap.AncodeGrams = flattenAncodes(b.ancodeKeys)
	snap.PairLemmaOff, snap.PairLemmas = flattenPairLemmas(b.attPair, b.attLemma, len(b.pairTexts))

	snap.PairTexts = slices.Clone(b.pairTexts)
	snap.PairAncodes = slices.Clone(b.pairAncodes)
	snap.LemmaTexts = slices.Clone(b.lemmaTexts)

	snap.Exact, snap.ExactMask = buildExactHash(
		len(b.nodes),
		func(textID uint32) (uint32, bool) {
			n := b.nodes[textID]
			if n == nil || !n.final {
				return 0, false
			}

			return n.state, true
		},
		func(textID uint32) []byte { return b.textIdx.Get(textID) },
	)

	return snap
}

func flattenAncodes(lists [][]uint32) (off []uint32, flat []uint32) {
	off = make([]uint32, len(lists)+1)

	for i, list := range lists {
		off[i+1] = off[i] + uint32(len(list))
		flat = append(flat, list...)
	}

	return off, flat
}

// flattenPairLemmas groups the attach log by pair in O(n) using counting,
// then deduplicates lemma ids inside each (tiny) window preserving order.
func flattenPairLemmas(attPair, attLemma []uint32, pairs int) (off []uint32, vals []uint32) {
	off = make([]uint32, pairs+1)

	for _, p := range attPair {
		off[p+1]++
	}

	for i := 1; i <= pairs; i++ {
		off[i] += off[i-1]
	}

	vals = make([]uint32, len(attPair))
	cursor := make([]uint32, pairs)
	copy(cursor, off[:pairs])

	for i, p := range attPair {
		vals[cursor[p]] = attLemma[i]
		cursor[p]++
	}

	newOff := make([]uint32, pairs+1)
	w := uint32(0)

	for p := 0; p < pairs; p++ {
		start := w

		for i := off[p]; i < off[p+1]; i++ {
			v := vals[i]

			dup := false
			for j := start; j < w; j++ {
				if vals[j] == v {
					dup = true

					break
				}
			}

			if !dup {
				vals[w] = v
				w++
			}
		}

		newOff[p] = start
	}

	newOff[pairs] = w

	return newOff, vals[:w]
}
