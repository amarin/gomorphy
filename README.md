# gomorphy

Morphological analysis library for Russian, powered by OpenCorpora dictionary.

Go reimplementation of PyMorphy2 with a compact binary format, mmap loading and
zero-allocation lookups.

## Features

- **Exact lookup** — all grammatical readings of a wordform (POS, case, number, …)
- **Lemma resolution** — initial form + base tags for any wordform
- **Fuzzy search** — Levenshtein automaton over a CSR trie, rune-level metrics
- **Nearest-N** — iterative distance widening to find the true N closest words
- **Programmatic building** — create dictionaries in memory without XML
- **Compact binary format** — delta/varint encoding, mmap-backed, ~300 MB for the full OpenCorpora dictionary

## Quick start

### Install

```bash
go install github.com/amarin/gomorphy/cmd/gomorphy@latest
go install github.com/amarin/gomorphy/cmd/opencorpora_update@latest
```

### Download and compile the dictionary

```bash
opencorpora_update -l        # download + unpack + compile
# produces .data/opencorpora/opencorpora.dict (~300 MB)
```

### Query

```bash
gomorphy -dict .data/opencorpora/opencorpora.dict lookup кота
# кота  NOUN,anim,masc,sing,gent  lemma#140411
# кота  NOUN,anim,masc,sing,accs  lemma#140411

gomorphy -dict .data/opencorpora/opencorpora.dict lemmas кота
# #140411  кот  NOUN,anim,masc

gomorphy -dict .data/opencorpora/opencorpora.dict fuzzy кот 1
# 0  кот
# 1  код
# 1  крот
# …

gomorphy -dict .data/opencorpora/opencorpora.dict top кот 5
# 0  кот
# 1  бот
# 1  вот
# 1  гот
# 1  дот
```

## Library usage

```go
import "github.com/amarin/gomorphy/pkg/dictionary"

d, err := dictionary.Open("opencorpora.dict")
if err != nil {
    log.Fatal(err)
}
defer d.Close()

forms, err := d.Lookup("кота")   // []Wordform
lemmas, err := d.Lemmas("кота")  // []LemmaRef
near, err := d.Fuzzy("кот", 2)   // []FuzzyMatch
top5, err := d.FuzzyTop("кот", 5) // N nearest by edit distance
```

## Building from source

```bash
make build          # compile CLI binaries
make lint           # golangci-lint
make test           # go test -race ./...
go test -tags integration -run TestFuzzyFullDict ./pkg/dictionary/
```

## Project structure

```
cmd/gomorphy           CLI: lookup, lemmas, fuzzy, top
cmd/opencorpora_update CLI: download + compile dictionary
pkg/dictionary         public API (Open, Lookup, Lemmas, Fuzzy, FuzzyTop, Builder)
internal/build         Builder + CSR snapshot
internal/format        sectioned binary format (varint/delta, xxh3 checksums)
internal/xmlscan       streaming dict.xml parser
internal/intern        string interning table (zero-alloc on hit)
internal/stringsx      byte arena + offset table
internal/mmapx         mmap reader
pkg/opencorpora        OpenCorpora download/unpack
```

## License

See [LICENSE](LICENSE).
