# DAWG packing density with a dense alphabet: checked against real payload, and a 2-byte-code trap

**Date:** 2026-09-15
**Status:** the hypothesis (the ~36% effect holds with payload) is confirmed; a separate hypothesis about the 2-byte alphabet being usable as-is is disproven (a bug was found and fixed), then re-measured after the fix and confirmed too (the effect is small, but real and correct)
**Verified by:** Aleksey Marin (asmadews@gmail.com)
**Related documents:** [0001-dawg-alphabet-density.md](0001-dawg-alphabet-density.md), [docs/superpowers/specs/2026-09-15-dawg-alphabet-harness-design.md](../superpowers/specs/2026-09-15-dawg-alphabet-harness-design.md), [docs/superpowers/plans/2026-09-15-dawg-alphabet-harness.md](../superpowers/plans/2026-09-15-dawg-alphabet-harness.md)

## Context

[0001-dawg-alphabet-density.md](0001-dawg-alphabet-density.md) measured the effect of switching `words.dawg` from byte-wise UTF-8 (2 bytes/Cyrillic character) to a dense "1 byte = 1 character" alphabet: ~36.5% reduction in the serialized `dict.dic` array, ~29.2% reduction in the minimized automaton's node count. That measurement was done on 3,065,312 real OpenCorpora wordforms **without** the payload suffix that real `words.dawg` keys actually carry (`internal/dawgbuild.go:43-53`: `key + PayloadSeparator + base64(uint32)`).

The dawgdic unit format (`internal/dawg.go:11-20`) reserves exactly bits 0-7 of the `uint32` unit for the edge label — this is a hard structural constraint of the format (compatibility with `ReadDAWG` for pymorphy2's raw `words.dawg`), not a gomorphy design decision. The engine (`FollowByte`/`Follow`, `dawg.go:150-181`) walks the key byte by byte regardless of how many bytes logically represent one character — so an N-byte alphabet is a codec at the key-construction boundary, not an engine change. The free-list allocator (`dawgbuild_freelist.go`) confirms this: `hint[256]uint32` is indexed by a single label byte, and `fits`/`alloc` operate on the node's real `labels []byte`, not on the alphabet's size.

## Hypothesis

**H1:** If the dense 1-byte alphabet's effect is measured on keys with a real payload (not bare words), the size reduction will stay close to ~36.5% — the payload (base64, already 1 byte/character, a fixed length of ~9 bytes per key) shouldn't meaningfully dilute the effect of compressing the variable part (the word).

**H2:** The same mechanism ("one character = N consecutive byte transitions") can be safely generalized to a dense 2-byte alphabet (up to 65536 codes) by simple big-endian packing of the code into 2 bytes, with no engine changes.

## Experiment

**Data:** the same 3,065,312 wordforms as in [0001](0001-dawg-alphabet-density.md) (`grep -o '<f t="[^"]*"' .data/opencorpora/dict.xml | sed -E 's/<f t="//;s/"$//' | sort -u`, saved to `/tmp/wordforms.txt`), a sample of 102,178 words (every 30th, to keep build time manageable).

**Tools:** `pkg/morphology/internal/alphabet.go` (new package code, branch `dawg-alphabet-harness`, commit `c15a972`) — an `Alphabet` interface, `IdentityAlphabet` (raw UTF-8, the baseline), `DenseAlphabet` (a dense alphabet, 1 or 2 bytes wide). The harness — `pkg/morphology/internal/alphabet_compare_integration_test.go` (`//go:build integration`, `TestAlphabetCompare`), calling `BuildDAWG` directly on realistic keys `word_in_alphabet + PayloadSeparator + base64(uint32)` with **random** (not sequential) payload values.

**Measurement method:** `len(dawg.dict)*4 + len(dawg.guide)` — the real serializable size (the dictionary array plus the guide array), not just the allocator's internal capacity. Also build time (`time.Since`) and the number of "live" edges from the root via `ForEachChild` (a check that the guide traversal even works).

**Why random rather than sequential payload values:** earlier in this same session, before any committed code existed, there was a one-off check (not saved) with sequential `uint32` values (0,1,2,…) — the DAWG build didn't finish within 15 minutes even on just 510,000 keys. Hypothesis: sequential big-endian values give alphabetically-adjacent words (which already share prefixes in the trie) nearly identical payload suffixes too, which pathologically loads the free-list allocator. With random values the build runs normally. This anomaly itself wasn't separately diagnosed — the harness simply uses random values to avoid triggering it.

**Reproduction steps:**
```bash
grep -o '<f t="[^"]*"' .data/opencorpora/dict.xml | sed -E 's/<f t="//;s/"$//' | sort -u > /tmp/wordforms.txt
go test -tags=integration ./pkg/morphology/internal/ -run TestAlphabetCompare -v -timeout 900s
```

## Results

### First pass: H1 confirmed, H2 — a methodological error, not just a number

The first run (before the fix, commit `fc3b1a8`) gave:

| Variant | Bytes (`dict` only) | Build time | Reduction |
|---|---|---|---|
| identity (baseline) | 12,190,840 (12.19 MB) | ~121-125 s | 0.0% |
| dense-1 | 7,782,908 (7.78 MB) | ~41-43 s | 36.2% |
| dense-2 | 9,787,376 (9.79 MB) | **0.34 s** | **19.7%** |

H1 was confirmed immediately: 36.2% against an expected ~36.5% — the effect holds with a real payload with virtually no change.

But dense-2 looked suspiciously good and suspiciously fast. A final review of the branch (before merging) found the real cause: the width-2 `DenseAlphabet` packed the code big-endian into 2 bytes (`byte(code>>8), byte(code)`). Codes are assigned starting from 2 (0 and 1 are reserved for the guide sentinel and `PayloadSeparator`), and this corpus has only 46 distinct runes — meaning **every** code is ≤ 255, and the high byte is **always** `0x00`.

Byte `0x00` is the engine's guide-traversal sentinel (`ForEachChild`, `dawg.go:245-258`: `for label != 0 { ... }`). So dense-2 was literally writing a reserved byte into the key body. Consequences:

1. **The guide traversal for dense-2 was completely dead** (verified empirically: 0 child edges from the root via `ForEachChild`). `pkg/morphology/fuzzy.go` is built entirely on `ForEachChild`/`HasPayloadChild` — meaning a dictionary with this encoding would build, but would be unusable for fuzzy search.
2. **The reported numbers were an artifact, not a measurement.** In the free-list allocator, `base ^ label` is XOR; for `label=0` this is the identity (`base^0==base`), so edges labeled 0 are effectively free to place — this explains both the "anomalously fast" build (0.34 s) and the understated size (few real collisions).

So the confusion of "reserve **code** 0/1" ≠ "reserve **byte** 0x00/0x01" is exactly the methodological error found in the first pass.

### The fix and second pass: H2 confirmed (in its corrected form), the effect is small but real

The width-2 encoding was reworked into a positional base-254 scheme: `k := code-2`, `hi := 2 + k/254`, `lo := 2 + k%254` — both bytes are guaranteed to be in `[2,255]`, never 0 or 1. Capacity: `254*254 = 64516` codes (it would have been `65534` with naive big-endian packing — that number was never correctly reachable to begin with).

Re-measurement (the same corpus, commit `c15a972`), now with a guide-traversal check (root children > 0) as part of the harness itself:

| Variant | Bytes (`dict`+`guide`) | Build time | Reduction | Live edges from root |
|---|---|---|---|---|
| identity (baseline) | 18,286,260 (18.29 MB) | 2 m 10.9 s | 0.0% | 11 |
| dense-1 | 11,674,362 (11.67 MB) | 44.4 s | **36.2%** | 40 |
| dense-2 | 17,842,908 (17.84 MB) | 1 m 37.4 s | **2.4%** | 1 |

(The absolute numbers grew compared to the first pass not because the dictionary got bigger — `bytes` now also counts the `guide` array, not just `dict`. Comparisons within a single pass remain valid.)

dense-1 didn't change (the bug only affected width 2) — 36.2%, confirming H1 again.

dense-2 after the fix: the build became two orders of magnitude slower (0.34 s -> 97.4 s — edges labeled 0 are no longer "free"), and the size reduction dropped from a fictitious 19.7% to an honest **2.4%**. This is expectedly small: the corpus only has 46 distinct runes, which under the positional 2-byte scheme means a **constant** high byte (`k/254` is always 0, since 46≪254) — meaning dense-2 on this corpus structurally degenerates into "dense-1 plus a constant prefix byte per character," which is exactly why there's only one "live" edge from the root (byte `2`) instead of the expected variety. This isn't a residual bug: sampled `Contains` checks (the full path through both bytes) all pass.

## Conclusions

1. **H1 is confirmed without reservation.** The dense 1-byte alphabet gives ~36% reduction in `words.dawg` with a real payload key too — the [0001](0001-dawg-alphabet-density.md) methodology (bare words) didn't overstate the effect.
2. **H2 in its original form was wrong: "reserve a code" ≠ "reserve a byte."** Naive big-endian packing of an N-bit code into several bytes can smuggle in a reserved byte value (here, 0x00, the guide sentinel) even if the numeric code itself was never 0 or 1. Any future multi-byte packing in this format needs a positional scheme where **every individual byte** stays outside the reserved range — not just the final number.
3. **This specific bug wouldn't have been caught by a round-trip test** (`Encode`->`Decode` matches the original string) and wasn't caught by two independent task reviews during implementation — only by the final review of the whole branch, which explicitly checked the structural property. A round-trip check at the code level doesn't inspect the byte stream the engine actually sees; the test added as a result of the fix (`TestDenseAlphabetEncodedBytesNeverReserved`) checks exactly that, and would have failed on the old implementation.
4. **On the OpenCorpora corpus (46 runes), a 2-byte dense alphabet has no practical value** — 1 byte already covers the alphabet with huge headroom (254 capacity against 46 used), and an honest 2-byte variant loses to the 1-byte one both in size (2.4% vs. 36.2%) and in build speed. Width 2 remains architecturally valid (fully functional, guide traversal confirmed) — but its practical value would only appear for an alphabet that doesn't fit in 254 characters (a hypothetical scenario — a multilingual dictionary via a future UniMorph Stage 16, if and when its turn comes).
5. **The implementation is harness-only, with no production wiring.** Nothing in `import.go`/`parse.go`/`open.go`/`save.go`/the `.dat` format was changed; `Alphabet`/`DenseAlphabet` is reusable, tested code that isn't wired in yet. The decision to roll the 1-byte alphabet into production (a build flag, a version marker in `.dat`, reading it in `Open()`) is a separate future step, not made within this session.
6. **The DAWG-build anomaly with sequential payload values** (mentioned in "Experiment" above) remains undiagnosed — all that's known is that it exists and that random values avoid it. A separate task, if a practical need arises.

## Sources

<No third-party sources were used; every reference in the text is to files and documents in this repository, verified by reading and running code (including an empirical check via `go -overlay` with no mutation of the working tree, during the final branch review) in this session>
