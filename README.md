# gomorphy

Morphological analysis library for Russian, powered by OpenCorpora dictionary.

Go reimplementation of [pymorphy2](https://github.com/pymorphy2/pymorphy2)
with a compact binary format, mmap loading and zero-allocation lookups.

**Platforms**: Unix only (Linux, macOS, BSD) — dictionary loading uses
`syscall.Mmap`. Windows builds compile but `morphology.Open` returns an error
at runtime; native Windows mmap support is tracked in `docs/en/todo.md`.

## Features

- **Exact lookup** — all grammatical readings of a wordform (POS, case, number, ...)
- **Lemma resolution** — initial form + base tags for any wordform
- **Fuzzy search** — Levenshtein automaton over a paradigm/DAWG trie, rune-level metrics
- **Nearest-N** — iterative distance widening to find the true N closest words
- **Multiple dictionaries at once** — `MultiDictionary` aggregates Parse/Lemma/Fuzzy across several open dictionaries
- **Multiple sources** — pymorphy2, OpenCorpora `dict.xml`, and UniMorph TSV (all dense-1-byte-alphabet by default)
- **Compact binary format** — sectioned, mmap-backed: ~13 MB for the full OpenCorpora dictionary (source `dict.xml` is ~400 MB)

## Quick start

```bash
# Install
go install github.com/amarin/gomorphy/cmd/gomorphy@latest

# Download and compile a dictionary (pymorphy2-dicts-ru, from PyPI)
gomorphy update pymorphy

# Query
gomorphy lookup -d .data/pymorphy/pymorphy.dat кота
```

> **Note:** `gomorphy update opencorpora` (downloading `dict.opcorpora.xml.bz2`
> from opencorpora.org) is also supported, but opencorpora.org is currently
> unreachable and its continued availability is not guaranteed — `pymorphy`
> is the recommended source for new setups. See [docs/en/cli.md](docs/en/cli.md).

## Documentation

Full documentation (English): **[docs/en/index.md](docs/en/index.md)**.
Minimal Russian subset (installation/CLI/library basics only):
[docs/ru/index.md](docs/ru/index.md).

| Document | English | Русский |
|----------|---------|---------|
| Installation | [docs/en/installation.md](docs/en/installation.md) | [docs/ru/installation.md](docs/ru/installation.md) |
| CLI: lookup, lemmas, fuzzy, top, download, build, update | [docs/en/cli.md](docs/en/cli.md) | [docs/ru/cli.md](docs/ru/cli.md) |
| Programmatic use: opening a dictionary, lookups, MultiDictionary | [docs/en/library.md](docs/en/library.md) | [docs/ru/library.md](docs/ru/library.md) |
| Why there's no built-in MCP server | [docs/en/mcp.md](docs/en/mcp.md) | [docs/ru/mcp.md](docs/ru/mcp.md) |
| Roadmap, implementation history, research, design specs | [docs/en/todo.md](docs/en/todo.md) | — (English only) |

## Related projects

Other Go implementations of Russian morphological analysis:

- [jus1d/gomorphy](https://github.com/jus1d/gomorphy)
- [AlexMaxy/gomorphy](https://github.com/AlexMaxy/gomorphy)
- [SteosOfficial/SteosMorphy](https://github.com/SteosOfficial/SteosMorphy)

A feature/architecture comparison against these is tracked in
[docs/en/todo.md](docs/en/todo.md).

## Project structure

```
cmd/gomorphy               CLI: lookup, lemmas, fuzzy, top, cli, download, unpack, build, update
pkg/morphology              public API (Open, OpenPyMorphy, CompileFromXML, CompileFromUniMorph, Parse, Lemma, Fuzzy, MultiDictionary)
pkg/morphology/internal     TagSet + Paradigm + DAWG + sectioned binary format (not for direct use)
pkg/morphology/tagmap       native tag -> universal (UniMorph) feature bundle normalizer
pkg/morphology/importers    pymorphy2, OpenCorpora dict.xml, and UniMorph TSV importers
pkg/opencorpora             OpenCorpora dict.xml download/unpack
pkg/pymorphy                pymorphy2-dicts-ru (PyPI) download/unpack
pkg/unimorph                UniMorph TSV download (github.com/unimorph/<iso>)
internal/xmlscan            streaming dict.xml parser
internal/mmapx              mmap reader
```

## Building from source

```bash
make build              # compile CLI binaries
make lint                # golangci-lint
make test                # go test -race ./...
make test-integration    # + integration tests (needs real dictionary data, see docs/en/todo.md)
```

## License

See [LICENSE](LICENSE).
