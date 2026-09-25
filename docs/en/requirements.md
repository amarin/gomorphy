# gomorphy library requirements

> **Historical requirements (pre-1.0 design).** The current API is described in
> [library.md](library.md) and [scenarios.md](scenarios.md); items below that were
> superseded are marked.

## Context

A morphological analysis library based on the OpenCorpora (`dict.xml`)
and PyMorphy2 (compiled `words.dawg` + `paradigms.array`) dictionaries.
Supports multiple morphology sources with different grammatical tag
sets. The library doesn't manage concurrent dictionaries — the user
creates and closes `Dictionary` instances themselves (FT9). *(Superseded:
`MultiDictionary` queries several open dictionaries as one and closes them
together.)*

Reference metrics for the OpenCorpora dictionary (measured on
`dict.xml`, rev 417257):

| Entity | Value |
|---|---|
| Lemmas | 391,842 (unique texts: 380,900) |
| Wordform `<f>` lines | 5,141,267 |
| Total attributed lines (`l`+`f`) | 5,533,109 |
| Unique wordform texts | 3,065,312 (~77.6 MB UTF-8) |
| Unique grammeme sets ("ancodes") | 876 |
| Unique (text, ancode) pairs | 5,393,737 |
| Links / link types / grammemes | 258,650 / 27 / ~120 |

Reference metrics for the PyMorphy2 dictionary (pymorphy2-dicts-ru):

| Entity | Value |
|---|---|
| Lexemes (dictionary entries) | ~400,000 |
| Words in the DAWG | ~5,000,000 |
| Unique paradigms | ~3,000 |
| Unique suffixes | ~5,000 |
| On-disk size (all) | ~19 MB |
| In-memory size (all) | ~15 MB |

## Functional requirements

### FT1 — fast in-memory loading from dict.xml
Reading `dict.xml` with a minimal number of allocations. A
specialized, fixed-schema scanner (`internal/xmlscan`), no generic XML
parser. Success metric: building the full dictionary with no growth in
GC pressure, with large allocations only for the final structures.

### FT2 — fast wordform attribute lookup
Getting a wordform's grammatical characteristics with no per-query
allocations. In the new format: traversing the DAWG to a final state
-> `(para_id, form_idx)` -> index arithmetic over the paradigm ->
suffix + tag. O(len) traversal + O(1) text reconstruction.

### FT3 — fast writing and reading of a compiled dictionary
The compiled dictionary file: reads via mmap (or a large-block read) +
slicing by the catalog's sections. The format is compatible with
PyMorphy2's format (`words.dawg` + `paradigms.array` +
`suffixes.json` + `gramtab-*.json`) or is an extension of it with
additional sections. *(Superseded: the single on-disk format is GMOR, not
pymorphy2-compatible; `OpenBytes` also opens it from memory, e.g. `//go:embed`.)*

### FT4 — minimizing the compiled dictionary's size
Paradigms + a DAWG provide baseline compression (~15 MB for Russian).
Additionally: zstd-compressing the cold sections, narrowing ID types
(uint16/uint8). Target: 15-25 MB without zstd, 8-15 MB with zstd.
*(Superseded: zstd compression is not implemented — the section flag is
reserved, every section is stored uncompressed.)*

### FT5 — the base form for a given wordform
The base form is computed from the paradigm: `stem +
suffix[form_idx=0]`. Lemma lookup: DAWG -> `(para_id, form_idx)` ->
`stem + suffix[0]`. For words with several lemmas (homonymy): every
variant from the DAWG's values.

### FT6 — fuzzy search
Finding words within Levenshtein distance <= k. A joint traversal of
the DAWG and a Levenshtein DFA, pruned by the threshold. A rune-level
metric (not byte-level). Results ordered by (distance, word). For
FuzzyTop — iterative distance widening.

### FT7 — the library's dictionary-update scenario
The full cycle: download -> compile -> write to disk -> load from
disk. Support for two sources:
- OpenCorpora: `dict.xml.bz2` -> scanning -> paradigm extraction -> DAWG -> .dat
- PyMorphy2: a directory with `words.dawg` + `paradigms.array` -> a
  direct load or conversion into the unified format.

*(Superseded: UniMorph (`CompileFromUniMorph`) and TSV (`ImportTSV`) are
also supported sources.)*

### FT8 — programmatically creating and populating dictionaries
Builder API: `AddGrammeme`, `AddLemma`, `AddForm` -> `Compile` ->
`SaveTo`. The file format is the same for dictionaries from every source.
*(Superseded: there is no `AddGrammeme`/`Compile`; the API is
`NewBuilder` -> `AddForm`/`AddLemma` -> `Build` -> `SaveTo`, with opaque tag
strings.)*

### FT9 — thread safety for reads
An immutable snapshot after loading/compiling. Reads are
concurrency-safe with no locks. One `Dictionary` instance is one
language/source. Several dictionaries are several instances (managed
by the caller). The Builder isn't thread-safe — a single writer.
*(Superseded in part: `MultiDictionary` aggregates several instances.)*

### FT10 — language independence
The internal format contains no language-specific logic. Handling the
letter Ё, inflection rules, the grammeme set — all of this is defined
by the dictionary's configuration (TagSet + CharPolicy), not by the
library's code. Dictionaries for different languages (Russian,
Ukrainian, English, etc.) are supported with arbitrary grammatical tag sets.
*(Superseded in part: when no CharPolicy is given, the library picks a default
by language — е→ё for `"ru"`/`""`, none otherwise; the UniMorph importer
accepts only `"ru"`.)*

### FT11 — importing from different formats
The library provides importers for different sources:
- OpenCorpora XML (`dict.xml`) — extracting paradigms from lemmas
- PyMorphy2 (`words.dawg` + `paradigms.array`) — reading directly
- The programmatic API (Builder) — populating by hand

*(Superseded: UniMorph TSV (`CompileFromUniMorph`) and plain TSV
(`ImportTSV`) importers were added.)*

An importer converts a source into the internal format. The on-disk
format is the same regardless of the source.

### FT12 — normalizing tag sets
Different morphology sources use different grammeme sets:
- OpenCorpora: `NOUN,anim,masc,sing,nomn`
- Other systems: `S,animate,masculine,singular,nominative`
- Custom ones: arbitrary strings

The library stores tags as opaque strings. TagSet defines the mapping
from a source's set into the canonical one. Unknown tags are kept as-is.

## Requirements' impact on the implementation

- **FT1** -> reusing `internal/xmlscan` (the dict.xml scanner)
- **FT2** -> paradigms + a DAWG: an O(len) DAWG traversal -> O(1) index arithmetic
- **FT3** -> a unified file format (compatible with or close to pymorphy2's)
- **FT4** -> paradigms (~3K templates) + a DAWG (a minimized word graph)
- **FT5** -> the base form = `stem + suffix[0]` from the paradigm
- **FT6** -> Levenshtein-over-DAWG (a joint traversal)
- **FT7** -> two importers (OpenCorpora XML, PyMorphy2) + the Builder API
- **FT8** -> Builder: AddGrammeme/AddLemma/AddForm -> paradigm extraction -> DAWG
- **FT9** -> an immutable snapshot, no global state, several instances
- **FT10** -> CharPolicy (a set of substitutable characters) — configured per dictionary
- **FT11** -> `pkg/morphology/importers/` — a separate package per format
- **FT12** -> `pkg/morphology/tags` — TagSet with name mapping *(superseded:
  `pkg/morphology/tagmap`)*

## Architecture (a layered model)

```
┌─────────────────────────────────────────────────┐
│  CLI (cmd/gomorphy)                             │
│  interactive mode + CLI arguments                │
│  lookup/fuzzy/top/lemmas/import                  │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────┐
│  Public API (pkg/morphology)                    │
│  Dictionary: Open/Import/Parse/Lemma/Fuzzy/Close │
│  immutable, thread-safe for reads                │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────┐
│  Internal format (pkg/morphology/internal)      │
│  paradigm + DAWG (unified across all languages)  │
│  TagSet: tag normalization                       │
│  CharPolicy: character substitutions (Ё, etc.)   │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────┐
│  Importers (pkg/morphology/importers)           │
│  opencorpora/  — dict.xml → paradigms → DAWG    │
│  pymorphy2/    — .dawg + .array → Dictionary    │
└─────────────────────────────────────────────────┘
```

## Reusable code (from the current implementation)

| Package | Status | Usage |
|---|---|---|
| `internal/xmlscan` | Reused | The dict.xml scanner, for the OpenCorpora importer |
| `internal/intern` | Reused | String interning (suffixes, prefixes) *(superseded: removed)* |
| `internal/stringsx` | Reused | The string arena (during compilation) *(superseded: removed)* |
| `internal/mmapx` | Reused | An mmap reader for `.dat` files |
| `pkg/opencorpora` | Reused | Downloading/unpacking dict.xml.bz2 |
| `cmd/gomorphy` | Adapted | CLI: new commands (import), multi-dict |
| `cmd/opencorpora_update` | Adapted | CLI: merged with import *(superseded: removed; see `gomorphy download`/`build`)* |
| `internal/build` | **Replaced** | The new model: paradigm + DAWG |
| `internal/format` | **Replaced** | The new file format |
| `pkg/dictionary` | **Replaced** | The new public facade |
