// Command contenthash shows keying a cache on dictionary content:
// ContentHash ignores the info section (build time, library version), so it
// survives a re-save or a reopen, and changes only when the words do.
//
// Run: go run ./examples/contenthash
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func build(words ...string) *morphology.Dictionary {
	b := morphology.NewBuilder(morphology.BuilderOptions{Language: "ru"})
	for _, w := range words {
		if err := b.AddLemma(w, "NOUN"); err != nil {
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
	d := build("кот", "код")
	path := filepath.Join(os.TempDir(), "gomorphy-contenthash.dat")
	defer func() { _ = os.Remove(path) }()
	if err := d.SaveTo(path); err != nil {
		log.Fatal(err)
	}
	reopened, err := morphology.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = reopened.Close() }()

	fmt.Println("same after save+open:", d.ContentHash() == reopened.ContentHash())
	fmt.Println("same words, rebuilt: ", d.ContentHash() == build("код", "кот").ContentHash())
	fmt.Println("one word added:      ", d.ContentHash() == build("кот", "код", "кит").ContentHash())
	// Output:
	// same after save+open: true
	// same words, rebuilt:  true
	// one word added:       false
}
