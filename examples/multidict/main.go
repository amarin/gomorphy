// Command multidict shows querying several independently compiled
// dictionaries as one via MultiDictionary — e.g. a main dictionary plus a
// custom supplementary one. Every Reading is tagged with which dictionary
// (by registration order) produced it, via its Dict field.
//
// Run: go run ./examples/multidict
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/amarin/gomorphy/pkg/morphology"
)

const mainXML = `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">существительное</grammeme>
  <grammeme id="nomn">им. п.</grammeme>
  <grammeme id="anim">одуш.</grammeme>
  <grammeme id="masc">м. р.</grammeme>
  <grammeme id="sing">ед. ч.</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="кот">
   <l t="кот"><g v="NOUN"/><g v="anim"/><g v="masc"/><g v="sing"/></l>
   <f t="кот"><g v="nomn"/></f>
  </lemma>
 </lemmata>
</dictionary>`

const extraXML = `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">существительное</grammeme>
  <grammeme id="nomn">им. п.</grammeme>
  <grammeme id="inan">неодуш.</grammeme>
  <grammeme id="masc">м. р.</grammeme>
  <grammeme id="sing">ед. ч.</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="дом">
   <l t="дом"><g v="NOUN"/><g v="inan"/><g v="masc"/><g v="sing"/></l>
   <f t="дом"><g v="nomn"/></f>
  </lemma>
 </lemmata>
</dictionary>`

func main() {
	main1, err := morphology.CompileFromXML(strings.NewReader(mainXML), nil)
	if err != nil {
		log.Fatal(err)
	}
	extra, err := morphology.CompileFromXML(strings.NewReader(extraXML), nil)
	if err != nil {
		log.Fatal(err)
	}

	multi := morphology.NewMultiDictionary(main1, extra)
	defer func() { _ = multi.Close() }()

	for _, word := range []string{"кот", "дом"} {
		for _, r := range multi.Parse(word) {
			fmt.Printf("dict#%d: %s -> %s (%s)\n", r.Dict, r.Word, r.Normal, r.Tag)
		}
	}
	// Output:
	// dict#0: кот -> кот (NOUN,anim,masc,sing,nomn)
	// dict#1: дом -> дом (NOUN,inan,masc,sing,nomn)
}
