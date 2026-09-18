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
In gomorphy it is represented as a `(textID, ancodeID)` pair — text + a set of grammemes.
In pymorphy2: `(para_id, form_idx)` → `(stem + suffix, tag)`.
Example: "кота" is a wordform of the lemma "кот", with grammemes NOUN,anim,masc,sing,gent.

### Stem (стем)
The part of a word shared by all forms of a lemma. Computed as the longest common
prefix of all form texts. In pymorphy2 the stem is stored once, and suffixes from
the paradigm are appended to it. Not used in the current gomorphy.

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
A set of substitutable characters for search. Empty by default (language-neutral).
For Russian: `[{from: 'е', to: 'ё'}]`. Allows finding
words with "ё" when "е" is typed, and vice versa.

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
with a single byte. In gomorphy it is implemented as CSR (Compressed Sparse Row) —
three parallel arrays. Replaced by a DAWG in the new format.

### CSR (Compressed Sparse Row)
A way of storing sparse lists in three flat arrays instead of an array of
pointers. For a trie: `StateOff[i]` is the start of state i's transition window,
`TransLabel[StateOff[i]:StateOff[i+1]]` are the transition labels,
`TransTarget[...]` are the target states.

### Exact-hash
An open-addressing hash table for fast O(1) lookup: `hash(text) → trie
state`. Absent in the new format: the DAWG provides O(len) lookup
without an additional structure.

### Arena (арена)
A contiguous byte buffer holding all unique texts back to back. A text is
accessible by index via an offset table. In the new format texts are
stored in suffixes/prefixes rather than a separate arena.

### Interning (интернирование)
The process of matching a byte slice to an existing text in the arena or
adding a new one. Hash of `[]byte` → open-addressing table → on a hit
the existing id is returned with no allocation.

### Snapshot / Dictionary (снимок)
The immutable `Dictionary` structure — the entire in-memory dictionary state:
tagset, suffixes, prefixes, paradigms, words DAWG, prediction DAWGs.
Reads are thread-safe without locks. One instance = one language/source.

### Pair (пара)
The storage unit for a wordform in the current gomorphy: a `(textID, ancodeID)`
tuple. Not used in the new format: the DAWG stores `(para_id, form_idx)`.

### Posting list (постинг-лист)
A list of `(textID, ancodeID)` pair identifiers attached to a final
trie state. Absent in the new format: the DAWG contains `(para_id, form_idx)`.

---

## Format and encoding

### GMRF format (current)
The binary file format of the compiled dictionary: a header (magic "GMRF",
version, indexOffset), data sections, a section directory (name → offset + size),
and a trailer (xxh3-64 checksum).

### GMOR format (new)
A binary format with an extended section directory and zstd compression.
Sections: meta, tagset, prefixes (shared across all shards), suffixes-N,
paradigms-N, words.dawg-N (one set per shard — see
docs/superpowers/specs/2026-09-14-suffix-sharding-design.md),
prediction-N, probability.

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
A data compression format. Used for cold sections in the new format.
Decompressed once at load time.

---

## Processes

### Compile / Build (компиляция)
Converting source data into a compiled binary dictionary.
- OpenCorpora: dict.xml → paradigms → DAWG → .dat
- PyMorphy2: a directory of files → .dat

### Builder (сборщик)
The mutable compilation phase: accumulates grammemes, lemmas, and wordforms via
`AddGrammeme`/`AddLemma`/`AddForm`. Not thread-safe (single writer).
`Build()` compiles the data into an immutable `Dictionary`.

### Import (импорт)
Reading a dictionary from an external format and converting it to the internal format.
- `OpenPyMorphy(dir)`: reads words.dawg + paradigms.array + strings
- `CompileFromXML(r)`: dict.xml → paradigms → DAWG

### Lookup / Parse (точный поиск)
Looking up a wordform in the dictionary. In the new format:
1. DAWG lookup → `(para_id, form_idx)`.
2. Index arithmetic over the paradigm → suffix + tag.
3. Text reconstruction: `stem + suffix`.

### Fuzzy (нечёткий поиск)
Searching for words within Levenshtein distance ≤ k: a joint traversal of the DAWG
and a Levenshtein DFA with threshold pruning. Distance measured in runes (not bytes).

### Lemma (лемматизация)
Deriving the citation form from a wordform. In the new format:
1. DAWG lookup → `(para_id, form_idx)`.
2. `stem + suffix[0]` — the citation form.

### Prediction (предсказание)
Parsing out-of-dictionary words by their endings. Uses prediction DAWGs
(separate DAWGs for 1–5-letter endings).

### OpenCorpora
An open Russian-language morphological dictionary project
(http://opencorpora.org). Source format — XML (`dict.xml`).

### PyMorphy2
A morphological analyzer for the Russian language. Storage format:
DAWG + paradigms + strings. The `opennota/morph` (Go) library reads
this format directly.
