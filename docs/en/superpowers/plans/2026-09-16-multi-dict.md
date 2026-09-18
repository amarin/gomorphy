# morphology.MultiDictionary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `morphology.MultiDictionary`, a wrapper over `[]*Dictionary` that aggregates `Parse`/`Lemma`/`Close` across an arbitrary set of already-open dictionaries, tagging every returned `Reading`/`LemmaRef` with which dictionary produced it.

**Architecture:** `Reading` and `LemmaRef` gain a `Dict int` field (same shape as the existing `Shard int` — an index, not a string), zero-valued and unused outside `MultiDictionary`. `MultiDictionary.Parse`/`Lemma` call each held dictionary's own method in parallel (one goroutine per dictionary, mirroring `Dictionary.exact`'s existing per-shard fan-out), stamp the `Dict` index onto each result, and concatenate in registration order — no cross-dictionary sort, dedup, or priority. `MultiDictionary.DictInfo(i)` looks up a dictionary's `BuildInfo` (`Source`/`SourceVersion`, now populated by both importers) by that same index.

**Tech Stack:** Go (stdlib `sync`, `errors`), `github.com/stretchr/testify` (assert/require), reusing `pkg/morphology/fixture_test.go`'s existing `buildFixtureDir`/`stdWords`/`addWord`/`writeFile` helpers.

**Spec:** [docs/superpowers/specs/2026-09-16-multi-dict-design.md](../specs/2026-09-16-multi-dict-design.md)

## Global Constraints

- No CLI changes (`cmd/gomorphy` keeps its single `-dict` flag) — Go API only.
- No merge/priority/dedup logic across dictionaries' overlapping results — return everything, tagged, in registration order.
- `Reading`/`LemmaRef`'s new `Dict` field must default to `0` and must not change behavior for any existing `Dictionary.Parse`/`Lemma` call made outside `MultiDictionary`.
- No uniqueness enforcement on dictionaries' `Source`/`SourceVersion` identity — the int index is what always disambiguates.
- `go test ./... -race` and `go build -tags=integration ./...` must stay green.
- `golangci-lint run ./...` must report 0 new issues (2 pre-existing, unrelated issues — `pkg/morphology/shard_test.go:67` errcheck, `pkg/morphology/importers/opencorpora/import.go:480` staticcheck — are not this work's concern).

---

## Task 1: `Reading.Dict`/`LemmaRef.Dict` fields + `MultiDictionary`

**Files:**
- Modify: `pkg/morphology/parse.go:14-22` (`Reading` struct)
- Modify: `pkg/morphology/lemma.go:4-9` (`LemmaRef` struct)
- Create: `pkg/morphology/multidict.go`
- Create: `pkg/morphology/multidict_test.go`

**Interfaces:**
- Produces: `Reading.Dict int`, `LemmaRef.Dict int` (both zero by default). `type MultiDictionary struct{...}`, `func NewMultiDictionary(dicts ...*Dictionary) *MultiDictionary`, `func (m *MultiDictionary) Len() int`, `func (m *MultiDictionary) DictInfo(i int) *BuildInfo`, `func (m *MultiDictionary) Parse(word string) []Reading`, `func (m *MultiDictionary) Lemma(word string) []LemmaRef`, `func (m *MultiDictionary) Close() error`.

This is one task because the struct fields have no independent test value without `MultiDictionary` to set them, and `MultiDictionary` cannot compile without the fields — they land and get reviewed together.

- [ ] **Step 1: Write the failing tests**

Create `pkg/morphology/multidict_test.go`:

```go
package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// twoDictFixture builds two independent pymorphy2 fixture dictionaries:
// dictA knows "кот" and "яблоко"; dictB knows "кот" and "груша". "кот" is
// deliberately present in both, to exercise the overlap case; "яблоко" and
// "груша" each exist in exactly one dictionary. dictA also carries
// meta.json (source_version/source_revision), dictB does not, to exercise
// DictInfo's SourceVersion for a populated vs. absent case side by side.
func twoDictFixture(t *testing.T) (dictA, dictB *morphology.Dictionary) {
	t.Helper()

	wordsA := map[string]uint32{}
	addWord(wordsA, "кот", 0, 0)
	addWord(wordsA, "яблоко", 2, 0)
	dirA := buildFixtureDir(t, wordsA, nil, nil)
	writeFile(t, dirA, "meta.json", []byte(`[["source_version","0.92"],["source_revision","417257"]]`))
	dictA, err := morphology.OpenPyMorphy(dirA)
	require.NoError(t, err)

	wordsB := map[string]uint32{}
	addWord(wordsB, "кот", 0, 0)
	addWord(wordsB, "груша", 2, 0)
	dirB := buildFixtureDir(t, wordsB, nil, nil)
	dictB, err = morphology.OpenPyMorphy(dirB)
	require.NoError(t, err)

	return dictA, dictB
}

func TestMultiDictionary_Len(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)
	assert.Equal(t, 2, m.Len())

	empty := morphology.NewMultiDictionary()
	assert.Equal(t, 0, empty.Len())
}

func TestMultiDictionary_ParseWordInOneDict(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	readings := m.Parse("яблоко")
	require.NotEmpty(t, readings)
	for _, r := range readings {
		assert.Equal(t, 0, r.Dict, "яблоко only exists in dictA (index 0)")
	}

	readings = m.Parse("груша")
	require.NotEmpty(t, readings)
	for _, r := range readings {
		assert.Equal(t, 1, r.Dict, "груша only exists in dictB (index 1)")
	}
}

func TestMultiDictionary_ParseWordInBothDicts(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	readings := m.Parse("кот")
	require.NotEmpty(t, readings)

	var dicts []int
	for _, r := range readings {
		dicts = append(dicts, r.Dict)
	}
	assert.Contains(t, dicts, 0, "кот exists in dictA")
	assert.Contains(t, dicts, 1, "кот exists in dictB")

	// Registration order: every dictA reading must precede every dictB
	// reading (Parse concatenates per-dictionary results in order, it
	// does not interleave or sort across dictionaries).
	lastA := -1
	firstB := len(readings)
	for i, r := range readings {
		if r.Dict == 0 && i > lastA {
			lastA = i
		}
		if r.Dict == 1 && i < firstB {
			firstB = i
		}
	}
	assert.Less(t, lastA, firstB, "all dictA readings must come before all dictB readings")
}

func TestMultiDictionary_ParseWordInNoDict(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	assert.Nil(t, m.Parse("несуществующееслово"))
}

func TestMultiDictionary_Lemma(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	refs := m.Lemma("кот")
	require.NotEmpty(t, refs)

	var dicts []int
	for _, r := range refs {
		dicts = append(dicts, r.Dict)
	}
	assert.Contains(t, dicts, 0)
	assert.Contains(t, dicts, 1)
}

func TestMultiDictionary_DictInfo(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	infoA := m.DictInfo(0)
	require.NotNil(t, infoA)
	assert.Equal(t, "pymorphy2", infoA.Source)
	assert.Equal(t, "0.92/417257", infoA.SourceVersion)

	infoB := m.DictInfo(1)
	require.NotNil(t, infoB)
	assert.Equal(t, "pymorphy2", infoB.Source)
	assert.Empty(t, infoB.SourceVersion, "dictB has no meta.json")

	assert.Nil(t, m.DictInfo(-1))
	assert.Nil(t, m.DictInfo(2))
}

func TestMultiDictionary_Close(t *testing.T) {
	dictA, dictB := twoDictFixture(t)
	m := morphology.NewMultiDictionary(dictA, dictB)

	assert.NoError(t, m.Close())
}

func TestMultiDictionary_Empty(t *testing.T) {
	m := morphology.NewMultiDictionary()

	assert.Nil(t, m.Parse("кот"))
	assert.Nil(t, m.Lemma("кот"))
	assert.NoError(t, m.Close())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/morphology/ -run TestMultiDictionary -v`
Expected: FAIL to compile — `morphology.NewMultiDictionary`, `MultiDictionary`, `Reading.Dict`, `LemmaRef.Dict` are all undefined.

- [ ] **Step 3: Add the `Dict` field to `Reading` and `LemmaRef`**

In `pkg/morphology/parse.go`, change the `Reading` struct:

```go
// Reading — one parse of a wordform.
type Reading struct {
	Word   string  // wordform as stored in the dictionary (with "ё")
	Normal string  // lemma (base form)
	Tag    string  // grammeme tag, e.g. "NOUN,anim,masc,sing,nomn"
	Para   uint16  // paradigm id — unique only together with Shard
	Form   uint16  // form index within the paradigm
	Shard  int     // dictionary shard index; always 0 for unsharded dictionaries
	Dict   int     // dictionary index in MultiDictionary; always 0 for Dictionary.Parse directly
	Prob   float64 // probability of this reading (0 if probability data is unavailable)
}
```

In `pkg/morphology/lemma.go`, change the `LemmaRef` struct:

```go
// LemmaRef — a reference to a lemma (base form): text, tag of paradigm
// form 0.
type LemmaRef struct {
	Normal string // lemma (base form)
	Tag    string // tag of the lemma (paradigm form 0)
	Para   uint16 // paradigm id — unique only together with Shard
	Shard  int    // dictionary shard index; always 0 for unsharded dictionaries
	Dict   int    // dictionary index in MultiDictionary; always 0 for Dictionary.Lemma directly
}
```

Both are additive — every existing construction site uses keyed struct literals (`readingForm` in `parse.go`, `Lemma` in `lemma.go`), so no other file needs to change for this step alone.

- [ ] **Step 4: Create `pkg/morphology/multidict.go`**

```go
// MultiDictionary aggregates Parse/Lemma/Close across an arbitrary set of
// already-open dictionaries. See
// docs/superpowers/specs/2026-09-16-multi-dict-design.md.
package morphology

import (
	"errors"
	"sync"
)

// MultiDictionary — a set of independently opened dictionaries, queried as
// a single whole. Each *Dictionary in the set retains its own lifecycle
// (mmap etc.) — MultiDictionary itself opens or imports nothing, it only
// aggregates Parse/Lemma and owns closing the whole set at once.
type MultiDictionary struct {
	dicts []*Dictionary
}

// NewMultiDictionary wraps already-open dictionaries into a single set.
// The order of dicts fixes the indexing of Reading.Dict/LemmaRef.Dict and
// the order in which Parse/Lemma results are concatenated — both always
// follow registration order and are never re-sorted.
func NewMultiDictionary(dicts ...*Dictionary) *MultiDictionary {
	return &MultiDictionary{dicts: dicts}
}

// Len returns the number of dictionaries in the set.
func (m *MultiDictionary) Len() int { return len(m.dicts) }

// DictInfo returns the diagnostic metadata of the dictionary at index i
// (the same index carried by Reading.Dict/LemmaRef.Dict), or nil if the
// index is out of range, or that dictionary has no info section (see
// Dictionary.Info).
func (m *MultiDictionary) DictInfo(i int) *BuildInfo {
	if i < 0 || i >= len(m.dicts) {
		return nil
	}
	return m.dicts[i].Info()
}

// Parse parses word across all dictionaries in the set in parallel (one
// goroutine per dictionary — the same pattern Dictionary.exact already
// uses for shards within a single dictionary). The result is the
// concatenation of each dictionary's Parse in the set's registration
// order, with Reading.Dict set; there is no sorting or deduplication
// across dictionaries beyond what each Dictionary.Parse already does
// internally. Returns nil if no dictionary produced any readings.
func (m *MultiDictionary) Parse(word string) []Reading {
	results := make([][]Reading, len(m.dicts))

	var wg sync.WaitGroup
	for i, d := range m.dicts {
		wg.Add(1)
		go func(i int, d *Dictionary) {
			defer wg.Done()
			readings := d.Parse(word)
			for j := range readings {
				readings[j].Dict = i
			}
			results[i] = readings
		}(i, d)
	}
	wg.Wait()

	var out []Reading
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}

// Lemma parses word across all dictionaries in the set and returns their
// base forms. Same registration-order-concatenation semantics as Parse.
func (m *MultiDictionary) Lemma(word string) []LemmaRef {
	results := make([][]LemmaRef, len(m.dicts))

	var wg sync.WaitGroup
	for i, d := range m.dicts {
		wg.Add(1)
		go func(i int, d *Dictionary) {
			defer wg.Done()
			refs := d.Lemma(word)
			for j := range refs {
				refs[j].Dict = i
			}
			results[i] = refs
		}(i, d)
	}
	wg.Wait()

	var out []LemmaRef
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}

// Close closes every dictionary in the set (Dictionary.Close is a no-op
// for dictionaries not opened via Open), aggregating all errors via
// errors.Join. After Close the set must not be used, nor its dictionaries.
func (m *MultiDictionary) Close() error {
	var errs []error
	for _, d := range m.dicts {
		if err := d.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./pkg/morphology/ -run TestMultiDictionary -v`
Expected: PASS (all 8 test functions).

- [ ] **Step 6: Full test + integration build + lint**

Run: `go test ./... -race`
Run: `go build -tags=integration ./...`
Run: `golangci-lint run ./...`
Expected: all green; lint shows only the 2 known pre-existing issues (`pkg/morphology/shard_test.go:67`, `pkg/morphology/importers/opencorpora/import.go:480`), nothing new.

- [ ] **Step 7: Commit**

```bash
git add pkg/morphology/parse.go pkg/morphology/lemma.go pkg/morphology/multidict.go pkg/morphology/multidict_test.go
git commit -m "feat(morphology): add MultiDictionary, aggregates Parse/Lemma/Close across N dictionaries

Reading/LemmaRef gain a Dict int index (same shape as the existing
Shard), zero-valued outside MultiDictionary. Overlapping results from
multiple dictionaries are returned in full, tagged by index, in
registration order - no merge/priority policy."
```

---

## Final Verification

- [ ] `go test ./... -race` — all green.
- [ ] `go build -tags=integration ./...` — compiles clean.
- [ ] `golangci-lint run ./...` — only the 2 known pre-existing issues.
- [ ] `go vet ./...` — clean.
