package morphology_test

import (
	"os"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// benchDict is a Builder dictionary of 10 masculine nouns × 7 forms:
// dense alphabet, CharPolicy е→ё, prediction — the shape of a lexicon
// dictionary.
func benchDict(b *testing.B) *morphology.Dictionary {
	b.Helper()
	stems := []string{"кот", "ход", "лес", "дом", "стол", "мост", "сад", "нос", "рот", "лёд"}
	endings := []struct{ suffix, tag string }{
		{"", "NOUN,inan,masc,sing,nomn"},
		{"а", "NOUN,inan,masc,sing,gent"},
		{"у", "NOUN,inan,masc,sing,datv"},
		{"ом", "NOUN,inan,masc,sing,ablt"},
		{"е", "NOUN,inan,masc,sing,loct"},
		{"ы", "NOUN,inan,masc,plur,nomn"},
		{"ов", "NOUN,inan,masc,plur,gent"},
	}
	bl := morphology.NewBuilder(morphology.BuilderOptions{})
	for _, s := range stems {
		for _, e := range endings {
			if err := bl.AddForm(s+e.suffix, s, e.tag); err != nil {
				b.Fatal(err)
			}
		}
	}
	d, err := bl.Build()
	if err != nil {
		b.Fatal(err)
	}
	return d
}

func benchParse(b *testing.B, d *morphology.Dictionary, word string) {
	b.Helper()
	if len(d.Parse(word)) == 0 {
		b.Fatalf("no readings for %q", word)
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = d.Parse(word)
	}
}

func BenchmarkParseKnown(b *testing.B)     { benchParse(b, benchDict(b), "кота") }
func BenchmarkParseKnownYo(b *testing.B)   { benchParse(b, benchDict(b), "леда") } // е→ё: «лёда»
func BenchmarkParsePredicted(b *testing.B) { benchParse(b, benchDict(b), "бота") }

func BenchmarkLemma(b *testing.B) {
	d := benchDict(b)
	b.ReportAllocs()
	for b.Loop() {
		_ = d.Lemma("кота")
	}
}

func BenchmarkIsKnown(b *testing.B) {
	d := benchDict(b)
	b.ReportAllocs()
	for b.Loop() {
		_ = d.IsKnown("кота")
	}
}

func BenchmarkFuzzyTop(b *testing.B) {
	d := benchDict(b)
	b.ReportAllocs()
	for b.Loop() {
		_ = d.FuzzyTop("кат", 5)
	}
}

// BenchmarkRealDict measures a real compiled dictionary. Set
// GOMORPHY_BENCH_DICT to a .dat file, e.g. .data/pymorphy/pymorphy.dat.
func BenchmarkRealDict(b *testing.B) {
	path := os.Getenv("GOMORPHY_BENCH_DICT")
	if path == "" {
		b.Skip("GOMORPHY_BENCH_DICT not set")
	}
	d, err := morphology.Open(path)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = d.Close() }()

	for _, w := range []string{"кота", "стали", "ежик"} {
		b.Run("ParseKnown/"+w, func(b *testing.B) { benchParse(b, d, w) })
	}
	for _, w := range []string{"бутявкающий", "глокая"} {
		b.Run("ParsePredicted/"+w, func(b *testing.B) { benchParse(b, d, w) })
	}
	b.Run("Lemma", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = d.Lemma("стали")
		}
	})
	b.Run("FuzzyTop", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = d.FuzzyTop("карова", 5)
		}
	})
}
