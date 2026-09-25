// Command tagmap shows comparing tags across dictionaries of different
// origin: an OpenCorpora tag and a UniMorph tag for the same form look
// nothing alike, but tagmap.Map normalizes both to one UniMorph feature
// bundle. Dictionary.TagSetName gives the dictName Map expects.
//
// Run: go run ./examples/tagmap
package main

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/amarin/gomorphy/pkg/morphology/tagmap"
)

const dictXML = `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">сущ.</grammeme><grammeme id="anim">одуш.</grammeme>
  <grammeme id="masc">м. р.</grammeme><grammeme id="sing">ед. ч.</grammeme>
  <grammeme id="nomn">им. п.</grammeme><grammeme id="gent">род. п.</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="кот">
   <l t="кот"><g v="NOUN"/><g v="anim"/><g v="masc"/><g v="sing"/></l>
   <f t="кот"><g v="nomn"/></f><f t="кота"><g v="gent"/></f>
  </lemma>
 </lemmata>
</dictionary>`

func main() {
	oc, err := morphology.CompileFromXML(strings.NewReader(dictXML), nil)
	if err != nil {
		log.Fatal(err)
	}
	um, err := morphology.CompileFromUniMorph(strings.NewReader("кот\tкота\tN;GEN;SG\n"),
		morphology.UniMorphOptions{Language: "ru"})
	if err != nil {
		log.Fatal(err)
	}

	a, b := oc.Parse("кота")[0], um.Parse("кота")[0]
	ba, _ := tagmap.Map(oc.TagSetName(), a.Tag)
	bb, _ := tagmap.Map(um.TagSetName(), b.Tag)
	fmt.Printf("%s: %s\n%s: %s\n", oc.TagSetName(), a.Tag, um.TagSetName(), b.Tag)
	fmt.Println("same case and number:", sameDims(ba, bb, tagmap.DimCase, tagmap.DimNumber))
	fmt.Println("identical bundles:", reflect.DeepEqual(ba, bb))
	// Output:
	// opencorpora: NOUN,anim,masc,sing,gent
	// unimorph: N;GEN;SG
	// same case and number: true
	// identical bundles: false
}

// sameDims reports whether both bundles carry equal values for every dim.
func sameDims(a, b tagmap.Bundle, dims ...tagmap.Dimension) bool {
	value := func(x tagmap.Bundle, d tagmap.Dimension) string {
		for _, f := range x.Features {
			if f.Dim == d {
				return f.Value
			}
		}
		return ""
	}
	for _, d := range dims {
		if value(a, d) == "" || value(a, d) != value(b, d) {
			return false
		}
	}
	return true
}
