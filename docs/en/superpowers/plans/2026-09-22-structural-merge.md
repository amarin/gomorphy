# Structural `Merge` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the entries-based `morphology.Merge` with an id-preserving structural merge. The merged dictionary keeps the base's prediction, probability, TagSet name (tagmap), reading order and size. Overlays apply as a fold (last wins in replace mode). EN and RU docs are brought up to date.

**Architecture:** A new `internal.MergeDictionaries` copies the base's TagSet/Prefixes/Suffixes/Paradigms with **unchanged ids** and appends overlay structures to them, remapping each overlay paradigm form-for-form (suffix/prefix text, tag name → output id). Only words DAWGs of shards that changed are rebuilt, and clean shards are cloned. The base's prediction DAWGs stay valid as-is because ids are stable. Probability is carried over (fast path) or filtered and rebuilt with a new value-DAWG builder. The public `Merge` keeps its signature. `MergeWithOptions` adds `RebuildPrediction`, and a compatibility check rejects mixing languages or two known tagmap vocabularies.

**Tech Stack:** Go 1.27, testify, cobra (CLI). No new dependencies.

**Spec:** [docs/en/superpowers/specs/2026-09-22-structural-merge-design.md](../specs/2026-09-22-structural-merge-design.md)

## Global Constraints

- Inputs are never mutated. The result must not alias any input's memory (mmap'd DAWGs are deep-copied via `Clone`).
- Base ids (tags, prefixes, per-shard suffixes and paradigms) are **stable**: output arrays start with the base arrays verbatim and only grow.
- Output is always dense (`*DenseAlphabet`, width 1 with width-2 fallback).
- Word identity = exact stored key (no CharPolicy е/ё substitution).
- Multiple overlays = fold: `Merge(b,[o1,o2],m) ≡ Merge(Merge(b,[o1],m),[o2],m)`.
- Output TagSet name = base's. Language mismatch, or two different tagmap-known TagSet names → `ErrIncompatibleDictionaries`.
- Performance on `pymorphy.dat` + small overlay: ≤ 40 s, peak RSS ≤ 1.5 GB, size ≤ base × 1.02.
- `go test ./...`, `go vet ./...`, `golangci-lint run ./...` green after every task. `gofmt -l` clean on touched files.
- Commit messages follow the repo style (`feat(morphology): …`, `fix(…)`, `docs: …`, `test(…)`) and end with the `Co-Authored-By` attribution line.

## File Structure

| File | Change | Responsibility |
|---|---|---|
| `pkg/morphology/tagmap/tagmap.go` | modify | `Known(name) bool` |
| `pkg/morphology/internal/dawg.go` | modify | `Clone`, `Empty`, `WalkValues` |
| `pkg/morphology/internal/dawgbuild.go` | modify | leaf values in `dbNode`, `BuildIntDAWG` |
| `pkg/morphology/internal/dawg_values_test.go` | create | tests for the above |
| `pkg/morphology/internal/alphabet.go` | modify | `(*DenseAlphabet).Runes` |
| `pkg/morphology/internal/prediction.go` | modify | `WordValue`, `BuildPredictionFrom` |
| `pkg/morphology/internal/merge.go` | create | the structural merge engine |
| `pkg/morphology/internal/merge_test.go` | create | engine tests |
| `pkg/morphology/merge.go` | rewrite | public API, compatibility checks |
| `pkg/morphology/merge_test.go`, `merge_internal_test.go` | modify | fold semantics, structure preservation |
| `pkg/morphology/merge_realdict_test.go` | create | gated real-data golden test |
| `cmd/gomorphy/merge.go`, `merge_test.go` | modify | `--rebuild-prediction` |
| `docs/en/cli.md`, `docs/en/library.md`, `pkg/morphology/dictionary.go` | modify | EN docs |
| `docs/ru/cli.md`, `docs/ru/library.md`, `README.md`, `CHANGELOG.md` | modify | RU docs, README, CHANGELOG |
| `docs/en/implementation/stage-19-builder-tsv-merge.md`, `docs/en/todo.md` | modify | write-up, roadmap close-out |

Mapping of the review findings to tasks: #1 prediction → Tasks 2–7; #2 probability → Tasks 2, 6, 7; #3 tagmap → Tasks 1, 7; #4 size/speed → Tasks 5, 6, 9; #5 fold semantics → Task 4 (engine) + Task 7 (public tests); #6 docs → Tasks 10–12.

---

### Task 0: Commit the pending `ImportTSV` empty-wordform fix

The working tree already contains the review fix: `ImportTSV` rejects an empty wordform column (`pkg/morphology/import_tsv.go`), with a test `TestImportTSVEmptyWordError` and a `docs/en/library.md:121` wording fix.

- [ ] **Step 1: Verify**

Run: `go test ./pkg/morphology/ -run ImportTSV`
Expected: PASS

- [ ] **Step 2: Commit**

```bash
git add pkg/morphology/import_tsv.go pkg/morphology/import_tsv_test.go docs/en/library.md
git commit -m "fix(morphology): reject an empty wordform column in ImportTSV"
```

---

### Task 1: `tagmap.Known`

**Files:**
- Modify: `pkg/morphology/tagmap/tagmap.go`
- Test: `pkg/morphology/tagmap/tagmap_test.go`

**Interfaces:**
- Produces: `func Known(dictName string) bool`

- [ ] **Step 1: Write the failing test** (append to `tagmap_test.go`, which is in package `tagmap_test`; add missing imports)

```go
func TestKnown(t *testing.T) {
	for _, name := range []string{"opencorpora", "opencorpora-int", "unimorph"} {
		assert.True(t, tagmap.Known(name), name)
	}
	for _, name := range []string{"", "builder", "tsv", "merge"} {
		assert.False(t, tagmap.Known(name), name)
	}
}
```

- [ ] **Step 2: Run it**: `go test ./pkg/morphology/tagmap/ -run TestKnown` → FAIL (`undefined: tagmap.Known`)

- [ ] **Step 3: Implement** (append to `tagmap.go`)

```go
// Known reports whether dictName (a dictionary's TagSet.Name) is a
// source this package can normalize — i.e. whether Map(dictName, …)
// returns ok=true.
func Known(dictName string) bool {
	_, ok := sources[dictName]
	return ok
}
```

- [ ] **Step 4: Run it**: `go test ./pkg/morphology/tagmap/` → PASS

- [ ] **Step 5: Commit**: `git commit -am "feat(tagmap): add Known for registered TagSet names"`

---

### Task 2: DAWG primitives — `Clone`, `Empty`, `WalkValues`, `BuildIntDAWG`

**Files:**
- Modify: `pkg/morphology/internal/dawg.go`, `pkg/morphology/internal/dawgbuild.go`
- Create: `pkg/morphology/internal/dawg_values_test.go`

**Interfaces:**
- Produces:
  - `func (d *DAWG) Clone() *DAWG` (nil-safe, deep copy)
  - `func (d *DAWG) Empty() bool` (nil-safe; true when there are no keys)
  - `func (d *DAWG) WalkValues(fn func(key string, value uint32))` (value DAWGs, e.g. `p_t_given_w`)
  - `func BuildIntDAWG(keys []string, values []uint32) (*DAWG, error)` (keys unique, any order; values `< 1<<31`)

Background: in dawgdic a terminal node's unit has `hasLeafBit`, and its value lives in the unit at `base` (child label 0): `Value(index)` reads `dict[index ^ offset]`. Today `place` always writes `isLeafBit` (value 0) there. The allocator already reserves the `base` slot (`slotAllocator.commit` marks `base`). Two things must change for non-zero values:
1. The minimization signature (`chainSig`) must distinguish leaves by value.
2. Base reuse through `p.link` (a node whose first child is a shared chain reuses the twin's base) must be keyed by value too, or two leaves with different values would share one value unit.

Payload DAWGs (words/prediction) have all values 0, so both changes are no-ops for them. Step 1 pins that down with a golden hash.

- [ ] **Step 1: Pin today's payload-DAWG bytes (guard against regressions)**

Create `dawg_values_test.go`:

```go
package internal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math/rand/v2"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
)

// payloadDAWGGolden is the SHA-256 of BuildDAWGWithValues over
// goldenPayloadKeys, captured before leaf values were added to the builder:
// words/prediction DAWG bytes must not change.
const payloadDAWGGolden = "CAPTURE-IN-STEP-2"

func goldenPayloadKeys() ([]string, []uint32) {
	r := rand.New(rand.NewPCG(7, 11))
	letters := []rune("абвгдеёжзик")
	seen := map[string]bool{}
	var keys []string
	var vals []uint32
	for len(keys) < 5000 {
		n := 1 + r.IntN(7)
		rs := make([]rune, n)
		for i := range rs {
			rs[i] = letters[r.IntN(len(letters))]
		}
		k := string(rs)
		if seen[k] {
			continue
		}
		seen[k] = true
		keys = append(keys, k)
		vals = append(vals, r.Uint32())
	}
	return keys, vals
}

func TestPayloadDAWGBytesStable(t *testing.T) {
	keys, vals := goldenPayloadKeys()
	d, err := BuildDAWGWithValues(keys, vals)
	require.NoError(t, err)
	sum := sha256.Sum256(d.Bytes())
	got := hex.EncodeToString(sum[:])
	t.Logf("payload DAWG sha256 = %s", got)
	assert.Equal(t, payloadDAWGGolden, got)
}
```

- [ ] **Step 2: Capture the golden value on the unchanged builder**

Run: `go test ./pkg/morphology/internal/ -run TestPayloadDAWGBytesStable -v`
Expected: FAIL, logging `payload DAWG sha256 = <hex>`. Paste that hex into `payloadDAWGGolden`, re-run → PASS.

- [ ] **Step 3: Write the failing tests for the new primitives** (append to `dawg_values_test.go`)

```go
func sortedKV(m map[string]uint32) ([]string, []uint32) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vals := make([]uint32, len(keys))
	for i, k := range keys {
		vals[i] = m[k]
	}
	return keys, vals
}

func TestBuildIntDAWGFindAndWalk(t *testing.T) {
	want := map[string]uint32{
		"кот:NOUN": 500, "кот:VERB": 100, "кота:NOUN": 500,
		"ab": 1, "cb": 2, "abc": 3, "b": 0, // shared suffixes with different values
	}
	keys, vals := sortedKV(want)
	d, err := BuildIntDAWG(keys, vals)
	require.NoError(t, err)

	for k, v := range want {
		assert.True(t, d.Contains(k), k)
		assert.Equal(t, v, d.Find(k), k)
	}
	assert.False(t, d.Contains("ко"))

	got := map[string]uint32{}
	d.WalkValues(func(k string, v uint32) { got[k] = v })
	assert.Equal(t, want, got)

	d2, err := ParseDAWG(d.Bytes())
	require.NoError(t, err)
	assert.Equal(t, uint32(500), d2.Find("кот:NOUN"))
}

func TestBuildIntDAWGRandom(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	letters := []rune("абвгд:")
	want := map[string]uint32{}
	for len(want) < 3000 {
		n := 1 + r.IntN(6)
		rs := make([]rune, n)
		for i := range rs {
			rs[i] = letters[r.IntN(len(letters))]
		}
		want[string(rs)] = r.Uint32N(1_000_000)
	}
	keys, vals := sortedKV(want)
	// Unsorted input must be accepted too.
	r.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i]; vals[i], vals[j] = vals[j], vals[i] })

	d, err := BuildIntDAWG(keys, vals)
	require.NoError(t, err)
	for k, v := range want {
		require.Equal(t, v, d.Find(k), k)
	}
	n := 0
	d.WalkValues(func(string, uint32) { n++ })
	assert.Equal(t, len(want), n)
}

func TestBuildIntDAWGErrors(t *testing.T) {
	_, err := BuildIntDAWG([]string{"a"}, nil)
	assert.Error(t, err, "length mismatch")
	_, err = BuildIntDAWG([]string{"a", "a"}, []uint32{1, 2})
	assert.Error(t, err, "duplicate key")
	_, err = BuildIntDAWG([]string{"a"}, []uint32{1 << 31})
	assert.Error(t, err, "value out of range")
}

func TestWalkValuesPymorphyIntDAWG(t *testing.T) {
	want := map[string]uint32{"кот:NOUN": 500, "кота:NOUN": 7}
	dict, guide := testdawg.Build(want)
	d, err := ReadDAWG(bytes.NewReader(testdawg.Marshal(dict, guide)))
	require.NoError(t, err)
	got := map[string]uint32{}
	d.WalkValues(func(k string, v uint32) { got[k] = v })
	assert.Equal(t, want, got)
}

func TestDAWGCloneAndEmpty(t *testing.T) {
	d, err := BuildDAWGWithValues([]string{"кот"}, []uint32{7})
	require.NoError(t, err)
	c := d.Clone()
	assert.Equal(t, d.Bytes(), c.Bytes())
	c.dict[0] ^= 1
	assert.NotEqual(t, d.Bytes(), c.Bytes(), "clone must not alias the original")
	assert.False(t, d.Empty())

	e, err := BuildDAWG(nil)
	require.NoError(t, err)
	assert.True(t, e.Empty())

	var nilDAWG *DAWG
	assert.Nil(t, nilDAWG.Clone())
	assert.True(t, nilDAWG.Empty())
}
```

- [ ] **Step 4: Run them**: `go test ./pkg/morphology/internal/ -run 'IntDAWG|WalkValues|CloneAndEmpty'` → FAIL (undefined symbols)

- [ ] **Step 5: Implement in `dawg.go`** (add `"slices"` to the imports)

```go
// Clone returns a deep copy of d whose arrays don't alias d's — so a
// DAWG parsed from an mmap'd file can outlive the mapping. nil-safe.
func (d *DAWG) Clone() *DAWG {
	if d == nil {
		return nil
	}
	return &DAWG{dict: slices.Clone(d.dict), guide: slices.Clone(d.guide)}
}

// Empty reports whether the DAWG holds no keys (nil-safe).
func (d *DAWG) Empty() bool {
	if d == nil || len(d.guide) == 0 {
		return true
	}
	return guideChild(d.guide, 0) == 0
}

// WalkValues visits every key of a value DAWG (dawgdic IntDAWG layout, e.g.
// pymorphy2's p_t_given_w.intdawg or a BuildIntDAWG result) with its
// integer value, in trie order. Payload DAWGs (words.dawg) should use Walk.
func (d *DAWG) WalkValues(fn func(key string, value uint32)) {
	var walk func(index uint32, prefix []byte)
	walk = func(index uint32, prefix []byte) {
		if index != 0 && d.HasValue(index) {
			fn(string(prefix), d.Value(index))
		}
		d.ForEachChild(index, func(label byte, next uint32) {
			walk(next, append(prefix, label))
		})
	}
	walk(0, nil)
}
```

- [ ] **Step 6: Implement in `dawgbuild.go`**

6a. `dbNode` gets a value:

```go
type dbNode struct {
	label byte
	first int32  // first child (head of the chain); 0 — no children
	next  int32  // next sibling in the parent's chain; 0 — none
	leaf  bool   // the node is terminal (has a value)
	value uint32 // the terminal's value (BuildIntDAWG); 0 for payload DAWGs
}
```

6b. Rename `insertKeys` to `insertKeyValues(keys []string, values []uint32, onInserted func(i int))`. After `b.nodes[b.path[len(b.path)-1]].leaf = true`, add:

```go
		if values != nil {
			b.nodes[b.path[len(b.path)-1]].value = values[i]
		}
```

Keep `insertKeys` as a thin wrapper so existing callers (`buildDAWGWithPayload`, `BuildDAWGWithValuesProgress`) don't change:

```go
func (b *dawgBuilder) insertKeys(keys []string, onInserted func(i int)) {
	b.insertKeyValues(keys, nil, onInserted)
}
```

6c. In `chainSig`, after computing `f` and before appending it, fold the value in, but only when non-zero (so payload DAWG signatures stay byte-identical):

```go
		if b.nodes[n].next != 0 {
			f |= 2
		}
		v := b.nodes[n].value
		if v != 0 {
			f |= 4
		}
		b.sigBuf = append(b.sigBuf, f)
		if v != 0 {
			b.sigBuf = append(b.sigBuf, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
		}
```

6d. Key base reuse by value. In `placer`, change `link map[int32]uint32` to `link map[linkKey]uint32` (and `make(map[linkKey]uint32)` in `newPlacer`), with:

```go
// linkKey identifies a reusable base: the shared first child plus the
// value stored in the base's value unit (0 for payload DAWGs, so their
// layout is unchanged).
type linkKey struct {
	first int32
	value uint32
}
```

In `place`, compute `lk := linkKey{first: first, value: node.value}` and use `p.link[lk]` in both the lookup and the store. Write the value unit with the value:

```go
	if node.leaf {
		setAt(&p.dic, base, isLeafBit|node.value)
	}
```

6e. Add the public builder (next to `BuildDAWGWithValuesBytes`):

```go
// BuildIntDAWG builds a dawgdic value DAWG (the IntDAWG layout pymorphy2
// uses for p_t_given_w.intdawg): every key maps to a uint32 value stored
// in its terminal node, read back with Find/WalkValues. keys must be
// unique (any order); every value must be < 1<<31 (the top bit is the
// dawgdic leaf flag).
func BuildIntDAWG(keys []string, values []uint32) (*DAWG, error) {
	if len(keys) != len(values) {
		return nil, fmt.Errorf("dawg: keys and values must have same length")
	}
	order := make([]int, len(keys))
	for i := range order {
		order[i] = i
		if values[i] >= isLeafBit {
			return nil, fmt.Errorf("dawg: value %d for key %q exceeds 31 bits", values[i], keys[i])
		}
	}
	sort.Slice(order, func(a, b int) bool { return keys[order[a]] < keys[order[b]] })
	sk := make([]string, len(keys))
	sv := make([]uint32, len(keys))
	for i, j := range order {
		sk[i], sv[i] = keys[j], values[j]
		if i > 0 && sk[i] == sk[i-1] {
			return nil, fmt.Errorf("dawg: duplicate key %q", sk[i])
		}
	}
	b := newDawgBuilder()
	b.insertKeyValues(sk, sv, nil)
	return b.compile()
}
```

- [ ] **Step 7: Run all internal tests** (the golden hash included): `go test ./pkg/morphology/internal/` → PASS

- [ ] **Step 8: Commit**

```bash
git add pkg/morphology/internal/dawg.go pkg/morphology/internal/dawgbuild.go pkg/morphology/internal/dawg_values_test.go
git commit -m "feat(internal): add value DAWGs (BuildIntDAWG, WalkValues) and DAWG Clone/Empty"
```

---

### Task 3: `DenseAlphabet.Runes` and `BuildPredictionFrom`

**Files:**
- Modify: `pkg/morphology/internal/alphabet.go`, `pkg/morphology/internal/prediction.go`
- Test: `pkg/morphology/internal/prediction_test.go`, `pkg/morphology/internal/alphabet_test.go` (append to the existing files; create them if absent, in package `internal`)

**Interfaces:**
- Produces:
  - `func (a *DenseAlphabet) Runes() []rune` (a copy, in code order)
  - `type WordValue struct { Word string; Value uint32 }` (plain-text word, `para<<16|form`)
  - `func BuildPredictionFrom(pairs []WordValue, paradigms []Paradigm, tagSet *TagSet, productive func(tag string) bool) (*DAWG, error)`
  - `BuildPrediction(d, productive)` keeps its signature and behavior, now a wrapper.

- [ ] **Step 1: Failing tests**

```go
func TestDenseAlphabetRunes(t *testing.T) {
	a, err := NewDenseAlphabet(1, []string{"ба", "в"})
	require.NoError(t, err)
	assert.Equal(t, []rune("абв"), a.Runes())
	r := a.Runes()
	r[0] = 'z'
	assert.Equal(t, []rune("абв"), a.Runes(), "Runes returns a copy")
}

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
	pred, err := BuildPredictionFrom(pairs, d.Paradigms[0], d.TagSet, all)
	require.NoError(t, err)

	require.NoError(t, BuildPrediction(d, all))
	assert.Equal(t, d.Prediction[0].Bytes(), pred.Bytes())
}
```

- [ ] **Step 2: Run**: `go test ./pkg/morphology/internal/ -run 'Runes|BuildPredictionFrom'` → FAIL

- [ ] **Step 3: Implement.** `alphabet.go` (add `"slices"` import):

```go
// Runes returns the alphabet's runes in code order (code 2 first) — a
// copy, safe to modify. Feeding string(a.Runes()) back into
// NewDenseAlphabet reproduces a superset-compatible corpus.
func (a *DenseAlphabet) Runes() []rune {
	return slices.Clone(a.runeOf)
}
```

`prediction.go`: replace the body of `BuildPrediction` with the wrapper and move the counting into `BuildPredictionFrom`:

```go
// WordValue is one raw words.dawg reading: a plain-text wordform and its
// (paradigm<<16 | form) value.
type WordValue struct {
	Word  string
	Value uint32
}

// BuildPrediction rebuilds d.Prediction from d.Words[0] (see
// BuildPredictionFrom). The dictionary must be unsharded and still raw
// (plain-text keys); a sharded dictionary is left untouched (no-op).
func BuildPrediction(d *Dictionary, productive func(tag string) bool) error {
	if d == nil || len(d.Words) != 1 {
		return nil
	}
	var pairs []WordValue
	d.Words[0].Walk(func(word string, values [][]byte) {
		for _, v := range values {
			if len(v) >= 4 {
				pairs = append(pairs, WordValue{Word: word, Value: binary.BigEndian.Uint32(v[:4])})
			}
		}
	})
	var paradigms []Paradigm
	if len(d.Paradigms) > 0 {
		paradigms = d.Paradigms[0]
	}
	pred, err := BuildPredictionFrom(pairs, paradigms, d.TagSet, productive)
	if err != nil {
		return err
	}
	d.Prediction = []*DAWG{pred}
	return nil
}

// BuildPredictionFrom builds the pymorphy2 KnownSuffixAnalyzer prediction
// DAWG for prefix id 0 from raw (word, value) readings resolved against
// paradigms (shard 0) and tagSet. For every reading whose tag is
// productive, the word's last 1..5 runes become suffix keys; readings
// sharing a (suffix, paradigm, form) triple accumulate a count. Each
// triple becomes one payload: count(BE16) + para(BE16) + form(BE16).
func BuildPredictionFrom(pairs []WordValue, paradigms []Paradigm, tagSet *TagSet, productive func(tag string) bool) (*DAWG, error) {
	type predKey struct {
		suffix string
		para   uint16
		form   uint16
	}
	counts := make(map[predKey]int)
	for _, p := range pairs {
		para, form := uint16(p.Value>>16), uint16(p.Value)
		if int(para) >= len(paradigms) || int(form) >= paradigms[para].Len() {
			continue
		}
		tag := ""
		if tagSet != nil {
			tag = tagSet.TagName(paradigms[para].Tag(int(form)))
		}
		if !productive(tag) {
			continue
		}
		rr := []rune(p.Word)
		max := min(predictionMaxSuffix, len(rr))
		for l := 1; l <= max; l++ {
			counts[predKey{suffix: string(rr[len(rr)-l:]), para: para, form: form}]++
		}
	}

	keys := make([]string, 0, len(counts))
	values := make([][]byte, 0, len(counts))
	for k, count := range counts {
		count = min(count, predictionMaxCount)
		buf := make([]byte, 6)
		binary.BigEndian.PutUint16(buf[:2], uint16(count))
		binary.BigEndian.PutUint16(buf[2:4], k.para)
		binary.BigEndian.PutUint16(buf[4:6], k.form)
		keys = append(keys, k.suffix)
		values = append(values, buf)
	}
	return BuildDAWGWithValuesBytes(keys, values)
}
```

- [ ] **Step 4: Run**: `go test ./pkg/morphology/...` → PASS (the existing prediction tests still pass through the wrapper)

- [ ] **Step 5: Commit**: `git commit -am "refactor(internal): split BuildPredictionFrom out of BuildPrediction; add DenseAlphabet.Runes"`

---

### Task 4: Merge engine, part 1: overlay decision with fold semantics (finding #5)

**Files:**
- Create: `pkg/morphology/internal/merge.go`, `pkg/morphology/internal/merge_test.go`

**Interfaces:**
- Produces (used by Tasks 5–6):
  - `type MergeMode int`; `const MergeAdd, MergeReplace`
  - `type overlayReading struct { shard int; para, form uint16 }`
  - `type winner struct { overlay int; readings []overlayReading }`
  - `func decideOverlays(base *Dictionary, overlays []*Dictionary, mode MergeMode) (map[string]*winner, error)`
  - `func wordReadings(d *Dictionary) (map[string][]overlayReading, error)`
  - `func hasWord(d *Dictionary, word string) bool`, `func shardHasWord(w *DAWG, a Alphabet, word string) bool`
  - `func decodeKey(a Alphabet, key string) (string, error)`, `func encodeKey(a Alphabet, word string) (string, bool)`

- [ ] **Step 1: Failing tests** (`internal/merge_test.go`)

```go
package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// engineDict builds a dense single-shard dictionary from entries.
func engineDict(t *testing.T, entries ...BuildEntry) *Dictionary {
	t.Helper()
	d, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)
	require.NoError(t, RecompileDense(d))
	return d
}

func e(word, lemma, tag string) BuildEntry { return BuildEntry{Word: word, Lemma: lemma, Tag: tag} }

func TestDecideOverlaysFold(t *testing.T) {
	base := engineDict(t, e("кот", "кот", "B"))
	o1 := engineDict(t, e("кот", "кот", "O1"), e("пёс", "пёс", "O1"))
	o2 := engineDict(t, e("кот", "кот", "O2"), e("пёс", "пёс", "O2"))
	overlays := []*Dictionary{o1, o2}

	add, err := decideOverlays(base, overlays, MergeAdd)
	require.NoError(t, err)
	assert.NotContains(t, add, "кот", "add: a base word is never taken")
	require.Contains(t, add, "пёс")
	assert.Equal(t, 0, add["пёс"].overlay, "add: the first overlay to supply a new word wins")

	rep, err := decideOverlays(base, overlays, MergeReplace)
	require.NoError(t, err)
	assert.Equal(t, 1, rep["кот"].overlay, "replace: the last overlay wins over the base")
	assert.Equal(t, 1, rep["пёс"].overlay, "replace: the last overlay wins over an earlier overlay")
}

func TestHasWordExactKey(t *testing.T) {
	d := engineDict(t, e("ёж", "ёж", "N"))
	assert.True(t, hasWord(d, "ёж"))
	assert.False(t, hasWord(d, "еж"), "no CharPolicy substitution")
	assert.False(t, hasWord(d, "ё"), "a key prefix is not a word")
	assert.False(t, hasWord(d, "wifi"), "a word the alphabet can't encode is absent")
}

func TestWordReadingsDecodesDenseKeys(t *testing.T) {
	d := engineDict(t, e("кот", "кот", "N"), e("кота", "кот", "G"))
	rs, err := wordReadings(d)
	require.NoError(t, err)
	assert.Len(t, rs["кот"], 1)
	assert.Len(t, rs["кота"], 1)
	assert.Equal(t, uint16(1), rs["кота"][0].form)
}
```

- [ ] **Step 2: Run**: `go test ./pkg/morphology/internal/ -run 'DecideOverlays|HasWord|WordReadings'` → FAIL

- [ ] **Step 3: Implement** `internal/merge.go` (the first part; later tasks append to it)

```go
package internal

import (
	"encoding/binary"
	"fmt"
)

// MergeMode is MergeDictionaries' word-level conflict policy; values
// mirror the public morphology.MergeMode one for one.
type MergeMode int

const (
	// MergeAdd takes an overlay word only if neither the base nor an
	// earlier overlay has it.
	MergeAdd MergeMode = iota
	// MergeReplace lets an overlay word's readings replace whatever the
	// word had so far (base or earlier overlay) — the last overlay wins.
	MergeReplace
)

// overlayReading is one reading of an overlay wordform, in the overlay's
// own id space.
type overlayReading struct {
	shard      int
	para, form uint16
}

// winner is the overlay whose readings a word gets in the output.
type winner struct {
	overlay  int
	readings []overlayReading
}

// decideOverlays folds overlays over the base in order and returns, for
// every word an overlay supplies to the output, the winning overlay and
// its readings. Merge(b,[o1,o2]) == Merge(Merge(b,[o1]),[o2]).
func decideOverlays(base *Dictionary, overlays []*Dictionary, mode MergeMode) (map[string]*winner, error) {
	winners := make(map[string]*winner)
	for oi, o := range overlays {
		words, err := wordReadings(o)
		if err != nil {
			return nil, fmt.Errorf("overlay %d: %w", oi, err)
		}
		for word, rs := range words {
			if mode == MergeAdd {
				if _, taken := winners[word]; taken || hasWord(base, word) {
					continue
				}
			}
			winners[word] = &winner{overlay: oi, readings: rs}
		}
	}
	return winners, nil
}

// wordReadings enumerates every wordform of d (decoded to plain text)
// with its readings, across all shards.
func wordReadings(d *Dictionary) (map[string][]overlayReading, error) {
	out := make(map[string][]overlayReading)
	for s, w := range d.Words {
		if w == nil {
			continue
		}
		var walkErr error
		w.Walk(func(key string, vals [][]byte) {
			if walkErr != nil {
				return
			}
			word, err := decodeKey(d.Alphabet, key)
			if err != nil {
				walkErr = err
				return
			}
			for _, v := range vals {
				if len(v) < 4 {
					continue
				}
				out[word] = append(out[word], overlayReading{
					shard: s,
					para:  binary.BigEndian.Uint16(v[:2]),
					form:  binary.BigEndian.Uint16(v[2:4]),
				})
			}
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	return out, nil
}

// decodeKey turns a stored words.dawg key into plain text.
func decodeKey(a Alphabet, key string) (string, error) {
	if a == nil {
		return key, nil
	}
	word, err := a.Decode([]byte(key))
	if err != nil {
		return "", fmt.Errorf("decode DAWG key: %w", err)
	}
	return word, nil
}

// encodeKey turns plain text into a words.dawg key; ok is false when the
// alphabet can't encode it.
func encodeKey(a Alphabet, word string) (string, bool) {
	if a == nil {
		return word, true
	}
	b, err := a.Encode(word)
	if err != nil {
		return "", false
	}
	return string(b), true
}

// hasWord reports whether word is an exact key of any of d's shards (no
// CharPolicy substitution).
func hasWord(d *Dictionary, word string) bool {
	for _, w := range d.Words {
		if shardHasWord(w, d.Alphabet, word) {
			return true
		}
	}
	return false
}

// shardHasWord reports whether word is an exact key of one words DAWG.
func shardHasWord(w *DAWG, a Alphabet, word string) bool {
	if w == nil || word == "" {
		return false
	}
	key, ok := encodeKey(a, word)
	if !ok {
		return false
	}
	idx := w.Follow(key, 0)
	return idx != 0 && w.HasPayloadChild(idx)
}
```

- [ ] **Step 4: Run**: `go test ./pkg/morphology/internal/` → PASS

- [ ] **Step 5: Commit**: `git commit -am "feat(internal): merge engine overlay decision with fold semantics"` (add the new files first)

---

### Task 5: Merge engine, part 2: id-preserving paradigm remap

**Files:**
- Modify: `pkg/morphology/internal/merge.go`, `pkg/morphology/internal/merge_test.go`

**Interfaces:**
- Consumes: `overlayReading`, `wordReadings` (Task 4); `paradigmLimit`, `suffixShardLimit`, `encodeU16s` (`build.go`)
- Produces:
  - `var mergeSuffixLimit = suffixShardLimit`
  - `type mergeShard struct { suffixes []string; suffixIdx map[string]uint16; paradigms []Paradigm; paraIdx map[string]uint16; base *DAWG; placed []WordValue; dirty bool }`
  - `type merger struct { tagSet *TagSet; prefixes []string; prefixIdx map[string]uint16; shards []*mergeShard; cache map[paraRef]paraLoc }`
  - `type paraLoc struct { shard int; para uint16 }`
  - `func newMerger(base *Dictionary) *merger`
  - `func (m *merger) place(oi int, o *Dictionary, r overlayReading) (paraLoc, error)`

- [ ] **Step 1: Failing tests** (append to `internal/merge_test.go`)

```go
func TestPlaceRemapsIntoBaseIDSpace(t *testing.T) {
	base := engineDict(t, e("кот", "кот", "N,nomn"), e("кота", "кот", "N,gent"))
	baseTags := append([]string(nil), base.TagSet.Tags...)
	baseSuffixes := append([]string(nil), base.Suffixes[0]...)
	nBase := len(base.Paradigms[0])

	o := engineDict(t, e("кит", "кит", "N,nomn"), e("кита", "кит", "N,gent"), e("ура", "ура", "INTJ"))
	rs, err := wordReadings(o)
	require.NoError(t, err)

	m := newMerger(base)
	loc, err := m.place(0, o, rs["кита"][0])
	require.NoError(t, err)
	assert.Equal(t, paraLoc{shard: 0, para: 0}, loc, "a paradigm shaped like base paradigm 0 dedups onto it")
	assert.Len(t, m.shards[0].paradigms, nBase)

	again, err := m.place(0, o, rs["кит"][0])
	require.NoError(t, err)
	assert.Equal(t, loc, again, "same overlay paradigm → cached location")

	loc2, err := m.place(0, o, rs["ура"][0])
	require.NoError(t, err)
	assert.Equal(t, paraLoc{shard: 0, para: uint16(nBase)}, loc2, "a new shape is appended")

	assert.Equal(t, baseTags, m.tagSet.Tags[:len(baseTags)], "base tag ids are stable")
	assert.Contains(t, m.tagSet.Tags, "INTJ")
	assert.Equal(t, baseSuffixes, m.shards[0].suffixes[:len(baseSuffixes)], "base suffix ids are stable")
	assert.Equal(t, baseTags, base.TagSet.Tags, "the base TagSet is not mutated")
	assert.Equal(t, baseSuffixes, base.Suffixes[0], "the base suffixes are not mutated")
}

func TestPlaceOpensNewShardOnSuffixOverflow(t *testing.T) {
	old := mergeSuffixLimit
	mergeSuffixLimit = 2
	t.Cleanup(func() { mergeSuffixLimit = old })

	base := engineDict(t, e("кот", "кот", "N"), e("кота", "кот", "G")) // suffixes "", "а"
	o := engineDict(t, e("стол", "стол", "N"), e("столом", "стол", "I")) // needs new suffix "ом"
	rs, err := wordReadings(o)
	require.NoError(t, err)

	m := newMerger(base)
	loc, err := m.place(0, o, rs["столом"][0])
	require.NoError(t, err)
	assert.Equal(t, 1, loc.shard)
	assert.Len(t, m.shards, 2)
	assert.Nil(t, m.shards[1].base, "an appended shard has no base DAWG")
}
```

- [ ] **Step 2: Run**: `go test ./pkg/morphology/internal/ -run Place` → FAIL

- [ ] **Step 3: Implement** (append to `internal/merge.go`; add `"slices"` to the imports)

```go
// mergeSuffixLimit caps a merge target shard's suffix count before a new
// shard is opened. A var (not suffixShardLimit directly) so tests can
// lower it.
var mergeSuffixLimit = suffixShardLimit

// mergeShard is one output shard under construction: the base shard's
// suffixes/paradigms verbatim (ids stable) plus whatever the overlays
// append, and the overlay readings placed into it.
type mergeShard struct {
	suffixes  []string
	suffixIdx map[string]uint16 // lazily built text → id
	paradigms []Paradigm
	paraIdx   map[string]uint16 // lazily built paradigm data key → id
	base      *DAWG             // the base shard's words DAWG; nil for an appended shard
	placed    []WordValue       // overlay readings targeting this shard
	dirty     bool              // the words DAWG must be rebuilt
}

type paraRef struct {
	overlay, shard int
	para           uint16
}

type paraLoc struct {
	shard int
	para  uint16
}

// merger holds the output id spaces while overlay paradigms are remapped
// into them.
type merger struct {
	tagSet    *TagSet
	prefixes  []string
	prefixIdx map[string]uint16
	shards    []*mergeShard
	cache     map[paraRef]paraLoc
}

func newMerger(base *Dictionary) *merger {
	m := &merger{
		tagSet:    cloneTagSet(base.TagSet),
		prefixes:  slices.Clone(base.Prefixes),
		prefixIdx: make(map[string]uint16, len(base.Prefixes)),
		cache:     make(map[paraRef]paraLoc),
	}
	if len(m.prefixes) == 0 {
		m.prefixes = []string{""}
	}
	for i, p := range m.prefixes {
		if _, ok := m.prefixIdx[p]; !ok {
			m.prefixIdx[p] = uint16(i)
		}
	}
	for i, w := range base.Words {
		s := &mergeShard{base: w}
		if i < len(base.Suffixes) {
			s.suffixes = slices.Clone(base.Suffixes[i])
		}
		if i < len(base.Paradigms) {
			s.paradigms = slices.Clone(base.Paradigms[i])
		}
		m.shards = append(m.shards, s)
	}
	if len(m.shards) == 0 {
		m.shards = []*mergeShard{{}}
	}
	return m
}

func cloneTagSet(t *TagSet) *TagSet {
	if t == nil {
		return NewTagSet("")
	}
	out := NewTagSet(t.Name)
	out.Tags = slices.Clone(t.Tags)
	for i, name := range out.Tags {
		if _, ok := out.Index[name]; !ok {
			out.Index[name] = uint16(i)
		}
	}
	return out
}

func (s *mergeShard) suffixIndex() map[string]uint16 {
	if s.suffixIdx == nil {
		s.suffixIdx = make(map[string]uint16, len(s.suffixes))
		for i, text := range s.suffixes {
			if _, ok := s.suffixIdx[text]; !ok {
				s.suffixIdx[text] = uint16(i)
			}
		}
	}
	return s.suffixIdx
}

func (s *mergeShard) internSuffix(text string) (uint16, error) {
	idx := s.suffixIndex()
	if id, ok := idx[text]; ok {
		return id, nil
	}
	if len(s.suffixes) >= suffixShardLimit {
		return 0, fmt.Errorf("shard exceeded %d unique suffixes", suffixShardLimit)
	}
	id := uint16(len(s.suffixes))
	s.suffixes = append(s.suffixes, text)
	idx[text] = id
	return id, nil
}

func (s *mergeShard) paradigmIndex() map[string]uint16 {
	if s.paraIdx == nil {
		s.paraIdx = make(map[string]uint16, len(s.paradigms))
		for i, p := range s.paradigms {
			k := string(encodeU16s(p.Data()))
			if _, ok := s.paraIdx[k]; !ok {
				s.paraIdx[k] = uint16(i)
			}
		}
	}
	return s.paraIdx
}

func (m *merger) internPrefix(text string) (uint16, error) {
	if id, ok := m.prefixIdx[text]; ok {
		return id, nil
	}
	if len(m.prefixes) >= 1<<16 {
		return 0, fmt.Errorf("too many prefixes (max %d)", 1<<16)
	}
	id := uint16(len(m.prefixes))
	m.prefixes = append(m.prefixes, text)
	m.prefixIdx[text] = id
	return id, nil
}

// targetShard returns the shard a paradigm with the given suffix texts
// goes to: the last shard while it has room, otherwise a new empty one.
func (m *merger) targetShard(sufTexts []string) int {
	last := len(m.shards) - 1
	t := m.shards[last]
	idx := t.suffixIndex()
	fresh := make(map[string]bool)
	for _, s := range sufTexts {
		if _, ok := idx[s]; !ok {
			fresh[s] = true
		}
	}
	if len(t.suffixes)+len(fresh) <= mergeSuffixLimit && len(t.paradigms) < paradigmLimit {
		return last
	}
	m.shards = append(m.shards, &mergeShard{})
	return last + 1
}

// place remaps the overlay paradigm behind r into the output id spaces
// (form-for-form: suffix text, prefix text and tag name → output ids),
// deduplicating against the target shard's paradigms, and returns where
// it landed. The form index is preserved.
func (m *merger) place(oi int, o *Dictionary, r overlayReading) (paraLoc, error) {
	ref := paraRef{overlay: oi, shard: r.shard, para: r.para}
	if loc, ok := m.cache[ref]; ok {
		return loc, nil
	}
	if r.shard >= len(o.Paradigms) || int(r.para) >= len(o.Paradigms[r.shard]) {
		return paraLoc{}, fmt.Errorf("overlay %d: shard %d: paradigm %d out of range", oi, r.shard, r.para)
	}
	p := o.Paradigms[r.shard][r.para]
	var oSuffixes []string
	if r.shard < len(o.Suffixes) {
		oSuffixes = o.Suffixes[r.shard]
	}

	n := p.Len()
	sufTexts := make([]string, n)
	tagIDs := make([]uint16, n)
	prefIDs := make([]uint16, n)
	for i := 0; i < n; i++ {
		sufTexts[i] = stringAt(oSuffixes, p.Suffix(i))
		tag := ""
		if o.TagSet != nil {
			tag = o.TagSet.TagName(p.Tag(i))
		}
		tid, err := m.tagSet.Add(tag)
		if err != nil {
			return paraLoc{}, err
		}
		tagIDs[i] = tid
		pid, err := m.internPrefix(stringAt(o.Prefixes, p.Prefix(i)))
		if err != nil {
			return paraLoc{}, err
		}
		prefIDs[i] = pid
	}

	shard := m.targetShard(sufTexts)
	t := m.shards[shard]
	sufIDs := make([]uint16, n)
	for i, text := range sufTexts {
		id, err := t.internSuffix(text)
		if err != nil {
			return paraLoc{}, fmt.Errorf("shard %d: %w", shard, err)
		}
		sufIDs[i] = id
	}

	para := NewParadigm(sufIDs, tagIDs, prefIDs)
	key := string(encodeU16s(para.Data()))
	idx := t.paradigmIndex()
	id, ok := idx[key]
	if !ok {
		if len(t.paradigms) >= paradigmLimit {
			return paraLoc{}, fmt.Errorf("shard %d exceeded %d unique paradigms", shard, paradigmLimit)
		}
		id = uint16(len(t.paradigms))
		t.paradigms = append(t.paradigms, para)
		idx[key] = id
	}
	loc := paraLoc{shard: shard, para: id}
	m.cache[ref] = loc
	return loc, nil
}

// stringAt returns ar[i], or "" when i is out of range (mirrors the
// engine's strAt in pkg/morphology/parse.go).
func stringAt(ar []string, i uint16) string {
	if int(i) < len(ar) {
		return ar[i]
	}
	return ""
}
```

- [ ] **Step 4: Run**: `go test ./pkg/morphology/internal/` → PASS

- [ ] **Step 5: Commit**: `git commit -am "feat(internal): id-preserving overlay paradigm remap for merge"`

---

### Task 6: Merge engine, part 3: `MergeDictionaries` (shards, alphabet, prediction, probability)

**Files:**
- Modify: `pkg/morphology/internal/merge.go`, `pkg/morphology/internal/merge_test.go`

**Interfaces:**
- Consumes: Tasks 2–5.
- Produces:
  - `type MergeOptions struct { Mode MergeMode; RebuildPrediction bool; Productive func(tag string) bool }`
  - `var ErrPredictionSharded = errors.New("prediction rebuild needs a single-shard output")`
  - `func MergeDictionaries(base *Dictionary, overlays []*Dictionary, opts MergeOptions) (*Dictionary, error)`: the output has no `Info` (the public layer stamps it)

- [ ] **Step 1: Failing tests** (append to `internal/merge_test.go`; add `"bytes"` if needed)

```go
func allProductive(string) bool { return true }

// twoShardBase glues two single-shard builds into one 2-shard dense
// dictionary. Both builds register tags "N" then "G", so shard 1's tag
// ids are valid under shard 0's TagSet.
func twoShardBase(t *testing.T) *Dictionary {
	t.Helper()
	a, err := BuildDictionaryFromEntries(BuildOptions{}, []BuildEntry{e("кот", "кот", "N"), e("кота", "кот", "G")})
	require.NoError(t, err)
	b, err := BuildDictionaryFromEntries(BuildOptions{}, []BuildEntry{e("мышь", "мышь", "N"), e("мыши", "мышь", "G")})
	require.NoError(t, err)
	d := NewDictionary("ru", a.TagSet,
		[][]string{a.Suffixes[0], b.Suffixes[0]}, a.Prefixes,
		[][]Paradigm{a.Paradigms[0], b.Paradigms[0]},
		[]*DAWG{a.Words[0], b.Words[0]}, RussianCharPolicy())
	require.NoError(t, RecompileDense(d))
	return d
}

// readingsOf returns word's raw values across all shards (exact key).
func readingsOf(d *Dictionary, word string) [][]byte {
	var out [][]byte
	for _, w := range d.Words {
		for _, it := range w.SimilarItems(word, nil, d.Alphabet) {
			if it.Key == word {
				out = append(out, it.Values...)
			}
		}
	}
	return out
}

func TestMergeDictionariesReusesCleanShard(t *testing.T) {
	base := twoShardBase(t)
	over := engineDict(t, e("шок", "шок", "N")) // letters already in the base alphabet
	out, err := MergeDictionaries(base, []*Dictionary{over}, MergeOptions{Mode: MergeAdd})
	require.NoError(t, err)

	require.Len(t, out.Words, 2)
	assert.Same(t, base.Alphabet, out.Alphabet, "alphabet reused when it covers the overlay")
	assert.Equal(t, base.Words[0].Bytes(), out.Words[0].Bytes(), "clean shard 0 is reused byte-for-byte")
	assert.NotSame(t, base.Words[0], out.Words[0], "…but cloned, not aliased")
	assert.NotEmpty(t, readingsOf(out, "шок"), "overlay word lands in the target (last) shard")
	assert.NotEmpty(t, readingsOf(out, "мыши"))
}

func TestMergeDictionariesExtendsAlphabet(t *testing.T) {
	base := engineDict(t, e("кот", "кот", "N"))
	over := engineDict(t, e("wifi", "wifi", "N"))
	out, err := MergeDictionaries(base, []*Dictionary{over}, MergeOptions{Mode: MergeAdd})
	require.NoError(t, err)
	assert.NotSame(t, base.Alphabet, out.Alphabet)
	assert.NotEmpty(t, readingsOf(out, "wifi"))
	assert.NotEmpty(t, readingsOf(out, "кот"))
}

func TestMergeDictionariesReplaceAcrossShards(t *testing.T) {
	base := twoShardBase(t)
	over := engineDict(t, e("мыши", "мыши", "X"))
	out, err := MergeDictionaries(base, []*Dictionary{over}, MergeOptions{Mode: MergeReplace})
	require.NoError(t, err)
	vals := readingsOf(out, "мыши")
	require.Len(t, vals, 1, "all base readings of a replaced word are dropped")
	assert.NotEmpty(t, readingsOf(out, "кот"))
	assert.NotEmpty(t, readingsOf(out, "мышь"))
}

func withProbability(t *testing.T, d *Dictionary, kv map[string]uint32) *Dictionary {
	t.Helper()
	keys, vals := sortedKV(kv)
	p, err := BuildIntDAWG(keys, vals)
	require.NoError(t, err)
	d.Probability = p
	return d
}

func TestMergeDictionariesProbability(t *testing.T) {
	newBase := func() *Dictionary {
		return withProbability(t, engineDict(t, e("кот", "кот", "N"), e("кота", "кот", "G")),
			map[string]uint32{"кот:N": 500, "кота:G": 300})
	}

	// Fast path: nothing removed, overlay without probability → verbatim copy.
	base := newBase()
	out, err := MergeDictionaries(base, []*Dictionary{engineDict(t, e("шок", "шок", "N"))}, MergeOptions{Mode: MergeAdd})
	require.NoError(t, err)
	assert.Equal(t, base.Probability.Bytes(), out.Probability.Bytes())
	assert.NotSame(t, base.Probability, out.Probability)

	// Overlay probability is carried for the words it wins.
	over := withProbability(t, engineDict(t, e("шок", "шок", "N")), map[string]uint32{"шок:N": 900})
	out, err = MergeDictionaries(newBase(), []*Dictionary{over}, MergeOptions{Mode: MergeAdd})
	require.NoError(t, err)
	assert.Equal(t, uint32(500), out.Probability.Find("кот:N"))
	assert.Equal(t, uint32(900), out.Probability.Find("шок:N"))

	// A replaced word loses its base probability entries.
	out, err = MergeDictionaries(newBase(), []*Dictionary{engineDict(t, e("кот", "кот", "V"))}, MergeOptions{Mode: MergeReplace})
	require.NoError(t, err)
	assert.Equal(t, uint32(0), out.Probability.Find("кот:N"))
	assert.Equal(t, uint32(300), out.Probability.Find("кота:G"))
}

func TestMergeDictionariesPrediction(t *testing.T) {
	base := engineDictRaw(t, e("кот", "кот", "N"), e("кота", "кот", "G"))
	require.NoError(t, BuildPrediction(base, allProductive))
	require.NoError(t, RecompileDense(base))
	over := engineDict(t, e("шок", "шок", "N"))

	out, err := MergeDictionaries(base, []*Dictionary{over}, MergeOptions{Mode: MergeAdd})
	require.NoError(t, err)
	require.Len(t, out.Prediction, 1)
	assert.Equal(t, base.Prediction[0].Bytes(), out.Prediction[0].Bytes(), "prediction carried verbatim by default")
	assert.Empty(t, out.Prediction[0].SimilarItems("ок", nil, nil), "overlay words don't feed carried prediction")

	out, err = MergeDictionaries(base, []*Dictionary{over}, MergeOptions{Mode: MergeAdd, RebuildPrediction: true, Productive: allProductive})
	require.NoError(t, err)
	assert.NotEmpty(t, out.Prediction[0].SimilarItems("ок", nil, nil), "rebuilt prediction covers overlay words")

	_, err = MergeDictionaries(twoShardBase(t), []*Dictionary{over}, MergeOptions{Mode: MergeAdd, RebuildPrediction: true, Productive: allProductive})
	assert.ErrorIs(t, err, ErrPredictionSharded)
}

// engineDictRaw is engineDict without the dense recompile.
func engineDictRaw(t *testing.T, entries ...BuildEntry) *Dictionary {
	t.Helper()
	d, err := BuildDictionaryFromEntries(BuildOptions{}, entries)
	require.NoError(t, err)
	return d
}
```

- [ ] **Step 2: Run**: `go test ./pkg/morphology/internal/ -run MergeDictionaries` → FAIL

- [ ] **Step 3: Implement** (append to `internal/merge.go`; imports now `encoding/binary`, `errors`, `fmt`, `slices`, `sort`, `strings`)

```go
// MergeOptions configures MergeDictionaries.
type MergeOptions struct {
	Mode MergeMode
	// RebuildPrediction replaces the base's prediction DAWGs with a single
	// prefix-0 DAWG rebuilt from the merged shard 0. Requires a
	// single-shard output (ErrPredictionSharded otherwise) and Productive.
	RebuildPrediction bool
	// Productive filters prediction tags (the engine's productive()).
	Productive func(tag string) bool
}

// ErrPredictionSharded is returned when RebuildPrediction is requested
// but the merged dictionary has more than one shard: the engine resolves
// prediction against shard 0 only.
var ErrPredictionSharded = errors.New("prediction rebuild needs a single-shard output")

// MergeDictionaries merges overlays into base structurally: base ids
// (tags, prefixes, per-shard suffixes and paradigms) are kept verbatim
// and only grow, overlay readings are remapped into them, and only the
// words DAWGs of changed shards are rebuilt. The base's prediction and
// probability therefore stay valid. The result shares no memory with
// the inputs and has no Info.
func MergeDictionaries(base *Dictionary, overlays []*Dictionary, opts MergeOptions) (*Dictionary, error) {
	if base == nil {
		return nil, errors.New("merge: nil base")
	}
	winners, err := decideOverlays(base, overlays, opts.Mode)
	if err != nil {
		return nil, err
	}
	words := make([]string, 0, len(winners))
	for w := range winners {
		words = append(words, w)
	}
	sort.Strings(words)

	m := newMerger(base)
	for _, w := range words {
		win := winners[w]
		for _, r := range win.readings {
			loc, err := m.place(win.overlay, overlays[win.overlay], r)
			if err != nil {
				return nil, err
			}
			s := m.shards[loc.shard]
			s.placed = append(s.placed, WordValue{Word: w, Value: uint32(loc.para)<<16 | uint32(r.form)})
			s.dirty = true
		}
	}
	if opts.RebuildPrediction {
		if len(m.shards) != 1 {
			return nil, ErrPredictionSharded
		}
		if opts.Productive == nil {
			return nil, errors.New("merge: Productive is required to rebuild prediction")
		}
	}

	removed := make(map[string]bool)
	if opts.Mode == MergeReplace {
		for _, w := range words {
			for i, s := range m.shards {
				if i < len(base.Words) && shardHasWord(base.Words[i], base.Alphabet, w) {
					removed[w] = true
					s.dirty = true
				}
			}
		}
	}

	alphabet, changed, err := mergeAlphabet(base, words)
	if err != nil {
		return nil, err
	}

	out := &Dictionary{
		Language:   base.Language,
		TagSet:     m.tagSet,
		Prefixes:   m.prefixes,
		CharPolicy: base.CharPolicy,
		Alphabet:   alphabet,
	}
	var shard0 []WordValue
	for i, s := range m.shards {
		out.Suffixes = append(out.Suffixes, s.suffixes)
		out.Paradigms = append(out.Paradigms, s.paradigms)
		if !s.dirty && !changed && s.base != nil {
			out.Words = append(out.Words, s.base.Clone())
			continue
		}
		pairs, err := shardPairs(s.base, base.Alphabet, removed)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, s.placed...)
		if i == 0 {
			shard0 = pairs
		}
		w, err := buildWordsDAWG(pairs, alphabet)
		if err != nil {
			return nil, fmt.Errorf("merge: shard %d: %w", i, err)
		}
		out.Words = append(out.Words, w)
	}

	if opts.RebuildPrediction {
		if shard0 == nil { // shard 0 was reused verbatim
			if shard0, err = shardPairs(base.Words[0], base.Alphabet, nil); err != nil {
				return nil, err
			}
		}
		pred, err := BuildPredictionFrom(shard0, out.Paradigms[0], out.TagSet, opts.Productive)
		if err != nil {
			return nil, fmt.Errorf("merge: prediction: %w", err)
		}
		out.Prediction = []*DAWG{pred}
	} else {
		for _, p := range base.Prediction {
			out.Prediction = append(out.Prediction, p.Clone())
		}
	}

	if out.Probability, err = mergeProbability(base, overlays, winners, removed); err != nil {
		return nil, fmt.Errorf("merge: probability: %w", err)
	}
	return out, nil
}

// mergeAlphabet picks the output's dense alphabet: the base's when it
// can encode every overlay word (changed=false), otherwise a new one over
// the base runes (or all base words for a non-dense base) plus the
// overlay words — width 1, falling back to width 2.
func mergeAlphabet(base *Dictionary, newWords []string) (Alphabet, bool, error) {
	if da, ok := base.Alphabet.(*DenseAlphabet); ok {
		fits := true
		for _, w := range newWords {
			if _, err := da.Encode(w); err != nil {
				fits = false
				break
			}
		}
		if fits {
			return da, false, nil
		}
		a, err := denseAlphabetFor(append([]string{string(da.Runes())}, newWords...))
		return a, true, err
	}
	corpus := slices.Clone(newWords)
	for _, w := range base.Words {
		if w == nil {
			continue
		}
		var walkErr error
		w.Walk(func(key string, _ [][]byte) {
			word, err := decodeKey(base.Alphabet, key)
			if err != nil && walkErr == nil {
				walkErr = err
			}
			corpus = append(corpus, word)
		})
		if walkErr != nil {
			return nil, false, walkErr
		}
	}
	a, err := denseAlphabetFor(corpus)
	return a, true, err
}

func denseAlphabetFor(corpus []string) (*DenseAlphabet, error) {
	if a, err := NewDenseAlphabet(1, corpus); err == nil {
		return a, nil
	}
	return NewDenseAlphabet(2, corpus)
}

// shardPairs returns a base shard's readings as plain-text pairs,
// skipping removed words. nil w yields nil.
func shardPairs(w *DAWG, a Alphabet, removed map[string]bool) ([]WordValue, error) {
	if w == nil {
		return nil, nil
	}
	var pairs []WordValue
	var walkErr error
	w.Walk(func(key string, vals [][]byte) {
		if walkErr != nil {
			return
		}
		word, err := decodeKey(a, key)
		if err != nil {
			walkErr = err
			return
		}
		if removed[word] {
			return
		}
		for _, v := range vals {
			if len(v) >= 4 {
				pairs = append(pairs, WordValue{Word: word, Value: binary.BigEndian.Uint32(v[:4])})
			}
		}
	})
	return pairs, walkErr
}

// buildWordsDAWG encodes pairs under a and builds a words DAWG.
func buildWordsDAWG(pairs []WordValue, a Alphabet) (*DAWG, error) {
	keys := make([]string, len(pairs))
	vals := make([]uint32, len(pairs))
	for i, p := range pairs {
		k, ok := encodeKey(a, p.Word)
		if !ok {
			return nil, fmt.Errorf("encode %q", p.Word)
		}
		keys[i], vals[i] = k, p.Value
	}
	return BuildDAWGWithValues(keys, vals)
}

// mergeProbability carries the base's p(tag|word) DAWG (keys
// "word:tag"): verbatim when nothing was removed and no overlay supplies
// probability, otherwise filtered (replaced words dropped) and extended
// with the winning overlays' entries, then rebuilt.
func mergeProbability(base *Dictionary, overlays []*Dictionary, winners map[string]*winner, removed map[string]bool) (*DAWG, error) {
	added := make(map[string]uint32)
	for word, win := range winners {
		o := overlays[win.overlay]
		if o.Probability == nil {
			continue
		}
		for _, r := range win.readings {
			key := word + ":" + readingTag(o, r)
			if v := o.Probability.Find(key); v > 0 {
				added[key] = v
			}
		}
	}
	if len(added) == 0 && len(removed) == 0 {
		return base.Probability.Clone(), nil
	}

	kv := make(map[string]uint32)
	if base.Probability != nil {
		base.Probability.WalkValues(func(key string, v uint32) {
			if i := strings.LastIndexByte(key, ':'); i >= 0 && removed[key[:i]] {
				return
			}
			kv[key] = v
		})
	}
	for k, v := range added {
		kv[k] = v
	}
	if len(kv) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	vals := make([]uint32, len(keys))
	for i, k := range keys {
		vals[i] = kv[k]
	}
	return BuildIntDAWG(keys, vals)
}

func readingTag(o *Dictionary, r overlayReading) string {
	if o.TagSet == nil || r.shard >= len(o.Paradigms) || int(r.para) >= len(o.Paradigms[r.shard]) {
		return ""
	}
	p := o.Paradigms[r.shard][r.para]
	if int(r.form) >= p.Len() {
		return ""
	}
	return o.TagSet.TagName(p.Tag(int(r.form)))
}
```

- [ ] **Step 4: Run**: `go test ./pkg/morphology/internal/` → PASS. Also `go vet ./pkg/morphology/internal/`.

- [ ] **Step 5: Commit**: `git commit -am "feat(internal): structural MergeDictionaries (shard reuse, alphabet, prediction, probability)"`

---

### Task 7: Public API on top of the engine (findings #1–#3, #5)

**Files:**
- Rewrite: `pkg/morphology/merge.go`
- Modify: `pkg/morphology/merge_test.go`, `pkg/morphology/merge_internal_test.go`

**Interfaces:**
- Consumes: `internal.MergeDictionaries`, `internal.MergeOptions`, `internal.ErrPredictionSharded`, `(*internal.DAWG).Empty`, `tagmap.Known`, `productive` (`parse.go`)
- Produces:
  - `func Merge(base *Dictionary, overlays []*Dictionary, mode MergeMode) (*Dictionary, error)` (same signature)
  - `type MergeOptions struct { Mode MergeMode; RebuildPrediction bool }`
  - `func MergeWithOptions(base *Dictionary, overlays []*Dictionary, opts MergeOptions) (*Dictionary, error)`
  - `var ErrIncompatibleDictionaries`, `var ErrPredictionSharded`

- [ ] **Step 1: Update and add the public tests** (`merge_test.go`; add imports `errors`, `path/filepath`, `github.com/amarin/gomorphy/pkg/morphology/tagmap`)

Replace the tail of `TestMergeMultipleOverlays` (the `щенок` block) with fold semantics:

```go
	readings = merged.Parse("щенок")
	require.Len(t, readings, 1, "add is a fold: overlay 2 skips a word overlay 1 already added")
	assert.Equal(t, mergeTagMascNomn, readings[0].Tag)
```

and change the `overlay2` comment on the `щенок` line to `// already added by overlay 1: dropped`.

Add:

```go
func TestMergeReplaceLastOverlayWins(t *testing.T) {
	base := buildFromTriples(t, [3]string{"кот", "кот", mergeTagMascNomn})
	o1 := buildFromTriples(t, [3]string{"кот", "кот", mergeTagVerb}, [3]string{"пёс", "пёс", mergeTagVerb})
	o2 := buildFromTriples(t, [3]string{"кот", "кот", mergeTagFemnNomn}, [3]string{"пёс", "пёс", mergeTagFemnNomn})

	merged, err := morphology.Merge(base, []*morphology.Dictionary{o1, o2}, morphology.MergeReplace)
	require.NoError(t, err)
	for _, w := range []string{"кот", "пёс"} {
		rs := merged.Parse(w)
		require.Len(t, rs, 1, w)
		assert.Equal(t, mergeTagFemnNomn, rs[0].Tag, "%s: the last overlay wins", w)
	}
}

// pymorphyBase is a pymorphy2-shape fixture (TagSet "opencorpora-int")
// with prediction and probability.
func pymorphyBase(t *testing.T) *morphology.Dictionary {
	t.Helper()
	m := map[string]uint32{}
	stdWords(m)
	pred := map[string]uint32{}
	addPrediction(pred, "ёнк", 2, 0, 0)
	addPrediction(pred, "ёнка", 3, 0, 1)
	return buildFixture(t, m, pred, map[string]uint32{
		"кот:NOUN,anim,masc,sing,nomn": 500,
		"кот:VERB,impf,trans":          100,
	})
}

func TestMergeKeepsBaseStructure(t *testing.T) {
	base := pymorphyBase(t)
	overlay := buildFromTriples(t, [3]string{"пёс", "пёс", mergeTagMascNomn})

	merged, err := morphology.Merge(base, []*morphology.Dictionary{overlay}, morphology.MergeAdd)
	require.NoError(t, err)

	assert.Equal(t, base.TagSetName(), merged.TagSetName(), "the base TagSet name is kept")
	assert.Equal(t, base.Parse("кот"), merged.Parse("кот"), "base readings identical: Para, Prob, order")
	assert.Equal(t, base.Parse("котёнка"), merged.Parse("котёнка"), "base prediction still works")

	rs := merged.Parse("пёс")
	require.Len(t, rs, 1)
	assert.Equal(t, "пёс", rs[0].Normal)
	assert.Equal(t, mergeTagMascNomn, rs[0].Tag)

	_, ok := tagmap.Map(merged.TagSetName(), merged.Parse("кот")[0].Tag)
	assert.True(t, ok, "tagmap still recognizes the merged dictionary")
}

func TestMergeReplaceDropsBaseProbability(t *testing.T) {
	overlay := buildFromTriples(t, [3]string{"кот", "кот", mergeTagVerb})
	merged, err := morphology.Merge(pymorphyBase(t), []*morphology.Dictionary{overlay}, morphology.MergeReplace)
	require.NoError(t, err)
	rs := merged.Parse("кот")
	require.Len(t, rs, 1)
	assert.Equal(t, mergeTagVerb, rs[0].Tag)
	assert.Equal(t, 0.0, rs[0].Prob, "a replaced word's base probability is dropped")
}

func TestMergeCarriesOverlayProbability(t *testing.T) {
	base := buildFromTriples(t, [3]string{"дом", "дом", mergeTagMascNomn})
	merged, err := morphology.Merge(base, []*morphology.Dictionary{pymorphyBase(t)}, morphology.MergeAdd)
	require.NoError(t, err)
	rs := merged.Parse("кот")
	require.Len(t, rs, 2)
	assert.Equal(t, mergeTagMascNomn, rs[0].Tag, "overlay probability keeps NOUN first")
	assert.InDelta(t, 0.0005, rs[0].Prob, 1e-9)
}

func TestMergeResultOutlivesInputs(t *testing.T) {
	dir := t.TempDir()
	basePath, overPath := filepath.Join(dir, "base.dat"), filepath.Join(dir, "over.dat")
	require.NoError(t, pymorphyBase(t).SaveTo(basePath))
	require.NoError(t, buildFromTriples(t, [3]string{"пёс", "пёс", mergeTagMascNomn}).SaveTo(overPath))

	base, err := morphology.Open(basePath)
	require.NoError(t, err)
	over, err := morphology.Open(overPath)
	require.NoError(t, err)
	want := base.Parse("котёнка")

	merged, err := morphology.Merge(base, []*morphology.Dictionary{over}, morphology.MergeAdd)
	require.NoError(t, err)
	require.NoError(t, base.Close())
	require.NoError(t, over.Close())

	assert.Equal(t, want, merged.Parse("котёнка"), "prediction survives closing the mmap'd inputs")
	assert.NotEmpty(t, merged.Parse("кот"))
	assert.NotEmpty(t, merged.Parse("пёс"))
}

func TestMergeRejectsLanguageMismatch(t *testing.T) {
	b := morphology.NewBuilder(morphology.BuilderOptions{Language: "en"})
	require.NoError(t, b.AddLemma("cat", "N"))
	en, err := b.Build()
	require.NoError(t, err)
	_, err = morphology.Merge(buildSmallDict(t), []*morphology.Dictionary{en}, morphology.MergeAdd)
	assert.True(t, errors.Is(err, morphology.ErrIncompatibleDictionaries), "got %v", err)
}

func TestMergeWithOptionsRebuildPrediction(t *testing.T) {
	base := buildFromTriples(t, [3]string{"кот", "кот", mergeTagMascNomn}, [3]string{"кота", "кот", mergeTagMascGent})
	overlay := buildFromTriples(t, [3]string{"мышь", "мышь", mergeTagFemnNomn}, [3]string{"мыши", "мышь", mergeTagFemnGent})

	kept, err := morphology.Merge(base, []*morphology.Dictionary{overlay}, morphology.MergeAdd)
	require.NoError(t, err)
	assert.Empty(t, kept.Parse("камыши"), "carried base prediction knows nothing about -ыши")

	rebuilt, err := morphology.MergeWithOptions(base, []*morphology.Dictionary{overlay},
		morphology.MergeOptions{Mode: morphology.MergeAdd, RebuildPrediction: true})
	require.NoError(t, err)
	var tags []string
	for _, r := range rebuilt.Parse("камыши") {
		tags = append(tags, r.Tag)
	}
	assert.Contains(t, tags, mergeTagFemnGent)
}
```

In `merge_internal_test.go`:
- `TestMergeDenseOutputAndPrediction`: change the last assertion to `assert.Equal(t, "builder", merged.TagSetName(), "the base TagSet name is kept")` and the prediction message to `"the base prediction is carried"`.
- `TestMergeInfoInheritance`: build the overlay with `Language: "en"` too (a mismatch is now an error):

```go
	ob := NewBuilder(BuilderOptions{Language: "en"})
	require.NoError(t, ob.AddForm("мышь", "мышь", femnNomn))
	require.NoError(t, ob.AddForm("мыши", "мышь", femnGent))
	overlay, err := ob.Build()
	require.NoError(t, err)
```

  and change its last assertion to `assert.Equal(t, "builder", merged.TagSetName())`.
- Add the tag-vocabulary check:

```go
func TestMergeRejectsMixedKnownTagSets(t *testing.T) {
	base := mergeTestDict(t, [3]string{"кот", "кот", mascNomn})
	overlay := mergeTestDict(t, [3]string{"пёс", "пёс", mascNomn})
	base.d.TagSet.Name = "opencorpora-int"
	overlay.d.TagSet.Name = "unimorph"
	_, err := Merge(base, []*Dictionary{overlay}, MergeAdd)
	assert.ErrorIs(t, err, ErrIncompatibleDictionaries)

	overlay.d.TagSet.Name = "builder" // opaque vocabularies may always be merged in
	_, err = Merge(base, []*Dictionary{overlay}, MergeAdd)
	assert.NoError(t, err)
}
```

- [ ] **Step 2: Run**: `go test ./pkg/morphology/ -run Merge` → FAIL (new symbols; old engine semantics)

- [ ] **Step 3: Rewrite `pkg/morphology/merge.go`**

```go
package morphology

import (
	"errors"
	"fmt"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
	"github.com/amarin/gomorphy/pkg/morphology/tagmap"
)

// MergeMode is the word-level conflict policy Merge applies to every
// overlay. Overlays are applied in order, as a fold:
// Merge(b, [o1, o2], m) equals Merge(Merge(b, [o1], m), [o2], m). A word
// is identified by its exact stored form (no е/ё substitution).
type MergeMode int

const (
	// MergeAdd takes an overlay word only if it is absent from the base
	// and from every earlier overlay; words already present keep their
	// readings untouched.
	MergeAdd MergeMode = iota

	// MergeReplace lets an overlay word's readings fully replace whatever
	// the word had so far — the base's or an earlier overlay's. The last
	// overlay that has the word wins.
	MergeReplace
)

// ErrIncompatibleDictionaries is returned by Merge when an overlay's
// language differs from the base's, or when base and overlay use two
// different tag vocabularies that pkg/morphology/tagmap both knows
// (e.g. "opencorpora-int" and "unimorph") and so cannot share one TagSet.
var ErrIncompatibleDictionaries = errors.New("morphology: merge: incompatible dictionaries")

// ErrPredictionSharded is returned by MergeWithOptions when
// RebuildPrediction is set but the merged dictionary has more than one
// shard.
var ErrPredictionSharded = internal.ErrPredictionSharded

// MergeOptions configures MergeWithOptions.
type MergeOptions struct {
	// Mode is the conflict policy (default MergeAdd).
	Mode MergeMode

	// RebuildPrediction replaces the base's prediction with one rebuilt
	// from every word of the merged dictionary, overlays included. By
	// default the base's prediction is carried over unchanged (overlay
	// words don't feed it) — the right choice for a large base such as
	// pymorphy2. Requires a single-shard result (ErrPredictionSharded).
	RebuildPrediction bool
}

// Merge is MergeWithOptions(base, overlays, MergeOptions{Mode: mode}).
func Merge(base *Dictionary, overlays []*Dictionary, mode MergeMode) (*Dictionary, error) {
	return MergeWithOptions(base, overlays, MergeOptions{Mode: mode})
}

// MergeWithOptions merges overlays into base and returns a new, dense,
// Savable dictionary. The merge is structural: the base's tag set,
// paradigms and suffixes keep their ids, overlay paradigms are remapped
// into them, and only the parts that changed are rebuilt. So the result
// keeps everything the base had:
//
//   - reading order and probabilities (p(tag|word)) of untouched words,
//     plus the probabilities an overlay carries for the words it adds;
//   - prediction for out-of-dictionary words (see RebuildPrediction);
//   - the base's TagSet name, so pkg/morphology/tagmap keeps working.
//     Overlay tags are appended verbatim.
//
// A replaced word loses all of its base readings and base probabilities.
// Inputs are never mutated, and the result shares no memory with them:
// it stays valid after the inputs are closed. The result reports
// BuildInfo{Source: "merge"} with the base's SourceVersion and
// Description, and the base's language and CharPolicy.
//
// Errors: ErrIncompatibleDictionaries (language or tag vocabulary
// mismatch), ErrPredictionSharded, ErrNoEntries (the result has no
// words), and nil-input / unknown-mode errors.
func MergeWithOptions(base *Dictionary, overlays []*Dictionary, opts MergeOptions) (*Dictionary, error) {
	if base == nil || base.d == nil {
		return nil, fmt.Errorf("morphology: merge: base dictionary is nil")
	}
	if opts.Mode != MergeAdd && opts.Mode != MergeReplace {
		return nil, fmt.Errorf("morphology: merge: unknown mode %d", int(opts.Mode))
	}
	ins := make([]*internal.Dictionary, len(overlays))
	for i, o := range overlays {
		if o == nil || o.d == nil {
			return nil, fmt.Errorf("morphology: merge: overlay %d is nil", i)
		}
		if err := checkMergeCompatible(base.d, o.d); err != nil {
			return nil, fmt.Errorf("morphology: merge: overlay %d: %w", i, err)
		}
		ins[i] = o.d
	}

	d, err := internal.MergeDictionaries(base.d, ins, internal.MergeOptions{
		Mode:              internal.MergeMode(opts.Mode),
		RebuildPrediction: opts.RebuildPrediction,
		Productive:        productive,
	})
	if err != nil {
		return nil, fmt.Errorf("morphology: merge: %w", err)
	}
	if dictionaryEmpty(d) {
		return nil, fmt.Errorf("morphology: merge: %w", ErrNoEntries)
	}

	info := &internal.BuildInfo{Source: "merge"}
	if base.d.Info != nil {
		info.SourceVersion = base.d.Info.SourceVersion
		info.Description = base.d.Info.Description
	}
	d.Info = info
	return &Dictionary{d: d}, nil
}

// checkMergeCompatible rejects overlays that can't share the base's
// language or tag vocabulary.
func checkMergeCompatible(base, overlay *internal.Dictionary) error {
	if base.Language != overlay.Language {
		return fmt.Errorf("%w: language %q differs from the base's %q", ErrIncompatibleDictionaries, overlay.Language, base.Language)
	}
	bn, on := tagSetNameOf(base), tagSetNameOf(overlay)
	if bn != on && tagmap.Known(bn) && tagmap.Known(on) {
		return fmt.Errorf("%w: tag set %q cannot be mixed into the base's %q", ErrIncompatibleDictionaries, on, bn)
	}
	return nil
}

func tagSetNameOf(d *internal.Dictionary) string {
	if d.TagSet == nil {
		return ""
	}
	return d.TagSet.Name
}

func dictionaryEmpty(d *internal.Dictionary) bool {
	for _, w := range d.Words {
		if !w.Empty() {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Run**: `go test ./pkg/morphology/...` → PASS (`ExampleMerge`/`ExampleMergeReplace` included). Also `go vet ./...` and `golangci-lint run ./pkg/morphology/...`.

- [ ] **Step 5: Commit**: `git commit -am "feat(morphology): structural Merge keeps prediction, probability and TagSet; fold semantics for overlays"`

---

### Task 8: CLI `--rebuild-prediction`

**Files:**
- Modify: `cmd/gomorphy/merge.go`, `cmd/gomorphy/merge_test.go`

**Interfaces:**
- Consumes: `morphology.MergeWithOptions`, `morphology.MergeOptions`

- [ ] **Step 1: Failing test** (append to `cmd/gomorphy/merge_test.go`)

```go
func TestMergeCommand_RebuildPrediction(t *testing.T) {
	base := buildBuilderDat(t, [3]string{"кот", "кот", mergeCLIBaseTag})
	overlay := buildBuilderDat(t, [3]string{"мыши", "мыши", mergeCLIOverlayTag})
	out := filepath.Join(t.TempDir(), "merged.dat")

	root := newTestRootCmd(newMergeCommand())
	root.SetArgs([]string{"merge", "--mode", "add", "--rebuild-prediction", "-o", out, base, overlay})
	require.NoError(t, root.Execute())

	assert.Contains(t, openTags(t, out, "камыши"), mergeCLIOverlayTag, "rebuilt prediction covers overlay words")
}
```

- [ ] **Step 2: Run**: `go test ./cmd/gomorphy/ -run RebuildPrediction` → FAIL (`unknown flag`)

- [ ] **Step 3: Implement.** In `merge.go`, give `runMerge` a `rebuildPrediction bool` parameter and call:

```go
	merged, err := morphology.MergeWithOptions(dicts[0], dicts[1:], morphology.MergeOptions{
		Mode:              mode,
		RebuildPrediction: rebuildPrediction,
	})
```

In `newMergeCommand`:

```go
	cmd.Flags().Bool("rebuild-prediction", false,
		"rebuild out-of-dictionary prediction from all merged words (default: keep the base's; single-shard output only)")
	…
		rebuild, _ := cmd.Flags().GetBool("rebuild-prediction")
		return runMerge(cmd, args, output, mode, rebuild)
```

Update the other `runMerge` call sites in tests, if any, to pass `false`.

- [ ] **Step 4: Run**: `go test ./cmd/gomorphy/` → PASS

- [ ] **Step 5: Commit**: `git commit -am "feat(cmd/gomorphy): merge --rebuild-prediction"`

---

### Task 9: Real-data gated golden test and measurements (finding #4)

**Files:**
- Create: `pkg/morphology/merge_realdict_test.go`

- [ ] **Step 1: Write the gated test**

```go
package morphology

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMergeRealDictionary merges a small overlay into a real .dat
// (GOMORPHY_MERGE_BASE, e.g. .data/pymorphy/pymorphy.dat) and checks that
// every sampled base word parses identically, OOV prediction is
// unchanged, the TagSet name is kept, and the output grows ≤ 2%.
func TestMergeRealDictionary(t *testing.T) {
	path := os.Getenv("GOMORPHY_MERGE_BASE")
	if path == "" {
		t.Skip("GOMORPHY_MERGE_BASE not set")
	}
	base, err := Open(path)
	require.NoError(t, err)
	defer func() { _ = base.Close() }()

	ob := NewBuilder(BuilderOptions{Language: base.Language()})
	require.NoError(t, ob.AddForm("криптобиржа", "криптобиржа", "NOUN,inan,femn sing,nomn"))
	require.NoError(t, ob.AddForm("криптобиржи", "криптобиржа", "NOUN,inan,femn sing,gent"))
	overlay, err := ob.Build()
	require.NoError(t, err)

	start := time.Now()
	merged, err := Merge(base, []*Dictionary{overlay}, MergeAdd)
	require.NoError(t, err)
	t.Logf("merge took %v", time.Since(start))

	assert.Equal(t, base.TagSetName(), merged.TagSetName())

	var sample []string
	i := 0
	base.d.Words[0].Walk(func(key string, _ [][]byte) {
		if i++; i%250 == 0 && len(sample) < 20000 {
			w, err := base.d.Alphabet.Decode([]byte(key))
			require.NoError(t, err)
			sample = append(sample, w)
		}
	})
	sample = append(sample, "бутявкающий", "глокая", "куздра", "штеко", "будланула", "бокра")
	for _, w := range sample {
		require.Equal(t, base.Parse(w), merged.Parse(w), w)
	}
	assert.NotEmpty(t, merged.Parse("криптобиржи"))

	out := filepath.Join(t.TempDir(), "merged.dat")
	require.NoError(t, merged.SaveTo(out))
	bi, err := os.Stat(path)
	require.NoError(t, err)
	oi, err := os.Stat(out)
	require.NoError(t, err)
	t.Logf("size %d → %d (%.2f%%)", bi.Size(), oi.Size(), 100*float64(oi.Size()-bi.Size())/float64(bi.Size()))
	assert.LessOrEqual(t, float64(oi.Size()), float64(bi.Size())*1.02)
}
```

(If the base is sharded, the sample walks shard 0 only, which is enough for a golden check.)

- [ ] **Step 2: Run on all three real dictionaries**

```bash
for d in pymorphy/pymorphy.dat opencorpora/opencorpora.dat unimorph/ru/unimorph.dat; do GOMORPHY_MERGE_BASE=$PWD/.data/$d go test ./pkg/morphology/ -run TestMergeRealDictionary -v -timeout 30m; done
```

Expected: PASS for all three. Merge time on pymorphy ≈ 25–40 s.

Note: `unimorph.dat` has TagSet `unimorph` and the overlay is `builder`, so it's allowed. The overlay tags are pymorphy-style strings and just become opaque additions.

- [ ] **Step 3: Measure the CLI end-to-end** (record the numbers for the write-up)

```bash
go build -o /tmp/gm ./cmd/gomorphy
printf 'криптобиржа\tкриптобиржа\tNOUN,inan,femn sing,nomn\n' > /tmp/o.tsv
/tmp/gm import tsv /tmp/o.tsv -o /tmp/o.dat
/usr/bin/time -l /tmp/gm merge --mode replace -o /tmp/m.dat .data/pymorphy/pymorphy.dat /tmp/o.dat
/tmp/gm -d /tmp/m.dat lookup бутявкающий
/tmp/gm -d /tmp/m.dat lookup стали
```

Expected: ≤ 40 s real time, ≤ 1.5 GB max RSS, `/tmp/m.dat` ≈ 16 MB. `бутявкающий` gives the same 5 predicted readings as the base, and `стали` lists `стать` first, as the base does.

If the time or RSS budget is missed, profile (`go test -cpuprofile`) before changing anything. The expected hot spot is `BuildDAWGWithValues` over ~5 M keys (the measured floor is ~23 s).

- [ ] **Step 4: Commit**: `git add pkg/morphology/merge_realdict_test.go && git commit -m "test(morphology): gated real-dictionary golden test for structural merge"`

---

### Task 10: EN docs and godoc (finding #6, EN part)

**Files:**
- Modify: `docs/en/cli.md` (the `merge` section), `docs/en/library.md` (the `Merge` section and the Builder intro), `pkg/morphology/dictionary.go` (package doc), `pkg/morphology/example_test.go`

- [ ] **Step 1: `docs/en/cli.md`, `merge` section.** Replace the "(dense 1-byte alphabet, prediction rebuilt)" sentence and the bullets with:

```markdown
Reads the base dictionary plus one or more overlays (compiled `.dat`
files) and writes a merged `.dat` without touching the inputs. The merge
is structural: the base keeps its paradigms, tag set name (so tag
normalization keeps working), probabilities and out-of-dictionary
prediction; overlay words are added into that structure. Merging a small
overlay into `pymorphy.dat` takes ~25–40 s (rebuilding the word index
dominates) and grows the file by the overlay's size only.

- `--mode add` — overlay words that the base (or an earlier overlay)
  already has are skipped; new words are added.
- `--mode replace` — an overlay word's readings replace the word's
  existing ones; with several overlays the last one wins.
- `--rebuild-prediction` — rebuild prediction from all merged words
  (useful when merging thematic dictionaries with each other); by default
  the base's prediction is kept, and overlay words don't feed it.

Inputs must share a language; two different known tag vocabularies
(e.g. `opencorpora-int` and `unimorph`) are rejected.
```

Add a flag-table row: `| --rebuild-prediction | rebuild prediction from all merged words (single-shard output only) |`.

- [ ] **Step 2: `docs/en/library.md`.** In "Building a dictionary from scratch", change "All three produce the same in-memory format as `CompileFrom*Dense`: dense 1-byte alphabet, prediction rebuilt" to say that Builder/ImportTSV rebuild prediction from their own words, and Merge keeps the base's (link the Merge section). Replace the `### Merge` section body with the semantics from Step 1 plus:

```go
merged, err := morphology.Merge(base, overlays, morphology.MergeAdd)

merged, err = morphology.MergeWithOptions(base, overlays, morphology.MergeOptions{
    Mode:              morphology.MergeReplace,
    RebuildPrediction: true,
})
```

and list the errors `ErrIncompatibleDictionaries` and `ErrPredictionSharded` in the sentinel-errors section next to `ErrNoEntries`/`ErrBuilderClosed`.

- [ ] **Step 3: Package doc** (`dictionary.go`, `# Merging dictionaries`): replace the paragraph with:

```go
// # Merging dictionaries
//
// [Merge] adds overlay dictionaries into a base one without rebuilding it:
// the base keeps its tag set, probabilities and prediction. [MergeAdd]
// keeps the base reading for words present in both, [MergeReplace] swaps
// in the overlay's (the last overlay wins); see ExampleMerge and
// [MergeWithOptions]. The CLI equivalent is
// `gomorphy merge --mode add -o merged.dat base.dat overlay.dat`.
```

- [ ] **Step 4: Example.** Add `ExampleMergeWithOptions` to `example_test.go`:

```go
// ExampleMergeWithOptions merges two thematic dictionaries and rebuilds
// prediction so the overlay's endings are predicted too.
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
	for _, r := range merged.Parse("мыши") {
		fmt.Println(r.Normal, r.Tag)
	}
	// Output:
	// мышь NOUN,gent
}
```

- [ ] **Step 5: Run**: `go test ./pkg/morphology/ -run Example` → PASS. Check with `grep -rn "prediction rebuilt" docs/en pkg/morphology/*.go` that no stale merge claim is left (Builder/TSV mentions are fine).

- [ ] **Step 6: Commit**: `git commit -am "docs: describe structural merge (EN docs, godoc, example)"`

---

### Task 11: RU docs, README, CHANGELOG (finding #6, rest)

**Files:**
- Modify: `docs/ru/cli.md`, `docs/ru/library.md`, `README.md`, `CHANGELOG.md`

- [ ] **Step 1: `docs/ru/cli.md`.** Replace the section `### \`merge\` / \`split\` — пока не реализованы` with three sections: `### \`import\` — сборка \`.dat\` из TSV словоформ`, `### \`merge\` — объединение словарей` and `### \`split\` — пока не реализована`. They are faithful Russian translations of the EN `import`, `merge` (as updated in Task 10) and `split` sections: same examples, bullets and flag tables. Keep the command names, flags and code blocks untranslated.

- [ ] **Step 2: `docs/ru/library.md`.** Add a section `## Сборка словаря с нуля` (subsections `Builder`, `ImportTSV`, `Merge`) translating the EN "Building a dictionary from scratch" section as updated in Task 10. Extend the RU sentinel-errors list with `ErrNoEntries`, `ErrBuilderClosed`, `ErrIncompatibleDictionaries`, `ErrPredictionSharded`.

- [ ] **Step 3: `README.md`.** Line ~53: `| CLI: lookup, lemmas, fuzzy, top, download, build, update, import tsv, merge |`. Line ~54: add ", building (Builder, ImportTSV) and merging dictionaries". Line ~74: add `import, merge` to the `cmd/gomorphy` list.

- [ ] **Step 4: `CHANGELOG.md`.** Insert above `## [1.0.0]`:

```markdown
## [Unreleased]

### Added
- `morphology.Builder` (`NewBuilder`/`AddForm`/`AddLemma`/`Build`) and
  `morphology.ImportTSV` — build a dictionary from your own wordforms;
  CLI `gomorphy import tsv`.
- `morphology.Merge` / `MergeWithOptions` — structural merge of compiled
  dictionaries (`MergeAdd`/`MergeReplace`, overlays applied in order)
  keeping the base's tag set, probabilities and prediction; CLI
  `gomorphy merge --mode add|replace [--rebuild-prediction]`.
- `tagmap.Known`.
- Internal: value DAWGs (`BuildIntDAWG`, `WalkValues`), `DAWG.Clone`.
```

- [ ] **Step 5: Commit**: `git commit -am "docs: RU docs, README and CHANGELOG for builder/TSV/merge"`

---

### Task 12: Write-up and roadmap close-out

**Files:**
- Modify: `docs/en/implementation/stage-19-builder-tsv-merge.md`, `docs/en/todo.md`

- [ ] **Step 1: Write-up.** Add a section `## Structural merge (2026-09-2x)` to the Stage 19 write-up. It should cover: why the entries-based merge was replaced (the review table from the spec), the id-preserving design in 5–6 bullets, the measured numbers from Task 9 (time, RSS, size for pymorphy/opencorpora/unimorph), and a link to the spec and this plan.

- [ ] **Step 2: `todo.md`.** Mark the "Stage 19.1 — Structural merge" items `[x]`, set its heading status to `DONE <date>`, and move its row into the "Completed stages" table. In the Stage 19 section, reword the `Merge` bullet to point at the structural merge.

- [ ] **Step 3: Final verification**

```bash
go build ./... && go vet ./... && go test ./... && golangci-lint run ./... && gofmt -l pkg cmd
```

Expected: all green, and `gofmt -l` prints nothing new. `pkg/morphology/compile_test.go` and `fuzzy.go` are pre-existing, so ignore them.

- [ ] **Step 4: Commit**: `git commit -am "docs: close out Stage 19.1 structural merge"`
