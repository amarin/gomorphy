// Command typos shows typo-tolerant lookup with Fuzzy and FuzzyTop: the
// query is lower-cased, and for a Russian dictionary е in the query matches
// a stored ё at distance 0 (both 1.2.0+), so «Елка» finds «ёлка» exactly.
//
// Run: go run ./examples/typos
package main

import (
	"fmt"
	"log"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func main() {
	b := morphology.NewBuilder(morphology.BuilderOptions{Language: "ru"})
	for _, w := range []string{"ёлка", "кот", "код", "кит", "каток"} {
		if err := b.AddLemma(w, "NOUN"); err != nil {
			log.Fatal(err)
		}
	}
	d, err := b.Build()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = d.Close() }()

	// Suggestions within one edit.
	for _, m := range d.Fuzzy("кот", 1) {
		fmt.Println("кот ~", m.Word, m.Distance)
	}
	// е→ё and the capital letter cost nothing.
	for _, m := range d.Fuzzy("Елка", 0) {
		fmt.Println("Елка ~", m.Word, m.Distance)
	}
	// The two nearest words, whatever the distance.
	for _, m := range d.FuzzyTop("катк", 2) {
		fmt.Println("катк ~", m.Word, m.Distance)
	}
	// Output:
	// кот ~ кот 0
	// кот ~ кит 1
	// кот ~ код 1
	// Елка ~ ёлка 0
	// катк ~ каток 1
	// катк ~ кит 2
}
