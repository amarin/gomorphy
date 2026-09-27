# NER support, release 1.3.0 (items G–K) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Quality-of-life additions that the NER work in lexicon relies on, all natural extensions
of the core: tag helpers `Grammemes`/`HasGrammeme`/`POS` (G), lexeme access `Forms`/`Inflect` (H),
`Parse` performance with `ParseAppend` and benchmarks (I), homonymous Builder lemmas of different
parts of speech kept apart (J), a 2-byte alphabet fallback for Builder/ImportTSV (K), and
ending-based prediction for sharded dictionaries — OpenCorpora and UniMorph imports, sharded `Merge`
results (L, added 2026-09-27). Ship as 1.3.0.

**Architecture:** G adds a zero-allocation grammeme tokenizer in `internal` (so the builder can use
it too) and thin public wrappers. J changes only phase 1 (grouping) of
`internal.BuildDictionaryFromEntries`. H reads the existing paradigm table
(`[suffix_0..N-1 | tag_0..N-1 | prefix_0..N-1]`, `internal/paradigm.go`) exactly the way
`readingForm` already does, just for every form instead of form 0. K moves `Merge`'s
width-1-then-2 alphabet choice into `internal/alphabet.go` and uses it in `RecompileDense`. I first
adds benchmarks and records a baseline, then adds allocation-free lookup primitives in `internal`
(`DenseAlphabet.EncodeRune`, `DAWG.LookupEach`, `DAWG.FindJoined`) and rebuilds `Parse` on them
behind a new `ParseAppend`. L generalizes `internal.BuildPrediction` to N shards (one prefix-0
DAWG whose 8-byte values carry the shard), stores it in new `pred-sharded-N` sections, resolves
the shard in `predictForPrefix` (on top of I's rewrite) and builds it by default in
`CompileFromXML*`/`CompileFromUniMorph*`. The binary format changes only additively (L).

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

- The GMOR binary format changes only additively (spec L): new `pred-sharded-N` sections
  with 8-byte `count|para|form|shard` values for dictionaries with more than one shard. Single-shard
  files (pymorphy2, Builder, ImportTSV) stay byte-identical; existing `.dat` files open and parse
  unchanged; gomorphy 1.2.x opens new files without prediction instead of failing. A file never
  holds both `prediction-N` and `pred-sharded-N`.
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
- **`Parse` results must not change** in content or order for any existing dictionary file
  (L adds predicted readings only for newly built OpenCorpora/UniMorph dictionaries): every existing test
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
| `pkg/morphology/internal/dictionary.go` | modify | `PredictionSharded` |
| `pkg/morphology/internal/prediction.go`, `prediction_test.go` | modify | N-shard `BuildPrediction`/`BuildPredictionFrom` |
| `pkg/morphology/internal/merge.go`, `merge_test.go`, `pkg/morphology/merge.go`, `cmd/gomorphy/merge.go` | modify | sharded `RebuildPrediction`, `ErrPredictionSharded` deprecated |
| `pkg/morphology/save.go` | modify | `pred-sharded-N` sections |
| `pkg/morphology/prediction_sharded_test.go` | create | sharded lookup, save/open tests |
| `pkg/morphology/open.go` | modify | `predictionSections`; `XMLOptions`, `CompileFromXMLWithOptions`, `finishCompiled` |
| `pkg/morphology/importers/unimorph/import.go` | modify | `Options.NoPrediction` |
| `pkg/morphology/compile_prediction_test.go` | create | import prediction tests, real-dictionary check |
| `cmd/gomorphy/build.go`, `update.go`, `build_test.go`, `dict_test.go` | modify | `--no-prediction` |
| `docs/en/scenarios.md`, `docs/en/comparison.md` | modify | docs (L) |
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

Predicted readings carry the shard their paradigm lives in (shard 0 until Task 11, which lets
sharded prediction set it), so they need no special case — the predicted paradigm of `r.Shard` is
expanded and `Predicted` stays true.

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

### Task 10: Sharded prediction builder and `Merge` (item L, part 1)

**Files:**
- Modify: `pkg/morphology/internal/dictionary.go` (field `PredictionSharded`)
- Modify: `pkg/morphology/internal/prediction.go` (`BuildPrediction`, `BuildPredictionFrom`)
- Modify: `pkg/morphology/internal/prediction_test.go`
- Modify: `pkg/morphology/internal/merge.go` (`MergeOptions` doc, `ErrPredictionSharded` removed,
  prediction rebuild over all shards)
- Modify: `pkg/morphology/internal/merge_test.go`
- Modify: `pkg/morphology/merge.go` (`ErrPredictionSharded` defined here, deprecated; docs)
- Modify: `cmd/gomorphy/merge.go:92,98` (help text)

**Interfaces:**
- Consumes: `shardPairs(w *DAWG, a Alphabet, removed map[string]bool) ([]WordValue, error)`
  (`internal/merge.go`), `BuildDAWGWithValuesBytes`.
- Produces:
  - `internal.Dictionary.PredictionSharded bool` — `true` when every `Prediction` value is 8 bytes
    `count|para|form|shard`, `false` for 6-byte `count|para|form` (shard 0).
  - `func BuildPrediction(d *Dictionary, productive func(tag string) bool) error` — now for any
    shard count, raw or dense.
  - `func BuildPredictionFrom(pairs [][]WordValue, paradigms [][]Paradigm, tagSet *TagSet, productive func(tag string) bool) (pred *DAWG, sharded bool, err error)`
    — `pairs[s]` are shard `s`'s readings; `sharded == len(pairs) > 1`.
  - `morphology.ErrPredictionSharded` — kept, never returned.

Value layout (spec L): key = the last 1..5 runes of a wordform; value = `count(BE16) | para(BE16) |
form(BE16)` for one shard, `count(BE16) | para(BE16) | form(BE16) | shard(BE16)` for more. The
format follows `PredictionSharded`, never the shard count alone: `Merge` can carry a 6-byte
prediction into a multi-shard result, where it still means shard 0.

- [ ] **Step 1: Write the failing tests**

In `pkg/morphology/internal/prediction_test.go`, delete `TestBuildPredictionNoOpWhenSharded`,
replace `TestBuildPredictionFromMatchesBuildPrediction` and append the rest:

```go
func TestBuildPredictionFromMatchesBuildPrediction(t *testing.T) {
	all := func(string) bool { return true }
	d, err := BuildDictionaryFromEntries(BuildOptions{}, []BuildEntry{
		{Word: "кот", Lemma: "кот", Tag: "NOUN,nomn"},
		{Word: "кота", Lemma: "кот", Tag: "NOUN,gent"},
		{Word: "мышь", Lemma: "мышь", Tag: "NOUN,nomn"},
		{Word: "мыши", Lemma: "мышь", Tag: "NOUN,gent"},
	})
	require.NoError(t, err)

	var pairs []WordValue
	d.Words[0].Walk(func(w string, vals [][]byte) {
		for _, v := range vals {
			pairs = append(pairs, WordValue{Word: w, Value: binary.BigEndian.Uint32(v[:4])})
		}
	})
	pred, sharded, err := BuildPredictionFrom([][]WordValue{pairs}, d.Paradigms, d.TagSet, all)
	require.NoError(t, err)
	assert.False(t, sharded, "one shard keeps 6-byte values")

	require.NoError(t, BuildPrediction(d, all))
	assert.False(t, d.PredictionSharded)
	assert.Equal(t, d.Prediction[0].Bytes(), pred.Bytes())
}

// twoShardCorpus is a raw two-shard dictionary sharing one TagSet: shard 0
// holds кошка/кошки, shard 1 окно/окна. Each shard's only lemma is its
// paradigm 0, so only the shard number tells their predictions apart.
func twoShardCorpus(t *testing.T) *Dictionary {
	t.Helper()
	a, err := BuildDictionaryFromEntries(BuildOptions{}, []BuildEntry{
		{Word: "кошка", Lemma: "кошка", Tag: "NOUN,sing,nomn"},
		{Word: "кошки", Lemma: "кошка", Tag: "NOUN,sing,gent"},
	})
	require.NoError(t, err)
	b, err := BuildDictionaryFromEntries(BuildOptions{}, []BuildEntry{
		{Word: "окно", Lemma: "окно", Tag: "NOUN,sing,nomn"},
		{Word: "окна", Lemma: "окно", Tag: "NOUN,sing,gent"},
	})
	require.NoError(t, err)
	// b's tag ids must mean the same tags in a's TagSet (same insertion order).
	for f := 0; f < 2; f++ {
		require.Equal(t, b.TagSet.TagName(b.Paradigms[0][0].Tag(f)), a.TagSet.TagName(b.Paradigms[0][0].Tag(f)))
	}
	return NewDictionary("ru", a.TagSet,
		[][]string{a.Suffixes[0], b.Suffixes[0]}, a.Prefixes,
		[][]Paradigm{a.Paradigms[0], b.Paradigms[0]},
		[]*DAWG{a.Words[0], b.Words[0]}, RussianCharPolicy())
}

type shardedPredValue struct{ Count, Para, Form, Shard uint16 }

// shardedPredValues returns the decoded 8-byte payloads for an exact
// prediction suffix key (empty when the key is absent).
func shardedPredValues(t *testing.T, pred *DAWG, key string) []shardedPredValue {
	t.Helper()
	var out []shardedPredValue
	for _, it := range pred.SimilarItems(key, nil, nil) {
		if it.Key != key {
			continue
		}
		for _, v := range it.Values {
			require.Len(t, v, 8)
			out = append(out, shardedPredValue{
				Count: binary.BigEndian.Uint16(v[0:2]),
				Para:  binary.BigEndian.Uint16(v[2:4]),
				Form:  binary.BigEndian.Uint16(v[4:6]),
				Shard: binary.BigEndian.Uint16(v[6:8]),
			})
		}
	}
	return out
}

func TestBuildPredictionSharded(t *testing.T) {
	d := twoShardCorpus(t)
	require.NoError(t, BuildPrediction(d, predProductive))

	require.Len(t, d.Prediction, 1, "one DAWG for prefix 0, shared by all shards")
	assert.True(t, d.PredictionSharded)
	assert.Equal(t, []shardedPredValue{{Count: 1, Para: 0, Form: 1, Shard: 1}},
		shardedPredValues(t, d.Prediction[0], "на"), "«на» comes only from окна (shard 1)")
	assert.Equal(t, []shardedPredValue{{Count: 1, Para: 0, Form: 1, Shard: 0}},
		shardedPredValues(t, d.Prediction[0], "ки"), "«ки» comes only from кошки (shard 0)")
	assert.ElementsMatch(t, []shardedPredValue{
		{Count: 1, Para: 0, Form: 0, Shard: 0}, // кошка
		{Count: 1, Para: 0, Form: 1, Shard: 1}, // окна
	}, shardedPredValues(t, d.Prediction[0], "а"))
}

func TestBuildPredictionShardedDenseMatchesRaw(t *testing.T) {
	raw := twoShardCorpus(t)
	require.NoError(t, BuildPrediction(raw, predProductive))

	dense := twoShardCorpus(t)
	require.NoError(t, RecompileDense(dense))
	require.NotNil(t, dense.Alphabet)
	require.NoError(t, BuildPrediction(dense, predProductive))

	assert.True(t, dense.PredictionSharded)
	assert.Equal(t, raw.Prediction[0].Bytes(), dense.Prediction[0].Bytes(),
		"keys are decoded through the alphabet, so a dense dictionary predicts the same")
}
```

In `pkg/morphology/internal/merge_test.go`, in `TestMergeDictionariesPrediction` add
`assert.False(t, out.PredictionSharded)` after the first `require.Len(t, out.Prediction, 1)`, and
replace its last two lines (the `ErrPredictionSharded` case) with:

```go
	out, err = MergeDictionaries(twoShardBase(t), []*Dictionary{over}, MergeOptions{Mode: MergeAdd, RebuildPrediction: true, Productive: allProductive})
	require.NoError(t, err, "a sharded result rebuilds prediction too")
	require.Len(t, out.Prediction, 1)
	assert.True(t, out.PredictionSharded)
	assert.NotEmpty(t, out.Prediction[0].SimilarItems("ок", nil, nil), "rebuilt sharded prediction covers overlay words")
```

and append:

```go
func TestMergeDictionariesCarriesShardedPrediction(t *testing.T) {
	base := twoShardBase(t)
	require.NoError(t, BuildPrediction(base, allProductive))
	require.True(t, base.PredictionSharded)

	out, err := MergeDictionaries(base, []*Dictionary{engineDict(t, e("шок", "шок", "N"))}, MergeOptions{Mode: MergeAdd})
	require.NoError(t, err)
	assert.True(t, out.PredictionSharded, "the carried prediction keeps its value format")
	require.Len(t, out.Prediction, 1)
	assert.Equal(t, base.Prediction[0].Bytes(), out.Prediction[0].Bytes())
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/internal/ -run 'Prediction' -count=1`
Expected: FAIL to compile — `BuildPredictionFrom` returns 2 values, `PredictionSharded` undefined.

- [ ] **Step 3: Implement**

`pkg/morphology/internal/dictionary.go` — add after `Prediction`:

```go
	Prediction  []*DAWG
	// PredictionSharded is the value format of Prediction: true for 8-byte
	// count|para|form|shard values (dictionaries with more than one shard,
	// saved as pred-sharded-N), false for 6-byte count|para|form
	// values resolved against shard 0 (saved as prediction-N).
	PredictionSharded bool
```

`pkg/morphology/internal/prediction.go` — replace `BuildPrediction` and `BuildPredictionFrom`:

```go
// BuildPrediction rebuilds d.Prediction (one DAWG, prefix id 0) from the
// words of every shard of d and sets d.PredictionSharded. Keys are decoded
// through d.Alphabet, so d may be raw or dense. A dictionary with no shards
// is left untouched.
func BuildPrediction(d *Dictionary, productive func(tag string) bool) error {
	if d == nil || len(d.Words) == 0 {
		return nil
	}
	pairs := make([][]WordValue, len(d.Words))
	for s, w := range d.Words {
		p, err := shardPairs(w, d.Alphabet, nil)
		if err != nil {
			return fmt.Errorf("prediction: shard %d: %w", s, err)
		}
		pairs[s] = p
	}
	pred, sharded, err := BuildPredictionFrom(pairs, d.Paradigms, d.TagSet, productive)
	if err != nil {
		return err
	}
	d.Prediction = []*DAWG{pred}
	d.PredictionSharded = sharded
	return nil
}

// BuildPredictionFrom builds the pymorphy2 KnownSuffixAnalyzer prediction
// DAWG for prefix id 0. pairs[s] are shard s's raw (word, value) readings,
// resolved against paradigms[s] and tagSet. For every reading whose tag is
// productive, the word's last 1..5 runes become suffix keys; readings
// sharing a (suffix, paradigm, form, shard) key accumulate a count. With
// one shard each key becomes count(BE16) + para(BE16) + form(BE16); with
// more, shard(BE16) is appended and sharded is true.
func BuildPredictionFrom(pairs [][]WordValue, paradigms [][]Paradigm, tagSet *TagSet, productive func(tag string) bool) (*DAWG, bool, error) {
	type predKey struct {
		suffix            string
		para, form, shard uint16
	}
	sharded := len(pairs) > 1
	counts := make(map[predKey]int)
	for s, shard := range pairs {
		if s >= len(paradigms) {
			break
		}
		ps := paradigms[s]
		for _, p := range shard {
			para, form := uint16(p.Value>>16), uint16(p.Value)
			if int(para) >= len(ps) || int(form) >= ps[para].Len() {
				continue
			}
			tag := ""
			if tagSet != nil {
				tag = tagSet.TagName(ps[para].Tag(int(form)))
			}
			if !productive(tag) {
				continue
			}
			rr := []rune(p.Word)
			max := min(predictionMaxSuffix, len(rr))
			for l := 1; l <= max; l++ {
				counts[predKey{suffix: string(rr[len(rr)-l:]), para: para, form: form, shard: uint16(s)}]++
			}
		}
	}

	width := 6
	if sharded {
		width = 8
	}
	keys := make([]string, 0, len(counts))
	values := make([][]byte, 0, len(counts))
	for k, count := range counts {
		count = min(count, predictionMaxCount)
		buf := make([]byte, width)
		binary.BigEndian.PutUint16(buf[0:2], uint16(count))
		binary.BigEndian.PutUint16(buf[2:4], k.para)
		binary.BigEndian.PutUint16(buf[4:6], k.form)
		if sharded {
			binary.BigEndian.PutUint16(buf[6:8], k.shard)
		}
		keys = append(keys, k.suffix)
		values = append(values, buf)
	}
	pred, err := BuildDAWGWithValuesBytes(keys, values)
	return pred, sharded, err
}
```

Add `"fmt"` to the file's imports.

`pkg/morphology/internal/merge.go`:
- `MergeOptions.RebuildPrediction` doc: `// RebuildPrediction replaces the base's prediction with one prefix-0
  DAWG rebuilt (BuildPredictionFrom) from every shard of the merged words. Requires Productive.`
- Delete `ErrPredictionSharded` and the `len(m.shards) != 1` check (keep the `Productive == nil`
  check).
- Delete `shard0` and `shard0Collected` and the `if i == 0 { … }` block in the shard loop.
- Replace the whole `if opts.RebuildPrediction { … } else { … }` block after the loop with:

```go
	if opts.RebuildPrediction {
		pairs := make([][]WordValue, len(out.Words))
		for i, w := range out.Words {
			if pairs[i], err = shardPairs(w, out.Alphabet, nil); err != nil {
				return nil, fmt.Errorf("merge: prediction: shard %d: %w", i, err)
			}
		}
		pred, sharded, err := BuildPredictionFrom(pairs, out.Paradigms, out.TagSet, opts.Productive)
		if err != nil {
			return nil, fmt.Errorf("merge: prediction: %w", err)
		}
		out.Prediction = []*DAWG{pred}
		out.PredictionSharded = sharded
	} else {
		for _, p := range base.Prediction {
			out.Prediction = append(out.Prediction, p.Clone())
		}
		out.PredictionSharded = base.PredictionSharded
	}
```

Re-reading the merged words through `shardPairs` walks unchanged shards once more; it only runs
with `RebuildPrediction` and removes the "shard 0 reused verbatim" special case.

`pkg/morphology/merge.go`:

```go
// ErrPredictionSharded was returned by MergeWithOptions when
// RebuildPrediction met a result with more than one shard.
//
// Deprecated: since 1.3.0 prediction is rebuilt for any shard count and
// this error is never returned.
var ErrPredictionSharded = errors.New("morphology: prediction rebuild needs a single-shard output")
```

In `MergeOptions.RebuildPrediction` replace `Requires a single-shard result (ErrPredictionSharded).`
with `Works for any number of shards.`; in the `MergeWithOptions` doc remove
`ErrPredictionSharded, ` from the "Errors:" list.

`cmd/gomorphy/merge.go`: in `Long` replace `(unless --rebuild-prediction is set; single-shard output
only).` with `(unless --rebuild-prediction is set).`; in the flag help replace
`(default: keep the base's; single-shard output only)` with `(default: keep the base's)`.

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/... ./cmd/... -count=1`
Expected: PASS. `TestMergeDictionariesZeroShardBaseRebuildPrediction` must stay green (zero
shards → an empty prediction DAWG, `sharded == false`). Builder/ImportTSV snapshot tests stay
green: one shard produces byte-identical 6-byte values.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/internal/dictionary.go pkg/morphology/internal/prediction.go pkg/morphology/internal/prediction_test.go pkg/morphology/internal/merge.go pkg/morphology/internal/merge_test.go pkg/morphology/merge.go cmd/gomorphy/merge.go
git commit -m "feat(morphology): build ending-based prediction for sharded dictionaries

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 11: Sharded prediction in lookup, `SaveTo` and `Open` (item L, part 2)

**Files:**
- Modify: `pkg/morphology/parse.go` (`predictForPrefix` as rewritten by Task 9)
- Modify: `pkg/morphology/open.go` (`parseContainer`, new `predictionSections`)
- Modify: `pkg/morphology/save.go` (`sections`, `SaveTo` doc)
- Create: `pkg/morphology/prediction_sharded_test.go`

**Interfaces:**
- Consumes: `internal.Dictionary.PredictionSharded`, `internal.BuildPrediction` (Task 10);
  `Dictionary.Forms` (Task 3); `predictForPrefix(dst []Reading, id int, word string, splits []int, seen map[readingKey]bool) []Reading` (Task 9).
- Produces: section name `pred-sharded-N` (N = prefix id); `Reading.Shard` of predicted
  readings taken from the value.

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/prediction_sharded_test.go`:

```go
package morphology

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// shardedPredictionDict is a dense two-shard dictionary with sharded
// prediction: shard 0 holds кошка/кошки, shard 1 окно/окна, each as its
// shard's paradigm 0 with the same tags.
func shardedPredictionDict(t *testing.T) *Dictionary {
	t.Helper()
	build := func(entries ...internal.BuildEntry) *internal.Dictionary {
		d, err := internal.BuildDictionaryFromEntries(internal.BuildOptions{
			Language: "ru", CharPolicy: internal.RussianCharPolicy(), TagSetName: "tsv",
		}, entries)
		require.NoError(t, err)
		return d
	}
	a := build(
		internal.BuildEntry{Word: "кошка", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,nomn"},
		internal.BuildEntry{Word: "кошки", Lemma: "кошка", Tag: "NOUN,anim,femn,sing,gent"},
	)
	b := build(
		internal.BuildEntry{Word: "окно", Lemma: "окно", Tag: "NOUN,anim,femn,sing,nomn"},
		internal.BuildEntry{Word: "окна", Lemma: "окно", Tag: "NOUN,anim,femn,sing,gent"},
	)
	d := internal.NewDictionary("ru", a.TagSet,
		[][]string{a.Suffixes[0], b.Suffixes[0]}, a.Prefixes,
		[][]internal.Paradigm{a.Paradigms[0], b.Paradigms[0]},
		[]*internal.DAWG{a.Words[0], b.Words[0]}, internal.RussianCharPolicy())
	require.NoError(t, internal.BuildPrediction(d, productive))
	require.True(t, d.PredictionSharded)
	require.NoError(t, internal.RecompileDense(d))
	return &Dictionary{d: d}
}

// predictedFrom reports whether Parse(word) has a predicted reading with
// this shard and lemma, returning it.
func predictedFrom(d *Dictionary, word string, shard int, normal string) (Reading, bool) {
	for _, r := range d.Parse(word) {
		if r.Predicted && r.Shard == shard && r.Normal == normal {
			return r, true
		}
	}
	return Reading{}, false
}

func TestParsePredictsAcrossShards(t *testing.T) {
	d := shardedPredictionDict(t)

	r, ok := predictedFrom(d, "бревна", 1, "бревно")
	require.True(t, ok, "бревна is predicted from окна (shard 1): %+v", d.Parse("бревна"))
	assert.Equal(t, "NOUN,anim,femn,sing,gent", r.Tag)

	_, ok = predictedFrom(d, "мошки", 0, "мошка")
	assert.True(t, ok, "мошки is predicted from кошки (shard 0): %+v", d.Parse("мошки"))

	var words []string
	for _, f := range d.Forms(r) {
		words = append(words, f.Word)
		assert.True(t, f.Predicted)
		assert.Equal(t, 1, f.Shard)
	}
	assert.Equal(t, []string{"бревно", "бревна"}, words, "Forms of a shard-1 predicted reading")
}

func TestParseSkipsPredictionForMissingShard(t *testing.T) {
	d := shardedPredictionDict(t)
	v := make([]byte, 8)
	binary.BigEndian.PutUint16(v[0:2], 1) // count
	binary.BigEndian.PutUint16(v[2:4], 0) // para
	binary.BigEndian.PutUint16(v[4:6], 1) // form
	binary.BigEndian.PutUint16(v[6:8], 5) // shard 5 does not exist
	pred, err := internal.BuildDAWGWithValuesBytes([]string{"на"}, [][]byte{v})
	require.NoError(t, err)
	d.d.Prediction = []*internal.DAWG{pred}

	assert.Empty(t, d.Parse("бревна"))
}

func TestShardedPredictionSaveOpen(t *testing.T) {
	d := shardedPredictionDict(t)
	path := filepath.Join(t.TempDir(), "sharded.dat")
	require.NoError(t, d.SaveTo(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	cont, err := internal.OpenContainer(data)
	require.NoError(t, err)
	_, _, err = cont.Section("pred-sharded-0")
	require.NoError(t, err)
	_, _, err = cont.Section("prediction-0")
	require.Error(t, err, "gomorphy 1.2.x reads only prediction-N and must find none")

	fromFile, err := Open(path)
	require.NoError(t, err)
	defer func() { _ = fromFile.Close() }()
	fromBytes, err := OpenBytes(data)
	require.NoError(t, err)
	for name, o := range map[string]*Dictionary{"Open": fromFile, "OpenBytes": fromBytes} {
		assert.True(t, o.d.PredictionSharded, name)
		assert.Equal(t, d.Parse("бревна"), o.Parse("бревна"), name)
		assert.Equal(t, d.Parse("мошки"), o.Parse("мошки"), name)
	}
}

func TestSingleShardPredictionKeepsOldSection(t *testing.T) {
	b := NewBuilder(BuilderOptions{})
	require.NoError(t, b.AddForm("кот", "кот", "NOUN,anim,masc,sing,nomn"))
	require.NoError(t, b.AddForm("кота", "кот", "NOUN,anim,masc,sing,gent"))
	d, err := b.Build()
	require.NoError(t, err)
	require.False(t, d.d.PredictionSharded)

	path := filepath.Join(t.TempDir(), "single.dat")
	require.NoError(t, d.SaveTo(path))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	cont, err := internal.OpenContainer(data)
	require.NoError(t, err)
	_, _, err = cont.Section("prediction-0")
	require.NoError(t, err)
	_, _, err = cont.Section("pred-sharded-0")
	require.Error(t, err)

	o, err := OpenBytes(data)
	require.NoError(t, err)
	assert.False(t, o.d.PredictionSharded)
	assert.Equal(t, d.Parse("бота"), o.Parse("бота"))
}

func TestOpenRejectsBothPredictionKinds(t *testing.T) {
	d := shardedPredictionDict(t)
	sections, err := d.sections(nil)
	require.NoError(t, err)
	var sharded internal.Section
	for _, s := range sections {
		if s.Name == "pred-sharded-0" {
			sharded = s
		}
	}
	require.NotEmpty(t, sharded.Name)
	sections = append(sections, internal.Section{Name: "prediction-0", Data: sharded.Data, Flags: sharded.Flags})
	path := filepath.Join(t.TempDir(), "both.dat")
	require.NoError(t, internal.SaveContainer(path, sections))

	_, err = Open(path)
	assert.ErrorContains(t, err, "prediction")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ -run 'Shard|BothPrediction|SingleShardPrediction' -count=1`
Expected: FAIL — `бревна` gets no shard-1 reading (prediction resolves against shard 0), the file
has `prediction-0` instead of `pred-sharded-0`, and the "both" file opens.

- [ ] **Step 3: Implement**

`pkg/morphology/parse.go`, in `predictForPrefix` (the Task 9 version): delete
`const predictionShard = 0`, and in the `LookupEach` callback replace the lines from
`if len(v) < 6 {` down to `r := x.readingForm(predictionShard, wordStart+found, paraNum, form)` with:

```go
			if len(v) < 6 {
				return
			}
			count := int(binary.BigEndian.Uint16(v[:2]))
			paraNum := binary.BigEndian.Uint16(v[2:4])
			form := binary.BigEndian.Uint16(v[4:6])
			shard := 0 // 6-byte values (prediction-N) always mean shard 0
			if len(v) >= 8 {
				shard = int(binary.BigEndian.Uint16(v[6:8]))
			}

			para, ok := x.paradigm(shard, paraNum) // false for a missing shard
			if !ok || form >= uint16(para.Len()) {
				return
			}
			if !productive(x.paradigmTag(para, int(form))) {
				return
			}
			totalCount += count

			r := x.readingForm(shard, wordStart+found, paraNum, form)
```

and replace the doc paragraph `// Predictions always resolve against shard 0: …` with:

```go
// A 6-byte value (count|para|form) resolves against shard 0; an 8-byte
// value (count|para|form|shard, sharded dictionaries) carries its shard.
// Values naming a missing shard or paradigm are skipped.
```

`pkg/morphology/open.go`, in `parseContainer` replace the `prediction-%d` loop with:

```go
	plain, err := predictionSections(cont, "prediction-%d")
	if err != nil {
		return nil, err
	}
	sharded, err := predictionSections(cont, "pred-sharded-%d")
	if err != nil {
		return nil, err
	}
	switch {
	case len(plain) > 0 && len(sharded) > 0:
		return nil, fmt.Errorf("morphology: both prediction-N and pred-sharded-N sections present")
	case len(sharded) > 0:
		d.Prediction, d.PredictionSharded = sharded, true
	default:
		d.Prediction = plain
	}
```

and add below `parseContainer`:

```go
// predictionSections parses the consecutive sections named by format
// (N = 0, 1, …) up to the first missing one.
func predictionSections(cont *internal.Container, format string) ([]*internal.DAWG, error) {
	var out []*internal.DAWG
	for i := 0; ; i++ {
		data, _, err := cont.Section(fmt.Sprintf(format, i))
		if err != nil {
			return out, nil
		}
		pred, err := internal.ParseDAWG(data)
		if err != nil {
			return nil, err
		}
		out = append(out, pred)
	}
}
```

`pkg/morphology/save.go`: in `sections` replace the prediction loop with:

```go
	predName := "prediction-%d"
	if x.d.PredictionSharded {
		predName = "pred-sharded-%d"
	}
	for i, pred := range x.d.Prediction {
		if pred == nil {
			continue
		}
		sections = append(sections, internal.Section{
			Name: fmt.Sprintf(predName, i), Data: pred.Bytes(), Flags: noCompression,
		})
	}
```

and in the `SaveTo` doc replace `prediction-N,` with
`prediction-N (or pred-sharded-N, 8-byte values with a shard, for sharded prediction),`.

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/... -race -count=1`
Expected: PASS, including every existing snapshot test (`readingsSnapshot`, `semanticSnapshot`,
`TestMergeRealDictionary`, content-hash tests): single-shard files are unchanged.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/parse.go pkg/morphology/open.go pkg/morphology/save.go pkg/morphology/prediction_sharded_test.go
git commit -m "feat(morphology): read, save and resolve sharded prediction

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 12: Prediction for OpenCorpora/UniMorph imports, CLI, measurements (item L, part 3)

**Files:**
- Modify: `pkg/morphology/open.go` (`XMLOptions`, `CompileFromXMLWithOptions`, `finishCompiled`;
  `CompileFromXML`, `CompileFromXMLDense`, `CompileFromUniMorph`, `CompileFromUniMorphDense` go
  through `finishCompiled`)
- Modify: `pkg/morphology/importers/unimorph/import.go` (`Options.NoPrediction`)
- Modify: `cmd/gomorphy/build.go`, `cmd/gomorphy/update.go:28`, `cmd/gomorphy/build_test.go`,
  `cmd/gomorphy/dict_test.go:204`
- Create: `pkg/morphology/compile_prediction_test.go`
- Modify: `pkg/morphology/example_test.go` (outputs that now include predicted readings)

**Interfaces:**
- Consumes: `internal.BuildPrediction` (Task 10), `productive` (`parse.go`).
- Produces:
  - `type XMLOptions struct { Progress opencorpora.Progress; Dense bool; NoPrediction bool }`
  - `func CompileFromXMLWithOptions(r interface{ Read([]byte) (int, error) }, opts XMLOptions) (*Dictionary, error)`
  - `UniMorphOptions.NoPrediction bool` (field of `unimorph.Options`)
  - CLI `gomorphy build opencorpora|unimorph --no-prediction`

Prediction is built in `pkg/morphology`, not in the importers: `productive` lives there and the
importer packages cannot import `pkg/morphology` (cycle). `unimorph.CompileFromTSV` itself keeps
building no prediction; `NoPrediction` is honoured by `morphology.CompileFromUniMorph*`.

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/compile_prediction_test.go`:

```go
package morphology_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func assertAllPredicted(t *testing.T, d *morphology.Dictionary, word string) {
	t.Helper()
	readings := d.Parse(word)
	require.NotEmpty(t, readings, "%s must be predicted", word)
	for _, r := range readings {
		assert.True(t, r.Predicted, "%s: %+v", word, r)
	}
}

func TestCompileFromXMLPredictsByDefault(t *testing.T) {
	for name, compile := range map[string]func() (*morphology.Dictionary, error){
		"CompileFromXML": func() (*morphology.Dictionary, error) {
			return morphology.CompileFromXML(strings.NewReader(exampleDictXML), nil)
		},
		"CompileFromXMLDense": func() (*morphology.Dictionary, error) {
			return morphology.CompileFromXMLDense(strings.NewReader(exampleDictXML), nil)
		},
	} {
		d, err := compile()
		require.NoError(t, err, name)
		assertAllPredicted(t, d, "бота") // like кота
	}

	d, err := morphology.CompileFromXMLWithOptions(strings.NewReader(exampleDictXML),
		morphology.XMLOptions{Dense: true, NoPrediction: true})
	require.NoError(t, err)
	assert.Nil(t, d.Parse("бота"), "NoPrediction keeps the 1.2 behaviour")
	assert.NotEmpty(t, d.Parse("кота"))
}

func TestCompileFromUniMorphPredictsByDefault(t *testing.T) {
	opts := morphology.UniMorphOptions{Language: "ru"}
	d, err := morphology.CompileFromUniMorphDense(strings.NewReader(uniMorphTSV), opts)
	require.NoError(t, err)
	assertAllPredicted(t, d, "бота")

	opts.NoPrediction = true
	d, err = morphology.CompileFromUniMorphDense(strings.NewReader(uniMorphTSV), opts)
	require.NoError(t, err)
	assert.Nil(t, d.Parse("бота"))
}

// UniMorph rus has five parts of speech, all open classes; productive()
// splits on "," and must filter none of them (spec L, "Tag filter").
func TestCompileFromUniMorphPredictsAllPOS(t *testing.T) {
	tsv := "стол\tстолами\tN;INS;PL\n" +
		"синий\tсиними\tADJ;INS;PL\n" +
		"читать\tчитали\tV;PST;PL\n" +
		"читать\tчитавшими\tV.PTCP;ACT;PST;INS;PL\n" +
		"читать\tчитая\tV.CVB;PRS\n"
	d, err := morphology.CompileFromUniMorph(strings.NewReader(tsv), morphology.UniMorphOptions{Language: "ru"})
	require.NoError(t, err)

	pos := map[string]bool{}
	for _, w := range []string{"стульями", "красными", "писали", "писавшими", "пиная"} {
		assertAllPredicted(t, d, w)
		for _, r := range d.Parse(w) {
			p, _, _ := strings.Cut(r.Tag, ";")
			pos[p] = true
		}
	}
	assert.Equal(t, map[string]bool{"N": true, "ADJ": true, "V": true, "V.PTCP": true, "V.CVB": true}, pos)
}

// TestRealDictionaryPredictsUnknownWords runs against a real .dat built by
// this version (GOMORPHY_BENCH_DICT, as for BenchmarkRealDict).
func TestRealDictionaryPredictsUnknownWords(t *testing.T) {
	path := os.Getenv("GOMORPHY_BENCH_DICT")
	if path == "" {
		t.Skip("GOMORPHY_BENCH_DICT not set")
	}
	d, err := morphology.Open(path)
	require.NoError(t, err)
	defer func() { _ = d.Close() }()
	for _, w := range []string{"кракозябрами", "шмурдяковый", "перепрокрустить"} {
		assertAllPredicted(t, d, w)
	}
}
```

In `cmd/gomorphy/build_test.go` append (`fixtureXML` is in `dict_test.go`, `newTestRootCmd` is the
file's existing root-command helper):

```go
func TestBuildCommand_NoPrediction(t *testing.T) {
	xmlPath := filepath.Join(t.TempDir(), "dict.xml")
	require.NoError(t, os.WriteFile(xmlPath, []byte(fixtureXML("кот")), 0o644))

	for _, noPrediction := range []bool{false, true} {
		outPath := filepath.Join(t.TempDir(), "out.dat")
		args := []string{"build", "opencorpora", "-i", xmlPath, "-o", outPath}
		if noPrediction {
			args = append(args, "--no-prediction")
		}
		root := newTestRootCmd(newBuildCommand())
		root.SetOut(&bytes.Buffer{})
		root.SetArgs(args)
		require.NoError(t, root.Execute())

		d, err := morphology.Open(outPath)
		require.NoError(t, err)
		if noPrediction {
			assert.Nil(t, d.Parse("бот"), "--no-prediction")
		} else {
			assert.NotEmpty(t, d.Parse("бот"), "prediction by default (like кот)")
		}
		require.NoError(t, d.Close())
	}
}

func TestBuildCommand_NoPredictionRejectedForPymorphy(t *testing.T) {
	root := newTestRootCmd(newBuildCommand())
	root.SetArgs([]string{"build", "pymorphy", "--no-prediction", "-i", t.TempDir(), "-o", filepath.Join(t.TempDir(), "x.dat")})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--no-prediction")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ ./cmd/gomorphy/ -run 'Predict|NoPrediction' -count=1`
Expected: FAIL to compile — `XMLOptions`, `CompileFromXMLWithOptions`,
`UniMorphOptions.NoPrediction` undefined.

- [ ] **Step 3: Implement**

`pkg/morphology/importers/unimorph/import.go`, append to `Options`:

```go
	// NoPrediction skips the ending-based prediction for
	// out-of-dictionary words that morphology.CompileFromUniMorph and
	// its variants build by default. ImportFromTSV and CompileFromTSV in
	// this package never build prediction and ignore it.
	NoPrediction bool
```

`pkg/morphology/open.go` — replace the bodies of `CompileFromXML` and `CompileFromXMLDense`, add
the new API next to them:

```go
// XMLOptions configures CompileFromXMLWithOptions.
type XMLOptions struct {
	// Progress reports import progress; nil means no reporting.
	Progress opencorpora.Progress
	// Dense recompiles every shard's words DAWG to one dense 1-byte
	// alphabet (see CompileFromXMLDense).
	Dense bool
	// NoPrediction skips the ending-based prediction for
	// out-of-dictionary words, built by default: Parse then returns nil
	// for a word the dictionary does not contain.
	NoPrediction bool
}

// CompileFromXMLWithOptions compiles an OpenCorpora dictionary from
// dict.xml. Unless opts.NoPrediction is set it builds ending-based
// prediction, so Parse returns Predicted readings for unknown words.
func CompileFromXMLWithOptions(r interface{ Read([]byte) (int, error) }, opts XMLOptions) (*Dictionary, error) {
	d, err := opencorpora.CompileFromXML(r, opts.Progress)
	if err != nil {
		return nil, err
	}
	return finishCompiled(d, opts.Dense, !opts.NoPrediction)
}

// CompileFromXML compiles an OpenCorpora dictionary from dict.xml, with
// prediction; progress is an optional callback for reporting progress.
// See CompileFromXMLWithOptions.
func CompileFromXML(r interface{ Read([]byte) (int, error) }, progress opencorpora.Progress) (*Dictionary, error) {
	return CompileFromXMLWithOptions(r, XMLOptions{Progress: progress})
}

// CompileFromXMLDense is CompileFromXMLWithOptions with Dense set.
func CompileFromXMLDense(r interface{ Read([]byte) (int, error) }, progress opencorpora.Progress) (*Dictionary, error) {
	return CompileFromXMLWithOptions(r, XMLOptions{Progress: progress, Dense: true})
}

// finishCompiled builds prediction (unless prediction is false) and then,
// if dense is set, recompiles the words DAWGs to a dense alphabet — the
// same order Builder uses.
func finishCompiled(d *internal.Dictionary, dense, prediction bool) (*Dictionary, error) {
	if prediction {
		if err := internal.BuildPrediction(d, productive); err != nil {
			return nil, fmt.Errorf("morphology: build prediction: %w", err)
		}
	}
	if dense {
		if err := internal.RecompileDense(d); err != nil {
			return nil, fmt.Errorf("morphology: compile dense: %w", err)
		}
	}
	return &Dictionary{d: d}, nil
}
```

Keep the existing doc comments of `CompileFromXMLDense` (dense-alphabet references) and add the
sentence "Builds prediction unless told otherwise, see CompileFromXMLWithOptions." to it and to
`CompileFromUniMorph`/`CompileFromUniMorphDense`. Their bodies become:

```go
// CompileFromUniMorph
	d, err := unimorph.CompileFromTSV(r, opts)
	if err != nil {
		return nil, err
	}
	return finishCompiled(d, false, !opts.NoPrediction)

// CompileFromUniMorphDense
	d, err := unimorph.CompileFromTSV(r, opts)
	if err != nil {
		return nil, err
	}
	return finishCompiled(d, true, !opts.NoPrediction)
```

The `*File` variants are unchanged: they call these.

`cmd/gomorphy/build.go`:
- `runBuild(cmd *cobra.Command, typ, input, output, lang string, noPrediction bool) error`.
- `case "opencorpora":` —
  `d, err = morphology.CompileFromXMLWithOptions(f, morphology.XMLOptions{Progress: progress, Dense: true, NoPrediction: noPrediction})`
  (keep the dense-by-default comment).
- `case "pymorphy":` — first line:
  ```go
  if noPrediction {
  	return fmt.Errorf("build pymorphy: --no-prediction is not supported: pymorphy2 prediction comes from the source files")
  }
  ```
- `case "unimorph":` — `morphology.UniMorphOptions{Language: lang, NoPrediction: noPrediction}`.
- In `newBuildCommand`:
  ```go
  cmd.Flags().Bool("no-prediction", false,
  	"skip ending-based prediction for out-of-dictionary words (opencorpora, unimorph)")
  ```
  read it with `noPrediction, _ := cmd.Flags().GetBool("no-prediction")` and pass it on.

`cmd/gomorphy/update.go:28`: `return runBuild(cmd, typ, "", output, lang, false)`.

- [ ] **Step 4: Run the whole suite and fix example outputs**

Run: `go test ./... -race -count=1`
Expected: the new tests PASS. `cmd/gomorphy/dict_test.go`
`TestResolveDictionaries_FlagTakesPriorityOverEnv` asserts `assert.Empty(t, got.Parse("груша"), …)`
on a `CompileFromXML` fixture; it still passes (no suffix of «груша» occurs in «кот»), but it now
tests prediction by accident — replace it with `assert.False(t, got.IsKnown("груша"), …)` keeping
the message. `Example*` functions built on `exampleDictXML`/`exampleDictXML2`
through `CompileFromXML` (e.g. `ExampleNewMultiDictionary`) may now print extra readings: a word
known to one dictionary is predicted by the other. For each failing example, check that every new
output line is a `Predicted` reading from the dictionary that lacks the word, then update its
`// Output:` block to the new output. Do not pass `NoPrediction` just to keep an old output; if
the example's prose says an unknown word yields nothing, correct the prose. Re-run until green.

- [ ] **Step 5: Measure on real dictionaries**

```bash
cd /Users/asmarin/dev/mine/gomorphy
for t in opencorpora unimorph; do
  /usr/bin/time -p go run ./cmd/gomorphy build $t -o /tmp/gomorphy-$t-pred.dat
  /usr/bin/time -p go run ./cmd/gomorphy build $t --no-prediction -o /tmp/gomorphy-$t-nopred.dat
done
ls -l /tmp/gomorphy-*-pred.dat /tmp/gomorphy-*-nopred.dat
for t in opencorpora unimorph; do
  GOMORPHY_BENCH_DICT=/tmp/gomorphy-$t-pred.dat go test ./pkg/morphology/ \
    -run TestRealDictionaryPredictsUnknownWords -bench BenchmarkRealDict -benchmem -count=5 \
    | tee /tmp/gomorphy-bench-$t-pred.txt
done
```

Expected: `TestRealDictionaryPredictsUnknownWords` PASS for both; `BenchmarkRealDict`'s
`ParsePredicted` sub-benchmark now reports real work for OpenCorpora (it returned nil before).
Record sizes, build times and `ParsePredicted` numbers for the Task 13 write-up. **Gate:** if
prediction grows either file by more than 50%, stop and report to the owner before Task 13 —
the default-on decision (spec L, G6) is to be revisited.

- [ ] **Step 6: Lint and commit**

Run: `gofmt -l . && go vet ./... && golangci-lint run ./...` — clean.

```bash
git add pkg/morphology/open.go pkg/morphology/importers/unimorph/import.go pkg/morphology/compile_prediction_test.go pkg/morphology/example_test.go cmd/gomorphy/build.go cmd/gomorphy/update.go cmd/gomorphy/build_test.go cmd/gomorphy/dict_test.go
git commit -m "feat(morphology): predict unknown words in OpenCorpora and UniMorph dictionaries

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 12a: pymorphy2-style pruning of imported prediction (item L, part 4)

Added 2026-09-27 after Task 12's measurement tripped the 50% gate (OpenCorpora +126%, UniMorph +67%,
UniMorph `ParsePredicted` ≈340 µs). Spec L, "Pruning".

**Files:**
- Modify: `pkg/morphology/internal/prediction.go` (`PredictionPruning`, `ImportPredictionPruning`,
  `BuildPredictionPruned`, pruning in the shared builder)
- Modify: `pkg/morphology/internal/prediction_test.go`
- Modify: `pkg/morphology/open.go` (`finishCompiled` uses the pruned build)
- Modify: `pkg/morphology/compile_prediction_test.go` (fixtures with ≥3 lemmas per paradigm; the real-dictionary
  nonce word)
- Modify: `cmd/gomorphy/build_test.go` (fixture with ≥3 lemmas)

**Interfaces:**
- Consumes: `BuildPrediction`, `BuildPredictionFrom`, `shardPairs` (Task 10), `FirstGrammeme` (Task 1),
  `finishCompiled` (Task 12).
- Produces:
  - `type PredictionPruning struct { MinEndingFreq, MinParadigmPopularity, MaxFormsPerClass int }` — zero value
    keeps everything.
  - `var ImportPredictionPruning = PredictionPruning{MinEndingFreq: 2, MinParadigmPopularity: 3, MaxFormsPerClass: 1}`
  - `func BuildPredictionPruned(d *Dictionary, productive func(tag string) bool, p PredictionPruning) error`
  - `BuildPrediction` and `BuildPredictionFrom` keep their signatures and behaviour (= zero pruning), so
    Builder/ImportTSV/Merge output stays byte-identical.

Rules (pymorphy2's `_suffixes_prediction_data`):
1. Paradigm popularity = number of form-0 readings of `(shard, para)` across the input pairs. With
   `MinParadigmPopularity > 0`, readings of paradigms below it are ignored entirely (they feed neither counts nor
   ending frequencies).
2. For every remaining productive reading and each suffix length 1..5: `counts[(suffix, para, form, shard)]++` and
   `endingFreq[suffix]++`.
3. With `MinEndingFreq > 0`, keys whose `endingFreq[suffix] < MinEndingFreq` are dropped.
4. With `MaxFormsPerClass > 0`, keys are grouped by `(suffix, FirstGrammeme(tag of para/form))`; each group keeps
   its `MaxFormsPerClass` entries with the highest count, ties broken by lowest `(shard, para, form)`.
5. Encoding (6 vs 8 bytes, `sharded`) is unchanged.

- [ ] **Step 1: Write the failing tests**

Append to `pkg/morphology/internal/prediction_test.go`:

```go
// pruningCorpus: paradigm A (кошка/кошки, мышка/мышки, пешка/пешки — 3
// lemmas), paradigm B (окно/окна — 1 lemma), paradigm C (ток/токи, сок/соки,
// бок/боки, рок/роки — 4 lemmas) and paradigm D (single-form ADJS легки,
// мягки, жарки — 3 lemmas). All tags productive.
func pruningCorpus(t *testing.T) *Dictionary {
	t.Helper()
	var entries []BuildEntry
	for _, s := range []string{"кошк", "мышк", "пешк"} {
		entries = append(entries,
			BuildEntry{Word: s + "а", Lemma: s + "а", Tag: "NOUN,femn,sing,nomn"},
			BuildEntry{Word: s + "и", Lemma: s + "а", Tag: "NOUN,femn,sing,gent"})
	}
	entries = append(entries,
		BuildEntry{Word: "окно", Lemma: "окно", Tag: "NOUN,neut,sing,nomn"},
		BuildEntry{Word: "окна", Lemma: "окно", Tag: "NOUN,neut,sing,gent"})
	for _, s := range []string{"ток", "сок", "бок", "рок"} {
		entries = append(entries,
			BuildEntry{Word: s, Lemma: s, Tag: "NOUN,masc,sing,nomn"},
			BuildEntry{Word: s + "и", Lemma: s, Tag: "NOUN,masc,plur,nomn"})
	}
	for _, w := range []string{"легки", "мягки", "жарки"} {
		entries = append(entries, BuildEntry{Word: w, Lemma: w, Tag: "ADJS,plur"})
	}
	d, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)
	require.Len(t, d.Words, 1)
	return d
}

func TestBuildPredictionPrunedZeroIsUnpruned(t *testing.T) {
	a, b := pruningCorpus(t), pruningCorpus(t)
	require.NoError(t, BuildPrediction(a, predProductive))
	require.NoError(t, BuildPredictionPruned(b, predProductive, PredictionPruning{}))
	assert.Equal(t, a.Prediction[0].Bytes(), b.Prediction[0].Bytes())
	assert.Equal(t, a.PredictionSharded, b.PredictionSharded)
}

func TestBuildPredictionPrunedParadigmPopularity(t *testing.T) {
	d := pruningCorpus(t)
	require.NoError(t, BuildPredictionPruned(d, predProductive, PredictionPruning{MinParadigmPopularity: 3}))
	assert.Empty(t, predValues(t, d.Prediction[0], "на"), "окно's paradigm has 1 lemma")
	assert.Empty(t, predValues(t, d.Prediction[0], "кно"))
	pA, _ := buildLookup(t, d, "кошка")
	assert.Equal(t, []struct{ Count, Para, Form uint16 }{{3, pA, 0}}, predValues(t, d.Prediction[0], "шка"))
}

func TestBuildPredictionPrunedEndingFreq(t *testing.T) {
	d := pruningCorpus(t)
	require.NoError(t, BuildPredictionPruned(d, predProductive, PredictionPruning{MinEndingFreq: 2}))
	assert.Empty(t, predValues(t, d.Prediction[0], "ошка"), "only кошка ends in «ошка»")
	assert.Empty(t, predValues(t, d.Prediction[0], "кошка"))
	assert.NotEmpty(t, predValues(t, d.Prediction[0], "шка"), "кошка, мышка, пешка")
	assert.Empty(t, predValues(t, d.Prediction[0], "на"), "only окна ends in «на»")
	assert.Empty(t, predValues(t, d.Prediction[0], "кна"))
	assert.NotEmpty(t, predValues(t, d.Prediction[0], "а"), "кошка, мышка, пешка, окна")
}

func TestBuildPredictionPrunedMaxFormsPerClass(t *testing.T) {
	d := pruningCorpus(t)
	require.NoError(t, BuildPredictionPruned(d, predProductive, PredictionPruning{MaxFormsPerClass: 1}))
	pC, fC := buildLookup(t, d, "токи")
	pD, fD := buildLookup(t, d, "легки")
	// «ки»: NOUN A/1 (кошки…, 3), NOUN C/1 (токи…, 4), ADJS D/0 (легки…, 3):
	// one per class — C wins NOUN, D is the only ADJS.
	assert.ElementsMatch(t, []struct{ Count, Para, Form uint16 }{
		{4, pC, fC},
		{3, pD, fD},
	}, predValues(t, d.Prediction[0], "ки"))
}

func TestImportPredictionPruningIsPymorphyDefault(t *testing.T) {
	assert.Equal(t, PredictionPruning{MinEndingFreq: 2, MinParadigmPopularity: 3, MaxFormsPerClass: 1}, ImportPredictionPruning)
}
```

In `pkg/morphology/compile_prediction_test.go`:
- add fixtures with ≥3 lemmas per paradigm and switch the default-on tests to them:

```go
// predictionXML: three nouns sharing one paradigm (""/"а"), so the
// pymorphy2-style pruning (≥3 lemmas per paradigm, ≥2 readings per ending)
// keeps their endings.
const predictionXML = `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">сущ</grammeme><grammeme id="inan">неод</grammeme><grammeme id="masc">м</grammeme>
  <grammeme id="sing">ед</grammeme><grammeme id="nomn">им</grammeme><grammeme id="gent">род</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="кот"><l t="кот"><g v="NOUN"/><g v="inan"/><g v="masc"/><g v="sing"/></l><f t="кот"><g v="nomn"/></f><f t="кота"><g v="gent"/></f></lemma>
  <lemma id="2" text="лот"><l t="лот"><g v="NOUN"/><g v="inan"/><g v="masc"/><g v="sing"/></l><f t="лот"><g v="nomn"/></f><f t="лота"><g v="gent"/></f></lemma>
  <lemma id="3" text="скот"><l t="скот"><g v="NOUN"/><g v="inan"/><g v="masc"/><g v="sing"/></l><f t="скот"><g v="nomn"/></f><f t="скота"><g v="gent"/></f></lemma>
 </lemmata>
</dictionary>`

const predictionTSV = "кот\tкот\tN;NOM;SG\nкот\tкота\tN;GEN;SG\n" +
	"лот\tлот\tN;NOM;SG\nлот\tлота\tN;GEN;SG\n" +
	"скот\tскот\tN;NOM;SG\nскот\tскота\tN;GEN;SG\n"
```

  `TestCompileFromXMLPredictsByDefault` and `TestCompileFromUniMorphPredictsByDefault` use `predictionXML` /
  `predictionTSV` instead of `exampleDictXML` / `uniMorphTSV` (keep «бота» as the unknown word and «кота» as the
  known one); add to the XML test that the tiny `exampleDictXML` (one lemma per paradigm) now predicts nothing:
  `assert.Nil(t, tiny.Parse("бота"), "pruning drops paradigms with fewer than 3 lemmas")`.
- `TestCompileFromUniMorphPredictsAllPOS`: give every POS three lemmas that share its paradigm, e.g.

```go
	tsv := ""
	for _, s := range []string{"стол", "вол", "кол"} {
		tsv += s + "\t" + s + "ами\tN;INS;PL\n" + s + "\t" + s + "\tN;NOM;SG\n"
	}
	for _, s := range []string{"син", "зимн", "летн"} {
		tsv += s + "ий\t" + s + "ими\tADJ;INS;PL\n" + s + "ий\t" + s + "ий\tADJ;NOM;SG;MASC\n"
	}
	for _, s := range []string{"чит", "кат", "мот"} {
		tsv += s + "ать\t" + s + "ать\tV;NFIN\n" +
			s + "ать\t" + s + "али\tV;PST;PL\n" +
			s + "ать\t" + s + "авшими\tV.PTCP;ACT;PST;INS;PL\n" +
			s + "ать\t" + s + "ая\tV.CVB;PRS\n"
	}
```

  Keep the five probe words and the expected set of five POS. If one probe no longer reaches its POS because of
  the per-class cap, pick a probe whose ending is unique to that POS in the fixture (e.g. «писавшими» for V.PTCP)
  and say so in the report — the assertion stays "all five POS are predicted".
- `TestRealDictionaryPredictsUnknownWords`: replace «кракозябрами» by a word absent from both sources — verify with
  `grep -c -w <word> .data/opencorpora/dict.xml .data/unimorph/ru/data` (0 and 0) before using it, e.g. try
  «шмуклерами», «кракозяврами».

In `cmd/gomorphy/build_test.go`, `TestBuildCommand_NoPrediction` builds from a three-lemma fixture instead of
`fixtureXML("кот")`: write a local `const threeCatsXML` with lemmas кот, лот, скот (nomn form only, same tags as
`fixtureXML`) and keep «бот» as the probe.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/internal/ -run 'Pruned|PymorphyDefault' -count=1`
Expected: FAIL to compile — `BuildPredictionPruned`, `PredictionPruning`, `ImportPredictionPruning` undefined.

- [ ] **Step 3: Implement**

`pkg/morphology/internal/prediction.go`:

```go
// PredictionPruning trims a prediction DAWG the way pymorphy2's dictionary
// compiler does. The zero value keeps everything.
type PredictionPruning struct {
	// MinEndingFreq drops suffix keys attested by fewer readings.
	MinEndingFreq int
	// MinParadigmPopularity ignores paradigms used by fewer lemmas
	// (form-0 readings).
	MinParadigmPopularity int
	// MaxFormsPerClass keeps, per suffix and part of speech (the tag's
	// first grammeme), only this many most attested (paradigm, form)
	// entries; 0 keeps all.
	MaxFormsPerClass int
}

// ImportPredictionPruning is pymorphy2's compile default, used for the
// OpenCorpora and UniMorph imports.
var ImportPredictionPruning = PredictionPruning{MinEndingFreq: 2, MinParadigmPopularity: 3, MaxFormsPerClass: 1}
```

- `BuildPrediction(d, productive)` becomes `return BuildPredictionPruned(d, productive, PredictionPruning{})`;
  `BuildPredictionPruned` holds today's `BuildPrediction` body and calls `buildPrediction(pairs, d.Paradigms,
  d.TagSet, productive, p)`.
- `BuildPredictionFrom(pairs, paradigms, tagSet, productive)` becomes a one-line wrapper over
  `buildPrediction(..., PredictionPruning{})`; `buildPrediction` holds today's body plus the rules above:
  compute popularity first (only when `MinParadigmPopularity > 0`), count `endingFreq` alongside `counts`, then
  filter keys (rules 3-4) before encoding. Keep the zero-pruning path allocation-equivalent to today (no grouping
  maps when both `MinEndingFreq` and `MaxFormsPerClass` are 0). Sort within a class group with
  `slices.SortFunc` by (count desc, shard, para, form asc).

`pkg/morphology/open.go`, `finishCompiled`: call
`internal.BuildPredictionPruned(d, productive, internal.ImportPredictionPruning)`; update its doc and the doc of
`XMLOptions.NoPrediction`/`CompileFromXMLWithOptions` with one sentence: "Prediction is pruned like pymorphy2's
(paradigms of at least 3 lemmas, endings attested at least twice, the most attested form per ending and part of
speech)."

- [ ] **Step 4: Run the tests**

Run: `go test ./... -race -count=1`
Expected: PASS. Builder/ImportTSV/Merge snapshot tests unchanged (zero pruning).

- [ ] **Step 5: Measure again**

```bash
cd /Users/asmarin/dev/mine/gomorphy
for t in opencorpora unimorph; do
  /usr/bin/time -p go run ./cmd/gomorphy build $t -o /tmp/gomorphy-$t-pruned.dat
done
ls -l /tmp/gomorphy-*-nopred.dat /tmp/gomorphy-*-pred.dat /tmp/gomorphy-*-pruned.dat
for t in opencorpora unimorph; do
  GOMORPHY_BENCH_DICT=/tmp/gomorphy-$t-pruned.dat go test ./pkg/morphology/ \
    -run TestRealDictionaryPredictsUnknownWords -bench BenchmarkRealDict -benchmem -count=5 \
    | tee /tmp/gomorphy-bench-$t-pruned.txt
done
```

(`/tmp/gomorphy-*-nopred.dat` and `-pred.dat` exist from Task 12; rebuild `-nopred` with `--no-prediction` if they
are gone.) Expected: `TestRealDictionaryPredictsUnknownWords` PASS on both. Record sizes (growth vs `-nopred`),
build times and `ParsePredicted` medians. **Gate (ruling R13):** if either growth is still > 50% or the UniMorph
`ParsePredicted` median of either probe word is > 50 µs, commit anyway and report DONE_WITH_CONCERNS with the
numbers — the owner decides before Task 13.

- [ ] **Step 6: Lint and commit**

Run: `gofmt -l . && go vet ./... && golangci-lint run ./...` — clean.

```bash
git add pkg/morphology/internal/prediction.go pkg/morphology/internal/prediction_test.go pkg/morphology/open.go pkg/morphology/compile_prediction_test.go cmd/gomorphy/build_test.go
git commit -F <message file>   # "feat(morphology): prune imported prediction like pymorphy2" + trailer
```

---

### Task 13: Documentation, CHANGELOG, version 1.3.0

**Files:**
- Modify: `pkg/morphology/version.go:7`, `CHANGELOG.md`
- Modify: `docs/en/library.md`, `docs/ru/library.md`
- Modify: `docs/en/todo.md` (Forms/Inflect section; Completed stages row)
- Modify: `docs/en/implementation.md:190-201` (Metrics table)
- Modify: `docs/en/implementation/ner-support.md` (append the 1.3.0 part)
- Modify: `docs/en/scenarios.md` (scenario 2 note, scenario 3), `docs/en/comparison.md`
  (prediction row, summary)

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
- Ending-based prediction for OpenCorpora and UniMorph dictionaries: `Parse`
  returns `Predicted` readings for unknown words instead of nil. Built by
  default; opt out with `XMLOptions.NoPrediction` (new
  `CompileFromXMLWithOptions`), `UniMorphOptions.NoPrediction` or
  `gomorphy build --no-prediction`. Stored in new `pred-sharded-N`
  sections; gomorphy 1.2.x opens such files without prediction.

### Changed
- `Parse`: no goroutine for single-shard dictionaries, no per-rune/per-value
  allocations; allocations per known word <N> (was 22–24), per predicted word
  <M> (was 44) — see implementation/ner-support.md.
- Builder/ImportTSV group forms into lexemes by lemma **and** part-of-speech
  class: «знать» NOUN and «знать» INFN become two lemmas. Dictionaries rebuilt
  from homonymous input differ from 1.2.0 output; existing files are unaffected.

- `MergeOptions.RebuildPrediction` (CLI `merge --rebuild-prediction`) works
  for merged dictionaries with more than one shard.

### Deprecated
- `ErrPredictionSharded` — never returned any more.

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
  `Forms`/`Inflect`. Prediction (L): wherever the page says OpenCorpora/UniMorph dictionaries have
  no prediction, say they have it by default since 1.3.0; document `XMLOptions`,
  `CompileFromXMLWithOptions`, `UniMorphOptions.NoPrediction`, `build --no-prediction`, and that
  `RebuildPrediction` works for any shard count; mention `IsKnown` as the way to tell a guess from
  a dictionary word.

- [ ] **Step 4: `docs/ru/library.md`** — minimal Russian subset: `Grammemes`/`HasGrammeme`/`POS`,
  `Forms`/`Inflect` (one example each), `ParseAppend`, one sentence on the POS-class grouping,
  one sentence on prediction for OpenCorpora/UniMorph and `--no-prediction`.

- [ ] **Step 5: `docs/en/todo.md`** — replace the body of "Word inflection / form generation
  (`Forms`, `Inflect`) — PLANNED" with a short "DONE in 1.3.0" note: the shape shipped
  (`Inflect(r, want ...string)`, variadic, native tokens; forms of predicted readings allowed and
  flagged), the ranking rule, link to `implementation/ner-support.md`; keep open item (1)'s
  tagmap-based universal query as a v2 idea. Add a Completed-stages row:

```markdown
| — NER support for lexicon: 1.3.0 (tag helpers, Forms/Inflect, Parse performance, POS-aware Builder, 2-byte alphabet fallback, sharded prediction) | DONE | [implementation/ner-support.md](implementation/ner-support.md) |
```

  In the Builder/merge section (around "a sharded merge output keeps prediction absent, same as
  OpenCorpora dictionaries"), replace that clause with "sharded outputs get sharded prediction
  (1.3.0)".

- [ ] **Step 5a: `docs/en/scenarios.md` and `docs/en/comparison.md`** — scenarios: replace
  "OpenCorpora and UniMorph imports have no prediction: `Parse` returns `nil` for an unknown word."
  with the 1.3.0 behaviour (predicted by default, `IsKnown` to check, `--no-prediction` /
  `NoPrediction` to turn it off) and add a `1.3.0` line to that scenario's **History**. comparison:
  in the "Prediction for unknown words" row replace "for pymorphy2 dictionaries and for dictionaries
  built with Builder/ImportTSV (not for OpenCorpora or UniMorph imports)" with "for every
  dictionary source (pymorphy2's own, built for OpenCorpora/UniMorph imports and Builder/ImportTSV;
  opt-out)". In "All wordforms of a lemma" and "Inflection / form generation" rows replace
  "not a dedicated API" / "not implemented" with `Forms` / `Inflect` (H), and update the Summary
  paragraph that says the alternatives are closer fits for inflection.

- [ ] **Step 6: `docs/en/implementation.md` Metrics table** — replace the `Parse (exact)`,
  `Parse (prediction)`, `Lemmas` rows' values with the measured medians from
  `/tmp/gomorphy-bench-real-after.txt` (pymorphy.dat; OpenCorpora for exact where available), each
  with "(measured 1.3.0, <CPU>)", and link the write-up.

- [ ] **Step 7: Write-up** — append to `docs/en/implementation/ner-support.md` a "1.3.0" part: one
  paragraph per item G–L (L: the format decision and why the section is new, the measured file
  sizes and build times with and without prediction and `ParsePredicted` on OpenCorpora/UniMorph
  from Task 12 Step 5); a "Performance" table (benchmark, before, after: ns/op, B/op, allocs/op,
  fixture and real dictionary, CPU from `go test` output header); the `TestParseAllocs` numbers;
  whether the spec targets (Parse < 10 µs, prediction < 50 µs) hold; the Discrepancies D-12…D-22
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
K → `dense_recompile_test.go`, `builder_internal_test.go`; L → `internal/prediction_test.go`,
`internal/merge_test.go`, `prediction_sharded_test.go`, `compile_prediction_test.go`,
`cmd/gomorphy/build_test.go`.

- [ ] **Step 9: Commit**

```bash
git add pkg/morphology/version.go CHANGELOG.md docs/en/library.md docs/ru/library.md docs/en/todo.md docs/en/implementation.md docs/en/implementation/ner-support.md docs/en/scenarios.md docs/en/comparison.md
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

- **D-19 (L).** The spec says the importers build prediction. They cannot: `productive` lives in
  `pkg/morphology`, which imports the importer packages. Prediction is built by the
  `morphology.CompileFrom*` wrappers (`finishCompiled`); `unimorph.Options.NoPrediction` is a field
  of the importer's options that only those wrappers honour, and is documented as such.
- **D-20 (L).** Generalizing `BuildPrediction` through `shardPairs` makes it accept dense
  dictionaries too (keys are decoded through the alphabet); the "must still be raw" precondition
  goes away. `Merge`'s rebuild re-reads all output shards instead of special-casing a reused
  shard 0.
- **D-21 (L).** `pymorphy` has no `--no-prediction`: its prediction comes from the source files.
  The CLI rejects the flag for `pymorphy` with an error rather than silently ignoring it.
- **D-22 (L).** Examples and CLI tests built on OpenCorpora XML fixtures via `CompileFromXML` now
  get prediction; `MultiDictionary` examples may print extra predicted readings. Task 12 updates
  their outputs instead of opting out.

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
- **Q9 (L).** RESOLVED 2026-09-27 (owner): prediction for sharded dictionaries is in 1.3.0 as item
  L — one prefix-0 DAWG with the shard in an 8-byte value, new `pred-sharded-N` sections,
  built by default with an opt-out. Revisit the default if Task 12 Step 5 measures more than 50%
  file growth.
- **Q10 (L, G).** If Q7 is taken (`productive` switches to `internal.NextGrammeme`, splitting on
  `;` too), UniMorph tags get split into tokens for the first time. UniMorph `rus` has no
  nonproductive POS today (only `N`, `ADJ`, `V`, `V.PTCP`, `V.CVB`), and `CONJ`/`INTJ` would be
  filtered correctly anyway; `TestCompileFromUniMorphPredictsAllPOS` guards the outcome either way.
