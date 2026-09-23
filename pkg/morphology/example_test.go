package morphology_test

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
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

// exampleUniMorphTSV is a minimal UniMorph TSV fragment (lemma<TAB>
// wordform<TAB>bundle), used by the CompileFromUniMorph* examples below.
const exampleUniMorphTSV = "кот\tкот\tN;NOM;SG\n" +
	"кот\tкота\tN;ACC;SG\n"

// mustCompileExampleDict compiles exampleDictXML, the fixture every example
// below that doesn't specifically demonstrate compilation itself reuses.
func mustCompileExampleDict() *morphology.Dictionary {
	d, err := morphology.CompileFromXML(strings.NewReader(exampleDictXML), nil)
	if err != nil {
		log.Fatal(err)
	}
	return d
}

// mustPyMorphyExampleDir builds a minimal pymorphy2-format directory (the
// same shape OpenPyMorphy reads from a real pymorphy2-dicts-ru install:
// words.dawg + paradigms.array + JSON side tables) in a temp directory, so
// ExampleOpenPyMorphy needs no external data. Reuses fixture_test.go's
// payloadSeparator/b64/readingValue helpers (same package).
func mustPyMorphyExampleDir() string {
	dir, err := os.MkdirTemp("", "gomorphy-example-pymorphy")
	if err != nil {
		log.Fatal(err)
	}

	write := func(name string, data []byte) {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			log.Fatal(err)
		}
	}

	// paradigms.array: 1 paradigm, 2 forms — suffix ids | tag ids | prefix ids.
	paraData := []uint16{0, 1, 0, 1, 0, 0}
	paradigms := make([]byte, 4+2*len(paraData))
	binary.LittleEndian.PutUint16(paradigms[0:], 1)                     // paradigm count
	binary.LittleEndian.PutUint16(paradigms[2:], uint16(len(paraData))) // paradigm 0 length
	for i, v := range paraData {
		binary.LittleEndian.PutUint16(paradigms[4+2*i:], v)
	}
	write("paradigms.array", paradigms)
	write("suffixes.json", []byte(`["","а"]`))
	write("paradigm-prefixes.json", []byte(`[""]`))
	write("gramtab-opencorpora-int.json", []byte(`["NOUN,anim,masc,sing,nomn","NOUN,anim,masc,sing,gent"]`))

	words := map[string]uint32{
		"кот" + payloadSeparator + b64(readingValue(0, 0)):  0,
		"кота" + payloadSeparator + b64(readingValue(0, 1)): 0,
	}
	wordsDAWG, guide := testdawg.Build(words)
	write("words.dawg", testdawg.Marshal(wordsDAWG, guide))

	return dir
}

// ExampleCompileFromXML shows the simplest way to embed gomorphy in a
// program: compile a dictionary directly from an in-memory dict.xml (or any
// io.Reader - a file, an HTTP response body, etc.), with no separate
// download/build step. This is the same entry point gomorphy_build's
// "compile" command and the OpenCorpora importer use internally.
func ExampleCompileFromXML() {
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

// ExampleCompileFromXMLFile shows compiling straight from a dict.xml file on
// disk, rather than an in-memory io.Reader (see ExampleCompileFromXML) -
// what `gomorphy build opencorpora -i dict.xml` uses internally.
func ExampleCompileFromXMLFile() {
	path := filepath.Join(os.TempDir(), "gomorphy-example-dict.xml")
	if err := os.WriteFile(path, []byte(exampleDictXML), 0o644); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.Remove(path) }()

	d, err := morphology.CompileFromXMLFile(path, nil)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range d.Parse("кота") {
		fmt.Println(r.Normal, r.Tag)
	}
	// Output:
	// кот NOUN,anim,masc,sing,gent
}

// ExampleOpenPyMorphy shows reading a pymorphy2 dictionary directly from its
// source directory (words.dawg, paradigms.array, ...), without a separate
// compile-to-.dat step. A real directory comes from `gomorphy download
// pymorphy`; this example builds a minimal one inline (mustPyMorphyExampleDir)
// so it needs no external data.
func ExampleOpenPyMorphy() {
	dir := mustPyMorphyExampleDir()
	defer func() { _ = os.RemoveAll(dir) }()

	d, err := morphology.OpenPyMorphy(dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range d.Parse("кота") {
		fmt.Println(r.Normal, r.Tag)
	}
	// Output:
	// кот NOUN,anim,masc,sing,gent
}

// ExampleCompileFromUniMorph shows compiling a dictionary from a UniMorph
// TSV (lemma<TAB>wordform<TAB>bundle). Language is mandatory; "ru" is
// currently the only accepted value (see
// docs/en/implementation/stage-16-import-unimorph.md).
func ExampleCompileFromUniMorph() {
	d, err := morphology.CompileFromUniMorph(strings.NewReader(exampleUniMorphTSV), morphology.UniMorphOptions{Language: "ru"})
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range d.Parse("кота") {
		fmt.Println(r.Normal, r.Tag)
	}
	// Output:
	// кот N;ACC;SG
}

// ExampleCompileFromUniMorphFile shows the same compilation, straight from a
// TSV file on disk rather than an in-memory io.Reader (see
// ExampleCompileFromUniMorph).
func ExampleCompileFromUniMorphFile() {
	path := filepath.Join(os.TempDir(), "gomorphy-example.tsv")
	if err := os.WriteFile(path, []byte(exampleUniMorphTSV), 0o644); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.Remove(path) }()

	d, err := morphology.CompileFromUniMorphFile(path, morphology.UniMorphOptions{Language: "ru"})
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range d.Parse("кота") {
		fmt.Println(r.Normal, r.Tag)
	}
	// Output:
	// кот N;ACC;SG
}

// ExampleDictionary_SaveTo shows saving a compiled dictionary to a .dat
// file, to be reopened later with Open (see ExampleOpen) instead of
// recompiling from source every time.
func ExampleDictionary_SaveTo() {
	d := mustCompileExampleDict()

	path := filepath.Join(os.TempDir(), "gomorphy-example-savetotest.dat")
	if err := d.SaveTo(path); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.Remove(path) }()

	fmt.Println("saved")
	// Output:
	// saved
}

// ExampleOpen shows the on-disk embedding path: compile a dictionary once,
// save it to a .dat file with SaveTo, then later reopen it from its
// absolute path with Open - the way a long-running service loads a
// prebuilt dictionary at startup. Open mmaps the hot sections, so the
// returned Dictionary must be closed with Close when no longer needed.
func ExampleOpen() {
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

// ExampleNewBuilder shows creating a dictionary from scratch, entirely in
// memory with no source files: register wordforms with AddForm/AddLemma,
// then Build the dictionary. The wordform's tag is an opaque string stored
// verbatim (no mapping onto the OpenCorpora grammeme set) — useful for
// small thematic dictionaries.
func ExampleNewBuilder() {
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

	for _, r := range d.Parse("кота") {
		fmt.Println(r.Normal, r.Tag)
	}
	// Output:
	// кот NOUN,anim,masc,sing,gent
}

// ExampleImportTSV shows building a dictionary from a tab-separated stream
// (lemma<TAB>wordform[<TAB>tags]) — the same format the `gomorphy import
// tsv` CLI command reads. The third column is optional; tags are opaque and
// registered automatically as grammemes.
func ExampleImportTSV() {
	tsv := "кот\tкот\tNOUN,anim,masc,sing,nomn\n" +
		"кот\tкота\tNOUN,anim,masc,sing,gent\n" +
		"мышь\tмыши\tNOUN,anim,femn,sing,gent\n"

	d, err := morphology.ImportTSV(strings.NewReader(tsv), morphology.BuilderOptions{Language: "ru"})
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range d.Parse("кота") {
		fmt.Println(r.Normal, r.Tag)
	}
	// Output:
	// кот NOUN,anim,masc,sing,gent
}

// ExampleMerge shows combining two compiled dictionaries into one. The
// base's readings win for words present in both (MergeAdd); words unique to
// the overlay are added. Input dictionaries are never mutated.
func ExampleMerge() {
	base := mustCompileExampleDict()

	overlay := morphology.NewBuilder(morphology.BuilderOptions{Language: "ru"})
	if err := overlay.AddLemma("котёнок", "NOUN,anim,masc,sing,nomn"); err != nil {
		log.Fatal(err)
	}
	overlayDict, err := overlay.Build()
	if err != nil {
		log.Fatal(err)
	}

	merged, err := morphology.Merge(base, []*morphology.Dictionary{overlayDict}, morphology.MergeAdd)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range merged.Parse("котёнок") {
		fmt.Println(r.Normal, r.Tag)
	}
	// Output:
	// котёнок NOUN,anim,masc,sing,nomn
}

// ExampleMergeReplace shows merging in "replace" mode: for a word present
// in both dictionaries the overlay readings fully replace the base's.
func ExampleMergeReplace() {
	overlay := morphology.NewBuilder(morphology.BuilderOptions{Language: "ru"})
	// "кот" is in the base fixture too, but with different readings here.
	if err := overlay.AddForm("кот", "кот", "VERB,impf,trans"); err != nil {
		log.Fatal(err)
	}
	overlayDict, err := overlay.Build()
	if err != nil {
		log.Fatal(err)
	}

	merged, err := morphology.Merge(mustCompileExampleDict(), []*morphology.Dictionary{overlayDict}, morphology.MergeReplace)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range merged.Parse("кот") {
		fmt.Println(r.Normal, r.Tag)
	}
	// Output:
	// кот VERB,impf,trans
}

// ExampleMergeWithOptions merges two thematic dictionaries and rebuilds
// prediction so words with the overlay's endings are predicted too.
func ExampleMergeWithOptions() {
	base := morphology.NewBuilder(morphology.BuilderOptions{})
	_ = base.AddForm("кот", "кот", "NOUN,nomn")
	_ = base.AddForm("кота", "кот", "NOUN,gent")
	baseDict, err := base.Build()
	if err != nil {
		log.Fatal(err)
	}
	overlay := morphology.NewBuilder(morphology.BuilderOptions{})
	_ = overlay.AddForm("мышь", "мышь", "NOUN,nomn")
	_ = overlay.AddForm("мыши", "мышь", "NOUN,gent")
	overlayDict, err := overlay.Build()
	if err != nil {
		log.Fatal(err)
	}

	merged, err := morphology.MergeWithOptions(baseDict, []*morphology.Dictionary{overlayDict},
		morphology.MergeOptions{Mode: morphology.MergeAdd, RebuildPrediction: true})
	if err != nil {
		log.Fatal(err)
	}
	// "камыши" is not itself in either dictionary; only the rebuilt
	// prediction (fed by the overlay's "мыши" ending) can analyse it.
	for _, r := range merged.Parse("камыши") {
		fmt.Println(r.Normal, r.Tag)
	}
	// Output:
	// камышь NOUN,gent
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
