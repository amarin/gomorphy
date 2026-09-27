// Command inflect shows the 1.3.0 lexeme-access API: Forms lists every
// wordform of a lexeme, Inflect picks the ones matching requested
// grammemes, and Grammemes/HasGrammeme/POS read a tag's grammemes without
// splitting the native string by hand. It also shows two things unique to
// a Builder-made dictionary: homonymous lemmas grouped by part-of-speech
// class («знать» NOUN vs. INFN), and Forms/Inflect working on a predicted
// (out-of-dictionary) reading just as well as on a real one.
//
// Run: go run ./examples/inflect
package main

import (
	"fmt"
	"log"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func main() {
	b := morphology.NewBuilder(morphology.BuilderOptions{Language: "ru"})
	for _, f := range [][3]string{
		{"кот", "кот", "NOUN,anim,masc,sing,nomn"},
		{"кота", "кот", "NOUN,anim,masc,sing,gent"},
		{"коты", "кот", "NOUN,anim,masc,plur,nomn"},
		{"котов", "кот", "NOUN,anim,masc,plur,gent"},
		// «знать» is a homonym: a NOUN (nobility) and an INFN (to know)
		// with the same wordform. Builder groups entries by (lemma,
		// POS class), so this becomes two lexemes, not one.
		{"знать", "знать", "NOUN,femn,sing,nomn"},
		{"знать", "знать", "INFN"},
	} {
		if err := b.AddForm(f[0], f[1], f[2]); err != nil {
			log.Fatal(err)
		}
	}
	d, err := b.Build()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = d.Close() }()

	fmt.Println("# Forms of a parsed word")
	r := d.Parse("кота")[0]
	for _, f := range d.Forms(r) {
		fmt.Println(f.Word, f.Tag)
	}

	fmt.Println("# Inflect into plural genitive")
	for _, f := range d.Inflect(r, "plur", "gent") {
		fmt.Println(f.Word, f.Tag)
	}

	fmt.Println("# Homonym: two lexemes for one wordform")
	for _, hr := range d.Parse("знать") {
		fmt.Println(hr.Normal, hr.Tag, "POS="+morphology.POS(hr.Tag))
	}

	fmt.Println("# Forms/Inflect on a predicted word")
	pr := d.Parse("рота")[0] // unknown word, guessed from "кота"'s ending
	fmt.Println(pr.Predicted, pr.Tag)
	for _, f := range d.Inflect(pr, "nomn", "sing") {
		fmt.Println(f.Word, f.Predicted)
	}

	fmt.Println("# HasGrammeme without parsing the tag string")
	fmt.Println(r.HasGrammeme("gent"), r.HasGrammeme("plur"))
	// Output:
	// # Forms of a parsed word
	// кот NOUN,anim,masc,sing,nomn
	// кота NOUN,anim,masc,sing,gent
	// коты NOUN,anim,masc,plur,nomn
	// котов NOUN,anim,masc,plur,gent
	// # Inflect into plural genitive
	// котов NOUN,anim,masc,plur,gent
	// # Homonym: two lexemes for one wordform
	// знать NOUN,femn,sing,nomn POS=NOUN
	// знать INFN POS=INFN
	// # Forms/Inflect on a predicted word
	// true NOUN,anim,masc,sing,gent
	// рот true
	// # HasGrammeme without parsing the tag string
	// true false
}
