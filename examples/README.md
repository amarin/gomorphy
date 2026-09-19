# Examples

Minimal, runnable programs for each `pkg/morphology` entry point. See
[docs/en/library.md](../docs/en/library.md) for the full API reference
these examples are drawn from.

| Example | Entry point | Needs external data? |
|---|---|---|
| [opencorpora](opencorpora/main.go) | `CompileFromXML` | no — a tiny `dict.xml` fragment is inlined |
| [unimorph](unimorph/main.go) | `CompileFromUniMorph` | no — a tiny TSV fragment is inlined |
| [multidict](multidict/main.go) | `NewMultiDictionary` | no — two tiny `dict.xml` fragments are inlined |
| [pymorphy](pymorphy/main.go) | `OpenPyMorphyDense` | yes — `gomorphy download pymorphy` first |
| [open](open/main.go) | `Open` | yes — a compiled `.dat` (`gomorphy update <type>` first) |

Run any of them directly from the repository root:

```bash
go run ./examples/opencorpora
go run ./examples/unimorph
go run ./examples/multidict

gomorphy download pymorphy
go run ./examples/pymorphy -dir .data/pymorphy/data

gomorphy update opencorpora
go run ./examples/open -dat .data/opencorpora/opencorpora.dat
```

Each example is a self-contained `package main` under 50 lines — read
the source directly, there's nothing else to it.
