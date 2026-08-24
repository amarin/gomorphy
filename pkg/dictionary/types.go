// Package dictionary provides the public API of gomorphy: opening compiled
// dictionaries, exact lookup of wordforms and lemmas, programmatic building.
//
// A Dictionary is an immutable snapshot backed by a read-only memory mapping.
// All read methods are safe for concurrent use by multiple goroutines;
// independent instances share no global state (FT9).
//
// # Opening a compiled dictionary
//
//	d, err := dictionary.Open("opencorpora.dat")
//	if err != nil {
//		return err
//	}
//	defer d.Close()
//
//	forms, err := d.Lookup("кота")
//	// forms[0].Grammemes == ["NOUN","anim","masc","gent"], ...
//
// # Building programmatically (FT8)
//
//	b := dictionary.NewBuilder()
//	id, _ := b.AddLemma("кот", "NOUN", "anim", "masc")
//	_ = b.AddForm(id, "кота", "NOUN", "anim", "masc", "gent")
//	d, _ := b.Compile()
//	_ = d.SaveTo("my.dict")
package dictionary

import (
	"errors"
)

var (
	// ErrClosed is returned by Dictionary methods after Close.
	ErrClosed = errors.New("dictionary: closed")

	// ErrNotImplemented marks APIs delivered as stubs in early stages.
	ErrNotImplemented = errors.New("dictionary: not implemented")

	// ErrNotFound is returned when a word has no dictionary entry.
	ErrNotFound = errors.New("dictionary: not found")
)

// Wordform is one dictionary reading of a word: surface text with its
// grammatical description and owning lemma. Value type, safe to copy.
type Wordform struct {
	Text      string   // surface form, e.g. "кота"
	Ancode    string   // comma-joined grammeme names, e.g. "NOUN,anim,masc,gent"
	Grammemes []string // decomposed grammemes of Ancode, order significant
	LemmaID   uint32   // dense lemma identifier
}

// LemmaRef identifies a lemma: dense id plus its citation text.
type LemmaRef struct {
	ID   uint32
	Text string
}

// FuzzyMatch is a fuzzy-search hit (FT6, implemented in stage 9).
type FuzzyMatch struct {
	Text     string
	Distance int
}
