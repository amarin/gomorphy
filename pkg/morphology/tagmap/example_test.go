package tagmap_test

import (
	"fmt"

	"github.com/amarin/gomorphy/pkg/morphology/tagmap"
)

// ExampleMap normalizes the same form's tag from two sources and compares
// the case dimension, which both mark.
func ExampleMap() {
	oc, _ := tagmap.Map("opencorpora", "NOUN,anim,masc,sing,gent")
	um, _ := tagmap.Map("unimorph", "N;GEN;SG")

	caseOf := func(b tagmap.Bundle) string {
		for _, f := range b.Features {
			if f.Dim == tagmap.DimCase {
				return f.Value
			}
		}
		return ""
	}
	fmt.Println(caseOf(oc), caseOf(um))
	// Output: GEN GEN
}

// ExampleKnown shows which tag sets can be normalized: the bundled
// importers' ones, but not a Builder dictionary's own tags.
func ExampleKnown() {
	for _, name := range []string{"opencorpora", "opencorpora-int", "unimorph", "builder"} {
		fmt.Println(name, tagmap.Known(name))
	}
	// Output:
	// opencorpora true
	// opencorpora-int true
	// unimorph true
	// builder false
}
