package build

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/amarin/gomorphy/internal/format"
)

// Section names of the compiled dictionary container.
const (
	secMeta          = "meta"
	secTextsData     = "texts.data"
	secTextsOffs     = "texts.offs"
	secGrammemes     = "grammemes"
	secAncodeOffs    = "ancodes.offs"
	secAncodeGrams   = "ancodes.grams"
	secLemmas        = "lemmas"
	secLemmasAncodes = "lemmas.ancodes"
	secPairTexts     = "pairs.text"
	secPairAncodes   = "pairs.ancode"
	secPairLemmaOffs = "pair.lemmas.offs"
	secPairLemmas    = "pair.lemmas"
	secTrieStateOff  = "trie.stateOff"
	secTrieLabel     = "trie.label"
	secTrieTarget    = "trie.target"
	secTrieFinals    = "trie.finals"
	secPostingsOffs  = "postings.offs"
	secPostings      = "postings"
	secExact         = "exact"
)

const metaWords = 8 // counts stored in meta section

func putU32s(dst []byte, vals []uint32) []byte {
	for _, v := range vals {
		dst = binary.LittleEndian.AppendUint32(dst, v)
	}

	return dst
}

func putU64s(dst []byte, vals []uint64) []byte {
	for _, v := range vals {
		dst = binary.LittleEndian.AppendUint64(dst, v)
	}

	return dst
}

// padTo8 appends zero bytes so the payload length becomes a multiple of 8,
// keeping following fixed-width sections aligned in the file.
func padTo8(dst []byte) []byte {
	if rem := len(dst) % 8; rem != 0 {
		dst = append(dst, make([]byte, 8-rem)...)
	}

	return dst
}

// SaveTo serializes the snapshot into a checksummed sectioned file (FT3/FT4).
func (s *Snapshot) SaveTo(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("build: create %s: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	w := format.NewWriter(f)

	if err := w.WriteHeader(format.Version); err != nil {
		return fmt.Errorf("build: write header: %w", err)
	}

	meta := make([]byte, 0, metaWords*4)
	meta = putU32s(meta, []uint32{
		uint32(s.TextCount()),
		uint32(len(s.GrammemeNames)),
		uint32(len(s.AncodeOff) - 1),
		uint32(s.LemmaCount()),
		uint32(s.PairCount()),
		uint32(s.StateCount()),
		uint32(len(s.Exact)),
		uint32(s.ExactMask),
	})

	grammemes := make([]byte, 0, 16*len(s.GrammemeNames))
	for _, name := range s.GrammemeNames {
		grammemes = binary.AppendUvarint(grammemes, uint64(len(name)))
		grammemes = append(grammemes, name...)
	}

	exact := make([]byte, 0, len(s.Exact)*16)
	for i := range s.Exact {
		exact = binary.LittleEndian.AppendUint64(exact, s.Exact[i].Hash)
		exact = binary.LittleEndian.AppendUint32(exact, s.Exact[i].State)
		exact = binary.LittleEndian.AppendUint32(exact, 0) // padding
	}

	sections := []struct {
		name string
		data []byte
	}{
		{secMeta, meta},
		{secTextsData, s.TextData},
		{secTextsOffs, padTo8(mustDelta32(s.TextOffs))},
		{secGrammemes, grammemes},
		{secAncodeOffs, padTo8(mustDelta32(s.AncodeOff))},
		{secAncodeGrams, padTo8(putU32s(nil, s.AncodeGrams))},
		{secLemmas, padTo8(putU32s(nil, s.LemmaTexts))},
		{secLemmasAncodes, padTo8(putU32s(nil, s.LemmaAncodes))},
		{secPairTexts, padTo8(putU32s(nil, s.PairTexts))},
		{secPairAncodes, padTo8(putU32s(nil, s.PairAncodes))},
		{secPairLemmaOffs, padTo8(mustDelta32(s.PairLemmaOff))},
		{secPairLemmas, padTo8(putU32s(nil, s.PairLemmas))},
		{secTrieStateOff, padTo8(mustDelta32(s.StateOff))},
		{secTrieLabel, s.TransLabel},
		{secTrieTarget, padTo8(mustDelta32(s.TransTarget))},
		{secTrieFinals, putU64s(nil, s.Finals)},
		{secPostingsOffs, padTo8(mustDelta32(s.PostingsOff))},
		{secPostings, padTo8(mustDelta32(s.Postings))},
		{secExact, exact},
	}

	for _, sec := range sections {
		if err := w.WriteSection(sec.name, sec.data); err != nil {
			return fmt.Errorf("build: write section %s: %w", sec.name, err)
		}
	}

	if err := w.Finalize(); err != nil {
		return fmt.Errorf("build: finalize %s: %w", path, err)
	}

	return f.Close()
}

func mustDelta32(vals []uint32) []byte {
	return format.AppendDelta32(nil, vals)
}
