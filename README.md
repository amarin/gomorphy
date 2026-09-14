# gomorphy

Morphological analysis library for Russian, powered by OpenCorpora dictionary.

Go reimplementation of PyMorphy2 with a compact binary format, mmap loading and
zero-allocation lookups.

**Platforms**: Unix only (Linux, macOS, BSD) — dictionary loading uses
`syscall.Mmap`. Windows builds compile but `morphology.Open` returns an error
at runtime; native Windows mmap support is tracked in `docs/todo.md`.

## Features

- **Exact lookup** — all grammatical readings of a wordform (POS, case, number, ...)
- **Lemma resolution** — initial form + base tags for any wordform
- **Fuzzy search** — Levenshtein automaton over a CSR trie, rune-level metrics
- **Nearest-N** — iterative distance widening to find the true N closest words
- **Programmatic building** — create dictionaries in memory without XML
- **Compact binary format** — delta/varint encoding, mmap-backed, ~300 MB for the full OpenCorpora dictionary

## Quick start

```bash
# Install
go install github.com/amarin/gomorphy/cmd/gomorphy@latest
go install github.com/amarin/gomorphy/cmd/opencorpora_update@latest

# Download and compile dictionary
opencorpora_update -l

# Query
gomorphy -dict .data/opencorpora/opencorpora.dict lookup кота
```

## Documentation

| Document | Description |
|----------|-------------|
| [docs/installation.md](docs/installation.md) | Установка библиотеки и CLI-утилит |
| [docs/cli.md](docs/cli.md) | Использование CLI: lookup, lemmas, fuzzy, top |
| [docs/library.md](docs/library.md) | Программное использование: подключение словаря, поиск, создание собственных словарей |
| [docs/mcp.md](docs/mcp.md) | Почему нет встроенного MCP-сервера |

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

## Building from source

```bash
make build          # compile CLI binaries
make lint           # golangci-lint
make test           # go test -race ./...
```

## License

See [LICENSE](LICENSE).
