# gomorphy

Morphological analysis library for Russian: pymorphy2, OpenCorpora and
UniMorph data, or dictionaries built from your own wordforms.

Go reimplementation of [pymorphy2](https://github.com/pymorphy2/pymorphy2)
with a compact binary format and mmap loading.

**Platforms**: `morphology.Open` (mmap) works on Unix (Linux, macOS, BSD).
On Windows use `morphology.OpenBytes` — a dictionary already in memory, e.g.
embedded with `//go:embed`; native Windows mmap is tracked in
`docs/en/todo.md`.

## Features

- **Exact lookup** — all grammatical readings of a wordform (POS, case, number, ...)
- **Lemma resolution** — initial form + base tags for any wordform
- **Known word vs. guess** — `IsKnown` and `Reading.Predicted` separate dictionary words from ending-based predictions (dictionary-based NER)
- **Fuzzy search** — Levenshtein automaton over a paradigm/DAWG trie, rune-level metrics; е in a query matches ё
- **Nearest-N** — iterative distance widening to find the true N closest words
- **Your own dictionaries** — `Builder`/`ImportTSV` (or `gomorphy import tsv`) build a dictionary from your wordforms; `Merge` adds them into a base dictionary
- **Multiple dictionaries at once** — `MultiDictionary` aggregates Parse/Lemma/Fuzzy/IsKnown across several open dictionaries
- **Multiple sources** — pymorphy2, OpenCorpora `dict.xml`, and UniMorph TSV (all dense-1-byte-alphabet by default)
- **Embeddable** — `OpenBytes` opens a dictionary from memory (`//go:embed`), no data file at run time
- **Compact binary format** — sectioned, mmap-backed: ~10 MB for the full OpenCorpora dictionary (source `dict.xml` is ~400 MB)

What each of these is for, with runnable examples: [docs/en/scenarios.md](docs/en/scenarios.md).

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

Using the Go API instead of the CLI? See [examples/](examples/README.md)
for runnable programs — most need no download at all
(`go run ./examples/ner`).

## Documentation

Full documentation (English): **[docs/en/index.md](docs/en/index.md)**.
Russian subset (scenarios, installation, CLI, library):
[docs/ru/index.md](docs/ru/index.md).

| Document | English | Русский |
|----------|---------|---------|
| Usage scenarios: what each feature is for, since which version | [docs/en/scenarios.md](docs/en/scenarios.md) | [docs/ru/scenarios.md](docs/ru/scenarios.md) |
| Installation | [docs/en/installation.md](docs/en/installation.md) | [docs/ru/installation.md](docs/ru/installation.md) |
| CLI: lookup, lemmas, fuzzy, top, download, build, update, import tsv, merge | [docs/en/cli.md](docs/en/cli.md) | [docs/ru/cli.md](docs/ru/cli.md) |
| Programmatic use: opening a dictionary, lookups, MultiDictionary, building (Builder, ImportTSV) and merging dictionaries | [docs/en/library.md](docs/en/library.md) | [docs/ru/library.md](docs/ru/library.md) |
| Why there's no built-in MCP server | [docs/en/mcp.md](docs/en/mcp.md) | [docs/ru/mcp.md](docs/ru/mcp.md) |
| Comparison with other Go morphological analyzers | [docs/en/comparison.md](docs/en/comparison.md) | — (English only) |
| Roadmap, implementation history, research, design specs | [docs/en/todo.md](docs/en/todo.md) | — (English only) |
| Release history | [CHANGELOG.md](CHANGELOG.md) | — (English only) |

## Related projects

Other Go implementations of Russian morphological analysis:

- [jus1d/gomorphy](https://github.com/jus1d/gomorphy)
- [AlexMaxy/gomorphy](https://github.com/AlexMaxy/gomorphy)
- [SteosOfficial/SteosMorphy](https://github.com/SteosOfficial/SteosMorphy)

A full feature/architecture/license comparison against these is in
[docs/en/comparison.md](docs/en/comparison.md).

## Project structure

```
cmd/gomorphy               CLI: lookup, lemmas, fuzzy, top, cli, download, unpack, build, update, import, merge, version
pkg/morphology              public API (Open, OpenBytes, OpenPyMorphy, CompileFrom*, Builder, ImportTSV, Merge, Parse, Lemma, IsKnown, Fuzzy, MultiDictionary)
pkg/morphology/internal     TagSet + Paradigm + DAWG + sectioned binary format (not for direct use)
pkg/morphology/tagmap       native tag -> universal (UniMorph) feature bundle normalizer
pkg/morphology/importers    pymorphy2, OpenCorpora dict.xml, and UniMorph TSV importers
pkg/opencorpora             OpenCorpora dict.xml download/unpack
pkg/pymorphy                pymorphy2-dicts-ru (PyPI) download/unpack
pkg/unimorph                UniMorph TSV download (github.com/unimorph/<iso>)
pkg/common                  data paths and logging shared by the download loaders
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
