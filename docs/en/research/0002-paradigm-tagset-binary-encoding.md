# Binary encoding of paradigms and the tag-combination dictionary instead of text

**Date:** 2026-09-15
**Status:** the hypothesis is disproven for paradigms (already implemented); for
the tag-combination dictionary — **confirmed, with a significant effect**
(corrected mid-review at the user's request — see "Correction" below)
**Verified by:** Aleksey Marin (asmadews@gmail.com)
**Related documents:** [0001-dawg-alphabet-density.md](0001-dawg-alphabet-density.md), [implementation/stage-17-optimize.md](../implementation/stage-17-optimize.md)

## Context

The compiled GMOR dictionary (`pkg/morphology/internal/format.go`,
`pkg/morphology/save.go`) is a container of named sections. On the real,
full OpenCorpora dictionary (`.data/opencorpora/opencorpora.dat`, rebuilt
2026-09-15 after this session's tag/deduplication fixes, 2 shards,
15.3 MB), the sections break down as:

| Section | Size | Share of file |
|---|---|---|
| `words.dawg-0` + `words.dawg-1` | 13,711,276 bytes | ~85.2% |
| `suffixes-0` + `suffixes-1` | 1,457,339 bytes | ~9.1% |
| `paradigms-0` + `paradigms-1` | 694,980 bytes | ~4.3% |
| `tagset` | 223,692 bytes | ~1.4% |

Paradigm encoding — `pkg/morphology/internal/paradigm.go` (`Paradigm`,
`NewParadigm`) and `format.go` (`EncodeParadigms`, `format.go:468-483`):
each paradigm is a flat `[]uint16` of the form `[suffix_0..N | tag_0..N |
prefix_0..N]` (numeric ids, not text), serialized as `[count
uint32][for each: len uint32, len×uint16]`.

`tagset` encoding — `EncodeTagSet`/`DecodeTagSet` (`format.go:439-
459`): `json.Marshal({name, tags: []string})`. It's important to be
precise here: "tags" is **`TagSet.Tags`, an array of unique grammeme
COMBINATIONS** (full morphological tags like
`"NOUN,anim,masc,sing,nomn"`, registered via `tagSet.Add(frm.gramm)`
during import), not an array of individual grammemes ("NOUN", "anim",
etc. one at a time). Each such combination is itself already a string
with internal comma separators between the grammemes it contains.

## Correction (the user corrected the first version of this document)

The first version of this document called the `tagset` section's
contents a "list of unique tags" and compared only two ways of packing
**the same** flat text (JSON vs. the compact `EncodeStrings`) — the
effect turned out small (5.4% of the section). The user pointed out two
inaccuracies:

1. Terminological: this isn't "unique tags" but **unique tag
   combinations** — each element of `TagSet.Tags` itself consists of
   several grammemes joined by commas.
2. Substantive: the real redundancy isn't in JSON's syntax (the array's
   quotes/commas) but in the fact that **the same individual grammemes
   repeat across many different combinations** (e.g.
   "NOUN,anim,masc,sing,nomn" and "NOUN,anim,masc,sing,gent" share 4 of
   5 grammemes) — and the first version of the experiment didn't check
   this redundancy at all, comparing only the wrapper around the same
   uncompressed text.

The "Experiment" section below is expanded with a second, deeper run
that directly checks this redundancy.

## Hypothesis

While looking at `.data/opencorpora/opencorpora.dat` in a text/hex
viewer, the user saw readable text at offset `0x1ee` (494) and took it
for a text-form list of paradigms. Hypothesis: if a dictionary of
individual grammemes (grammeme -> numeric code) were built, and both
paradigms and tag combinations were stored as binary `[length,
code, code, ...]` structures instead of text — the final dictionary's
size should shrink noticeably, and this block's read/write speed
should improve.

## Experiment

**Data:** the actually-built `.data/opencorpora/opencorpora.dat` (the
same file the user was viewing), 2 shards, 6028 unique tag combinations
after this session's `paradigmKeyHash`/deduplication fix (built via
`./deploy/gomorphy_build compile`).

**Tools:** a temporary `_test.go` in `pkg/morphology/internal` (the
file isn't kept in the repository), calling the package's existing
exported functions (`OpenContainer`, `Container.Section`,
`DecodeTagSet`, `EncodeStrings`) — not an invented tool.

**Measurement method (two runs):**

1. *First run* (see "Correction" above): compare the `tagset`
   section's size under the current encoding (`EncodeTagSet`, JSON)
   against the same `EncodeStrings` encoding already used for
   `suffixes`/`prefixes` — but **without** splitting combinations into
   individual grammemes.
2. *Second run* (following the discussion with the user): split each
   of the 6028 combinations by comma into individual grammemes, build a
   dictionary of unique grammemes, and compute the size of the
   hypothetical encoding "grammeme dictionary (`EncodeStrings`) + per
   combination: 1 length byte + N index bytes into the dictionary" —
   i.e. exactly the binary `[length, code, code, ...]` the user
   proposed, but applied to the tag-combination dictionary rather than
   to paradigms.

Additionally (to estimate the read-time cost of such an encoding), the
`TagSet.TagName` code and `pkg/morphology/open.go` were read, to
establish at what point in the dictionary's lifecycle `tagset` gets
decoded and how "hot" that path is.

**Reproduction steps:**
```bash
./deploy/gomorphy_build compile
# then — a temporary _test.go in pkg/morphology/internal:
# OpenContainer(data).Section("tagset") -> DecodeTagSet -> ts.Tags;
# for each element of ts.Tags: strings.Split(combo, ",") -> grammemes;
# build a dictionary of unique grammemes, compute the bytes for the
# hypothetical encoding, and compare against
# len(EncodeTagSet(ts)) / len(EncodeStrings(ts.Tags))
```

## Results

**What's actually at offset `0x1ee`.** The section catalog shows:
`tagset` occupies the range `0x1d0`–`0x36ba0` (464–224,156 bytes, which
matches the user-cited `0x036b9a` as the section's end); `paradigms-0`
starts only at `0x19a860` (1,681,504) — a megabyte and a half later.
The bytes at `0x1ee` are `"NOUN,anim,masc,sing,nomn",
"NOUN,anim,masc,sing,gent","NOUN,anim...` — a JSON array of tag
combinations (`tagset`), not paradigms.

**Paradigms.** `Paradigm.Data()` and `EncodeParadigms` confirm: this is
already a flat array of `uint16` codes, without a single byte of text —
the user's hypothesis for this section is **already implemented**, no
further action needed.

**Tag-combination dictionary (`tagset`) — first run** (wrapper only, no
splitting into grammemes):

| Encoding | Size | Δ vs. JSON |
|---|---|---|
| Current (JSON, `EncodeTagSet`) | 223,692 bytes | — |
| `EncodeStrings(ts.Tags)` (same wrapper, more compact) | 211,605 bytes | −5.4% |

**Tag-combination dictionary — second run** (grammeme dictionary +
indices):

- Combinations (`TagSet.Tags`): **6028**.
- Unique individual grammemes across all combinations: **100** (fits in
  a 1-byte index).
- Total grammeme occurrences across all combinations: **42,321**
  (**7.02** grammemes per combination on average).

| Encoding | Size | Δ vs. JSON | Δ vs. `EncodeStrings` |
|---|---|---|---|
| Current (JSON) | 223,692 bytes | — | — |
| `EncodeStrings(ts.Tags)` | 211,605 bytes | −5.4% | — |
| Grammeme dictionary (500 bytes) + indices (48,349 bytes) = **48,849 bytes** | **48,849 bytes** | **−78.2%** | **−76.9%** |

Round-tripping the grammeme dictionary through
`EncodeStrings`/`DecodeStrings` is lossless.

Scaled to the whole file (16,087,391 bytes by section total): a saving
of 174,843 bytes — **≈1.09% of the dictionary's total size** (it would
have been 0.075% by the first, incomplete run).

**Decoding cost on read.** `pkg/morphology/open.go:95` calls
`DecodeTagSet` once, when the file is opened (`Open`); the result is a
fully materialized `TagSet.Tags []string` in memory. `TagName(id)`
(`tagset.go:46`) after that is just `t.Tags[id]`, O(1), with no
decoding at all on the `Parse()` path (the hot path). So switching to
the "grammeme dictionary + indices" encoding adds exactly one extra bit
of work — **once, during `Open()`** — to reconstruct 6028 strings via
`strings.Join` from an average of 7 indices each (42,321 join
operations total) — in practice microseconds, immeasurable next to
reading/mmap-ing `words.dawg` (13.7 MB). The `Parse()`/`TagName()` path
doesn't change at all.

## Conclusions

1. **Paradigms are already binary** — this section isn't about them;
   there's no need to implement the user's idea for them (it matches
   the idea, but the code is already like that).
2. **The first version of this document underestimated the effect by an
   order of magnitude**, comparing only the wrapper (JSON vs. compact
   strings) and not measuring the real redundancy of repeated grammemes
   within combinations, as the user rightly pointed out. Accounting for
   splitting into grammemes, the effect is **not 5.4% of the section /
   0.075% of the file, but 78.2% of the section / ≈1.09% of the file**.
3. **The read cost after such an encoding doesn't grow on the hot
   path**: `tagset` is already materialized into `[]string` once at
   `Open()`; `TagName` is an O(1) access into the ready-made array both
   before and after the on-disk format change. The user's
   counter-argument ("storing indices is more expensive to extract") is
   valid in general for schemes where decoding happens on every
   request, but doesn't apply here specifically: the only decoding site
   is a one-time file load, not the word-lookup path. The one real
   scenario where this could matter is a very short-lived process that
   reopens the dictionary on every call (e.g. `gomorphy lookup` as a
   one-off CLI command with no persistent process); even then the cost
   is tens of thousands of small-string joins, i.e. reliably smaller
   than the time to open/read the file from disk.
4. **The real weight of the dictionary is still set by `words.dawg`**
   (~85.2% of the file) — the lever there is an order of magnitude
   bigger
   ([0001-dawg-alphabet-density.md](0001-dawg-alphabet-density.md),
   ~36.5% on the test data). But the `tagset` finding (−174.8 KB,
   ~1.09% of the file) no longer looks negligible on its own — it's the
   same order of magnitude as the "narrowing ID types" from Stage 17
   (~40% savings on the paradigms section, also not the only lever, but
   already done and accounted for).

**On versioning** (a question that was explicitly asked): the project
is pre-1.0.0, no `.dat` file has shipped to users (already recorded in
[2026-09-14-suffix-sharding-design.md](../superpowers/specs/2026-09-14-suffix-sharding-design.md))
— changing `tagset`'s byte layout **costs nothing compatibility-wise
right now**, regardless of whether it's done before or after the
release. If the change had to happen **after** 1.0.0, a section-encoding
version marker would be needed — following the precedent already set in
this project by the `CompressionNone`/`CompressionZstd` id in section
flags (`format.go`, Stage 17): `Container.Section` would need to
explicitly distinguish "old JSON" from "new dictionary+indices" and not
silently corrupt data when reading a file in the old format.

**Revised recommendation:** unlike the first version of this document —
the effect (−174.8 KB, 78.2% of the section) is significant enough to
treat this as a full-fledged backlog candidate for Stage 17, alongside
the DAWG dense alphabet and zstd compression of cold sections, rather
than "not worth a separate task." Lower priority than the DAWG dense
alphabet (the effect there is an order of magnitude bigger, and there's
no payload uncertainty there like the one the DAWG finding has), but
the implementation is comparably simple and low-risk: a new format for
a single section, decoding is a pure function with no side effects on
the read path.

## Sources

<No third-party sources were used; every reference in the text is to
files and documents in this repository, verified by reading them in
this session>
