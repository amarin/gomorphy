// Command open shows the on-disk deployment path: open an already-compiled
// .dat file with Open — the way a long-running service loads a prebuilt
// dictionary at startup. Open mmaps the hot sections, so the returned
// Dictionary must be closed with Close when no longer needed.
//
// Build a .dat first with the CLI, then point this example at it:
//
//	gomorphy update pymorphy
//	go run ./examples/open -dat .data/pymorphy/pymorphy.dat
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func main() {
	path := flag.String("dat", "", "path to a compiled .dat file (see `gomorphy build`/`gomorphy update`)")
	word := flag.String("word", "кота", "word to look up")
	flag.Parse()

	if *path == "" {
		log.Fatal("-dat is required; run `gomorphy update pymorphy` (or opencorpora/unimorph) first")
	}

	d, err := morphology.Open(*path)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = d.Close() }()

	for _, r := range d.Parse(*word) {
		fmt.Printf("%s -> %s (%s)\n", r.Word, r.Normal, r.Tag)
	}
}
