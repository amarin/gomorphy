# NER support, release 1.2.0 (items A–F) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unblock the lexicon module (and its first host, genodex) with small, mostly-fix changes:
a `Predicted` flag on readings plus `IsKnown` (A), `OpenBytes` for embedded dictionaries (B),
consistent lower-casing in Builder/ImportTSV/Fuzzy (C), a public `CharPolicy` wired into
`BuilderOptions` and `Fuzzy` (D), `Dictionary.ContentHash` (E), and a minimal `go` directive plus
a documented `Close` lifecycle (F). Ship as 1.2.0.

**Architecture:** Every item is a local change in `pkg/morphology` (public API) with at most a
one-function touch in `pkg/morphology/internal`. A marks readings in `predictForPrefix` (the only
place predictions are built) and adds an exact-only lookup. B splits `Open` into "get bytes" +
a shared `openBytes` parser. C/D thread lower-casing and the policy through the existing
`buildFromEntries` helper and the Fuzzy DP row. E factors the section list out of `SaveTo` so the
hash and the file are produced by the same encoder. F is `go.mod` (`go 1.25.0` + `toolchain
go1.27.1`, no dependency changes) + docs.
The binary format does not change.

**Tech Stack:** Go (toolchain 1.27.1; the `go` directive is lowered to 1.25.0 in Task 10 by the
owner policy "current Go minus two minor versions", spec item F / G3),
testify, cobra (CLI), `github.com/zeebo/xxh3` (already a direct dependency). No new dependencies.

**Spec:** [docs/en/superpowers/specs/2026-09-24-ner-support-design.md](../specs/2026-09-24-ner-support-design.md)

## Global Constraints

- The GMOR binary format does not change (no `internal.Version` bump, no new sections).
- Scope criterion (spec): add API only where the caller has no workaround. No convenience
  wrappers, no aliases/entities/rules logic — that belongs to lexicon.
- Public names exactly as in the spec: `Reading.Predicted`, `LemmaRef.Predicted`,
  `Dictionary.IsKnown`, `MultiDictionary.IsKnown`, `OpenBytes(data []byte)`, `CharPolicy`,
  `Substitution`, `RussianCharPolicy()`, `NoCharPolicy()`, `BuilderOptions.CharPolicy`,
  `Dictionary.ContentHash() string`.
- **CharPolicy is exposed as type aliases of the internal types** (`type CharPolicy =
  internal.CharPolicy`), not as a new struct — see Discrepancy D-2. This is what makes
  `UniMorphOptions.CharPolicy` usable from outside with zero type changes.
- **Do not use stdlib or language features newer than Go 1.25** in new code. Task 10 lowers the
  `go` directive to `go 1.25.0` (owner policy "current Go minus two minor versions"); from then
  on `go vet` (the `stdversion` analyzer) and the compiler's `-lang` check reject newer ones.
  Everything up to 1.25 is allowed: `strings.SplitSeq`/`FieldsSeq`/`Lines`, range-over-func,
  `iter`, `b.Loop`, `t.Context`, `sync.WaitGroup.Go`. Not allowed: Go 1.26+ additions (for
  example `new(expr)`, `errors.AsType`).
- Lower-casing uses `strings.ToLower` everywhere (the same function `Parse` already uses,
  `parse.go:33`). Tags are never lower-cased or normalized.
- CharPolicy stays **directional** (query rune `From` matches stored rune `To`), exactly like
  `SimilarItems` (`internal/similar_items.go:46-55`). `Fuzzy` must agree with `Parse`.
- Errors are wrapped with context (`fmt.Errorf("morphology: ...: %w", err)`), matching
  `open.go`/`save.go`.
- Run after every task: `go build ./... && go vet ./... && go test ./... -race -count=1`, and
  `golangci-lint run ./...` before the task's commit. `gofmt -l .` must print nothing.
- Commit messages follow the repo style (`feat(morphology): …`, `fix(morphology): …`,
  `docs: …`, `build: …`) and end with the line
  `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Stage only the files the task lists (`git add <paths>`, never `git add -A`).

## File Structure

| File | Change | Responsibility |
|---|---|---|
| `pkg/morphology/parse.go` | modify | `Reading.Predicted`; set it in `predictForPrefix` |
| `pkg/morphology/lemma.go` | modify | `LemmaRef.Predicted` |
| `pkg/morphology/known.go` | create | `Dictionary.IsKnown`, `MultiDictionary.IsKnown` |
| `pkg/morphology/known_test.go` | create | tests for A |
| `cmd/gomorphy/lookup.go`, `lookup_test.go` | modify | `(predicted)` marker |
| `pkg/morphology/open.go` | modify | `OpenBytes`, shared `openBytes`; `Close` doc |
| `pkg/morphology/open_bytes_test.go` | create | tests for B |
| `pkg/morphology/builder.go` | modify | lower-casing in `AddForm`; `BuilderOptions.CharPolicy`; policy default |
| `pkg/morphology/import_tsv.go` | modify | lower-casing |
| `pkg/morphology/fuzzy.go` | modify | lower-case query; CharPolicy in the DP row |
| `pkg/morphology/fuzzy_internal_test.go` | modify | new `fuzzyWalkShard` argument |
| `pkg/morphology/case_test.go` | create | tests for C |
| `pkg/morphology/char_policy.go` | create | public `CharPolicy`/`Substitution` aliases + constructors |
| `pkg/morphology/char_policy_test.go` | create | tests for D |
| `pkg/morphology/fuzzy_test.go` | modify | ё/е distance expectation |
| `pkg/morphology/save.go` | modify | `sections()` shared by `SaveTo` and `ContentHash` |
| `pkg/morphology/content_hash.go` | create | `ContentHash` |
| `pkg/morphology/content_hash_test.go` | create | tests for E |
| `pkg/morphology/dictionary.go` | modify | hash cache fields; type doc (OpenBytes, lifecycle) |
| `pkg/morphology/lifecycle_test.go` | create | strings survive `Close` |
| `pkg/morphology/example_test.go` | modify | `ExampleOpenBytes`, `ExampleDictionary_IsKnown` |
| `go.mod`, `go.sum` | modify | `go 1.25.0`, `toolchain go1.27.1` (dependencies unchanged) |
| `pkg/morphology/version.go` | modify | `1.2.0` |
| `CHANGELOG.md`, `docs/en/library.md`, `docs/ru/library.md`, `docs/en/cli.md`, `docs/ru/cli.md`, `docs/en/comparison.md`, `docs/en/todo.md`, `docs/en/installation.md`, `docs/ru/installation.md` | modify | docs |
| `docs/en/implementation/ner-support.md` | create | implementation write-up (1.2.0 part) |

## Planning-time findings (verified against 1.1.0 source)

These were checked while writing this plan; tasks rely on them.

1. **Returned strings do not alias the mmap region.** Audit of every path that produces a string
   handed to the caller:
   - `internal.DecodeStrings` builds `string(data[p:p+n])` — a copy (`internal/format.go:437`).
     Suffixes/prefixes are therefore heap strings.
   - `internal.DecodeTagSet`, `DecodeBuildInfo` go through `encoding/json` — copies.
   - `DecodeAlphabet` copies into `[]rune`.
   - `SimilarItems` builds `Item.Key` from the *query* runes (`prefix + string(key[...])`,
     `internal/similar_items.go:64`), never from the DAWG; values come from `b64d`, which
     allocates (`internal/dawg.go:329-336`).
   - `Fuzzy` builds `FuzzyMatch.Word` via `string(f.path)` or `Alphabet.Decode` — copies.
   - The only aliasing is `internal.ParseDAWG` (`internal/dawg.go:118-119`, `unsafe.Slice` over
     the unit array, plus the `guide` sub-slice). Those arrays are read by every in-flight query,
     which is why `Close` during a query crashes, but nothing returned to the caller points into
     them. Task 11 locks this in with a test that reads every returned string after `Close`.
2. **Case bug shape.** A Builder dictionary containing only «Москва» *does* return a reading for
   `Parse("москва")` today — but it is a **prediction** (suffix «осква» from the prediction DAWG,
   `Form 0`, `Normal "москва"`), indistinguishable from a real reading before item A. Measured on
   1.1.0: `Parse("москва") = [{Word:москва Normal:москва Tag:NOUN,… Form:0}]`, while
   `Fuzzy("Москва", 0)` finds the stored capitalized key. Tests in Task 5 therefore assert
   `Predicted == false`, not just "non-empty".
3. **`go` directive.** The owner policy (spec item F, G3) is "current Go minus two minor
   versions": Go 1.27 is current, so `go 1.25.0` + `toolchain go1.27.1`. Current dependency
   `go` lines: `golang.org/x/sys v0.47.0` → 1.25.0 (the highest; enters only through
   `cmd/gomorphy` → `golang.org/x/term v0.31.0`, which declares 1.23.0),
   `klauspost/cpuid/v2 v2.4.0` → 1.24.0, `zeebo/xxh3 v1.1.0` → 1.22. So 1.25.0 needs **no
   dependency downgrades**, and `strings.SplitSeq` (1.24, `parse.go:279`) stays. Re-verified
   2026-09-24 in a scratch copy: `go mod edit -go=1.25.0 -toolchain=go1.27.1 && go mod tidy`
   keeps `go 1.25.0` with unchanged requirements; `go vet ./...` and `go build ./...` are clean;
   `-go=1.24.0` + tidy is raised back to `go 1.25.0` (by `x/sys`). The technical floor after
   downgrades would be 1.22 (set by `xxh3`) — not used, by policy.
4. **Builder output is deterministic.** Building the same entries twice and saving yields
   byte-identical sections except `info` (checked 20×, scratch test) — `BuildPredictionFrom`
   iterates a map, but `buildDAWGWithPayload` sorts keys. Task 9's "rebuild → equal hash" test is
   therefore sound.

---

### Task 1: `Reading.Predicted` and `LemmaRef.Predicted` (item A, part 1)

**Files:**
- Modify: `pkg/morphology/parse.go:13-23` (Reading), `parse.go:176` (predictForPrefix)
- Modify: `pkg/morphology/lemma.go:5-11`, `lemma.go:35`
- Create: `pkg/morphology/known_test.go`

**Interfaces:**
- Produces: `Reading.Predicted bool`, `LemmaRef.Predicted bool` (both appended as the last field).

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/known_test.go`:

```go
package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// predictionFixture is the pymorphy2 fixture with two prediction suffixes,
// the same one TestParsePredictedForm uses: "котёнка" is absent from the
// dictionary and is predicted via "ёнка".
func predictionFixture(t *testing.T) *morphology.Dictionary {
	t.Helper()
	m := map[string]uint32{}
	stdWords(m)
	pred := map[string]uint32{}
	addPrediction(pred, "ёнк", 2, 0, 0)
	addPrediction(pred, "ёнка", 3, 0, 1)
	return buildFixture(t, m, pred, nil)
}

func TestParsePredictedFlagPyMorphy(t *testing.T) {
	d := predictionFixture(t)

	exact := d.Parse("кота")
	require.NotEmpty(t, exact)
	for _, r := range exact {
		assert.False(t, r.Predicted, "dictionary reading %+v", r)
	}

	predicted := d.Parse("котёнка")
	require.NotEmpty(t, predicted)
	for _, r := range predicted {
		assert.True(t, r.Predicted, "predicted reading %+v", r)
	}
}

// Every Builder dictionary gets a prediction DAWG (buildFromEntries calls
// BuildPrediction), so even a 4-word dictionary "predicts" "бота" from "кота".
func TestParsePredictedFlagBuilder(t *testing.T) {
	d := buildSmallDict(t)

	for _, r := range d.Parse("кота") {
		assert.False(t, r.Predicted, "dictionary reading %+v", r)
	}
	predicted := d.Parse("бота")
	require.NotEmpty(t, predicted, "builder dictionaries predict unknown words")
	for _, r := range predicted {
		assert.True(t, r.Predicted, "predicted reading %+v", r)
	}
}

func TestLemmaPredictedFlag(t *testing.T) {
	d := buildSmallDict(t)

	refs := d.Lemma("кота")
	require.Len(t, refs, 1)
	assert.False(t, refs[0].Predicted)

	refs = d.Lemma("бота")
	require.NotEmpty(t, refs)
	for _, ref := range refs {
		assert.True(t, ref.Predicted, "%+v", ref)
	}
}

// botDict knows "бот"/"бота" exactly; buildSmallDict only predicts "бота".
func botDict(t *testing.T) *morphology.Dictionary {
	t.Helper()
	return buildFromTriples(t,
		[3]string{"бот", "бот", "NOUN,inan,masc,sing,nomn"},
		[3]string{"бота", "бот", "NOUN,inan,masc,sing,gent"},
	)
}

func TestMultiDictionaryParseMixesPredictedAndExact(t *testing.T) {
	m := morphology.NewMultiDictionary(buildSmallDict(t), botDict(t))

	var sawPredicted, sawExact bool
	for _, r := range m.Parse("бота") {
		switch r.Dict {
		case 0:
			assert.True(t, r.Predicted, "dict 0 does not know бота: %+v", r)
			sawPredicted = true
		case 1:
			assert.False(t, r.Predicted, "dict 1 knows бота: %+v", r)
			sawExact = true
		}
	}
	assert.True(t, sawPredicted)
	assert.True(t, sawExact)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ -run 'Predicted' -count=1`
Expected: FAIL — compile error `r.Predicted undefined (type morphology.Reading has no field or method Predicted)`.

- [ ] **Step 3: Implement**

In `pkg/morphology/parse.go`, append the field to `Reading` (keep existing fields and comments):

```go
	Prob   float64 // probability of this reading (0 if probability data is unavailable)
	// Predicted is true when the reading was produced by suffix prediction
	// (the word is absent from the dictionary), false for dictionary
	// readings. Parse returns either only dictionary readings or only
	// predicted ones for a given word; see also Dictionary.IsKnown.
	Predicted bool
}
```

Update the `Parse` doc comment's second sentence to: `For out-of-dictionary words, it tries to
predict readings from the prediction-DAWG (suffixes); such readings have Predicted set.`

In `predictForPrefix`, directly after `r := x.readingForm(predictionShard, wordStart+it.Key, paraNum, form)`:

```go
				r.Predicted = true
```

In `pkg/morphology/lemma.go`:

```go
type LemmaRef struct {
	Normal string // lemma (base form)
	Tag    string // tag of the lemma (paradigm form 0)
	Para   uint16 // paradigm id — unique only together with Shard
	Shard  int    // dictionary shard index; always 0 for unsharded dictionaries
	Dict   int    // dictionary index in MultiDictionary; always 0 for Dictionary.Lemma directly
	// Predicted is true when the lemma comes from predicted readings (the
	// word is absent from the dictionary). A single Dictionary.Parse never
	// mixes predicted and dictionary readings, and MultiDictionary.Lemma
	// never merges refs across dictionaries, so every reading behind one
	// LemmaRef shares this value.
	Predicted bool
}
```

and in `Lemma`:

```go
		out = append(out, LemmaRef{Normal: r.Normal, Tag: tag, Para: r.Para, Shard: r.Shard, Predicted: r.Predicted})
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./pkg/morphology/... -count=1`
Expected: PASS (all existing tests too — `readingsSnapshot`/`semanticSnapshot` compare whole
`Reading` values, and the flag is identical on both sides).

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/parse.go pkg/morphology/lemma.go pkg/morphology/known_test.go
git commit -m "feat(morphology): Reading.Predicted and LemmaRef.Predicted

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: `Dictionary.IsKnown` / `MultiDictionary.IsKnown` (item A, part 2)

**Files:**
- Create: `pkg/morphology/known.go`
- Modify: `pkg/morphology/known_test.go` (append), `pkg/morphology/example_test.go` (append)

**Interfaces:**
- Consumes: `Reading.Predicted` (Task 1) — only in tests.
- Produces: `func (x *Dictionary) IsKnown(word string) bool`,
  `func (m *MultiDictionary) IsKnown(word string) bool`.

- [ ] **Step 1: Write the failing tests**

Append to `pkg/morphology/known_test.go`:

```go
func TestIsKnown(t *testing.T) {
	d := buildSmallDict(t)

	assert.True(t, d.IsKnown("кота"))
	assert.True(t, d.IsKnown("КОТА"), "input is lower-cased like Parse")
	assert.False(t, d.IsKnown("бота"), "predicted by Parse, but not in the dictionary")
	assert.NotEmpty(t, d.Parse("бота"))
	assert.False(t, d.IsKnown(""))

	var nilDict *morphology.Dictionary
	assert.False(t, nilDict.IsKnown("кот"))
}

// parseDict's pymorphy2 fixture stores "ёж" and uses RussianCharPolicy:
// IsKnown applies the same е→ё substitution as Parse.
func TestIsKnownAppliesCharPolicy(t *testing.T) {
	d := parseDict(t)
	assert.True(t, d.IsKnown("еж"))
	assert.True(t, d.IsKnown("ёж"))
}

// IsKnown(w) is true exactly when Parse(w) returns non-predicted readings.
func TestIsKnownAgreesWithParse(t *testing.T) {
	d := predictionFixture(t)
	for _, w := range []string{"кот", "кота", "мышь", "ежик", "котёнка", "неттакогослова"} {
		rs := d.Parse(w)
		known := len(rs) > 0 && !rs[0].Predicted
		assert.Equal(t, known, d.IsKnown(w), w)
	}
}

func TestMultiDictionaryIsKnown(t *testing.T) {
	m := morphology.NewMultiDictionary(buildSmallDict(t), botDict(t))
	assert.True(t, m.IsKnown("кота"))
	assert.True(t, m.IsKnown("бота"))
	assert.False(t, m.IsKnown("зебра"))
	assert.False(t, morphology.NewMultiDictionary().IsKnown("кот"))
}
```

Append to `pkg/morphology/example_test.go`:

```go
// ExampleDictionary_IsKnown shows telling a dictionary word from an unknown
// one without looking at readings (Parse would predict readings for many
// unknown words; IsKnown never predicts).
func ExampleDictionary_IsKnown() {
	d := mustCompileExampleDict()

	fmt.Println(d.IsKnown("Кота"), d.IsKnown("бота"))
	// Output:
	// true false
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ -run 'IsKnown' -count=1`
Expected: FAIL — `d.IsKnown undefined`.

- [ ] **Step 3: Implement**

Create `pkg/morphology/known.go`:

```go
package morphology

import "strings"

// IsKnown reports whether word has at least one dictionary reading. It runs
// the same lookup Parse runs first — the input is lower-cased, the
// dictionary's CharPolicy is applied (е→ё for Russian), every shard is
// searched — but never falls back to prediction: IsKnown(w) is true exactly
// when Parse(w) returns readings with Predicted == false. It does not build
// Reading values.
func (x *Dictionary) IsKnown(word string) bool {
	if x == nil || x.d == nil {
		return false
	}
	word = strings.ToLower(word)
	for _, dawg := range x.d.Words {
		if dawg == nil {
			continue
		}
		for _, it := range dawg.SimilarItems(word, x.d.CharPolicy, x.d.Alphabet) {
			for _, v := range it.Values {
				if len(v) >= 4 { // the guard Dictionary.reading applies
					return true
				}
			}
		}
	}
	return false
}

// IsKnown reports whether any dictionary in the set knows word (see
// Dictionary.IsKnown). Dictionaries are checked in registration order and
// the search stops at the first hit.
func (m *MultiDictionary) IsKnown(word string) bool {
	for _, d := range m.dicts {
		if d.IsKnown(word) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./pkg/morphology/ -run 'IsKnown|Predicted' -count=1 -v`
Expected: PASS, including `ExampleDictionary_IsKnown`.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/known.go pkg/morphology/known_test.go pkg/morphology/example_test.go
git commit -m "feat(morphology): Dictionary.IsKnown and MultiDictionary.IsKnown

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: CLI `lookup` marks predicted readings (item A, part 3)

**Files:**
- Modify: `cmd/gomorphy/lookup.go:12-25`
- Modify: `cmd/gomorphy/lookup_test.go`
- Modify: `docs/en/cli.md:36-44`, `docs/ru/cli.md` (the matching `lookup` format paragraph)

**Interfaces:**
- Consumes: `Reading.Predicted` (Task 1).
- Produces: `lookup` output line = `<word>\t<lemma>\t<tag>\tpara#<dict>/<shard>/<para>\t<dict label>` plus
  `\t(predicted)` for predicted readings. Existing columns are unchanged, so scripts reading the
  first five columns keep working.

- [ ] **Step 1: Write the failing test**

Append to `cmd/gomorphy/lookup_test.go` (add `"path/filepath"` and
`"github.com/amarin/gomorphy/pkg/morphology"` to its imports):

```go
// buildBuilderDat saves a tiny Builder dictionary (кот/кота). Builder
// dictionaries carry a prediction DAWG, so "бота" is predicted.
func buildBuilderDat(t *testing.T) string {
	t.Helper()
	b := morphology.NewBuilder(morphology.BuilderOptions{})
	require.NoError(t, b.AddLemma("кот", "NOUN,anim,masc,sing,nomn"))
	require.NoError(t, b.AddForm("кота", "кот", "NOUN,anim,masc,sing,gent"))
	d, err := b.Build()
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "builder.dat")
	require.NoError(t, d.SaveTo(path))
	return path
}

func TestDoLookup_MarksPredicted(t *testing.T) {
	m := mustResolveOne(t, buildBuilderDat(t))
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	require.NoError(t, doLookup(&buf, m, "бота"))
	assert.Contains(t, buf.String(), "\t(predicted)\n")

	buf.Reset()
	require.NoError(t, doLookup(&buf, m, "кота"))
	assert.NotContains(t, buf.String(), "(predicted)")
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./cmd/gomorphy/ -run TestDoLookup_MarksPredicted -count=1`
Expected: FAIL — `"кота\tкот\t…"` output does not contain `"\t(predicted)\n"` for «бота».

- [ ] **Step 3: Implement**

In `cmd/gomorphy/lookup.go` replace the loop and extend the doc comment:

```go
// doLookup writes every reading of word to w, tab-separated: Word, Normal,
// Tag, a para#Dict/Shard/Para composite (Dict is always 0 for a
// single-dictionary resolution, since resolveDictionaries always produces
// a MultiDictionary), and the source dictionary's name/version (dictLabel).
// Readings produced by suffix prediction (the word is not in the
// dictionary) get one more column, "(predicted)".
func doLookup(w io.Writer, m *morphology.MultiDictionary, word string) error {
	readings := m.Parse(word)
	if len(readings) == 0 {
		return fmt.Errorf("lookup %q: no readings", word)
	}
	for _, r := range readings {
		marker := ""
		if r.Predicted {
			marker = "\t(predicted)"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\tpara#%d/%d/%d\t%s%s\n",
			r.Word, r.Normal, r.Tag, r.Dict, r.Shard, r.Para, dictLabel(m, r.Dict), marker)
	}
	return nil
}
```

In `docs/en/cli.md` replace the "Format:" paragraph of `lookup` with:

```markdown
Format: `<word>\t<lemma>\t<tag>\tpara#<dict>/<shard>/<para>\t<dictionary>` — the `dict`
component indicates which merged dictionary (0-based) the reading came
from, when multiple `-d` flags were given; `<dictionary>` is its name and
version. A reading guessed by suffix prediction (the word is not in the
dictionary) ends with one more column, `(predicted)`.
```

Make the equivalent edit in `docs/ru/cli.md` (Russian): «Разбор, угаданный по окончанию (слова нет
в словаре), заканчивается ещё одной колонкой `(predicted)`.»

- [ ] **Step 4: Run the tests**

Run: `go test ./cmd/gomorphy/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/gomorphy/lookup.go cmd/gomorphy/lookup_test.go docs/en/cli.md docs/ru/cli.md
git commit -m "feat(cli): lookup marks predicted readings

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: `OpenBytes` (item B)

**Files:**
- Modify: `pkg/morphology/open.go:155-176`
- Modify: `pkg/morphology/dictionary.go:40-42` (type doc)
- Create: `pkg/morphology/open_bytes_test.go`
- Modify: `pkg/morphology/example_test.go` (append)

**Interfaces:**
- Produces: `func OpenBytes(data []byte) (*Dictionary, error)`; unexported
  `func openBytes(data []byte) (*internal.Dictionary, error)` used by `Open` and `OpenBytes`.

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/open_bytes_test.go`:

```go
package morphology_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// savedSmallDict saves buildSmallDict to a temp file and returns the path
// and the file's bytes.
func savedSmallDict(t *testing.T) (string, []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "small.dat")
	require.NoError(t, buildSmallDict(t).SaveTo(path))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return path, data
}

func TestOpenBytesMatchesOpen(t *testing.T) {
	path, data := savedSmallDict(t)

	fromFile, err := morphology.Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, fromFile.Close()) }()

	fromBytes, err := morphology.OpenBytes(data)
	require.NoError(t, err)

	words := []string{"кот", "кота", "мышь", "мыши", "бота"}
	cmpSnapshots(t, readingsSnapshot(fromFile, words...), readingsSnapshot(fromBytes, words...))
	assert.Equal(t, fromFile.Fuzzy("кот", 1), fromBytes.Fuzzy("кот", 1))
	assert.Equal(t, fromFile.Info(), fromBytes.Info())
	assert.Equal(t, fromFile.TagSetName(), fromBytes.TagSetName())
	assert.NoError(t, fromBytes.Close(), "Close is a no-op for OpenBytes")
}

func TestOpenBytesChecksumMismatch(t *testing.T) {
	_, data := savedSmallDict(t)
	bad := slices.Clone(data)
	bad[len(bad)-1] ^= 0xFF

	_, err := morphology.OpenBytes(bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestOpenBytesRejectsGarbage(t *testing.T) {
	_, err := morphology.OpenBytes(nil)
	require.Error(t, err)
	_, err = morphology.OpenBytes([]byte("not a dictionary at all"))
	require.Error(t, err)
}

// //go:embed data has no alignment guarantee. A buffer shifted by one byte
// makes every section misaligned, so ParseDAWG takes its copying path.
func TestOpenBytesMisaligned(t *testing.T) {
	_, data := savedSmallDict(t)
	buf := make([]byte, len(data)+1)
	copy(buf[1:], data)

	d, err := morphology.OpenBytes(buf[1:])
	require.NoError(t, err)
	rs := d.Parse("кота")
	require.Len(t, rs, 1)
	assert.Equal(t, "кот", rs[0].Normal)
	assert.False(t, rs[0].Predicted)
}
```

Append to `pkg/morphology/example_test.go`:

```go
// ExampleOpenBytes shows opening a dictionary that is already in memory —
// typically one embedded into the binary with //go:embed — without writing
// it to disk and without mmap.
func ExampleOpenBytes() {
	path := filepath.Join(os.TempDir(), "gomorphy-example-bytes.dat")
	if err := mustCompileExampleDict().SaveTo(path); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.Remove(path) }()

	data, err := os.ReadFile(path) // with //go:embed: var data []byte
	if err != nil {
		log.Fatal(err)
	}
	d, err := morphology.OpenBytes(data)
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range d.Parse("кота") {
		fmt.Println(r.Normal, r.Tag)
	}
	// Output:
	// кот NOUN,anim,masc,sing,gent
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ -run 'OpenBytes' -count=1`
Expected: FAIL — `undefined: morphology.OpenBytes`.

- [ ] **Step 3: Implement**

In `pkg/morphology/open.go` replace `Open` and add `OpenBytes`/`openBytes` right after it:

```go
// Open loads a dictionary from a GMOR file (the single on-disk format,
// SaveTo). Hot sections (words.dawg) are mapped via mmap without copying;
// the result must be closed with the Close method (see Close for the
// lifecycle rules). Not supported on Windows yet — use OpenBytes there.
func Open(path string) (*Dictionary, error) {
	mm, err := mmapx.Open(path)
	if err != nil {
		return nil, fmt.Errorf("morphology: open %s: %w", path, err)
	}
	d, err := openBytes(mm.Bytes())
	if err != nil {
		_ = mm.Close()
		return nil, fmt.Errorf("morphology: open %s: %w", path, err)
	}
	return &Dictionary{d: d, mm: mm}, nil
}

// OpenBytes opens a dictionary in the GMOR format (as written by SaveTo)
// from data — typically a file embedded with //go:embed. The checksum is
// verified like Open. data is not copied: DAWG sections whose unit array
// is 4-byte aligned in memory are used in place (zero-copy), misaligned
// ones are copied (//go:embed gives no alignment guarantee, so expect
// copies there). data must stay unmodified and reachable for as long as
// the Dictionary is used; Close releases nothing. Works on every platform,
// Windows included (no mmap involved).
func OpenBytes(data []byte) (*Dictionary, error) {
	d, err := openBytes(data)
	if err != nil {
		return nil, fmt.Errorf("morphology: open bytes: %w", err)
	}
	return &Dictionary{d: d}, nil
}

// openBytes validates a GMOR container (magic, version, checksum, catalog)
// and assembles the internal dictionary from its sections. The result may
// alias data (see internal.ParseDAWG).
func openBytes(data []byte) (*internal.Dictionary, error) {
	cont, err := internal.OpenContainer(data)
	if err != nil {
		return nil, err
	}
	return parseContainer(cont)
}
```

In `pkg/morphology/dictionary.go` replace the `Dictionary` doc comment with:

```go
// Dictionary is an immutable dictionary: loaded by an importer, built with
// Builder/ImportTSV/Merge, opened from a GMOR file (Open, mmap-backed) or
// from memory (OpenBytes). Dictionaries opened via Open must be closed with
// Close when no longer needed; see Close for when that is safe.
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./pkg/morphology/ -count=1`
Expected: PASS, including `ExampleOpenBytes`.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/open.go pkg/morphology/dictionary.go pkg/morphology/open_bytes_test.go pkg/morphology/example_test.go
git commit -m "feat(morphology): OpenBytes opens a dictionary from memory

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Builder and ImportTSV lower-case words and lemmas (item C, part 1)

**Files:**
- Modify: `pkg/morphology/builder.go:58-83` (AddForm)
- Modify: `pkg/morphology/import_tsv.go:23-39` (doc), `:70-76` (loop)
- Create: `pkg/morphology/case_test.go`

**Interfaces:**
- Consumes: `Reading.Predicted`, `Dictionary.IsKnown` (Tasks 1–2) in tests.
- Produces: behaviour — every word/lemma stored by Builder/ImportTSV is `strings.ToLower`ed.

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/case_test.go`:

```go
package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tagGeoxNomn = "NOUN,inan,femn,Sgtm,Geox,sing,nomn"
	tagGeoxGent = "NOUN,inan,femn,Sgtm,Geox,sing,gent"
)

// Before the fix «Москва» was stored verbatim and Parse (which lower-cases
// its input) only reached it through prediction — see the plan's
// "Planning-time findings" #2. The assertions on Predicted are the point.
func TestBuilderLowercasesWordAndLemma(t *testing.T) {
	b := morphology.NewBuilder(morphology.BuilderOptions{})
	require.NoError(t, b.AddLemma("Москва", tagGeoxNomn))
	require.NoError(t, b.AddForm("Москвы", "Москва", tagGeoxGent))
	d, err := b.Build()
	require.NoError(t, err)

	for _, q := range []string{"москва", "Москва", "МОСКВА"} {
		rs := d.Parse(q)
		require.Len(t, rs, 1, q)
		assert.Equal(t, "москва", rs[0].Word, q)
		assert.Equal(t, "москва", rs[0].Normal, q)
		assert.Equal(t, tagGeoxNomn, rs[0].Tag, "tags are stored verbatim")
		assert.False(t, rs[0].Predicted, q)
		assert.True(t, d.IsKnown(q), q)
	}

	rs := d.Parse("Москвы")
	require.Len(t, rs, 1)
	assert.Equal(t, "москва", rs[0].Normal)
	assert.False(t, rs[0].Predicted)
}

func TestBuilderCaseVariantsCollapse(t *testing.T) {
	b := morphology.NewBuilder(morphology.BuilderOptions{})
	require.NoError(t, b.AddLemma("Кот", "NOUN,anim,masc,sing,nomn"))
	require.NoError(t, b.AddLemma("кот", "NOUN,anim,masc,sing,nomn"))
	d, err := b.Build()
	require.NoError(t, err)
	assert.Len(t, d.Parse("кот"), 1, "«Кот» and «кот» are the same entry after lower-casing")
}

func TestImportTSVLowercases(t *testing.T) {
	d := importTSVString(t, "Москва\tМосквы\t"+tagGeoxGent+"\n")

	rs := d.Parse("москвы")
	require.Len(t, rs, 1)
	assert.Equal(t, "москвы", rs[0].Word)
	assert.Equal(t, "москва", rs[0].Normal)
	assert.False(t, rs[0].Predicted)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ -run 'Lowercases|CaseVariants' -count=1`
Expected: FAIL — e.g. `Should be false` on `rs[0].Predicted` for «москва» (the prediction
fallback), and `"Москвы" != "москвы"`-style mismatches for TSV.

- [ ] **Step 3: Implement**

In `pkg/morphology/builder.go`, `AddForm`:

```go
// AddForm registers one entry: wordform text "word", its lemma "lemma",
// and an opaque grammeme tag "tag". tag may be "" (a reading with no
// grammemes). An empty word is an error (as are whitespace-only words, per
// the TSV trim rule). An empty lemma means the wordform is its own lemma
// (auto-lemma). word and lemma are lower-cased (strings.ToLower): Parse
// lower-cases its input, so a mixed-case form would otherwise be
// unreachable. tag is stored verbatim.
func (b *Builder) AddForm(word, lemma, tag string) error {
	if b.closed {
		return ErrBuilderClosed
	}
	if strings.TrimSpace(word) == "" {
		return fmt.Errorf("morphology: builder: word must not be empty")
	}
	if lemma == "" {
		lemma = word
	}
	entry := internal.BuildEntry{Word: strings.ToLower(word), Lemma: strings.ToLower(lemma), Tag: tag}
	if b.seen == nil {
		b.seen = make(map[internal.BuildEntry]bool)
	}
	if b.seen[entry] {
		return nil
	}
	b.seen[entry] = true
	b.entries = append(b.entries, entry)
	return nil
}
```

In `pkg/morphology/import_tsv.go` replace the last doc sentence
`Input is never lower-cased and tags are never normalized; one entry per line is fully general.`
with `The wordform and lemma columns are lower-cased (as Builder.AddForm does); tags are stored
verbatim. One entry per line is fully general.` and change the append to:

```go
		entries = append(entries, internal.BuildEntry{
			Word:  strings.ToLower(word),
			Lemma: strings.ToLower(lemma),
			Tag:   tag,
		})
```

(ImportTSV builds entries itself and never calls `AddForm` — it needs its own lower-casing; see
Discrepancy D-11.)

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/... ./cmd/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/builder.go pkg/morphology/import_tsv.go pkg/morphology/case_test.go
git commit -m "fix(morphology): Builder and ImportTSV lower-case words and lemmas

Parse lower-cases its input, so mixed-case forms were reachable only
through prediction.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: `Fuzzy`/`FuzzyTop` lower-case the query (item C, part 2)

**Files:**
- Modify: `pkg/morphology/fuzzy.go:34-60`
- Modify: `pkg/morphology/case_test.go` (append)

**Interfaces:**
- Produces: behaviour — `Fuzzy`/`FuzzyTop` (and via delegation the `MultiDictionary` variants)
  apply `strings.ToLower` to the query.

- [ ] **Step 1: Write the failing tests**

Append to `pkg/morphology/case_test.go`:

```go
func TestFuzzyLowercasesQuery(t *testing.T) {
	d := fuzzyDict(t)

	got := d.Fuzzy("КОТ", 0)
	require.Len(t, got, 1)
	assert.Equal(t, "кот", got[0].Word)
	assert.Equal(t, d.Fuzzy("кот", 1), d.Fuzzy("Кот", 1))
	assert.Equal(t, d.FuzzyTop("кот", 3), d.FuzzyTop("КОТ", 3))

	m := morphology.NewMultiDictionary(d)
	assert.Equal(t, m.Fuzzy("кот", 1), m.Fuzzy("КОТ", 1))
	assert.Equal(t, m.FuzzyTop("кот", 3), m.FuzzyTop("КОТ", 3))
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./pkg/morphology/ -run TestFuzzyLowercasesQuery -count=1`
Expected: FAIL — `"[]" should have 1 item(s)` for `Fuzzy("КОТ", 0)`.

- [ ] **Step 3: Implement**

In `pkg/morphology/fuzzy.go` add `"strings"` to the imports; make `word = strings.ToLower(word)`
the first statement of both `Fuzzy` and `FuzzyTop`, and add to both doc comments:
`The query is lower-cased, like Parse's input.`

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/fuzzy.go pkg/morphology/case_test.go
git commit -m "fix(morphology): Fuzzy and FuzzyTop lower-case the query

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: Public `CharPolicy`, `BuilderOptions.CharPolicy` (item D, part 1)

**Files:**
- Create: `pkg/morphology/char_policy.go`
- Modify: `pkg/morphology/builder.go:26-35` (BuilderOptions), `:113-118` (buildFromEntries)
- Modify: `pkg/morphology/open.go:14-17` (UniMorphOptions doc)
- Create: `pkg/morphology/char_policy_test.go`

**Interfaces:**
- Produces:
  - `type Substitution = internal.Substitution` (`struct{ From, To rune }`)
  - `type CharPolicy = internal.CharPolicy` (`struct{ Substitutions []Substitution }`, method
    `Substitute(r rune) (rune, bool)`)
  - `func NewCharPolicy(subs ...Substitution) *CharPolicy`
  - `func RussianCharPolicy() *CharPolicy` — е→ё
  - `func NoCharPolicy() *CharPolicy` — explicit empty policy
  - `BuilderOptions.CharPolicy *CharPolicy` — nil → `defaultCharPolicy(language)`
  - unexported `func defaultCharPolicy(language string) *CharPolicy` — `"ru"` or `""` → Russian
    (е→ё), any other language → `NoCharPolicy()` (owner decision 2026-09-24, answer 16; applies to
    Builder and ImportTSV alike)

Why aliases, not a new struct: `UniMorphOptions` is `= unimorph.Options`, whose field is
`CharPolicy *internal.CharPolicy` (`importers/unimorph/import.go:37`). The `unimorph` package cannot
import `morphology` (cycle: `morphology` imports `unimorph`), so its field can never be
`*morphology.CharPolicy`. With `type CharPolicy = internal.CharPolicy` the existing field already
has the public type, and callers can write `morphology.UniMorphOptions{CharPolicy:
morphology.NoCharPolicy()}` — no change to `unimorph`, fully source-compatible.

Why an explicit empty policy is needed: `internal.BuildDictionaryFromEntries` maps a nil policy to
`RussianCharPolicy()` (`internal/build.go:259-262`), and a non-nil policy with zero substitutions
is stored as-is (`EncodeMeta` writes count 0, `DecodeMeta` returns a non-nil empty policy).

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/char_policy_test.go`:

```go
package morphology_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tagFemnNomn = "NOUN,inan,femn,sing,nomn"

func yolkaDict(t *testing.T, opts morphology.BuilderOptions) *morphology.Dictionary {
	t.Helper()
	b := morphology.NewBuilder(opts)
	require.NoError(t, b.AddLemma("ёлка", tagFemnNomn))
	d, err := b.Build()
	require.NoError(t, err)
	return d
}

func TestBuilderDefaultCharPolicyIsRussian(t *testing.T) {
	d := yolkaDict(t, morphology.BuilderOptions{})

	assert.True(t, d.IsKnown("елка"))
	rs := d.Parse("елка")
	require.Len(t, rs, 1)
	assert.Equal(t, "ёлка", rs[0].Word)
	assert.False(t, rs[0].Predicted)
}

func TestBuilderNoCharPolicy(t *testing.T) {
	d := yolkaDict(t, morphology.BuilderOptions{CharPolicy: morphology.NoCharPolicy()})

	assert.True(t, d.IsKnown("ёлка"))
	assert.False(t, d.IsKnown("елка"))
	for _, r := range d.Parse("елка") {
		assert.True(t, r.Predicted, "only a prediction may answer «елка»: %+v", r)
	}
}

func TestBuilderCharPolicySurvivesSaveOpen(t *testing.T) {
	d := yolkaDict(t, morphology.BuilderOptions{CharPolicy: morphology.NoCharPolicy()})
	path := filepath.Join(t.TempDir(), "nopolicy.dat")
	require.NoError(t, d.SaveTo(path))

	got, err := morphology.Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, got.Close()) }()
	assert.False(t, got.IsKnown("елка"))
	assert.True(t, got.IsKnown("ёлка"))
}

func TestBuilderCustomCharPolicy(t *testing.T) {
	pol := morphology.NewCharPolicy(morphology.Substitution{From: 'и', To: 'і'})
	b := morphology.NewBuilder(morphology.BuilderOptions{CharPolicy: pol})
	require.NoError(t, b.AddLemma("міръ", "NOUN,inan,masc,sing,nomn"))
	d, err := b.Build()
	require.NoError(t, err)

	assert.True(t, d.IsKnown("миръ"))
	assert.False(t, d.IsKnown("елка"))
}

// The default policy is chosen by language: е→ё for "ru" and for an empty
// Language (which means "ru"), no substitutions for any other language.
func TestDefaultCharPolicyByLanguage(t *testing.T) {
	const tsv = "ёлка\tёлка\t" + tagFemnNomn + "\n"
	cases := []struct {
		language string
		yo       bool
	}{{"", true}, {"ru", true}, {"en", false}, {"uk", false}}
	for _, c := range cases {
		t.Run("language="+c.language, func(t *testing.T) {
			b := yolkaDict(t, morphology.BuilderOptions{Language: c.language})
			assert.Equal(t, c.yo, b.IsKnown("елка"), "Builder")
			assert.True(t, b.IsKnown("ёлка"), "Builder")

			d, err := morphology.ImportTSV(strings.NewReader(tsv), morphology.BuilderOptions{Language: c.language})
			require.NoError(t, err)
			assert.Equal(t, c.yo, d.IsKnown("елка"), "ImportTSV")
			assert.True(t, d.IsKnown("ёлка"), "ImportTSV")
		})
	}
}

// An explicit policy wins over the language default.
func TestExplicitCharPolicyOverridesLanguage(t *testing.T) {
	d := yolkaDict(t, morphology.BuilderOptions{Language: "en", CharPolicy: morphology.RussianCharPolicy()})
	assert.True(t, d.IsKnown("елка"))
}

func TestUniMorphOptionsCharPolicyFromOutside(t *testing.T) {
	d, err := morphology.CompileFromUniMorph(strings.NewReader("ёж\tёж\tN;NOM;SG\n"),
		morphology.UniMorphOptions{Language: "ru", CharPolicy: morphology.NoCharPolicy()})
	require.NoError(t, err)
	assert.True(t, d.IsKnown("ёж"))
	assert.False(t, d.IsKnown("еж"))
}

func TestCharPolicyConstructors(t *testing.T) {
	to, ok := morphology.RussianCharPolicy().Substitute('е')
	assert.True(t, ok)
	assert.Equal(t, 'ё', to)
	_, ok = morphology.NoCharPolicy().Substitute('е')
	assert.False(t, ok)
	assert.NotNil(t, morphology.NoCharPolicy(), "an explicit empty policy, not nil")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ -run 'CharPolicy' -count=1`
Expected: FAIL — `undefined: morphology.NoCharPolicy` (and the other constructors).

- [ ] **Step 3: Implement**

Create `pkg/morphology/char_policy.go`:

```go
package morphology

import "github.com/amarin/gomorphy/pkg/morphology/internal"

// Substitution is a pair of runes lookup treats as interchangeable in one
// direction: a query rune From also matches a stored rune To. For Russian
// that is е→ё, so «елка» finds «ёлка» (but «ёлка» does not find «елка»).
//
// It is an alias of the internal type so that every options struct that
// already carries a policy (UniMorphOptions.CharPolicy) accepts it as is.
type Substitution = internal.Substitution

// CharPolicy is a dictionary's set of lookup substitutions. It is stored in
// the dictionary ("meta" section) when the dictionary is built and applied
// by Parse, Lemma, IsKnown, Fuzzy and FuzzyTop. Pre-reform orthography
// (ѣ→е, final ъ) is not a CharPolicy concern: normalize such text before
// calling gomorphy.
type CharPolicy = internal.CharPolicy

// NewCharPolicy returns a policy with the given substitutions.
func NewCharPolicy(subs ...Substitution) *CharPolicy { return internal.NewCharPolicy(subs...) }

// RussianCharPolicy returns the Russian policy: е→ё.
func RussianCharPolicy() *CharPolicy { return internal.RussianCharPolicy() }

// NoCharPolicy returns an explicit empty policy: exact rune matching only.
// Unlike a nil *CharPolicy in options, which means "the language default",
// it disables substitutions.
func NoCharPolicy() *CharPolicy { return internal.NewCharPolicy() }

// defaultCharPolicy is the policy Builder and ImportTSV apply when the
// caller leaves BuilderOptions.CharPolicy nil: е→ё for Russian ("ru", and
// "" which means "ru"), no substitutions for any other language.
func defaultCharPolicy(language string) *CharPolicy {
	switch language {
	case "", "ru":
		return RussianCharPolicy()
	default:
		return NoCharPolicy()
	}
}
```

In `pkg/morphology/builder.go`, add the field to `BuilderOptions`:

```go
	// CharPolicy is the lookup substitution policy stored in the built
	// dictionary. nil means the language default: е→ё for "ru" (and for an
	// empty Language, which means "ru"), no substitutions for any other
	// language. Use NoCharPolicy to disable substitutions explicitly, or
	// RussianCharPolicy to get е→ё for another language.
	CharPolicy *CharPolicy
```

and in `buildFromEntries` replace the `internal.BuildDictionaryFromEntries` call's options:

```go
	language := opts.Language
	if language == "" {
		language = "ru" // ImportTSV does not default Language; NewBuilder does
	}
	policy := opts.CharPolicy
	if policy == nil {
		policy = defaultCharPolicy(language)
	}
	d, err := internal.BuildDictionaryFromEntries(internal.BuildOptions{
		Language:   language,
		CharPolicy: policy,
		TagSetName: tagSetName,
	}, entries)
```

In `pkg/morphology/open.go` extend the `UniMorphOptions` doc comment with: `Its CharPolicy field
takes a *CharPolicy (NoCharPolicy, RussianCharPolicy, NewCharPolicy); nil means the language
default.`

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/char_policy.go pkg/morphology/char_policy_test.go pkg/morphology/builder.go pkg/morphology/open.go
git commit -m "feat(morphology): public CharPolicy and BuilderOptions.CharPolicy

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: `Fuzzy` applies the dictionary's CharPolicy (item D, part 2)

**Files:**
- Modify: `pkg/morphology/fuzzy.go` (doc comment `:18-33`, `fuzzyWalk` `:109-136`,
  `fuzzyWalkShard` `:141-156`, `fuzzySearch` `:161-169`, `nextRow` `:266-278`)
- Modify: `pkg/morphology/fuzzy_internal_test.go:40`
- Modify: `pkg/morphology/fuzzy_test.go:40-47` (`TestFuzzyRuneMetricYo`)
- Modify: `pkg/morphology/char_policy_test.go` (append)

**Interfaces:**
- Consumes: `CharPolicy`, `NoCharPolicy` (Task 7).
- Produces: `func fuzzyWalkShard(words *internal.DAWG, alphabet internal.Alphabet, pol *internal.CharPolicy, word string, k int) []FuzzyMatch`.

- [ ] **Step 1: Write/adjust the failing tests**

In `pkg/morphology/fuzzy_test.go` replace `TestFuzzyRuneMetricYo` with:

```go
// The fixture uses RussianCharPolicy (е→ё): a query «е» matches a stored
// «ё» at cost 0, like Parse. The substitution is one-way: a query «ё» still
// costs 1 against a stored «е».
func TestFuzzyRuneMetricYo(t *testing.T) {
	d := fuzzyDict(t)

	assert.Equal(t, map[string]int{"ежик": 0, "ёжик": 0},
		countByDistance(t, d.Fuzzy("ежик", 1)))
	assert.Equal(t, map[string]int{"ёж": 0, "ёжик": 2},
		countByDistance(t, d.Fuzzy("ёж", 2)))
}
```

Append to `pkg/morphology/char_policy_test.go`:

```go
func TestFuzzyAppliesCharPolicy(t *testing.T) {
	ru := yolkaDict(t, morphology.BuilderOptions{})
	got := ru.Fuzzy("елка", 0)
	require.Len(t, got, 1)
	assert.Equal(t, morphology.FuzzyMatch{Word: "ёлка", Distance: 0}, got[0])
	assert.Equal(t, got, ru.FuzzyTop("елка", 1))

	none := yolkaDict(t, morphology.BuilderOptions{CharPolicy: morphology.NoCharPolicy()})
	assert.Empty(t, none.Fuzzy("елка", 0))
	got = none.Fuzzy("елка", 1)
	require.Len(t, got, 1)
	assert.Equal(t, 1, got[0].Distance)
}
```

In `pkg/morphology/fuzzy_internal_test.go:40` change the call to
`fuzzyWalkShard(dawg, alphabet, nil, "код", 1)`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ -run 'Fuzzy' -count=1`
Expected: FAIL — compile error in `fuzzy_internal_test.go` (`too many arguments in call to
fuzzyWalkShard`); once Step 3 lands the old expectation `"ёжик": 1` would also fail.

- [ ] **Step 3: Implement**

In `pkg/morphology/fuzzy.go`:

1. Doc comment of `Fuzzy`: replace `"ё/е" counts as one substitution` with `the dictionary's
   CharPolicy applies: a query rune that the policy substitutes (е for Russian) matches the
   substituted stored rune (ё) at cost 0, exactly as Parse finds «ёлка» for «елка»`.
2. In `fuzzyWalk`: `results[shard] = fuzzyWalkShard(dawg, x.d.Alphabet, x.d.CharPolicy, word, k)`.
3. Replace `fuzzyWalkShard` and the `fuzzySearch` struct:

```go
// fuzzyWalkShard — a single pass of the joint traversal of one DAWG shard
// and banded Levenshtein DP. alphabet is the dictionary's Alphabet (nil
// for raw UTF-8 DAWGs); pol is its CharPolicy (nil = exact runes only).
func fuzzyWalkShard(words *internal.DAWG, alphabet internal.Alphabet, pol *internal.CharPolicy, word string, k int) []FuzzyMatch {
	f := &fuzzySearch{
		words:    words,
		alphabet: alphabet,
		pol:      pol,
		q:        []rune(word),
		k:        k,
		path:     make([]byte, 0, 32),
	}

	row := f.rowFor(0)
	for j := range row {
		row[j] = j
	}
	f.visit(0, 0, row)
	return f.out
}

// fuzzySearch carries the state of a single traversal: rows[depth] is the
// DP row after depth runes of the path, path is the current DAWG path's
// bytes. alphabet decodes path's bytes into runes (nil = raw UTF-8); pol
// makes a substituted query rune match its target at cost 0.
type fuzzySearch struct {
	words    *internal.DAWG
	alphabet internal.Alphabet
	pol      *internal.CharPolicy
	q        []rune
	k        int
	rows     [][]int
	path     []byte
	out      []FuzzyMatch
}
```

4. In `nextRow` replace the cost computation and add the helper:

```go
		cost := 1
		if f.q[j-1] == r || f.substitutes(f.q[j-1], r) {
			cost = 0
		}
```

```go
// substitutes reports whether the CharPolicy lets query rune q match the
// stored rune r (one direction only, like SimilarItems).
func (f *fuzzySearch) substitutes(q, r rune) bool {
	if f.pol == nil {
		return false
	}
	for _, s := range f.pol.Substitutions {
		if s.From == q && s.To == r {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/... -count=1`
Expected: PASS — `TestFuzzyDenseAlphabetMatchesRawResults` stays green (both sides carry the same
policy); the examples (`ко`, `кот`) contain no е, so their outputs are unchanged.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/fuzzy.go pkg/morphology/fuzzy_test.go pkg/morphology/fuzzy_internal_test.go pkg/morphology/char_policy_test.go
git commit -m "feat(morphology): Fuzzy applies the dictionary's CharPolicy

е in the query now matches ё in the dictionary at cost 0 (was 1),
consistent with Parse.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 9: `Dictionary.ContentHash` (item E)

**Files:**
- Modify: `pkg/morphology/save.go` (extract `sections`)
- Create: `pkg/morphology/content_hash.go`
- Modify: `pkg/morphology/dictionary.go:43-46` (cache fields; add `"sync"` import)
- Create: `pkg/morphology/content_hash_test.go`

**Interfaces:**
- Consumes: `OpenBytes` (Task 4) in tests.
- Produces: `func (x *Dictionary) ContentHash() string` (32 lower-case hex chars, xxh3-128);
  unexported `func (x *Dictionary) sections(info *internal.BuildInfo) ([]internal.Section, error)`
  (info == nil omits the `info` section).

Design notes:
- The hash covers exactly what `SaveTo` writes except `info`, in container order, each section
  framed as `name, 0x00, uint64 LE length, data` so that section boundaries cannot shift.
- `Dictionary` keeps no `internal.Container` after `Open` (only `d` and `mm`), so the spec's
  "cheap for Opened dictionaries (sections already located)" does not hold: the hash re-encodes
  the sections — for `words.dawg` that is a full copy (`DAWG.Bytes`). The result is computed once
  and cached (`sync.Once`); `Dictionary` is immutable, so the cache never goes stale. See
  Discrepancy D-1.
- Encoding the same `internal.Dictionary` is deterministic (planning finding #4), and an
  `Open`ed dictionary re-encodes to the bytes it was read from (every decoder round-trips), so the
  hash is equal before and after `SaveTo`/`Open`/`OpenBytes`.

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/content_hash_test.go`:

```go
package morphology_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentHashShape(t *testing.T) {
	h := buildSmallDict(t).ContentHash()
	assert.Len(t, h, 32)
	assert.Regexp(t, `^[0-9a-f]{32}$`, h)

	var nilDict *morphology.Dictionary
	assert.Equal(t, "", nilDict.ContentHash())
}

func TestContentHashStableAcrossSaveOpen(t *testing.T) {
	d := buildSmallDict(t)
	want := d.ContentHash()

	path := filepath.Join(t.TempDir(), "a.dat")
	require.NoError(t, d.SaveTo(path))

	opened, err := morphology.Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, opened.Close()) }()
	assert.Equal(t, want, opened.ContentHash())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	fromBytes, err := morphology.OpenBytes(data)
	require.NoError(t, err)
	assert.Equal(t, want, fromBytes.ContentHash())
}

// Re-saving writes a new BuiltAt into "info"; ContentHash ignores "info",
// including Source.
func TestContentHashIgnoresInfo(t *testing.T) {
	a := morphology.NewBuilder(morphology.BuilderOptions{Source: "a"})
	b := morphology.NewBuilder(morphology.BuilderOptions{Source: "b"})
	for _, bl := range []*morphology.Builder{a, b} {
		require.NoError(t, bl.AddLemma("кот", "NOUN,anim,masc,sing,nomn"))
		require.NoError(t, bl.AddForm("кота", "кот", "NOUN,anim,masc,sing,gent"))
	}
	da, err := a.Build()
	require.NoError(t, err)
	db, err := b.Build()
	require.NoError(t, err)

	require.NotEqual(t, da.Info().Source, db.Info().Source)
	assert.Equal(t, da.ContentHash(), db.ContentHash())

	dir := t.TempDir()
	p1, p2 := filepath.Join(dir, "1.dat"), filepath.Join(dir, "2.dat")
	require.NoError(t, da.SaveTo(p1))
	require.NoError(t, da.SaveTo(p2))
	o1, err := morphology.Open(p1)
	require.NoError(t, err)
	defer func() { _ = o1.Close() }()
	o2, err := morphology.Open(p2)
	require.NoError(t, err)
	defer func() { _ = o2.Close() }()
	assert.Equal(t, o1.ContentHash(), o2.ContentHash())
}

func TestContentHashDeterministicRebuild(t *testing.T) {
	assert.Equal(t, buildSmallDict(t).ContentHash(), buildSmallDict(t).ContentHash())
}

func TestContentHashChangesWithContent(t *testing.T) {
	base := buildFromTriples(t,
		[3]string{"кот", "кот", mergeTagMascNomn},
		[3]string{"кота", "кот", mergeTagMascGent},
	)
	changed := buildFromTriples(t,
		[3]string{"кот", "кот", mergeTagMascNomn},
		[3]string{"коту", "кот", mergeTagMascGent},
	)
	assert.NotEqual(t, base.ContentHash(), changed.ContentHash())

	policy := morphology.NewBuilder(morphology.BuilderOptions{CharPolicy: morphology.NoCharPolicy()})
	require.NoError(t, policy.AddForm("кот", "кот", mergeTagMascNomn))
	require.NoError(t, policy.AddForm("кота", "кот", mergeTagMascGent))
	withoutPolicy, err := policy.Build()
	require.NoError(t, err)
	assert.NotEqual(t, base.ContentHash(), withoutPolicy.ContentHash(), "meta (CharPolicy) is content")
}

func TestContentHashPyMorphy(t *testing.T) {
	d := parseDict(t)
	assert.Len(t, d.ContentHash(), 32)
	assert.Equal(t, d.ContentHash(), d.ContentHash(), "cached value is stable")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/morphology/ -run ContentHash -count=1`
Expected: FAIL — `d.ContentHash undefined`.

- [ ] **Step 3: Implement**

Replace `pkg/morphology/save.go` with:

```go
package morphology

import (
	"fmt"
	"time"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// SaveTo writes the dictionary to a GMOR file — the single on-disk format.
// Sections: meta, info, tagset, prefixes, suffixes-N, paradigms-N,
// words.dawg-N (one set per shard, N starting at 0), prediction-N,
// probability, alphabet (all three if present).
func (x *Dictionary) SaveTo(path string) error {
	if x == nil || x.d == nil {
		return fmt.Errorf("morphology: nil dictionary")
	}

	// info is a copy of x.d.Info (if an importer populated it, e.g.
	// Source), with BuiltAt/LibraryVersion set anew on every save; x.d
	// itself is not mutated (Dictionary is immutable).
	info := internal.BuildInfo{}
	if x.d.Info != nil {
		info = *x.d.Info
	}
	info.BuiltAt = time.Now().UTC()
	info.LibraryVersion = Version

	sections, err := x.sections(&info)
	if err != nil {
		return fmt.Errorf("morphology: SaveTo: %w", err)
	}
	return internal.SaveContainer(path, sections)
}

// sections encodes the dictionary into GMOR sections in container order —
// the single encoder behind SaveTo and ContentHash. info == nil omits the
// "info" section.
//
// Compression is not yet implemented (see docs/en/todo.md, "Stage 17") —
// all sections are written as-is. words.dawg-N will always stay
// CompressionNone: it is aliased from mmap without copying, while a
// compressed section would require full decompression into memory on
// load.
func (x *Dictionary) sections(info *internal.BuildInfo) ([]internal.Section, error) {
	const noCompression = internal.CompressionNone
	sections := []internal.Section{
		{Name: "meta", Data: internal.EncodeMeta(x.d.Language, x.d.CharPolicy), Flags: noCompression},
	}
	if info != nil {
		sections = append(sections, internal.Section{Name: "info", Data: internal.EncodeBuildInfo(info), Flags: noCompression})
	}
	sections = append(sections,
		internal.Section{Name: "tagset", Data: internal.EncodeTagSet(x.d.TagSet), Flags: noCompression},
		internal.Section{Name: "prefixes", Data: internal.EncodeStrings(x.d.Prefixes), Flags: noCompression},
	)
	for i, suffixes := range x.d.Suffixes {
		sections = append(sections, internal.Section{
			Name: fmt.Sprintf("suffixes-%d", i), Data: internal.EncodeStrings(suffixes), Flags: noCompression,
		})
	}
	for i, paradigms := range x.d.Paradigms {
		sections = append(sections, internal.Section{
			Name: fmt.Sprintf("paradigms-%d", i), Data: internal.EncodeParadigms(paradigms), Flags: noCompression,
		})
	}
	for i, words := range x.d.Words {
		sections = append(sections, internal.Section{
			Name: fmt.Sprintf("words.dawg-%d", i), Data: words.Bytes(), Flags: internal.CompressionNone,
		})
	}
	for i, pred := range x.d.Prediction {
		if pred == nil {
			continue
		}
		sections = append(sections, internal.Section{
			Name: fmt.Sprintf("prediction-%d", i), Data: pred.Bytes(), Flags: noCompression,
		})
	}
	if x.d.Probability != nil {
		sections = append(sections, internal.Section{
			Name: "probability", Data: x.d.Probability.Bytes(), Flags: noCompression,
		})
	}
	if x.d.Alphabet != nil {
		data, err := internal.EncodeAlphabet(x.d.Alphabet)
		if err != nil {
			return nil, err
		}
		sections = append(sections, internal.Section{
			Name: "alphabet", Data: data, Flags: noCompression,
		})
	}
	return sections, nil
}
```

In `pkg/morphology/dictionary.go` add `"sync"` to the imports and extend the struct:

```go
type Dictionary struct {
	d  *internal.Dictionary
	mm *mmapx.Region

	hashOnce sync.Once // guards hash; see ContentHash
	hash     string
}
```

Create `pkg/morphology/content_hash.go`:

```go
package morphology

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/zeebo/xxh3"
)

// ContentHash returns a stable hex digest (xxh3-128, 32 lower-case hex
// characters) of the dictionary's content: every section SaveTo writes
// except "info" (BuiltAt, LibraryVersion, Source…). Re-saving an unchanged
// dictionary, or opening it with Open/OpenBytes, keeps the digest; two
// dictionaries with equal ContentHash parse every word identically. The
// digest describes the encoding, not only the semantics: the same words
// with a different alphabet or CharPolicy hash differently.
//
// The first call encodes every section (for a large dictionary that is a
// copy of its words DAWG); the result is cached. Returns "" for a nil
// dictionary. Like every other method it must not be called after Close.
func (x *Dictionary) ContentHash() string {
	if x == nil || x.d == nil {
		return ""
	}
	x.hashOnce.Do(func() { x.hash = x.computeContentHash() })
	return x.hash
}

func (x *Dictionary) computeContentHash() string {
	sections, err := x.sections(nil)
	if err != nil {
		return "" // only an Alphabet type EncodeAlphabet cannot write
	}
	h := xxh3.New128()
	var size [8]byte
	for _, s := range sections {
		_, _ = h.WriteString(s.Name)
		_, _ = h.Write([]byte{0})
		binary.LittleEndian.PutUint64(size[:], uint64(len(s.Data)))
		_, _ = h.Write(size[:])
		_, _ = h.Write(s.Data)
	}
	sum := h.Sum128().Bytes()
	return hex.EncodeToString(sum[:])
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./pkg/morphology/... -count=1 -race`
Expected: PASS (existing `save_test.go` round-trip tests confirm `SaveTo` output is unchanged).

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/save.go pkg/morphology/content_hash.go pkg/morphology/content_hash_test.go pkg/morphology/dictionary.go
git commit -m "feat(morphology): Dictionary.ContentHash ignores the info section

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 10: `go` directive by policy — `go 1.25.0` (item F, part 1)

**Files:**
- Modify: `go.mod` (and `go.sum` only if tidy changes it — it should not)
- Modify: `docs/en/installation.md:5`, `docs/ru/installation.md:5`

**Interfaces:** none (build metadata only).

Owner policy (spec item F, open question G3): the `go` directive is the current Go release minus
two minor versions — Go 1.27 now, so `go 1.25.0`, plus `toolchain go1.27.1` for development.
Dependencies are **not** downgraded (`x/sys v0.47.0`, the highest requirement, declares exactly
1.25.0), and `strings.SplitSeq` in `pkg/morphology/parse.go` stays (Go 1.24). If an actual result
differs from "Expected", stop and record it in the task report; do not downgrade dependencies or
lower the directive below 1.25.0 to make it pass.

- [ ] **Step 1: Set the directive**

Run:
```bash
go mod edit -go=1.25.0 -toolchain=go1.27.1
go mod tidy
grep -E '^go |^toolchain|x/sys|x/term|cpuid' go.mod
git diff --stat go.sum
```
Expected:
```
go 1.25.0
toolchain go1.27.1
	golang.org/x/term v0.31.0
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
```
and no change to `go.sum` (the requirement versions are those of 1.1.0; if a commit of Tasks 1–9
bumped one of them, expect that version instead — what matters is that tidy leaves `go 1.25.0`).

- [ ] **Step 2: Confirm the directive is not below what the dependencies need**

Run: `go mod why -m golang.org/x/sys golang.org/x/term`
Expected: both are reached only from `github.com/amarin/gomorphy/cmd/gomorphy` (the library
packages need only `zeebo/xxh3` → `klauspost/cpuid/v2`).

Run (a scratch check, then restore):
```bash
cp go.mod "$TMPDIR/gomorphy-go.mod.bak"
go mod edit -go=1.24.0 && go mod tidy && grep '^go ' go.mod
cp "$TMPDIR/gomorphy-go.mod.bak" go.mod && go mod tidy && grep -E '^go |^toolchain' go.mod
```
Expected: `go 1.25.0` after the 1.24 attempt (tidy raises it back because `x/sys v0.47.0`
declares go 1.25.0), then `go 1.25.0` / `toolchain go1.27.1` again after the restore. So 1.25 is
both the policy value and what the current dependency set allows without downgrades.

- [ ] **Step 3: Check the code against 1.25 semantics**

Run: `go vet ./...`
Expected: no findings. With `go 1.25.0` in `go.mod` the compiler builds every package with
`-lang=go1.25`, and vet's `stdversion` analyzer rejects any stdlib symbol newer than the file's
Go version, so a clean vet is the evidence that nothing needs 1.26+. `strings.SplitSeq` (1.24)
and range-over-func (1.23) are fine.

- [ ] **Step 4: Verify at `go 1.25.0`**

Run: `go build ./... && go test ./... -race -count=1`
Expected: all tests PASS (444 at planning time plus the ones added by Tasks 1–9).

Then, if a Go 1.25 toolchain is available (already in the module cache under
`golang.org/toolchain`, or downloadable, ~70 MB):
`GOTOOLCHAIN=go1.25.0 go test ./... -count=1` (any 1.25.x patch release works; `GOTOOLCHAIN`
overrides the `toolchain` line) — Expected: PASS. If no 1.25 toolchain can be obtained, the
Step 3 vet result (compiler `-lang=go1.25` + `stdversion`) is the evidence; say so in the task
report.

- [ ] **Step 5: Update the documented requirement**

In `docs/en/installation.md` and `docs/ru/installation.md` replace `- Go 1.27.1+` with
`- Go 1.25+`, and add one sentence (in Russian in `docs/ru`): "The `go` directive follows the
policy "current Go release minus two minor versions" and is raised when a new Go minor version
ships."

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum docs/en/installation.md docs/ru/installation.md
git commit -m "build: set the go directive to 1.25.0 (current Go minus two)

Policy: the go directive is the current Go release minus two minor
versions (1.27 now). 1.25.0 needs no dependency downgrades
(golang.org/x/sys v0.47.0, via the CLI's x/term, declares 1.25.0) and
keeps strings.SplitSeq. toolchain go1.27.1 stays for development.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

(If `go.sum` did not change, adding it is a no-op.)

---

### Task 11: `Close` lifecycle documentation and the no-aliasing guarantee (item F, part 2)

**Files:**
- Modify: `pkg/morphology/open.go:178-188` (`Close` doc)
- Modify: `pkg/morphology/multidict.go:189-191` (`MultiDictionary.Close` doc)
- Create: `pkg/morphology/lifecycle_test.go`

**Interfaces:** documentation plus a regression test; no API change.

- [ ] **Step 1: Write the test**

Create `pkg/morphology/lifecycle_test.go`:

```go
package morphology_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReturnedStringsSurviveClose reads every string the query API returned
// after Close has unmapped the file. If any of them pointed into the
// mapping, this would crash with SIGSEGV/SIGBUS. Planning-time audit: only
// the DAWG unit/guide arrays alias the mapping; suffixes, prefixes, tags,
// info and matched keys are heap copies.
func TestReturnedStringsSurviveClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "small.dat")
	require.NoError(t, buildSmallDict(t).SaveTo(path))

	d, err := morphology.Open(path)
	require.NoError(t, err)

	readings := append(d.Parse("кота"), d.Parse("бота")...)
	lemmas := d.Lemma("мыши")
	fuzzy := d.FuzzyTop("кот", 5)
	info := d.Info()
	names := []string{d.Language(), d.TagSetName(), d.ContentHash()}
	require.NoError(t, d.Close())

	var sb strings.Builder
	for _, r := range readings {
		sb.WriteString(r.Word + r.Normal + r.Tag)
	}
	for _, l := range lemmas {
		sb.WriteString(l.Normal + l.Tag)
	}
	for _, m := range fuzzy {
		sb.WriteString(m.Word)
	}
	sb.WriteString(info.Source + info.LibraryVersion)
	for _, n := range names {
		sb.WriteString(n)
	}

	out := sb.String()
	assert.Contains(t, out, "кота")
	assert.Contains(t, out, "мышь")
	assert.Contains(t, out, "NOUN,anim")
}
```

- [ ] **Step 2: Run the test**

Run: `go test ./pkg/morphology/ -run TestReturnedStringsSurviveClose -count=1 -race`
Expected: PASS (this locks in finding #1; it is not expected to fail first).

- [ ] **Step 3: Document the lifecycle**

Replace the `Close` doc comment in `pkg/morphology/open.go`:

```go
// Close releases the mmap region of a dictionary opened with Open. It is a
// no-op for dictionaries that are imported, built (Builder, ImportTSV,
// Merge) or opened with OpenBytes.
//
// Close must not be called while other goroutines may still call methods
// on the Dictionary (directly or through a MultiDictionary): in-flight
// Parse/ParseAppend/Lemma/IsKnown/Fuzzy/FuzzyTop/ContentHash calls read the
// mapping, and unmapping it under them crashes the process with SIGSEGV or
// SIGBUS — not a recoverable panic. Values already returned (Reading,
// LemmaRef, FuzzyMatch, BuildInfo and all their strings) are independent
// copies and stay valid after Close. A caller that swaps dictionaries at
// runtime must retire the old one only after its in-flight calls have
// finished (for example, hold a sync.RWMutex read lock around each call
// and take the write lock before Close). The dictionary must not be used
// after Close.
```

(The `ParseAppend` name in this comment is added in 1.3.0; until then write
`Parse/Lemma/IsKnown/Fuzzy/FuzzyTop/ContentHash`.)

In `pkg/morphology/multidict.go` extend the `MultiDictionary.Close` doc comment with:
`The same rule as Dictionary.Close applies: no calls may be in flight on the set or any of its
dictionaries.`

- [ ] **Step 4: Run all tests**

Run: `go vet ./... && go test ./... -race -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/open.go pkg/morphology/multidict.go pkg/morphology/lifecycle_test.go
git commit -m "docs(morphology): document the Close lifecycle; test returned strings outlive Close

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 12: Documentation, CHANGELOG, version 1.2.0

**Files:**
- Modify: `pkg/morphology/version.go:7`
- Modify: `CHANGELOG.md`
- Modify: `docs/en/library.md`, `docs/ru/library.md`
- Modify: `docs/en/comparison.md:35`
- Modify: `docs/en/todo.md` (Completed stages table; Stage 20 note)
- Create: `docs/en/implementation/ner-support.md`

- [ ] **Step 1: Version**

`pkg/morphology/version.go`: `const Version = "1.2.0"`.

- [ ] **Step 2: CHANGELOG**

Under `## [Unreleased]` insert (fill the date with `date +%F` on the release day):

```markdown
## [1.2.0] - <release date, YYYY-MM-DD>

Support for dictionary-based NER in the lexicon module (see
[implementation/ner-support.md](docs/en/implementation/ner-support.md)).

### Added
- `Reading.Predicted`, `LemmaRef.Predicted`: tell a dictionary reading from a
  suffix-prediction guess. `Dictionary.IsKnown` / `MultiDictionary.IsKnown`:
  exact lookup only, never predicts. CLI `lookup` appends `(predicted)`.
- `OpenBytes(data)`: open a dictionary from memory (e.g. `//go:embed`), no mmap,
  works on Windows.
- Public `CharPolicy`, `Substitution`, `NewCharPolicy`, `RussianCharPolicy`,
  `NoCharPolicy`; `BuilderOptions.CharPolicy`; `UniMorphOptions.CharPolicy` is now
  settable from outside the module.
- `Dictionary.ContentHash()`: a digest of the content that ignores the `info`
  section, stable across re-saves.

### Fixed
- `Builder.AddForm`/`AddLemma` and `ImportTSV` lower-case words and lemmas. Before,
  a form added as «Москва» was reachable only as a prediction.
- `Fuzzy`/`FuzzyTop` lower-case the query.

### Changed
- `Fuzzy`/`FuzzyTop` apply the dictionary's CharPolicy: for Russian, е in the
  query matches ё in the dictionary at distance 0 (was 1).
- The default CharPolicy of Builder/ImportTSV depends on the language: е→ё only
  for `Language` "ru" (an empty `Language` means "ru"). A dictionary built with
  any other `Language` and no explicit `CharPolicy` no longer gets the Russian
  е→ё substitution (before, every built dictionary did); pass
  `RussianCharPolicy()` to keep the old behaviour.
- `go.mod` declares `go 1.25.0` (was 1.27.1) with `toolchain go1.27.1`: the
  `go` directive now follows the policy "current Go minus two minor versions".
  Dependencies are unchanged.
- `Close` documents that it must not run concurrently with other calls on the
  dictionary.
```

- [ ] **Step 3: `docs/en/library.md`**

- "Opening a dictionary": add `func OpenBytes(data []byte) (*Dictionary, error)` to the signature
  list and a bullet: "**`OpenBytes(data)`** — the same format as `Open`, from memory (e.g.
  `//go:embed`). Checksum verified; `data` is not copied and must outlive the dictionary;
  misaligned sections are copied. No mmap, so it works on Windows; `Close` is a no-op."
- Replace the paragraph after the `defer d.Close()` example with the lifecycle rule from Task 11
  (no calls in flight during `Close`; returned values stay valid).
- "Exact wordform lookup": add `Predicted bool` to the `Reading` listing with its comment, and a
  subsection "Dictionary words vs. predictions" showing:
  ```go
  d.IsKnown("кота")          // true: in the dictionary
  d.IsKnown("бота")          // false, although Parse("бота") may predict readings
  d.Parse("бота")[0].Predicted // true
  ```
- "Building a dictionary from scratch": state that `AddForm`/`AddLemma`/`ImportTSV` lower-case
  words and lemmas; document `BuilderOptions.CharPolicy` (nil = language default, `NoCharPolicy()`,
  `NewCharPolicy(Substitution{From: 'и', To: 'і'})`).
- "Fuzzy search": the query is lower-cased; the CharPolicy applies (е→ё costs 0 for Russian).
- "Diagnostic metadata": add `ContentHash()` (what it covers, that it ignores `info`, that the
  first call encodes the sections and is cached).
- "Multiple dictionaries at once": add `IsKnown`.

- [ ] **Step 4: `docs/ru/library.md` (minimal subset, Russian)**

Add a short section «Словарные разборы и предсказания» (`Predicted`, `IsKnown`), one sentence
each for `OpenBytes`, lower-casing in Builder/ImportTSV, `BuilderOptions.CharPolicy` and
`ContentHash`, and the `Close` rule («`Close` нельзя вызывать, пока другие горутины работают со
словарём; возвращённые строки остаются валидными»).

- [ ] **Step 5: `docs/en/comparison.md:35`**

Replace `yes — an ending-based prediction DAWG (pymorphy2- and OpenCorpora-sourced)` with
`yes — an ending-based prediction DAWG for pymorphy2 dictionaries and for dictionaries built with
Builder/ImportTSV (not for OpenCorpora or UniMorph imports); readings carry a Predicted flag`.

- [ ] **Step 6: Implementation write-up**

Create `docs/en/implementation/ner-support.md` with sections: *Context* (lexicon/genodex, link to
the spec); *1.2.0* — one paragraph per item A–F with what changed and why; *Findings* — the four
planning-time findings above (aliasing audit, «Москва» predicted, the `go` directive policy with
the dependency table, determinism); *Deviations from the spec* — Discrepancies D-1…D-11 below that
apply to 1.2.0. Add a row to `docs/en/todo.md`'s "Completed stages" table:

```markdown
| — NER support for lexicon: 1.2.0 (known-word flag, OpenBytes, case/ё, CharPolicy, ContentHash, go 1.25) | DONE | [implementation/ner-support.md](implementation/ner-support.md) |
```

In `docs/en/todo.md`, Stage 20 section, add one line: "Consumer note (2026-09-24): the lexicon
module does not need synonyms for NER; this stage stays unscheduled."

- [ ] **Step 7: Full verification**

Run:
```bash
gofmt -l . && go build ./... && go vet ./... && go test ./... -race -count=1 && golangci-lint run ./...
```
Expected: `gofmt -l` prints nothing; everything else PASS / no findings.

Self-review: walk the spec's "Testing" bullets A–F and point to the test for each (A:
`known_test.go`; B: `open_bytes_test.go`; C: `case_test.go`; D: `char_policy_test.go`,
`fuzzy_test.go`; E: `content_hash_test.go`; F: Task 10 Step 6 + `lifecycle_test.go`).

- [ ] **Step 8: Commit**

```bash
git add pkg/morphology/version.go CHANGELOG.md docs/en/library.md docs/ru/library.md docs/en/comparison.md docs/en/todo.md docs/en/implementation/ner-support.md
git commit -m "release: 1.2.0

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

## Discrepancies with the spec (found while planning)

- **D-1 (E).** "Cheap for Opened dictionaries (sections already located)": `Dictionary` does not
  keep the `internal.Container` after `Open` (fields are `d` and `mm` only), so the hash must
  re-encode sections, including a full `words.dawg` copy. The plan caches the result
  (`sync.Once`). Keeping the container for a zero-copy fast path is possible but would need to
  produce the identical digest for in-memory dictionaries — deferred (open question Q5).
- **D-2 (D).** "`UniMorphOptions.CharPolicy` becomes `*morphology.CharPolicy` (converted to the
  internal type)" is not implementable while `UniMorphOptions` is an alias of `unimorph.Options`:
  `unimorph` cannot import `morphology` (import cycle). The plan exposes `CharPolicy` and
  `Substitution` as **type aliases** of the internal types, which makes the existing field usable
  from outside with no type change at all (strictly more compatible than the spec's plan).
- **D-3 (C).** "A form added as «Москва» is never found": it is returned, but as a prediction,
  indistinguishable from a real reading before item A (planning finding #2). Same fix; tests must
  assert `Predicted == false`.
- **D-4 (D).** "`nil` → language default (ru: е→ё)": today the Builder applies е→ё for every
  language (`internal/build.go:259-262`). Following the spec changes that for non-"ru" Builder
  dictionaries (listed under "Changed"). Owner confirmed 2026-09-24 (answer 16, Q3): е→ё only
  for "ru" and the empty language; no substitutions for any other language.
- **D-5 (F).** Superseded by the owner decision 2026-09-24 (answer 15, spec G3): the `go`
  directive follows "current Go minus two minor versions" → `go 1.25.0`. No dependency
  downgrades (`x/sys v0.47.0` via the CLI's `x/term` declares exactly 1.25.0) and
  `strings.SplitSeq` stays. The technical floor after downgrades would be 1.22
  (`zeebo/xxh3 v1.1.0`) — recorded, not used. `go.mod` had no `toolchain` line — the plan adds
  `toolchain go1.27.1`.
- **D-6 (F).** The spec's aliasing concern is verified clean: no returned string aliases the
  mapping; only DAWG arrays do (finding #1). No code change needed, just a regression test.
- **D-7 (D).** "A substitution pair costs 0": direction was unspecified. The plan keeps the
  CharPolicy directional (query е → stored ё), consistent with `Parse`/`SimilarItems`; a query «ё»
  against a stored «е» still costs 1.
- **D-8 (D).** Behaviour change not listed in the spec's Compatibility section: existing
  `TestFuzzyRuneMetricYo` asserts е/ё distance 1; it becomes 0. CHANGELOG lists it under
  "Changed".
- **D-9 (A).** Open question G2 is moot on the current code: `MultiDictionary.Lemma` concatenates
  per-dictionary results without cross-dictionary dedup, and a single `Parse` returns either only
  exact or only predicted readings, so every reading behind one `LemmaRef` has the same flag.
- **D-10 (E).** "xxh3-128 over the section payloads": the plan frames each section as
  `name, 0x00, len, data` so that moving bytes between adjacent sections cannot keep the digest.
- **D-11 (C).** "`ImportTSV` inherits it": `ImportTSV` builds `internal.BuildEntry` values itself
  and never calls `AddForm`; it needs its own lower-casing (Task 5 does both).

## Open questions

- **Q1 (spec G1).** RESOLVED 2026-09-24 (owner, answer 15): single module, `cmd/gomorphy` stays
  inside. With `go 1.25.0` the CLI's `x/term`/`x/sys` need no pinning, so the main argument for a
  split is gone; revisit only if a consumer complains about the CLI dependencies in its `go.sum`.
- **Q2 (spec G3).** RESOLVED 2026-09-24 (owner, answer 15): policy "current Go minus two minor
  versions" → `go 1.25.0` + `toolchain go1.27.1` (Task 10). New code may use anything up to
  Go 1.25; the directive is raised when a new Go minor ships.
- **Q3.** RESOLVED 2026-09-24 (owner, answer 16): the Builder/ImportTSV default CharPolicy is
  by language — е→ё only for "ru" (an empty `Language` = "ru"), no substitutions otherwise
  (Task 7; CHANGELOG "Changed").
- **Q4.** `lookup` marks predictions as an extra trailing column `(predicted)`. Acceptable for
  scripts, or prefer a flag (`--no-predict`) / a different marker?
- **Q5.** `ContentHash` for `Open`ed dictionaries costs one full encode (cached). If lexicon calls
  it on every hot-reload check of large dictionaries, should `Open` retain the container and hash
  its raw section bytes (same digest, no copy)? Needs a proof that raw section bytes always equal
  the re-encoded ones (true for every section today).
- **Q6 (spec G2).** Resolved as moot (D-9); keep the documented rule "true only if all readings
  behind it are predicted" for the future case where cross-dictionary dedup appears.
- **Q7.** Should `Fuzzy`'s CharPolicy also be symmetric (ё in the query matching е in the
  dictionary)? Not done: `Parse` is one-way, and `Fuzzy` should agree with it.
