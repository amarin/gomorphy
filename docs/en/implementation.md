# gomorphy implementation (as-is + plan)

This document describes the legacy (pre-1.0) implementation (stages 0-10)
and the plan for the current implementation (stages 11-18) that replaced the
internal storage format. The "legacy" sections are kept for history; the
current public API is described in [library.md](library.md) and
[scenarios.md](scenarios.md).

Matches the requirements in [requirements.md](requirements.md) and the
plan in [todo.md](todo.md).

## Stage descriptions

### The original (legacy, pre-1.0) implementation (done, replaced)

- [Stage 0. Repository analysis and preparation](implementation/stage-0-analysis.md)
- [Stage 1. Format primitives](implementation/stage-1-format.md)
- [Stage 2. String interning](implementation/stage-2-intern.md)
- [Stage 3. dict.xml scanner](implementation/stage-3-xmlscan.md)
- [Stage 4. Builder and CSR structures](implementation/stage-4-builder-csr.md)
- [Stage 5. Compiler and file loader](implementation/stage-5-compiler-loader.md)
- [Stage 6. Public facade pkg/dictionary](implementation/stage-6-facade.md)
- [Stage 7. End-to-end OpenCorpora integration](implementation/stage-7-opencorpora.md)
- [Stage 8. FT5 lemma lookup](implementation/stage-8-lemmas.md)
- [Stage 9. FT6 fuzzy search](implementation/stage-9-fuzzy.md)
- [Stage 10. Finalization](implementation/stage-10-finalize.md)

### The current implementation

- [x] [Stage 11. Internal format: TagSet + Paradigm + DAWG reader](implementation/stage-11-internal-format.md) — DONE
- [x] [Stage 12. PyMorphy2 import](implementation/stage-12-import-pymorphy2.md) — DONE
- [x] [Stage 13. Public API: Parse, Lemma, Fuzzy](implementation/stage-13-public-api.md) — DONE
- [x] [Stage 14. Serialization: a unified on-disk format](implementation/stage-14-serialization.md) — DONE
- [x] [Stage 15. OpenCorpora import](implementation/stage-15-import-opencorpora.md) — DONE
- [x] [Stage 16. UniMorph import](implementation/stage-16-import-unimorph.md) — DONE
- [ ] [Stage 17. Narrowing ID types and zstd](implementation/stage-17-optimize.md) — PARTIAL (types narrowed, format groundwork; zstd not implemented)
- [x] [Stage 18. Finalization: CLI, documentation, tests](implementation/stage-18-finalize.md) — DONE
- [x] [Stage 19. Builder API, TSV import, Merge](implementation/stage-19-builder-tsv-merge.md) — DONE (1.1.0); CLI batch modes and the skill remain open in [todo.md](todo.md)
- [x] [NER support: Reading.Predicted, IsKnown, OpenBytes, CharPolicy, ContentHash](implementation/ner-support.md) — DONE (1.2.0)
- [ ] [Stage 20. Synonym database: groups, tags, sidecar file](todo.md)

Project terminology is in [glossary.md](glossary.md).

## The legacy implementation (stages 0-10, pre-1.0, replaced)

### Repository layout

```
pkg/dictionary    the library's public facade (FT7-FT9)
internal/build    Builder: building a dictionary from XML or programmatically, CSR structures
internal/format   the file format: header, section catalog, varint/delta encoding
internal/xmlscan  a byte scanner for dict.xml with no per-token allocations
internal/intern   a string interning table (open addressing)
internal/stringsx a string arena + an offset table
internal/mmapx    an mmap reader for sections
pkg/opencorpora   the OpenCorpora downloader/unpacker (unchanged behavior)
cmd/opencorpora_update CLI: update, unpack, compile
cmd/gomorphy      CLI: exact lookup, lemmas, fuzzy search over a .dat
```

### The in-memory data model (legacy)

Every entity is normalized into lookup tables; a wordform is a pair of
identifiers.

- `grammemes []string` — grammeme names, id = index.
- Ancodes: CSR `ancodeOff []uint32` + `ancodeGrams []uint32` — 876
  unique grammeme sets, id = uint16.
- `textsArena []byte` + `textOff []uint32` — the text arena.
- The CSR trie: `stateOff`, `TransLabel`, `TransTarget`, `Finals`.
- Exact-hash: open-addressing `hash(text) -> trie state`.
- Posting lists: `PostingsOff` + `Postings` (`lemmaId, ancodeId` pairs).

### File format (legacy)

```
magic "GMRF" | version u32 | xxh3 checksum
catalog: [section name, offset u64, size u64] x N
sections: meta, grammemes, ancodes, textsArena, textOff,
        states, transitions, finals, exactHash, postings,
        lemmaIndex, rowAncodes, links
```

### Runtime (legacy)

```go
func Open(path string) (*Dictionary, error)
func (d *Dictionary) Lookup(word string) ([]Wordform, error)
func (d *Dictionary) Lemmas(word string) ([]LemmaRef, error)
func (d *Dictionary) Fuzzy(word string, maxDist int) ([]FuzzyMatch, error)
func (d *Dictionary) FuzzyTop(word string, maxWords int) ([]FuzzyMatch, error)
func (d *Dictionary) SaveTo(path string) error
```

## The current implementation (stages 11-18 and later)

### Repository layout (after stage 18)

```
pkg/morphology/               the public facade: Open, Parse, Lemma, Fuzzy
pkg/morphology/internal/      the internal format: TagSet, Paradigm, DAWG, Dictionary
pkg/morphology/importers/     importers from different formats
pkg/morphology/importers/pymorphy2/   reads words.dawg + paradigms.array
pkg/morphology/importers/opencorpora/ dict.xml -> paradigms -> DAWG
pkg/morphology/importers/unimorph/    TSV -> paradigms -> DAWG

internal/xmlscan              (reused) the dict.xml scanner
internal/mmapx                (reused) the mmap reader
pkg/opencorpora               (reused) the OpenCorpora loader

cmd/gomorphy                  CLI: lookup/fuzzy/top/lemmas, cli, download/unpack/build/update
                               (folded cmd/gomorphy_build's update/compile into this single
                               binary during the CLI redesign, see docs/en/superpowers/specs/
                               2026-09-16-cli-redesign-design.md)
```

### The in-memory data model (current)

```
Dictionary
  TagSet         *TagSet        // the grammeme set (name -> id)
  Suffixes       []string       // the suffix set (id -> text)
  Prefixes       []string       // the prefix set (id -> text)
  Paradigms      []Paradigm     // inflection templates
  Words          *DAWG          // words -> (para_id, form_idx)
  Prediction     []*DAWG        // prediction by ending (optional)
  Probability    *DAWG          // probabilities (optional)
  CharPolicy     *CharPolicy    // one-way lookup substitutions (е→ё for "ru" by default)
  Alphabet       Alphabet       // dense key alphabet (optional)
```

DAWG reader (the dawgdic format):
- `Dictionary []uint32` — the node array (label + offset + leaf bits)
- `Guide []byte` — navigation: child + sibling, 2 bytes per node
- Lookup: `followByte(r, idx)` -> O(1), `find(key)` -> O(len)
- е/ё handling: `SimilarItems(key)` applies the dictionary's CharPolicy on the
  fly — one-way (a query `е` also matches a stored `ё`, not vice versa); the
  default is `е→ё` for `"ru"`, none for other languages

### File format (current)

```
magic "GMOR" | version u32 | xxh3 checksum
catalog: [section name, offset u64, size u64, flags u8] x N
sections:
  meta            — language + CharPolicy
  info            — BuildInfo: source, versions, build time (optional)
  tagset          — a JSON array of grammeme names (shared across all shards)
  prefixes        — varint-length-prefixed strings (shared across all shards)
  suffixes-N      — varint-length-prefixed strings (one set per shard, N from 0)
  paradigms-N     — a uint16 array (N suffixes + N tags + N prefixes; one set per shard)
  words.dawg-N    — dictionary uint32[] + guide byte[] (one per shard)
  prediction-N    — prediction DAWGs (optional)
  probability     — a probability DAWG (optional)
  alphabet        — the dense key alphabet (optional)
```

The per-section flags byte reserves a compression id (zstd), but compression
is not implemented: every section is written uncompressed.

### Runtime (current)

A summary; the authoritative reference is [library.md](library.md) and godoc.

```go
// Opening from different sources
func Open(path string) (*Dictionary, error)                      // from a .dat file (mmap)
func OpenBytes(data []byte) (*Dictionary, error)                 // from memory, e.g. //go:embed
func OpenPyMorphy(dir string) (*Dictionary, error)               // from a pymorphy2 directory
func CompileFromXML(r io.Reader, progress opencorpora.Progress) (*Dictionary, error) // from dict.xml
func CompileFromUniMorph(r io.Reader, opts UniMorphOptions) (*Dictionary, error)     // from a UniMorph TSV
// …Dense variants (OpenPyMorphyDense, CompileFromXMLDense, CompileFromUniMorphDense, …)
// and …File variants re-encode keys with a dense alphabet.

// Building your own dictionary (1.1.0)
func NewBuilder(opts BuilderOptions) *Builder                     // AddForm / AddLemma / Build
func ImportTSV(r io.Reader, opts BuilderOptions) (*Dictionary, error)
func Merge(base *Dictionary, overlays []*Dictionary, mode MergeMode) (*Dictionary, error)

// Public API (FT2, FT5, FT6)
func (d *Dictionary) Parse(word string) []Reading                // exact + prediction
func (d *Dictionary) IsKnown(word string) bool                   // exact only, no prediction
func (d *Dictionary) Lemma(word string) []LemmaRef               // the base form
func (d *Dictionary) Fuzzy(word string, maxDist int) []FuzzyMatch // fuzzy search
func (d *Dictionary) FuzzyTop(word string, maxWords int) []FuzzyMatch

// Serialization (FT3, FT7)
func (d *Dictionary) SaveTo(path string) error

// Info
func (d *Dictionary) Language() string
func (d *Dictionary) TagSetName() string
func (d *Dictionary) Info() *BuildInfo
func (d *Dictionary) ContentHash() string

// Closing
func (d *Dictionary) Close() error
```

- `Reading` — a value (text, base form, tags, probability, `Predicted`, shard/dictionary index).
- Reads are concurrency-safe (an immutable snapshot).
- One instance = one language/source. Several dictionaries can be queried
  as one through `MultiDictionary` (`NewMultiDictionary`).

### Lookup (new)

- **Exact (Parse)**: a DAWG lookup -> `(para_id, form_idx)` -> index
  arithmetic over the paradigm -> `stem + suffix` + tag. O(len) DAWG traversal.
- **Prediction**: if the word isn't found — a lookup over the
  prediction DAWGs (1-5 letter endings -> sets of readings).
- **Lemmas (Lemma)**: DAWG -> `(para_id, 0)` -> `stem + suffix[0]`.
- **Fuzzy**: a joint traversal of the DAWG and a Levenshtein DFA,
  pruned by a threshold k. A rune-level metric.

## Metrics (checkpoints)

| Metric | Legacy (stage 10) | Target (stage 18) |
|---|---|---|
| .dat size (OpenCorpora) | ~305 MB | ~20-30 MB |
| .dat size (PyMorphy2) | — | ~15-20 MB |
| .dat size with zstd | — | ~10-15 MB |
| Building the .dat (OpenCorpora, 3.06M lemmas) | ~24 h (before the free-list fix) | ~24 s |
| Loading | mmap, ms | mmap, ms |
| Parse (exact) | < 10 us | < 10 us |
| Parse (prediction) | none | < 50 us |
| Lemmas | < 10 us | < 10 us |
| Fuzzy k<=2 | fractions of a second | fractions of a second |
| pymorphy2 support | no | yes |
| UniMorph support | no | yes (only `"ru"` is accepted today) |
| Multiple dictionaries | at the application level | `MultiDictionary` |
| Language neutrality | no (Russian) | yes |
