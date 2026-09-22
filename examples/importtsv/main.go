// Command importtsv shows building a dictionary from a tab-separated
// wordform stream with ImportTSV — the same format the `gomorphy import
// tsv` CLI command reads: lemma<TAB>wordform[<TAB>tags]. The tags column is
// optional; tags are opaque and registered automatically as grammemes.
//
// Run: go run ./examples/importtsv
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func main() {
	tsv := "кот\tкот\tNOUN,anim,masc,sing,nomn\n" +
		"кот\tкота\tNOUN,anim,masc,sing,gent\n" +
		"мышь\tмыши\tNOUN,anim,femn,sing,gent\n"

	d, err := morphology.ImportTSV(strings.NewReader(tsv), morphology.BuilderOptions{Language: "ru"})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = d.Close() }()

	for _, word := range []string{"кот", "кота", "мыши"} {
		for _, r := range d.Parse(word) {
			fmt.Println(r.Word, "->", r.Normal, "("+r.Tag+")")
		}
	}
}
