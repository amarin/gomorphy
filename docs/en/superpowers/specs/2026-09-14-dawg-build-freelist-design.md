# DAWG double-array construction: free-list placement

## Problem

`BuildDAWG`/`BuildDAWGWithValues` (`pkg/morphology/internal/dawgbuild.go`) build
a minimized DAWG in two phases: (1) incremental list-form construction with
sibling-chain minimization (mirrors dawgdic's register-based minimization),
then (2) depth-first packing of that list-form trie into a double-array
(`dictionary uint32[]` + `guide byte[]`, the dawgdic/pymorphy2 wire format).

Phase 2 (`compileImpl`) is the bottleneck. For every node it calls
`findBaseBitset`, which searches for a free `base` slot by scanning
sequentially from `base := 1` over a bitset (`used []uint64`,
pre-sized to `max(totalNodes*32, 1<<30)` bits) until it finds a slot where
`base` itself and every `base^label` child slot are free. This scan touches
already-used slots just as often as free ones, so the average scan length
grows with how full the array already is. Total construction time trends
toward O(n²) in the number of trie nodes. This matches the observed
behavior: dictionaries at pymorphy2 scale build quickly, but the ~1.5M
wordform OpenCorpora import (many more DAWG *keys* than that once
`word + PayloadSeparator + payload` keys are counted) takes ~24h.

The file also carries a written but never-wired `FreeList` type
(`findBaseWithFreeList`) plus four other unused `findBase*` variants
(`findBase`, `findBaseBool`, `findBaseBoolBlock`, `findBaseBoolRandom`) —
abandoned experiments toward the same fix, none of them reaching the actual
call site in `compileImpl`.

## Goal

Replace the linear bitset scan with the free-list placement technique used
by dawgdic, Darts, and cedar (Aoe 1989) for exactly this problem, so
construction time scales near-linearly in the number of nodes instead of
quadratically. No changes to:
- the on-disk DAWG format (`dictionary uint32[]` + `guide byte[]`, read by
  `dawg.go`'s `FollowByte`/`Find`/`ForEachChild`/etc.),
- the public API (`BuildDAWG`, `BuildDAWGWithValues`,
  `BuildDAWGWithValuesProgress`),
- the phase-1 list-form minimization algorithm.

This is a pure internal-algorithm swap inside `compileImpl` and its helpers.

## Design

### `slotAllocator` (new, `pkg/morphology/internal/dawgbuild_freelist.go`)

Owns double-array slot placement during the build. Replaces `used []uint64`
+ `findBaseBitset` (and all five other unused `findBase*` variants, which are
deleted).

State:
- `free []bool` (or bitset) — O(1) point membership test: "is slot X taken?"
  Needed only for validating individual candidate/child slots, not for
  searching.
- `listNext`, `listPrev []int32` — doubly-linked list threaded through slots
  that are currently free. Search for a candidate `base` walks this list, so
  it only ever visits free slots — it never re-touches slots already taken,
  which is the actual fix (the old code's cost came from re-scanning taken
  slots every time).
- `hint [256]uint32` — last successful `base` found for a given first-label
  byte. A new search starts at `hint[labels[0]]` (falling back to the free
  list head if that slot is no longer usable) rather than at the list head
  every time. Standard companion heuristic used alongside the free list in
  dawgdic/cedar.

Operations:
- `alloc(index uint32, labels []byte) (base uint32, ok bool)` — find a free
  `base` such that `encodable(index^base)` and every `base^label` for
  `labels` is free; mark `base` and all child slots used (unlink from free
  list, set membership bit); update `hint[labels[0]]`.
- `markUsed(slot uint32)` — for slots consumed without going through
  `alloc`'s child-marking path.
- `grow(newSize uint32)` — extend all backing arrays and append the new
  slots to the tail of the free list. Doubles capacity, matching the
  existing `setAt` growth strategy. Replaces the current fixed
  `max(totalNodes*32, 1<<30)` pre-sizing — no more mandatory 128MB+
  bitset for small dictionaries, and no arbitrary cap for large ones.

`compileImpl`'s `dfs` closure changes only at the two call sites that
currently call `findBaseBitset` / touch `used` directly: they call
`allocator.alloc(...)` / `allocator.markUsed(...)` instead. The DFS
structure, the `link` map (merged-node base reuse), and `unitAt`/`encodable`
are unchanged.

### Removed

- `used []uint64` bitset in `compileImpl`, `estimatedSize` pre-sizing.
- `FreeList`, `rangeEntry`, `newFreeList`, `findBaseWithFreeList`.
- `findBaseBitset`, `findBaseBoolRandom`, `findBaseBoolBlock`, `findBaseBool`,
  `findBase`, `ubit` (all dead or superseded by `slotAllocator`).
- `setBit`/`testBit` free functions if nothing else uses them after the
  above removals (fold into `slotAllocator` if still needed internally).

## Testing

1. **Unit tests for `slotAllocator` in isolation**
   (`dawgbuild_freelist_test.go`): alloc/free correctness on small hand-built
   cases, growth across the doubling boundary, hint reuse, collision
   handling (candidate rejected because a child slot is taken → search
   continues). No DAWG-level concepts here — just the allocator contract.

2. **Existing correctness regression, unchanged inputs/expectations**:
   `dawgbuild_test.go`, `dawg_test.go`, `zz_findbase_test.go`,
   `zz_debug_follow_test.go`, `zz_debug_stress_test.go`, `zz_debug_test.go`,
   the `testdawg` reference package, and the pymorphy2-dicts-ru integration
   test (stage 12) all must still pass unchanged — output is the same DAWG,
   just built faster.

3. **Synthetic scaling benchmark** (new): generate keys at
   N ∈ {100K, 500K, 1M, 2M, 5M} (random fixed-alphabet strings of realistic
   wordform length, e.g. 4–12 bytes) and measure wall-clock build time.
   Assert time-per-key does not grow with N (allow some slack for GC/alloc
   noise, but the shape must be flat/linear, not quadratic). This is the
   gate before touching real data — no need to run the real 24h compile to
   see whether the fix works.

4. **Real end-to-end run, once, after (3) passes**: `gomorphy_build compile`
   against the actual OpenCorpora `dict.xml`. Confirm wall-clock drops from
   ~24h to a small number, `Parse`/`Lookup`/`Fuzzy` results are unchanged
   (existing property/integration tests from stage 15 cover this), and the
   produced `.dat` file opens and behaves identically to a file built with
   the old algorithm on a small fixture.

## Risks

- Double-array placement is correctness-critical, bit-level code. Mitigated
  by: reusing a well-published, widely-implemented algorithm (not inventing
  a new one) and by testing `slotAllocator` in isolation before touching
  `compileImpl`.
- Growth/doubling interacting with the linked list needs care (appended
  slots must be correctly spliced into the free list tail, and `hint`
  entries pointing at slots that later become the boundary must still be
  validated for freeness before reuse — `alloc` must never trust a hint
  blindly, only as a search starting point).