// Command builder shows creating a dictionary from scratch with the public
// Builder API: register wordforms with AddForm/AddLemma, then Build — no
// source files, no external data, entirely in memory. Tags are opaque
// strings stored verbatim (no OpenCorpora grammeme mapping).
//
// Run: go run ./examples/builder
package main

import (
	"fmt"
	"log"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func main() {
	b := morphology.NewBuilder(morphology.BuilderOptions{Language: "ru"})
	if err := b.AddLemma("кот", "NOUN,anim,masc,sing,nomn"); err != nil {
		log.Fatal(err)
	}
	if err := b.AddForm("кота", "кот", "NOUN,anim,masc,sing,gent"); err != nil {
		log.Fatal(err)
	}

	d, err := b.Build()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = d.Close() }()

	for _, r := range d.Parse("кота") {
		fmt.Println(r.Word, "->", r.Normal, "("+r.Tag+")")
	}
}
