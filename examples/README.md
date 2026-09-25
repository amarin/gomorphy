# Examples

Minimal, runnable programs for the `pkg/morphology` entry points and the
usage scenarios in [docs/en/scenarios.md](../docs/en/scenarios.md)
([Russian](../docs/ru/scenarios.md)). See
[docs/en/library.md](../docs/en/library.md) for the full API reference.

| Example | Entry point | Needs external data? |
|---|---|---|
| [opencorpora](opencorpora/main.go) | `CompileFromXML` | no — a tiny `dict.xml` fragment is inlined |
| [unimorph](unimorph/main.go) | `CompileFromUniMorph` | no — a tiny TSV fragment is inlined |
| [multidict](multidict/main.go) | `NewMultiDictionary` | no — two tiny `dict.xml` fragments are inlined |
| [builder](builder/main.go) | `NewBuilder` | no — wordforms registered in code |
| [importtsv](importtsv/main.go) | `ImportTSV` | no — a tiny TSV stream is inlined |
| [merge](merge/main.go) | `Merge` | no — two tiny dictionaries are built in code |
| [ner](ner/main.go) | `IsKnown`, `Reading.Predicted` — dictionary-based NER | no — a names dictionary is built in code |
| [typos](typos/main.go) | `Fuzzy`, `FuzzyTop` — typos, е/ё | no — words registered in code |
| [embed](embed/main.go) | `OpenBytes` + `//go:embed` | no — the tiny `names.dat` is committed (regenerate: `go run ./examples/embed/gen`) |
| [contenthash](contenthash/main.go) | `ContentHash` — cache keys | no — dictionaries are built in code |
| [tagmap](tagmap/main.go) | `tagmap.Map` — compare tags across sources | no — two tiny fragments are inlined |
| [pymorphy](pymorphy/main.go) | `OpenPyMorphyDense` | yes — `gomorphy download pymorphy` first |
| [open](open/main.go) | `Open` | yes — a compiled `.dat` (`gomorphy update <type>` first) |

Run any of them directly from the repository root:

```bash
go run ./examples/opencorpora
go run ./examples/unimorph
go run ./examples/multidict
go run ./examples/builder
go run ./examples/importtsv
go run ./examples/merge
go run ./examples/ner
go run ./examples/typos
go run ./examples/embed
go run ./examples/contenthash
go run ./examples/tagmap

gomorphy download pymorphy
go run ./examples/pymorphy -dir .data/pymorphy/data

gomorphy update opencorpora
go run ./examples/open -dat .data/opencorpora/opencorpora.dat
```

Each example is a short, self-contained `package main` — read the source
directly, there's nothing else to it. Examples that need no external data
end with an `// Output:` comment; `go test ./examples/` runs each of them
and checks its output against that comment.
