package morphology_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/amarin/gomorphy/pkg/morphology/tagmap"
)

// exampleDictXML is a minimal, valid OpenCorpora dict.xml fragment used by
// the examples below to build a working Dictionary with no external data
// files - the same shape morphology.CompileFromXML expects from a real,
// full dict.xml.
const exampleDictXML = `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">существительное</grammeme>
  <grammeme id="nomn">им. п.</grammeme>
  <grammeme id="gent">род. п.</grammeme>
  <grammeme id="anim">одуш.</grammeme>
  <grammeme id="inan">неодуш.</grammeme>
  <grammeme id="masc">м. р.</grammeme>
  <grammeme id="sing">ед. ч.</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="кот">
   <l t="кот"><g v="NOUN"/><g v="anim"/><g v="masc"/><g v="sing"/></l>
   <f t="кот"><g v="nomn"/></f>
   <f t="кота"><g v="gent"/></f>
  </lemma>
  <lemma id="2" text="код">
   <l t="код"><g v="NOUN"/><g v="inan"/><g v="masc"/><g v="sing"/></l>
   <f t="код"><g v="nomn"/></f>
  </lemma>
 </lemmata>
</dictionary>`

// exampleDictXML2 is a second, independent dict.xml fragment used by
// ExampleNewMultiDictionary to demonstrate combining several dictionaries.
const exampleDictXML2 = `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">существительное</grammeme>
  <grammeme id="nomn">им. п.</grammeme>
  <grammeme id="gent">род. п.</grammeme>
  <grammeme id="inan">неодуш.</grammeme>
  <grammeme id="masc">м. р.</grammeme>
  <grammeme id="sing">ед. ч.</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="дом">
   <l t="дом"><g v="NOUN"/><g v="inan"/><g v="masc"/><g v="sing"/></l>
   <f t="дом"><g v="nomn"/></f>
   <f t="дома"><g v="gent"/></f>
  </lemma>
 </lemmata>
</dictionary>`

// mustCompileExampleDict compiles exampleDictXML, the fixture every example
// below that doesn't specifically demonstrate compilation itself reuses.
func mustCompileExampleDict() *morphology.Dictionary {
	d, err := morphology.CompileFromXML(strings.NewReader(exampleDictXML), nil)
	if err != nil {
		log.Fatal(err)
	}
	return d
}

// Example_compileFromXML shows the simplest way to embed gomorphy in a
// program: compile a dictionary directly from an in-memory dict.xml (or any
// io.Reader - a file, an HTTP response body, etc.), with no separate
// download/build step. This is the same entry point gomorphy_build's
// "compile" command and the OpenCorpora importer use internally.
func Example_compileFromXML() {
	d, err := morphology.CompileFromXML(strings.NewReader(exampleDictXML), nil)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range d.Parse("кота") {
		fmt.Println(r.Normal, r.Tag)
	}
	// Output:
	// кот NOUN,anim,masc,sing,gent
}

// Example_openByAbsolutePath shows the on-disk embedding path: compile a
// dictionary once, save it to a .dat file with SaveTo, then later reopen it
// from its absolute path with Open - the way a long-running service loads
// a prebuilt dictionary at startup. Open mmaps the hot sections, so the
// returned Dictionary must be closed with Close when no longer needed.
func Example_openByAbsolutePath() {
	d := mustCompileExampleDict()

	path := filepath.Join(os.TempDir(), "gomorphy-example.dat")
	if err := d.SaveTo(path); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.Remove(path) }()

	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	reopened, err := morphology.Open(absPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = reopened.Close() }()

	for _, r := range reopened.Parse("кота") {
		fmt.Println(r.Normal, r.Tag)
	}
	// Output:
	// кот NOUN,anim,masc,sing,gent
}

// ExampleDictionary_Lemma shows looking up a word's lemma (base form) and
// its tag directly, without walking Parse's readings by hand.
func ExampleDictionary_Lemma() {
	d := mustCompileExampleDict()

	for _, l := range d.Lemma("кота") {
		fmt.Println(l.Normal, l.Tag)
	}
	// Output:
	// кот NOUN,anim,masc,sing,nomn
}

// ExampleDictionary_Fuzzy shows finding dictionary words within a given
// Levenshtein distance of a (possibly misspelled) query - useful for
// "did you mean" style lookups.
func ExampleDictionary_Fuzzy() {
	d := mustCompileExampleDict()

	for _, m := range d.Fuzzy("ко", 1) {
		fmt.Println(m.Word, m.Distance)
	}
	// Output:
	// код 1
	// кот 1
}

// ExampleDictionary_FuzzyTop shows finding the N dictionary words nearest
// to a query, widening the search distance automatically until enough
// matches are found.
func ExampleDictionary_FuzzyTop() {
	d := mustCompileExampleDict()

	for _, m := range d.FuzzyTop("кот", 3) {
		fmt.Println(m.Word, m.Distance)
	}
	// Output:
	// кот 0
	// код 1
	// кота 1
}

// ExampleNewMultiDictionary shows querying several independently opened
// dictionaries as one - e.g. a main dictionary plus a custom supplementary
// one. Every Reading/LemmaRef/FuzzyMatch is tagged with which dictionary
// (by registration order) produced it via its Dict field.
func ExampleNewMultiDictionary() {
	main := mustCompileExampleDict()
	extra, err := morphology.CompileFromXML(strings.NewReader(exampleDictXML2), nil)
	if err != nil {
		log.Fatal(err)
	}

	multi := morphology.NewMultiDictionary(main, extra)
	defer func() { _ = multi.Close() }()

	for _, r := range multi.Parse("кот") {
		fmt.Println(r.Dict, r.Normal, r.Tag)
	}
	for _, r := range multi.Parse("дом") {
		fmt.Println(r.Dict, r.Normal, r.Tag)
	}
	// Output:
	// 0 кот NOUN,anim,masc,sing,nomn
	// 1 дом NOUN,inan,masc,sing,nomn
}

// ExampleMultiDictionary_DictTagSetName shows combining a Reading from
// MultiDictionary with pkg/morphology/tagmap to get a universal tag
// comparable across dictionaries with different native tag syntax:
// Reading.Dict says which dictionary produced the Reading, and
// DictTagSetName(reading.Dict) gives tagmap.Map the dictName it needs.
func ExampleMultiDictionary_DictTagSetName() {
	main := mustCompileExampleDict()
	extra, err := morphology.CompileFromXML(strings.NewReader(exampleDictXML2), nil)
	if err != nil {
		log.Fatal(err)
	}

	multi := morphology.NewMultiDictionary(main, extra)
	defer func() { _ = multi.Close() }()

	for _, r := range multi.Parse("дом") {
		bundle, ok := tagmap.Map(multi.DictTagSetName(r.Dict), r.Tag)
		if !ok {
			log.Fatal("unrecognized dictName")
		}
		for _, f := range bundle.Features {
			fmt.Println(f.Value)
		}
	}
	// Output:
	// N
	// INAN
	// NOM
	// SG
	// MASC
}
