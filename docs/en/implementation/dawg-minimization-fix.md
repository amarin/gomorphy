# Fix: DAWG minimization was not working (bug in chainSig) — DONE

> Moved from `docs/todo.md` during the documentation cleanup (2026-09-14).

## Problem

While preparing Stage 17 (compressing "cold" sections), measuring the
sections of the real `.dat` showed that `words.dawg` was 99.33% of the
file (425.8 MB out of 428.7 MB), and the double-array layout occupied
70,968,276 slots for 3,065,312 trie nodes (density ~4.3%). The cause
turned out to run deeper than packing density: `chainSig`
(`dawgbuild.go`) encoded the id of the node **itself** into the
sibling-chain signature instead of the id of its **child** (the
comment above the function literally said "the child is compared by
id", but the code compared something else). A node's id is always
fresh/unique, so the signatures of two structurally identical suffix
chains never matched — `register` found zero matches at all (0 hits
out of 35,484,001 on the full dictionary), suffix minimization did not
work, and the DAWG was effectively built as an unminimized trie.

## Solution

A one-line fix: encode `b.nodes[n].first` (the child's id) into the
signature instead of `n`. The regression test
`TestBuildDAWGMinimizesSharedSuffixes` (`dawgbuild_test.go`) builds a
DAWG from keys with a common long suffix and checks that the
double-array stays an order of magnitude smaller than the
"unminimized" estimate — without the fix the test fails (18,176 slots
instead of the expected <3,150).

## Result (full `dict.xml`, `gomorphy_build compile`)

| Metric | Before the fix | After the fix |
|---|---|---|
| `.dat` size (OpenCorpora) | 428.7 MB | 14.6 MB (**~29×**) |
| Double-array slots | 70,968,276 | ~2M |
| Packing density | ~4.3% | ~51% |

`lookup`/`fuzzy` results are identical before and after the fix
(verified both manually and with a full `go test ./... -race` run,
125/125 green). This also closes most of the original Stage 17 goal
(`.dat` size) without zstd and without changing the format — see
`docs/en/todo.md`.
