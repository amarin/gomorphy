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
- **Fuzzy search** — Levenshtein automaton over a paradigm/DAWG trie, rune-level metrics
- **Nearest-N** — iterative distance widening to find the true N closest words
- **Multiple dictionaries at once** — `MultiDictionary` aggregates Parse/Lemma/Fuzzy across several open dictionaries
- **Multiple sources** — pymorphy2 (as-is or recompiled with a dense alphabet) and OpenCorpora `dict.xml`
- **Compact binary format** — sectioned, mmap-backed: ~13 MB for the full OpenCorpora dictionary (source `dict.xml` is ~400 MB)

## Quick start

```bash
# Install
go install github.com/amarin/gomorphy/cmd/gomorphy@latest

# Download and compile dictionary
gomorphy update opencorpora

# Query
gomorphy lookup -d .data/opencorpora/opencorpora.dat кота
```

## Documentation

| Document | Description |
|----------|-------------|
| [docs/installation.md](docs/installation.md) | Установка библиотеки и CLI-утилит |
| [docs/cli.md](docs/cli.md) | Использование CLI: lookup, lemmas, fuzzy, top, download, build, update |
| [docs/library.md](docs/library.md) | Программное использование: открытие словаря, поиск, MultiDictionary |
| [docs/mcp.md](docs/mcp.md) | Почему нет встроенного MCP-сервера |
| [docs/index.md](docs/index.md) | Полный указатель документации проекта |

## Project structure

```
cmd/gomorphy               CLI: lookup, lemmas, fuzzy, top, cli, download, unpack, build, update
pkg/morphology              public API (Open, OpenPyMorphy, CompileFromXML, Parse, Lemma, Fuzzy, MultiDictionary)
pkg/morphology/internal     TagSet + Paradigm + DAWG + sectioned binary format (not for direct use)
pkg/morphology/tagmap       native tag -> universal (UniMorph) feature bundle normalizer
pkg/morphology/importers    pymorphy2 and OpenCorpora dict.xml importers
pkg/opencorpora             OpenCorpora dict.xml download/unpack
pkg/pymorphy                pymorphy2-dicts-ru (PyPI) download/unpack
internal/xmlscan            streaming dict.xml parser
internal/mmapx              mmap reader
```

## Building from source

```bash
make build              # compile CLI binaries
make lint                # golangci-lint
make test                # go test -race ./...
make test-integration    # + integration tests (needs real dictionary data, see docs/todo.md)
```

## License

See [LICENSE](LICENSE).
