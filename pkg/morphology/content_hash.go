package morphology

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/zeebo/xxh3"
)

// ContentHash returns a stable hex digest (xxh3-128, 32 lower-case hex
// characters) of the dictionary's content: every section SaveTo writes
// except "info" (BuiltAt, LibraryVersion, Source…). Re-saving an unchanged
// dictionary, or opening it with Open/OpenBytes, keeps the digest; two
// dictionaries with equal ContentHash parse every word identically. The
// digest describes the encoding, not only the semantics: the same words
// with a different alphabet or CharPolicy hash differently.
//
// The first call encodes every section (for a large dictionary that is a
// copy of its words DAWG); the result is cached. Returns "" in two cases:
// a nil receiver, and an internal encoding failure while assembling the
// sections (today only Alphabet's EncodeAlphabet can fail this way; a
// CharPolicy that would make encoding fail is instead rejected at build
// time — see CharPolicy). Like every other method it must not be called
// after Close: on an Open'ed dictionary that reads unmapped memory.
func (x *Dictionary) ContentHash() string {
	if x == nil || x.d == nil {
		return ""
	}
	x.hashOnce.Do(func() { x.hash = x.computeContentHash() })
	return x.hash
}

func (x *Dictionary) computeContentHash() string {
	sections, err := x.sections(nil)
	if err != nil {
		return "" // only an Alphabet type EncodeAlphabet cannot write
	}
	h := xxh3.New128()
	var size [8]byte
	for _, s := range sections {
		_, _ = h.WriteString(s.Name)
		_, _ = h.Write([]byte{0})
		binary.LittleEndian.PutUint64(size[:], uint64(len(s.Data)))
		_, _ = h.Write(size[:])
		_, _ = h.Write(s.Data)
	}
	sum := h.Sum128().Bytes()
	return hex.EncodeToString(sum[:])
}
