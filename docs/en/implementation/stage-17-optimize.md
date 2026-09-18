# Stage 17. Narrowing ID types and zstd — PARTIALLY DONE

> Status as of 2026-09-14: narrowing the types and the format
> groundwork for compression are done; the actual zstd implementation
> is deferred to the post-1.0.0 backlog — see `docs/en/todo.md`.

## Stage contents

Reducing file size via:
1. Narrowing types: uint16 for paradigm/suffix/tag/prefix IDs.
2. zstd-compressing the cold sections (suffixes, prefixes, tagset, paradigms).

### Narrowing types

Current limits (measured on OpenCorpora):
- Paradigms: ~3K -> uint16 (65K max)
- Suffixes: ~5K -> uint16 (65K max)
- Prefixes: ~3 -> uint16
- Tags: ~1K -> uint16 (65K max)

A paradigm's flat array: `[]uint16` instead of `[]uint32`.
Savings: ~40% on the paradigms section.

### zstd compression

Cold sections (held in memory once loaded, decompressed once):
- `suffixes` — strings, compress well
- `prefixes` — strings
- `tagset` — JSON
- `paradigms` — a uint16 array (partially deduplicated already)

Hot sections (used on every lookup):
- `words.dawg` — **not compressed** (zero-copy mmap)

A compression flag in the section catalog: `flags & flag_compressed != 0`.
Decompression on load: `zstd.NewReader` -> bytes.Buffer.

### Expected effect

| Section | Before compression | After compression |
|---|---|---|
| suffixes | ~0.5 MB | ~0.3 MB |
| prefixes | ~0.1 MB | ~0.05 MB |
| tagset | ~0.1 MB | ~0.05 MB |
| paradigms | ~3-4 MB | ~1-2 MB |
| **Total** | ~4-5 MB | ~2-3 MB |

Savings: ~2 MB. Not critical for pymorphy2 (~15 MB), but significant
for OpenCorpora (~305 MB -> ~20-25 MB).

## Verification (tests)

- Unit test: roundtrip compressed -> uncompressed -> identical data.
- Unit test: a compressed section is smaller than the uncompressed one
  (assert size < threshold).
- Unit test: roundtrip uncompressed -> identical data (backward compat).
- Benchmark: Parse before and after (expecting 0% regression on words.dawg).
- `go test ./pkg/morphology/... -race` — green.

## Manual verification

- `make compile`: compare the `.dat` size before and after.
- `gomorphy -dict pymorphy2.dat lookup кота` -> identical.
- Loading: `time gomorphy -dict pymorphy2.dat lookup кота` -> no regression.

## Implementation (partial, as of 2026-09-14)

### Narrowing ID types — DONE

Implemented along the way in stages 11-18, no separate work was needed:
`Paradigm` stores its data as `[]uint16` (`paradigm.go`), `TagSet.Index`
is `map[string]uint16` (`tagset.go`).

### zstd compression — implementation deferred, but reassessed

After the DAWG minimization fix (see
[dawg-minimization-fix.md](dawg-minimization-fix.md)), the `.dat` for
the full OpenCorpora dictionary is 14.6 MB instead of the expected ~305
MB, and of that, the "cold" sections (suffixes, tagset, paradigms) are
~2.86 MB (~19%, not the low single-digit percent it looked like before
the fix). The savings from compression are still significant, but the
implementation itself (picking a library/level, roundtrip tests,
measuring the `Parse` regression) has been split off into a separate
future task that doesn't block 1.0.0 — see `docs/en/todo.md`.

### Format groundwork for extensible compression — DONE

Before this change, a section's flag was a single "compressed/not
compressed" bit with no algorithm identifier — extending it after
release would have meant either guessing the algorithm from the
section's data signature (unreliable: a section is an arbitrary blob
with no self-describing header), or bolting on a second flag after the
fact on top of already-released files. Compression had never been
written to a single file yet, so this was the one moment when the
catalog's byte layout could be redesigned with no versioning and no
compatibility risk for users.

Done (`pkg/morphology/internal/format.go`):
- The catalog entry flags (`Entry.Flags`/`Section.Flags`, the same
  byte, the same `entrySize`) were redefined: the low 4 bits are the
  section's explicit compression algorithm id (`CompressionNone=0`,
  `CompressionZstd=1`, more as new algorithms are implemented), the
  high 4 bits are reserved for future-version flags independent of compression.
- `Container.Section`, on reading an unknown/unimplemented algorithm,
  returns `ErrUnsupportedCompression` with the algorithm id in the
  message — an explicit upgrade error instead of data corruption or a
  silent ignore.
- `validateSections`, on writing, rejects reserved bits and unknown
  algorithm ids — extending it for a new algorithm is a one-line change.
- Compression is chosen **per section**, not for the whole file:
  `words.dawg` is always `CompressionNone` (aliased from mmap with no
  copying), the choice of algorithm for the other sections is left to
  the future implementation.

### Remaining implementation (a separate future task, after 1.0.0)

- Library: `klauspost/compress` (pure Go, no cgo).
- zstd compression/decompression for the suffixes, prefixes, tagset,
  and paradigms sections; level — maximum compression (the file is
  built rarely and read often, and cold-section decompression happens
  once on load).
- ~~Also — an analysis of the double-array DAWG's packing density~~ — DONE,
  see the next section.

### DAWG packing density analysis — DONE, a backlog candidate exists

The full breakdown with experiments and numbers:
[docs/research/0001-dawg-alphabet-density.md](../research/0001-dawg-alphabet-density.md).
Short version:

- Permuting/reordering the byte label alphabet (by frequency or
  otherwise) — **has no effect**: the free-list allocator is
  insensitive to a label's numeric value, only to the graph's topology.
  This branch doesn't need to be pursued further.
- Narrowing the alphabet's domain — switching from per-byte UTF-8 (2
  bytes per Cyrillic letter) to "1 byte = 1 character" — has a real
  effect: on a test set (3,065,312 OpenCorpora wordforms, **without** a
  payload), ~36.5% reduction in the serialized array's size and ~29.2%
  reduction in the number of nodes in the minimized automaton.

**Next steps (backlog, doesn't block 1.0.0), in priority order:**

1. **Re-verify on realistic data** — the experiment only measured words
   with no payload; the real `words.dawg` is `word +
   PayloadSeparator + base64(value)`, where the payload is less regular
   and could reduce the win. A cheap check (reusing the methodology
   from the research document) is needed BEFORE deciding whether to
   invest in the implementation.
2. **Assess whether it's worth it at all** — the current full OpenCorpora
   `.dat` is already 14.6 MB (after the minimization fix), and
   `words.dawg` is the uncompressible (zero-copy mmap) hot section that
   this optimization specifically targets (unlike the zstd plan above,
   which only touches cold sections). If step 1 confirms an effect
   around 30%+ on real data, this is the largest remaining lever for
   shrinking the `.dat`; if the effect turns out small on payload data,
   don't do it at all — the complexity wouldn't pay off.
3. **If the decision is to proceed**: an alphabet codec (character ->
   dense byte code, code `0` strictly reserved as a sentinel — see the
   finding in the research document about corrupting the guide
   traversal), with an explicit version/algorithm marker on disk
   (similar to `CompressionNone`/`CompressionZstd` in `format.go`, see
   above) — **mandatory**, because `ReadDAWG`/`ParseDAWG`
   (`pkg/morphology/internal/dawg.go`) must keep reading pymorphy2's
   original `words.dawg` (raw UTF-8) untouched; the new alphabet only
   applies to dictionaries gomorphy builds itself
   (`BuildDAWG`/`BuildDAWGWithValues` for OpenCorpora/a future UniMorph import).
4. Tests: roundtrip (build with the codec -> save -> open -> lookup
   identical to a direct build), a regression test for code `0`, a
   `Parse` benchmark before/after on the full dictionary.

Separately from the list above: the technique itself ("which alphabet
to pick") is no longer an open question — only whether it applies to
real payload data (step 1) and whether the complexity is justified
(step 2) remain open.

### `tagset` encoding analysis — DONE, a backlog candidate exists

Full breakdown: [docs/research/0002-paradigm-tagset-binary-encoding.md](../research/0002-paradigm-tagset-binary-encoding.md).
Short version: the `tagset` section (unique grammeme combinations,
currently JSON) shrinks by **78.2%** (223,692 -> 48,849 bytes, ~1.09%
of the whole file's size) when switching to encoding it as "a
dictionary of ~100 individual grammemes + an index list per
combination", with no change to the hot read path (`TagSet.Tags` is
already materialized into a `[]string` once on `Open()`, `TagName` is
O(1)). Paradigms are already binary (`uint16` codes), there's nothing
to do there — that part of the original hypothesis about paradigms
doesn't need to be pursued.

A backlog candidate, lower priority than the DAWG's dense alphabet (an
order of magnitude smaller effect, but a simpler implementation with no
open questions about applicability to payload data): a new `tagset`
section format + an encoding version marker (similar to
`CompressionNone`/`CompressionZstd`), if rolled out after 1.0.0 — before
release, no versioning is needed at all (there are no `.dat` files in
production yet).

### Along the way: the info section

A separate (not part of Stage 17's original plan) optional `info`
section with build diagnostics metadata — see
[info-section.md](info-section.md).
