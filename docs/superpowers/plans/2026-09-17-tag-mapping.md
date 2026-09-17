# Universal Tag Mapping (`pkg/morphology/tagmap`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `pkg/morphology/tagmap`, a package that normalizes a
dictionary's native grammatical tag string into a UniMorph feature
`Bundle`, so tags from different dictionary sources (OpenCorpora,
pymorphy2) can be compared for the same grammatical meaning.

**Architecture:** One small package with four files: a shared `Bundle`
data model + merge/sort helper, a tokenizer+table pair per source format
(`opencorpora`, `opencorpora-int`), and a public `Map(dictName, tag)`
entry point that dispatches to the right pair by name. No changes to any
other package — callers (e.g. code using `morphology.MultiDictionary`)
call `tagmap.Map` themselves.

**Tech Stack:** Go (this repo's existing toolchain), `github.com/stretchr/testify`
(already vendored) for test assertions.

**Spec:** [docs/superpowers/specs/2026-09-17-tag-mapping-design.md](../specs/2026-09-17-tag-mapping-design.md)

## Global Constraints

- New package path: `pkg/morphology/tagmap` (module `github.com/amarin/gomorphy`,
  so imported as `github.com/amarin/gomorphy/pkg/morphology/tagmap`).
- `Map`'s direction is native → universal only. Do **not** add a reverse
  (`Unmap`) function in this plan — it's an explicit spec non-goal.
- Do **not** modify `internal.TagSet`, `morphology.MultiDictionary`,
  `morphology.Reading`, or `morphology.LemmaRef`. This package is called
  by application code on top of the existing public API.
- Mapping tables for `opencorpora` and `opencorpora-int` are two
  independent Go map literals, even where their content ends up
  identical — never derive one from the other or share a literal between
  them.
- An unknown token in a tag (not in the source's table) goes into
  `Bundle.Unmapped`, verbatim, and is never an error. `Map` returns
  `ok=false` only when `dictName` itself is not a registered source.
- `Bundle.Features` is always sorted by `Dimension` in the order the
  `Dimension` constants are declared (see Task 1) — this is what makes
  two bundles from different sources comparable with `assert.Equal`/
  `slices.Equal`, regardless of the input tag's own token order.
- Every test file's package name matters: white-box tests of unexported
  tokenizers/tables/helpers use `package tagmap`; black-box tests of the
  public `Map` function and the integration test use `package tagmap_test`.
- Run `go test ./pkg/morphology/tagmap/... -race` after every task; it
  must stay green. Run `golangci-lint run ./pkg/morphology/tagmap/...`
  before the final commit of each task.

---

## Task 1: `Bundle` data model + `buildBundle` merge/sort helper

**Files:**
- Create: `pkg/morphology/tagmap/bundle.go`
- Test: `pkg/morphology/tagmap/bundle_test.go`

**Interfaces:**
- Produces:
  - `type Dimension uint8` with constants `DimPartOfSpeech`, `DimAnimacy`,
    `DimCase`, `DimNumber`, `DimGender`, `DimTense`, `DimAspect`,
    `DimMood`, `DimVoice`, `DimPerson` (in this exact declaration order —
    this order IS the canonical sort order used everywhere else in this
    plan).
  - `type Feature struct { Dim Dimension; Value string }`
  - `type Bundle struct { Features []Feature; Unmapped []string }`
  - `func buildBundle(tokens []string, table map[string]Feature) Bundle`
    — for each token, looks it up in `table`; a hit is merged into the
    result keyed by `Feature.Dim` (a later token whose `Feature.Dim`
    collides with an earlier one overwrites it — "later wins"); a miss
    is appended to `Bundle.Unmapped` verbatim, in encounter order.
    Output `Features` is sorted by `Dimension` per the declared constant
    order. Both `Features` and `Unmapped` are `nil` (not empty non-nil
    slices) when there is nothing to put in them.

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/tagmap/bundle_test.go`:

```go
package tagmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildBundleCanonicalOrderIndependentOfInputOrder(t *testing.T) {
	table := map[string]Feature{
		"nomn": {DimCase, "NOM"},
		"NOUN": {DimPartOfSpeech, "N"},
		"sing": {DimNumber, "SG"},
	}

	// Input order deliberately scrambled relative to Dimension
	// declaration order (Case, PartOfSpeech, Number) to prove the
	// output order comes from Dimension, not from input order.
	b := buildBundle([]string{"nomn", "NOUN", "sing"}, table)

	assert.Equal(t, []Feature{
		{DimPartOfSpeech, "N"},
		{DimCase, "NOM"},
		{DimNumber, "SG"},
	}, b.Features)
	assert.Nil(t, b.Unmapped)
}

func TestBuildBundleUnknownTokenGoesToUnmapped(t *testing.T) {
	table := map[string]Feature{
		"NOUN": {DimPartOfSpeech, "N"},
	}

	b := buildBundle([]string{"NOUN", "Slng"}, table)

	assert.Equal(t, []Feature{{DimPartOfSpeech, "N"}}, b.Features)
	assert.Equal(t, []string{"Slng"}, b.Unmapped)
}

func TestBuildBundleLaterTokenWinsOnDimensionCollision(t *testing.T) {
	table := map[string]Feature{
		"nomn": {DimCase, "NOM"},
		"gent": {DimCase, "GEN"},
	}

	b := buildBundle([]string{"nomn", "gent"}, table)

	assert.Equal(t, []Feature{{DimCase, "GEN"}}, b.Features)
}

func TestBuildBundleEmptyInputProducesNilBundle(t *testing.T) {
	b := buildBundle(nil, map[string]Feature{"NOUN": {DimPartOfSpeech, "N"}})

	assert.Nil(t, b.Features)
	assert.Nil(t, b.Unmapped)
}

func TestBuildBundleTwoSourcesSameMeaningCompareEqual(t *testing.T) {
	// Simulates the cross-source scenario this whole package exists for:
	// two different token orders/tables producing the same Feature set
	// must compare equal.
	tableA := map[string]Feature{
		"NOUN": {DimPartOfSpeech, "N"},
		"sing": {DimNumber, "SG"},
	}
	tableB := map[string]Feature{
		"N_TAG":   {DimPartOfSpeech, "N"},
		"SG_TAG":  {DimNumber, "SG"},
	}

	a := buildBundle([]string{"NOUN", "sing"}, tableA)
	b := buildBundle([]string{"SG_TAG", "N_TAG"}, tableB)

	assert.Equal(t, a.Features, b.Features)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/morphology/tagmap/... -run TestBuildBundle -v`
Expected: FAIL — compile error, `undefined: Feature` (or similar; the
package doesn't exist yet).

- [ ] **Step 3: Write the implementation**

Create `pkg/morphology/tagmap/bundle.go`:

```go
// Package tagmap normalizes a dictionary's native grammatical tag string
// into a universal feature bundle (the UniMorph Schema), so tags from
// different dictionary sources (OpenCorpora, pymorphy2) can be compared
// for the same grammatical meaning. See
// docs/superpowers/specs/2026-09-17-tag-mapping-design.md.
package tagmap

import "sort"

// Dimension is one of the UniMorph Schema's dimensions of meaning
// (Sylak-Glassman 2016), restricted to the ones a mapping table in this
// package actually uses. Declaration order fixes the canonical sort
// order for Bundle.Features — this is what makes bundles built from
// different sources comparable regardless of the input tag's own token
// order. New dimensions are appended here only when a mapping table
// needs them; existing ones must never be reordered (it would silently
// change every existing Bundle's canonical order).
type Dimension uint8

const (
	DimPartOfSpeech Dimension = iota
	DimAnimacy
	DimCase
	DimNumber
	DimGender
	DimTense
	DimAspect
	DimMood
	DimVoice
	DimPerson
)

// Feature is a single UniMorph feature value within a Dimension, e.g.
// {DimCase, "NOM"}.
type Feature struct {
	Dim   Dimension
	Value string
}

// Bundle is a native tag normalized into UniMorph terms. Features is
// sorted by Dimension so two Bundles expressing the same grammatical
// meaning from different sources compare equal via reflect.DeepEqual/
// assert.Equal, independent of source token order. Unmapped holds,
// verbatim and in encounter order, every input token that had no entry
// in the source's mapping table — never merged into Features, never
// dropped silently.
type Bundle struct {
	Features []Feature
	Unmapped []string
}

// buildBundle maps each token through table, merging hits by Dimension
// (a later token whose Feature shares a Dimension with an earlier one
// overwrites it) and collecting misses into Unmapped verbatim. The
// result's Features is sorted by Dimension's declaration order.
func buildBundle(tokens []string, table map[string]Feature) Bundle {
	var (
		byDim map[Dimension]Feature
		order []Dimension
		unmapped []string
	)

	for _, tok := range tokens {
		f, ok := table[tok]
		if !ok {
			unmapped = append(unmapped, tok)
			continue
		}
		if byDim == nil {
			byDim = make(map[Dimension]Feature)
		}
		if _, exists := byDim[f.Dim]; !exists {
			order = append(order, f.Dim)
		}
		byDim[f.Dim] = f
	}

	if len(order) == 0 {
		return Bundle{Unmapped: unmapped}
	}

	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	features := make([]Feature, 0, len(order))
	for _, d := range order {
		features = append(features, byDim[d])
	}
	return Bundle{Features: features, Unmapped: unmapped}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/morphology/tagmap/... -run TestBuildBundle -v`
Expected: PASS (all 5 tests).

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/tagmap/bundle.go pkg/morphology/tagmap/bundle_test.go
git commit -m "feat(tagmap): add Bundle data model and buildBundle merge/sort helper"
```

---

## Task 2: OpenCorpora tokenizer + mapping table

**Files:**
- Create: `pkg/morphology/tagmap/opencorpora.go`
- Test: `pkg/morphology/tagmap/opencorpora_test.go`

**Interfaces:**
- Consumes: `Feature`, `Dimension` constants, `buildBundle` (Task 1).
- Produces:
  - `func tokenizeOpenCorpora(tag string) []string` — splits a
    `dict.xml`-style combined tag on `,`. Empty input returns `nil`.
  - `var openCorporaTable map[string]Feature` — a representative subset
    of OpenCorpora grammemes (not exhaustive — see Global Constraints):

    | token | Feature |
    |---|---|
    | `NOUN` | `{DimPartOfSpeech, "N"}` |
    | `VERB` | `{DimPartOfSpeech, "V"}` |
    | `INFN` | `{DimPartOfSpeech, "V"}` |
    | `ADJF` | `{DimPartOfSpeech, "ADJ"}` |
    | `ADVB` | `{DimPartOfSpeech, "ADV"}` |
    | `NPRO` | `{DimPartOfSpeech, "PRO"}` |
    | `NUMR` | `{DimPartOfSpeech, "NUM"}` |
    | `anim` | `{DimAnimacy, "ANIM"}` |
    | `inan` | `{DimAnimacy, "INAN"}` |
    | `nomn` | `{DimCase, "NOM"}` |
    | `gent` | `{DimCase, "GEN"}` |
    | `datv` | `{DimCase, "DAT"}` |
    | `accs` | `{DimCase, "ACC"}` |
    | `ablt` | `{DimCase, "INS"}` |
    | `loct` | `{DimCase, "ESS"}` |
    | `voct` | `{DimCase, "VOC"}` |
    | `sing` | `{DimNumber, "SG"}` |
    | `plur` | `{DimNumber, "PL"}` |
    | `masc` | `{DimGender, "MASC"}` |
    | `femn` | `{DimGender, "FEM"}` |
    | `neut` | `{DimGender, "NEUT"}` |
    | `pres` | `{DimTense, "PRS"}` |
    | `past` | `{DimTense, "PST"}` |
    | `futr` | `{DimTense, "FUT"}` |
    | `perf` | `{DimAspect, "PFV"}` |
    | `impf` | `{DimAspect, "IPFV"}` |
    | `indc` | `{DimMood, "IND"}` |
    | `impr` | `{DimMood, "IMP"}` |
    | `actv` | `{DimVoice, "ACT"}` |
    | `pssv` | `{DimVoice, "PASS"}` |
    | `1per` | `{DimPerson, "1"}` |
    | `2per` | `{DimPerson, "2"}` |
    | `3per` | `{DimPerson, "3"}` |

    (`INFN`→`V` and `loct`→`ESS` are deliberate simplifications, matching
    the direction already sketched in `docs/unimorph.md` §5.4 — `INFN` is
    OpenCorpora's separate part-of-speech grammeme for infinitives, folded
    here into the same `V` as `VERB`; `loct` (OpenCorpora's locative/
    prepositional case) matches UniMorph's `ESS`, per `docs/unimorph.md`
    §4's note that `ESS` is how the Russian UniMorph dataset represents
    this case.)

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/tagmap/opencorpora_test.go`:

```go
package tagmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenizeOpenCorporaSplitsOnComma(t *testing.T) {
	assert.Equal(t,
		[]string{"NOUN", "anim", "masc", "sing", "nomn"},
		tokenizeOpenCorpora("NOUN,anim,masc,sing,nomn"))
}

func TestTokenizeOpenCorporaSingleToken(t *testing.T) {
	assert.Equal(t, []string{"NOUN"}, tokenizeOpenCorpora("NOUN"))
}

func TestTokenizeOpenCorporaEmptyTagIsNil(t *testing.T) {
	assert.Nil(t, tokenizeOpenCorpora(""))
}

func TestOpenCorporaTableCoversKotNominativeSingular(t *testing.T) {
	// "кот" (NOUN,anim,masc,sing,nomn) — the exact example already used
	// throughout docs/todo.md and docs/implementation/ for this dictionary.
	b := buildBundle(tokenizeOpenCorpora("NOUN,anim,masc,sing,nomn"), openCorporaTable)

	assert.Equal(t, []Feature{
		{DimPartOfSpeech, "N"},
		{DimAnimacy, "ANIM"},
		{DimCase, "NOM"},
		{DimNumber, "SG"},
		{DimGender, "MASC"},
	}, b.Features)
	assert.Nil(t, b.Unmapped)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/morphology/tagmap/... -run TestTokenizeOpenCorpora -v`
and
`go test ./pkg/morphology/tagmap/... -run TestOpenCorporaTable -v`
Expected: FAIL — `undefined: tokenizeOpenCorpora` / `undefined: openCorporaTable`.

- [ ] **Step 3: Write the implementation**

Create `pkg/morphology/tagmap/opencorpora.go`:

```go
package tagmap

import "strings"

// tokenizeOpenCorpora splits a dict.xml-style combined tag
// ("NOUN,anim,masc,sing,nomn") into its comma-separated grammeme tokens.
func tokenizeOpenCorpora(tag string) []string {
	if tag == "" {
		return nil
	}
	return strings.Split(tag, ",")
}

// openCorporaTable maps OpenCorpora grammeme names (as produced by
// pkg/morphology/importers/opencorpora's import, TagSet.Name ==
// "opencorpora") to UniMorph features. A representative subset, not
// exhaustive — see docs/superpowers/plans/2026-09-17-tag-mapping.md's
// Global Constraints. Grows incrementally as uncovered grammemes are
// found; an uncovered grammeme is not an error (see Bundle.Unmapped).
var openCorporaTable = map[string]Feature{
	"NOUN": {DimPartOfSpeech, "N"},
	"VERB": {DimPartOfSpeech, "V"},
	"INFN": {DimPartOfSpeech, "V"},
	"ADJF": {DimPartOfSpeech, "ADJ"},
	"ADVB": {DimPartOfSpeech, "ADV"},
	"NPRO": {DimPartOfSpeech, "PRO"},
	"NUMR": {DimPartOfSpeech, "NUM"},

	"anim": {DimAnimacy, "ANIM"},
	"inan": {DimAnimacy, "INAN"},

	"nomn": {DimCase, "NOM"},
	"gent": {DimCase, "GEN"},
	"datv": {DimCase, "DAT"},
	"accs": {DimCase, "ACC"},
	"ablt": {DimCase, "INS"},
	"loct": {DimCase, "ESS"},
	"voct": {DimCase, "VOC"},

	"sing": {DimNumber, "SG"},
	"plur": {DimNumber, "PL"},

	"masc": {DimGender, "MASC"},
	"femn": {DimGender, "FEM"},
	"neut": {DimGender, "NEUT"},

	"pres": {DimTense, "PRS"},
	"past": {DimTense, "PST"},
	"futr": {DimTense, "FUT"},

	"perf": {DimAspect, "PFV"},
	"impf": {DimAspect, "IPFV"},

	"indc": {DimMood, "IND"},
	"impr": {DimMood, "IMP"},

	"actv": {DimVoice, "ACT"},
	"pssv": {DimVoice, "PASS"},

	"1per": {DimPerson, "1"},
	"2per": {DimPerson, "2"},
	"3per": {DimPerson, "3"},
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/morphology/tagmap/... -v`
Expected: PASS (all tests from Task 1 and Task 2).

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/tagmap/opencorpora.go pkg/morphology/tagmap/opencorpora_test.go
git commit -m "feat(tagmap): add OpenCorpora tokenizer and grammeme mapping table"
```

---

## Task 3: pymorphy2 (`opencorpora-int`) tokenizer + mapping table

**Files:**
- Create: `pkg/morphology/tagmap/pymorphy2int.go`
- Test: `pkg/morphology/tagmap/pymorphy2int_test.go`

**Interfaces:**
- Consumes: `Feature`, `Dimension` constants, `buildBundle` (Task 1).
- Produces:
  - `func tokenizeOpenCorporaInt(tag string) []string` — splits a
    pymorphy2 `gramtab-opencorpora-int.json` tag
    (`"NOUN,anim,masc sing,nomn"` — lemma-part and form-changing part
    separated by a space, each comma-joined) into a flat token list,
    discarding the structural space. A tag with no space (single group)
    is still split by comma. Empty input returns `nil`.
  - `var openCorporaIntTable map[string]Feature` — **the same 33
    entries as `openCorporaTable`** (Task 2's table), written out again
    independently in this file. Do not import or alias Task 2's table —
    per the spec's Decision, the two tables are deliberately kept
    separate even where content is identical, because the two sources'
    token spellings are not assumed identical without this kind of
    explicit, checkable duplication. (The one real example already
    verified in `docs/todo.md` — `"NOUN,anim,masc sing,nomn"` from a real
    `gramtab-opencorpora-int.json` — uses the exact same token spellings
    as OpenCorpora's `NOUN`, `anim`, `masc`, `sing`, `nomn`, which is why
    this table's content matches Task 2's; if a future real-data check
    finds a source where spellings diverge, only that source's table
    changes.)

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/tagmap/pymorphy2int_test.go`:

```go
package tagmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenizeOpenCorporaIntSplitsSpaceThenComma(t *testing.T) {
	// Real example from a pymorphy2 gramtab-opencorpora-int.json, quoted
	// in docs/todo.md's "Универсальный маппинг тегов между словарями"
	// section.
	assert.Equal(t,
		[]string{"NOUN", "anim", "masc", "sing", "nomn"},
		tokenizeOpenCorporaInt("NOUN,anim,masc sing,nomn"))
}

func TestTokenizeOpenCorporaIntSingleGroupNoSpace(t *testing.T) {
	assert.Equal(t,
		[]string{"NOUN", "anim", "masc"},
		tokenizeOpenCorporaInt("NOUN,anim,masc"))
}

func TestTokenizeOpenCorporaIntEmptyTagIsNil(t *testing.T) {
	assert.Nil(t, tokenizeOpenCorporaInt(""))
}

func TestOpenCorporaIntTableCoversKotNominativeSingular(t *testing.T) {
	b := buildBundle(tokenizeOpenCorporaInt("NOUN,anim,masc sing,nomn"), openCorporaIntTable)

	assert.Equal(t, []Feature{
		{DimPartOfSpeech, "N"},
		{DimAnimacy, "ANIM"},
		{DimCase, "NOM"},
		{DimNumber, "SG"},
		{DimGender, "MASC"},
	}, b.Features)
	assert.Nil(t, b.Unmapped)
}

func TestOpenCorporaAndOpenCorporaIntAgreeOnKotNominativeSingular(t *testing.T) {
	oc := buildBundle(tokenizeOpenCorpora("NOUN,anim,masc,sing,nomn"), openCorporaTable)
	pm := buildBundle(tokenizeOpenCorporaInt("NOUN,anim,masc sing,nomn"), openCorporaIntTable)

	assert.Equal(t, oc.Features, pm.Features)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/morphology/tagmap/... -run TestTokenizeOpenCorporaInt -v`
and
`go test ./pkg/morphology/tagmap/... -run TestOpenCorporaIntTable -v`
Expected: FAIL — `undefined: tokenizeOpenCorporaInt` / `undefined: openCorporaIntTable`.

- [ ] **Step 3: Write the implementation**

Create `pkg/morphology/tagmap/pymorphy2int.go`:

```go
package tagmap

import "strings"

// tokenizeOpenCorporaInt splits a pymorphy2 gramtab-opencorpora-int tag
// ("NOUN,anim,masc sing,nomn" — lemma-part and form-changing part
// separated by a space, each comma-joined) into a flat grammeme token
// list; the structural space is discarded, both parts are just
// grammemes to the mapping table. A tag with a single group (no space)
// is still split by comma.
func tokenizeOpenCorporaInt(tag string) []string {
	if tag == "" {
		return nil
	}
	var tokens []string
	for _, group := range strings.Fields(tag) {
		tokens = append(tokens, strings.Split(group, ",")...)
	}
	return tokens
}

// openCorporaIntTable maps pymorphy2's gramtab-opencorpora-int grammeme
// names (TagSet.Name == "opencorpora-int") to UniMorph features.
// Deliberately not shared with openCorporaTable (see this task's doc
// comment in the implementation plan) even though its content is
// currently identical — the two sources are independent inputs to this
// package and are kept independently editable.
var openCorporaIntTable = map[string]Feature{
	"NOUN": {DimPartOfSpeech, "N"},
	"VERB": {DimPartOfSpeech, "V"},
	"INFN": {DimPartOfSpeech, "V"},
	"ADJF": {DimPartOfSpeech, "ADJ"},
	"ADVB": {DimPartOfSpeech, "ADV"},
	"NPRO": {DimPartOfSpeech, "PRO"},
	"NUMR": {DimPartOfSpeech, "NUM"},

	"anim": {DimAnimacy, "ANIM"},
	"inan": {DimAnimacy, "INAN"},

	"nomn": {DimCase, "NOM"},
	"gent": {DimCase, "GEN"},
	"datv": {DimCase, "DAT"},
	"accs": {DimCase, "ACC"},
	"ablt": {DimCase, "INS"},
	"loct": {DimCase, "ESS"},
	"voct": {DimCase, "VOC"},

	"sing": {DimNumber, "SG"},
	"plur": {DimNumber, "PL"},

	"masc": {DimGender, "MASC"},
	"femn": {DimGender, "FEM"},
	"neut": {DimGender, "NEUT"},

	"pres": {DimTense, "PRS"},
	"past": {DimTense, "PST"},
	"futr": {DimTense, "FUT"},

	"perf": {DimAspect, "PFV"},
	"impf": {DimAspect, "IPFV"},

	"indc": {DimMood, "IND"},
	"impr": {DimMood, "IMP"},

	"actv": {DimVoice, "ACT"},
	"pssv": {DimVoice, "PASS"},

	"1per": {DimPerson, "1"},
	"2per": {DimPerson, "2"},
	"3per": {DimPerson, "3"},
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/morphology/tagmap/... -v`
Expected: PASS (all tests from Tasks 1-3).

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/tagmap/pymorphy2int.go pkg/morphology/tagmap/pymorphy2int_test.go
git commit -m "feat(tagmap): add pymorphy2 opencorpora-int tokenizer and mapping table"
```

---

## Task 4: Public `Map` API

**Files:**
- Create: `pkg/morphology/tagmap/tagmap.go`
- Test: `pkg/morphology/tagmap/tagmap_test.go`

**Interfaces:**
- Consumes: `buildBundle` (Task 1); `tokenizeOpenCorpora`,
  `openCorporaTable` (Task 2); `tokenizeOpenCorporaInt`,
  `openCorporaIntTable` (Task 3).
- Produces: `func Map(dictName, tag string) (Bundle, bool)` — the
  package's only exported function. `dictName` is a dictionary's
  `TagSet.Name` (`"opencorpora"` or `"opencorpora-int"` are the only two
  registered sources right now). Returns `ok=false` only when `dictName`
  is not registered; any other input always returns `ok=true` (possibly
  with a Bundle that has `Unmapped` entries).

This is the task whose test file switches to `package tagmap_test`
(black-box): it tests the public contract only, not internals.

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/tagmap/tagmap_test.go`:

```go
package tagmap_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/tagmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapOpenCorporaKnownTag(t *testing.T) {
	b, ok := tagmap.Map("opencorpora", "NOUN,anim,masc,sing,nomn")
	require.True(t, ok)

	assert.Equal(t, []tagmap.Feature{
		{Dim: tagmap.DimPartOfSpeech, Value: "N"},
		{Dim: tagmap.DimAnimacy, Value: "ANIM"},
		{Dim: tagmap.DimCase, Value: "NOM"},
		{Dim: tagmap.DimNumber, Value: "SG"},
		{Dim: tagmap.DimGender, Value: "MASC"},
	}, b.Features)
}

func TestMapUnregisteredDictNameReturnsFalse(t *testing.T) {
	b, ok := tagmap.Map("some-future-source", "whatever,tag")

	assert.False(t, ok)
	assert.Equal(t, tagmap.Bundle{}, b)
}

func TestMapUnmappedTokenPassesThrough(t *testing.T) {
	b, ok := tagmap.Map("opencorpora", "NOUN,Slng")
	require.True(t, ok)

	assert.Equal(t, []string{"Slng"}, b.Unmapped)
}

func TestMapOpenCorporaAndOpenCorporaIntAgree(t *testing.T) {
	// The real documented example (docs/todo.md's tag-mapping section):
	// same word "кот", same grammatical meaning, two different native
	// tag syntaxes — must normalize to the same Bundle.Features.
	oc, ok := tagmap.Map("opencorpora", "NOUN,anim,masc,sing,nomn")
	require.True(t, ok)
	pm, ok := tagmap.Map("opencorpora-int", "NOUN,anim,masc sing,nomn")
	require.True(t, ok)

	assert.Equal(t, oc.Features, pm.Features)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/morphology/tagmap/... -run TestMap -v`
Expected: FAIL — `undefined: tagmap.Map` (package `tagmap` has no such
export yet).

- [ ] **Step 3: Write the implementation**

Create `pkg/morphology/tagmap/tagmap.go`:

```go
package tagmap

// source pairs a tokenizer with its mapping table for one dictionary
// format (one TagSet.Name).
type source struct {
	tokenize func(tag string) []string
	table    map[string]Feature
}

// sources is the registry of dictionary formats this package knows how
// to normalize, keyed by TagSet.Name.
var sources = map[string]source{
	"opencorpora":     {tokenize: tokenizeOpenCorpora, table: openCorporaTable},
	"opencorpora-int": {tokenize: tokenizeOpenCorporaInt, table: openCorporaIntTable},
}

// Map normalizes tag (as found in a Dictionary's TagSet, i.e. a value
// TagSet.TagName would return) into a Bundle, using dictName (the
// dictionary's TagSet.Name) to pick the tokenizer and mapping table.
//
// ok is false only when dictName is not a source this package knows
// about at all — a caller bug (asking for an unregistered source), not a
// property of tag's content. Any other input always returns ok=true;
// tokens absent from the source's table land in Bundle.Unmapped rather
// than causing an error (see Bundle's doc comment).
func Map(dictName, tag string) (Bundle, bool) {
	src, known := sources[dictName]
	if !known {
		return Bundle{}, false
	}
	return buildBundle(src.tokenize(tag), src.table), true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/morphology/tagmap/... -v`
Expected: PASS (all tests from Tasks 1-4).

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/tagmap/tagmap.go pkg/morphology/tagmap/tagmap_test.go
git commit -m "feat(tagmap): add public Map entry point"
```

---

## Task 5: Integration test against real dictionaries

**Files:**
- Create: `pkg/morphology/tagmap/integration_test.go`

**Interfaces:**
- Consumes: `tagmap.Map` (Task 4); `morphology.CompileFromXMLFile`,
  `morphology.OpenPyMorphy`, `morphology.Dictionary.Parse`,
  `morphology.Reading.Tag` (all pre-existing public API in
  `pkg/morphology`, unmodified by this plan).
- Produces: nothing consumed by later tasks — this is a leaf
  verification, gated behind the `integration` build tag like the
  repo's existing real-dictionary tests
  (`pkg/morphology/importers/opencorpora/real_dict_integration_test.go`,
  `pkg/morphology/importers/pymorphy2/full_dict_integration_test.go`).

This test needs a real OpenCorpora `dict.xml` and a real pymorphy2 data
directory on disk. It reuses the same environment variables and
default-path convention as the existing integration tests
(`GOMORPHY_DICT_XML` defaulting to `.data/opencorpora/dict.xml` found by
walking up from the working directory; `GOMORPHY_PYMORPHY2_DIR` with no
default, must be set) — skip, don't fail, when either is unavailable.

- [ ] **Step 1: Write the test**

Create `pkg/morphology/tagmap/integration_test.go`:

```go
//go:build integration

// Integration test here requires a real OpenCorpora dict.xml and a real
// pymorphy2 data directory on disk. Run explicitly:
//
//	go test -tags=integration ./pkg/morphology/tagmap/... -v
package tagmap_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/amarin/gomorphy/pkg/morphology/tagmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dictXMLEnvVar       = "GOMORPHY_DICT_XML"
	dictXMLDefaultPath  = ".data/opencorpora/dict.xml"
	pymorphyDirEnvVar   = "GOMORPHY_PYMORPHY2_DIR"
)

// findDictXML mirrors
// pkg/morphology/importers/opencorpora/real_dict_integration_test.go's
// findRealDictXML: walks up from the working directory looking for
// .data/opencorpora/dict.xml, so this test passes both from the package
// dir and from the repo root.
func findDictXML() string {
	if p := os.Getenv(dictXMLEnvVar); p != "" {
		return p
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, dictXMLDefaultPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// nominativeSingularNounTag finds, among readings, the one whose tag
// looks like a nominative singular noun — robust to the two different
// tag syntaxes (opencorpora vs opencorpora-int both use these exact
// substrings, just arranged differently) without needing to know which
// dictionary produced readings.
func nominativeSingularNounTag(t *testing.T, readings []morphology.Reading) string {
	t.Helper()
	for _, r := range readings {
		if strings.Contains(r.Tag, "NOUN") &&
			strings.Contains(r.Tag, "nomn") &&
			strings.Contains(r.Tag, "sing") {
			return r.Tag
		}
	}
	t.Fatalf("no NOUN/nomn/sing reading found among %d readings", len(readings))
	return ""
}

func TestMapAgreesAcrossRealOpenCorporaAndPymorphy2Dictionaries(t *testing.T) {
	xmlPath := findDictXML()
	if xmlPath == "" {
		t.Skipf("dict.xml not found (%s to override)", dictXMLEnvVar)
	}
	pymorphyDir := os.Getenv(pymorphyDirEnvVar)
	if pymorphyDir == "" {
		t.Skipf("%s not set", pymorphyDirEnvVar)
	}

	ocDict, err := morphology.CompileFromXMLFile(xmlPath, nil)
	require.NoError(t, err)
	defer func() { _ = ocDict.Close() }()

	pmDict, err := morphology.OpenPyMorphy(pymorphyDir)
	require.NoError(t, err)
	defer func() { _ = pmDict.Close() }()

	ocTag := nominativeSingularNounTag(t, ocDict.Parse("кот"))
	pmTag := nominativeSingularNounTag(t, pmDict.Parse("кот"))

	ocBundle, ok := tagmap.Map("opencorpora", ocTag)
	require.True(t, ok)
	pmBundle, ok := tagmap.Map("opencorpora-int", pmTag)
	require.True(t, ok)

	assert.Equal(t, ocBundle.Features, pmBundle.Features,
		"opencorpora tag %q and pymorphy2 tag %q should normalize to the same universal bundle",
		ocTag, pmTag)
	assert.Empty(t, ocBundle.Unmapped,
		"opencorpora tag %q has grammemes outside the representative table", ocTag)
	assert.Empty(t, pmBundle.Unmapped,
		"pymorphy2 tag %q has grammemes outside the representative table", pmTag)
}
```

- [ ] **Step 2: Run the test**

Run: `go test -tags=integration ./pkg/morphology/tagmap/... -v`

Expected: either PASS (if `.data/opencorpora/dict.xml` and a pymorphy2
data directory — set `GOMORPHY_PYMORPHY2_DIR` to its path — are present
on this machine, matching the pattern already used by
`docs/todo.md`/other integration tests to obtain them), or SKIP with the
message naming which environment variable to set, if the data isn't
present. Either outcome is acceptable — this step is about confirming the
test compiles and runs its intended branch, not about requiring real data
to be present in every environment.

If it runs and FAILs (not skips), investigate before proceeding — a
failure here means either the representative tables (Tasks 2-3) are
missing a grammeme `кот`'s reading actually uses, or the two sources
disagree on its meaning, both of which are real bugs to fix in Tasks 2/3
before continuing.

- [ ] **Step 3: Commit**

```bash
git add pkg/morphology/tagmap/integration_test.go
git commit -m "test(tagmap): add real-dictionary cross-source integration check"
```

---

## Task 6: Update `docs/todo.md`'s tag-mapping section

**Files:**
- Modify: `docs/todo.md` (its final section, "Универсальный маппинг
  тегов между словарями")

**Interfaces:** none — documentation only, no code.

- [ ] **Step 1: Update the section heading**

In `docs/todo.md`, find this line (the file's last heading):

```
### Универсальный маппинг тегов между словарями — НЕ СПРОЕКТИРОВАНО
```

Replace it with:

```
### Универсальный маппинг тегов между словарями — РЕШЕНО И РЕАЛИЗОВАНО 2026-09-17 (native → universal)
```

- [ ] **Step 2: Append the resolution note**

At the very end of the file (after the last bullet point, which currently
ends with "...если маппинг появится, он логично встраивается именно в
этот путь."), append:

```markdown

**Открытые вопросы — решены 2026-09-17, см.
[docs/superpowers/specs/2026-09-17-tag-mapping-design.md](superpowers/specs/2026-09-17-tag-mapping-design.md)
и [docs/research/0008-dictionary-export-feasibility.md](research/0008-dictionary-export-feasibility.md):**
1. Таблица соответствий — отдельный пакет `pkg/morphology/tagmap`, не
   часть `TagSet` и не внешняя конфигурация: `TagSet` остаётся общим
   интернером строк, не знающим о синтаксисе и семантике тегов.
2. Opaque-теги и непокрытые граммемы — единое правило: как есть, в
   `Bundle.Unmapped`, без ошибки. `tagmap.Map` возвращает `ok=false`
   только для незарегистрированного `dictName` целиком, не за
   содержимое тега.
3. Формат universal-тега — независимый набор признаков UniMorph
   (`tagmap.Bundle`/`tagmap.Feature`/`tagmap.Dimension`), в
   каноническом порядке измерений (не зависящем от исходного порядка
   токенов в теге источника) — не привязан к одному из существующих
   словарных наборов.
4. Связь с multi-dict — `MultiDictionary`/`Reading`/`LemmaRef` не
   изменены; вызывающий код сам вызывает `tagmap.Map(dictName,
   reading.Tag)`, когда нужен universal-тег.

Реализованы таблицы для `opencorpora` и `opencorpora-int` (pymorphy2) —
представительное подмножество граммем (часть речи, одушевлённость,
падеж, число, род, время, вид, наклонение, залог, лицо), не
исчерпывающее покрытие; расширяется по мере находок непокрытых токенов.
Обратное направление (`universal → native`, нужное для экспорта) —
сознательно не реализовано в этом инкременте, отдельная будущая задача
(см. `0008-dictionary-export-feasibility.md`).
```

- [ ] **Step 3: Commit**

```bash
git add docs/todo.md
git commit -m "docs: mark universal tag mapping (native -> universal) shipped in todo.md"
```

---

## Final verification (run after Task 6, before declaring the plan done)

- [ ] `go build ./...` — green.
- [ ] `go test ./... -race` — green.
- [ ] `go test -tags=integration ./pkg/morphology/tagmap/... -v` — PASS or
  SKIP (not FAIL), as in Task 5.
- [ ] `golangci-lint run ./pkg/morphology/tagmap/...` — clean.
