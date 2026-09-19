// Command pymorphy shows reading a pymorphy2 dictionary directly from its
// source directory (words.dawg, paradigms.array, ...) with OpenPyMorphy —
// no separate compile-to-.dat step needed.
//
// Unlike the opencorpora/unimorph examples, this one needs real data: a
// pymorphy2 dictionary's binary DAWG can't be hand-written inline. Fetch one
// first:
//
//	gomorphy download pymorphy
//	go run ./examples/pymorphy -dir .data/pymorphy/data
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func main() {
	dir := flag.String("dir", "", "path to an unpacked pymorphy2 dictionary directory (see `gomorphy download pymorphy`)")
	word := flag.String("word", "кота", "word to look up")
	flag.Parse()

	if *dir == "" {
		log.Fatal("-dir is required; run `gomorphy download pymorphy` first, then pass its data directory")
	}

	// OpenPyMorphyDense additionally recompiles words.dawg under a dense
	// 1-byte alphabet — same readings, faster and more compact in memory
	// (see docs/en/implementation/pymorphy2-dense-alphabet.md). Use
	// OpenPyMorphy instead for the raw, non-dense dictionary.
	d, err := morphology.OpenPyMorphyDense(*dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range d.Parse(*word) {
		fmt.Printf("%s -> %s (%s)\n", r.Word, r.Normal, r.Tag)
	}
}
