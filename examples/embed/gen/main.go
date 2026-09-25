// Command gen writes examples/embed/names.dat, the tiny dictionary the embed
// example compiles into its binary. Rerun it after a format change:
//
//	go run ./examples/embed/gen
package main

import (
	"log"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func main() {
	b := morphology.NewBuilder(morphology.BuilderOptions{Language: "ru", Source: "names"})
	for _, f := range [][3]string{
		{"Москва", "Москва", "GEO,nomn"},
		{"Москвы", "Москва", "GEO,gent"},
		{"Пушкин", "Пушкин", "PERSON,nomn"},
		{"Пушкина", "Пушкин", "PERSON,gent"},
	} {
		if err := b.AddForm(f[0], f[1], f[2]); err != nil {
			log.Fatal(err)
		}
	}
	d, err := b.Build()
	if err != nil {
		log.Fatal(err)
	}
	if err := d.SaveTo("examples/embed/names.dat"); err != nil {
		log.Fatal(err)
	}
}
