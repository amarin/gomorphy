package build_test

import (
	"reflect"
	"testing"

	"github.com/amarin/gomorphy/internal/build"
)

func mustBuildSmall(t *testing.T) *build.Snapshot {
	t.Helper()

	b := build.NewBuilder()

	if _, err := b.AddGrammeme("NOUN"); err != nil {
		t.Fatal(err)
	}

	for _, g := range []string{"anim", "masc", "fem", "VERB", "gent", "datv"} {
		if _, err := b.AddGrammeme(g); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		lemma string
		base  []string
		forms []struct {
			text string
			gr   []string
		}
	}{
		{
			lemma: "кот",
			base:  []string{"NOUN", "anim", "masc"},
			forms: []struct {
				text string
				gr   []string
			}{
				{"кота", []string{"NOUN", "anim", "masc", "gent"}},
				{"коту", []string{"NOUN", "anim", "masc", "datv"}},
			},
		},
		{
			lemma: "пила",
			base:  []string{"NOUN", "fem"},
			forms: []struct {
				text string
				gr   []string
			}{
				{"пилы", []string{"NOUN", "fem", "gent"}},
			},
		},
		{
			lemma: "пилить",
			base:  []string{"VERB"},
			forms: []struct {
				text string
				gr   []string
			}{
				{"пила", []string{"VERB"}},
				{"пилю", []string{"VERB"}},
			},
		},
		{
			lemma: "стекло",
			base:  []string{"NOUN"},
			forms: []struct {
				text string
				gr   []string
			}{
				{"стекла", []string{"NOUN", "gent"}},
			},
		},
		{
			lemma: "стекать",
			base:  []string{"VERB"},
			forms: []struct {
				text string
				gr   []string
			}{
				{"стекла", []string{"VERB"}},
			},
		},
	}

	for _, c := range cases {
		id, err := b.AddLemma(c.lemma, c.base...)
		if err != nil {
			t.Fatalf("AddLemma(%q): %v", c.lemma, err)
		}

		for _, f := range c.forms {
			if err := b.AddForm(id, f.text, f.gr...); err != nil {
				t.Fatalf("AddForm(%q): %v", f.text, err)
			}
		}
	}

	return b.Build()
}

func TestSmallLookupEveryWord(t *testing.T) {
	snap := mustBuildSmall(t)

	found := map[string]bool{}

	for _, w := range []string{
		"кот", "кота", "коту",
		"пила", "пилы", "пилю",
		"стекло", "стекла",
	} {
		state, ok := snap.LookupWord([]byte(w))
		if !ok {
			t.Fatalf("word %q not found", w)
		}

		if !snap.IsFinal(state) {
			t.Fatalf("state for %q not final", w)
		}

		found[w] = true
	}

	for _, w := range []string{"ко", "коt", "кот123", "", "пил", "стекл", "xyz"} {
		if _, ok := snap.LookupWord([]byte(w)); ok {
			t.Fatalf("absent word %q unexpectedly found", w)
		}
	}

	if !found["пила"] || !found["стекла"] {
		t.Fatal("sanity")
	}
}

func TestHomonymPostings(t *testing.T) {
	snap := mustBuildSmall(t)

	// «пила» is both a noun base form and a verb form of «пилить».
	state, ok := snap.LookupWord([]byte("пила"))
	if !ok {
		t.Fatal("пила not found")
	}

	posts := snap.PostingsOf(state)
	if len(posts) != 2 {
		t.Fatalf("expected 2 postings for homonym пила, got %d", len(posts))
	}

	lemmas := map[uint32]bool{}
	for _, p := range posts {
		if snap.PairTexts[p] != snap.LemmaTexts[0] && !isPairTextOf(snap, p, "пила") {
			t.Fatalf("pair %d text mismatch", p)
		}

		for _, l := range snap.PairLemmasOf(p) {
			lemmas[l] = true
		}
	}

	// Expect lemmas «пила»(1) and «пилить»(2).
	if !lemmas[1] || !lemmas[2] {
		t.Fatalf("homonym lemmas missing: %v", lemmas)
	}
}

func isPairTextOf(snap *build.Snapshot, pair uint32, want string) bool {
	return string(snap.Text(snap.PairTexts[pair])) == want
}

func TestSharedFormAcrossLemmasSameAncode(t *testing.T) {
	b := build.NewBuilder()

	l1, err := b.AddLemma("ток", "NOUN")
	if err != nil {
		t.Fatal(err)
	}

	l2, err := b.AddLemma("токать", "VERB")
	if err != nil {
		t.Fatal(err)
	}

	// Same (text, ancode) from two different lemmas.
	if err := b.AddForm(l1, "тока", "NOUN", "gent"); err != nil {
		t.Fatal(err)
	}

	if err := b.AddForm(l2, "тока", "NOUN", "gent"); err != nil {
		t.Fatal(err)
	}

	snap := b.Build()

	state, ok := snap.LookupWord([]byte("тока"))
	if !ok {
		t.Fatal("тока not found")
	}

	posts := snap.PostingsOf(state)
	if len(posts) != 1 {
		t.Fatalf("expected deduplicated single posting, got %d", len(posts))
	}

	got := snap.PairLemmasOf(posts[0])
	if len(got) != 2 || got[0] != uint32(l1) || got[1] != uint32(l2) {
		t.Fatalf("expected both lemmas on shared pair, got %v", got)
	}
}

func TestDuplicateFormDeduplication(t *testing.T) {
	b := build.NewBuilder()

	l, _ := b.AddLemma("кот", "NOUN")

	for i := 0; i < 3; i++ {
		if err := b.AddForm(l, "кота", "NOUN", "gent"); err != nil {
			t.Fatal(err)
		}
	}

	snap := b.Build()

	state, ok := snap.LookupWord([]byte("кота"))
	if !ok {
		t.Fatal("кота not found")
	}

	if posts := snap.PostingsOf(state); len(posts) != 1 {
		t.Fatalf("expected 1 posting after dedup, got %d", len(posts))
	}

	if n := len(snap.PairLemmasOf(0)); n != 1 {
		t.Fatalf("expected 1 lemma attachment after dedup, got %d", n)
	}
}

func TestEmptyBuilder(t *testing.T) {
	snap := build.NewBuilder().Build()

	if snap.LemmaCount() != 0 || snap.PairCount() != 0 || snap.StateCount() != 1 {
		t.Fatalf("unexpected counts: lemmas=%d pairs=%d states=%d",
			snap.LemmaCount(), snap.PairCount(), snap.StateCount())
	}

	if _, ok := snap.LookupWord([]byte("x")); ok {
		t.Fatal("empty dictionary must not find anything")
	}
}

func TestBuilderErrors(t *testing.T) {
	b := build.NewBuilder()

	if _, err := b.AddLemma(""); err == nil {
		t.Fatal("expected error for empty lemma text")
	}

	if err := b.AddForm(42, "x", "NOUN"); err == nil {
		t.Fatal("expected error for unknown lemma")
	}

	if err := b.AddForm(0, ""); err == nil {
		t.Fatal("expected error for empty form text")
	}

	if _, err := b.AddGrammeme(""); err == nil {
		t.Fatal("expected error for empty grammeme")
	}
}

func TestBuildDeterminism(t *testing.T) {
	buildOnce := func() *build.Snapshot {
		b := build.NewBuilder()
		l1, _ := b.AddLemma("кот", "NOUN")
		l2, _ := b.AddLemma("собака", "NOUN")

		_, _ = l1, l2

		_ = b.AddForm(l2, "собаки", "NOUN", "gent")
		_ = b.AddForm(l1, "кота", "NOUN", "gent")

		return b.Build()
	}

	a, c := buildOnce(), buildOnce()

	if !reflect.DeepEqual(a, c) {
		t.Fatal("two builds of the same data differ")
	}
}

func TestAncodeOrderSignificant(t *testing.T) {
	b := build.NewBuilder()

	l, err := b.AddLemma("x", "A", "B")
	if err != nil {
		t.Fatal(err)
	}

	// Same grammemes in a different order must form a DIFFERENT ancode.
	if err := b.AddForm(l, "y", "B", "A"); err != nil {
		t.Fatal(err)
	}

	snap := b.Build()

	aBase := snap.PairAncodes[0]
	aForm := snap.PairAncodes[1]

	if aBase == aForm {
		t.Fatal("ordered ancodes collapsed: order must be significant")
	}

	if got := snap.Ancode(aBase); len(got) != 2 || got[0] >= got[1] {
		t.Fatalf("base ancode unexpected: %v", got)
	}

	if n1, n2 := snap.GrammemeNames[snap.Ancode(aBase)[0]], snap.GrammemeNames[snap.Ancode(aForm)[0]]; n1 != "A" || n2 != "B" {
		t.Fatalf("order not preserved: base starts %q, form starts %q", n1, n2)
	}
}

func TestAncodeSameOrderShared(t *testing.T) {
	b := build.NewBuilder()

	l, _ := b.AddLemma("x", "A", "B")

	if err := b.AddForm(l, "y", "A", "B"); err != nil {
		t.Fatal(err)
	}

	snap := b.Build()

	if snap.PairAncodes[0] != snap.PairAncodes[1] {
		t.Fatal("same-order sets must share one ancode")
	}

	if n := len(snap.AncodeOff) - 1; n != 1 { // exactly one: A,B
		t.Fatalf("expected 1 ancode total, got %d", n)
	}
}
