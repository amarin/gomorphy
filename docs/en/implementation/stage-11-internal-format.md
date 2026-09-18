# Stage 11. Internal format: TagSet + Paradigm + DAWG reader

## Stage contents

Building the foundation of the new internal format: primitives for
storing morphological data. This stage is independent of the current
`internal/build` and creates a new package, `pkg/morphology/internal/`.

### TagSet (`tagset.go`)

The set of grammatical tags for a specific dictionary/language.

```go
type TagSet struct {
    Name     string
    Tags     []string          // id -> name
    TagIndex map[string]uint16 // name -> id
}
```

- Adding tags: `Add(name string) uint16` (deduplicated by name)
- Lookup: `ID(name string) (uint16, bool)`, `Name(id uint16) string`
- Serialization: a JSON string array (as in pymorphy2's gramtab)

### Paradigm (`paradigm.go`)

An inflection/conjugation template. A flat uint16 array:

```
[suffix_0, ..., suffix_N-1 | tag_0, ..., tag_N-1 | prefix_0, ..., prefix_N-1]
```

- Paradigm length = `len(data) / 3`
- Form i's suffix: `Suffix(id, i) uint16`
- Form i's tag: `Tag(id, i) uint16`
- Form i's prefix: `Prefix(id, i) uint16`

### DAWG reader (`dawg.go`)

Reading the dawgdic format (pymorphy2's words.dawg). Format:
- `dictionary`: a uint32 array. Each node is a uint32 with an 8-bit
  label, a 22-bit offset, and leaf/extension flags.
- `guide`: a byte array. 2 bytes per node: child + sibling.

Methods:
- `followByte(lbl byte, index uint32) uint32` — follow a byte transition
- `followRune(r rune, index uint32) uint32` — follow a rune transition (1-4 bytes)
- `follow(s string, index uint32) uint32` — follow a string
- `find(key string) uint32` — exact lookup -> the node's value
- `valuesForIndex(index uint32) [][]byte` — all values (walking suffixed values)

### CharPolicy (`char_policy.go`)

A set of interchangeable characters for search. Empty by default
(language-neutral). For Russian: `[{from: 'е', to: 'ё'}]`.

```go
type CharPolicy struct {
    Substitutions []Substitution
}

type Substitution struct {
    From rune
    To   rune
}
```

### Similar items (`similar_items.go`)

Traversing the DAWG while accounting for character substitutions. For
every character in `CharPolicy.From` encountered during traversal, also
try `CharPolicy.To`. Implementation: a recursive traversal that "forks"
at a substitutable character.

### Dictionary (`dictionary.go`)

An immutable dictionary snapshot:

```go
type Dictionary struct {
    Language    string
    TagSet      *TagSet
    Suffixes    []string
    Prefixes    []string
    Paradigms   []Paradigm
    Words       *DAWG
    Prediction  []*DAWG
    Probability *DAWG
    CharPolicy  *CharPolicy
}
```

## Verification (tests)

- Unit test: DAWG reader — building a test DAWG in memory, find/follow.
- Unit test: similarItems with the `е→ё` CharPolicy — checking the substitution.
- Unit test: Paradigm — index arithmetic over the flat array.
- Unit test: TagSet — Add/ID/Name, deduplication.
- Unit test: Dictionary — construction from components.
- `go test ./pkg/morphology/... -race` — green.

## Implementation (the actual API)

Package `pkg/morphology/internal` (package `internal`) — reference: the
dawgdic format/`opennota/morph` (fetch: dict.go, guide.go, dawg.go, completer.go).

Deviations from the plan:

- The DAWG's methods are **exported** (needed by importers/pymorphy2 in
  the next package): `FollowByte`, `FollowRune`, `Follow`, `Find`,
  `HasValue`, `Value`, `ValuesForIndex`, `SimilarItems`. The internal
  recursion is `similarItemsRecursive`.
- `TagSet`: a `Name` field and a `TagName(id)` method (a method named
  `Name` can't coexist with a field of the same name in Go).
- Constructors: `NewTagSet`, `NewParadigm(suffixes, tags, prefixes)`,
  `NewDAWG(dict, guide)`, `ReadDAWG(r io.Reader)` (the words.dawg format:
  `[uint32 n][n×uint32][uint32 g][g×2 bytes]`), `NewCharPolicy(subs...)`,
  `RussianCharPolicy()`, `NewDictionary(language, tagSet, suffixes, prefixes,
  paradigms, words, charPolicy)`.
- The dictionary unit's format: bits 0-7 label, bit 8 has_leaf, bit 9
  extension, bits 10-31 offset (for a value unit, the value itself, with
  bit 31 as isLeaf). Transition: `next = index ^ offset ^ label`.
- The payload separator `payloadSeparator = 0x01`; values in words.dawg
  are base64-encoded entry bytes (for `>HH`, 4 big-endian bytes).
- Tests: 19 unit tests including an independent fixture builder for a
  dictionary+guide (`dawg_test.go: buildTestDAWG`) that checks the
  reader "from the other side".
- Done: `go test -race ./pkg/morphology/...` green, `go vet` clean.
