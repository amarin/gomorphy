# Stage 19. Builder API, TSV import, Merge — DONE 2026-09-22

Thematic dictionaries with no external resources: a public `Builder`
API, a TSV importer, and a `Merge` helper, plus the `gomorphy import tsv`
and `gomorphy merge` CLI commands. What shipped is the core; the import
report, batch query modes, a "thematic dictionary" SKILL.md and the CLI
`--summary` flag stay open (see [todo.md](../todo.md), Stage 19).

> **Status: DONE, 2026-09-22.** Scope: pkg/morphology Builder + TSV
> import + Merge; cmd/gomorphy `import tsv` and `merge`. Design spec:
> [2026-09-21-builder-tsv-merge-design.md](../superpowers/specs/2026-09-21-builder-tsv-merge-design.md).

## What shipped

- **Public `Builder`** (`pkg/morphology/builder.go`) — `NewBuilder` /
  `AddForm` / `AddLemma` / `Build`; `BuilderOptions{Language, Source}`;
  sentinels `ErrNoEntries` and `ErrBuilderClosed`. A builder is
  single-use: `Build` consumes it, a second call returns
  `ErrBuilderClosed`. Underneath sits the shared unexported helper
  `buildFromEntries` (same entry-triples pipeline as `ImportTSV`).
- **`ImportTSV`** (`pkg/morphology/import_tsv.go`) — `ImportTSV(r
  io.Reader, opts BuilderOptions) (*Dictionary, error)`. Words are fed
  in as `lemma<TAB>wordform[<TAB>tags]`; blank lines and `#`-comments
  are skipped, fields are trimmed per-field, a missing lemma makes the
  wordform its own lemma, and line-numbered errors are returned for
  malformed rows. Tags are registered verbatim (never normalized or
  lower-cased); `Source` defaults to `"tsv"`. There is deliberately no
  `ImportTSVFile` — the CLI opens the file and passes the reader.
- **`Merge`** (`pkg/morphology/merge.go`) — `Merge(base, overlays,
  mode)` combines dictionaries without mutating the inputs. `MergeAdd`
  keeps an existing base wordform's readings and adds only words new to
  the base; `MergeReplace` swaps in the overlay's readings for words
  present in both. The output's `BuildInfo.Source` is `"merge"`; the
  base contributes language, `SourceVersion` and `Description`.
- **CLI** (`cmd/gomorphy`) — `gomorphy import tsv <file> [-o out.dat]
  [--source name]` (`-o` required, `--source` defaults to `tsv`) and
  `gomorphy merge --mode add|replace -o out.dat base overlay ...` (`-o`
  required and may not alias any input). `split` remains a stub.

## Raw-to-post-process split

A built dictionary goes through two stages, copied from the importers:

1. **Raw build.** `buildFromEntries` runs the shared pipeline
   (`BuildDictionaryFromEntries`) over the entry triples: group by lemma,
   fix form 0 to the lemma itself, compute the LCP stem, shard with
   FillOnDemand (see the suffix-sharding spec, same limits). The result is
   raw: no Alphabet, no Prediction, no Info.
2. **Post-process.** `buildFromEntries` then rebuilds prediction
   (`BuildPrediction`, dense) and re-encodes the DAWGs onto one dense
   1-byte alphabet shared across all shards (`RecompileDense`), and stamps
   `BuildInfo`. Prediction is rebuilt for every build/TSV/merge that stays
   single-shard (BuildPrediction is a no-op otherwise); probability is
   never rebuilt — it is external corpus-frequency data a self-made
   dictionary does not have, exactly like OpenCorpora dictionaries today.

## Prediction order

`Build` runs `BuildPrediction` on the **raw** dictionary, *before*
`RecompileDense` — its documented precondition: the dictionary must be
unsharded and still raw (plain-text wordform keys), so it is a no-op
when `len(d.Words) != 1`. For every reading whose tag is productive it
indexes the wordform's last 1..5 runes as suffix keys (matching
`parse.go`'s `suffixSplits(word, 5)`), accumulating an attestation count
per (suffix, paradigm, form) triple. `productive` is injected as a
callback because it lives in the engine package (`parse.go`), which
`internal/` must not import; building a dictionary from entries should
not drag morphological parsing logic in.

## What the caller decides

`BuilderOptions.Language` (default `"ru"`) and `Source` (default
`"builder"`, `"tsv"` for the importer) are caller-controlled and land in
`BuildInfo`; `TagSet` names come from the same options (`builder`/`tsv`/
`merge`). `Merge` inherits the base's language and CharPolicy — overlays
are expected to match it.

## Verification (done)

See todo.md Stage 19 "Automated checks" for the green test list
(`builder_test.go`, `import_tsv_test.go`, `merge_test.go`,
`internal/*_test.go`, CLI `import_test.go`/`merge_test.go`); the full
suite runs clean under `go test -race ./...`. Cross-links:
[dictionary.go](../../pkg/morphology/dictionary.go) — the facade this
stage extends; [unimorph.md](../unimorph.md) — the other source-agnostic
importer; [mcp.md](../mcp.md) — why a built-in MCP server is not coming.