package build

import (
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"

	"github.com/amarin/gomorphy/internal/format"
	"github.com/amarin/gomorphy/internal/mmapx"
)

// OpenFile memory-maps a dictionary file previously written by SaveTo and
// returns its immutable snapshot plus the owning region. Fixed-width sections
// are sliced in place (no per-element parsing, FT3); offset/posting streams
// are delta-decoded once. The returned Region must stay open for as long as
// the snapshot is used.
func OpenFile(path string) (*Snapshot, *mmapx.Region, error) {
	region, err := mmapx.Open(path)
	if err != nil {
		return nil, nil, err
	}

	snap, err := loadFromRegion(region.Bytes())
	if err != nil {
		_ = region.Close()

		return nil, nil, fmt.Errorf("build: open %s: %w", path, err)
	}

	return snap, region, nil
}

func loadFromRegion(reg []byte) (*Snapshot, error) {
	reader, err := format.OpenRead(bytesAt(reg), int64(len(reg)))
	if err != nil {
		return nil, err
	}

	entryOf := map[string]format.Entry{}
	for _, e := range reader.Entries() {
		entryOf[e.Name] = e
	}

	raw := func(name string) ([]byte, error) {
		e, ok := entryOf[name]
		if !ok {
			return nil, format.ErrUnknownSection
		}

		return reg[e.Offset : e.Offset+e.Size], nil
	}

	rawU32 := func(name string) ([]uint32, error) {
		b, err := raw(name)
		if err != nil {
			return nil, err
		}

		if len(b) == 0 {
			return nil, nil
		}

		if len(b)%4 != 0 {
			return nil, fmt.Errorf("section %s: size %d not multiple of 4", name, len(b))
		}

		return unsafe.Slice((*uint32)(unsafe.Pointer(&b[0])), len(b)/4), nil
	}

	// rawU32Count reads exactly want elements, ignoring alignment padding.
	rawU32Count := func(name string, want int) ([]uint32, error) {
		full, err := rawU32(name)
		if err != nil {
			return nil, err
		}

		if len(full) < want {
			return nil, fmt.Errorf("section %s: need %d words, have %d", name, want, len(full))
		}

		return full[:want], nil
	}

	rawU64 := func(name string) ([]uint64, error) {
		b, err := raw(name)
		if err != nil {
			return nil, err
		}

		if len(b) == 0 {
			return nil, nil
		}

		if len(b)%8 != 0 {
			return nil, fmt.Errorf("section %s: size %d not multiple of 8", name, len(b))
		}

		return unsafe.Slice((*uint64)(unsafe.Pointer(&b[0])), len(b)/8), nil
	}

	deltaU32 := func(name string) ([]uint32, error) {
		b, err := raw(name)
		if err != nil {
			return nil, err
		}

		return format.ParseDelta32(b)
	}

	metaBytes, err := raw(secMeta)
	if err != nil {
		return nil, err
	}

	if len(metaBytes) < metaWords*4 {
		return nil, fmt.Errorf("meta section truncated: %d bytes", len(metaBytes))
	}

	texts := binary.LittleEndian.Uint32(metaBytes[0:])
	grammemes := binary.LittleEndian.Uint32(metaBytes[4:])
	ancodes := binary.LittleEndian.Uint32(metaBytes[8:])
	lemmas := binary.LittleEndian.Uint32(metaBytes[12:])
	pairs := binary.LittleEndian.Uint32(metaBytes[16:])
	states := binary.LittleEndian.Uint32(metaBytes[20:])
	exactN := binary.LittleEndian.Uint32(metaBytes[24:])
	exactMask := uint64(binary.LittleEndian.Uint32(metaBytes[28:]))

	snap := &Snapshot{
		GrammemeNames: make([]string, grammemes),
		LemmaTexts:    make([]uint32, lemmas),
		PairTexts:     make([]uint32, pairs),
		PairAncodes:   make([]uint32, pairs),
		PairLemmaOff:  make([]uint32, pairs+1),
		StateOff:      make([]uint32, states+1),
		Finals:        make([]uint64, (states+63)/64),
		PostingsOff:   make([]uint32, states+1),
		Exact:         make([]HashEntry, exactN),
	}

	if snap.TextData, err = raw(secTextsData); err != nil {
		return nil, err
	}

	if snap.TextOffs, err = deltaU32(secTextsOffs); err != nil {
		return nil, err
	}

	if uint32(len(snap.TextOffs)-1) != texts {
		return nil, fmt.Errorf("texts: meta says %d, got %d", texts, len(snap.TextOffs)-1)
	}

	gblob, err := raw(secGrammemes)
	if err != nil {
		return nil, err
	}

	for i := 0; i < int(grammemes); i++ {
		n, used := binary.Uvarint(gblob)
		if used <= 0 || n > uint64(len(gblob)-used) {
			return nil, fmt.Errorf("grammemes: malformed entry %d", i)
		}

		snap.GrammemeNames[i] = string(gblob[used : used+int(n)])
		gblob = gblob[used+int(n):]
	}

	if snap.AncodeOff, err = deltaU32(secAncodeOffs); err != nil {
		return nil, err
	}

	if snap.AncodeGrams, err = rawU32Count(secAncodeGrams, int(snap.AncodeOff[len(snap.AncodeOff)-1])); err != nil {
		return nil, err
	}

	if uint32(len(snap.AncodeOff)-1) != ancodes {
		return nil, fmt.Errorf("ancodes: meta says %d, got %d", ancodes, len(snap.AncodeOff)-1)
	}

	if snap.LemmaTexts, err = rawU32Count(secLemmas, int(lemmas)); err != nil {
		return nil, err
	}

	if snap.PairTexts, err = rawU32Count(secPairTexts, int(pairs)); err != nil {
		return nil, err
	}

	if snap.PairAncodes, err = rawU32Count(secPairAncodes, int(pairs)); err != nil {
		return nil, err
	}

	if snap.PairLemmaOff, err = deltaU32(secPairLemmaOffs); err != nil {
		return nil, err
	}

	if snap.PairLemmas, err = rawU32Count(secPairLemmas, int(snap.PairLemmaOff[len(snap.PairLemmaOff)-1])); err != nil {
		return nil, err
	}

	if snap.StateOff, err = deltaU32(secTrieStateOff); err != nil {
		return nil, err
	}

	if snap.TransLabel, err = raw(secTrieLabel); err != nil {
		return nil, err
	}

	if snap.TransTarget, err = deltaU32(secTrieTarget); err != nil {
		return nil, err
	}

	if snap.Finals, err = rawU64(secTrieFinals); err != nil {
		return nil, err
	}

	if snap.PostingsOff, err = deltaU32(secPostingsOffs); err != nil {
		return nil, err
	}

	if snap.Postings, err = deltaU32(secPostings); err != nil {
		return nil, err
	}

	eblob, err := raw(secExact)
	if err != nil {
		return nil, err
	}

	for i := range snap.Exact {
		if len(eblob) < 16 {
			return nil, fmt.Errorf("exact: truncated at entry %d", i)
		}

		snap.Exact[i] = HashEntry{
			Hash:  binary.LittleEndian.Uint64(eblob),
			State: binary.LittleEndian.Uint32(eblob[8:]),
		}
		eblob = eblob[16:]
	}

	snap.ExactMask = exactMask

	if err := validate(snap); err != nil {
		return nil, err
	}

	return snap, nil
}

func validate(s *Snapshot) error {
	switch {
	case s.TextCount() != 0 && len(s.TextOffs) == 0:
		return fmt.Errorf("text offsets missing")
	case s.StateCount() >= 0 && len(s.TransTarget) != len(s.TransLabel):
		return fmt.Errorf("trie: %d labels vs %d targets", len(s.TransLabel), len(s.TransTarget))
	case s.PairCount() > 0 && len(s.PairLemmaOff) != s.PairCount()+1:
		return fmt.Errorf("pair lemma offsets: got %d windows", len(s.PairLemmaOff)-1)
	default:
		return nil
	}
}

// bytesAt adapts a byte slice to io.ReaderAt without copying.
type bytesAt []byte

func (b bytesAt) ReadAt(p []byte, off int64) (int, error) {
	switch {
	case off < 0:
		return 0, fmt.Errorf("bytesAt: negative offset %d", off)
	case off >= int64(len(b)):
		return 0, io.EOF
	}

	n := copy(p, b[off:])
	if n < len(p) {
		return n, io.EOF
	}

	return n, nil
}
