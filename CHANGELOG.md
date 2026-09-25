# Changelog

All notable changes to this project are documented here. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and
versioning follows [Semantic Versioning](https://semver.org/).

Per-feature narrative and design rationale live in
[docs/en/implementation/](docs/en/implementation/); this file is a
summary, not a duplicate — see
[docs/en/todo.md](docs/en/todo.md)'s "Completed stages" table for the
full list with links.

## [Unreleased]

## [1.2.1] - 2026-09-26

Godoc, the markdown docs and the examples were reviewed against the 1.2.0
code as a whole, and the defects that review found are fixed (see
[implementation/docs-audit-1.2.1.md](docs/en/implementation/docs-audit-1.2.1.md)).
The `.dat` format is unchanged.

**Upgrading from 1.2.0:**
- `ImportTSV` (and `gomorphy import tsv`) with no entries now fails with
  `ErrNoEntries` instead of returning an empty dictionary.
- The messages of `ErrIncompatibleDictionaries` and `ErrPredictionSharded`
  gained a `morphology: ` prefix; `errors.Is` checks are unaffected, exact
  string comparisons are not.

### Added
- Usage scenarios — what each feature is for, which calls and example
  solve it, since which version and how its behaviour changed:
  [docs/en/scenarios.md](docs/en/scenarios.md),
  [docs/ru/scenarios.md](docs/ru/scenarios.md).
- Examples `ner` (IsKnown/Predicted), `typos` (Fuzzy, е/ё), `embed`
  (`//go:embed` + OpenBytes), `contenthash`, `tagmap`; `go test ./examples/`
  runs every example that needs no data and checks its `// Output:` block.
- `common.NewLoaderLogger` — the logger the download loaders use (see Fixed);
  `common.DownloadFile`, `common.WriteFileAtomic` — atomic download/write.
- `ExampleDictionary_Parse`, `ExampleNewCharPolicy`, `ExampleNoCharPolicy`,
  `ExampleDictionary_ContentHash`, `ExampleMultiDictionary_IsKnown`,
  `tagmap.ExampleMap`, `tagmap.ExampleKnown`.

### Fixed
- `pymorphy.NewLoader`, `opencorpora.NewLoader` and `unimorph.NewLoader` no
  longer panic («logging: set backend first») when the caller never called
  `logging.Init` — the usual case for library code; only the `gomorphy` CLI
  configured it. Without `logging.Init` the loaders now log nothing; with it
  they log as before. Assign the exported `Logger` field to use your own logger.
- Loaders' `Sync` now unpacks a newly downloaded archive over the existing
  copy; before, the old `dict.xml` / pymorphy2 `data/` stayed after an
  update. pymorphy2's `data/` is replaced as a whole, so files of an older
  release don't linger.
- The opencorpora and pymorphy loaders create their files under the
  `dataPath` passed to `NewLoader`; before, they created `./.data/<domain>`
  and failed when a custom `dataPath` did not exist yet.
- Downloads are atomic and check the HTTP status: a failed or interrupted
  download keeps the previous file, and the opencorpora loader no longer
  saves an error page as the archive.
- `Sync(skipDownload=true)` makes no network requests (it used to query
  the source first).
- `Dictionary.Language`, `Fuzzy` and `FuzzyTop` return zero values for a
  nil dictionary instead of panicking, like `Parse` and `Lemma`; a nil
  `MultiDictionary` member behaves as an empty dictionary.
- `ImportTSV` returns `ErrNoEntries` for a stream with no entries, as
  `Builder.Build` does (see Upgrading).
- `ErrIncompatibleDictionaries` and `ErrPredictionSharded` messages start
  with `morphology: ` like the package's other sentinel errors.
- Documentation that contradicted the code: the Russian `library.md` (dense
  dictionaries can be saved and fuzzy-searched; е/ё distance), CLI output
  samples (`fuzzy`/`top` print a dictionary column), the glossary's
  CharPolicy (one-way, default by language), and godoc of `Info`,
  `TagSetName`, `ContentHash`, `FuzzyTop`, `AddForm`, `ImportTSV` and the
  OpenCorpora `Progress` callback.
- `morphology.Version` is `1.2.1`.

### Changed
- `make test-integration` runs with a 60-minute timeout: two packages take
  more than the default 10 minutes under `-race`.

## [1.2.0] - 2026-09-25

Support for dictionary-based NER in the lexicon module (see
[implementation/ner-support.md](docs/en/implementation/ner-support.md)).

**Upgrading from 1.1.0:**
- Rebuild UniMorph dictionaries and Builder/TSV dictionaries built from
  mixed-case input — otherwise their capitalised forms stay unreachable by
  exact lookup (see Fixed).
- Dictionaries built with a non-Russian `Language` and no explicit
  `CharPolicy` no longer get е→ё; pass `RussianCharPolicy()` to keep it.
- `Fuzzy` distances between е and ё drop from 1 to 0 for Russian
  dictionaries; adjust thresholds that relied on the old metric.
- Existing `.dat` files open unchanged; the binary format is the same.

### Added
- `Reading.Predicted`, `LemmaRef.Predicted`: tell a dictionary reading from a
  suffix-prediction guess. `Dictionary.IsKnown` / `MultiDictionary.IsKnown`:
  exact lookup only, never predicts. CLI `lookup` appends `(predicted)`.
- `OpenBytes(data)`: open a dictionary from memory (e.g. `//go:embed`), no mmap,
  works on Windows.
- Public `CharPolicy`, `Substitution`, `NewCharPolicy`, `RussianCharPolicy`,
  `NoCharPolicy`; `BuilderOptions.CharPolicy`; `UniMorphOptions.CharPolicy` is now
  settable from outside the module.
- `Dictionary.ContentHash()`: a digest of the content that ignores the `info`
  section, stable across re-saves.

### Fixed
- `Builder.AddForm`/`AddLemma` and `ImportTSV` lower-case words and lemmas. Before,
  a form added as «Москва» was reachable only as a prediction.
- `Fuzzy`/`FuzzyTop` lower-case the query.
- The UniMorph importer (`unimorph.ImportFromTSV`) now lower-cases lemma and
  wordform text on read, same as Builder/ImportTSV. Before, a mixed-case
  UniMorph row (e.g. «Аббас») was stored verbatim and unreachable by exact
  lookup, since `Parse`/`Fuzzy`/`IsKnown` lower-case their query. **UniMorph
  dictionaries and Builder/TSV dictionaries built by 1.1.0 from mixed-case
  input must be rebuilt** to be reachable by exact lookup — rebuilding is
  the only way to apply this fix to existing `.dat` files.
- A `CharPolicy` with more than 255 substitutions is rejected with an error
  when the dictionary is built (Builder, ImportTSV, the UniMorph importer).
  Before, it was accepted and the process panicked later, on `SaveTo`.

### Changed
- `Fuzzy`/`FuzzyTop` apply the dictionary's CharPolicy: for Russian, е in the
  query matches ё in the dictionary at distance 0 (was 1).
- The default CharPolicy of Builder/ImportTSV depends on the language: е→ё only
  for `Language` "ru" (an empty `Language` means "ru"). A dictionary built with
  any other `Language` and no explicit `CharPolicy` no longer gets the Russian
  е→ё substitution (before, every built dictionary did); pass
  `RussianCharPolicy()` to keep the old behaviour.
- `go.mod` declares `go 1.25.0` (was 1.27.1) with `toolchain go1.27.1`: the
  `go` directive now follows the policy "current Go minus two minor versions".
  Dependencies are unchanged.
- `Close` documents that it must not run concurrently with other calls on the
  dictionary.

## [1.1.0] - 2026-09-23

Build dictionaries from your own wordforms and merge compiled
dictionaries (Stage 19 and 19.1, see
[implementation/stage-19-builder-tsv-merge.md](docs/en/implementation/stage-19-builder-tsv-merge.md)).

### Added
- `morphology.Builder` (`NewBuilder`/`AddForm`/`AddLemma`/`Build`) and
  `morphology.ImportTSV` — build a dictionary from your own wordforms;
  CLI `gomorphy import tsv`.
- `morphology.Merge` / `MergeWithOptions` — structural merge of compiled
  dictionaries (`MergeAdd`/`MergeReplace`, overlays applied in order)
  keeping the base's tag set, probabilities and prediction; CLI
  `gomorphy merge --mode add|replace [--rebuild-prediction]`.
- `tagmap.Known`.
- Internal: value DAWGs (`BuildIntDAWG`, `WalkValues`), `DAWG.Clone`.

### Fixed
- `morphology.Version` (printed by `gomorphy version` and written to
  `BuildInfo.LibraryVersion` on every `SaveTo`) now matches the release;
  1.0.0 still reported `0.1.0`.
- `ImportTSV` rejects a row with an empty wordform column instead of
  silently attaching its tags to an empty word.

## [1.0.0] - 2026-09-19

A complete rewrite of the library from the ground up (see "Changed"
below) into a paradigm+DAWG storage design, three dictionary sources,
and a unified CLI.

### Added

**Core library (`pkg/morphology`)**

- `Parse`/`Lemma`/`Fuzzy`/`FuzzyTop` — exact lookup (sorted by
  probability when available), lemma resolution, and Levenshtein-DFA
  fuzzy/typo search over the DAWG.
- Ending-based prediction for out-of-dictionary words, and
  probability-weighted readings, for pymorphy2-sourced dictionaries.
- `MultiDictionary` — query several open dictionaries at once,
  aggregating `Parse`/`Lemma` results with per-reading dictionary
  attribution.
- A sectioned, mmap-backed binary format (`GMOR`): `Open`/`SaveTo`
  round-trip, `meta`/`tagset`/`prefixes`/`suffixes`/`paradigms`/
  `words.dawg`/`prediction`/`probability`/`alphabet` sections, shard
  support for large dictionaries.
- A dense 1-byte DAWG alphabet, on by default for every CLI-built
  dictionary across all three sources (`~36%` smaller `words.dawg`
  on the reference data) — raw/non-dense stays available from the Go
  API for anyone who needs it.

**Dictionary sources and CLI**

- Three importers, all reachable from both the library and the CLI:
  pymorphy2 (`OpenPyMorphy`), OpenCorpora `dict.xml`
  (`CompileFromXML`), and UniMorph TSV (`CompileFromUniMorph`).
- The `gomorphy` CLI: `lookup`/`lemmas`/`fuzzy`/`top`/`cli`
  (interactive mode) for queries, `download`/`unpack`/`build`/`update`
  for fetching and compiling any of the three sources (folding the
  earlier separate `opencorpora_update`-style tooling into one
  binary).

**Cross-dictionary tooling**

- `pkg/morphology/tagmap` — normalizes any of the three sources' native
  tags into a common UniMorph-schema feature bundle
  (`tagmap.Map(dictName, tag)`), so tags from different dictionaries
  can be compared for the same grammatical meaning.
- `Dictionary.TagSetName()`/`MultiDictionary.DictTagSetName(i)` — the
  `dictName` `tagmap.Map` needs, reachable without out-of-band
  knowledge of which importer built a dictionary.

### Fixed

Found and fixed during the rewrite (see
[implementation/code-review-pre-1.0-triage.md](docs/en/implementation/code-review-pre-1.0-triage.md)
and the linked design docs for the full analysis of each):

- A DAWG minimization bug (`chainSig`) that inflated `.dat` size by
  roughly 29x.
- A suffix-count overflow that could corrupt sharded dictionaries.
- Corrupted wordform tags under certain paradigm-dedup conditions.
- Comparative-degree paradigms ("по-" prefixed forms, e.g.
  "покрасивее") not merging into their base adjective's paradigm.

### Changed

- **Full rewrite.** Everything user-facing changed from the pre-1.0.0
  `v0.1.0` prototype: the old `internal/index` custom trie format and
  `pkg/dictionary` facade are gone, replaced by the paradigm+DAWG
  design in `pkg/morphology`; the old `opencorpora_update`/
  `opencorpora_test` command pair is gone, replaced by the single
  `gomorphy` CLI. There is no supported migration path from `v0.1.0`
  data or API — rebuild your dictionary with `gomorphy build`.

## [0.1.0] - 2023-11-30

Initial prototype: an OpenCorpora-only analyzer (`pkg/dictionary`) over
a custom trie format (`internal/index`), with separate
`opencorpora_update`/`opencorpora_test` command-line utilities.
Superseded entirely by 1.0.0's redesign (see "Changed" above).
