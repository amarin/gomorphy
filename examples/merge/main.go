// Command merge shows combining two compiled dictionaries into one with
// Merge — e.g. a base dictionary plus a thematic overlay. MergeAdd keeps the
// base's readings for words present in both and adds overlay-only words;
// MergeReplace fully replaces the base's readings with the overlay's for
// shared words. Inputs are never mutated.
//
// Run: go run ./examples/merge
package main

import (
	"fmt"
	"log"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func build(dict map[string]string) *morphology.Dictionary {
	b := morphology.NewBuilder(morphology.BuilderOptions{Language: "ru"})
	for word, lemma := range dict {
		if err := b.AddForm(word, lemma, "NOUN,anim,masc,sing,nomn"); err != nil {
			log.Fatal(err)
		}
	}
	d, err := b.Build()
	if err != nil {
		log.Fatal(err)
	}
	return d
}

func main() {
	base := build(map[string]string{"кот": "кот"})
	defer func() { _ = base.Close() }()
	overlay := build(map[string]string{"котёнок": "котёнок"})
	defer func() { _ = overlay.Close() }()

	merged, err := morphology.Merge(base, []*morphology.Dictionary{overlay}, morphology.MergeAdd)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = merged.Close() }()

	for _, word := range []string{"кот", "котёнок"} {
		for _, r := range merged.Parse(word) {
			fmt.Println(r.Word, "->", r.Normal, "("+r.Tag+")")
		}
	}
	// Output:
	// кот -> кот (NOUN,anim,masc,sing,nomn)
	// котёнок -> котёнок (NOUN,anim,masc,sing,nomn)
}
