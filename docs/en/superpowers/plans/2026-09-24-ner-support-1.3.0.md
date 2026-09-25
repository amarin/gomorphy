# NER support, release 1.3.0 (items G–K) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Quality-of-life additions that the NER work in lexicon relies on, all natural extensions
of the core: tag helpers `Grammemes`/`HasGrammeme`/`POS` (G), lexeme access `Forms`/`Inflect` (H),
`Parse` performance with `ParseAppend` and benchmarks (I), homonymous Builder lemmas of different
parts of speech kept apart (J), and a 2-byte alphabet fallback for Builder/ImportTSV (K). Ship as
1.3.0.

**Architecture:** G adds a zero-allocation grammeme tokenizer in `internal` (so the builder can use
it too) and thin public wrappers. J changes only phase 1 (grouping) of
`internal.BuildDictionaryFromEntries`. H reads the existing paradigm table
(`[suffix_0..N-1 | tag_0..N-1 | prefix_0..N-1]`, `internal/paradigm.go`) exactly the way
`readingForm` already does, just for every form instead of form 0. K moves `Merge`'s
width-1-then-2 alphabet choice into `internal/alphabet.go` and uses it in `RecompileDense`. I first
adds benchmarks and records a baseline, then adds allocation-free lookup primitives in `internal`
(`DenseAlphabet.EncodeRune`, `DAWG.LookupEach`, `DAWG.FindJoined`) and rebuilds `Parse` on them
behind a new `ParseAppend`. The binary format does not change.

**Tech Stack:** Go (`go 1.25.0` directive after 1.2.0 — owner policy "current Go minus two minor
versions" — toolchain 1.27.1), testify. No new
dependencies.

**Spec:** [docs/en/superpowers/specs/2026-09-24-ner-support-design.md](../specs/2026-09-24-ner-support-design.md)

**Prerequisite:** the 1.2.0 plan
([2026-09-24-ner-support-1.2.0.md](2026-09-24-ner-support-1.2.0.md)) is merged. This plan uses
`Reading.Predicted`, `Dictionary.IsKnown`, the test helpers `predictionFixture`/`botDict`
(`pkg/morphology/known_test.go`) and `yolkaDict` (`char_policy_test.go`), and the `go 1.25.0`
directive.

## Global Constraints

- The GMOR binary format does not change. Existing `.dat` files open and parse unchanged.
- **Go 1.25 at most**: `go.mod` says `go 1.25.0` (owner policy "current Go minus two minor
  versions", 1.2.0 Task 10). Everything up to 1.25 is allowed — `strings.SplitSeq`/`FieldsSeq`,
  range-over-func, `iter`, `t.Context()`, `sync.WaitGroup.Go`, and `b.Loop()` (Go 1.24), which
  the benchmarks here use (`for b.Loop() { … }`; no `b.ResetTimer()` needed — setup before the
  first `b.Loop()` call is not timed). Nothing from Go 1.26+. `go vet ./...` (analyzer
  `stdversion`) catches violations.
- Public names exactly as in the spec: `Grammemes(tag string) []string`,
  `HasGrammeme(tag, g string) bool`, `POS(tag string) string`, `(Reading).HasGrammeme(g string) bool`,
  `(*Dictionary).Forms(r Reading) []Reading`, `(*Dictionary).Inflect(r Reading, want ...string) []Reading`,
  the same two on `*MultiDictionary`, `(*Dictionary).ParseAppend(dst []Reading, word string) []Reading`.
- Tags stay native strings. `tagmap` is **not** extended (spec G: UniMorph Schema has no
  Name/Surn/Patr/Geox dimensions).
- Grammeme separators are exactly `,`, ` ` (space) and `;`.
- **`Parse` results must not change** in content or order for any dictionary: every existing test
  that snapshots `Parse` (`readingsSnapshot`, `semanticSnapshot`, `TestMergeRealDictionary`) must
  stay green, and Task 8 adds an equivalence test of the new lookup against `SimilarItems`.
- Run after every task: `go build ./... && go vet ./... && go test ./... -race -count=1`, and
  `golangci-lint run ./...` before the task's commit; `gofmt -l .` prints nothing.
- Commit messages follow the repo style and end with
  `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Stage only the files the task lists.

## File Structure

| File | Change | Responsibility |
|---|---|---|
| `pkg/morphology/internal/tag.go` | create | `NextGrammeme`, `FirstGrammeme`, `POSClass` |
| `pkg/morphology/internal/tag_test.go` | create | tokenizer tests |
| `pkg/morphology/tag.go` | create | public `Grammemes`, `HasGrammeme`, `POS`, `Reading.HasGrammeme` |
| `pkg/morphology/tag_test.go` | create | public tag helper tests + `AllocsPerRun` |
| `pkg/morphology/internal/build.go` | modify | group by (lemma, POS class) |
| `pkg/morphology/internal/build_test.go` | modify | grouping tests |
| `pkg/morphology/builder.go` | modify | `AddForm` doc (grouping rule) |
| `pkg/morphology/builder_test.go` | modify | homonym tests |
| `pkg/morphology/lexeme.go` | create | `Forms`, `Inflect` (Dictionary + MultiDictionary) |
| `pkg/morphology/lexeme_test.go` | create | lexeme tests |
| `pkg/morphology/example_test.go` | modify | `ExampleDictionary_Forms`, `ExampleDictionary_Inflect` |
| `pkg/morphology/internal/alphabet.go` | modify | `NewDenseAlphabetFor`, `DenseAlphabet.EncodeRune` |
| `pkg/morphology/internal/merge.go` | modify | use `NewDenseAlphabetFor` |
| `pkg/morphology/internal/dense_recompile.go` | modify | 2-byte fallback |
| `pkg/morphology/internal/dense_recompile_test.go` | modify | fallback tests |
| `pkg/morphology/builder_internal_test.go` | modify | wide-alphabet builder test |
| `pkg/morphology/bench_test.go` | create | benchmarks (fixture + real dictionary) |
| `pkg/morphology/internal/similar_items.go` | modify | `followRuneVia` fast path |
| `pkg/morphology/internal/lookup_each.go` | create | `LookupEach`, `forEachValue`, `FindJoined` |
| `pkg/morphology/internal/lookup_each_test.go` | create | equivalence tests |
| `pkg/morphology/parse.go` | modify | `ParseAppend`, single-shard path, new primitives |
| `pkg/morphology/parse_alloc_test.go` | create | allocation bounds |
| `pkg/morphology/race_on_test.go`, `race_off_test.go` (+ same in `internal/`) | create | `raceEnabled` guard for `AllocsPerRun` |
| `pkg/morphology/version.go`, `CHANGELOG.md`, `docs/en/library.md`, `docs/ru/library.md`, `docs/en/todo.md`, `docs/en/implementation.md`, `docs/en/implementation/ner-support.md` | modify | docs |

## Planning-time measurements (1.1.0 source, Builder dictionary кот/кота/мышь/мыши)

`testing.AllocsPerRun(200, …)` on `d.Parse(w)`: `"кот"` → **22**, `"кота"` → **24**,
`"бота"` (predicted) → **44**. Sources, from reading the code: a goroutine + `WaitGroup` even for
one shard (`parse.go:51-62`); `[]rune(key)` and `Item` slices in `SimilarItems`; one
`Alphabet.Encode(string(r))` per rune (a `string` and a `[]byte`, `internal/similar_items.go:30`);
`b64d` per value plus the completer's `key`/`indexStack` slices (`internal/dawg.go:232-240`);
`key+":"+tag` per reading when probabilities exist (`parse.go:97`); `suffixSplits` allocating
`[]rune` plus two strings per split (`parse.go:291-305`); the `seen` key concatenation
(`parse.go:177`).

---

### Task 1: Tag helpers (item G)

**Files:**
- Create: `pkg/morphology/internal/tag.go`, `pkg/morphology/internal/tag_test.go`
- Create: `pkg/morphology/tag.go`, `pkg/morphology/tag_test.go`
- Create: `pkg/morphology/race_on_test.go`, `pkg/morphology/race_off_test.go`

**Interfaces:**
- Produces (tests): `const raceEnabled bool` in package `morphology_test` — `AllocsPerRun`
  assertions are skipped under `-race` (the race detector changes allocation counts). Task 9 reuses it.
- Produces (internal):
  - `func NextGrammeme(tag string, i int) (tok string, next int)` — the token starting at or after
    byte offset `i` (leading separators skipped) and the offset just past it; `tok == ""` at the end.
  - `func FirstGrammeme(tag string) string`
  - `func POSClass(tag string) string` — `FirstGrammeme` mapped through the part-of-speech class
    table (used by Task 2).
- Produces (public): `Grammemes`, `HasGrammeme`, `POS`, `Reading.HasGrammeme`.

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/race_on_test.go`:

```go
//go:build race

package morphology_test

// raceEnabled: the race detector changes allocation counts, so
// AllocsPerRun bounds are only checked without it.
const raceEnabled = true
```

Create `pkg/morphology/race_off_test.go`:

```go
//go:build !race

package morphology_test

const raceEnabled = false
```

Create `pkg/morphology/internal/tag_test.go`:

```go
package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNextGrammeme(t *testing.T) {
	tag := "NOUN,anim,masc,Surn sing,ablt"
	var got []string
	for i := 0; ; {
		tok, next := NextGrammeme(tag, i)
		if tok == "" {
			break
		}
		got = append(got, tok)
		i = next
	}
	assert.Equal(t, []string{"NOUN", "anim", "masc", "Surn", "sing", "ablt"}, got)

	tok, next := NextGrammeme(" ,;", 0)
	assert.Equal(t, "", tok)
	assert.Equal(t, 3, next)
}

func TestFirstGrammeme(t *testing.T) {
	assert.Equal(t, "NOUN", FirstGrammeme("NOUN,anim,masc sing,nomn"))
	assert.Equal(t, "N", FirstGrammeme("N;GEN;SG"))
	assert.Equal(t, "", FirstGrammeme(""))
	assert.Equal(t, "INFN", FirstGrammeme(" INFN,impf"))
}

func TestPOSClass(t *testing.T) {
	for tag, want := range map[string]string{
		"NOUN,anim,masc,sing,nomn":  "NOUN",
		"INFN,impf,tran":            "VERB",
		"VERB,impf,tran,sing,1per":  "VERB",
		"PRTF,impf,pres,actv":       "VERB",
		"PRTS,perf,past,pssv":       "VERB",
		"GRND,impf,pres":            "VERB",
		"ADJF,Qual,masc,sing,nomn":  "ADJF",
		"ADJS,masc,sing":            "ADJF",
		"COMP,Qual":                 "ADJF",
		"V;PRS;1;SG":                "V",
		"V.PTCP;ACT;PRS":            "V",
		"V.CVB;PRS":                 "V",
		"V.MSDR":                    "V",
		"ADJ;NOM;SG":                "ADJ",
		"":                          "",
	} {
		assert.Equal(t, want, POSClass(tag), tag)
	}
}
```

Create `pkg/morphology/tag_test.go`:

```go
package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
)

func TestGrammemes(t *testing.T) {
	assert.Equal(t, []string{"NOUN", "anim", "masc", "Surn", "sing", "ablt"},
		morphology.Grammemes("NOUN,anim,masc,Surn sing,ablt"))
	assert.Equal(t, []string{"N", "GEN", "SG"}, morphology.Grammemes("N;GEN;SG"))
	assert.Nil(t, morphology.Grammemes(""))
	assert.Nil(t, morphology.Grammemes(", ;"))
}

func TestHasGrammeme(t *testing.T) {
	tag := "NOUN,anim,masc,Surn sing,ablt"
	assert.True(t, morphology.HasGrammeme(tag, "Surn"))
	assert.True(t, morphology.HasGrammeme(tag, "ablt"))
	assert.True(t, morphology.HasGrammeme("N;GEN;SG", "GEN"))
	assert.False(t, morphology.HasGrammeme(tag, "Sur"), "whole tokens only")
	assert.False(t, morphology.HasGrammeme(tag, "Surn sing"), "a separator is never part of a token")
	assert.False(t, morphology.HasGrammeme(tag, ""))
	assert.False(t, morphology.HasGrammeme("", "NOUN"))

	if !raceEnabled {
		allocs := testing.AllocsPerRun(100, func() { _ = morphology.HasGrammeme(tag, "ablt") })
		assert.Equal(t, 0.0, allocs, "HasGrammeme must not allocate")
	}
}

func TestPOS(t *testing.T) {
	assert.Equal(t, "NOUN", morphology.POS("NOUN,anim,masc sing,nomn"))
	assert.Equal(t, "INFN", morphology.POS("INFN,impf,tran"), "POS is the raw first grammeme, no class mapping")
	assert.Equal(t, "N", morphology.POS("N;GEN;SG"))
	assert.Equal(t, "", morphology.POS(""))
}

func TestReadingHasGrammeme(t *testing.T) {
	rs := parseDict(t).Parse("кота")
	assert.True(t, rs[0].HasGrammeme("gent"))
	assert.False(t, rs[0].HasGrammeme("nomn"))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ ./pkg/morphology/internal/ -run 'Grammeme|POS' -count=1`
Expected: FAIL — `undefined: NextGrammeme`, `undefined: morphology.Grammemes`.

- [ ] **Step 3: Implement**

Create `pkg/morphology/internal/tag.go`:

```go
package internal

// isGrammemeSep reports whether c separates grammemes in a native tag:
// ',' and ' ' (OpenCorpora/pymorphy2, e.g. "NOUN,anim,masc sing,nomn") and
// ';' (UniMorph, e.g. "N;GEN;SG").
func isGrammemeSep(c byte) bool { return c == ',' || c == ' ' || c == ';' }

// NextGrammeme returns the first grammeme token of tag at or after byte
// offset i (leading separators are skipped) and the offset just past it.
// tok is "" once the tag is exhausted. It does not allocate: tok is a
// substring of tag.
func NextGrammeme(tag string, i int) (tok string, next int) {
	for i < len(tag) && isGrammemeSep(tag[i]) {
		i++
	}
	j := i
	for j < len(tag) && !isGrammemeSep(tag[j]) {
		j++
	}
	return tag[i:j], j
}

// FirstGrammeme returns tag's first grammeme — the part of speech in every
// tag format gomorphy imports — or "" for an empty tag.
func FirstGrammeme(tag string) string {
	tok, _ := NextGrammeme(tag, 0)
	return tok
}

// posClasses folds parts of speech that belong to one lexeme into one
// class: an OpenCorpora verb lexeme mixes INFN, VERB, PRTF, PRTS and GRND
// forms, an adjective lexeme ADJF, ADJS and COMP; UniMorph verb lexemes mix
// V, V.PTCP, V.CVB and V.MSDR.
var posClasses = map[string]string{
	"INFN": "VERB", "PRTF": "VERB", "PRTS": "VERB", "GRND": "VERB",
	"ADJS": "ADJF", "COMP": "ADJF",
	"V.PTCP": "V", "V.CVB": "V", "V.MSDR": "V",
}

// POSClass returns the part-of-speech class of tag: its first grammeme,
// with the forms of one lexeme folded together (see posClasses). "" for an
// empty tag.
func POSClass(tag string) string {
	pos := FirstGrammeme(tag)
	if c, ok := posClasses[pos]; ok {
		return c
	}
	return pos
}
```

Create `pkg/morphology/tag.go`:

```go
package morphology

import "github.com/amarin/gomorphy/pkg/morphology/internal"

// Grammemes splits a native tag into grammeme tokens. Separators are ',',
// ' ' and ';', which covers OpenCorpora/pymorphy2 ("NOUN,anim,masc,Surn
// sing,ablt") and UniMorph ("N;GEN;SG"). Empty tokens are dropped; nil for
// a tag without tokens.
func Grammemes(tag string) []string {
	var out []string
	for i := 0; ; {
		tok, next := internal.NextGrammeme(tag, i)
		if tok == "" {
			return out
		}
		out = append(out, tok)
		i = next
	}
}

// HasGrammeme reports whether tag contains the grammeme g as a whole token
// (same separators as Grammemes). It does not allocate.
func HasGrammeme(tag, g string) bool {
	if g == "" {
		return false
	}
	for i := 0; ; {
		tok, next := internal.NextGrammeme(tag, i)
		if tok == "" {
			return false
		}
		if tok == g {
			return true
		}
		i = next
	}
}

// POS returns tag's first grammeme — the part of speech in every tag format
// gomorphy imports ("NOUN", "INFN", "N", …) — or "" for an empty tag.
func POS(tag string) string { return internal.FirstGrammeme(tag) }

// HasGrammeme reports whether the reading's tag contains the grammeme g.
func (r Reading) HasGrammeme(g string) bool { return HasGrammeme(r.Tag, g) }
```

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/internal/tag.go pkg/morphology/internal/tag_test.go pkg/morphology/tag.go pkg/morphology/tag_test.go pkg/morphology/race_on_test.go pkg/morphology/race_off_test.go
git commit -m "feat(morphology): tag helpers Grammemes, HasGrammeme, POS

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Builder keeps homonymous lemmas of different POS apart (item J)

**Files:**
- Modify: `pkg/morphology/internal/build.go:53-66` (lemmaGroup), `:269-300` (phases 1–2)
- Modify: `pkg/morphology/internal/build_test.go` (append)
- Modify: `pkg/morphology/builder.go` (`AddForm` doc)
- Modify: `pkg/morphology/builder_test.go` (append)

**Interfaces:**
- Consumes: `internal.POSClass` (Task 1).
- Produces: behaviour — entries are grouped into paradigms by `(lemma, POSClass(tag))`. Always on:
  no `BuilderOptions` field and no opt-out (owner decision 2026-09-24, answer 17; spec G5).

Grouping rule (deterministic, insertion order):
1. An entry whose `POSClass(tag)` is `""` (empty or POS-less tag) joins the **first** group of its
   lemma, or starts one. This keeps today's behaviour for inputs without POS, e.g. a lemma row
   without tags followed by tagged forms.
2. An entry with class `p` joins the lemma's group whose class is `p`; otherwise it adopts the
   lemma's first class-less group (sets its class to `p`); otherwise it starts a new group.
3. A lemma whose entries all share one class produces exactly the paradigm it produces today.

Why a class, not the raw first grammeme (spec says "POS(tag of the lemma form)"): an OpenCorpora
verb lexeme has INFN, VERB, PRTF, PRTS and GRND forms under one lemma; grouping by raw POS would
split every verb into up to five paradigms and give «знаю» the lemma tag VERB instead of INFN.
Each entry only carries its own tag, so the class has to be computed per entry. See Discrepancy
D-12.

- [ ] **Step 1: Write the failing tests**

Append to `pkg/morphology/internal/build_test.go`:

```go
// «знать» is both a NOUN (nobility) and a verb (to know). The verb's INFN,
// VERB and PRTF forms are one lexeme (one POS class).
func TestBuildDictionaryFromEntriesSplitsLemmaByPOSClass(t *testing.T) {
	entries := []BuildEntry{
		{Word: "знать", Lemma: "знать", Tag: "NOUN,inan,femn,sing,nomn"},
		{Word: "знати", Lemma: "знать", Tag: "NOUN,inan,femn,sing,gent"},
		{Word: "знать", Lemma: "знать", Tag: "INFN,impf,tran"},
		{Word: "знаю", Lemma: "знать", Tag: "VERB,impf,tran,sing,1per,pres,indc"},
		{Word: "знающий", Lemma: "знать", Tag: "PRTF,impf,tran,pres,actv,masc,sing,nomn"},
	}
	dict, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)

	require.Len(t, dict.Paradigms[0], 2, "one NOUN and one VERB-class paradigm")
	pNoun, _ := buildLookup(t, dict, "знати")
	pVerb, fVerb := buildLookup(t, dict, "знаю")
	pPrtf, _ := buildLookup(t, dict, "знающий")
	assert.NotEqual(t, pNoun, pVerb)
	assert.Equal(t, pVerb, pPrtf)
	assert.Equal(t, uint16(1), fVerb, "form 0 of the verb paradigm is its INFN «знать»")
	assert.Equal(t, "INFN,impf,tran", dict.TagSet.TagName(dict.Paradigms[0][pVerb].Tag(0)))
}

// A tag-less entry joins the lemma's first group, and a class-less group
// is adopted by the first tagged entry: same single paradigm as before.
func TestBuildDictionaryFromEntriesEmptyTagJoinsLemmaGroup(t *testing.T) {
	entries := []BuildEntry{
		{Word: "кот", Lemma: "кот", Tag: ""},
		{Word: "кота", Lemma: "кот", Tag: "NOUN,anim,masc,sing,gent"},
		{Word: "коту", Lemma: "кот", Tag: ""},
	}
	dict, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)
	require.Len(t, dict.Paradigms[0], 1)
	assert.Equal(t, 3, dict.Paradigms[0][0].Len())
}
```

Append to `pkg/morphology/builder_test.go`:

```go
func TestBuilderSplitsHomonymousLemmasByPOS(t *testing.T) {
	d := buildFromTriples(t,
		[3]string{"знать", "знать", "NOUN,inan,femn,sing,nomn"},
		[3]string{"знати", "знать", "NOUN,inan,femn,sing,gent"},
		[3]string{"знать", "знать", "INFN,impf,tran"},
		[3]string{"знаю", "знать", "VERB,impf,tran,sing,1per,pres,indc"},
	)

	refs := d.Lemma("знать")
	require.Len(t, refs, 2, "a NOUN lemma and an INFN lemma")
	assert.NotEqual(t, refs[0].Para, refs[1].Para)

	verb := d.Lemma("знаю")
	require.Len(t, verb, 1)
	assert.Equal(t, "INFN,impf,tran", verb[0].Tag)

	noun := d.Lemma("знати")
	require.Len(t, noun, 1)
	assert.Equal(t, "NOUN,inan,femn,sing,nomn", noun[0].Tag)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ ./pkg/morphology/internal/ -run 'POS|Homonymous|EmptyTagJoins' -count=1`
Expected: FAIL — `"[…]" should have 2 item(s), but has 1` (paradigms / lemmas), and
`verb[0].Tag` is `NOUN,inan,femn,sing,nomn`. (`TestBuildDictionaryFromEntriesEmptyTagJoinsLemmaGroup`
already passes — it guards the unchanged behaviour.)

- [ ] **Step 3: Implement**

In `pkg/morphology/internal/build.go`:

1. Extend `lemmaGroup` and its doc:

```go
// lemmaGroup accumulates all forms of one lexeme: one lemma text and one
// part-of-speech class (POSClass; "" until a tagged form arrives). Groups
// are kept in creation order, so the output (shard assignment, paradigm
// and suffix ids, DAWG contents) is deterministic across runs of the same
// input rather than dependent on Go's randomized map iteration order.
type lemmaGroup struct {
	text  string
	pos   string
	forms []buildForm
	seen  map[buildFormKey]bool // "text\x00tag" -> already appended
}
```

2. Replace phase 1 (the `var order []string` … loop) with:

```go
	// Phase 1: group entries by (lemma text, POS class) in first-appearance
	// order, so homonymous lemmas of different parts of speech («знать»
	// NOUN / INFN) get separate paradigms while one lexeme's INFN/VERB/PRTF…
	// forms stay together. A POS-less entry joins the lemma's first group.
	var order []*lemmaGroup
	byLemma := make(map[string][]*lemmaGroup)
	groupFor := func(lemma, pos string) *lemmaGroup {
		groups := byLemma[lemma]
		if pos == "" && len(groups) > 0 {
			return groups[0]
		}
		if pos != "" {
			for _, g := range groups {
				if g.pos == pos {
					return g
				}
			}
			for _, g := range groups {
				if g.pos == "" {
					g.pos = pos
					return g
				}
			}
		}
		g := &lemmaGroup{text: lemma, pos: pos}
		byLemma[lemma] = append(groups, g)
		order = append(order, g)
		return g
	}
	for _, e := range entries {
		lemma := e.Lemma
		if lemma == "" {
			lemma = e.Word
		}
		groupFor(lemma, POSClass(e.Tag)).add(e.Word, e.Tag)
	}
```

3. In phase 2 replace

```go
	for _, lemma := range order {
		g := byLemma[lemma]
		forms := buildForms(g)
```

with

```go
	for _, g := range order {
		forms := buildForms(g)
```

4. Update step 1 of the `BuildDictionaryFromEntries` doc list to: `Entries are grouped by (lemma
   text, POSClass of the tag) in insertion order (an empty lemma falls back to the entry's own
   word; a POS-less entry joins the lemma's first group).`

In `pkg/morphology/builder.go`, append to the `AddForm` doc comment:

```go
// Forms are grouped into lexemes (paradigms) by lemma and part-of-speech
// class — the tag's first grammeme, with INFN/VERB/PRTF/PRTS/GRND,
// ADJF/ADJS/COMP and V/V.PTCP/V.CVB/V.MSDR each counted as one class — so
// «знать» NOUN and «знать» INFN become two lemmas. A form with an empty or
// POS-less tag joins the lemma's first lexeme.
```

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/... ./cmd/... -count=1`
Expected: PASS — including `TestBuilderTripleDedup` (кот NOUN + кот VERB still yields two
readings, now from two paradigms) and all merge tests.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/internal/build.go pkg/morphology/internal/build_test.go pkg/morphology/builder.go pkg/morphology/builder_test.go
git commit -m "fix(morphology): Builder keeps homonymous lemmas of different POS apart

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: `Dictionary.Forms` (item H, part 1)

**Files:**
- Create: `pkg/morphology/lexeme.go`
- Create: `pkg/morphology/lexeme_test.go`

**Interfaces:**
- Consumes: `x.paradigm`, `x.paradigmAffix`, `x.paradigmTag` (`parse.go:231-265`), `Reading.Predicted`.
- Produces: `func (x *Dictionary) Forms(r Reading) []Reading`.

Algorithm (paradigm layout `[suffix_i | tag_i | prefix_i]`, ids into `Suffixes[shard]`,
`TagSet`, `Prefixes`):
1. Resolve `para := Paradigms[r.Shard][r.Para]`; reject `r.Word == ""`, an unknown paradigm, or
   `r.Form >= para.Len()`.
2. `prefix, suffix := affixes of r.Form`; the stem is `r.Word` minus that prefix and suffix. If
   `r.Word` does not start with `prefix` or (after it) end with `suffix`, the reading does not
   belong to this dictionary → nil. (`readingForm` trims silently; `Forms` must not invent forms.)
3. Form `i` = `prefix_i + stem + suffix_i`, `Tag = tag_i`, `Form = i`; `Normal` = form 0's text
   (the same reconstruction as `readingForm`); `Para`, `Shard`, `Dict`, `Predicted` copied from
   `r`; `Prob = 0`.

Predicted readings resolve against shard 0 like `predictForPrefix` does, so they need no special
case — the predicted paradigm is expanded and `Predicted` stays true.

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/lexeme_test.go`:

```go
package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type wordTag struct{ Word, Tag string }

func wordTags(rs []morphology.Reading) []wordTag {
	out := make([]wordTag, len(rs))
	for i, r := range rs {
		out[i] = wordTag{r.Word, r.Tag}
	}
	return out
}

func TestFormsPyMorphy(t *testing.T) {
	d := parseDict(t)
	rs := d.Parse("кота")
	require.Len(t, rs, 1)

	forms := d.Forms(rs[0])
	assert.Equal(t, []wordTag{
		{"кот", "NOUN,anim,masc,sing,nomn"},
		{"кота", "NOUN,anim,masc,sing,gent"},
	}, wordTags(forms))
	for i, f := range forms {
		assert.Equal(t, uint16(i), f.Form)
		assert.Equal(t, "кот", f.Normal)
		assert.Equal(t, rs[0].Para, f.Para)
		assert.False(t, f.Predicted)
	}
}

// «кот» is a NOUN (paradigm 0, two forms) and a VERB (paradigm 1, one form).
func TestFormsFollowsTheGivenHomonym(t *testing.T) {
	d := parseDict(t)
	for _, r := range d.Parse("кот") {
		forms := d.Forms(r)
		switch r.Tag {
		case "VERB,impf,trans":
			assert.Equal(t, []wordTag{{"кот", "VERB,impf,trans"}}, wordTags(forms))
		default:
			assert.Len(t, forms, 2)
		}
	}
}

// prefixedFixture adds paradigm 3 = {form 0: stem, form 1: "наи"+stem}
// (prefix id 2 in paradigm-prefixes.json ["","по","наи"]).
func prefixedFixture(t *testing.T) *morphology.Dictionary {
	t.Helper()
	m := map[string]uint32{}
	stdWords(m)
	addWord(m, "лучший", 3, 0)
	addWord(m, "наилучший", 3, 1)
	dir := buildFixtureDir(t, m, nil, nil)
	writeParadigms(t, dir, [][]uint16{
		{0, 1, 0, 1, 0, 0},
		{0, 2, 0},
		{0, 1, 0, 1, 0, 0},
		{0, 0, 0, 1, 0, 2}, // suffixes 0,0 | tags 0,1 | prefixes 0,2
	})
	d, err := morphology.OpenPyMorphy(dir)
	require.NoError(t, err)
	return d
}

func TestFormsWithParadigmPrefix(t *testing.T) {
	d := prefixedFixture(t)
	rs := d.Parse("наилучший")
	require.Len(t, rs, 1)

	forms := d.Forms(rs[0])
	require.Len(t, forms, 2)
	assert.Equal(t, "лучший", forms[0].Word)
	assert.Equal(t, "наилучший", forms[1].Word)
	assert.Equal(t, "лучший", forms[1].Normal)
}

func TestFormsOfPredictedReading(t *testing.T) {
	d := predictionFixture(t)
	rs := d.Parse("котёнка")
	require.Len(t, rs, 1)
	require.True(t, rs[0].Predicted)

	forms := d.Forms(rs[0])
	assert.Equal(t, []wordTag{
		{"котёнк", "NOUN,anim,masc,sing,nomn"},
		{"котёнка", "NOUN,anim,masc,sing,gent"},
	}, wordTags(forms))
	for _, f := range forms {
		assert.True(t, f.Predicted)
	}
}

// After item J the verb lexeme of «знать» no longer contains «знати».
func TestFormsBuilderLexeme(t *testing.T) {
	d := buildFromTriples(t,
		[3]string{"знать", "знать", "NOUN,inan,femn,sing,nomn"},
		[3]string{"знати", "знать", "NOUN,inan,femn,sing,gent"},
		[3]string{"знать", "знать", "INFN,impf,tran"},
		[3]string{"знаю", "знать", "VERB,impf,tran,sing,1per,pres,indc"},
	)
	rs := d.Parse("знаю")
	require.Len(t, rs, 1)
	assert.Equal(t, []wordTag{
		{"знать", "INFN,impf,tran"},
		{"знаю", "VERB,impf,tran,sing,1per,pres,indc"},
	}, wordTags(d.Forms(rs[0])))
}

func TestFormsRejectsForeignReadings(t *testing.T) {
	d := parseDict(t)
	assert.Nil(t, d.Forms(morphology.Reading{}), "zero reading")
	assert.Nil(t, d.Forms(morphology.Reading{Word: "кот", Para: 99}), "unknown paradigm")
	assert.Nil(t, d.Forms(morphology.Reading{Word: "кот", Para: 0, Form: 5}), "form out of range")
	assert.Nil(t, d.Forms(morphology.Reading{Word: "кот", Para: 0, Form: 1}), "«кот» lacks form 1's suffix «а»")

	var nilDict *morphology.Dictionary
	assert.Nil(t, nilDict.Forms(d.Parse("кота")[0]))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ -run 'Forms' -count=1`
Expected: FAIL — `d.Forms undefined`.

- [ ] **Step 3: Implement**

Create `pkg/morphology/lexeme.go`:

```go
package morphology

import "strings"

// Forms returns every form of r's lexeme — the paradigm r.Para in r.Shard,
// with the stem derived from r.Word and r.Form — in paradigm order (form 0
// is the lemma). r must be a reading of this dictionary (from Parse,
// ParseAppend or Forms); a reading that does not fit its paradigm yields
// nil. Every returned Reading carries r's Para, Shard, Dict and Predicted,
// Normal = form 0, Prob = 0. For a predicted reading the forms are
// generated from the predicted paradigm and are as much a guess as the
// reading itself.
func (x *Dictionary) Forms(r Reading) []Reading {
	if x == nil || x.d == nil || r.Word == "" {
		return nil
	}
	para, ok := x.paradigm(r.Shard, r.Para)
	if !ok || int(r.Form) >= para.Len() {
		return nil
	}
	prefix, suffix := x.paradigmAffix(r.Shard, para, int(r.Form))
	if !strings.HasPrefix(r.Word, prefix) || !strings.HasSuffix(r.Word[len(prefix):], suffix) {
		return nil
	}
	stem := r.Word[len(prefix) : len(r.Word)-len(suffix)]

	p0, s0 := x.paradigmAffix(r.Shard, para, 0)
	normal := p0 + stem + s0
	out := make([]Reading, 0, para.Len())
	for i := 0; i < para.Len(); i++ {
		p, s := x.paradigmAffix(r.Shard, para, i)
		out = append(out, Reading{
			Word:      p + stem + s,
			Normal:    normal,
			Tag:       x.paradigmTag(para, i),
			Para:      r.Para,
			Form:      uint16(i),
			Shard:     r.Shard,
			Dict:      r.Dict,
			Predicted: r.Predicted,
		})
	}
	return out
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/lexeme.go pkg/morphology/lexeme_test.go
git commit -m "feat(morphology): Dictionary.Forms lists a reading's lexeme

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: `Dictionary.Inflect` (item H, part 2)

**Files:**
- Modify: `pkg/morphology/lexeme.go` (append)
- Modify: `pkg/morphology/lexeme_test.go` (append)

**Interfaces:**
- Consumes: `Forms` (Task 3), `Grammemes`, `HasGrammeme` (Task 1).
- Produces: `func (x *Dictionary) Inflect(r Reading, want ...string) []Reading`.

Ranking: a form qualifies when its tag contains every grammeme in `want` (`HasGrammeme`). Among
qualifying forms, fewer "extra differences from `r.Tag`" first, where the difference is the size of
the symmetric difference of the two grammeme sets; ties keep paradigm order (stable sort). With no
`want`, every form qualifies and `r`'s own form comes first (difference 0).

- [ ] **Step 1: Write the failing tests**

Append to `pkg/morphology/lexeme_test.go`:

```go
func koshkaDict(t *testing.T) *morphology.Dictionary {
	t.Helper()
	return buildFromTriples(t,
		[3]string{"кошка", "кошка", "NOUN,anim,femn,sing,nomn"},
		[3]string{"кошки", "кошка", "NOUN,anim,femn,sing,gent"},
		[3]string{"кошке", "кошка", "NOUN,anim,femn,sing,datv"},
		[3]string{"кошки", "кошка", "NOUN,anim,femn,plur,nomn"},
		[3]string{"кошек", "кошка", "NOUN,anim,femn,plur,gent"},
	)
}

func TestInflect(t *testing.T) {
	d := koshkaDict(t)
	rs := d.Parse("кошка")
	require.Len(t, rs, 1)
	r := rs[0]

	// sing,nomn → plur: plur,nomn differs in 2 grammemes (sing/plur),
	// plur,gent in 4.
	assert.Equal(t, []wordTag{
		{"кошки", "NOUN,anim,femn,plur,nomn"},
		{"кошек", "NOUN,anim,femn,plur,gent"},
	}, wordTags(d.Inflect(r, "plur")))

	assert.Equal(t, []wordTag{
		{"кошки", "NOUN,anim,femn,sing,gent"},
		{"кошек", "NOUN,anim,femn,plur,gent"},
	}, wordTags(d.Inflect(r, "gent")))

	assert.Equal(t, []wordTag{{"кошек", "NOUN,anim,femn,plur,gent"}},
		wordTags(d.Inflect(r, "plur", "gent")))
	assert.Equal(t, "кошке", d.Inflect(r, "datv")[0].Word)
	assert.Nil(t, d.Inflect(r, "ablt"), "no such form")

	all := d.Inflect(r)
	require.Len(t, all, 5)
	assert.Equal(t, "кошка", all[0].Word, "r itself ranks first")
}

func TestInflectFromNonLemmaForm(t *testing.T) {
	d := koshkaDict(t)
	rs := d.Parse("кошек")
	require.Len(t, rs, 1)
	got := d.Inflect(rs[0], "sing", "nomn")
	require.Len(t, got, 1)
	assert.Equal(t, "кошка", got[0].Word)
	assert.Equal(t, "кошка", got[0].Normal)
}

func TestInflectForeignReading(t *testing.T) {
	d := koshkaDict(t)
	assert.Nil(t, d.Inflect(morphology.Reading{}, "gent"))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ -run 'Inflect' -count=1`
Expected: FAIL — `d.Inflect undefined`.

- [ ] **Step 3: Implement**

Append to `pkg/morphology/lexeme.go` (add `"cmp"` and `"slices"` to its imports):

```go
// Inflect returns the forms of r's lexeme (see Forms) whose tags contain
// every grammeme in want, best first: fewest grammemes differing from
// r.Tag (size of the symmetric difference of the two grammeme sets), then
// paradigm order. With no want it returns every form, r's own first.
// Grammemes are native tokens (see Grammemes), e.g. "gent", "plur" or
// "GEN". nil when no form matches.
func (x *Dictionary) Inflect(r Reading, want ...string) []Reading {
	forms := x.Forms(r)
	if len(forms) == 0 {
		return nil
	}
	src := Grammemes(r.Tag)

	type candidate struct {
		reading Reading
		diff    int
	}
	var cands []candidate
	for _, f := range forms {
		if hasAllGrammemes(f.Tag, want) {
			cands = append(cands, candidate{f, grammemeDiff(src, Grammemes(f.Tag))})
		}
	}
	if len(cands) == 0 {
		return nil
	}
	slices.SortStableFunc(cands, func(a, b candidate) int { return cmp.Compare(a.diff, b.diff) })
	out := make([]Reading, len(cands))
	for i, c := range cands {
		out[i] = c.reading
	}
	return out
}

func hasAllGrammemes(tag string, want []string) bool {
	for _, g := range want {
		if !HasGrammeme(tag, g) {
			return false
		}
	}
	return true
}

// grammemeDiff is the size of the symmetric difference of two grammeme
// lists treated as sets.
func grammemeDiff(a, b []string) int {
	n := 0
	for _, g := range a {
		if !slices.Contains(b, g) {
			n++
		}
	}
	for _, g := range b {
		if !slices.Contains(a, g) {
			n++
		}
	}
	return n
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/lexeme.go pkg/morphology/lexeme_test.go
git commit -m "feat(morphology): Dictionary.Inflect picks forms by grammemes

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: `MultiDictionary.Forms`/`Inflect` and examples (item H, part 3)

**Files:**
- Modify: `pkg/morphology/lexeme.go` (append)
- Modify: `pkg/morphology/lexeme_test.go` (append)
- Modify: `pkg/morphology/example_test.go` (append)

**Interfaces:**
- Produces: `func (m *MultiDictionary) Forms(r Reading) []Reading`,
  `func (m *MultiDictionary) Inflect(r Reading, want ...string) []Reading` — dispatch by `r.Dict`.

- [ ] **Step 1: Write the failing tests**

Append to `pkg/morphology/lexeme_test.go`:

```go
// Dict 0 (кот/мышь) only predicts «кошки»; Dict 1 knows it (two readings:
// sing,gent and plur,nomn).
func TestMultiDictionaryFormsDispatchByDict(t *testing.T) {
	m := morphology.NewMultiDictionary(buildSmallDict(t), koshkaDict(t))
	var known []morphology.Reading
	for _, r := range m.Parse("кошки") {
		if r.Dict == 1 {
			known = append(known, r)
		}
	}
	require.Len(t, known, 2)
	for _, r := range known {
		forms := m.Forms(r)
		require.Len(t, forms, 5)
		for _, f := range forms {
			assert.Equal(t, 1, f.Dict)
		}
	}
	got := m.Inflect(known[0], "plur", "gent")
	require.Len(t, got, 1)
	assert.Equal(t, "кошек", got[0].Word)

	assert.Nil(t, m.Forms(morphology.Reading{Word: "кот", Dict: 7}), "unknown Dict")
	assert.Nil(t, m.Inflect(morphology.Reading{Word: "кот", Dict: -1}))
}
```

Append to `pkg/morphology/example_test.go`:

```go
// ExampleDictionary_Forms lists every form of a word's lexeme.
func ExampleDictionary_Forms() {
	d := mustCompileExampleDict()

	r := d.Parse("кота")[0]
	for _, f := range d.Forms(r) {
		fmt.Println(f.Word, f.Tag)
	}
	// Output:
	// кот NOUN,anim,masc,sing,nomn
	// кота NOUN,anim,masc,sing,gent
}

// ExampleDictionary_Inflect puts a word into another grammatical form.
func ExampleDictionary_Inflect() {
	d := mustCompileExampleDict()

	r := d.Parse("кот")[0]
	for _, f := range d.Inflect(r, "gent") {
		fmt.Println(f.Word)
	}
	// Output:
	// кота
}
```

(If `ExampleDictionary_Forms` fails only on line order, check the OpenCorpora importer's form order
for `exampleDictXML` and make the expected output match the importer — the example documents actual
behaviour; do not change the importer.)

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ -run 'MultiDictionaryForms|Example.*Forms|Example.*Inflect' -count=1`
Expected: FAIL — `m.Forms undefined`.

- [ ] **Step 3: Implement**

Append to `pkg/morphology/lexeme.go`:

```go
// Forms returns every form of r's lexeme from the dictionary r came from
// (r.Dict); nil if r.Dict is out of range. See Dictionary.Forms.
func (m *MultiDictionary) Forms(r Reading) []Reading {
	if r.Dict < 0 || r.Dict >= len(m.dicts) {
		return nil
	}
	return m.dicts[r.Dict].Forms(r)
}

// Inflect is Dictionary.Inflect on the dictionary r came from (r.Dict);
// nil if r.Dict is out of range.
func (m *MultiDictionary) Inflect(r Reading, want ...string) []Reading {
	if r.Dict < 0 || r.Dict >= len(m.dicts) {
		return nil
	}
	return m.dicts[r.Dict].Inflect(r, want...)
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/ -count=1`
Expected: PASS, both examples included.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/lexeme.go pkg/morphology/lexeme_test.go pkg/morphology/example_test.go
git commit -m "feat(morphology): MultiDictionary.Forms and Inflect; examples

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: 2-byte alphabet fallback for Builder/ImportTSV (item K)

**Files:**
- Modify: `pkg/morphology/internal/alphabet.go` (append `NewDenseAlphabetFor`)
- Modify: `pkg/morphology/internal/merge.go:514-556` (drop `denseAlphabetFor`, call the new function)
- Modify: `pkg/morphology/internal/dense_recompile.go:8-46`
- Modify: `pkg/morphology/internal/dense_recompile_test.go:104-119`
- Modify: `pkg/morphology/builder_internal_test.go` (append)

**Interfaces:**
- Produces: `func NewDenseAlphabetFor(corpus []string) (*DenseAlphabet, error)` — width 1 when the
  corpus has ≤ 254 distinct runes, else width 2 (≤ 64516), else error.

`RecompileDense` is also used by `CompileFromXMLDense`, `CompileFromUniMorphDense` and
`OpenPyMorphyDense`; they gain the same fallback (previously a hard error). The read path
(`SimilarItems`, `Fuzzy`, `maxWordRunesInShard`, `EncodeAlphabet`/`DecodeAlphabet`) already supports
width 2 — `Merge` has produced width-2 dictionaries since 1.1.0.

- [ ] **Step 1: Write the failing tests**

In `pkg/morphology/internal/dense_recompile_test.go` replace
`TestRecompileDenseAlphabetOverflowReturnsError` with:

```go
// 255 distinct runes exceed width 1's capacity of 254: RecompileDense
// falls back to a width-2 alphabet instead of failing.
func TestRecompileDenseFallsBackToWidth2(t *testing.T) {
	var words []string
	for r := rune(0x400); r < 0x400+255; r++ {
		words = append(words, string(r))
	}
	values := make([]uint32, len(words))
	for i := range values {
		values[i] = uint32(i)
	}
	d := buildTestDictionary(t, [][]string{words}, [][]uint32{values})

	require.NoError(t, RecompileDense(d))
	require.NotNil(t, d.Alphabet)
	assert.Equal(t, 2, d.Alphabet.Width())
	for i, w := range words {
		items := d.Words[0].SimilarItems(w, nil, d.Alphabet)
		require.Len(t, items, 1, w)
		require.Len(t, items[0].Values, 1)
		assert.Equal(t, uint32(i), binary.BigEndian.Uint32(items[0].Values[0]))
	}
}

func TestNewDenseAlphabetForWidths(t *testing.T) {
	a, err := NewDenseAlphabetFor([]string{"кот", "мышь"})
	require.NoError(t, err)
	assert.Equal(t, 1, a.Width())

	var many []string
	for r := rune(0x10000); r < 0x10000+255; r++ {
		many = append(many, string(r))
	}
	a, err = NewDenseAlphabetFor(many)
	require.NoError(t, err)
	assert.Equal(t, 2, a.Width())

	for r := rune(0x10000 + 255); r < 0x10000+254*254+1; r++ {
		many = append(many, string(r))
	}
	_, err = NewDenseAlphabetFor(many)
	assert.Error(t, err, "64517 distinct runes exceed width 2")
}
```

Append to `pkg/morphology/builder_internal_test.go` (add `"path/filepath"` to its imports):

```go
// TestBuilderWideAlphabetFallback builds a dictionary over 300 distinct
// runes — more than a 1-byte dense alphabet holds — and checks it works end
// to end: 2-byte alphabet, Parse, Fuzzy, SaveTo/Open.
func TestBuilderWideAlphabetFallback(t *testing.T) {
	b := NewBuilder(BuilderOptions{Language: "zh"})
	var words []string
	for i := 0; i < 300; i++ {
		w := string(rune(0x4E00 + i))
		words = append(words, w)
		require.NoError(t, b.AddLemma(w, "X"))
	}
	d, err := b.Build()
	require.NoError(t, err)
	require.NotNil(t, d.d.Alphabet)
	assert.Equal(t, 2, d.d.Alphabet.Width())

	path := filepath.Join(t.TempDir(), "wide.dat")
	require.NoError(t, d.SaveTo(path))
	opened, err := Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, opened.Close()) }()

	for _, dict := range []*Dictionary{d, opened} {
		for _, w := range []string{words[0], words[150], words[299]} {
			rs := dict.Parse(w)
			require.Len(t, rs, 1, w)
			assert.False(t, rs[0].Predicted)
			assert.Equal(t, []FuzzyMatch{{Word: w}}, dict.Fuzzy(w, 0))
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ ./pkg/morphology/internal/ -run 'Width2|DenseAlphabetFor|WideAlphabet' -count=1`
Expected: FAIL — `undefined: NewDenseAlphabetFor`; after adding it, `RecompileDense` still errors
with `corpus has 255 distinct runes, exceeds 1-byte alphabet's capacity of 254`.

- [ ] **Step 3: Implement**

Append to `pkg/morphology/internal/alphabet.go`:

```go
// NewDenseAlphabetFor builds the narrowest DenseAlphabet for corpus: width
// 1 when it has at most 254 distinct runes, width 2 otherwise (at most
// 64516). Returns NewDenseAlphabet's error when even width 2 is too narrow.
func NewDenseAlphabetFor(corpus []string) (*DenseAlphabet, error) {
	if a, err := NewDenseAlphabet(1, corpus); err == nil {
		return a, nil
	}
	return NewDenseAlphabet(2, corpus)
}
```

In `pkg/morphology/internal/merge.go` delete `denseAlphabetFor` and change both call sites to
`NewDenseAlphabetFor(...)`.

In `pkg/morphology/internal/dense_recompile.go` change the doc's first sentence to `RecompileDense
rebuilds every shard of d.Words under one dense alphabet shared across the whole dictionary — 1 byte
per rune, or 2 bytes when the dictionary has more than 254 distinct runes (NewDenseAlphabetFor) —`
(keep the rest), and replace the alphabet construction with:

```go
	alphabet, err := NewDenseAlphabetFor(allWords)
	if err != nil {
		return fmt.Errorf("internal: recompile dense: build alphabet: %w", err)
	}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/... -count=1`
Expected: PASS (merge tests confirm the rename is behaviour-neutral).

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/internal/alphabet.go pkg/morphology/internal/merge.go pkg/morphology/internal/dense_recompile.go pkg/morphology/internal/dense_recompile_test.go pkg/morphology/builder_internal_test.go
git commit -m "fix(morphology): dense recompile falls back to a 2-byte alphabet

Builder/ImportTSV (and the *Dense importers) no longer fail on more
than 254 distinct runes.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: Benchmarks and baseline (item I, part 1)

**Files:**
- Create: `pkg/morphology/bench_test.go`

**Interfaces:**
- Produces: `BenchmarkParseKnown`, `BenchmarkParseKnownYo`, `BenchmarkParsePredicted`,
  `BenchmarkLemma`, `BenchmarkIsKnown`, `BenchmarkFuzzyTop`, `BenchmarkRealDict` (skipped unless
  `GOMORPHY_BENCH_DICT` points to a `.dat`). Task 9 adds `BenchmarkParseAppend`.

This task changes no production code: it records the "before" numbers.

- [ ] **Step 1: Write the benchmarks**

Create `pkg/morphology/bench_test.go`:

```go
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
```

- [ ] **Step 2: Run the fixture benchmarks and record the baseline**

Run:
```bash
go test ./pkg/morphology/ -run '^$' -bench . -benchmem -count=5 | tee /tmp/gomorphy-bench-before.txt
```
Expected: every benchmark reports `ns/op`, `B/op`, `allocs/op`; `BenchmarkRealDict` is SKIPped.
`BenchmarkParseKnown` allocs/op is in the low twenties (22–24 measured at planning time on a
smaller dictionary).

- [ ] **Step 3: Run the real-dictionary benchmarks**

The owner's compiled dictionaries live in the main checkout. Read them in place — do not copy,
modify or rebuild them:

```bash
GOMORPHY_BENCH_DICT=/Users/asmarin/dev/mine/gomorphy/.data/pymorphy/pymorphy.dat \
  go test ./pkg/morphology/ -run '^$' -bench RealDict -benchmem -count=5 | tee /tmp/gomorphy-bench-real-before.txt
GOMORPHY_BENCH_DICT=/Users/asmarin/dev/mine/gomorphy/.data/opencorpora/opencorpora.dat \
  go test ./pkg/morphology/ -run '^$' -bench 'RealDict/(ParseKnown|Lemma|FuzzyTop)' -benchmem -count=5 | tee -a /tmp/gomorphy-bench-real-before.txt
```
Expected: numbers for each sub-benchmark (OpenCorpora has no prediction, so its `ParsePredicted`
sub-benchmarks would `Fatal` with "no readings" — hence the filter). If the files are absent, note
it in the task report and continue with fixture numbers only.

Keep both `/tmp/gomorphy-bench-*-before.txt` files: Task 10 copies the medians into the write-up.

- [ ] **Step 4: Commit**

```bash
git add pkg/morphology/bench_test.go
git commit -m "test(morphology): Parse/Lemma/IsKnown/FuzzyTop benchmarks

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: Allocation-free lookup primitives in `internal` (item I, part 2)

**Files:**
- Modify: `pkg/morphology/internal/alphabet.go` (append `EncodeRune`)
- Modify: `pkg/morphology/internal/similar_items.go:21-35` (`followRuneVia`)
- Create: `pkg/morphology/internal/lookup_each.go`
- Create: `pkg/morphology/internal/lookup_each_test.go`
- Create: `pkg/morphology/internal/race_on_test.go`, `pkg/morphology/internal/race_off_test.go`
  (the same two files as Task 1, with `package internal`: `const raceEnabled = true` under
  `//go:build race`, `false` under `//go:build !race`)

**Interfaces:**
- Produces:
  - `func (a *DenseAlphabet) EncodeRune(r rune) (code [2]byte, n int, ok bool)` — the spec's
    "EncodeRune API on the alphabet". It returns the code by value; an interface method taking a
    `dst []byte` would make the buffer escape through the dynamic call. See Discrepancy D-15.
  - `func (d *DAWG) LookupEach(key string, pol *CharPolicy, alphabet Alphabet, fn func(found string, value []byte))`
    — `SimilarItems` without intermediate slices: for every matching stored key, in **exactly
    `SimilarItems`' order** (the straight match first, then substitution branches by position,
    recursively), `fn` is called once per payload value (the same order as `ValuesForIndex`).
    `found` is `key` itself (no allocation) when no substitution was taken. `value` points into a
    scratch buffer and is valid only during the call. Values whose base64 fails to decode are
    skipped (`ValuesForIndex` returns a nil entry for them, which every caller then drops by its
    length guard).
  - `func (d *DAWG) FindJoined(a string, sep byte, b string) uint32` — `Find(a + string(sep) + b)`
    without building the string.

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/internal/lookup_each_test.go`:

```go
package internal

import (
	"bytes"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type flatItem struct {
	key   string
	value []byte
}

func flattenSimilar(items []Item) []flatItem {
	var out []flatItem
	for _, it := range items {
		for _, v := range it.Values {
			if v != nil {
				out = append(out, flatItem{it.Key, v})
			}
		}
	}
	return out
}

func collectLookup(d *DAWG, key string, pol *CharPolicy, a Alphabet) []flatItem {
	var out []flatItem
	d.LookupEach(key, pol, a, func(found string, value []byte) {
		out = append(out, flatItem{found, bytes.Clone(value)})
	})
	return out
}

// lookupCorpus: several е/ё variants of one word, multi-value keys and
// 6-byte (prediction-shaped) values.
func lookupCorpus() ([]string, []uint32) {
	keys := []string{"ежик", "ёжик", "ежик", "ёжиек", "еле", "ёлё", "елё", "кот", "кота", "кот"}
	values := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	return keys, values
}

func TestLookupEachMatchesSimilarItems(t *testing.T) {
	keys, values := lookupCorpus()
	raw, err := BuildDAWGWithValues(keys, values)
	require.NoError(t, err)

	alphabet, err := NewDenseAlphabetFor(keys)
	require.NoError(t, err)
	encoded := make([]string, len(keys))
	for i, k := range keys {
		enc, err := alphabet.Encode(k)
		require.NoError(t, err)
		encoded[i] = string(enc)
	}
	dense, err := BuildDAWGWithValues(encoded, values)
	require.NoError(t, err)

	for _, pol := range []*CharPolicy{nil, NewCharPolicy(), RussianCharPolicy()} {
		for _, q := range []string{"ежик", "ёжик", "еле", "кот", "кота", "мышь", "", "е"} {
			want := flattenSimilar(raw.SimilarItems(q, pol, nil))
			assert.Equal(t, want, collectLookup(raw, q, pol, nil), "raw %q", q)

			wantDense := flattenSimilar(dense.SimilarItems(q, pol, alphabet))
			assert.Equal(t, wantDense, collectLookup(dense, q, pol, alphabet), "dense %q", q)
		}
	}
}

func TestLookupEachSixByteValues(t *testing.T) {
	d, err := BuildDAWGWithValuesBytes(
		[]string{"ёнка", "ёнка"},
		[][]byte{{0, 3, 0, 0, 0, 1}, {0, 1, 0, 2, 0, 0}},
	)
	require.NoError(t, err)
	want := flattenSimilar(d.SimilarItems("енка", RussianCharPolicy(), nil))
	require.Len(t, want, 2)
	assert.Equal(t, want, collectLookup(d, "енка", RussianCharPolicy(), nil))
}

func TestLookupEachFoundIsKeyWithoutSubstitution(t *testing.T) {
	d, err := BuildDAWGWithValues([]string{"кот"}, []uint32{1})
	require.NoError(t, err)
	key := "кот"
	d.LookupEach(key, RussianCharPolicy(), nil, func(found string, _ []byte) {
		assert.Equal(t, key, found)
	})
	if raceEnabled {
		return
	}
	allocs := testing.AllocsPerRun(100, func() {
		d.LookupEach(key, RussianCharPolicy(), nil, func(string, []byte) {})
	})
	assert.Equal(t, 0.0, allocs)
}

func TestFindJoined(t *testing.T) {
	d, err := BuildIntDAWG([]string{"кот:NOUN", "кот:VERB", "кота:NOUN"}, []uint32{500, 100, 7})
	require.NoError(t, err)
	for _, tc := range []struct{ a, b string }{
		{"кот", "NOUN"}, {"кот", "VERB"}, {"кота", "NOUN"}, {"кот", "ADJF"}, {"мышь", "NOUN"}, {"", "NOUN"},
	} {
		assert.Equal(t, d.Find(tc.a+":"+tc.b), d.FindJoined(tc.a, ':', tc.b), "%s:%s", tc.a, tc.b)
	}
	if !raceEnabled {
		assert.Equal(t, 0.0, testing.AllocsPerRun(100, func() { _ = d.FindJoined("кот", ':', "NOUN") }))
	}
}

func TestEncodeRuneMatchesEncode(t *testing.T) {
	var many []string
	for r := rune(0x400); r < 0x400+300; r++ {
		many = append(many, string(r))
	}
	for _, corpus := range [][]string{{"кот", "ёж"}, many} {
		a, err := NewDenseAlphabetFor(corpus)
		require.NoError(t, err)
		for _, w := range corpus {
			for _, r := range w {
				want, err := a.Encode(string(r))
				require.NoError(t, err)
				code, n, ok := a.EncodeRune(r)
				require.True(t, ok)
				assert.Equal(t, want, code[:n])
			}
		}
		_, _, ok := a.EncodeRune('ﬀ')
		assert.False(t, ok)
	}
}

func TestLookupEachUsesTestdawgFixtures(t *testing.T) {
	// The same fixture shape SimilarItems tests use (testdawg.Build).
	dict, guide := testdawg.Build(map[string]uint32{
		"ежик" + string(PayloadSeparator) + b64([]byte{0, 1, 0, 2}): 1,
		"ёжик" + string(PayloadSeparator) + b64([]byte{3, 4, 5, 6}): 2,
	})
	d := NewDAWG(dict, guide)
	assert.Equal(t, flattenSimilar(d.SimilarItems("ежик", RussianCharPolicy(), nil)),
		collectLookup(d, "ежик", RussianCharPolicy(), nil))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/internal/ -run 'LookupEach|FindJoined|EncodeRune' -count=1`
Expected: FAIL — `d.LookupEach undefined`, `a.EncodeRune undefined`, `d.FindJoined undefined`.

- [ ] **Step 3: Implement**

Append to `pkg/morphology/internal/alphabet.go`:

```go
// EncodeRune returns r's code (its first n bytes; n == width) without
// allocating. ok is false when r is not in the alphabet. Encode(string(r))
// returns the same bytes.
func (a *DenseAlphabet) EncodeRune(r rune) (code [2]byte, n int, ok bool) {
	c, found := a.codeOf[r]
	if !found {
		return code, 0, false
	}
	if a.width == 1 {
		code[0] = byte(c)
		return code, 1, true
	}
	k := c - 2 // base-254 two-digit encoding, see Encode
	code[0], code[1] = byte(2+k/254), byte(2+k%254)
	return code, 2, true
}
```

Replace `followRuneVia` in `pkg/morphology/internal/similar_items.go`:

```go
// followRuneVia follows one rune r from index, encoding it via alphabet.
// alphabet == nil follows raw UTF-8 (FollowRune). A *DenseAlphabet is
// encoded without allocating (EncodeRune). Returns 0 if alphabet cannot
// encode r — the same "no edge" contract as FollowByte/FollowRune.
func (d *DAWG) followRuneVia(alphabet Alphabet, r rune, index uint32) uint32 {
	switch a := alphabet.(type) {
	case nil:
		return d.FollowRune(r, index)
	case *DenseAlphabet:
		code, n, ok := a.EncodeRune(r)
		if !ok {
			return 0
		}
		for i := 0; i < n; i++ {
			if index = d.FollowByte(code[i], index); index == 0 {
				return 0
			}
		}
		return index
	default:
		code, err := alphabet.Encode(string(r))
		if err != nil {
			return 0
		}
		return d.followBytes(code, index)
	}
}
```

Create `pkg/morphology/internal/lookup_each.go`:

```go
package internal

import (
	"encoding/base64"
	"unicode/utf8"
)

// LookupEach is SimilarItems without the intermediate []Item: for every
// stored key matching key under pol (CharPolicy substitutions), in exactly
// SimilarItems' order, fn is called once per payload value, in
// ValuesForIndex order. found is the stored key's text — key itself (no
// allocation) when no substitution was taken. value is decoded into a
// scratch buffer and is valid only during the call; undecodable values are
// skipped.
func (d *DAWG) LookupEach(key string, pol *CharPolicy, alphabet Alphabet, fn func(found string, value []byte)) {
	d.lookupFrom(key, 0, 0, "", false, pol, alphabet, fn)
}

// lookupFrom matches key[pos:] from DAWG node index. head is the matched
// text for key[:pos] when substituted is true (a substitution was taken
// earlier on this path); otherwise the matched text is key[:pos] itself.
// It mirrors similarItemsRecursive: the straight path (no further
// substitutions) is reported first, then every substitution branch in
// position order.
func (d *DAWG) lookupFrom(key string, pos int, index uint32, head string, substituted bool,
	pol *CharPolicy, alphabet Alphabet, fn func(string, []byte)) {
	// Pass 1: the straight path. Like similarItemsRecursive, an exhausted
	// key looks for the payload edge right at index (the root for "").
	end, ok := index, true
	if pos < len(key) {
		end = d.followStringVia(alphabet, key[pos:], index)
		ok = end != 0
	}
	if ok {
		if sep := d.FollowByte(PayloadSeparator, end); sep != 0 {
			found := key
			if substituted {
				found = head + key[pos:]
			}
			d.forEachValue(sep, func(v []byte) { fn(found, v) })
		}
	}
	if pol == nil || len(pol.Substitutions) == 0 {
		return
	}
	// Pass 2: branch at every substitutable rune along the straight path.
	for i := pos; i < len(key); {
		r, size := utf8.DecodeRuneInString(key[i:])
		for _, s := range pol.Substitutions {
			if s.From != r {
				continue
			}
			if next := d.followRuneVia(alphabet, s.To, index); next != 0 {
				prefix := key[:i]
				if substituted {
					prefix = head + key[pos:i]
				}
				d.lookupFrom(key, i+size, next, prefix+string(s.To), true, pol, alphabet, fn)
			}
		}
		if index = d.followRuneVia(alphabet, r, index); index == 0 {
			return
		}
		i += size
	}
}

// followStringVia follows every rune of s from index (see followRuneVia);
// 0 when the path breaks.
func (d *DAWG) followStringVia(alphabet Alphabet, s string, index uint32) uint32 {
	for _, r := range s {
		if index = d.followRuneVia(alphabet, r, index); index == 0 {
			return 0
		}
	}
	return index
}

// forEachValue calls fn with every payload value under the
// PayloadSeparator node sep, in ValuesForIndex order, base64-decoded into
// a stack buffer. The completer's key and index stack also start on the
// stack; they spill to the heap only for unusually long payloads.
func (d *DAWG) forEachValue(sep uint32, fn func(value []byte)) {
	if len(d.guide) == 0 {
		return
	}
	var keyBuf [24]byte
	var stackBuf [24]uint32
	c := completer{dawg: d, key: keyBuf[:0], indexStack: append(stackBuf[:0], sep)}
	var out [18]byte
	for c.next() {
		dst := out[:]
		if n := base64.StdEncoding.DecodedLen(len(c.key)); n > len(dst) {
			dst = make([]byte, n)
		}
		n, err := base64.StdEncoding.Decode(dst, c.key)
		if err != nil {
			continue
		}
		fn(dst[:n])
	}
}

// FindJoined looks up the key a + string(sep) + b exactly, like Find, but
// without building the concatenated string.
func (d *DAWG) FindJoined(a string, sep byte, b string) uint32 {
	index := d.Follow(a, 0)
	if index == 0 {
		return 0
	}
	if index = d.FollowByte(sep, index); index == 0 {
		return 0
	}
	if index = d.Follow(b, index); index == 0 {
		return 0
	}
	return d.Value(index)
}
```

Note on pass 1: node index 0 is both the root and `FollowByte`'s "no edge" result, so an empty
remaining key must not go through `followStringVia` (it could not tell "stayed at the root" from
"failed"). The `ok` flag keeps the two apart; the equivalence test's `""` query locks this in.

- [ ] **Step 4: Run the tests; check escapes**

Run: `go test ./pkg/morphology/internal/ -count=1 -race`
Expected: PASS (the `AllocsPerRun` assertions are skipped under `-race` via `raceEnabled`).

Run: `go test ./pkg/morphology/internal/ -run 'LookupEach|FindJoined' -count=1`
Expected: PASS, including both zero-allocation assertions.

Run: `go build -gcflags=-m ./pkg/morphology/internal/ 2>&1 | grep -E 'lookup_each.go.*(escapes to heap|moved to heap)'`
Expected: no line for `keyBuf`, `stackBuf`, `out` or `c`. If one appears, restructure (e.g. make
`completer.next`'s receiver usage non-escaping) before moving on.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/internal/alphabet.go pkg/morphology/internal/similar_items.go pkg/morphology/internal/lookup_each.go pkg/morphology/internal/lookup_each_test.go pkg/morphology/internal/race_on_test.go pkg/morphology/internal/race_off_test.go
git commit -m "perf(morphology): allocation-free DAWG lookup primitives

DenseAlphabet.EncodeRune, DAWG.LookupEach (SimilarItems order, stack
buffers), DAWG.FindJoined.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 9: `ParseAppend`, single-shard fast path, fewer allocations (item I, part 3)

**Files:**
- Modify: `pkg/morphology/parse.go` (rewrite `Parse`, `exact`, `exactInShard`, `predict`,
  `predictForPrefix`, `suffixSplits`)
- Create: `pkg/morphology/parse_alloc_test.go`
- Modify: `pkg/morphology/bench_test.go` (append `BenchmarkParseAppend`)
- Modify: `pkg/morphology/open.go` (`Close` doc: add `ParseAppend` to the list, see 1.2.0 Task 11)

**Interfaces:**
- Consumes: `LookupEach`, `FindJoined` (Task 8).
- Produces: `func (x *Dictionary) ParseAppend(dst []Reading, word string) []Reading`.

- [ ] **Step 1: Write the failing tests**

(`raceEnabled` comes from `race_on_test.go`/`race_off_test.go`, created in Task 1.)

Create `pkg/morphology/parse_alloc_test.go`:

```go
package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAppendMatchesParse(t *testing.T) {
	for name, d := range map[string]*morphology.Dictionary{
		"builder":    buildSmallDict(t),
		"pymorphy":   predictionFixture(t),
		"twoShards":  buildTwoShardDict(t),
	} {
		for _, w := range []string{"кот", "КОТА", "мыши", "ежик", "бота", "котёнка", "слово000010", "слово065535", "неттакого"} {
			want := d.Parse(w)
			prefix := []morphology.Reading{{Word: "sentinel"}}
			got := d.ParseAppend(prefix, w)
			require.Equal(t, "sentinel", got[0].Word, "%s/%s: dst prefix kept", name, w)
			assert.Equal(t, want, nilIfEmpty(got[1:]), "%s/%s", name, w)
		}
	}
	var nilDict *morphology.Dictionary
	assert.Nil(t, nilDict.ParseAppend(nil, "кот"))
}

func nilIfEmpty(rs []morphology.Reading) []morphology.Reading {
	if len(rs) == 0 {
		return nil
	}
	return rs
}

// TestParseAllocs bounds allocations on a single-shard dense Builder
// dictionary. Planning-time baseline (1.1.0, Parse): кот 22, кота 24,
// бота (predicted) 44. When the measured value is lower than the bound,
// tighten the bound to it (+1 slack) and note both numbers in the
// implementation write-up.
func TestParseAllocs(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts differ under -race")
	}
	d := buildSmallDict(t)
	buf := make([]morphology.Reading, 0, 16)

	for _, tc := range []struct {
		word      string
		appendMax float64 // ParseAppend with a reused buffer
		parseMax  float64 // Parse (allocates the result slice)
	}{
		{"кот", 4, 5},   // form 0: Normal is the word itself
		{"кота", 5, 6},  // form 1: Normal = prefix0+stem+suffix0 (one concat)
		{"бота", 16, 17}, // predicted: Word and Normal concats, seen map
	} {
		require.NotEmpty(t, d.ParseAppend(buf[:0], tc.word))
		got := testing.AllocsPerRun(200, func() { buf = d.ParseAppend(buf[:0], tc.word) })
		t.Logf("ParseAppend(%q): %v allocs", tc.word, got)
		assert.LessOrEqual(t, got, tc.appendMax, "ParseAppend(%q)", tc.word)

		got = testing.AllocsPerRun(200, func() { _ = d.Parse(tc.word) })
		t.Logf("Parse(%q): %v allocs", tc.word, got)
		assert.LessOrEqual(t, got, tc.parseMax, "Parse(%q)", tc.word)
	}
}
```

Append to `pkg/morphology/bench_test.go`:

```go
func BenchmarkParseAppend(b *testing.B) {
	d := benchDict(b)
	buf := make([]morphology.Reading, 0, 16)
	b.ReportAllocs()
	for b.Loop() {
		buf = d.ParseAppend(buf[:0], "кота")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ -run 'ParseAppend|ParseAllocs' -count=1`
Expected: FAIL — `d.ParseAppend undefined`.

- [ ] **Step 3: Implement**

In `pkg/morphology/parse.go`: imports become `"cmp"`, `"encoding/binary"`, `"slices"`,
`"strings"`, `"sync"`, `"unicode/utf8"`, internal (drop `"sort"`). Replace everything from `Parse`
through `predictForPrefix`, and `suffixSplits`, with:

```go
// Parse parses word and returns all dictionary readings, sorted by
// probability (descending). For out-of-dictionary words, it tries to
// predict readings from the prediction-DAWG (suffixes); such readings have
// Predicted set. Returns nil if no readings are found. The input is
// lowercased.
func (x *Dictionary) Parse(word string) []Reading {
	out := x.ParseAppend(nil, word)
	if len(out) == 0 {
		return nil
	}
	return out
}

// ParseAppend is Parse that appends the readings to dst and returns the
// extended slice (dst unchanged when there are none). Reusing dst across
// calls avoids allocating the result slice; the appended readings are
// sorted by probability among themselves.
func (x *Dictionary) ParseAppend(dst []Reading, word string) []Reading {
	if x == nil || x.d == nil || len(x.d.Words) == 0 {
		return dst
	}
	word = strings.ToLower(word)
	n := len(dst)
	if dst = x.exactAppend(dst, word); len(dst) > n {
		return dst
	}
	return x.predictAppend(dst, word)
}

// shardExactResult — the result of exactInShard for a single shard.
type shardExactResult struct {
	readings []Reading
	hasProb  bool
}

// exactAppend appends the readings of a word found in the dictionary,
// accounting for CharPolicy substitutions (е→ё), sorted by probability.
// A single-shard dictionary is searched inline; several shards are
// queried in parallel (one goroutine per shard) and concatenated in shard
// order.
func (x *Dictionary) exactAppend(dst []Reading, word string) []Reading {
	start := len(dst)
	hasProb := false
	if len(x.d.Words) == 1 {
		dst, hasProb = x.exactInShard(dst, 0, x.d.Words[0], word)
	} else {
		results := make([]shardExactResult, len(x.d.Words))
		var wg sync.WaitGroup
		for shard, dawg := range x.d.Words {
			wg.Add(1)
			go func(shard int, dawg *internal.DAWG) {
				defer wg.Done()
				rs, hp := x.exactInShard(nil, shard, dawg, word)
				results[shard] = shardExactResult{readings: rs, hasProb: hp}
			}(shard, dawg)
		}
		wg.Wait()
		for _, r := range results {
			dst = append(dst, r.readings...)
			hasProb = hasProb || r.hasProb
		}
	}
	if hasProb {
		slices.SortStableFunc(dst[start:], func(a, b Reading) int { return cmp.Compare(b.Prob, a.Prob) })
	}
	return dst
}

// exactInShard appends a word's readings from a single shard. Read-only:
// safe to run in parallel with other shards.
func (x *Dictionary) exactInShard(dst []Reading, shard int, dawg *internal.DAWG, word string) ([]Reading, bool) {
	hasProb := false
	dawg.LookupEach(word, x.d.CharPolicy, x.d.Alphabet, func(found string, v []byte) {
		r, ok := x.reading(shard, found, v)
		if !ok {
			return
		}
		if x.d.Probability != nil {
			r.Prob = float64(x.d.Probability.FindJoined(found, ':', r.Tag)) / 1e6
			if r.Prob > 0 {
				hasProb = true
			}
		}
		dst = append(dst, r)
	})
	return dst, hasProb
}

// predictMaxSuffix is the longest word ending (in runes) looked up in the
// prediction-DAWG (internal.predictionMaxSuffix).
const predictMaxSuffix = 5

// readingKey dedups predicted readings by (Word, Normal, Tag).
type readingKey struct{ word, normal, tag string }

// predictAppend appends readings for an out-of-dictionary word predicted
// from its ending (pymorphy2's KnownSuffixAnalyzer, as in opennota/morph).
func (x *Dictionary) predictAppend(dst []Reading, word string) []Reading {
	if len(x.d.Prediction) == 0 {
		return dst
	}
	var splitBuf [predictMaxSuffix]int
	splits := suffixSplits(splitBuf[:0], word, predictMaxSuffix)
	if len(splits) == 0 {
		return dst
	}
	seen := make(map[readingKey]bool)
	for id, pref := range x.d.Prefixes {
		if id >= len(x.d.Prediction) || x.d.Prediction[id] == nil {
			continue
		}
		if !strings.HasPrefix(word, pref) {
			continue
		}
		dst = x.predictForPrefix(dst, id, word, splits, seen)
	}
	return dst
}

// predictForPrefix predicts readings against a single prefix's
// prediction-DAWG (x.d.Prediction[id]), widening from the longest suffix
// split toward shorter ones until at least 2 total matches accumulate —
// pymorphy2's KnownSuffixAnalyzer heuristic. splits are byte offsets into
// word (see suffixSplits). seen dedups (word, lemma, tag) across all
// prefixes tried by the caller and is mutated in place.
//
// Predictions always resolve against shard 0: prediction DAWGs are only
// built for unsharded dictionaries (pymorphy2 imports, Builder/ImportTSV).
func (x *Dictionary) predictForPrefix(dst []Reading, id int, word string, splits []int, seen map[readingKey]bool) []Reading {
	const predictionShard = 0
	totalCount := 0

	for i := len(splits) - 1; i >= 0; i-- {
		wordStart, wordEnd := word[:splits[i]], word[splits[i]:]
		// Prediction DAWGs are never recompiled under Dictionary.Alphabet
		// (see docs/en/superpowers/specs/2026-09-16-pymorphy2-dense-recompile-design.md's
		// non-goals): the alphabet is always nil here.
		x.d.Prediction[id].LookupEach(wordEnd, x.d.CharPolicy, nil, func(found string, v []byte) {
			if len(v) < 6 {
				return
			}
			count := int(binary.BigEndian.Uint16(v[:2]))
			paraNum := binary.BigEndian.Uint16(v[2:4])
			form := binary.BigEndian.Uint16(v[4:6])

			para, ok := x.paradigm(predictionShard, paraNum)
			if !ok || form >= uint16(para.Len()) {
				return
			}
			if !productive(x.paradigmTag(para, int(form))) {
				return
			}
			totalCount += count

			r := x.readingForm(predictionShard, wordStart+found, paraNum, form)
			r.Predicted = true
			k := readingKey{r.Word, r.Normal, r.Tag}
			if seen[k] {
				return
			}
			seen[k] = true
			dst = append(dst, r)
		})
		if totalCount > 1 {
			break
		}
	}
	return dst
}
```

and replace `suffixSplits` with:

```go
// suffixSplits appends to dst the byte offsets that split word into
// (start, end) with end = the last 1, 2, …, max runes (shortest end
// first). No allocation beyond dst's growth.
func suffixSplits(dst []int, word string, max int) []int {
	i := len(word)
	for n := 0; n < max && i > 0; n++ {
		_, size := utf8.DecodeLastRuneInString(word[:i])
		i -= size
		dst = append(dst, i)
	}
	return dst
}
```

Keep `reading`, `readingForm`, `paradigm`, `paradigmAffix`, `paradigmTag`, `strAt`, `productive`,
`nonproductiveGrammemes` unchanged.

Check nothing else calls the removed functions:
`grep -rn 'x\.exact(\|x\.predict(\|suffixSplits(' pkg/morphology/*.go` — Expected: only the new
definitions/uses in `parse.go`.

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/... ./cmd/... -count=1 -race`
Expected: PASS — in particular `TestParseAppendMatchesParse`, all `Parse`/`Lemma`/merge/save
snapshot tests, `TestShardedDictionaryPublicAPIRoundtrip` and the concurrent `shard_test.go` tests.

Run: `go test ./pkg/morphology/ -run TestParseAllocs -count=1 -v`
Expected: PASS; the log lines show the measured counts. Tighten the bounds in
`parse_alloc_test.go` to measured+1 if they are lower, and keep the numbers for Task 10.

If a bound is exceeded, find the escaping values with
`go build -gcflags=-m ./pkg/morphology/ 2>&1 | grep -E 'parse.go.*(escapes to heap|moved to heap)'`
(typical suspects: the `LookupEach` closure capturing `dst`/`hasProb`). Fix the escape; only if it
cannot be fixed, raise the bound and explain why in the write-up.

If `-gcflags=-m` shows the reading closure itself escapes because `LookupEach`'s `fn` is passed
through the recursive `lookupFrom`, that costs one allocation per call and is acceptable within the
bounds above.

- [ ] **Step 5: Benchmarks after**

Run:
```bash
go test ./pkg/morphology/ -run '^$' -bench . -benchmem -count=5 | tee /tmp/gomorphy-bench-after.txt
GOMORPHY_BENCH_DICT=/Users/asmarin/dev/mine/gomorphy/.data/pymorphy/pymorphy.dat \
  go test ./pkg/morphology/ -run '^$' -bench RealDict -benchmem -count=5 | tee /tmp/gomorphy-bench-real-after.txt
```
Expected: `BenchmarkParseKnown`/`ParsePredicted` allocs/op well below the Task 7 baseline, ns/op
not worse. If `benchstat` is installed, compare:
`benchstat /tmp/gomorphy-bench-before.txt /tmp/gomorphy-bench-after.txt`.

- [ ] **Step 6: Commit**

```bash
git add pkg/morphology/parse.go pkg/morphology/parse_alloc_test.go pkg/morphology/bench_test.go pkg/morphology/open.go
git commit -m "perf(morphology): ParseAppend; Parse without per-shard goroutines and per-rune allocations

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 10: Documentation, CHANGELOG, version 1.3.0

**Files:**
- Modify: `pkg/morphology/version.go:7`, `CHANGELOG.md`
- Modify: `docs/en/library.md`, `docs/ru/library.md`
- Modify: `docs/en/todo.md` (Forms/Inflect section; Completed stages row)
- Modify: `docs/en/implementation.md:190-201` (Metrics table)
- Modify: `docs/en/implementation/ner-support.md` (append the 1.3.0 part)

- [ ] **Step 1: Version** — `const Version = "1.3.0"`.

- [ ] **Step 2: CHANGELOG** — under `## [Unreleased]` (date from `date +%F` on release day):

```markdown
## [1.3.0] - <release date, YYYY-MM-DD>

### Added
- Tag helpers `Grammemes`, `HasGrammeme` (allocation-free), `POS`,
  `Reading.HasGrammeme` — native tokens split on `,`, space and `;`.
- Lexeme access: `Dictionary.Forms` / `Inflect` and the `MultiDictionary`
  equivalents (dispatch by `Reading.Dict`). Forms of predicted readings keep
  `Predicted`.
- `Dictionary.ParseAppend(dst, word)`.
- Benchmarks for Parse/Lemma/IsKnown/FuzzyTop (fixture and a real `.dat` via
  `GOMORPHY_BENCH_DICT`).

### Changed
- `Parse`: no goroutine for single-shard dictionaries, no per-rune/per-value
  allocations; allocations per known word <N> (was 22–24), per predicted word
  <M> (was 44) — see implementation/ner-support.md.
- Builder/ImportTSV group forms into lexemes by lemma **and** part-of-speech
  class: «знать» NOUN and «знать» INFN become two lemmas. Dictionaries rebuilt
  from homonymous input differ from 1.2.0 output; existing files are unaffected.

### Fixed
- Builder/ImportTSV (and the `*Dense` importers) fall back to a 2-byte alphabet
  instead of failing on more than 254 distinct characters.
```

Replace `<N>`/`<M>` with the measured `TestParseAllocs` values from Task 9 Step 4.

- [ ] **Step 3: `docs/en/library.md`** — new sections: "Tags" (`Grammemes`, `HasGrammeme`, `POS`,
  with OpenCorpora and UniMorph examples; note that `Surn`/`Name`/`Patr`/`Geox` are native tokens
  and not mapped by `tagmap`); "Word forms" (`Forms`, `Inflect`, the ranking rule, predicted
  readings); in "Exact wordform lookup" add `ParseAppend`; in "Building a dictionary from scratch"
  add the POS-class grouping rule and the 2-byte fallback; in "Multiple dictionaries" add
  `Forms`/`Inflect`.

- [ ] **Step 4: `docs/ru/library.md`** — minimal Russian subset: `Grammemes`/`HasGrammeme`/`POS`,
  `Forms`/`Inflect` (one example each), `ParseAppend`, one sentence on the POS-class grouping.

- [ ] **Step 5: `docs/en/todo.md`** — replace the body of "Word inflection / form generation
  (`Forms`, `Inflect`) — PLANNED" with a short "DONE in 1.3.0" note: the shape shipped
  (`Inflect(r, want ...string)`, variadic, native tokens; forms of predicted readings allowed and
  flagged), the ranking rule, link to `implementation/ner-support.md`; keep open item (1)'s
  tagmap-based universal query as a v2 idea. Add a Completed-stages row:

```markdown
| — NER support for lexicon: 1.3.0 (tag helpers, Forms/Inflect, Parse performance, POS-aware Builder, 2-byte alphabet fallback) | DONE | [implementation/ner-support.md](implementation/ner-support.md) |
```

- [ ] **Step 6: `docs/en/implementation.md` Metrics table** — replace the `Parse (exact)`,
  `Parse (prediction)`, `Lemmas` rows' values with the measured medians from
  `/tmp/gomorphy-bench-real-after.txt` (pymorphy.dat; OpenCorpora for exact where available), each
  with "(measured 1.3.0, <CPU>)", and link the write-up.

- [ ] **Step 7: Write-up** — append to `docs/en/implementation/ner-support.md` a "1.3.0" part: one
  paragraph per item G–K; a "Performance" table (benchmark, before, after: ns/op, B/op, allocs/op,
  fixture and real dictionary, CPU from `go test` output header); the `TestParseAllocs` numbers;
  whether the spec targets (Parse < 10 µs, prediction < 50 µs) hold; the Discrepancies D-12…D-18
  below.

- [ ] **Step 8: Full verification**

Run:
```bash
gofmt -l . && go build ./... && go vet ./... && go test ./... -race -count=1 && golangci-lint run ./...
```
Expected: no output from `gofmt -l`; everything else PASS / clean.

Self-review against the spec's Testing section: G → `tag_test.go`, `internal/tag_test.go`;
H → `lexeme_test.go` + examples; I → `bench_test.go`, `parse_alloc_test.go`,
`internal/lookup_each_test.go`; J → `builder_test.go`, `internal/build_test.go`;
K → `dense_recompile_test.go`, `builder_internal_test.go`.

- [ ] **Step 9: Commit**

```bash
git add pkg/morphology/version.go CHANGELOG.md docs/en/library.md docs/ru/library.md docs/en/todo.md docs/en/implementation.md docs/en/implementation/ner-support.md
git commit -m "release: 1.3.0

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

## Discrepancies with the spec (found while planning)

- **D-12 (J).** The spec's test case «стекло» (NOUN) / «стекло» (VERB, past neut of «стечь») does
  not exercise the change: the verb form's lemma is «стечь», not «стекло», so the Builder already
  puts it into a different paradigm. The plan tests «знать» (NOUN «знать» vs INFN «знать»), which
  really shares the lemma text.
- **D-13 (J).** "Group by `(lemma, POS(tag of the lemma form))`": an entry carries only its own
  tag, and grouping by the raw first grammeme would split every OpenCorpora verb into
  INFN/VERB/PRTF/PRTS/GRND paradigms and adjectives into ADJF/ADJS/COMP. The plan groups by a
  **POS class** (`internal.POSClass`, with a fold table for OpenCorpora and UniMorph) and defines
  what POS-less entries do.
- **D-14 (H).** `docs/en/todo.md` sketched `Inflect(r Reading, want []string)`; the spec says
  `want ...string`. The plan follows the spec.
- **D-15 (I).** "`EncodeRune` API on the alphabet": adding an interface method that writes into a
  caller buffer would make the buffer escape (dynamic call); the plan adds a concrete
  `(*DenseAlphabet).EncodeRune(r) ([2]byte, int, bool)` and a type switch in `followRuneVia`.
- **D-16 (I).** A naive callback version of `SimilarItems` changes result order (substitution
  branches before the straight match). `Parse` results must not change, so `LookupEach` walks the
  straight path first, then the branches — at the cost of following the straight path twice.
- **D-17 (I).** Benchmarks use `b.Loop()` (Go 1.24; allowed by the `go 1.25.0` directive from
  1.2.0, owner decision 2026-09-24, answer 15 — the earlier 1.22 plan forbade it).
  `AllocsPerRun` bounds are still skipped under `-race` (the race detector changes allocation
  counts); `b.Loop` does not change that.
- **D-18 (K).** `RecompileDense` is shared with `CompileFromXMLDense`, `CompileFromUniMorphDense`
  and `OpenPyMorphyDense`; the fallback therefore also changes them (from an error to a working
  2-byte dictionary). The spec only mentions Builder/ImportTSV. The existing test
  `TestRecompileDenseAlphabetOverflowReturnsError` asserts the old error and is rewritten.

## Open questions

- **Q1 (J).** RESOLVED 2026-09-24 (owner, answer 17): POS-class grouping is **on by default with
  no option** — no `BuilderOptions` field. Task 2 implements exactly that; the CHANGELOG lists it
  under "Changed".
- **Q2 (J).** Is the POS-class fold table right and complete? It assumes OpenCorpora-style POS
  tokens (INFN/VERB/PRTF/PRTS/GRND, ADJF/ADJS/COMP) and UniMorph Russian with the POS **first** in
  the bundle (`V.PTCP;…`). UniMorph does not guarantee feature order; a bundle with the POS in the
  middle would group wrongly. Alternatives: look up the POS token anywhere in the bundle via a
  known-POS set, or let the caller pass the class.
- **Q3 (H).** `Inflect` ranking by symmetric grammeme difference, ties by paradigm order — fine for
  lexicon, or should it prefer forms that keep the source's number/gender/animacy explicitly (a
  weighted metric)?
- **Q4 (H).** Should `Forms`/`Inflect` fill `Prob` for dictionaries with probabilities? The plan
  sets 0 (probabilities are p(tag|word) for the parsed word, not for generated forms).
- **Q5 (I).** Allocation bounds (`ParseAppend`: кот ≤ 4, кота ≤ 5, predicted ≤ 16) are estimates
  from reading the code; Task 9 tightens them to the measured values. Acceptable, or is there a
  hard target (e.g. 0 allocations for a known word with a reused buffer, which would need a
  non-closure visitor API in `internal`)?
- **Q6 (I).** `MultiDictionary.Parse` still spawns one goroutine per dictionary. Add a
  `MultiDictionary.ParseAppend` and a sequential path for small sets? Not in the spec; not planned.
- **Q7 (G).** `productive()` (`parse.go`) still splits tags on `,` only, so a pymorphy2 tag such as
  `"NOUN,anim,masc,Surn sing,ablt"` yields the token `"Surn sing"`. It does not affect the
  nonproductive list today, but switching it to `internal.NextGrammeme` would be consistent. Do
  it here or leave it?
- **Q8 (spec G1–G3).** RESOLVED 2026-09-24 (owner, answer 15): G1 — single module, `cmd/gomorphy`
  stays; G3 — `go 1.25.0` + `toolchain go1.27.1` by the policy "current Go minus two minor
  versions" (this plan relies on it for `b.Loop`). G2 stays as in the 1.2.0 plan (moot, D-9).
