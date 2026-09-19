// Command unimorph shows compiling a dictionary from a UniMorph TSV
// (lemma<TAB>wordform<TAB>bundle) — here, a tiny inline fragment, so this
// example runs with no external data. A real UniMorph dictionary (fetched
// via `gomorphy download unimorph`) has the same shape at a much larger
// scale — CompileFromUniMorph doesn't care which.
//
// Run: go run ./examples/unimorph
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/amarin/gomorphy/pkg/morphology"
)

const tsv = "кот\tкот\tN;NOM;SG\n" +
	"кот\tкота\tN;ACC;SG\n" +
	"кот\tкота\tN;GEN;SG\n"

func main() {
	// Language is mandatory; "ru" is the only value UniMorph import
	// currently accepts (see docs/en/implementation/stage-16-import-unimorph.md).
	d, err := morphology.CompileFromUniMorph(strings.NewReader(tsv), morphology.UniMorphOptions{Language: "ru"})
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range d.Parse("кота") {
		fmt.Printf("%s -> %s (%s)\n", r.Word, r.Normal, r.Tag)
	}
	// Output:
	// кота -> кот (N;ACC;SG)
	// кота -> кот (N;GEN;SG)
}
