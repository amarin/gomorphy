// Command opencorpora shows the simplest way to embed gomorphy: compile a
// dictionary directly from a dict.xml (here, a tiny inline fragment, so this
// example runs with no external data), then Parse a word.
//
// Run: go run ./examples/opencorpora
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// A minimal, valid OpenCorpora dict.xml fragment. A real dictionary (from
// opencorpora.org, or via `gomorphy download opencorpora`) has the same
// shape at a much larger scale — CompileFromXML doesn't care which.
const dictXML = `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">существительное</grammeme>
  <grammeme id="nomn">им. п.</grammeme>
  <grammeme id="gent">род. п.</grammeme>
  <grammeme id="anim">одуш.</grammeme>
  <grammeme id="masc">м. р.</grammeme>
  <grammeme id="sing">ед. ч.</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="кот">
   <l t="кот"><g v="NOUN"/><g v="anim"/><g v="masc"/><g v="sing"/></l>
   <f t="кот"><g v="nomn"/></f>
   <f t="кота"><g v="gent"/></f>
  </lemma>
 </lemmata>
</dictionary>`

func main() {
	// progress is an optional func(processed, total int) callback; nil is fine.
	d, err := morphology.CompileFromXML(strings.NewReader(dictXML), nil)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range d.Parse("кота") {
		fmt.Printf("%s -> %s (%s)\n", r.Word, r.Normal, r.Tag)
	}
	// Output: кота -> кот (NOUN,anim,masc,sing,gent)
}
