# DAWG Free-List Build Placement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `BuildDAWG`/`BuildDAWGWithValues` construction scale near-linearly
in the number of trie nodes instead of quadratically, so compiling the
~1.5M-wordform OpenCorpora dictionary drops from ~24h to a small number of
minutes, with no change to the DAWG wire format or public API.

**Architecture:** Replace the double-array packer's placement search
(`compileImpl` → `findBaseBitset`, a linear scan over a bitset from
`base=1` for every node) with a `slotAllocator` type that threads an
intrusive doubly-linked free list through unused array slots (Aoe 1989,
the technique dawgdic/Darts/cedar use), plus a 256-entry per-first-label
hint cache so searches start near a promising candidate. Six dead
`findBase*`/`FreeList`/`ubit` variants are deleted in the same pass.

**Tech Stack:** Go 1.27, `testify` (assert/require) for tests, existing
`pkg/morphology/internal` package (no new dependencies).

**Spec:** `docs/superpowers/specs/2026-09-14-dawg-build-freelist-design.md`

## Global Constraints

- No change to the on-disk DAWG format (`dictionary uint32[]` + `guide byte[]`)
  read by `dawg.go` — `unitAt`, `offset`, `encodable`, `hasLeafBit`,
  `isLeafBit`, `extensionBit` are unchanged and reused as-is.
- No change to the public API: `BuildDAWG`, `BuildDAWGWithValues`,
  `BuildDAWGWithValuesProgress` keep their exact signatures.
- No change to phase-1 list-form construction/minimization
  (`buildDAWGWithPayload`, `dawgBuilder.appendByte`/`closeSuffix`/
  `replaceOrRegister`/`chainSig`) or to `buildGuide`.
- All existing tests in `pkg/morphology/internal` must keep passing
  unchanged (same `Contains`/`Find`/`SimilarItems` results), except
  `zz_findbase_test.go`, which is deleted because it tests the dead
  `findBase`/`ubit` code being removed.

---

## File Structure

- **Create:** `pkg/morphology/internal/dawgbuild_freelist.go` — the
  `slotAllocator` type: free-list placement search, growth, hint cache.
  Isolated from `compileImpl` so it can be unit-tested on its own.
- **Create:** `pkg/morphology/internal/dawgbuild_freelist_test.go` — unit
  tests for `slotAllocator` alone (no DAWG/trie concepts).
- **Modify:** `pkg/morphology/internal/dawgbuild.go` — `compileImpl` calls
  `slotAllocator.alloc` instead of `findBaseBitset`; delete `FreeList`,
  `rangeEntry`, `newFreeList`, `(*FreeList).markUsed`,
  `(*FreeList).findFirstFreeOnwards`, `findBaseWithFreeList`,
  `findBaseBitset`, `findBaseBoolRandom`, `findBaseBoolBlock`,
  `findBaseBool`, `findBase`, `ubit` and its methods, `setBit`, `testBit`,
  and the `estimatedSize`/`used []uint64` pre-sizing in `compileImpl`.
- **Delete:** `pkg/morphology/internal/zz_findbase_test.go` — its only
  subject (`findBase`/`ubit`) no longer exists.
- **Create:** `pkg/morphology/internal/dawgbuild_scaling_test.go` — the
  synthetic scaling benchmark (skipped by default via `testing.Short()` /
  build tag, run explicitly for verification).

---

### Task 1: `slotAllocator` — free-list placement, in isolation

**Files:**
- Create: `pkg/morphology/internal/dawgbuild_freelist.go`
- Test: `pkg/morphology/internal/dawgbuild_freelist_test.go`

**Interfaces:**
- Consumes: `encodable(rel uint32) bool` (already defined in
  `dawgbuild.go`, same package — do not redefine it).
- Produces (used by Task 2):
  - `newSlotAllocator() *slotAllocator`
  - `(a *slotAllocator) alloc(index uint32, labels []byte) (base uint32, ok bool)`
  - `(a *slotAllocator) cap() uint32`

- [ ] **Step 1: Write the failing tests**

```go
package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlotAllocatorFirstAllocAvoidsRoot(t *testing.T) {
	a := newSlotAllocator()
	base, ok := a.alloc(0, []byte{0x41})
	require.True(t, ok)
	assert.NotZero(t, base, "base must never be slot 0 (reserved for DAWG root)")
	assert.True(t, encodable(0^base))
}

func TestSlotAllocatorRejectsCollidingBase(t *testing.T) {
	a := newSlotAllocator()
	// Force slot 5 and 5^0x10=21 to be used, so a later request for the
	// same label must not return base=5.
	base1, ok := a.alloc(0, []byte{0x10})
	require.True(t, ok)

	// Directly probe: base1 and base1^0x10 must now be reported used.
	base2, ok := a.alloc(1, []byte{0x10})
	require.True(t, ok)
	assert.NotEqual(t, base1, base2, "second alloc must not reuse base1's now-occupied slots")
	// The two children (base^0x10) must also be distinct and not equal to
	// either base, otherwise the double-array would alias two nodes.
	child1 := base1 ^ 0x10
	child2 := base2 ^ 0x10
	assert.NotEqual(t, child1, base2)
	assert.NotEqual(t, child2, base1)
}

func TestSlotAllocatorNoChildrenStillReservesBase(t *testing.T) {
	a := newSlotAllocator()
	base, ok := a.alloc(0, nil)
	require.True(t, ok)
	assert.NotZero(t, base)

	// A second alloc for the same index/no-children case must not reuse it.
	base2, ok := a.alloc(0, nil)
	require.True(t, ok)
	assert.NotEqual(t, base, base2)
}

func TestSlotAllocatorGrowsCapacityOnDemand(t *testing.T) {
	a := newSlotAllocator()
	assert.Less(t, a.cap(), uint32(1<<20), "must start small, not pre-sized to 1<<30")

	// Exhaust small capacities repeatedly; allocator must keep succeeding
	// by growing, never returning ok=false for a satisfiable request.
	seen := make(map[uint32]bool)
	for i := 0; i < 5000; i++ {
		base, ok := a.alloc(uint32(i), []byte{byte(i % 7), byte(i % 251)})
		require.True(t, ok)
		require.False(t, seen[base], "base %d reused", base)
		seen[base] = true
	}
	assert.Greater(t, a.cap(), uint32(5000))
}

func TestSlotAllocatorHintStartsSearchNearLastSuccess(t *testing.T) {
	a := newSlotAllocator()
	base1, ok := a.alloc(0, []byte{0x22, 0x30})
	require.True(t, ok)

	// A later alloc with the same first label should land at or after
	// base1 (the hint should skip the region already known to be occupied
	// near the head), not necessarily back at the very first free slot.
	base2, ok := a.alloc(1, []byte{0x22})
	require.True(t, ok)
	assert.GreaterOrEqual(t, base2, base1)
}
```

- [ ] **Step 2: Run tests to verify they fail (type doesn't exist yet)**

Run: `go test ./pkg/morphology/internal/... -run TestSlotAllocator -v`
Expected: FAIL — `undefined: newSlotAllocator` (compile error).

- [ ] **Step 3: Implement `slotAllocator`**

```go
package internal

// slotAllocator подбирает свободные base-слоты double-array раскладки
// методом intrusive doubly-linked free list (техника dawgdic/cedar/Darts,
// Aoe 1989): свободные слоты связаны в список через next/prev, поиск идёт
// только по нему — уже занятые слоты никогда не пересматриваются заново,
// в отличие от линейного сканирования битсета с нуля на каждый узел.
//
// Слот 0 зарезервирован под корень DAWG (compileImpl всегда размещает
// корень по индексу 0) и служит sentinel-значением "нет соседа" для
// списка свободных: он никогда не свободен, поэтому 0 однозначно значит
// "конец/начало списка".
type slotAllocator struct {
	used []bool
	next []uint32 // next[i]: следующий свободный слот после i; 0 — конца списка нет
	prev []uint32 // prev[i]: предыдущий свободный слот перед i; 0 — начала списка нет
	head uint32   // первый свободный слот; 0, если список пуст
	tail uint32   // последний свободный слот; 0, если список пуст
	hint [256]uint32
}

// maxSlotAllocatorCap — защитный потолок: реальные словари никогда его не
// достигают (сотни миллионов узлов), это лишь замена прежнему
// errDAWGBuild-пути на случай патологического набора ключей.
const maxSlotAllocatorCap = 1 << 30

func newSlotAllocator() *slotAllocator {
	return &slotAllocator{
		used: []bool{true}, // слот 0 занят с самого начала (корень)
		next: []uint32{0},
		prev: []uint32{0},
	}
}

func (a *slotAllocator) cap() uint32 { return uint32(len(a.used)) }

// grow расширяет ёмкость минимум до n слотов, добавляя новые слоты в
// хвост списка свободных по возрастанию индекса.
func (a *slotAllocator) grow(n uint32) {
	old := a.cap()
	if n <= old {
		return
	}
	a.used = append(a.used, make([]bool, n-old)...)
	a.next = append(a.next, make([]uint32, n-old)...)
	a.prev = append(a.prev, make([]uint32, n-old)...)
	for i := old; i < n; i++ {
		a.linkTail(i)
	}
}

func (a *slotAllocator) linkTail(i uint32) {
	if a.head == 0 {
		a.head = i
		a.tail = i
		return
	}
	a.next[a.tail] = i
	a.prev[i] = a.tail
	a.tail = i
}

func (a *slotAllocator) unlink(i uint32) {
	p, n := a.prev[i], a.next[i]
	if p != 0 {
		a.next[p] = n
	} else {
		a.head = n
	}
	if n != 0 {
		a.prev[n] = p
	} else {
		a.tail = p
	}
	a.prev[i], a.next[i] = 0, 0
}

// markUsed резервирует слот i, при необходимости расширяя ёмкость.
// No-op, если слот уже занят.
func (a *slotAllocator) markUsed(i uint32) {
	if i >= a.cap() {
		a.grow(i + 1)
	}
	if a.used[i] {
		return
	}
	a.used[i] = true
	a.unlink(i)
}

// isUsed сообщает, занят ли слот i (слоты за пределами текущей ёмкости
// считаются свободными — они ещё не были никому нужны).
func (a *slotAllocator) isUsed(i uint32) bool {
	return i < a.cap() && a.used[i]
}

// fits проверяет, что base сам свободен и все base^label для labels тоже
// свободны — т.е. узел можно разместить с этим base без коллизий.
func (a *slotAllocator) fits(base uint32, labels []byte) bool {
	if base == 0 || a.isUsed(base) {
		return false
	}
	for _, l := range labels {
		if a.isUsed(base ^ uint32(l)) {
			return false
		}
	}
	return true
}

// commit резервирует base и все дочерние слоты base^label.
func (a *slotAllocator) commit(base uint32, labels []byte) {
	a.markUsed(base)
	for _, l := range labels {
		a.markUsed(base ^ uint32(l))
	}
}

// alloc находит и резервирует base для узла на позиции index: сам base и
// все base^label для labels должны быть свободны, а index^base —
// представимо в offset-поле единицы словаря (encodable). Поиск идёт по
// списку свободных слотов начиная с подсказки для первого лейбла (если
// есть и всё ещё свободна), иначе с начала списка. При исчерпании списка
// без успеха — расширяет ёмкость вдвое и делает полный проход заново;
// такое случается только на границах роста, поэтому суммарная стоимость
// повторных полных проходов ограничена O(n log n), а не O(n) на узел.
func (a *slotAllocator) alloc(index uint32, labels []byte) (uint32, bool) {
	var startLabel byte
	if len(labels) > 0 {
		startLabel = labels[0]
	}

	base := a.head
	if h := a.hint[startLabel]; h != 0 && !a.isUsed(h) {
		base = h
	}

	for {
		for ; base != 0; base = a.next[base] {
			if encodable(index^base) && a.fits(base, labels) {
				a.commit(base, labels)
				a.hint[startLabel] = base
				return base, true
			}
		}
		if a.cap() >= maxSlotAllocatorCap {
			return 0, false
		}
		newCap := a.cap() * 2
		if newCap > maxSlotAllocatorCap {
			newCap = maxSlotAllocatorCap
		}
		a.grow(newCap)
		base = a.head
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/morphology/internal/... -run TestSlotAllocator -v`
Expected: PASS (all 5 tests).

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/internal/dawgbuild_freelist.go pkg/morphology/internal/dawgbuild_freelist_test.go
git commit -m "feat: add free-list slot allocator for double-array packing"
```

---

### Task 2: Wire `slotAllocator` into `compileImpl`, delete dead code

**Files:**
- Modify: `pkg/morphology/internal/dawgbuild.go:262-354` (`compileImpl`),
  and delete the dead functions/types listed in Global Constraints from
  the same file.
- Delete: `pkg/morphology/internal/zz_findbase_test.go`

**Interfaces:**
- Consumes: `slotAllocator`, `newSlotAllocator`, `(*slotAllocator).alloc`
  from Task 1.
- Produces: `compileImpl` behavior unchanged from the outside — same
  `*DAWG` output for the same input keys (verified by existing tests).

- [ ] **Step 1: Confirm the regression baseline passes before touching anything**

Run: `go test ./pkg/morphology/... -race -v 2>&1 | tail -60`
Expected: PASS (this is the pre-change baseline; note the run time for
`dawgbuild_test.go`'s stress tests for later comparison).

- [ ] **Step 2: Replace `compileImpl`'s body**

Replace the function in `pkg/morphology/internal/dawgbuild.go` (currently
lines 262-354, from `func (b *dawgBuilder) compileImpl(...)` through its
closing brace) with:

```go
// compileImpl — shared implementation for compileWithProgress and compileWithTotal.
func (b *dawgBuilder) compileImpl(totalNodes int32, progress func(processed, total int)) (*DAWG, error) {
	if b.nodes[b.root].first == 0 {
		dic := []uint32{1 << 10}
		dic[0] = 1 << 10
		return NewDAWG(dic, nil), nil
	}

	dic := make([]uint32, 1)
	dic[0] = 0

	alloc := newSlotAllocator()
	link := make(map[int32]uint32)

	processedNodes := int32(0)
	progressEvery := totalNodes / 10000
	if progressEvery < 10 {
		progressEvery = 10
	}

	var dfs func(n int32, index uint32) bool
	dfs = func(n int32, index uint32) bool {
		node := &b.nodes[n]
		first := node.first

		if first != 0 && b.merged[first] {
			if base, ok := link[first]; ok && encodable(index^base) {
				setAt(&dic, index, unitAt(index^base, node.label, node.leaf))
				processedNodes++
				if progress != nil && processedNodes%progressEvery == 0 {
					progress(int(processedNodes), 0)
				}
				return true
			}
		}

		b.labelsBuf = b.labelsBuf[:0]
		for e := first; e != 0; e = b.nodes[e].next {
			b.labelsBuf = append(b.labelsBuf, b.nodes[e].label)
		}

		base, ok := alloc.alloc(index, b.labelsBuf)
		if !ok {
			return false
		}

		setAt(&dic, index, unitAt(index^base, node.label, node.leaf))
		if node.leaf {
			setAt(&dic, base, isLeafBit)
		}
		if first != 0 && b.merged[first] {
			link[first] = base
		}

		for e := first; e != 0; e = b.nodes[e].next {
			if !dfs(e, base^uint32(b.nodes[e].label)) {
				return false
			}
		}

		processedNodes++
		if progress != nil && processedNodes%progressEvery == 0 {
			progress(int(processedNodes), 0)
		}
		return true
	}

	if !dfs(b.root, 0) {
		return nil, errDAWGBuild
	}

	if progress != nil {
		progress(int(processedNodes), 0)
	}

	guide := b.buildGuide(dic)
	if guide == nil {
		return nil, errDAWGBuild
	}
	return NewDAWG(dic, guide), nil
}
```

- [ ] **Step 3: Delete the now-dead code from `dawgbuild.go`**

Delete these declarations entirely (they have no remaining callers once
Step 2 lands): `FreeList`, `rangeEntry`, `newFreeList`,
`(*FreeList).markUsed`, `(*FreeList).findFirstFreeOnwards`, `setBit`,
`testBit`, `findBaseWithFreeList`, `findBaseBitset`, `findBaseBoolRandom`,
`findBaseBoolBlock`, `findBaseBool`, `findBase`, `ubit`, `(*ubit).set`,
`(*ubit).test`.

Run `grep -n "FreeList\|findBase\|ubit\|setBit\|testBit" pkg/morphology/internal/dawgbuild.go`
afterward — it must return no matches other than the `slotAllocator`
usage added in Step 2 (`slotAllocator` itself lives in the new file, so a
clean grep here means the sweep is complete).

- [ ] **Step 4: Delete `zz_findbase_test.go`**

```bash
rm pkg/morphology/internal/zz_findbase_test.go
```

Its only subject (`findBase`/`ubit`) no longer exists after Step 3.

- [ ] **Step 5: Build and run the full existing test suite**

Run: `go build ./... && go vet ./... && go test ./pkg/morphology/... -race -v 2>&1 | tail -80`
Expected: PASS, identical test set to Step 1's baseline minus the deleted
`TestDebugFindBase`. Every test that checked `Contains`/`Find`/
`SimilarItems`/roundtrip behavior must produce the same results as before
— this proves the swap is behavior-preserving, only faster.

- [ ] **Step 6: Commit**

```bash
git add pkg/morphology/internal/dawgbuild.go
git rm pkg/morphology/internal/zz_findbase_test.go
git commit -m "refactor: wire free-list allocator into compileImpl, drop dead findBase variants"
```

---

### Task 3: Synthetic scaling benchmark

**Files:**
- Create: `pkg/morphology/internal/dawgbuild_scaling_test.go`

**Interfaces:**
- Consumes: `BuildDAWGWithValues` (existing public API, unchanged
  signature: `func BuildDAWGWithValues(keys []string, values []uint32) (*DAWG, error)`).
- Produces: nothing consumed by later tasks — this is a standalone
  verification gate, run manually (`-run TestDAWGBuildScales`), not part
  of the default `go test ./...` fast path.

- [ ] **Step 1: Write the scaling test**

```go
package internal

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// TestDAWGBuildScales проверяет, что время сборки растёт линейно (не
// квадратично) с числом ключей — то самое свойство, ради которого
// заменён placement-алгоритм. Не входит в быстрый прогон CI: запускается
// явно (`go test -run TestDAWGBuildScales -v -timeout 30m ./...`), т.к.
// на верхней границе (5М) занимает не одну секунду даже при линейном
// масштабировании.
func TestDAWGBuildScales(t *testing.T) {
	if testing.Short() {
		t.Skip("synthetic scaling benchmark skipped in -short mode")
	}

	sizes := []int{100_000, 500_000, 1_000_000, 2_000_000, 5_000_000}
	var perKey []float64

	for _, n := range sizes {
		keys, values := randomWordformKeys(n, 12345)

		start := time.Now()
		d, err := BuildDAWGWithValues(keys, values)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("N=%d: BuildDAWGWithValues: %v", n, err)
		}
		if d == nil {
			t.Fatalf("N=%d: nil DAWG", n)
		}

		perN := elapsed.Seconds() / float64(n)
		perKey = append(perKey, perN)
		t.Logf("N=%9d  elapsed=%-12s  per-key=%.3fµs", n, elapsed, perN*1e6)
	}

	// Квадратичный рост дал бы кратный (не константный) рост per-key при
	// удвоении N; допускаем 3x запас на GC/аллокационный шум, но не
	// больше — O(n^2) на этом диапазоне размеров дал бы на порядки хуже.
	for i := 1; i < len(perKey); i++ {
		ratio := perKey[i] / perKey[0]
		if ratio > 3.0 {
			t.Errorf("per-key cost grew %.1fx from N=%d to N=%d (%.3fµs -> %.3fµs); "+
				"expected near-flat scaling, not quadratic blowup",
				ratio, sizes[0], sizes[i], perKey[0]*1e6, perKey[i]*1e6)
		}
	}
}

// randomWordformKeys генерирует N детерминированных (seed) ключей вида
// "word\x01payload" — с длиной и алфавитом, похожими на реальные
// словоформы (кириллица 4-12 символов) + 4-байтовый payload, как в
// BuildDAWGWithValues.
func randomWordformKeys(n int, seed int64) ([]string, []uint32) {
	rnd := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic test data
	alphabet := []rune("абвгдежзийклмнопрстуфхцчшщъыьэюя")

	keys := make([]string, n)
	values := make([]uint32, n)
	seen := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		var w []rune
		length := 4 + rnd.Intn(9) // 4..12
		for j := 0; j < length; j++ {
			w = append(w, alphabet[rnd.Intn(len(alphabet))])
		}
		key := fmt.Sprintf("%s-%d", string(w), i) // suffix guarantees uniqueness
		for seen[key] {
			key = key + "x"
		}
		seen[key] = true
		keys[i] = key
		values[i] = uint32(i)
	}
	return keys, values
}
```

- [ ] **Step 2: Run it and record the result**

Run: `go test ./pkg/morphology/internal/... -run TestDAWGBuildScales -v -timeout 30m`
Expected: PASS, with the per-N log lines showing per-key cost staying
roughly flat (not growing >3x from N=100K to N=5M). Record the actual
numbers in the task notes / PR description — this is the evidence the
fix works, without needing the real 24h dataset.

- [ ] **Step 3: Commit**

```bash
git add pkg/morphology/internal/dawgbuild_scaling_test.go
git commit -m "test: add synthetic scaling benchmark for DAWG construction"
```

---

### Task 4: Real end-to-end verification against OpenCorpora

**Files:** none (verification only, no source changes).

**Interfaces:** Consumes `gomorphy_build compile` (existing CLI,
`cmd/gomorphy_build/main.go`, unchanged) and the local
`.data/opencorpora/dict.xml` fixture already present in this checkout.

- [ ] **Step 1: Run the real compile and time it**

```bash
go build -o /tmp/gomorphy_build ./cmd/gomorphy_build
time /tmp/gomorphy_build compile -o /tmp/opencorpora-new.dat
```

Expected: completes in minutes, not ~24h. The command prints periodic
`[build]`/`[compile]` progress lines (existing `compileAndSave` progress
printer in `cmd/gomorphy_build/main.go`) — watch the `keys/s`/`nodes/s`
rate stay roughly constant rather than collapsing as the run progresses,
which is the real-data confirmation of Task 3's synthetic result.

- [ ] **Step 2: Spot-check correctness against the old `.dat`**

```bash
go run ./cmd/gomorphy -dict /tmp/opencorpora-new.dat lookup кота
go run ./cmd/gomorphy -dict .data/opencorpora/opencorpora.dict lookup кота
```

Expected: identical parse results between the freshly-built dictionary
and the pre-existing `.dat` in `.data/opencorpora/opencorpora.dict` (built
by the old algorithm) for several sample words, confirming the placement
algorithm swap didn't change lookup semantics.

- [ ] **Step 3: Record the wall-clock time**

No commit for this task (verification only) — report the measured
before/after build time in the PR description or a follow-up message.

---

## Self-Review Notes

- **Spec coverage:** `slotAllocator` (spec §Design) → Task 1.
  `compileImpl` integration + dead-code removal (spec §Removed) → Task 2.
  Synthetic benchmark (spec §Testing #3) → Task 3. Existing-test regression
  (spec §Testing #2) → Task 2 Step 5. Real end-to-end run (spec §Testing
  #4) → Task 4. All spec sections have a task.
- **Type consistency:** `alloc` returns `(uint32, bool)` consistently in
  Task 1's implementation and Task 2's call site (`base, ok := alloc.alloc(...)`).
  `newSlotAllocator()` takes no arguments in both places (initial capacity
  is implicit — starts at 1, grows on demand — matching the spec's "no
  arbitrary cap for large ones, no mandatory pre-sizing for small ones").
- **No placeholders:** every step has runnable code or an exact command.
