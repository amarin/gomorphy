# Stage 18. Finalization: documentation, tests

> **Update**: the "CLI" section below describes the original plan
> (`-dict` flags, `import pymorphy2/opencorpora/unimorph`,
> tab-completion) — it was **not implemented literally**, but the CLI
> was fully reworked separately (Stage 21, `gomorphy` on cobra) and
> closes the same underlying need differently: commands
> `lookup`/`lemmas`/`fuzzy`/`top`/`cli`/`download`/`unpack`/`build`/`update`,
> a `-d/--dictionary` flag (can be given several times — merged into a
> `MultiDictionary`). The current CLI description is
> [docs/cli.md](../cli.md), the redesign history is
> [pymorphy-source-and-cli.md](pymorphy-source-and-cli.md). The "CLI"
> section below is kept as-is for planning history, not as the current
> spec. What's left of this stage is documentation, tests, metrics; see
> "Final metrics" below, updated with facts.

## Stage contents (original plan, partially stale — see the note above)

Updating the CLI (every command via arguments + an interactive mode),
finalizing documentation, a full test run, updating the README.

### CLI (`cmd/gomorphy/main.go`) — original plan, see the note above

Every command is available via CLI arguments:

```bash
# Exact lookup
gomorphy -dict <path> lookup <word>

# Base form (lemma)
gomorphy -dict <path> lemma <word>

# Fuzzy search
gomorphy -dict <path> fuzzy <word> <maxDist>
gomorphy -dict <path> top <word> <maxWords>

# Import
gomorphy import pymorphy2 <dir> -o <path>
gomorphy import opencorpora <dict.xml> -o <path>
gomorphy import unimorph <rus.tsv> -o <path>

# Interactive mode
gomorphy -dict <path>
# > parse кота
# > lemma котам
# > fuzzy кот 1
# > top кот 3
# > import pymorphy2 /path/to/dir
# > help
```

### Multi-dict at the CLI level — original plan, see the note above

Implemented more broadly than planned: `-d/--dictionary` can be given
several times in one command — the CLI merges them into a single
`MultiDictionary` itself, with no need for separate calls from the
user. See [docs/cli.md](../cli.md) and [multi-dict.md](multi-dict.md).

### Documentation — DONE 2026-09-17

- `README.md`: the description, Quick start, Project structure —
  brought in line with the current code (was: the old `pkg/dictionary`
  architecture, a `.dat` size of ~300 MB; now: `pkg/morphology`, ~13 MB).
- `docs/library.md`: fully rewritten for the current `pkg/morphology`
  API (`Parse`/`Lemma`/`Fuzzy`/`FuzzyTop`/`MultiDictionary`/`SaveTo`) —
  the old version described an API that no longer exists
  (`dictionary.Open`, `Wordform`, `Builder.AddGrammeme`).
- `docs/cli.md` — already up to date, unchanged.
- `docs/index.md` — links added for the new `implementation/*.md` files
  (multi-dict, the dense alphabet, the pymorphy2 source, tag-mapping),
  a "Usage" section added, the research list expanded.
- godoc: an audit of exported symbols in `pkg/morphology` and its
  subpackages (`tagmap`, `importers/*`), `pkg/pymorphy`,
  `pkg/opencorpora`, `pkg/common` — all documented; dead code with zero
  usages was found and removed along the way (`pkg/common/interfaces.go`).
- Examples (`examples/`), an agent skill (`skills/use-dictionary/`) —
  **split out of this stage as a separate future task**, see
  `docs/en/todo.md` — this is new content with its own design, not a
  documentation fix.

### cmd/opencorpora_update — original plan, see the note above

Implemented differently: `gomorphy update <type>`
(`opencorpora`/`pymorphy`, with no `--skip-download` flag — for "only
compile from a local file" there's a separate command
`gomorphy build <type> -i <path>`). See [docs/cli.md](../cli.md).

## Verification (tests)

Run and confirmed (see `git log` on this file for the date of the last check):

- `go build ./...` — no errors.
- `go vet ./...` — no findings.
- `go test -race -count=1 ./...` — 274/274 green, 12 packages.
- `golangci-lint run ./...` — no findings (in the course of this
  finalization, 2 long-standing known findings were found and fixed —
  an errcheck on `defer opened.Close()` in a test, a staticcheck S1016
  on a manual struct literal instead of a type conversion in
  `pkg/morphology/importers/opencorpora/import.go` — both had been
  called "pre-existing, unrelated" in the multi-dict spec; closed as
  part of this stage).
- Dead code found and removed along the way:
  `pkg/common/interfaces.go` (`Dictionary`/`DomainDataLoader` —
  exported interfaces with zero usages anywhere in the codebase,
  a pre-redesign leftover).
- Integration tests (`-tags=integration`, need real dictionary data) —
  not run in CI/by default; see `docs/en/todo.md` for the environment
  variables (`GOMORPHY_DICT_XML`, `GOMORPHY_PYMORPHY2_DIR`).

## Manual verification

Current commands — see [docs/cli.md](../cli.md). A full cycle on real data:

```bash
gomorphy update pymorphy
gomorphy -d .data/pymorphy/pymorphy.dat lookup кота
gomorphy -d .data/pymorphy/pymorphy.dat lemmas кота
gomorphy -d .data/pymorphy/pymorphy.dat fuzzy кот 1
gomorphy -d .data/pymorphy/pymorphy.dat top кот 3

gomorphy update opencorpora
gomorphy -d .data/opencorpora/opencorpora.dat lookup кота

# Several dictionaries as one index
gomorphy -d .data/opencorpora/opencorpora.dat -d .data/pymorphy/pymorphy.dat lookup кота

gomorphy cli -d .data/opencorpora/opencorpora.dat   # interactive console
```

The UniMorph cycle (`gomorphy import unimorph .../update unimorph`) is
unavailable — Stage 16 hasn't started.

Full shell autocompletion (bash/zsh for the `gomorphy` command itself)
is **not implemented**; there's only command-name completion inside the
`gomorphy cli` interactive console (TAB — `lookup`/`lemmas`/`fuzzy`/
`top`/`exit`/`quit`, see [docs/cli.md](../cli.md)).

## Final metrics

> Update 2026-09-17: the table below was replaced with actually
> measured facts wherever measurement is possible; unmeasured,
> speculative expectations ("expected at planning time") were removed
> as unreliable rather than left next to the fact. Operation latency
> (Parse/Lemma/Fuzzy) is **not benchmarked** anywhere in the codebase —
> there isn't a single `Benchmark*` function; the rows below are
> honestly marked as unmeasured rather than filled in with an eyeballed
> guess.

| Metric | Value | How it was measured |
|---|---|---|
| `.dat` size (OpenCorpora, full dictionary) | ~13.3 MB | `ls -la .data/opencorpora/opencorpora.dat`, 2026-09-17 |
| `.dat` size (pymorphy2, full dictionary) | ~14.0 MB | `ls -la .data/pymorphy/pymorphy.dat`, 2026-09-17 |
| Source `dict.xml` (OpenCorpora) | ~401 MB | for comparison — how much more compact the `.dat` is |
| `.dat` size with zstd | — | not implemented, see `docs/en/todo.md`, Stage 17 remainder |
| Loading (`Open`) | mmap, no full in-memory read | architectural (mmap-backed sections), not benchmarked in us/ms |
| Parse/Lemma/Fuzzy latency | — | **not benchmarked** — no `Benchmark*` in the codebase |
| pymorphy2 support | yes | `OpenPyMorphy`/`OpenPyMorphyDense` |
| OpenCorpora support | yes | `CompileFromXML(File)` |
| UniMorph support (TSV) | no | Stage 16 hasn't started, see `docs/en/todo.md` |
| Multiple dictionaries | yes, from the library (`MultiDictionary`) | not "at the application level" — now part of the API |
| Tests | 274/274, `-race`, 12 packages | `go test -race -count=1 ./...`, 2026-09-17 |
| Linter | no findings | `golangci-lint run ./...`, 2026-09-17 |
