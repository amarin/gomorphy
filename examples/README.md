# Examples

Minimal, runnable programs for each `pkg/morphology` entry point. See
[docs/en/library.md](../docs/en/library.md) for the full API reference
these examples are drawn from.

| Example | Entry point | Needs external data? |
|---|---|---|
| [opencorpora](opencorpora/main.go) | `CompileFromXML` | no — a tiny `dict.xml` fragment is inlined |
| [unimorph](unimorph/main.go) | `CompileFromUniMorph` | no — a tiny TSV fragment is inlined |
| [multidict](multidict/main.go) | `NewMultiDictionary` | no — two tiny `dict.xml` fragments are inlined |
| [builder](builder/main.go) | `NewBuilder` | no — wordforms registered in code |
| [importtsv](importtsv/main.go) | `ImportTSV` | no — a tiny TSV stream is inlined |
| [merge](merge/main.go) | `Merge` | no — two tiny dictionaries are built in code |
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

gomorphy download pymorphy
go run ./examples/pymorphy -dir .data/pymorphy/data

gomorphy update opencorpora
go run ./examples/open -dat .data/opencorpora/opencorpora.dat
```

Each example is a self-contained `package main` under 50 lines — read
the source directly, there's nothing else to it.
