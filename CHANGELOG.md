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
