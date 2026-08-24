package build

import (
	"github.com/zeebo/xxh3"
)

// Snapshot is the immutable runtime dictionary image. All fields are values
// (flat slices); reading is safe from any number of goroutines (FT9).
type Snapshot struct {
	TextData []byte   // interned texts arena payload
	TextOffs []uint32 // text id -> [start,end) into TextData, len = texts+1

	GrammemeNames []string // grammeme id -> name

	AncodeOff   []uint32 // ancode id -> [start,end) into AncodeGrams
	AncodeGrams []uint32 // flat grammeme-id lists grouped by ancode

	LemmaTexts   []uint32 // dense lemma id -> text id
	LemmaAncodes []uint32 // dense lemma id -> base-form ancode id

	PairTexts    []uint32 // pair id -> text id
	PairAncodes  []uint32 // pair id -> ancode id
	PairLemmaOff []uint32 // pair id -> [start,end) into PairLemmas, len = pairs+1
	PairLemmas   []uint32 // flat lemma lists grouped by pair

	StateOff    []uint32 // trie CSR: state -> transitions window
	TransLabel  []byte   // trie CSR: transition labels, sorted per state
	TransTarget []uint32 // trie CSR: transition target states
	Finals      []uint64 // trie CSR: final-state bitmask
	PostingsOff []uint32 // trie CSR: state -> postings window
	Postings    []uint32 // trie CSR: sorted unique pair ids per final state

	Exact     []HashEntry // open-addressing exact-match table
	ExactMask uint64
}

// TextCount returns the number of distinct interned texts.
func (s *Snapshot) TextCount() int { return len(s.TextOffs) - 1 }

// LemmaCount returns the number of lemmas.
func (s *Snapshot) LemmaCount() int { return len(s.LemmaTexts) }

// PairCount returns the number of unique (text, ancode) pairs.
func (s *Snapshot) PairCount() int { return len(s.PairTexts) }

// StateCount returns the number of trie states.
func (s *Snapshot) StateCount() int { return len(s.StateOff) - 1 }

// Text returns the bytes of an interned text.
func (s *Snapshot) Text(textID uint32) []byte {
	return s.TextData[s.TextOffs[textID]:s.TextOffs[textID+1]]
}

// LemmaText returns the base-form text of a lemma.
func (s *Snapshot) LemmaText(lemma uint32) []byte {
	return s.Text(s.LemmaTexts[lemma])
}

// LemmaBaseGrams returns the constant grammeme ids of a lemma (POS and
// other tags shared by all its wordforms).
func (s *Snapshot) LemmaBaseGrams(lemma uint32) []uint32 {
	return s.Ancode(s.LemmaAncodes[lemma])
}

// AncodeGrams returns the grammeme ids of an ancode.
func (s *Snapshot) Ancode(ancodeID uint32) []uint32 {
	return s.AncodeGrams[s.AncodeOff[ancodeID]:s.AncodeOff[ancodeID+1]]
}

// PairLemmas returns the lemmas sharing a (text, ancode) pair.
func (s *Snapshot) PairLemmasOf(pairID uint32) []uint32 {
	return s.PairLemmas[s.PairLemmaOff[pairID]:s.PairLemmaOff[pairID+1]]
}

// Postings returns the sorted unique pair ids attached to a final state.
// The window is empty for non-final states.
func (s *Snapshot) PostingsOf(state uint32) []uint32 {
	return s.Postings[s.PostingsOff[state]:s.PostingsOff[state+1]]
}

// IsFinal reports whether a trie state terminates a complete word.
func (s *Snapshot) IsFinal(state uint32) bool {
	return s.Finals[state>>6]&(1<<(state&63)) != 0
}

// Step follows one transition byte from a state.
func (s *Snapshot) Step(state uint32, c byte) (uint32, bool) {
	lo := s.StateOff[state]
	hi := s.StateOff[state+1]

	for lo < hi {
		mid := (lo + hi) / 2

		switch {
		case s.TransLabel[mid] == c:
			return s.TransTarget[mid], true
		case s.TransLabel[mid] < c:
			lo = mid + 1
		default:
			hi = mid
		}
	}

	return 0, false
}

// Edges returns parallel label/target views of all outgoing transitions of
// a state, sorted by label. The slices alias internal arrays and must be
// treated as read-only.
func (s *Snapshot) Edges(state uint32) ([]byte, []uint32) {
	lo := s.StateOff[state]
	hi := s.StateOff[state+1]

	return s.TransLabel[lo:hi:hi], s.TransTarget[lo:hi:hi]
}

// Walk traverses the trie along text and reports whether every transition
// exists. It does not require the terminal state to be final.
func (s *Snapshot) Walk(text []byte) (uint32, bool) {
	state := uint32(0)

	for _, c := range text {
		next, ok := s.Step(state, c)
		if !ok {
			return 0, false
		}

		state = next
	}

	return state, true
}

// LookupWord resolves a word to its final trie state via the exact hash,
// verifying candidates by walking the trie (collision-safe).
func (s *Snapshot) LookupWord(text []byte) (uint32, bool) {
	if len(text) == 0 || len(s.Exact) == 0 {
		return 0, false
	}

	h := xxh3.Hash(text)

	pos := h & s.ExactMask

	for {
		slot := &s.Exact[pos]
		if slot.State == hashEmptyState {
			return 0, false
		}

		if slot.Hash == h {
			state, ok := s.Walk(text)
			if ok && state == slot.State && s.IsFinal(state) {
				return state, true
			}

			return 0, false
		}

		pos = (pos + 1) & s.ExactMask
	}
}
