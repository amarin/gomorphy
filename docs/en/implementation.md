# gomorphy implementation (as-is + plan)

This document describes the current implementation (stages 0-10) and
the plan for the new implementation (stages 11-18) that replaces the
internal storage format.

Matches the requirements in [requirements.md](requirements.md) and the
plan in [todo.md](todo.md).

## Stage descriptions

### The original implementation (done)

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

### The new implementation (done)

- [x] [Stage 11. Internal format: TagSet + Paradigm + DAWG reader](implementation/stage-11-internal-format.md) — DONE
- [x] [Stage 12. PyMorphy2 import](implementation/stage-12-import-pymorphy2.md) — DONE
- [x] [Stage 13. Public API: Parse, Lemma, Fuzzy](implementation/stage-13-public-api.md) — DONE
- [x] [Stage 14. Serialization: a unified on-disk format](implementation/stage-14-serialization.md) — DONE
- [x] [Stage 15. OpenCorpora import](implementation/stage-15-import-opencorpora.md) — DONE
- [x] [Stage 16. UniMorph import](implementation/stage-16-import-unimorph.md) — DONE
- [ ] [Stage 17. Narrowing ID types and zstd](implementation/stage-17-optimize.md)
- [ ] [Stage 18. Finalization: CLI, documentation, tests](implementation/stage-18-finalize.md)
- [ ] [Stage 19. Thematic dictionaries: TSV import, CLI batches, skills, MCP decision](todo.md)
- [ ] [Stage 20. Synonym database: groups, tags, sidecar file](todo.md)

Project terminology is in [glossary.md](glossary.md).

## The current implementation (stages 0-10)

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

### The in-memory data model (current)

Every entity is normalized into lookup tables; a wordform is a pair of
identifiers.

- `grammemes []string` — grammeme names, id = index.
- Ancodes: CSR `ancodeOff []uint32` + `ancodeGrams []uint32` — 876
  unique grammeme sets, id = uint16.
- `textsArena []byte` + `textOff []uint32` — the text arena.
- The CSR trie: `stateOff`, `TransLabel`, `TransTarget`, `Finals`.
- Exact-hash: open-addressing `hash(text) -> trie state`.
- Posting lists: `PostingsOff` + `Postings` (`lemmaId, ancodeId` pairs).

### File format (current)

```
magic "GMRF" | version u32 | xxh3 checksum
catalog: [section name, offset u64, size u64] x N
sections: meta, grammemes, ancodes, textsArena, textOff,
        states, transitions, finals, exactHash, postings,
        lemmaIndex, rowAncodes, links
```

### Runtime (current)

```go
func Open(path string) (*Dictionary, error)
func (d *Dictionary) Lookup(word string) ([]Wordform, error)
func (d *Dictionary) Lemmas(word string) ([]LemmaRef, error)
func (d *Dictionary) Fuzzy(word string, maxDist int) ([]FuzzyMatch, error)
func (d *Dictionary) FuzzyTop(word string, maxWords int) ([]FuzzyMatch, error)
func (d *Dictionary) SaveTo(path string) error
```

## The new implementation (stages 11-18)

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

### The in-memory data model (new)

```
Dictionary
  TagSet         *TagSet        // the grammeme set (name -> id)
  Suffixes       []string       // the suffix set (id -> text)
  Prefixes       []string       // the prefix set (id -> text)
  Paradigms      []Paradigm     // inflection templates
  Words          *DAWG          // words -> (para_id, form_idx)
  Prediction     []*DAWG        // prediction by ending (optional)
  Probability    *DAWG          // probabilities (optional)
  CharPolicy     *CharPolicy    // character substitutions (е<->ё, etc.)
```

DAWG reader (the dawgdic format):
- `Dictionary []uint32` — the node array (label + offset + leaf bits)
- `Guide []byte` — navigation: child + sibling, 2 bytes per node
- Lookup: `followByte(r, idx)` -> O(1), `find(key)` -> O(len)
- е/ё handling: `similarItems(key)` substitutes `е→ё` on the fly

### File format (new)

```
magic "GMOR" | version u32 | xxh3 checksum
catalog: [section name, offset u64, size u64, flags u8] x N
sections:
  meta            — language, counts, version
  tagset          — a JSON array of grammeme names (shared across all shards)
  prefixes        — varint-length-prefixed strings (shared across all shards)
  suffixes-N      — varint-length-prefixed strings (one set per shard, N from 0)
  paradigms-N     — a uint16 array (N suffixes + N tags + N prefixes; one set per shard)
  words.dawg-N    — dictionary uint32[] + guide byte[] (one per shard)
  prediction-N    — prediction DAWGs (optional)
  probability     — a probability DAWG (optional)
```

### Runtime (new)

```go
// Opening from different sources
func Open(path string) (*Dictionary, error)                      // from a .dat file
func OpenPyMorphy(dir string) (*Dictionary, error)               // from a pymorphy2 directory
func CompileFromXML(r io.Reader) (*Dictionary, error)            // from dict.xml
func CompileFromUniMorph(r io.Reader) (*Dictionary, error)       // from a UniMorph TSV

// Public API (FT2, FT5, FT6)
func (d *Dictionary) Parse(word string) []Reading                // exact + prediction
func (d *Dictionary) Lemma(word string) []LemmaRef               // the base form
func (d *Dictionary) Fuzzy(word string, maxDist int) []FuzzyMatch // fuzzy search
func (d *Dictionary) FuzzyTop(word string, maxWords int) []FuzzyMatch

// Serialization (FT3, FT7)
func (d *Dictionary) SaveTo(path string) error

// Info
func (d *Dictionary) Language() string
func (d *Dictionary) TagSet() *TagSet

// Closing
func (d *Dictionary) Close() error
```

- `Reading` — a value (text, base form, tags, probability, source).
- Reads are concurrency-safe (an immutable snapshot).
- One instance = one language/source. Several dictionaries = several instances.

### Lookup (new)

- **Exact (Parse)**: a DAWG lookup -> `(para_id, form_idx)` -> index
  arithmetic over the paradigm -> `stem + suffix` + tag. O(len) DAWG traversal.
- **Prediction**: if the word isn't found — a lookup over the
  prediction DAWGs (1-5 letter endings -> sets of readings).
- **Lemmas (Lemma)**: DAWG -> `(para_id, 0)` -> `stem + suffix[0]`.
- **Fuzzy**: a joint traversal of the DAWG and a Levenshtein DFA,
  pruned by a threshold k. A rune-level metric.

## Metrics (checkpoints)

| Metric | Current (stage 10) | Target (stage 18) |
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
| UniMorph support | no | yes (169 languages, opaque) |
| Multiple dictionaries | at the application level | at the application level |
| Language neutrality | no (Russian) | yes |
