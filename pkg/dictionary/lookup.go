package dictionary

import (
	"sort"
	"strings"

	"github.com/amarin/gomorphy/internal/build"
)

// Lookup returns all dictionary readings of word (FT2): one Wordform per
// (surface text, ancode, lemma) triple. A pair shared by several lemmas
// yields one reading per owning lemma — homonymous parses stay distinct
// even when their tag sets coincide. The reserve trie walk is used when
// the exact-hash probe misses.
//
// Wordform.Grammemes holds the full grammatical characterization: the
// lemma's constant tags (POS etc.) followed by the form's own tags, in
// that order.
//
// Pairs registered by AddLemma to make citation forms resolvable carry the
// bare base ancode; they anchor Lemmas() lookups but are not readings, so
// Lookup skips them (a citation whose text has no inflected forms yields
// ErrNotFound here while staying visible through Lemmas).
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
		pairGrams := d.snap.Ancode(d.snap.PairAncodes[pairID])

		for _, lemma := range d.snap.PairLemmasOf(pairID) {
			if citationPair(d.snap, pairID, lemma, pairGrams) {
				continue
			}

			names := fullGramNames(d.snap, lemma, pairGrams)

			forms = append(forms, Wordform{
				Text:      string(d.snap.Text(d.snap.PairTexts[pairID])),
				Ancode:    strings.Join(names, ","),
				Grammemes: names,
				LemmaID:   lemma,
			})
		}
	}

	if len(forms) == 0 {
		return nil, ErrNotFound
	}

	return forms, nil
}

// citationPair reports whether the pair merely anchors a lemma citation:
// its text and grammemes coincide with the lemma's own.
func citationPair(s *build.Snapshot, pairID, lemma uint32, pairGrams []uint32) bool {
	if s.PairTexts[pairID] != s.LemmaTexts[lemma] {
		return false
	}

	base := s.LemmaBaseGrams(lemma)
	if len(base) != len(pairGrams) {
		return false
	}

	for i := range base {
		if base[i] != pairGrams[i] {
			return false
		}
	}

	return true
}

// fullGramNames merges a lemma's base grammemes with the wordform's own
// grammemes, preserving order and dropping duplicates.
func fullGramNames(s *build.Snapshot, lemma uint32, formGrams []uint32) []string {
	base := s.LemmaBaseGrams(lemma)

	seen := make(map[uint32]struct{}, len(base)+len(formGrams))
	names := make([]string, 0, len(base)+len(formGrams))

	add := func(ids []uint32) {
		for _, id := range ids {
			if _, dup := seen[id]; dup {
				continue
			}

			seen[id] = struct{}{}
			names = append(names, s.GrammemeNames[id])
		}
	}

	add(base)
	add(formGrams)

	return names
}

// Lemmas returns the lemmas owning any reading of word (FT5), ordered by id.
// Each ref carries the citation text and its base grammemes.
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
		base := d.snap.LemmaBaseGrams(id)

		names := make([]string, len(base))
		for i, gid := range base {
			names[i] = d.snap.GrammemeNames[gid]
		}

		refs = append(refs, LemmaRef{ID: id, Text: string(d.snap.LemmaText(id)), Grammemes: names})
	}

	sort.Slice(refs, func(i, j int) bool { return refs[i].ID < refs[j].ID })

	return refs, nil
}
