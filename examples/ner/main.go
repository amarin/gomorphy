// Command ner shows dictionary-based named-entity lookup: a small name
// dictionary built with Builder flags known names in a text, and
// IsKnown/Reading.Predicted keep suffix prediction from turning every
// unknown word into a "name". Words are lower-cased on Build (1.2.0+), so
// a capitalised name in the text finds its dictionary entry.
//
// Run: go run ./examples/ner
package main

import (
	"fmt"
	"log"
	"strings"
	"unicode"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func main() {
	b := morphology.NewBuilder(morphology.BuilderOptions{Language: "ru", Source: "names"})
	for _, f := range [][3]string{
		{"Москва", "Москва", "GEO,nomn"},
		{"Москвы", "Москва", "GEO,gent"},
		{"Москве", "Москва", "GEO,loct"},
		{"Пушкин", "Пушкин", "PERSON,nomn"},
		{"Пушкина", "Пушкин", "PERSON,gent"},
	} {
		if err := b.AddForm(f[0], f[1], f[2]); err != nil {
			log.Fatal(err)
		}
	}
	names, err := b.Build()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = names.Close() }()

	text := "Памятник Пушкина стоит в Москве, памятника Тушкина нет"
	notLetter := func(r rune) bool { return !unicode.IsLetter(r) }
	for _, token := range strings.FieldsFunc(text, notLetter) {
		r := names.Parse(token)
		switch {
		case names.IsKnown(token):
			fmt.Printf("%-9s NAME %s (%s)\n", token, r[0].Normal, r[0].Tag)
		case len(r) > 0 && r[0].Predicted:
			// Parse still guesses by the ending — a guess, not a name.
			fmt.Printf("%-9s -    guess %s, Predicted=true\n", token, r[0].Normal)
		default:
			fmt.Printf("%-9s -\n", token)
		}
	}
	// Output:
	// Памятник  -
	// Пушкина   NAME пушкин (PERSON,gent)
	// стоит     -
	// в         -
	// Москве    NAME москва (GEO,loct)
	// памятника -    guess памятника, Predicted=true
	// Тушкина   -    guess тушкин, Predicted=true
	// нет       -
}
