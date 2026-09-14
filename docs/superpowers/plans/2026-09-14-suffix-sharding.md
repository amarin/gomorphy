# OpenCorpora Suffix-Overflow Sharding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `gomorphy_build compile` succeed on the real OpenCorpora `dict.xml` (65835 unique suffixes, 299 over uint16's 65536 limit) by splitting lemmas into independent-id-space shards instead of widening any id type.

**Architecture:** `opencorpora.ImportFromXML` gains a `FillOnDemand` sharding strategy that closes the current shard and starts a new one whenever the next lemma would push its unique-suffix count past 65536. `internal.Dictionary`'s `Suffixes`/`Paradigms`/`Words` fields become per-shard slices (`TagSet`/`Prefixes` stay shared across shards); `.dat` files gain numbered `suffixes-N`/`paradigms-N`/`words.dawg-N` sections mirroring the codebase's existing `prediction-N` convention; lookup fans out across shards in parallel goroutines and merges results. `Reading` gains an additive `Shard int` field so `Para`/`Form` stay disambiguable.

**Tech Stack:** Go 1.27, existing GMOR container format (`pkg/morphology/internal/format.go`), no new dependencies.

**Spec:** `docs/superpowers/specs/2026-09-14-suffix-sharding-design.md`

## Global Constraints

- No `.dat` file format compatibility to preserve — project is pre-1.0.0, nothing shipped yet (confirmed with the user).
- Suffix/tag/paradigm ids stay `uint16` — sharding, not widening, is the fix (spec decision).
- Paradigm id is **not** touched — it's packed into the DAWG payload (`uint32(paraID)<<16 | uint32(formIdx)`) and exposed as public `Reading.Para uint16`; out of scope for this work.
- `TagSet` and `Prefixes` are shared/global across all shards — only `Suffixes`, `Paradigms`, `Words` become per-shard.
- pymorphy2 importer is out of scope (its own `paradigms.array` source format is itself uint16-addressed) — it always produces exactly 1 shard.
- v1 ships exactly one `ShardingStrategy` implementation, `FillOnDemand`. The interface must allow adding more later without touching the build pipeline, container format, or lookup path.
- `go build ./...`, `go vet ./...`, and `go test ./... -race` must be green at the end of every task from Task 2 onward except where a task's own description says otherwise (Tasks 2–4 intentionally leave the module non-compiling as a whole until Task 5 lands — see the per-task notes).

---

## Task 1: `ShardingStrategy` interface and `FillOnDemand`

**Files:**
- Create: `pkg/morphology/importers/opencorpora/shard.go`
- Create: `pkg/morphology/importers/opencorpora/shard_test.go`

**Interfaces:**
- Produces: `type ShardingStrategy interface { Boundary(currentCount, newSuffixes, limit int) bool }`, `type FillOnDemand struct{}` implementing it.

This task is fully standalone — it doesn't touch `Dictionary` or `ImportFromXML` yet, so it compiles and tests in isolation.

- [ ] **Step 1: Write the failing test**

```go
// pkg/morphology/importers/opencorpora/shard_test.go
package opencorpora

import "testing"

func TestFillOnDemandBoundary(t *testing.T) {
	var s FillOnDemand
	cases := []struct {
		name                              string
		currentCount, newSuffixes, limit int
		want                              bool
	}{
		{"empty shard never closes", 0, 100, 10, false},
		{"fits exactly at limit", 5, 5, 10, false},
		{"one over limit closes", 5, 6, 10, true},
		{"already at limit, any addition closes", 10, 1, 10, true},
		{"zero-addition lemma never closes", 10, 0, 10, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := s.Boundary(tc.currentCount, tc.newSuffixes, tc.limit)
			if got != tc.want {
				t.Errorf("Boundary(%d, %d, %d) = %v, want %v",
					tc.currentCount, tc.newSuffixes, tc.limit, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/morphology/importers/opencorpora/... -run TestFillOnDemandBoundary -v`
Expected: FAIL — `undefined: FillOnDemand` (compile error, since `shard.go` doesn't exist yet).

- [ ] **Step 3: Write the implementation**

```go
// pkg/morphology/importers/opencorpora/shard.go
package opencorpora

// ShardingStrategy decides, while lemmas are processed in corpus order,
// when the current shard should close and a new one start. Each shard's
// suffix strings get their own independent id space (starts at 0,
// addressed as uint16), so no shard may hold more than limit unique
// suffixes.
//
// v1 ships exactly one implementation, FillOnDemand — see
// docs/superpowers/specs/2026-09-14-suffix-sharding-design.md for why
// (near-zero overhead at today's real N=2 shard count; a second
// strategy, e.g. paradigm-grouped, can be added later as another type
// satisfying this interface without touching ImportFromXML's structure).
type ShardingStrategy interface {
	// Boundary reports whether the CURRENT shard should close before a
	// lemma that would add newSuffixes new, not-yet-seen suffix strings
	// to it is placed — given the shard already holds currentCount
	// unique suffixes and no shard may exceed limit.
	Boundary(currentCount, newSuffixes, limit int) bool
}

// FillOnDemand packs lemmas into the current shard until adding the next
// lemma's new suffixes would exceed limit, then starts a new shard. A
// shard is never closed while still empty, so a single lemma needing
// more than limit suffixes on its own still gets a shard to itself
// rather than looping forever.
type FillOnDemand struct{}

func (FillOnDemand) Boundary(currentCount, newSuffixes, limit int) bool {
	return currentCount > 0 && currentCount+newSuffixes > limit
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/morphology/importers/opencorpora/... -run TestFillOnDemandBoundary -v`
Expected: PASS (5 subtests).

- [ ] **Step 5: Full verification and commit**

Run: `gofmt -l pkg/morphology/importers/opencorpora/shard.go pkg/morphology/importers/opencorpora/shard_test.go` — expect no output.
Run: `go build ./... && go vet ./... && go test ./... -race -count=1` — expect green (this task doesn't touch anything else, so the whole module still builds).

```bash
git add pkg/morphology/importers/opencorpora/shard.go pkg/morphology/importers/opencorpora/shard_test.go
git commit -m "opencorpora: add ShardingStrategy interface and FillOnDemand

Standalone building block for suffix-overflow sharding — see
docs/superpowers/specs/2026-09-14-suffix-sharding-design.md. Not yet
wired into ImportFromXML.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

## Task 2: `internal.Dictionary` becomes per-shard

**Files:**
- Modify: `pkg/morphology/internal/dictionary.go`
- Modify: `pkg/morphology/internal/dictionary_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `Dictionary.Suffixes [][]string`, `Dictionary.Paradigms [][]Paradigm`, `Dictionary.Words []*DAWG` (was `[]string`, `[]Paradigm`, `*DAWG`). `NewDictionary(language string, tagSet *TagSet, suffixes [][]string, prefixes []string, paradigms [][]Paradigm, words []*DAWG, charPolicy *CharPolicy) *Dictionary`. Index 0 is the only shard for an unsharded dictionary — no separate "unsharded" representation exists.

**Note:** after this task, `pkg/morphology/internal` itself builds and its own tests pass, but `pkg/morphology` and its `importers/*` subpackages (which all call the old `NewDictionary` signature or read the old field types) will **not** compile until Tasks 3–5 land. This is expected — verify only the internal package in this task's steps.

- [ ] **Step 1: Update the test to the new shape (write the failing test)**

Replace `pkg/morphology/internal/dictionary_test.go` in full:

```go
package internal

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology/internal/testdawg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDictionaryConstruction(t *testing.T) {
	ts := NewTagSet("opencorpora")
	_, err := ts.Add("NOUN")
	require.NoError(t, err)
	_, err = ts.Add("anim")
	require.NoError(t, err)

	paradigm := NewParadigm([]uint16{5, 6}, []uint16{0, 1}, []uint16{0, 0})
	words, _ := testdawg.Build(map[string]uint32{"кот": 0})

	dict := NewDictionary(
		"ru",
		ts,
		[][]string{{"кот", "коты"}},
		[]string{"а"},
		[][]Paradigm{{paradigm}},
		[]*DAWG{NewDAWG(words, nil)},
		RussianCharPolicy(),
	)

	require.NotNil(t, dict)
	assert.Equal(t, "ru", dict.Language)
	assert.Same(t, ts, dict.TagSet)
	require.Len(t, dict.Suffixes, 1)
	assert.Equal(t, []string{"кот", "коты"}, dict.Suffixes[0])
	assert.Equal(t, []string{"а"}, dict.Prefixes)
	require.Len(t, dict.Paradigms, 1)
	require.Len(t, dict.Paradigms[0], 1)
	assert.Equal(t, uint16(5), dict.Paradigms[0][0].Suffix(0))
	require.Len(t, dict.Words, 1)
	assert.NotNil(t, dict.Words[0])
	assert.NotNil(t, dict.CharPolicy)
	assert.Empty(t, dict.Prediction)
}

func TestDictionaryEmptyComponents(t *testing.T) {
	dict := NewDictionary("en", nil, nil, nil, nil, nil, nil)
	require.NotNil(t, dict)
	assert.Empty(t, dict.Suffixes)
	assert.Len(t, dict.Paradigms, 0)
}

func TestNewDictionaryPanicsOnShardCountMismatch(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on mismatched shard slice lengths")
		}
	}()
	NewDictionary("ru", nil, [][]string{{"a"}}, nil, nil, nil, nil)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/morphology/internal/... -run 'TestDictionaryConstruction|TestDictionaryEmptyComponents|TestNewDictionaryPanicsOnShardCountMismatch' -v`
Expected: FAIL to compile — `NewDictionary` still has the old signature.

- [ ] **Step 3: Update `dictionary.go`**

Replace `pkg/morphology/internal/dictionary.go` in full:

```go
package internal

// Dictionary — иммутабельный снимок словаря.
//
// Suffixes, Paradigms и Words — по одному элементу на шард; индекс 0 —
// единственный шард для несегментированных словарей (нет отдельного
// "нешардированного" представления). Шарды существуют потому, что
// suffix id адресуется uint16: словарь с суффиксов больше, чем помещается
// в один uint16-диапазон, делится на несколько шардов с независимыми
// id-пространствами (см. docs/superpowers/specs/2026-09-14-suffix-sharding-design.md).
// TagSet и Prefixes остаются общими для всех шардов.
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
	Info        *BuildInfo
}

// NewDictionary собирает Dictionary из компонентов. suffixes, paradigms и
// words обязаны иметь одинаковую длину (число шардов) — паникует иначе,
// как и NewParadigm паникует на несовпадении длин своих частей. Prediction,
// Probability и Info остаются nil; Prediction/Probability заполняются
// импортёрами при чтении prediction-файлов, Info — импортёром (обычно
// только Source) или SaveTo (BuiltAt/LibraryVersion — при каждом
// сохранении).
func NewDictionary(
	language string,
	tagSet *TagSet,
	suffixes [][]string,
	prefixes []string,
	paradigms [][]Paradigm,
	words []*DAWG,
	charPolicy *CharPolicy,
) *Dictionary {
	if len(suffixes) != len(paradigms) || len(paradigms) != len(words) {
		panic("internal: NewDictionary shard slices (suffixes, paradigms, words) must have equal length")
	}
	return &Dictionary{
		Language:   language,
		TagSet:     tagSet,
		Suffixes:   suffixes,
		Prefixes:   prefixes,
		Paradigms:  paradigms,
		Words:      words,
		CharPolicy: charPolicy,
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/morphology/internal/... -run 'TestDictionaryConstruction|TestDictionaryEmptyComponents|TestNewDictionaryPanicsOnShardCountMismatch' -v`
Expected: PASS (3 tests). Note `TestDictionaryEmptyComponents` passes `nil` for all three shard slices — `len(nil) == 0` for all three, so the panic guard doesn't trip.

- [ ] **Step 5: Run the whole internal package's tests**

Run: `go test ./pkg/morphology/internal/... -race -count=1`
Expected: PASS. (Other files in this package — `paradigm.go`, `format.go`, `dawgbuild.go`, etc. — don't reference `Dictionary.Suffixes`/`Paradigms`/`Words`, so they're unaffected.)

- [ ] **Step 6: Commit**

```bash
git add pkg/morphology/internal/dictionary.go pkg/morphology/internal/dictionary_test.go
git commit -m "internal: make Dictionary.Suffixes/Paradigms/Words per-shard

Index 0 is the only shard for unsharded dictionaries. TagSet and
Prefixes stay shared. Callers in pkg/morphology and pkg/morphology/
importers/* are updated in the following commits — the module doesn't
build as a whole until then.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

## Task 3: pymorphy2 importer wraps into a single shard

**Files:**
- Modify: `pkg/morphology/importers/pymorphy2/import.go:26-69` (the `ImportFromDir` function)
- Modify: `pkg/morphology/importers/pymorphy2/import_test.go:82-97` (`TestImportFromDir`'s assertions)
- Modify: `pkg/morphology/importers/pymorphy2/full_dict_integration_test.go:27-49` (`TestFullDictStructure`, `TestFullDictVseReadings`)

**Interfaces:**
- Consumes: `internal.NewDictionary` (Task 2's new signature), `internal.Dictionary.Suffixes [][]string` / `.Paradigms [][]Paradigm` / `.Words []*DAWG`.

**Note:** after this task, `pkg/morphology/importers/pymorphy2` builds and its own (non-`integration`-tagged) tests pass. `pkg/morphology` itself still won't compile until Task 4 lands (it also calls the old-shape fields via `save.go`/`open.go`/`parse.go`/`fuzzy.go`).

- [ ] **Step 1: Update `import.go`'s field assignments**

In `pkg/morphology/importers/pymorphy2/import.go`, find these three lines (around lines 57, 63, 69):

```go
	d.Suffixes = suffixes
```
```go
	d.Paradigms = paradigms
```
```go
	d.Words = words
```

Replace each line exactly as follows (three separate edits):

```go
	d.Suffixes = [][]string{suffixes}
```

```go
	d.Paradigms = [][]internal.Paradigm{paradigms}
```

```go
	d.Words = []*internal.DAWG{words}
```

- [ ] **Step 2: Verify the package builds**

Run: `go build ./pkg/morphology/importers/pymorphy2/...`
Expected: success.

- [ ] **Step 3: Update `import_test.go`'s assertions**

In `pkg/morphology/importers/pymorphy2/import_test.go`, replace this block (around lines 82–97):

```go
	assert.Equal(t, []string{"", "кот", "кота", "x"}, d.Suffixes)
	assert.Equal(t, []string{"", "по", "наи"}, d.Prefixes)

	require.Len(t, d.Paradigms, 1)
	p := d.Paradigms[0]
	assert.Equal(t, 2, p.Len())
	assert.Equal(t, uint16(10), p.Suffix(0))
	assert.Equal(t, uint16(1), p.Tag(1))
	assert.Equal(t, uint16(0), p.Prefix(1))

	require.NotNil(t, d.Words)
	items := d.Words.SimilarItems("кот", d.CharPolicy)
	require.Len(t, items, 1)
	assert.Equal(t, "кот", items[0].Key)
	require.Len(t, items[0].Values, 1)
	assert.Equal(t, []byte{0, 0, 0, 0}, items[0].Values[0])
```

with:

```go
	require.Len(t, d.Suffixes, 1)
	assert.Equal(t, []string{"", "кот", "кота", "x"}, d.Suffixes[0])
	assert.Equal(t, []string{"", "по", "наи"}, d.Prefixes)

	require.Len(t, d.Paradigms, 1)
	require.Len(t, d.Paradigms[0], 1)
	p := d.Paradigms[0][0]
	assert.Equal(t, 2, p.Len())
	assert.Equal(t, uint16(10), p.Suffix(0))
	assert.Equal(t, uint16(1), p.Tag(1))
	assert.Equal(t, uint16(0), p.Prefix(1))

	require.Len(t, d.Words, 1)
	require.NotNil(t, d.Words[0])
	items := d.Words[0].SimilarItems("кот", d.CharPolicy)
	require.Len(t, items, 1)
	assert.Equal(t, "кот", items[0].Key)
	require.Len(t, items[0].Values, 1)
	assert.Equal(t, []byte{0, 0, 0, 0}, items[0].Values[0])
```

- [ ] **Step 4: Update `full_dict_integration_test.go`**

Replace the full contents of `TestFullDictStructure` and `TestFullDictVseReadings` (lines 27–50):

```go
func TestFullDictStructure(t *testing.T) {
	d := importFullDict(t)

	require.Len(t, d.Paradigms, 1, "pymorphy2 import is never sharded")
	assert.Greater(t, len(d.Paradigms[0]), 1000, "полный словарь: ~3000 парадигм")
	require.Len(t, d.Suffixes, 1)
	assert.Greater(t, len(d.Suffixes[0]), 1000, "полный словарь: ~5K суффиксов")
	assert.Greater(t, len(d.TagSet.Tags), 500, "полный словарь: ~1K тегов")
	assert.Equal(t, []string{"", "по", "наи"}, d.Prefixes)

	require.Len(t, d.Words, 1)
	require.NotNil(t, d.Words[0])
	require.Len(t, d.Prediction, 3, "prediction-suffixes по числу префиксов")
	require.NotNil(t, d.Probability)
}

func TestFullDictVseReadings(t *testing.T) {
	d := importFullDict(t)

	items := d.Words[0].SimilarItems("все", d.CharPolicy)
	total := 0
	for _, it := range items {
		total += len(it.Values)
	}
	t.Logf("все: items=%d readings=%d", len(items), total)
	assert.GreaterOrEqual(t, total, 4, "Parse(\"все\") даёт ≥4 разбора")
}
```

- [ ] **Step 5: Run pymorphy2's non-integration tests**

Run: `go test ./pkg/morphology/importers/pymorphy2/... -race -count=1 -v`
Expected: PASS (`full_dict_integration_test.go` is behind `//go:build integration` and is skipped by this command — that's expected, it needs `GOMORPHY_PYMORPHY2_DIR` set and isn't part of the normal suite).

- [ ] **Step 6: Verify the integration-tagged file still compiles**

Run: `go vet -tags integration ./pkg/morphology/importers/pymorphy2/...`
Expected: success (type-checks `full_dict_integration_test.go` without requiring `GOMORPHY_PYMORPHY2_DIR` to be set, since `vet` doesn't execute tests).

- [ ] **Step 7: Commit**

```bash
git add pkg/morphology/importers/pymorphy2/import.go pkg/morphology/importers/pymorphy2/import_test.go pkg/morphology/importers/pymorphy2/full_dict_integration_test.go
git commit -m "pymorphy2: wrap importer output into a single shard

pymorphy2's own paradigms.array format is itself uint16-addressed at
the source, so this importer never needs more than one shard — see
docs/superpowers/specs/2026-09-14-suffix-sharding-design.md.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

## Task 4: `pkg/morphology` package becomes shard-aware (save, open, lookup)

**Files:**
- Modify: `pkg/morphology/save.go`
- Modify: `pkg/morphology/open.go:80-167` (`parseContainer`)
- Modify: `pkg/morphology/parse.go`
- Modify: `pkg/morphology/fuzzy.go`

**Interfaces:**
- Consumes: `internal.Dictionary.Suffixes [][]string` / `.Paradigms [][]Paradigm` / `.Words []*DAWG` (Task 2).
- Produces: `Reading.Shard int` (new field). `.dat` sections `suffixes-N`, `paradigms-N`, `words.dawg-N` (was `suffixes`, `paradigms`, `words.dawg`, unnumbered) — mirrors the existing `prediction-N` convention already used for `Dictionary.Prediction`.

**Note:** these four files are all in the same Go package (`pkg/morphology`), so they must be edited together for the package to compile — Go compiles per-package, not per-file. This task's steps therefore group naturally; there is one verification point at the end rather than one per file. After this task, `pkg/morphology` compiles and its own test suite (`pkg/morphology/*_test.go`, package `morphology_test`) passes — but `pkg/morphology/importers/opencorpora` still calls the old-shape `NewDictionary` until Task 5, so `go build ./...` for the whole module is still red until then.

- [ ] **Step 1: Update `save.go`**

Replace `pkg/morphology/save.go` in full:

```go
package morphology

import (
	"fmt"
	"time"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// SaveTo записывает словарь в файл GMOR — единый дисковый формат.
// Секции: meta, info, tagset, prefixes, suffixes-N, paradigms-N,
// words.dawg-N (по одному набору на шард, N от 0), prediction-N,
// probability (если есть).
func (x *Dictionary) SaveTo(path string) error {
	if x == nil || x.d == nil {
		return fmt.Errorf("morphology: nil dictionary")
	}

	// info — копия x.d.Info (если импортёр её заполнил, например Source),
	// с BuiltAt/LibraryVersion, проставленными заново при каждом
	// сохранении; сам x.d не мутируется (Dictionary иммутабелен).
	info := internal.BuildInfo{}
	if x.d.Info != nil {
		info = *x.d.Info
	}
	info.BuiltAt = time.Now().UTC()
	info.LibraryVersion = Version

	// Сжатие пока не реализовано (см. docs/todo.md, "Этап 17") — все секции
	// пишутся как есть. words.dawg-N всегда останется CompressionNone: она
	// алиасится из mmap без копирования, а сжатая секция требует полной
	// декомпрессии в память при загрузке.
	const noCompression = internal.CompressionNone
	sections := []internal.Section{
		{Name: "meta", Data: internal.EncodeMeta(x.d.Language, x.d.CharPolicy), Flags: noCompression},
		{Name: "info", Data: internal.EncodeBuildInfo(&info), Flags: noCompression},
		{Name: "tagset", Data: internal.EncodeTagSet(x.d.TagSet), Flags: noCompression},
		{Name: "prefixes", Data: internal.EncodeStrings(x.d.Prefixes), Flags: noCompression},
	}
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
	return internal.SaveContainer(path, sections)
}
```

- [ ] **Step 2: Update `open.go`'s `parseContainer`**

In `pkg/morphology/open.go`, replace the `parseContainer` function (lines 80–167) in full:

```go
// parseContainer собирает внутренний словарь из секций GMOR-файла.
func parseContainer(cont *internal.Container) (*internal.Dictionary, error) {
	meta, _, err := cont.Section("meta")
	if err != nil {
		return nil, err
	}
	language, policy, err := internal.DecodeMeta(meta)
	if err != nil {
		return nil, err
	}

	data, _, err := cont.Section("tagset")
	if err != nil {
		return nil, err
	}
	tagSet, err := internal.DecodeTagSet(data)
	if err != nil {
		return nil, err
	}

	data, _, err = cont.Section("prefixes")
	if err != nil {
		return nil, err
	}
	prefixes, err := internal.DecodeStrings(data)
	if err != nil {
		return nil, err
	}

	var suffixes [][]string
	for i := 0; ; i++ {
		data, _, err := cont.Section(fmt.Sprintf("suffixes-%d", i))
		if err != nil {
			break
		}
		s, err := internal.DecodeStrings(data)
		if err != nil {
			return nil, err
		}
		suffixes = append(suffixes, s)
	}
	if len(suffixes) == 0 {
		return nil, fmt.Errorf("morphology: no suffixes-N sections found")
	}

	var paradigms [][]internal.Paradigm
	for i := 0; ; i++ {
		data, _, err := cont.Section(fmt.Sprintf("paradigms-%d", i))
		if err != nil {
			break
		}
		p, err := internal.DecodeParadigms(data)
		if err != nil {
			return nil, err
		}
		paradigms = append(paradigms, p)
	}
	if len(paradigms) != len(suffixes) {
		return nil, fmt.Errorf("morphology: %d suffixes-N sections but %d paradigms-N sections", len(suffixes), len(paradigms))
	}

	var words []*internal.DAWG
	for i := 0; ; i++ {
		data, _, err := cont.Section(fmt.Sprintf("words.dawg-%d", i))
		if err != nil {
			break
		}
		w, err := internal.ParseDAWG(data)
		if err != nil {
			return nil, err
		}
		words = append(words, w)
	}
	if len(words) != len(suffixes) {
		return nil, fmt.Errorf("morphology: %d suffixes-N sections but %d words.dawg-N sections", len(suffixes), len(words))
	}

	d := internal.NewDictionary(language, tagSet, suffixes, prefixes, paradigms, words, policy)

	for i := 0; ; i++ {
		predData, _, err := cont.Section(fmt.Sprintf("prediction-%d", i))
		if err != nil {
			break
		}
		pred, err := internal.ParseDAWG(predData)
		if err != nil {
			return nil, err
		}
		d.Prediction = append(d.Prediction, pred)
	}

	if probData, _, err := cont.Section("probability"); err == nil {
		prob, err := internal.ParseDAWG(probData)
		if err != nil {
			return nil, err
		}
		d.Probability = prob
	}

	if infoData, _, err := cont.Section("info"); err == nil {
		info, err := internal.DecodeBuildInfo(infoData)
		if err != nil {
			return nil, err
		}
		d.Info = info
	}

	return d, nil
}
```

- [ ] **Step 3: Update `parse.go`**

Replace `pkg/morphology/parse.go` in full:

```go
package morphology

import (
	"encoding/binary"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// Reading — один разбор словоформы.
type Reading struct {
	Word   string  // словоформа как в словаре (с «ё»)
	Normal string  // начальная форма (лемма)
	Tag    string  // граммемный тег, например "NOUN,anim,masc,sing,nomn"
	Para   uint16  // id парадигмы — уникален только вместе с Shard
	Form   uint16  // индекс формы в парадигме
	Shard  int     // индекс шарда словаря; всегда 0 для нешардированных словарей
	Prob   float64 // вероятность разбора (0, если probability недоступен)
}

// Parse разбирает слово и возвращает все чтения словаря, отсортированные
// по вероятности (убыванию). Для слов вне словаря пытается предсказать
// чтения по prediction-DAWG (окончания). Возвращает nil, если разборы
// не найдены. Вход приводится к нижнему регистру.
func (x *Dictionary) Parse(word string) []Reading {
	if x == nil || x.d == nil || len(x.d.Words) == 0 {
		return nil
	}
	word = strings.ToLower(word)

	if readings := x.exact(word); len(readings) > 0 {
		return readings
	}
	return x.predict(word)
}

// shardExactResult — результат exactInShard для одного шарда.
type shardExactResult struct {
	readings []Reading
	hasProb  bool
}

// exact собирает чтения слова, найденного в словаре, с учётом подмен
// CharPolicy (е→ё) и сортирует по вероятности. Шарды опрашиваются
// параллельно (по горутине на шард), результаты склеиваются.
func (x *Dictionary) exact(word string) []Reading {
	results := make([]shardExactResult, len(x.d.Words))

	var wg sync.WaitGroup
	for shard, dawg := range x.d.Words {
		wg.Add(1)
		go func(shard int, dawg *internal.DAWG) {
			defer wg.Done()
			results[shard] = x.exactInShard(shard, dawg, word)
		}(shard, dawg)
	}
	wg.Wait()

	var readings []Reading
	hasProb := false
	for _, r := range results {
		readings = append(readings, r.readings...)
		hasProb = hasProb || r.hasProb
	}

	if hasProb {
		sort.SliceStable(readings, func(i, j int) bool {
			return readings[i].Prob > readings[j].Prob
		})
	}
	return readings
}

// exactInShard собирает чтения слова из одного шарда. Вызывается
// параллельно с другими шардами из exact — только чтение, общего
// изменяемого состояния между горутинами нет.
func (x *Dictionary) exactInShard(shard int, dawg *internal.DAWG, word string) shardExactResult {
	items := dawg.SimilarItems(word, x.d.CharPolicy)
	if len(items) == 0 {
		return shardExactResult{}
	}

	var res shardExactResult
	res.readings = make([]Reading, 0, len(items))
	for _, it := range items {
		for _, v := range it.Values {
			r, ok := x.reading(shard, it.Key, v)
			if !ok {
				continue
			}
			if x.d.Probability != nil {
				r.Prob = float64(x.d.Probability.Find(it.Key+":"+r.Tag)) / 1e6
				if r.Prob > 0 {
					res.hasProb = true
				}
			}
			res.readings = append(res.readings, r)
		}
	}
	return res
}

// predict ищет чтения для несловарного слова по окончаниям в prediction-DAWG
// (алгоритм KnownSuffixAnalyzer из pymorphy2, как в opennota/morph).
func (x *Dictionary) predict(word string) []Reading {
	if len(x.d.Prediction) == 0 {
		return nil
	}
	splits, ok := suffixSplits(word, 5)
	if !ok {
		return nil
	}

	var readings []Reading
	seen := make(map[string]bool)

	for id, pref := range x.d.Prefixes {
		if id >= len(x.d.Prediction) || x.d.Prediction[id] == nil {
			continue
		}
		if !strings.HasPrefix(word, pref) {
			continue
		}
		readings = append(readings, x.predictForPrefix(id, splits, seen)...)
	}
	return readings
}

// predictForPrefix predicts readings against a single prefix's
// prediction-DAWG (x.d.Prediction[id]), widening from the longest suffix
// split (splits[len(splits)-1]) toward shorter ones until at least 2 total
// matches accumulate — pymorphy2's KnownSuffixAnalyzer heuristic: trust a
// long, specific suffix match over a short, common one when it exists.
// seen dedups (word, lemma, tag) triples across all prefixes tried by the
// caller and is mutated in place.
//
// Predictions always resolve against shard 0: the prediction-DAWG feature
// currently exists only for pymorphy2 imports, which are never sharded
// (see docs/superpowers/specs/2026-09-14-suffix-sharding-design.md).
func (x *Dictionary) predictForPrefix(id int, splits [][2]string, seen map[string]bool) []Reading {
	const predictionShard = 0

	var readings []Reading
	totalCount := 0

	for i := len(splits) - 1; i >= 0; i-- {
		wordStart, wordEnd := splits[i][0], splits[i][1]
		for _, it := range x.d.Prediction[id].SimilarItems(wordEnd, x.d.CharPolicy) {
			for _, v := range it.Values {
				if len(v) < 6 {
					continue
				}
				count := int(binary.BigEndian.Uint16(v[:2]))
				paraNum := binary.BigEndian.Uint16(v[2:4])
				form := binary.BigEndian.Uint16(v[4:6])

				para, ok := x.paradigm(predictionShard, paraNum)
				if !ok || form >= uint16(para.Len()) {
					continue
				}
				if !productive(x.paradigmTag(para, int(form))) {
					continue
				}
				totalCount += count

				r := x.readingForm(predictionShard, wordStart+it.Key, paraNum, form)
				key := r.Word + "\x00" + r.Normal + "\x00" + r.Tag
				if seen[key] {
					continue
				}
				seen[key] = true
				readings = append(readings, r)
			}
		}
		if totalCount > 1 {
			break
		}
	}
	return readings
}

// reading декодирует payload-запись words.dawg (4 байта BE: para, form) в
// указанном шарде.
func (x *Dictionary) reading(shard int, word string, value []byte) (Reading, bool) {
	if len(value) < 4 {
		return Reading{}, false
	}
	para := binary.BigEndian.Uint16(value[:2])
	form := binary.BigEndian.Uint16(value[2:4])
	return x.readingForm(shard, word, para, form), true
}

// readingForm строит Reading по парадигме и форме в указанном шарде
// (норма = prefix₀ + stem + suffix₀ для form≠0, иначе — само слово).
func (x *Dictionary) readingForm(shard int, word string, paraNum, form uint16) Reading {
	para, ok := x.paradigm(shard, paraNum)
	if !ok || int(form) >= para.Len() {
		return Reading{Word: word, Para: paraNum, Form: form, Shard: shard}
	}

	prefix, suffix := x.paradigmAffix(shard, para, int(form))
	norm := word
	if form != 0 {
		stem := strings.TrimPrefix(word, prefix)
		stem = strings.TrimSuffix(stem, suffix)
		p0, s0 := x.paradigmAffix(shard, para, 0)
		norm = p0 + stem + s0
	}

	return Reading{
		Word:   word,
		Normal: norm,
		Tag:    x.paradigmTag(para, int(form)),
		Para:   paraNum,
		Form:   form,
		Shard:  shard,
	}
}

func (x *Dictionary) paradigm(shard int, id uint16) (internal.Paradigm, bool) {
	if shard < 0 || shard >= len(x.d.Paradigms) {
		return internal.Paradigm{}, false
	}
	if int(id) < len(x.d.Paradigms[shard]) {
		return x.d.Paradigms[shard][id], true
	}
	return internal.Paradigm{}, false
}

// paradigmAffix возвращает префикс и суффикс формы парадигмы для
// указанного шарда (пустые при выходе за границы). Prefixes общий для
// всех шардов; Suffixes — свой на шард.
func (x *Dictionary) paradigmAffix(shard int, para internal.Paradigm, form int) (prefix, suffix string) {
	if form >= para.Len() {
		return "", ""
	}
	var suffixes []string
	if shard >= 0 && shard < len(x.d.Suffixes) {
		suffixes = x.d.Suffixes[shard]
	}
	return strAt(x.d.Prefixes, para.Prefix(form)), strAt(suffixes, para.Suffix(form))
}

// paradigmTag возвращает имя тега формы парадигмы. TagSet общий для всех
// шардов, поэтому шард не нужен.
func (x *Dictionary) paradigmTag(para internal.Paradigm, form int) string {
	if form >= para.Len() {
		return ""
	}
	if x.d.TagSet == nil {
		return ""
	}
	return x.d.TagSet.TagName(para.Tag(form))
}

func strAt(ar []string, i uint16) string {
	if int(i) < len(ar) {
		return ar[i]
	}
	return ""
}

// productive — граммема не входит в nonproductiveGrammemes.
func productive(tag string) bool {
	if tag == "" {
		return false
	}
	for part := range strings.SplitSeq(tag, ",") {
		if slices.Contains(nonproductiveGrammemes, part) {
			return false
		}
	}
	return true
}

// nonproductiveGrammemes — граммемы, для которых предсказание не даёт
// продуктивных разборов (pymorphy2).
var nonproductiveGrammemes = []string{"NUMR", "NPRO", "PRED", "PREP", "CONJ", "PRCL", "INTJ", "Apro"}

func suffixSplits(word string, max int) ([][2]string, bool) {
	rr := []rune(word)
	n := len(rr)
	if n == 0 {
		return nil, false
	}
	if max > n {
		max = n
	}
	out := make([][2]string, 0, max)
	for i := 1; i <= max; i++ {
		out = append(out, [2]string{string(rr[:n-i]), string(rr[n-i:])})
	}
	return out, true
}
```

**Verification note for this step:** this reproduces `parse.go`'s existing `exact`/`predict`/`reading`/`readingForm`/`paradigm`/`paradigmAffix`/`paradigmTag`/`strAt`/`productive`/`nonproductiveGrammemes`/`suffixSplits`/`Reading` in full so the whole file is replaced consistently — cross-check against the current file before pasting to make sure no other exported helper in `parse.go` was missed (`grep -n '^func\|^var\|^type' pkg/morphology/parse.go` before editing, compare against this list).

- [ ] **Step 4: Update `fuzzy.go`**

Replace `pkg/morphology/fuzzy.go` in full:

```go
package morphology

import (
	"sort"
	"sync"
	"unicode/utf8"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// FuzzyMatch — найденное по нечёткому поиску слово и расстояние до запроса.
type FuzzyMatch struct {
	Word     string
	Distance int
}

// Fuzzy возвращает слова словаря в пределах расстояния Левенштейна maxDist
// от word (метрика по рунам: вставка/удаление/замена одной руны = 1,
// «ё/е» — одна замена). Результат отсортирован по (расстояние, слово),
// дубликаты слова (несколько чтений, в том числе из разных шардов)
// схлопнуты. Отрицательное maxDist трактуется как 0 (точный поиск).
// Пустой результат — слов нет.
func (x *Dictionary) Fuzzy(word string, maxDist int) []FuzzyMatch {
	if maxDist < 0 {
		maxDist = 0
	}
	return dedupeFuzzy(x.fuzzyWalk(word, maxDist))
}

// FuzzyTop возвращает до maxWords ближайших слов, упорядоченных по
// (расстояние, слово). Расстояние расширяется итеративно от 0 до верхней
// границы (len(query)+наибольшая длина слова в рунах среди всех шардов),
// пока не набраны maxWords слов или не пройден весь словарь. maxWords ≤ 0
// — точный поиск (слово само по себе, либо пусто).
func (x *Dictionary) FuzzyTop(word string, maxWords int) []FuzzyMatch {
	if maxWords <= 0 {
		return x.Fuzzy(word, 0)
	}

	maxRunes := x.maxWordRunes()
	if maxRunes == 0 {
		return nil
	}

	bound := utf8.RuneCountInString(word) + maxRunes
	seen := make(map[string]bool, maxWords)
	var out []FuzzyMatch
	for dist := 0; dist <= bound && len(out) < maxWords; dist++ {
		for _, m := range x.fuzzyWalk(word, dist) {
			if seen[m.Word] {
				continue
			}
			seen[m.Word] = true
			out = append(out, m)
			if len(out) >= maxWords {
				break
			}
		}
	}
	return out
}

func dedupeFuzzy(matches []FuzzyMatch) []FuzzyMatch {
	if len(matches) == 0 {
		return nil
	}
	out := make([]FuzzyMatch, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, m := range matches {
		if seen[m.Word] {
			continue
		}
		seen[m.Word] = true
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Distance != out[j].Distance {
			return out[i].Distance < out[j].Distance
		}
		return out[i].Word < out[j].Word
	})
	return out
}

// fuzzyWalk обходит каждый шард параллельно (по горутине на шард) и
// склеивает результаты. Шарды — независимые DAWG, поэтому обходы не
// делят изменяемое состояние.
func (x *Dictionary) fuzzyWalk(word string, k int) []FuzzyMatch {
	results := make([][]FuzzyMatch, len(x.d.Words))

	var wg sync.WaitGroup
	for shard, dawg := range x.d.Words {
		if dawg == nil {
			continue
		}
		wg.Add(1)
		go func(shard int, dawg *internal.DAWG) {
			defer wg.Done()
			results[shard] = fuzzyWalkShard(dawg, word, k)
		}(shard, dawg)
	}
	wg.Wait()

	var out []FuzzyMatch
	for _, r := range results {
		out = append(out, r...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Distance != out[j].Distance {
			return out[i].Distance < out[j].Distance
		}
		return out[i].Word < out[j].Word
	})
	return out
}

// fuzzyWalkShard — один проход совместного обхода одного шарда DAWG и
// banded DP Левенштейна.
func fuzzyWalkShard(words *internal.DAWG, word string, k int) []FuzzyMatch {
	f := &fuzzySearch{
		words: words,
		q:     []rune(word),
		k:     k,
		path:  make([]byte, 0, 32),
	}

	row := f.rowFor(0)
	for j := range row {
		row[j] = j
	}
	f.visit(0, 0, row)
	return f.out
}

// fuzzySearch переносит состояние одного обхода: rows[depth] — DP-строка
// после depth рун пути, path — байты текущего пути DAWG.
type fuzzySearch struct {
	words *internal.DAWG
	q     []rune
	k     int
	rows  [][]int
	path  []byte
	out   []FuzzyMatch
}

func (f *fuzzySearch) visit(state uint32, depth int, row []int) {
	if f.words.HasPayloadChild(state) {
		if dist := row[len(f.q)]; dist <= f.k {
			f.out = append(f.out, FuzzyMatch{Word: string(f.path), Distance: dist})
		}
	}

	f.words.ForEachChild(state, func(label byte, next uint32) {
		if label == internal.PayloadSeparator {
			return
		}
		start := len(f.path)
		f.path = append(f.path, label)
		f.explore(next, depth, row, start)
		f.path = f.path[:start]
	})
}

// explore завершает текущую руну (путь из байтов с позиции start) и
// применяет DP-переход; продолжает рекурсию, если строка ещё в пределах k.
func (f *fuzzySearch) explore(state uint32, depth int, row []int, start int) {
	tail := f.path[start:]

	if !utf8.FullRune(tail) {
		f.words.ForEachChild(state, func(label byte, next uint32) {
			if label == internal.PayloadSeparator {
				return
			}
			f.path = append(f.path, label)
			f.explore(next, depth, row, start)
			f.path = f.path[:len(f.path)-1]
		})
		return
	}

	r, _ := utf8.DecodeRune(tail)
	if r == utf8.RuneError {
		return
	}

	nrow := f.nextRow(depth, row, r)
	if minRow(nrow) <= f.k {
		f.visit(state, depth+1, nrow)
	}
}

func (f *fuzzySearch) nextRow(depth int, row []int, r rune) []int {
	nrow := f.rowFor(depth + 1)
	nrow[0] = row[0] + 1

	for j := 1; j <= len(f.q); j++ {
		cost := 1
		if f.q[j-1] == r {
			cost = 0
		}
		nrow[j] = min3(row[j]+1, nrow[j-1]+1, row[j-1]+cost)
	}
	return nrow
}

func (f *fuzzySearch) rowFor(depth int) []int {
	for len(f.rows) <= depth {
		f.rows = append(f.rows, make([]int, len(f.q)+1))
	}
	return f.rows[depth]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

func minRow(row []int) int {
	m := row[0]
	for _, v := range row[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// maxWordRunes возвращает наибольшую длину словоформы словаря в рунах,
// среди всех шардов.
func (x *Dictionary) maxWordRunes() int {
	best := 0
	for _, dawg := range x.d.Words {
		if dawg == nil {
			continue
		}
		if m := maxWordRunesInShard(dawg); m > best {
			best = m
		}
	}
	return best
}

// maxWordRunesInShard обходит DAG одного шарда. Обход дедуплицирует узлы
// по наибольшей достигнутой глубине — иначе разделяемые суффиксы
// считались бы экспоненциально. Продолжающие байты UTF-8 (0x80–0xBF)
// руну не добавляют.
func maxWordRunesInShard(words *internal.DAWG) int {
	seen := make(map[uint32]int)
	var rec func(state uint32, runes int)
	rec = func(state uint32, runes int) {
		if prev, ok := seen[state]; ok && prev >= runes {
			return
		}
		seen[state] = runes
		words.ForEachChild(state, func(label byte, next uint32) {
			if label == internal.PayloadSeparator {
				return
			}
			step := 1
			if label >= 0x80 && label <= 0xBF {
				step = 0
			}
			rec(next, runes+step)
		})
	}
	rec(0, 0)

	max := 0
	for _, depth := range seen {
		if depth > max {
			max = depth
		}
	}
	return max
}
```

**Verification note for this step:** before pasting, run `grep -n '^func\|^type' pkg/morphology/fuzzy.go` against the current file and confirm every symbol it lists appears in the replacement above (this reproduces the file in full, parameterizing `x.d.Words` from a single `*DAWG` to per-shard fan-out).

- [ ] **Step 5: Verify `pkg/morphology` compiles on its own**

Run: `go build ./pkg/morphology/`
Expected: **fails** — `pkg/morphology/importers/opencorpora/import.go` and `pkg/morphology/importers/pymorphy2/import.go` (already fixed in Task 3) are imported by `open.go`, and `opencorpora`'s `ImportFromXML` still calls the old-shape `NewDictionary` until Task 5. Confirm the *only* compile errors are inside `pkg/morphology/importers/opencorpora/import.go` — if `pkg/morphology`'s own files (`save.go`, `open.go`, `parse.go`, `fuzzy.go`, `dictionary.go`) produce any error, fix those before proceeding; they must be clean on their own.

To check `pkg/morphology`'s own files in isolation without `opencorpora`'s not-yet-fixed code blocking the build, temporarily verify with:

Run: `go vet ./pkg/morphology/... 2>&1 | grep -v 'importers/opencorpora'`
Expected: no output (no errors outside the not-yet-fixed `opencorpora` package).

- [ ] **Step 6: Commit**

```bash
git add pkg/morphology/save.go pkg/morphology/open.go pkg/morphology/parse.go pkg/morphology/fuzzy.go
git commit -m "morphology: thread per-shard storage through save/open/lookup

.dat sections become suffixes-N/paradigms-N/words.dawg-N, mirroring the
existing prediction-N convention. Lookup (exact, fuzzy) fans out across
shards in parallel goroutines and merges results. Reading gains an
additive Shard field so Para/Form stay disambiguable across shards.

pkg/morphology/importers/opencorpora still calls the old NewDictionary
shape — the module doesn't build as a whole until the next commit.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

## Task 5: `opencorpora.ImportFromXML` builds multiple shards

**Files:**
- Modify: `pkg/morphology/importers/opencorpora/import.go`
- Modify: `pkg/morphology/importers/opencorpora/import_test.go`

**Interfaces:**
- Consumes: `ShardingStrategy`/`FillOnDemand` (Task 1), `internal.NewDictionary`'s new signature (Task 2).
- Produces: `ImportFromXML` now returns a `*internal.Dictionary` whose `Suffixes`/`Paradigms`/`Words` may have more than one element.

**Note:** after this task, the whole module builds and `go test ./... -race` is green again — this is the task that closes the gap opened by Tasks 2–4.

- [ ] **Step 1: Replace `ImportFromXML` and add the shard-build helper type**

In `pkg/morphology/importers/opencorpora/import.go`, replace the `ImportFromXML` function (currently lines 76–197, from `func ImportFromXML(...)` through its closing `return dict, nil` / `}`) with:

```go
// suffixShardLimit — вместимость uint16 id-пространства суффиксов одного
// шарда.
const suffixShardLimit = 1 << 16

// shardBuild накапливает состояние (суффиксы, парадигмы, ключи DAWG)
// одного шарда во время ImportFromXML.
type shardBuild struct {
	suffixList     []string
	suffixTexts    map[string]uint16
	paradigmsDedup map[string]uint16 // paradigmKeyHash -> paradigmID
	paradigms      []internal.Paradigm
	dawgEntries    []dawgEntry
}

func newShardBuild() *shardBuild {
	return &shardBuild{
		suffixTexts:    make(map[string]uint16),
		paradigmsDedup: make(map[string]uint16),
	}
}

// ImportFromXML читает dict.xml из r и возвращает *internal.Dictionary с
// заполненными TagSet, Suffixes, Prefixes, Paradigms и Words (DAWG).
//
// Pipeline:
//
//  1. xmlscan собирает все леммы с словоформами.
//  2. Для каждой леммы вычисляется LCP-stem всех словоформ.
//  3. Каждая словоформа → (suffix_id, tag_id) в парадигму текущего шарда.
//     Suffix id адресуется uint16, поэтому леммы делятся на несколько
//     шардов с независимыми id-пространствами, если суффиксов больше,
//     чем помещается в один uint16-диапазон (см. FillOnDemand в shard.go
//     и docs/superpowers/specs/2026-09-14-suffix-sharding-design.md).
//  4. Парадигмы дедуплицируются через map[paradigmKey] → paradigmID,
//     отдельно на каждый шард.
//  5. Строится DAWG из ключей (stem + suffix, value=paraID<<16|formIdx),
//     отдельно на каждый шард.
//
// progress — необязательный callback для вывода прогресса. Вызывается
// периодически с (processed, total), где total — суммарное число ключей
// DAWG по всем шардам и processed растёт непрерывно через границы шардов
// (CLI видит один общий прогресс-бар, а не рестарт на каждом шарде).
func ImportFromXML(r io.Reader, tagSet *internal.TagSet, progress Progress) (*internal.Dictionary, error) {
	if tagSet == nil {
		tagSet = internal.NewTagSet("opencorpora")
	}

	// Phase 1: scan XML and collect lemmas + forms.
	var lemmas []lemmaEntry

	handler := &xmlHandler{
		tagSet:  tagSet,
		lemmas:  &lemmas,
		curForm: nil,
		err:     nil,
	}

	if err := xmlscan.New(r, handler).Scan(); err != nil {
		return nil, fmt.Errorf("opencorpora: scan: %w", err)
	}
	if handler.err != nil {
		return nil, fmt.Errorf("opencorpora: handler: %w", handler.err)
	}

	// Phase 2: build suffix text -> ID map and extract paradigms, one
	// shard at a time.
	strategy := FillOnDemand{}
	cur := newShardBuild()
	shards := []*shardBuild{cur}

	for _, lem := range lemmas {
		if len(lem.forms) == 0 {
			continue
		}

		stem := lcp(lem.forms)

		// How many suffixes would this lemma add to the CURRENT shard if
		// placed there? Dedup within the lemma's own forms too, so a
		// lemma reusing one suffix across several forms counts once.
		newInLemma := make(map[string]bool)
		for _, frm := range lem.forms {
			suffix := ""
			if len(stem) < len(frm.text) {
				suffix = frm.text[len(stem):]
			}
			if _, ok := cur.suffixTexts[suffix]; !ok {
				newInLemma[suffix] = true
			}
		}

		if strategy.Boundary(len(cur.suffixTexts), len(newInLemma), suffixShardLimit) {
			cur = newShardBuild()
			shards = append(shards, cur)
		}

		var suffixIDs []uint16
		var tagIDs []uint16

		for _, frm := range lem.forms {
			suffix := ""
			if len(stem) < len(frm.text) {
				suffix = frm.text[len(stem):]
			}

			sid, ok := cur.suffixTexts[suffix]
			if !ok {
				if len(cur.suffixList) >= suffixShardLimit {
					return nil, fmt.Errorf("opencorpora: shard %d exceeded %d unique suffixes despite sharding strategy", len(shards)-1, suffixShardLimit)
				}
				sid = uint16(len(cur.suffixList))
				cur.suffixTexts[suffix] = sid
				cur.suffixList = append(cur.suffixList, suffix)
			}
			suffixIDs = append(suffixIDs, sid)

			tid, err := tagSet.Add(frm.gramm)
			if err != nil {
				return nil, fmt.Errorf("opencorpora: %w", err)
			}
			tagIDs = append(tagIDs, tid)
		}

		hash := paradigmKeyHash(suffixIDs, tagIDs)
		paraID, ok := cur.paradigmsDedup[hash]
		if !ok {
			if len(cur.paradigms) >= 1<<16 {
				return nil, fmt.Errorf("opencorpora: shard %d exceeded 65536 unique paradigms", len(shards)-1)
			}
			paraID = uint16(len(cur.paradigms))
			cur.paradigmsDedup[hash] = paraID

			prefixes := make([]uint16, len(lem.forms))
			para := internal.NewParadigm(suffixIDs, tagIDs, prefixes)
			cur.paradigms = append(cur.paradigms, para)
		}

		for formIdx := 0; formIdx < len(lem.forms); formIdx++ {
			suffix := ""
			if len(stem) < len(lem.forms[formIdx].text) {
				suffix = lem.forms[formIdx].text[len(stem):]
			}
			dawgKey := stem + suffix
			val := uint32(paraID)<<16 | uint32(formIdx)
			cur.dawgEntries = append(cur.dawgEntries, dawgEntry{key: dawgKey, val: val})
		}
	}

	// Phase 3: dedup DAWG entries per shard, then build one DAWG per
	// shard, reporting progress cumulatively across all shards.
	totalEntries := 0
	for _, s := range shards {
		s.dawgEntries = dedupEntries(s.dawgEntries)
		totalEntries += len(s.dawgEntries)
	}

	suffixesPerShard := make([][]string, len(shards))
	paradigmsPerShard := make([][]internal.Paradigm, len(shards))
	wordsPerShard := make([]*internal.DAWG, len(shards))

	processedBefore := 0
	for i, s := range shards {
		dawgKeys := make([]string, len(s.dawgEntries))
		dawgValues := make([]uint32, len(s.dawgEntries))
		for j, e := range s.dawgEntries {
			dawgKeys[j] = e.key
			dawgValues[j] = e.val
		}

		base := processedBefore
		shardProgress := func(processed, _ int) {
			if progress != nil {
				progress(base+processed, totalEntries)
			}
		}
		dawg, err := internal.BuildDAWGWithValuesProgress(dawgKeys, dawgValues, shardProgress)
		if err != nil {
			return nil, fmt.Errorf("opencorpora: build DAWG (shard %d): %w", i, err)
		}

		suffixesPerShard[i] = s.suffixList
		paradigmsPerShard[i] = s.paradigms
		wordsPerShard[i] = dawg
		processedBefore += len(s.dawgEntries)
	}

	dict := internal.NewDictionary(
		"ru",
		tagSet,
		suffixesPerShard,
		nil, // Prefixes: OpenCorpora lemmas carry their own prefixes
		paradigmsPerShard,
		wordsPerShard,
		internal.RussianCharPolicy(),
	)
	dict.Info = &internal.BuildInfo{Source: "opencorpora"}

	return dict, nil
}
```

- [ ] **Step 2: Verify the module builds**

Run: `go build ./... && go vet ./...`
Expected: success — this is the task that makes the whole module compile again after Tasks 2–4.

- [ ] **Step 3: Update the basic fixture-based tests**

In `pkg/morphology/importers/opencorpora/import_test.go`, replace `TestImportFromXMLBasic` through `TestImportFromXMLNoForms` (lines 56–145) in full:

```go
func TestImportFromXMLBasic(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)
	require.NotNil(t, d)

	assert.Equal(t, "ru", d.Language)
	require.NotNil(t, d.TagSet)
	require.Len(t, d.Words, 1, "small fixture must not overflow into more than one shard")
	require.NotNil(t, d.Words[0])

	assert.Greater(t, len(d.TagSet.Tags), 0)
	require.Len(t, d.Suffixes, 1)
	assert.Greater(t, len(d.Suffixes[0]), 0)
	require.Len(t, d.Paradigms, 1)
	assert.Greater(t, len(d.Paradigms[0]), 0)

	items := d.Words[0].SimilarItems("кот", d.CharPolicy)
	assert.Greater(t, len(items), 0, "кот должен быть найден")
	if len(items) > 0 {
		assert.GreaterOrEqual(t, len(items[0].Values), 1, "кот имеет хотя бы один разбор")
	}
}

func TestImportFromXMLParadigmsDedup(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)

	require.Len(t, d.Paradigms, 1)
	assert.LessOrEqual(t, len(d.Paradigms[0]), 3, "число парадигм <= число лемм")
}

func TestImportFromXMLStemLCP(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)

	require.Len(t, d.Suffixes, 1)
	require.Greater(t, len(d.Suffixes[0]), 0)
	assert.Equal(t, "", d.Suffixes[0][0], "первый суффикс в шарде 0 должен быть пустым")
}

func TestImportFromXMLDAWGContains(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)
	require.Len(t, d.Words, 1)

	// DAWG keys include payload suffixes, so SimilarItems is the correct lookup.
	for _, w := range []string{"кот", "кота", "мышь", "мыши"} {
		items := d.Words[0].SimilarItems(w, d.CharPolicy)
		assert.Greater(t, len(items), 0, "слово %q должно быть найдено через SimilarItems", w)
	}
}

func TestImportFromXMLRoundtrip(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)
	require.Len(t, d.Words, 1)

	items := d.Words[0].SimilarItems("кот", d.CharPolicy)
	require.GreaterOrEqual(t, len(items), 1)
	require.GreaterOrEqual(t, len(items[0].Values), 1, "кот имеет хотя бы один разбор")

	for _, v := range items[0].Values {
		require.Len(t, v, 4, "значение должно быть 4 байта")
	}
}

func TestImportFromXMLNoForms(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">существительное</grammeme>
  <grammeme id="nomn">им. п.</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="пустая">
   <l g="NOUN">
   </l>
  </lemma>
  <lemma id="2" text="есть">
   <l g="NOUN">
    <f t="есть">
     <g v="nomn"/>
    </f>
   </l>
  </lemma>
 </lemmata>
</dictionary>`

	d, err := opencorpora.CompileFromXML(strings.NewReader(xml), nil)
	require.NoError(t, err)
	require.Len(t, d.Words, 1)

	items := d.Words[0].SimilarItems("есть", d.CharPolicy)
	assert.Greater(t, len(items), 0, "есть должно быть найдено")

	items2 := d.Words[0].SimilarItems("пустая", d.CharPolicy)
	assert.Equal(t, 0, len(items2), "пустая не должна быть найдена")
}
```

- [ ] **Step 4: Replace the overflow test with a sharding test**

Replace `TestImportFromXMLTooManySuffixes` (the function added during the pre-1.0 review-triage session, currently asserting a hard error) with:

```go
// TestImportFromXMLShardsOnSuffixOverflow verifies that crossing the
// uint16 suffix-id limit (65536) makes ImportFromXML shard the
// dictionary instead of erroring or silently wrapping ids. Before
// sharding, id 65536 would wrap to 0, colliding with the very first
// suffix ever registered — see the critical finding in
// docs/code-review-pre-1.0.md and the design in
// docs/superpowers/specs/2026-09-14-suffix-sharding-design.md.
func TestImportFromXMLShardsOnSuffixOverflow(t *testing.T) {
	const uniqueSuffixes = 1 << 16 // one more than fits in a single uint16 shard

	var xml strings.Builder
	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?><dictionary><lemmata>`)
	for i := 0; i < uniqueSuffixes; i++ {
		// Two forms sharing stem "слово": one bare (suffix ""), one with a
		// suffix unique to this lemma — pushes total unique suffixes past
		// the single-shard limit.
		fmt.Fprintf(&xml, `<lemma id="%d"><l t="слово"/><f t="слово"/><f t="слово%06d"/></lemma>`, i, i)
	}
	xml.WriteString(`</lemmata></dictionary>`)

	d, err := opencorpora.CompileFromXML(strings.NewReader(xml.String()), nil)
	require.NoError(t, err)
	require.Greater(t, len(d.Words), 1, "65536 unique suffixes must not fit in a single uint16 shard")
	require.Len(t, d.Suffixes, len(d.Words))
	require.Len(t, d.Paradigms, len(d.Words))

	for i, suffixes := range d.Suffixes {
		assert.LessOrEqual(t, len(suffixes), 1<<16, "shard %d exceeds the uint16 suffix-id space", i)
	}

	// Sample (not exhaustive — 65536 lemmas is already enough data)
	// wordforms across the range and confirm each is still findable in
	// whichever shard holds it.
	found := 0
	const sampleStride = 997
	for i := 0; i < uniqueSuffixes; i += sampleStride {
		word := fmt.Sprintf("слово%06d", i)
		for _, dawg := range d.Words {
			if len(dawg.SimilarItems(word, d.CharPolicy)) > 0 {
				found++
				break
			}
		}
	}
	assert.Equal(t, (uniqueSuffixes+sampleStride-1)/sampleStride, found,
		"every sampled sharded wordform must be findable in some shard")
}
```

- [ ] **Step 5: Update `TestImportFromXMLPropertyTest`**

Replace it (currently the last function in the file) with:

```go
func TestImportFromXMLPropertyTest(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)
	require.Len(t, d.Words, 1)

	words := []string{"кот", "кота", "мышь", "мыши"}
	for _, w := range words {
		items := d.Words[0].SimilarItems(w, d.CharPolicy)
		assert.GreaterOrEqual(t, len(items), 1, "слово %q должно быть найдено", w)
	}
}
```

- [ ] **Step 6: Run the opencorpora package's tests**

Run: `go test ./pkg/morphology/importers/opencorpora/... -race -count=1 -v`
Expected: PASS, including `TestImportFromXMLShardsOnSuffixOverflow` — this test builds two full DAWGs from 65536 synthetic lemmas; if it takes noticeably longer than the rest of the suite that's expected (it was already true of the honest-error version it replaces, which still had to scan all 65536 lemmas before failing).

- [ ] **Step 7: Run the full module test suite**

Run: `go build ./... && go vet ./... && go test ./... -race -count=1`
Expected: green — this is the point where the whole refactor is done and every existing test (across `pkg/common`, `internal/xmlscan`, `pkg/morphology`, `pkg/morphology/internal`, `pkg/morphology/importers/opencorpora`, `pkg/morphology/importers/pymorphy2`, `pkg/opencorpora`) passes again.

Run: `GOOS=windows GOARCH=amd64 go build ./...`
Expected: success (this project is Unix-only for mmap but must still cross-compile cleanly — established during the pre-1.0 review triage session).

- [ ] **Step 8: Commit**

```bash
git add pkg/morphology/importers/opencorpora/import.go pkg/morphology/importers/opencorpora/import_test.go
git commit -m "opencorpora: build multiple shards via FillOnDemand

ImportFromXML now shards lemmas by suffix-id overflow instead of
erroring past 65536 unique suffixes. TestImportFromXMLTooManySuffixes
(added during the pre-1.0 review-triage session, asserting the honest
error) is replaced by TestImportFromXMLShardsOnSuffixOverflow, which
asserts the same input now succeeds into 2 shards. This closes the
second critical bug from docs/code-review-pre-1.0.md — the whole
module builds and go test ./... -race is green again.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

## Task 6: Verify against the real corpus and update docs

**Files:**
- No source changes — verification only, plus documentation updates.
- Modify: `docs/todo.md` (mark the suffix-sharding item done)

**Interfaces:**
- Consumes: everything from Tasks 1–5.

- [ ] **Step 1: Compile the real OpenCorpora dictionary end-to-end**

Run: `go run ./cmd/gomorphy_build compile -o /tmp/sharded-real.dat`

Expected: succeeds (this currently fails on `master` with `too many unique suffixes (max 65536)` — that's exactly the bug this plan fixes). Note how many shards it built (should be 2, matching the empirical analysis in the spec: 391842 lemmas, 65835 unique suffixes).

- [ ] **Step 2: Spot-check lookups against the compiled dictionary**

Write a short throwaway script (do not commit it) that opens `/tmp/sharded-real.dat` via `morphology.Open` and calls `.Parse("кота")`, `.Parse("ежа")`, and a handful of other words already used as manual checks earlier in this project's history, confirming non-empty results. Delete the script when done; this step is manual verification, not a new automated test (the automated coverage is `TestImportFromXMLShardsOnSuffixOverflow` from Task 5 plus the existing `TestSaveOpenRoundtrip`/`TestOpenPyMorphy`-style tests, which already exercise the sharded save/open path structurally).

- [ ] **Step 3: Update `docs/todo.md`**

In `docs/todo.md`, find the section header `### Суффиксы: шардирование — СПЕКА ГОТОВА (release-blocking)` and change it to:

```
### Суффиксы: шардирование — ВЫПОЛНЕНО
```

Immediately below the existing paragraph describing the decision, add:

```
**Реализовано** (2026-09-14, план:
[2026-09-14-suffix-sharding.md](superpowers/plans/2026-09-14-suffix-sharding.md)):
`gomorphy_build compile` на полном `.data/opencorpora/dict.xml` теперь
успешно собирается в 2 шарда. `FillOnDemand` — единственная стратегия,
интерфейс `ShardingStrategy` открыт для дальнейших (paradigm-grouped и
т.п.), если понадобятся при большем переполнении.
```

- [ ] **Step 4: Final full-suite check and commit the docs update**

Run: `go build ./... && go vet ./... && go test ./... -race -count=1`
Expected: green.

```bash
git add docs/todo.md
git commit -m "docs: mark suffix-sharding done, verified against real dict.xml

gomorphy_build compile now succeeds end-to-end on the full OpenCorpora
dict.xml (2 shards). Closes the second critical bug from
docs/code-review-pre-1.0.md.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

## Self-Review Notes

- **Spec coverage:** shared TagSet/Prefixes (Task 2), per-shard Suffixes/Paradigms/Words (Task 2), numbered `.dat` sections mirroring `prediction-N` (Task 4), `ShardingStrategy`/`FillOnDemand` (Task 1), parallel goroutine fan-out for lookup (Task 4, both `exact` and `fuzzyWalk`), `Reading.Shard` (Task 4), `ImportFromXML` sharding (Task 5), `TestImportFromXMLTooManySuffixes` behavior change (Task 5), real-corpus verification (Task 6) — all spec sections have a task. Deferred-ideas (uint32/uint8 variants) and the lemma-loss follow-up are explicitly out of scope per the spec and are not tasks here.
- **Deviation from the spec's illustrative interface:** the spec sketched `ShardingStrategy.Assign(lemmas []lemmaEntry, limit int) (shardOf []int, numShards int)` — a whole-corpus precompute. This plan uses `Boundary(currentCount, newSuffixes, limit int) bool` instead, called inline during the single real accumulation pass. Reason found during planning: a separate precompute pass would need to reproduce the exact same suffix-accounting as the real Phase-2 loop or risk a shard boundary that doesn't actually hold once the real per-lemma-form logic runs (e.g. it dedups suffixes shared across a lemma's own forms) — inlining the decision into the one real pass makes divergence impossible by construction. The spec's own stated intent (FillOnDemand semantics, extensible interface, no format/pipeline impact) is preserved.
- **Type consistency:** `internal.Dictionary.{Suffixes,Paradigms,Words}` (Task 2) are consumed with matching types by `save.go`/`open.go`/`parse.go`/`fuzzy.go` (Task 4) and produced with matching types by `pymorphy2/import.go` (Task 3) and `opencorpora/import.go` (Task 5). `Reading.Shard int` (Task 4) is set by both `readingForm` call sites (`exactInShard`'s call via `reading`, and `predictForPrefix`'s direct call). `ShardingStrategy`/`FillOnDemand` (Task 1) is consumed by `opencorpora/import.go` (Task 5) with the exact method signature defined in Task 1.
