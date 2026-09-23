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

## Structural merge (2026-09-23)

The entries-based `Merge` above (the first "What shipped" bullet) was
replaced by an id-preserving structural merge. A 2026-09-22 review ran the
entries-based merge on the real bundled dictionaries (base + a 3-line TSV
overlay, `--mode replace`) and found it unusable:

| Base | Time | Peak RSS | Size before → after |
|---|---|---|---|
| `pymorphy.dat` | 138 s | 2.8 GB | 16 MB → 41 MB |
| `opencorpora.dat` | 141 s | 2.7 GB | 10 MB → 41 MB |

Regrouping every word by `(word, lemma, tag)` triples and rebuilding from
scratch (`buildFromEntries`) threw away pymorphy2's prefix paradigms and
shattered prediction (`Parse("бутявкающий")` went from 5 readings to none),
dropped probability (`p_t_given_w`, so `стали` reordered `стать`/`сталь`),
renamed the TagSet to `merge` (breaking `tagmap`), and behaved
inconsistently when folding multiple overlays.

Design: [2026-09-22-structural-merge-design.md](../superpowers/specs/2026-09-22-structural-merge-design.md).
Plan: [2026-09-22-structural-merge.md](../superpowers/plans/2026-09-22-structural-merge.md).

The fix, in brief:

- The base's `TagSet`, `Prefixes`, and each shard's `Suffixes`/`Paradigms`
  are copied **verbatim, ids unchanged**; overlay paradigms are remapped
  form-for-form into that id space and deduplicated against the target
  shard, instead of being flattened and rebuilt from text.
- Because base ids never move, the base's prediction DAWGs and probability
  DAWG stay valid as-is and are cloned rather than rebuilt (fast path);
  only shards whose contents actually changed have their words DAWG
  rebuilt.
- The output TagSet keeps the **base's name** (fixes `tagmap`), and
  overlays with a different *known* TagSet name (`opencorpora`,
  `opencorpora-int`, `unimorph`) are rejected with
  `ErrIncompatibleDictionaries` rather than silently mixed.
- Overlays fold in order for both modes: `MergeAdd` only takes a word that
  is absent from the base **and** every earlier overlay; `MergeReplace`
  lets the **last** overlay win.
- An optional `RebuildPrediction` (CLI: `gomorphy merge
  --rebuild-prediction`) rebuilds a fresh prefix-0 prediction DAWG from the
  merged shard 0, for merges between thematic dictionaries where the base
  has no prediction of its own worth preserving; it errors
  (`ErrPredictionSharded`) if the output ends up sharded.
- Inputs stay untouched: everything carried over is deep-copied
  (`(*DAWG).Clone()`), so the result outlives `Close`d, mmap'd inputs.

### Measurements (Task 9, 2026-09-23)

Real-dictionary golden test (`GOMORPHY_MERGE_BASE=<.dat>`, add-mode merge
of a small overlay, 20 000 sampled base words + 6 OOV probes — all
identical base vs. merged):

| Dictionary | Merge time | Size before → after |
|---|---|---|
| pymorphy | 25.06 s | 16,463,396 → 16,463,124 B (-0.00%) |
| opencorpora | 24.95 s | 10,669,636 → 10,669,604 B (-0.00%) |
| unimorph | 3.53 s | 11,150,107 → 11,149,347 B (-0.01%) |

CLI end-to-end measurements (`gomorphy merge --mode replace`, base + a
1-line TSV overlay), before/after against the old entries-based merge:

| Metric | pymorphy — old merge | pymorphy — structural | opencorpora — old merge | opencorpora — structural |
|---|---|---|---|---|
| Time | 138 s | 24.51 s | 141 s | 25.16 s |
| Peak RSS | 2.8 GB | 4.62 GB | 2.7 GB | 4.61 GB |
| Size before → after | 16 MB → 41 MB | 16,463,396 → 16,463,604 B (+0.001%) | 10 MB → 41 MB | 10,669,636 → 10,669,996 B (+0.003%) |

Time and size both improved sharply (5-6x faster; size effectively flat
instead of ×2.5-4). Peak RSS went **up** relative to the old merge — the
old merge's 2.8 GB came from building six small sharded DAWGs instead of
one; the structural merge still has to rebuild the (large) words DAWG for
any shard it touches, and that rebuild (`BuildDAWGWithValues` /
`dawgBuilder.newNode`, ~57% of allocated space in profiling) is the
dominant memory cost. See "RSS budget" below for the revised ceiling this
was measured against.

### RSS budget (revised 2026-09-23)

The design spec's original acceptance criterion was peak RSS ≤ 1.5 GB on
`pymorphy.dat`. That number was an unmeasured guess. A controller
measurement of `gomorphy build pymorphy` from source (i.e. just building
the base, no merge at all) shows it alone peaks at **24.3 s / 3.5 GB
RSS** — the DAWG builder is the floor, and no merge that rebuilds a words
DAWG can land under an absolute 1.5 GB ceiling.

The budget is revised to: **merge peak RSS ≤ 1.5× the base's own build
RSS**. Measured: 4.6 GB on a 3.5 GB base ≈ **1.3×** — within the revised
budget. Reducing DAWG-builder memory itself (so both `build` and `merge`
get cheaper) is a backlog item, not part of this stage — see
[todo.md](../todo.md), "DAWG builder memory".

## Verification (done)

See todo.md Stage 19 "Automated checks" for the green test list
(`builder_test.go`, `import_tsv_test.go`, `merge_test.go`,
`internal/*_test.go`, CLI `import_test.go`/`merge_test.go`); the full
suite runs clean under `go test -race ./...`. Cross-links:
[dictionary.go](../../pkg/morphology/dictionary.go) — the facade this
stage extends; [unimorph.md](../unimorph.md) — the other source-agnostic
importer; [mcp.md](../mcp.md) — why a built-in MCP server is not coming.