# pymorphy2 dense-alphabet recompile Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Recompile pymorphy2's `words.dawg` under gomorphy's dense 1-byte DAWG alphabet and wire it far enough into the read path that `Parse()` on the recompiled dictionary returns identical readings to `Parse()` on the raw pymorphy2 import.

**Architecture:** A new public `DAWG.Walk` method enumerates every (word, payload) pair in an existing DAWG by reusing the already-exported `ForEachChild`/`ValuesForIndex` primitives. `internal.Dictionary` gains an `Alphabet` field (nil by default — zero behavior change for every existing dictionary). `SimilarItems` gains an `Alphabet` parameter and a small `followRuneVia` helper that encodes each rune through the alphabet before following a DAWG edge (nil alphabet = today's raw-UTF-8 `FollowRune`, unchanged). A new `pymorphy2.RecompileDense` walks the raw import's `words.dawg`, builds a `DenseAlphabet` from the dictionary's own wordforms, re-encodes every key, and rebuilds via the already-existing `BuildDAWGWithValues`. A new `morphology.OpenPyMorphyDense` wraps it the same way `OpenPyMorphy` wraps `ImportFromDir`, so it's actually usable through the public API.

**Tech Stack:** Go (stdlib `encoding/binary`, `unicode/utf8`), `github.com/stretchr/testify` (assert/require, matching existing test style), `gofmt -r` for one mechanical signature-migration step.

**Spec:** [docs/en/superpowers/specs/2026-09-16-pymorphy2-dense-recompile-design.md](../specs/2026-09-16-pymorphy2-dense-recompile-design.md)

## Global Constraints

- Scope is `words.dawg` only. `Prediction`/`Probability` DAWGs, `fuzzy.go`, `.dat` serialization, and the `gomorphy_build` CLI are all untouched — do not touch them in this plan.
- `pymorphy2.ImportFromDir` is not modified in behavior — it must keep returning the raw UTF-8 passthrough dictionary exactly as today.
- `Dictionary.Alphabet == nil` must mean byte-for-byte the same behavior as before this plan, for every dictionary that doesn't explicitly set it (OpenCorpora, raw pymorphy2 via `ImportFromDir`).
- `DAWG.Walk` is a general-purpose public primitive on `internal.DAWG` — it must not take any pymorphy2-specific parameter or assumption.
- `go test ./... -race` and `go build -tags=integration ./...` must both stay green after every task.
- `golangci-lint run ./...` must report 0 issues on every task's touched packages.

---

## Task 1: `DAWG.Walk`

**Files:**
- Modify: `pkg/morphology/internal/dawg.go` (add `Walk` method after `HasPayloadChild`, currently ending at line 271)
- Modify: `pkg/morphology/internal/dawg_test.go` (add `TestDAWGWalk`)

**Interfaces:**
- Produces: `func (d *DAWG) Walk(fn func(key string, values [][]byte))` — visits every (key, values) pair stored in the DAWG. Later tasks (4) call this.

- [ ] **Step 1: Write the failing test**

Append to `pkg/morphology/internal/dawg_test.go`:

```go
func TestDAWGWalk(t *testing.T) {
	dict, guide := testdawg.Build(map[string]uint32{
		"кот" + string(PayloadSeparator) + base64.StdEncoding.EncodeToString([]byte{0, 0, 0, 1}):  0,
		"кот" + string(PayloadSeparator) + base64.StdEncoding.EncodeToString([]byte{0, 1, 0, 0}):  0,
		"кота" + string(PayloadSeparator) + base64.StdEncoding.EncodeToString([]byte{0, 2, 0, 0}): 0,
		"мышь" + string(PayloadSeparator) + base64.StdEncoding.EncodeToString([]byte{0, 3, 0, 0}): 0,
	})
	d := NewDAWG(dict, guide)

	found := map[string][][]byte{}
	d.Walk(func(key string, values [][]byte) {
		found[key] = append(found[key], values...)
	})

	require.Len(t, found, 3, "3 distinct words: кот, кота, мышь")
	require.Len(t, found["кот"], 2, "кот has 2 payload values (homonym)")
	assert.ElementsMatch(t, [][]byte{{0, 0, 0, 1}, {0, 1, 0, 0}}, found["кот"])
	require.Len(t, found["кота"], 1)
	assert.Equal(t, []byte{0, 2, 0, 0}, found["кота"][0])
	require.Len(t, found["мышь"], 1)
	assert.Equal(t, []byte{0, 3, 0, 0}, found["мышь"][0])
}

func TestDAWGWalkEmpty(t *testing.T) {
	dict, guide := testdawg.Build(map[string]uint32{})
	d := NewDAWG(dict, guide)

	calls := 0
	d.Walk(func(key string, values [][]byte) { calls++ })
	assert.Equal(t, 0, calls)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/morphology/internal/ -run TestDAWGWalk -v`
Expected: FAIL with "d.Walk undefined" (compile error).

- [ ] **Step 3: Write minimal implementation**

In `pkg/morphology/internal/dawg.go`, insert after `HasPayloadChild` (currently lines 260-271, right before `func b64d`):

```go
// Walk visits every (key, values) pair stored in the DAWG, in trie order.
// It reuses ForEachChild (edge traversal) and ValuesForIndex (payload
// enumeration under a PayloadSeparator edge) — the same primitives
// SimilarItems and ValuesForIndex already use for single-key lookups, just
// exhaustively instead of following a caller-given key. See
// docs/en/research/0005-pymorphy2-full-dawg-walk-cost.md for the validated
// approach and real-corpus timing (3,064,708 keys, 570ms).
func (d *DAWG) Walk(fn func(key string, values [][]byte)) {
	var walk func(index uint32, prefix []byte)
	walk = func(index uint32, prefix []byte) {
		d.ForEachChild(index, func(label byte, next uint32) {
			if label == PayloadSeparator {
				fn(string(prefix), d.ValuesForIndex(next))
				return
			}
			walk(next, append(prefix, label))
		})
	}
	walk(0, nil)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/morphology/internal/ -run TestDAWGWalk -v`
Expected: PASS (both `TestDAWGWalk` and `TestDAWGWalkEmpty`).

- [ ] **Step 5: Full package test + lint**

Run: `go test ./pkg/morphology/internal/... -race` and `golangci-lint run ./pkg/morphology/internal/...`
Expected: all green, 0 lint issues.

- [ ] **Step 6: Commit**

```bash
git add pkg/morphology/internal/dawg.go pkg/morphology/internal/dawg_test.go
git commit -m "feat(internal): add DAWG.Walk, a general full-trie enumeration primitive"
```

---

## Task 2: Alphabet-aware `SimilarItems`

**Files:**
- Modify: `pkg/morphology/internal/dictionary.go` (add `Alphabet` field)
- Modify: `pkg/morphology/internal/similar_items.go` (signature change + `followRuneVia`)
- Modify: `pkg/morphology/internal/similar_items_test.go` (fix existing calls, add new alphabet-aware tests)
- Modify (mechanical, via `gofmt -r`): `pkg/morphology/parse.go`, `pkg/morphology/importers/pymorphy2/import_test.go`, `pkg/morphology/importers/pymorphy2/full_dict_integration_test.go`, `pkg/morphology/importers/opencorpora/real_dict_integration_test.go`, `pkg/morphology/importers/opencorpora/import_test.go`, `pkg/morphology/internal/zz_debug_test.go`, `pkg/morphology/internal/dawgbuild_test.go`

**Interfaces:**
- Consumes: `DAWG.ForEachChild`, `DAWG.FollowByte` (existing, unchanged).
- Produces: `func (d *DAWG) SimilarItems(key string, pol *CharPolicy, alphabet Alphabet) []Item` (was 2-arg; every caller must now pass a third argument). `Dictionary.Alphabet Alphabet` field (nil default). Task 3 consumes both.

This task changes `SimilarItems`'s signature, which breaks every caller in the same commit — Go has no optional parameters, so the field addition, the signature change, and fixing all call sites must land together to keep the build green at the end of the task.

- [ ] **Step 1: Add the `Alphabet` field to `Dictionary`**

In `pkg/morphology/internal/dictionary.go`, change the struct:

```go
type Dictionary struct {
	Language    string
	TagSet      *TagSet
	Suffixes    [][]string
	Prefixes    []string
	Paradigms   [][]Paradigm
	Words       []*DAWG
	Prediction  []*DAWG
	Probability *DAWG
	CharPolicy  *CharPolicy
	Alphabet    Alphabet // nil = raw UTF-8 keys (today's behavior, unchanged)
	Info        *BuildInfo
}
```

(insert the `Alphabet` field between `CharPolicy` and `Info`; `NewDictionary`'s signature and body are unchanged — `Alphabet` stays nil until something sets it explicitly, same as `Prediction`/`Probability`/`Info` today).

- [ ] **Step 2: Write the failing tests for alphabet-aware `SimilarItems`**

Append to `pkg/morphology/internal/similar_items_test.go`:

```go
func TestSimilarItemsWithDenseAlphabet(t *testing.T) {
	corpus := []string{"кот", "кота", "мышь"}
	alphabet, err := NewDenseAlphabet(1, corpus)
	require.NoError(t, err)

	encode := func(s string) string {
		b, err := alphabet.Encode(s)
		require.NoError(t, err)
		return string(b)
	}

	dict, guide := testdawg.Build(map[string]uint32{
		encode("кот") + string(PayloadSeparator) + b64([]byte{1}):  0,
		encode("кота") + string(PayloadSeparator) + b64([]byte{2}): 0,
	})
	d := NewDAWG(dict, guide)

	items := d.SimilarItems("кот", NewCharPolicy(), alphabet)
	require.Len(t, items, 1)
	assert.Equal(t, "кот", items[0].Key)
	require.Len(t, items[0].Values, 1)
	assert.Equal(t, []byte{1}, items[0].Values[0])

	items = d.SimilarItems("кота", NewCharPolicy(), alphabet)
	require.Len(t, items, 1)
	assert.Equal(t, "кота", items[0].Key)

	items = d.SimilarItems("мышь", NewCharPolicy(), alphabet)
	assert.Empty(t, items, "мышь was not encoded into this DAWG")
}

func TestSimilarItemsWithDenseAlphabetAndCharPolicy(t *testing.T) {
	corpus := []string{"ежик", "ёжик"}
	alphabet, err := NewDenseAlphabet(1, corpus)
	require.NoError(t, err)

	encode := func(s string) string {
		b, err := alphabet.Encode(s)
		require.NoError(t, err)
		return string(b)
	}

	dict, guide := testdawg.Build(map[string]uint32{
		encode("ёжик") + string(PayloadSeparator) + b64([]byte{9}): 0,
	})
	d := NewDAWG(dict, guide)

	// "ежик" is not literally in the DAWG; the е→ё substitution in
	// CharPolicy, applied through alphabet.Encode, should find "ёжик".
	items := d.SimilarItems("ежик", RussianCharPolicy(), alphabet)
	require.Len(t, items, 1)
	assert.Equal(t, "ёжик", items[0].Key)
	require.Len(t, items[0].Values, 1)
	assert.Equal(t, []byte{9}, items[0].Values[0])
}

func TestSimilarItemsNilAlphabetUnchanged(t *testing.T) {
	// Same fixture and assertions as TestSimilarItemsFindsEAndYoVariants,
	// but passing alphabet explicitly as nil — guards the "nil means
	// identical to before" contract at the call-site level, not just by
	// inspection of followRuneVia.
	pol := RussianCharPolicy()
	dict, guide := testdawg.Build(map[string]uint32{
		"ежик" + string(PayloadSeparator) + b64([]byte{0, 1, 0, 2}): 1,
		"ёжик" + string(PayloadSeparator) + b64([]byte{3, 4, 5, 6}): 2,
	})
	d := NewDAWG(dict, guide)

	items := d.SimilarItems("ежик", pol, nil)
	require.Len(t, items, 2)
	assert.Equal(t, "ежик", items[0].Key)
	assert.Equal(t, "ёжик", items[1].Key)
}
```

Also fix the file's three pre-existing `SimilarItems(...)` calls (they currently pass 2 arguments) — append `, nil` to each:
- `d.SimilarItems("ежик", pol)` (two occurrences) → `d.SimilarItems("ежик", pol, nil)`
- `d.SimilarItems("мышь", pol)` → `d.SimilarItems("мышь", pol, nil)`

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./pkg/morphology/internal/ -run TestSimilarItems -v`
Expected: FAIL to compile — `SimilarItems` still takes 2 arguments; `NewDenseAlphabet`/`Alphabet` calls are fine (already exist from the harness), only the arity mismatch fails.

- [ ] **Step 4: Change `SimilarItems`'s signature and add `followRuneVia`**

Replace the full contents of `pkg/morphology/internal/similar_items.go`:

```go
package internal

import "unicode/utf8"

// Item is a search result: the found key and its payload values.
type Item struct {
	Key    string
	Values [][]byte
}

// SimilarItems looks up a key while accounting for CharPolicy character
// substitutions. For 'е→ё', it finds wordforms differing only in е/ё,
// e.g. "ежик" and "ёжик". alphabet encodes each rune before following an
// edge in the DAWG (nil = raw UTF-8, same behavior as the previous
// version); the returned Item.Key is always the original human-readable
// text and never needs decoding regardless of alphabet.
func (d *DAWG) SimilarItems(key string, pol *CharPolicy, alphabet Alphabet) []Item {
	return d.similarItemsRecursive("", []rune(key), 0, pol, alphabet)
}

// followRuneVia follows one rune r from index, encoding it via alphabet.
// alphabet == nil preserves the old behavior (FollowRune, raw UTF-8).
// Returns 0 if alphabet cannot encode r (a rune outside the corpus the
// alphabet was built from) — the same "no edge" contract as
// FollowByte/FollowRune.
func (d *DAWG) followRuneVia(alphabet Alphabet, r rune, index uint32) uint32 {
	if alphabet == nil {
		return d.FollowRune(r, index)
	}
	code, err := alphabet.Encode(string(r))
	if err != nil {
		return 0
	}
	return d.followBytes(code, index)
}

func (d *DAWG) similarItemsRecursive(prefix string, key []rune, index uint32, pol *CharPolicy, alphabet Alphabet) []Item {
	var items []Item

	startPos := utf8.RuneCountInString(prefix)
	endPos := len(key)
	wordPos := startPos

	for wordPos < endPos {
		r := key[wordPos]
		if pol != nil {
			for _, sub := range pol.Substitutions {
				if r == sub.From {
					if next := d.followRuneVia(alphabet, sub.To, index); next != 0 {
						newPrefix := prefix + string(key[startPos:wordPos]) + string(sub.To)
						items = append(items, d.similarItemsRecursive(newPrefix, key, next, pol, alphabet)...)
					}
				}
			}
		}
		if index = d.followRuneVia(alphabet, r, index); index == 0 {
			break
		}
		wordPos++
	}

	if wordPos == endPos {
		if sepIndex := d.FollowByte(PayloadSeparator, index); sepIndex != 0 {
			foundKey := prefix + string(key[startPos:])
			items = append([]Item{{Key: foundKey, Values: d.ValuesForIndex(sepIndex)}}, items...)
		}
	}

	return items
}
```

- [ ] **Step 5: Mechanically fix every other existing caller**

All other callers currently pass exactly 2 arguments (verified: no call site in the repo passes a 3rd argument yet). Use `gofmt -r` to append `, nil` everywhere in one shot — it rewrites at the AST level, so it's safe even for calls where the second argument itself contains parentheses (e.g. `RussianCharPolicy()`):

```bash
gofmt -r 'a.SimilarItems(b, c) -> a.SimilarItems(b, c, nil)' -w \
  pkg/morphology/parse.go \
  pkg/morphology/importers/pymorphy2/import_test.go \
  pkg/morphology/importers/pymorphy2/full_dict_integration_test.go \
  pkg/morphology/importers/opencorpora/real_dict_integration_test.go \
  pkg/morphology/importers/opencorpora/import_test.go \
  pkg/morphology/internal/zz_debug_test.go \
  pkg/morphology/internal/dawgbuild_test.go
```

Verify the rewrite touched every remaining call site and none was missed:

```bash
grep -rn "\.SimilarItems(" pkg/morphology/ | grep -v ", nil)" | grep -v ", alphabet)"
```

Expected: no output (every call site now ends in either `, nil)` or, in `similar_items_test.go`'s new tests from Step 2, `, alphabet)`).

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./pkg/morphology/... -race`
Run: `go build -tags=integration ./...` (compiles the `//go:build integration` files `gofmt -r` also touched, which plain `go build ./...` would silently skip)
Expected: all green.

- [ ] **Step 7: Lint**

Run: `golangci-lint run ./pkg/morphology/...`
Expected: 0 issues.

- [ ] **Step 8: Commit**

```bash
git add pkg/morphology/internal/dictionary.go pkg/morphology/internal/similar_items.go pkg/morphology/internal/similar_items_test.go pkg/morphology/parse.go pkg/morphology/importers/pymorphy2/import_test.go pkg/morphology/importers/pymorphy2/full_dict_integration_test.go pkg/morphology/importers/opencorpora/real_dict_integration_test.go pkg/morphology/importers/opencorpora/import_test.go pkg/morphology/internal/zz_debug_test.go pkg/morphology/internal/dawgbuild_test.go
git commit -m "feat(internal): thread an Alphabet through SimilarItems, add Dictionary.Alphabet

nil alphabet preserves today's raw-UTF-8 behavior exactly; every existing
caller is migrated to pass nil via gofmt -r, so this is a pure signature
extension with no behavior change until something sets Alphabet."
```

---

## Task 3: Wire `parse.go`'s `exactInShard` to `Dictionary.Alphabet`

**Files:**
- Modify: `pkg/morphology/parse.go:78-90` (`exactInShard`), `:144-152` (`predictForPrefix`)

**Interfaces:**
- Consumes: `Dictionary.Alphabet` (Task 2), `DAWG.SimilarItems(key, pol, alphabet)` (Task 2).

After Task 2's mechanical `gofmt -r` pass, both call sites currently read `..., x.d.CharPolicy, nil)`. This task changes only the `exactInShard` one (the one querying `d.Words`, which is in scope) to use `x.d.Alphabet` instead of the literal `nil`, and adds a comment to the `predictForPrefix` one (querying `d.Prediction`, out of scope) explaining why it deliberately keeps `nil`.

- [ ] **Step 1: Write the failing test**

This step has no *new* failing assertion by itself (existing dictionaries all have `Alphabet == nil`, so behavior is unchanged) — instead, write a test that would catch a future regression where `Alphabet` is set but not honored. Append to `pkg/morphology/internal/similar_items_test.go` is the wrong file (this needs a `Dictionary`+`Parse` level test, which lives in package `morphology`, not `internal`) — this is covered by Task 5's `TestOpenPyMorphyDense_MatchesRawParse`, which will fail at Task 5 time if this task's wiring is wrong or skipped. Skip a task-local test here; proceed directly to the change, and treat Task 5's round-trip test as this task's real regression guard.

- [ ] **Step 2: Make the change**

In `pkg/morphology/parse.go`, `exactInShard` (find the line `items := dawg.SimilarItems(word, x.d.CharPolicy, nil)` inside `func (x *Dictionary) exactInShard`):

```go
// exactInShard collects a word's readings from a single shard. It is
// called in parallel with other shards from exact — read-only, no mutable
// state shared between goroutines.
func (x *Dictionary) exactInShard(shard int, dawg *internal.DAWG, word string) shardExactResult {
	items := dawg.SimilarItems(word, x.d.CharPolicy, x.d.Alphabet)
	if len(items) == 0 {
		return shardExactResult{}
	}
```

In `predictForPrefix` (find the line `for _, it := range x.d.Prediction[id].SimilarItems(wordEnd, x.d.CharPolicy, nil) {`), add an explanatory comment immediately above it — leave the `nil` as-is:

```go
	for i := len(splits) - 1; i >= 0; i-- {
		wordStart, wordEnd := splits[i][0], splits[i][1]
		// Prediction DAWGs are never recompiled under Dictionary.Alphabet
		// (out of scope — see docs/en/superpowers/specs/2026-09-16-pymorphy2-dense-recompile-design.md's
		// non-goals): always nil here, even for a dictionary whose Words
		// DAWG uses a dense alphabet.
		for _, it := range x.d.Prediction[id].SimilarItems(wordEnd, x.d.CharPolicy, nil) {
			for _, v := range it.Values {
				if len(v) < 6 {
					continue
				}
```

- [ ] **Step 3: Run the existing parse test suite to confirm no regression**

Run: `go test ./pkg/morphology/... -race`
Expected: all green — every existing dictionary has `Alphabet == nil`, so `x.d.Alphabet` is `nil` everywhere today, identical to the literal `nil` it replaces.

- [ ] **Step 4: Lint**

Run: `golangci-lint run ./pkg/morphology/...`
Expected: 0 issues.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/parse.go
git commit -m "feat(morphology): honor Dictionary.Alphabet in exactInShard's word lookup

Prediction DAWG lookups stay hardcoded nil (out of scope, commented)."
```

---

## Task 4: `pymorphy2.RecompileDense`

**Files:**
- Create: `pkg/morphology/importers/pymorphy2/recompile.go`
- Create: `pkg/morphology/importers/pymorphy2/recompile_test.go`

**Interfaces:**
- Consumes: `ImportFromDir(dir) (*internal.Dictionary, error)` (existing), `DAWG.Walk` (Task 1), `internal.NewDenseAlphabet(width int, corpus []string) (*DenseAlphabet, error)` (existing), `Alphabet.Encode(s string) ([]byte, error)` (existing), `internal.BuildDAWGWithValues(keys []string, values []uint32) (*DAWG, error)` (existing), `Dictionary.Alphabet` (Task 2).
- Produces: `func RecompileDense(dir string) (*internal.Dictionary, error)`. Task 5 consumes this.

- [ ] **Step 1: Write the failing test**

Create `pkg/morphology/importers/pymorphy2/recompile_test.go` (reuses `makeFixtureDir` from `import_test.go`, same package):

```go
package pymorphy2_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/importers/pymorphy2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecompileDense(t *testing.T) {
	dir := makeFixtureDir(t)

	raw, err := pymorphy2.ImportFromDir(dir)
	require.NoError(t, err)

	dense, err := pymorphy2.RecompileDense(dir)
	require.NoError(t, err)
	require.NotNil(t, dense)
	require.NotNil(t, dense.Alphabet, "RecompileDense must set Alphabet")
	assert.Equal(t, "dense-1", dense.Alphabet.Name())

	for _, word := range []string{"кот", "кота"} {
		rawItems := raw.Words[0].SimilarItems(word, raw.CharPolicy, nil)
		denseItems := dense.Words[0].SimilarItems(word, dense.CharPolicy, dense.Alphabet)
		require.Len(t, denseItems, len(rawItems), "word %q", word)
		for i := range rawItems {
			assert.Equal(t, rawItems[i].Key, denseItems[i].Key, "word %q item %d", word, i)
			assert.Equal(t, rawItems[i].Values, denseItems[i].Values, "word %q item %d", word, i)
		}
	}

	// Everything except Words[0] must be copied through unchanged.
	assert.Equal(t, raw.Suffixes, dense.Suffixes)
	assert.Equal(t, raw.Prefixes, dense.Prefixes)
	assert.Equal(t, raw.Paradigms, dense.Paradigms)
	assert.Equal(t, raw.TagSet, dense.TagSet)
}

func TestRecompileDense_MissingDir(t *testing.T) {
	_, err := pymorphy2.RecompileDense(t.TempDir() + "/does-not-exist")
	require.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/morphology/importers/pymorphy2/ -run TestRecompileDense -v`
Expected: FAIL with "undefined: pymorphy2.RecompileDense" (compile error).

- [ ] **Step 3: Write the implementation**

Create `pkg/morphology/importers/pymorphy2/recompile.go`:

```go
// Package pymorphy2's RecompileDense rebuilds an imported dictionary's
// words.dawg under a dense 1-byte alphabet. See
// docs/en/superpowers/specs/2026-09-16-pymorphy2-dense-recompile-design.md.
package pymorphy2

import (
	"encoding/binary"
	"fmt"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// RecompileDense imports dir like ImportFromDir, then rebuilds Words[0]
// (and only Words[0] — Paradigms/Suffixes/Prefixes/Prediction/Probability
// are copied through unchanged, since they don't depend on words.dawg's
// key encoding) under a dense 1-byte alphabet built from the dictionary's
// own wordforms. The result's Parse() must return identical readings to
// ImportFromDir's, for any word the source dictionary itself resolves.
func RecompileDense(dir string) (*internal.Dictionary, error) {
	d, err := ImportFromDir(dir)
	if err != nil {
		return nil, err
	}

	var words []string
	var keys []string
	var values []uint32
	d.Words[0].Walk(func(word string, vals [][]byte) {
		words = append(words, word)
		for _, v := range vals {
			if len(v) < 4 {
				continue // matches Dictionary.reading's own guard, parse.go
			}
			keys = append(keys, word)
			values = append(values, binary.BigEndian.Uint32(v[:4]))
		}
	})

	alphabet, err := internal.NewDenseAlphabet(1, words)
	if err != nil {
		return nil, fmt.Errorf("pymorphy2: recompile: build alphabet: %w", err)
	}

	encodedKeys := make([]string, len(keys))
	for i, k := range keys {
		enc, err := alphabet.Encode(k)
		if err != nil {
			return nil, fmt.Errorf("pymorphy2: recompile: encode %q: %w", k, err)
		}
		encodedKeys[i] = string(enc)
	}

	dense, err := internal.BuildDAWGWithValues(encodedKeys, values)
	if err != nil {
		return nil, fmt.Errorf("pymorphy2: recompile: build DAWG: %w", err)
	}

	d.Words[0] = dense
	d.Alphabet = alphabet
	return d, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/morphology/importers/pymorphy2/ -run TestRecompileDense -v`
Expected: PASS.

- [ ] **Step 5: Full test + lint**

Run: `go test ./pkg/morphology/... -race`
Run: `golangci-lint run ./pkg/morphology/importers/pymorphy2/...`
Expected: all green, 0 lint issues.

- [ ] **Step 6: Commit**

```bash
git add pkg/morphology/importers/pymorphy2/recompile.go pkg/morphology/importers/pymorphy2/recompile_test.go
git commit -m "feat(pymorphy2): add RecompileDense, rebuilds words.dawg under a dense 1-byte alphabet"
```

---

## Task 5: `morphology.OpenPyMorphyDense` + round-trip `Parse()` test

**Files:**
- Modify: `pkg/morphology/open.go` (add `OpenPyMorphyDense`)
- Modify: `pkg/morphology/fixture_test.go` (extract `buildFixtureDir` out of `buildFixture`)
- Create: `pkg/morphology/dense_test.go`

**Interfaces:**
- Consumes: `pymorphy2.RecompileDense(dir) (*internal.Dictionary, error)` (Task 4).
- Produces: `func OpenPyMorphyDense(dir string) (*Dictionary, error)`.

- [ ] **Step 1: Refactor `buildFixture` to expose `buildFixtureDir`**

In `pkg/morphology/fixture_test.go`, replace the `buildFixture` function with:

```go
// buildFixtureDir assembles a pymorphy2 directory in t.TempDir() without
// opening it. words are the words.dawg keys; prediction is
// prediction-suffixes-0; prob is p_t_given_w.intdawg (nil means the file is
// not written).
func buildFixtureDir(t *testing.T, words, prediction, prob map[string]uint32) string {
	t.Helper()
	dir := t.TempDir()

	writeParadigms(t, dir, [][]uint16{
		{0, 1, 0, 1, 0, 0}, // paradigm 0: кот NOUN — ""(nomn) / "а"(gent)
		{0, 2, 0},          // paradigm 1: кот VERB — ""(VERB)
		{0, 1, 0, 1, 0, 0}, // paradigm 2: мышь NOUN
	})
	writeFile(t, dir, "suffixes.json", []byte(`["","а"]`))
	writeFile(t, dir, "paradigm-prefixes.json", []byte(`["","по","наи"]`))
	writeFile(t, dir, "gramtab-opencorpora-int.json", []byte(
		`["NOUN,anim,masc,sing,nomn","NOUN,anim,masc,sing,gent","VERB,impf,trans"]`,
	))

	wordsDAWG, guide := testdawg.Build(words)
	writeFile(t, dir, "words.dawg", testdawg.Marshal(wordsDAWG, guide))

	if len(prediction) > 0 {
		pDAWG, pGuide := testdawg.Build(prediction)
		writeFile(t, dir, "prediction-suffixes-0.dawg", testdawg.Marshal(pDAWG, pGuide))
	}
	if len(prob) > 0 {
		prDAWG, prGuide := testdawg.Build(prob)
		writeFile(t, dir, "p_t_given_w.intdawg", testdawg.Marshal(prDAWG, prGuide))
	}

	return dir
}

// buildFixture assembles a pymorphy2 directory via buildFixtureDir and
// opens it via OpenPyMorphy.
func buildFixture(t *testing.T, words, prediction, prob map[string]uint32) *morphology.Dictionary {
	t.Helper()
	dir := buildFixtureDir(t, words, prediction, prob)

	d, err := morphology.OpenPyMorphy(dir)
	require.NoError(t, err)
	require.NotNil(t, d)
	return d
}
```

- [ ] **Step 2: Run the existing fixture-based tests to confirm the refactor is behavior-preserving**

Run: `go test ./pkg/morphology/... -race`
Expected: all green (this step is a pure extraction — no test yet exercises `buildFixtureDir` or `OpenPyMorphyDense`).

- [ ] **Step 3: Write the failing round-trip test**

Create `pkg/morphology/dense_test.go`:

```go
package morphology_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenPyMorphyDense_MatchesRawParse(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	dir := buildFixtureDir(t, words, nil, nil)

	raw, err := morphology.OpenPyMorphy(dir)
	require.NoError(t, err)
	dense, err := morphology.OpenPyMorphyDense(dir)
	require.NoError(t, err)

	sample := []string{
		"кот", "кота", "мышь", "мыши", "ежик", "ёжик", "ёж",
		"код", "крот", "год", "дом", "дым", "стол", "стул", "лес",
	}
	for _, word := range sample {
		rawReadings := raw.Parse(word)
		denseReadings := dense.Parse(word)
		require.Equal(t, len(rawReadings), len(denseReadings), "word %q: reading count differs", word)
		for i := range rawReadings {
			assert.Equal(t, rawReadings[i].Word, denseReadings[i].Word, "word %q reading %d: Word", word, i)
			assert.Equal(t, rawReadings[i].Normal, denseReadings[i].Normal, "word %q reading %d: Normal", word, i)
			assert.Equal(t, rawReadings[i].Tag, denseReadings[i].Tag, "word %q reading %d: Tag", word, i)
			assert.Equal(t, rawReadings[i].Para, denseReadings[i].Para, "word %q reading %d: Para", word, i)
			assert.Equal(t, rawReadings[i].Form, denseReadings[i].Form, "word %q reading %d: Form", word, i)
		}
	}
}

func TestOpenPyMorphyDense_UnknownWordReturnsNil(t *testing.T) {
	words := map[string]uint32{}
	stdWords(words)
	dir := buildFixtureDir(t, words, nil, nil)

	dense, err := morphology.OpenPyMorphyDense(dir)
	require.NoError(t, err)
	assert.Nil(t, dense.Parse("несуществующееслово"))
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `go test ./pkg/morphology/ -run TestOpenPyMorphyDense -v`
Expected: FAIL with "undefined: morphology.OpenPyMorphyDense" (compile error).

- [ ] **Step 5: Add `OpenPyMorphyDense`**

In `pkg/morphology/open.go`, immediately after `OpenPyMorphy`:

```go
// OpenPyMorphyDense is like OpenPyMorphy, but recompiles words.dawg to a
// dense 1-byte alphabet before wrapping the dictionary into a Dictionary
// (see pymorphy2.RecompileDense and
// docs/en/superpowers/specs/2026-09-16-pymorphy2-dense-recompile-design.md).
func OpenPyMorphyDense(dir string) (*Dictionary, error) {
	d, err := pymorphy2.RecompileDense(dir)
	if err != nil {
		return nil, err
	}
	return &Dictionary{d: d}, nil
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./pkg/morphology/ -run TestOpenPyMorphyDense -v`
Expected: PASS (both tests).

- [ ] **Step 7: Full test + lint**

Run: `go test ./... -race`
Run: `golangci-lint run ./...`
Expected: all green, 0 lint issues.

- [ ] **Step 8: Commit**

```bash
git add pkg/morphology/open.go pkg/morphology/fixture_test.go pkg/morphology/dense_test.go
git commit -m "feat(morphology): add OpenPyMorphyDense; Parse() matches OpenPyMorphy on a fixture round-trip"
```

---

## Task 6: Real-corpus integration test (opt-in)

**Files:**
- Create: `pkg/morphology/dense_integration_test.go`

**Interfaces:**
- Consumes: `morphology.OpenPyMorphy`, `morphology.OpenPyMorphyDense` (Task 5).

This test is slow by nature — `RecompileDense` rebuilds `words.dawg` from scratch at real corpus scale (~3M keys), already documented elsewhere as a minutes-order operation (`docs/en/superpowers/specs/2026-09-15-dawg-alphabet-harness-design.md`). It is gated behind `//go:build integration` and an env var, matching `pkg/morphology/importers/pymorphy2/full_dict_integration_test.go`'s existing convention, so it never runs as part of a routine `go test ./...`.

- [ ] **Step 1: Write the test**

Create `pkg/morphology/dense_integration_test.go`:

```go
//go:build integration

// Compares Parse() between the raw pymorphy2 import and RecompileDense's
// dense-alphabet rebuild on a sample of real words. Needs a real pymorphy2
// dictionary directory (see pkg/pymorphy.Loader for how to get one — e.g.
// `.data/pymorphy/data` after a Loader.Sync(false)). Slow: run explicitly:
//
//	GOMORPHY_PYMORPHY2_DIR=.data/pymorphy/data \
//	  go test -tags=integration ./pkg/morphology/ -run TestOpenPyMorphyDense_RealCorpus -v -timeout 20m
package morphology_test

import (
	"os"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenPyMorphyDense_RealCorpus(t *testing.T) {
	dir := os.Getenv("GOMORPHY_PYMORPHY2_DIR")
	if dir == "" {
		t.Skip("GOMORPHY_PYMORPHY2_DIR not set")
	}

	raw, err := morphology.OpenPyMorphy(dir)
	require.NoError(t, err)
	dense, err := morphology.OpenPyMorphyDense(dir)
	require.NoError(t, err)

	sample := []string{
		"все", "занудами", "кот", "кота", "стол", "ежик", "ёжик",
		"пояснее", "поясней", "яснее", "ясней", "дом", "дома", "мама",
		"стали", "шли", "красивее",
	}
	for _, word := range sample {
		rawReadings := raw.Parse(word)
		denseReadings := dense.Parse(word)
		require.Equal(t, len(rawReadings), len(denseReadings), "word %q: reading count differs", word)
		for i := range rawReadings {
			assert.Equal(t, rawReadings[i].Word, denseReadings[i].Word, "word %q reading %d: Word", word, i)
			assert.Equal(t, rawReadings[i].Normal, denseReadings[i].Normal, "word %q reading %d: Normal", word, i)
			assert.Equal(t, rawReadings[i].Tag, denseReadings[i].Tag, "word %q reading %d: Tag", word, i)
		}
	}
}
```

- [ ] **Step 2: Run it against the real dictionary**

If `.data/pymorphy/data` isn't already populated, get it first (see `pkg/pymorphy.Loader.Sync`; this repo's session already has it downloaded from an earlier task). Then:

```bash
GOMORPHY_PYMORPHY2_DIR="$(pwd)/.data/pymorphy/data" go test -tags=integration ./pkg/morphology/ -run TestOpenPyMorphyDense_RealCorpus -v -timeout 20m
```

Expected: PASS (may take several minutes — `RecompileDense` rebuilds the full DAWG once inside the test).

- [ ] **Step 3: Confirm it's skipped by default**

Run: `go test ./pkg/morphology/...`
Expected: green, and this test doesn't even compile in (no `-tags=integration`), matching every other integration test in the repo.

- [ ] **Step 4: Lint**

Run: `golangci-lint run ./pkg/morphology/...`
Expected: 0 issues.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/dense_integration_test.go
git commit -m "test(morphology): add opt-in real-corpus round-trip test for OpenPyMorphyDense"
```

---

## Final Verification

- [ ] `go test ./... -race` — all green.
- [ ] `go build -tags=integration ./...` — compiles clean.
- [ ] `golangci-lint run ./...` — 0 issues.
- [ ] `go vet ./...` — clean.
- [ ] Manual sanity check: `GOMORPHY_PYMORPHY2_DIR="$(pwd)/.data/pymorphy/data" go test -tags=integration ./pkg/morphology/ -run TestOpenPyMorphyDense_RealCorpus -v -timeout 20m` passes on the real downloaded dictionary.
