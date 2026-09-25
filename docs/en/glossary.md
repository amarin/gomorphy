# gomorphy Glossary

Project terminology for developers unfamiliar with the specifics of
morphological analysis and gomorphy's internal architecture.

---

## Dictionary entities

### Lemma (лемма)
A group of wordforms united by a common meaning and inflection pattern.
In practice — a dictionary entry: the citation form plus all of its forms.
Example: the lemma "кот" (cat) includes the forms кот, кота, коту, котом, коты, котов, ...

### Citation form (начальная форма / Lemma text / Citation form)
The first form of a lemma — the one that appears as the dictionary's headword.
For nouns — nominative singular; for verbs — the infinitive.
In pymorphy2: `stem + suffix[form_idx=0]`.

### Wordform (словоформа / Form)
A specific form of a word with grammatical information attached.
In gomorphy (as in pymorphy2) the words DAWG maps it to `(para_id, form_idx)` →
`(prefix + stem + suffix, tag)`. The legacy (pre-1.0) GMRF format stored it as a
`(textID, ancodeID)` pair — text + a set of grammemes.
Example: "кота" is a wordform of the lemma "кот", with grammemes NOUN,anim,masc,sing,gent.

### Stem (стем)
The part of a word shared by all forms of a lemma. Computed as the longest common
prefix of all form texts. In pymorphy2 the stem is stored once, and suffixes from
the paradigm are appended to it. gomorphy (since 1.0) uses the same scheme; the
legacy (pre-1.0) GMRF format stored full form texts instead.

### Paradigm (парадигма)
An inflection/conjugation template — a flat array of `(suffix_id, tag_id, prefix_id)`
tuples for each form minus the stem. In pymorphy2, for Russian: ~3,000 unique
paradigms out of ~400K lexemes. A single paradigm is reused across different
lemmas that share the same inflection type.

### Ancode (анкод)
A unique set of grammemes (grammatical tags) attached to a wordform.
OpenCorpora has 876 unique ancodes. In pymorphy2 these are the paradigm tags
(tag id in the TagSet).

### Grammeme (граммема)
An elementary grammatical characteristic (tag). OpenCorpora has ~120
grammemes: part of speech (NOUN, VERB, ADJF, ...), gender (masc, femn, neut), case
(nomn, gent, datv, ...), number (sing, plur), and so on.

### TagSet (набор граммем)
The definition of the available grammemes for a specific dictionary/language. Includes:
- the language name (e.g. "ru")
- the list of grammeme names (e.g. ["NOUN", "anim", "masc", ...])
- a reverse index (name → id)

In pymorphy2: `gramtab-opencorpora-int.json`.

### CharPolicy (политика символов)
A dictionary's set of one-way lookup substitutions: a query rune `From` also
matches a stored rune `To`. For Russian: `е→ё`, so «елка» finds «ёлка» (but
«ёлка» does not find «елка»). The default depends on the dictionary language:
`е→ё` for `"ru"` (and `""`, which means `"ru"`), no substitutions otherwise.
Stored in the `meta` section at build time and applied by `Parse`, `Lemma`,
`IsKnown`, `Fuzzy` and `FuzzyTop` (in `Fuzzy`, an `е`→`ё` match costs distance 0).
At most 255 substitutions. Public API since 1.2.0: `NewCharPolicy`,
`RussianCharPolicy`, `NoCharPolicy` (set via `BuilderOptions.CharPolicy` /
`UniMorphOptions.CharPolicy`).

---

## Data structures

### DAWG (Directed Acyclic Word Graph)
A minimized trie — an automaton in which equivalent states
(with identical subtrees) are merged. A DAWG compresses shared prefixes and
suffixes of words. In pymorphy2: 5 million wordforms ≈ 7 MB.

dawgdic format:
- `dictionary`: uint32 array — nodes with a label (8 bits), offset (22 bits), and flags.
- `guide`: byte array — navigation (child + sibling, 2 bytes per node).

### Trie (префиксное дерево)
A tree in which the path from the root to a node forms a word. Each edge is labeled
with a single byte. In the legacy (pre-1.0) gomorphy it was implemented as CSR
(Compressed Sparse Row) — three parallel arrays. Replaced by a DAWG in the current format.

### CSR (Compressed Sparse Row) — legacy (pre-1.0)
A way of storing sparse lists in three flat arrays instead of an array of
pointers. For a trie: `StateOff[i]` is the start of state i's transition window,
`TransLabel[StateOff[i]:StateOff[i+1]]` are the transition labels,
`TransTarget[...]` are the target states.

### Exact-hash
An open-addressing hash table for fast O(1) lookup: `hash(text) → trie
state`. Legacy (pre-1.0); absent in the current format: the DAWG provides O(len) lookup
without an additional structure.

### Arena (арена)
A contiguous byte buffer holding all unique texts back to back. A text is
accessible by index via an offset table. Legacy (pre-1.0); in the current format texts are
stored in suffixes/prefixes rather than a separate arena.

### Interning (интернирование) — legacy (pre-1.0)
The process of matching a byte slice to an existing text in the arena or
adding a new one. Hash of `[]byte` → open-addressing table → on a hit
the existing id is returned with no allocation.

### Snapshot / Dictionary (снимок)
The immutable `Dictionary` structure — the entire in-memory dictionary state:
tagset, suffixes, prefixes, paradigms, words DAWG, prediction DAWGs.
Reads are thread-safe without locks. One instance = one language/source.

### Pair (пара) — legacy (pre-1.0)
The storage unit for a wordform in the legacy (pre-1.0) gomorphy: a `(textID, ancodeID)`
tuple. Not used in the current format: the DAWG stores `(para_id, form_idx)`.

### Posting list (постинг-лист) — legacy (pre-1.0)
A list of `(textID, ancodeID)` pair identifiers attached to a final
trie state. Absent in the current format: the DAWG contains `(para_id, form_idx)`.

### Dense alphabet (плотный алфавит)
An optional `alphabet` section that re-encodes every rune the dictionary uses
as a compact fixed-width code (1 or 2 bytes per rune) instead of UTF-8, shrinking the DAWGs. Produced by
the `…Dense` constructors (`OpenPyMorphyDense`, `CompileFromXMLDense`,
`CompileFromUniMorphDense`, …) and by every `Builder`, `ImportTSV` and `Merge`
result; lookups give the same results as without it.

---

## Format and encoding

### GMRF format (legacy, pre-1.0)
The binary file format of the compiled dictionary: a header (magic "GMRF",
version, indexOffset), data sections, a section directory (name → offset + size),
and a trailer (xxh3-64 checksum).

### GMOR format (current)
The binary format written by `SaveTo` and read by `Open`/`OpenBytes`: a header,
sections, an extended section directory and an xxh3 checksum. Each section has
a compression flag reserved for zstd, but compression is not implemented
(stage 17 is partial): every section is stored uncompressed, and a compressed
one is rejected on load. Sections: meta (language + CharPolicy), info
(BuildInfo, optional), tagset, prefixes (shared across all shards), suffixes-N,
paradigms-N, words.dawg-N (one set per shard — see
docs/en/superpowers/specs/2026-09-14-suffix-sharding-design.md),
prediction-N, probability, alphabet (optional, dense alphabet).

### pymorphy2 format
The pymorphy2 dictionary format:
- `words.dawg`: dictionary uint32[] + guide byte[]
- `paradigms.array`: uint16 count + N × (uint16 len + []uint16 data)
- `suffixes.json`, `paradigm-prefixes.json`: JSON arrays of strings
- `gramtab-opencorpora-int.json`: JSON array of tags
- `prediction-suffixes-N.dawg`: prediction DAWGs (optional)

### Delta encoding
A way of encoding monotonically increasing sequences: the difference
between adjacent elements (delta) is stored instead of absolute values.

### Zigzag encoding
Converting a signed int64 to an unsigned uint64: negative values
are interleaved with positive ones. Allows negative deltas to be varint-encoded.

### Varint (uvarint)
Variable-length integer encoding: values 0–127 take 1 byte,
128–16383 take 2 bytes, and so on.

### mmap (Memory-mapped file)
A technique for mapping a file into a process's virtual memory. Allows
reading file sections directly as `[]byte` slices without copying.

### Zero-copy loading
Loading a dictionary without element-by-element parsing: fixed-width sections
are reinterpreted via `unsafe.Slice` directly from the mmap region.

### zstd
A data compression format. Planned for cold sections of the GMOR format (a
compression flag is reserved per section), but not implemented: all sections
are currently stored uncompressed.

---

## Processes

### Compile / Build (компиляция)
Converting source data into a compiled binary dictionary.
- OpenCorpora: dict.xml → paradigms → DAWG → .dat (`CompileFromXML`)
- PyMorphy2: a directory of files → .dat (`OpenPyMorphy`)
- UniMorph: a TSV file → .dat (`CompileFromUniMorph`)
- Your own data: `Builder` / `ImportTSV` → .dat

### Builder (сборщик)
The public API for building a dictionary from your own data:
`NewBuilder(opts)`, then `AddForm(word, lemma, tag)` / `AddLemma(normal, tag)`,
then `Build()` compiles the entries into an immutable `Dictionary`. Words and
lemmas are lower-cased; tags are opaque strings stored verbatim. Single-use
(after `Build`, further calls return `ErrBuilderClosed`) and not thread-safe
(single writer).

### Import (импорт)
Reading a dictionary from an external format and converting it to the internal format.
- `OpenPyMorphy(dir)`: reads words.dawg + paradigms.array + strings
- `CompileFromXML(r, progress)`: dict.xml → paradigms → DAWG
- `CompileFromUniMorph(r, opts)`: UniMorph TSV → paradigms → DAWG
- `ImportTSV(r, opts)`: a `lemma<TAB>wordform[<TAB>tags]` stream → dictionary

### Lookup / Parse (точный поиск)
Looking up a wordform in the dictionary. In the current format:
1. DAWG lookup → `(para_id, form_idx)`.
2. Index arithmetic over the paradigm → suffix + tag.
3. Text reconstruction: `stem + suffix`.

### Fuzzy (нечёткий поиск)
Searching for words within Levenshtein distance ≤ k: a joint traversal of the DAWG
and a Levenshtein DFA with threshold pruning. Distance measured in runes (not bytes).

### Lemma (лемматизация)
Deriving the citation form from a wordform. In the current format:
1. DAWG lookup → `(para_id, form_idx)`.
2. `stem + suffix[0]` — the citation form.

### Prediction (предсказание)
Parsing out-of-dictionary words by their endings. Uses prediction DAWGs
(separate DAWGs for 1–5-letter endings).

### Predicted / known word (предсказанное / известное слово)
A `Reading` with `Predicted == true` came from prediction, not from the
dictionary. `IsKnown(word)` reports whether the word has at least one
dictionary reading — true exactly when `Parse` returns readings with
`Predicted == false`; it never falls back to prediction.

### Merge / overlay (слияние)
`Merge(base, overlays, mode)` combines a base dictionary with overlay
dictionaries into a new one: `MergeAdd` adds only words absent so far,
`MergeReplace` lets an overlay word's readings replace the earlier ones. The
base's tag set, prediction and CharPolicy are kept (`MergeWithOptions` can
rebuild prediction). CLI: `gomorphy merge`.

### MultiDictionary
A set of independently opened dictionaries queried as one whole
(`NewMultiDictionary(dicts...)`): `Parse`/`Lemma`/`Fuzzy`/`FuzzyTop`/`IsKnown`
concatenate results in registration order, `Reading.Dict` tells which
dictionary answered, `Close` closes them all.

### OpenBytes
Opens a GMOR dictionary from an in-memory byte slice (typically a file embedded
with `//go:embed`) instead of mmap-ing a file. Works on every platform,
Windows included.

### ContentHash
`Dictionary.ContentHash()` — a stable xxh3-128 hex digest of the dictionary's
content (every saved section except `info`). Unchanged by re-saving or
reopening; two dictionaries with equal hashes parse every word identically.

### OpenCorpora
An open Russian-language morphological dictionary project
(http://opencorpora.org). Source format — XML (`dict.xml`).

### PyMorphy2
A morphological analyzer for the Russian language. Storage format:
DAWG + paradigms + strings. The `opennota/morph` (Go) library reads
this format directly.
