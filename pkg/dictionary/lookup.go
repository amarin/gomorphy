package dictionary

import (
	"sort"
	"strings"

	"github.com/amarin/gomorphy/internal/build"
)

// Lookup returns all dictionary readings of word (FT2): one Wordform per
// distinct (surface text, ancode) pair. The reserve trie walk is used when
// the exact-hash probe misses.
func (d *Dictionary) Lookup(word string) ([]Wordform, error) {
	if err := d.checkClosed(); err != nil {
		return nil, err
	}

	state, ok := d.snap.LookupWord([]byte(word))
	if !ok {
		return nil, ErrNotFound
	}

	postings := d.snap.PostingsOf(state)

	forms := make([]Wordform, 0, len(postings))
	for _, pairID := range postings {
		gramIDs := d.snap.Ancode(d.snap.PairAncodes[pairID])

		names := make([]string, len(gramIDs))

		for i, id := range gramIDs {
			names[i] = d.snap.GrammemeNames[id]
		}

		forms = append(forms, Wordform{
			Text:      string(d.snap.Text(d.snap.PairTexts[pairID])),
			Ancode:    strings.Join(names, ","),
			Grammemes: names,
			LemmaID:   firstLemma(d.snap, pairID),
		})
	}

	return forms, nil
}

func firstLemma(s *build.Snapshot, pairID uint32) uint32 {
	if ls := s.PairLemmasOf(pairID); len(ls) > 0 {
		return ls[0]
	}

	return 0
}

// Lemmas returns the lemmas owning any reading of word (FT5), ordered by id.
func (d *Dictionary) Lemmas(word string) ([]LemmaRef, error) {
	if err := d.checkClosed(); err != nil {
		return nil, err
	}

	state, ok := d.snap.LookupWord([]byte(word))
	if !ok {
		return nil, ErrNotFound
	}

	seen := make(map[uint32]struct{})

	for _, pairID := range d.snap.PostingsOf(state) {
		for _, lemma := range d.snap.PairLemmasOf(pairID) {
			seen[lemma] = struct{}{}
		}
	}

	refs := make([]LemmaRef, 0, len(seen))
	for id := range seen {
		refs = append(refs, LemmaRef{ID: id, Text: string(d.snap.LemmaText(id))})
	}

	sort.Slice(refs, func(i, j int) bool { return refs[i].ID < refs[j].ID })

	return refs, nil
}

// Fuzzy finds words within edit distance maxDist of word (FT6).
// Stub until stage 9.
func (d *Dictionary) Fuzzy(word string, maxDist int) ([]FuzzyMatch, error) {
	if err := d.checkClosed(); err != nil {
		return nil, err
	}

	return nil, ErrNotImplemented
}
