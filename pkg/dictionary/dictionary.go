package dictionary

import (
	"fmt"
	"sync/atomic"

	"github.com/amarin/gomorphy/internal/build"
	"github.com/amarin/gomorphy/internal/mmapx"
)

// Dictionary is an immutable runtime snapshot (FT2, FT5, FT6 inputs).
// All methods are safe for concurrent reads. Instances are independent:
// each Open owns its own mmap region and no package-level state exists.
type Dictionary struct {
	snap   *build.Snapshot
	region *mmapx.Region
	closed atomic.Bool
}

// Open loads a compiled dictionary from path (FT7).
// The returned Dictionary must be closed with Close; all slices referencing
// the mapping become invalid afterwards.
func Open(path string) (*Dictionary, error) {
	snap, region, err := build.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("dictionary: %w", err)
	}

	return &Dictionary{snap: snap, region: region}, nil
}

// NewEmpty returns an in-memory dictionary without any words (FT8).
// It can be saved and re-opened like any other.
func NewEmpty() *Dictionary {
	return &Dictionary{snap: build.NewBuilder().Build()}
}

// Close releases the underlying memory mapping. Calling it while other
// goroutines still read the dictionary is a use-after-free bug: coordinate
// externally (the snapshot itself never mutates).
func (d *Dictionary) Close() error {
	if d.closed.Swap(true) {
		return nil
	}

	if d.region != nil {
		return d.region.Close()
	}

	return nil
}

// checkClosed reports whether the dictionary has been closed.
func (d *Dictionary) checkClosed() error {
	if d.closed.Load() {
		return ErrClosed
	}

	return nil
}

// SaveTo writes the compiled snapshot to path (FT3, FT8).
func (d *Dictionary) SaveTo(path string) error {
	if err := d.checkClosed(); err != nil {
		return err
	}

	return d.snap.SaveTo(path)
}

// Builder reconstructs a mutable Builder seeded from this dictionary's
// grammemes, ancodes, lemmas and wordforms (FT8). The dictionary itself is
// not modified; Compile on the result yields an independent new snapshot.
func (d *Dictionary) Builder() (*Builder, error) {
	if err := d.checkClosed(); err != nil {
		return nil, err
	}

	s := d.snap

	b := &Builder{inner: build.NewBuilderWithCapacity(s.TextCount(), s.LemmaCount(), s.PairCount())}

	for _, name := range s.GrammemeNames {
		if _, err := b.inner.AddGrammeme(name); err != nil {
			return nil, fmt.Errorf("dictionary: rebuild grammemes: %w", err)
		}
	}

	lemmaSeen := make([]bool, s.LemmaCount())

	for pairID := uint32(0); pairID < uint32(s.PairCount()); pairID++ {
		grammemes := gramNames(s, pairID)
		text := string(s.Text(s.PairTexts[pairID]))

		for _, lemma := range s.PairLemmasOf(pairID) {
			if int(lemma) >= len(lemmaSeen) {
				return nil, fmt.Errorf("dictionary: lemma %d out of range", lemma)
			}

			if !lemmaSeen[lemma] {
				lemmaSeen[lemma] = true

				if _, err := b.inner.AddLemma(string(s.LemmaText(lemma)), grammemes...); err != nil {
					return nil, fmt.Errorf("dictionary: rebuild lemmas: %w", err)
				}
			}

			// Duplicate attachments collapse into the same pair; only the
			// surface forms differing from the citation text matter here.
			if text != string(s.LemmaText(lemma)) {
				if err := b.inner.AddForm(int(lemma), text, grammemes...); err != nil {
					return nil, fmt.Errorf("dictionary: rebuild forms: %w", err)
				}
			}
		}
	}

	return b, nil
}

// gramNames decomposes the ancode of a pair into grammeme name strings.
func gramNames(s *build.Snapshot, pairID uint32) []string {
	gramIDs := s.Ancode(s.PairAncodes[pairID])

	names := make([]string, 0, len(gramIDs))
	for _, id := range gramIDs {
		names = append(names, s.GrammemeNames[id])
	}

	return names
}
