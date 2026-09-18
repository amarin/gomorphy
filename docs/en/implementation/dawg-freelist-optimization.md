# Speeding up DAWG build: free-list instead of O(n²) scanning — DONE

> Moved from `docs/todo.md` during the documentation cleanup (2026-09-14).

## Problem

After Stage 15, building the real OpenCorpora dictionary (dict.xml,
3,065,312 lemmas) took about 24 hours. The cause: `compileImpl`
(`pkg/morphology/internal/dawgbuild.go`) looked for a free `base` slot
in the double-array layout by linear bit-by-bit scanning from `base=1`
for every node; as the array filled up, the per-node search cost grew
together with the number of already-placed nodes, giving quadratic
asymptotic behavior.

## Solution

The placement algorithm was replaced with an intrusive doubly-linked
free list (the dawgdic/cedar/Darts technique, Aoe 1989): free slots are
linked into a list, the search only visits free slots, plus a hint
cache keyed by the label's first byte. The file format and the public
API did not change. Details are in the
[design spec](../superpowers/specs/2026-09-14-dawg-build-freelist-design.md)
and the [implementation plan](../superpowers/plans/2026-09-14-dawg-build-freelist.md).

## Result (full `dict.xml`, `gomorphy_build compile`)

| Metric | Before the fix | After the fix |
|---|---|---|
| Build time (OpenCorpora, 3.06M lemmas) | ~24 h | ~24 s |
| Peak RSS | — | ~8.2 GB |

A synthetic benchmark (`dawgbuild_scaling_test.go`, `-tags scaling`)
confirms sub-quadratic (not O(n²)) growth for 100K–5M keys.

It later turned out that most of the final `.dat` size was caused by a
separate DAWG minimization bug unrelated to this fix — see
[dawg-minimization-fix.md](dawg-minimization-fix.md).
