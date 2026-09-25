# Roadmap

Stages 0-15 are done. Details for each stage are in
[implementation.md](implementation.md) and the individual files under
[implementation/](implementation/); the same place holds the storage
redesign rationale (stages 11-18: CSR-trie + exact-hash + pairs ->
paradigms + DAWG), unplanned but significant findings and features
discovered along the way (DAWG build speedup, DAWG minimization fix,
critical pre-1.0 review bugs, multi-dict, the dense DAWG alphabet for
pymorphy2 and OpenCorpora, the pymorphy2 source in the CLI, tag mapping
between dictionaries), and the full path-to-1.0.0 release history.

This file (`todo.md`) is only what's still ahead: the current path to
version 1.0.0, unfinished/future stages, and open ideas. Everything
that's done lives in `implementation/` and isn't duplicated here.

## Formatting requirements

Each stage is a self-contained increment with a verifiable result. Any
agent can pick up where a previous one left off: after each stage the
repository builds (`go build ./...`), and the stage's tests are green.
Mark completed items with `[x]`.

## Completed stages

A short status with links to details — the write-ups themselves live in
`implementation/` and aren't repeated here.

| Stage | Status | Details |
|---|---|---|
| 0. Repository analysis and preparation | DONE | [implementation/stage-0-analysis.md](implementation/stage-0-analysis.md) |
| 1. Format primitives (internal/format) | DONE | [implementation/stage-1-format.md](implementation/stage-1-format.md) |
| 2. String interning | DONE | [implementation/stage-2-intern.md](implementation/stage-2-intern.md) |
| 3. dict.xml scanner (internal/xmlscan) | DONE | [implementation/stage-3-xmlscan.md](implementation/stage-3-xmlscan.md) |
| 4. Builder and CSR structures | DONE | [implementation/stage-4-builder-csr.md](implementation/stage-4-builder-csr.md) |
| 5. Compiler and file loader | DONE | [implementation/stage-5-compiler-loader.md](implementation/stage-5-compiler-loader.md) |
| 6. Public facade pkg/dictionary | DONE | [implementation/stage-6-facade.md](implementation/stage-6-facade.md) |
| 7. End-to-end OpenCorpora integration | DONE | [implementation/stage-7-opencorpora.md](implementation/stage-7-opencorpora.md) |
| 8. FT5 lemma lookup | DONE | [implementation/stage-8-lemmas.md](implementation/stage-8-lemmas.md) |
| 9. FT6 fuzzy search | DONE | [implementation/stage-9-fuzzy.md](implementation/stage-9-fuzzy.md) |
| 10. Finalization (first implementation) | DONE | [implementation/stage-10-finalize.md](implementation/stage-10-finalize.md) |
| — (storage redesign rationale, stages 11-18) | — | [implementation/redesign-rationale.md](implementation/redesign-rationale.md) |
| 11. Internal format: TagSet + Paradigm + DAWG reader | DONE | [implementation/stage-11-internal-format.md](implementation/stage-11-internal-format.md) |
| 12. PyMorphy2 import | DONE | [implementation/stage-12-import-pymorphy2.md](implementation/stage-12-import-pymorphy2.md) |
| 13. Public API: Parse, Lemma, Fuzzy | DONE | [implementation/stage-13-public-api.md](implementation/stage-13-public-api.md) |
| 14. Serialization: unified on-disk format | DONE | [implementation/stage-14-serialization.md](implementation/stage-14-serialization.md) |
| 15. OpenCorpora import | DONE | [implementation/stage-15-import-opencorpora.md](implementation/stage-15-import-opencorpora.md) |
| — DAWG build speedup (free list instead of O(n^2)) | DONE | [implementation/dawg-freelist-optimization.md](implementation/dawg-freelist-optimization.md) |
| — DAWG minimization fix (chainSig bug, ~29x .dat size) | DONE | [implementation/dawg-minimization-fix.md](implementation/dawg-minimization-fix.md) |
| 17. Narrowing ID types + format groundwork for compression + `info` section | PARTIAL (see below) | [implementation/stage-17-optimize.md](implementation/stage-17-optimize.md), [implementation/info-section.md](implementation/info-section.md) |
| — Pre-1.0.0 code review: findings triage + both critical bugs (suffix overflow, corrupted wordform tags) | DONE | [implementation/code-review-pre-1.0-triage.md](implementation/code-review-pre-1.0-triage.md) |
| — Multi-dict: `morphology.MultiDictionary` | DONE | [implementation/multi-dict.md](implementation/multi-dict.md) |
| — Dense 1-byte DAWG alphabet for `words.dawg` (pymorphy2, then generalized to OpenCorpora) | DONE (`.dat` serialization, `fuzzy.go`, Prediction/Probability, and the CLI default all closed 2026-09-19; the 2-byte read-path item was dropped, no consumer) | [implementation/pymorphy2-dense-alphabet.md](implementation/pymorphy2-dense-alphabet.md) |
| — pymorphy2 source (`pkg/pymorphy`) + integration into the `gomorphy` CLI | DONE | [implementation/pymorphy-source-and-cli.md](implementation/pymorphy-source-and-cli.md) |
| — Universal tag mapping between dictionaries (`pkg/morphology/tagmap`, native -> universal) | DONE (partial, see remaining backlog below) | [implementation/tag-mapping.md](implementation/tag-mapping.md) |
| — Stage 18 (documentation + tests + godoc audit; the CLI part was closed separately, see above) | DONE 2026-09-17 | [implementation/stage-18-finalize.md](implementation/stage-18-finalize.md) |
| 16. UniMorph import (importer, loader, public API, CLI, dense by default) | DONE 2026-09-19 | [implementation/stage-16-import-unimorph.md](implementation/stage-16-import-unimorph.md) |
| 19.1. Structural merge (id-preserving `Merge`, replaces the entries-based one) | DONE 2026-09-23 | [implementation/stage-19-builder-tsv-merge.md](implementation/stage-19-builder-tsv-merge.md#structural-merge-2026-09-23) |
| — NER support for lexicon: 1.2.0 (known-word flag, OpenBytes, case/ё, CharPolicy, ContentHash, go 1.25) | DONE | [implementation/ner-support.md](implementation/ner-support.md) |

## Unfinished/future stages

### Documentation audit after 1.2.0 (markdown + godoc) — DONE 2026-09-25, for release 1.2.1

**Why:** 1.2.0 shipped without a dedicated documentation pass (owner note
2026-09-25). The NER-support work touched the public API in many places,
and docs were updated task by task, not reviewed as a whole. Write-up:
[implementation/docs-audit-1.2.1.md](implementation/docs-audit-1.2.1.md).

- [x] godoc: package overview covers 1.2.0; doc/code contradictions fixed.
- [x] Runnable examples: new `examples/{ner,typos,embed,contenthash,tagmap}`,
      new `ExampleXxx` for Parse/Predicted, CharPolicy, ContentHash,
      MultiDictionary.IsKnown, tagmap; `go test ./examples/` checks every
      example's `// Output:` block.
- [x] Markdown (README, index, library, cli, installation, comparison,
      glossary, requirements, implementation, mcp) consistent with 1.2.0.
- [x] Russian docs mirror the English ones they translate (`library.md`
      and `cli.md` were substantially behind and partly wrong).
- [x] New: usage scenarios (`docs/en/scenarios.md`, `docs/ru/scenarios.md`)
      — what each feature is for, how, which example, since which version
      and how its behaviour changed (a per-feature excerpt of CHANGELOG).
- [x] Added to the same release: the download loaders panicked without
      `logging.Init` — fixed (`common.NewLoaderLogger`).
- [x] Code defects found by the audit (loaders: refresh, dataPath, HTTP
      status, offline `Sync(true)`, atomic downloads; nil receivers;
      `ImportTSV` empty input; sentinel prefixes) — fixed in 1.2.1, see the
      write-up.

### Stage 19.1 — Structural merge — DONE 2026-09-23

**Why:** the Stage 19 `Merge` rebuilds its output from `(word, lemma,
tag)` triples. A review on 2026-09-22 merged a 3-line overlay into the
real dictionaries and found that on a real base it loses prediction
(`бутявкающий` → no readings), probability (reading order changes:
`стали` → `сталь` first) and the TagSet name (`merge`, so `tagmap` stops
working). It also costs 138 s / 2.8 GB RSS / 16 → 41 MB. Multiple overlays
don't behave as a fold, and the RU docs/README/CHANGELOG are stale.

**What:** an id-preserving structural merge. The base's tags, prefixes,
suffixes and paradigms keep their ids, overlay paradigms are remapped
into them, and only changed shards' word DAWGs are rebuilt. The base
prediction and probability therefore stay valid. Design:
[2026-09-22-structural-merge-design.md](superpowers/specs/2026-09-22-structural-merge-design.md);
plan: [2026-09-22-structural-merge.md](superpowers/plans/2026-09-22-structural-merge.md).

Tasks (plan order; review finding # in brackets):

- [x] 0. Commit the pending `ImportTSV` empty-wordform fix
- [x] 1. `tagmap.Known` [#3]
- [x] 2. DAWG primitives: `BuildIntDAWG`, `WalkValues`, `Clone`, `Empty` [#1, #2]
- [x] 3. `DenseAlphabet.Runes`, `BuildPredictionFrom` [#1]
- [x] 4. Engine: overlay decision with fold semantics [#5]
- [x] 5. Engine: id-preserving paradigm remap, target shard [#1, #4]
- [x] 6. Engine: `MergeDictionaries` — shard reuse/rebuild, alphabet, prediction, probability [#1, #2, #4]
- [x] 7. Public `Merge`/`MergeWithOptions`, compatibility checks, tests [#1–#3, #5]
- [x] 8. CLI `merge --rebuild-prediction`
- [x] 9. Gated real-dictionary golden test and measurements (budget: pymorphy ≤ 40 s, ≤ 1.5× base build RSS, size ≤ +2%) [#4]
- [x] 10. EN docs, godoc, `ExampleMergeWithOptions` [#6]
- [x] 11. RU docs (`cli.md`, `library.md`), README, CHANGELOG `[Unreleased]` [#6]
- [x] 12. Write-up and roadmap close-out

### Stage 17 — remainder: zstd compression + a tagset encoding candidate

Narrowing ID types, format groundwork for compression, the density
analysis of DAWG packing/`tagset` encoding, and the dense 1-byte
`words.dawg` alphabet (for both pymorphy2 and OpenCorpora) are already
done (see the table above,
[docs/research/0001-dawg-alphabet-density.md](research/0001-dawg-alphabet-density.md),
[docs/research/0002-paradigm-tagset-binary-encoding.md](research/0002-paradigm-tagset-binary-encoding.md),
and [implementation/pymorphy2-dense-alphabet.md](implementation/pymorphy2-dense-alphabet.md)).
What's left — separate future tasks, **not blocking 1.0.0**, in priority
order:

1. zstd for the cold suffixes/prefixes/tagset/paradigms sections
   (`klauspost/compress`, maximum compression level).
2. A grammeme dictionary + index lists instead of JSON for the `tagset`
   section — found a ~78.2% effect on the section (~1.09% of the whole
   file) on a real dictionary, with no change to the hot read path.
   Low-risk implementation, no open applicability questions. Details in
   [implementation/stage-17-optimize.md](implementation/stage-17-optimize.md#tagset-encoding-analysis--done-a-backlog-candidate-exists).

Details and what's already done are in
[implementation/stage-17-optimize.md](implementation/stage-17-optimize.md).

### DAWG builder memory (post-1.0 backlog)

Building or merging a full-size Russian dictionary (pymorphy/opencorpora,
~5 M (word, value) pairs) peaks at **3.5–4.6 GB RSS**, dominated by
`BuildDAWGWithValues`/`dawgBuilder.newNode` in `internal/`. `gomorphy
build pymorphy` from source alone peaks at 3.5 GB/24.3 s (2026-09-23), and
`merge` on the same base adds only ~1.3× on top of that floor (see
[implementation/stage-19-builder-tsv-merge.md](implementation/stage-19-builder-tsv-merge.md#rss-budget-revised-2026-09-23)).
Not blocking 1.0.0; worth revisiting (e.g. streaming/chunking the words-DAWG
build) since it's the cost floor for both `build` and `merge`.

### Universal tag mapping: remaining backlog (not blocking 1.0.0)

The `native -> universal` direction is implemented (see the table
above). One item remains — details in
[implementation/tag-mapping.md](implementation/tag-mapping.md):

1. **`Unmap` (universal -> native)** — needed for dictionary export,
   see "Dictionary export" below; deliberately deferred until a
   concrete consumer appears (the ambiguity of the reverse mapping
   needs to be resolved against a specific exporter, not guessed at
   ahead of time).

~~`TagSet.Name` was unreachable from `pkg/morphology`'s public API~~ —
DONE 2026-09-19: `Dictionary.TagSetName()`/
`MultiDictionary.DictTagSetName(i)` (mirroring `Info`/`DictInfo`) now
give `tagmap.Map`'s `dictName` directly, without out-of-band knowledge
of which importer built a dictionary.

~~A `tagmap` `"unimorph"` source table~~ — DONE 2026-09-19: turned out
exactly as trivial as expected (see
[implementation/tag-mapping.md](implementation/tag-mapping.md)'s "The
UniMorph table"), with one real finding along the way — the real rus
data doesn't carry animacy/gender for every noun paradigm slot the way
OpenCorpora's tags do, a genuine data-coverage gap rather than a
mapping bug.

### Dictionary export: pymorphy2 / OpenCorpora — NOT STARTED, feasibility assessed

Feasibility assessment (no implementation) —
[docs/research/0008-dictionary-export-feasibility.md](research/0008-dictionary-export-feasibility.md).
Recommended order if this work is picked up:

1. **pymorphy2 export for pymorphy2-origin dictionaries (round-trip)** —
   low risk, all the serialization infrastructure (`DAWG.Bytes()`,
   `Paradigm.Data()`) already exists and is tested. A self-contained
   task, blocked by nothing.
2. **Universal export (any dictionary -> pymorphy2)** — blocked by
   `tagmap.Unmap` (see above) and needs a decision on suffix sharding
   and the dense alphabet when exporting dictionaries not of
   pymorphy2 origin.
3. **OpenCorpora XML export** — not recommended as a round-trip: import
   irreversibly merges lemma and form grammemes into a single tag, and
   schema metadata (ids, `<links>`, revisions) isn't stored in
   `internal.Dictionary`. If ever needed, specify it as "XML export for
   human/third-party-tool reading," not as a reversible operation.

### Word inflection / form generation (`Forms`, `Inflect`) — PLANNED, backlog after 1.0.0

Surfaced by [docs/en/comparison.md](comparison.md) (2026-09-19): two of
three alternative Go implementations (AlexMaxy/gomorphy, SteosMorphy)
can produce a specific grammatical form of a word or list a lemma's
full wordform set; gomorphy currently cannot. Design sketch, not yet
decided or scheduled:

- **The data is already there.** `internal.Paradigm` is a flat
  `[suffix_i | tag_i | prefix_i]` table — every form of a lemma, always
  fully expanded at import time (see `pkg/morphology/parse.go`'s
  `readingForm`, which already reconstructs any one form's stem from
  any other form via `word - prefix(form) - suffix(form)`, including
  suppletive lemmas like "человек"/"люди" and prefixed comparatives
  like "по-" — both already solved for `Parse`/`Lemma`). This is
  cheaper for gomorphy than it was for AlexMaxy/gomorphy, which had to
  port pymorphy2's whole heuristic generation engine
  (`units_by_analogy`/`units_by_hyphen`/`units_by_shape`) — gomorphy
  needs none of that, since paradigms are never generated on the fly.
- **Proposed shape**: methods on `*Dictionary` (and mirrored on
  `MultiDictionary` via `Reading.Dict`, same pattern as `Parse`/`Lemma`
  today), taking an already-resolved `Reading` rather than a bare word
  string — this reuses `Parse`'s homonym disambiguation instead of
  duplicating it:
  ```go
  func (x *Dictionary) Forms(r Reading) []Reading   // r's whole paradigm, form order (0 = lemma)
  func (x *Dictionary) Inflect(r Reading, want []string) []Reading  // Forms filtered by native tag tokens
  ```
- **Open questions for whoever picks this up**: (1) `Inflect`'s `want`
  matches native tag tokens (comma/semicolon/space-split depending on
  source, same as `Reading.Tag` already is) — a `tagmap`-based
  universal-query variant is a plausible v2, not required for v1;
  (2) whether to allow `Forms`/`Inflect` on a `Reading` produced by
  prediction (an OOV word) — the paradigm is a guess in that case,
  same caveat prediction already carries for `Parse`, so probably
  allow it but document the caveat rather than special-case it;
  (3) no design doc or implementation plan exists yet — this needs its
  own brainstorming pass before work starts, this is only a sketch.

## Path to version 1.0.0

Every item on the release checklist fixed on 2026-09-14 is done; the
full history (format groundwork, the pre-1.0.0 code review, the
comparative-degree fix, multi-dict, the dense alphabet rollout, the CLI
redesign, Stage 18) is in
[implementation/path-to-1.0.md](implementation/path-to-1.0.md).

**Release 1.0.0 — the next step, nothing blocks it.**

Everything else (the zstd implementation from Stage 17, Stage 19,
Stage 20, the remaining tag-mapping work, dictionary export, word
inflection/form generation, the skill + examples, Universal
Dependencies) is backlog after 1.0.0, to be prioritized and refined
separately before each task starts.

### Dictionary usage skill — NOT STARTED

Split out of Stage 18 (2026-09-17) as new content with its own design,
not a documentation fix: `skills/use-dictionary/SKILL.md` — an agent
skill: `lookup`/`lemmas`/`fuzzy`/`top`, working with several `.dat`
files via `-d`, interactive mode (`gomorphy cli`). Versioned together
with the library, copied into the agent's configuration. (Not to be
confused with the "thematic dictionary" skill from Stage 19 — this one
is about using a dictionary, that one is about building one.)

~~`examples/`~~ — DONE 2026-09-19: [examples/](../../examples/) — one
runnable `package main` per `pkg/morphology` entry point
(`CompileFromXML`, `CompileFromUniMorph`, `NewMultiDictionary`,
`OpenPyMorphyDense`, `Open`), each under 50 lines, with a
[examples/README.md](../../examples/README.md) index. The three that
don't need real dictionary data (opencorpora/unimorph/multidict) embed
a tiny inline fixture and run with no setup; the two that read a real
pymorphy2 directory or a compiled `.dat` take the path as a flag and
were verified against this session's already-downloaded real data.

### Universal Dependencies as a data source — ARCHIVED 2026-09-19, not being pursued

Exploratory research (kept for reference, not acted on):
[docs/research/0007-universal-dependencies-import-plan.md](research/0007-universal-dependencies-import-plan.md).
Three ways to use UD had been identified: (A) a full importer into
`.dat`; (B) a reference corpus for checking `Parse()`'s accuracy; (C)
UD FEATS' ready-made schema as the target for universal tag mapping
between dictionaries.

**Decision: shelve UD entirely, all three variants.** Reasoning: UD's
value proposition is annotated connected text (a treebank), which
isn't a priority for this project — ready-made wordform-set sources
(UniMorph, pymorphy2, OpenCorpora) are far more valuable for gomorphy's
actual goal (a morphological dictionary, not a disambiguation/parsing
benchmark). This forecloses variant A (the importer) and variant B (the
reference corpus) outright, since both are only meaningful if annotated
text itself has value here. Variant C (UD FEATS as the tag-mapping
schema) was already separately decided against on 2026-09-19 — see
`implementation/tag-mapping.md`'s "What's left": `tagmap` keeps its
UniMorph schema, for reasons independent of this archival (it's already
implemented/tested on UniMorph, and has a direct synergy with Stage 16,
see above).

The research's open questions (Q1-Q5) are moot — not being revisited
unless UD is reconsidered from scratch in the future.

~~Comparison with alternative Go implementations~~ — DONE 2026-09-19:
[docs/en/comparison.md](comparison.md) — architecture, dictionary
sourcing, API coverage, storage size (measured, not just claimed),
performance, license, and maintenance activity against
[jus1d/gomorphy](https://github.com/jus1d/gomorphy),
[AlexMaxy/gomorphy](https://github.com/AlexMaxy/gomorphy), and
[SteosOfficial/SteosMorphy](https://github.com/SteosOfficial/SteosMorphy).
Findings worth flagging: none of the three alternatives implement
fuzzy search or multiple open dictionaries; two of the three
(AlexMaxy, SteosMorphy) implement word inflection/generation, which
gomorphy doesn't; gomorphy is the only one of the four that doesn't
embed dictionary data (a real trade-off, not a strict win — no
download/build step needed for the alternatives, but also no
CC BY-SA/data-license obligation inherited by gomorphy itself).

## Stage 19. Thematic dictionaries: TSV import, CLI batches, skills, MCP decision — DONE 2026-09-22 (builder/TSV/merge core; the import report, batch query modes, the skill and CLI `--summary` stay open)

**Summary**: Make the library and CLI a convenient tool for an agent to
build **thematic dictionaries** with no external resources (no
internet, no base OpenCorpora dictionary): preparing a set of
wordforms in a simple text format -> import -> `.dat`. Comes with a
"thematic dictionary" skill for the agent (the "using the gomorphy
dictionary" skill is in Stage 18) and a settled decision not to
implement an MCP server ([docs/mcp.md](mcp.md)).

Line format — **TSV, the third column is optional**:

```
lemma<TAB>form[<TAB>comma-separated tags]
```

Key properties:
- **Optional tags.** The third column can be omitted: for a thematic
  dictionary, the "form -> lemma" link matters more than grammar. Without
  tags, the agent doesn't spend tokens picking OpenCorpora tags.
- **Opaque tags.** Tags are arbitrary strings, including custom user
  labels (`colloquial`, `archaic`, `naval`, etc.); they're registered
  automatically as grammemes. No mapping onto the OpenCorpora set at all.
- **Auto-lemma.** An empty lemma means lemma := wordform (for
  non-inflecting forms).
- **Deduplication.** Repeated (form, tags) pairs within a lemma are
  dropped by the Builder.

**Increment** — the Builder/TSV/merge core shipped on 2026-09-22 (design
spec [2026-09-21-builder-tsv-merge-design.md](superpowers/specs/2026-09-21-builder-tsv-merge-design.md),
write-up [implementation/stage-19-builder-tsv-merge.md](implementation/stage-19-builder-tsv-merge.md));
the import report, the batch query modes, the skill, and the CLI
`--summary` flag are NOT part of that pass and stay open:
- `pkg/morphology` (the current facade, the FT8 Builder in its new form
  stays here):
  - [x] **Builder API** (`builder.go`): public `Builder`
    (`NewBuilder`/`AddForm`/`AddLemma`/`Build`, `BuilderOptions{Language,
    Source}`, sentinels `ErrNoEntries`, `ErrBuilderClosed`) + the shared
    unexported `buildFromEntries` helper.
  - [x] **`ImportTSV`** (`import_tsv.go`): `ImportTSV(r io.Reader, opts
    BuilderOptions)` — streaming `bufio.Scanner` reads (1MB lines),
    splitting on `\t` into 2-3 fields -> `AddForm`, grammeme
    auto-registration, dedup, per-field trim, auto-lemma, line-numbered
    errors; `Source` defaults to `"tsv"`.
  - [ ] **`ImportTSVFile`** — NOT built: a deliberate non-goal this pass
    (the CLI opens the file and passes the reader — see the design spec).
  - [ ] **`report.go`** — an import report (lemma/form counts, lemmas
    without tags, warnings about diverging stems (LCP) and anomalously
    short paradigms) — still open.
  - [x] **Merge** (`merge.go`): `Merge(base, overlays, mode)` with
    `MergeAdd`/`MergeReplace`, inputs read-only, `Source = "merge"`,
    base's `SourceVersion`/`Description`/language/CharPolicy inherited.
    The original entries-based rebuild was replaced 2026-09-23 by an
    id-preserving **structural merge** (base tags/prefixes/suffixes/
    paradigms keep their ids; only changed shards' word DAWGs rebuild),
    see Stage 19.1 above and
    [implementation/stage-19-builder-tsv-merge.md](implementation/stage-19-builder-tsv-merge.md#structural-merge-2026-09-23).
  - [x] **Prediction rebuilt** by the shared build pipeline (`Build`/
    `ImportTSV`/`Merge` all run `BuildPrediction` then `RecompileDense`)
    — for any result that stays single-shard (the typical thematic
    dictionary); a sharded merge output keeps prediction absent, same as
    OpenCorpora dictionaries (engine resolves prediction against shard 0
    only).
  - **Probability is NOT rebuildable** from a dictionary's own wordforms+paradigms
    (it is external corpus-frequency data, `p_t_given_w.intdawg`). Built/merged
    dictionaries therefore carry none — exactly like OpenCorpora dictionaries
    today. Open roadmap items (see the builder/merge design spec
    [2026-09-21-builder-tsv-merge-design.md](superpowers/specs/2026-09-21-builder-tsv-merge-design.md)):
    a way to build/rebuild probability from scratch (corpus frequency input to
    the Builder) and a carry-forward policy for merges that would DROP it.
    Prediction, by contrast, IS rebuilt by the shared build pipeline by default.
- CLI `cmd/gomorphy`:
  - [x] `gomorphy import tsv <file> [-o out.dat] [--source name]` —
    shipped, `-o` required (the planned `--summary` flag is NOT built).
  - [ ] **batch query modes**: `lookup`/`lemmas`/`fuzzy` accept several
    words as arguments or read from stdin one word per line (the `-`
    argument); results are sectioned by word. NOT implemented — still
    open. Batching closes the main source of token bloat for an agent
    (see [docs/mcp.md](mcp.md), §Tokens).
- [ ] Skill `skills/thematic-dictionary/SKILL.md` — NOT STARTED, still
  open: how to prepare a TSV for arbitrary vocabulary (nouns, ship/region
  names that inflect like adjectives), paradigm tables, custom tags,
  working standalone without OpenCorpora, verification via batch `lookup`
  and the import report. (The "using the gomorphy dictionary" skill is in
  Stage 18.)
- [x] MCP decision: [docs/mcp.md](mcp.md) — already recorded: no built-in
  MCP server, with the reasoning (no token savings, overhead, already
  covered by CLI batching).

**Automated checks (tests)** — delivered ones are green (test files
`pkg/morphology/builder_test.go`, `import_tsv_test.go`, `merge_test.go`,
`cmd/gomorphy/import_test.go`, `merge_test.go`, plus the white-box
`*_internal_test.go` checks for dense-alphabet/prediction/BuildInfo):
- [x] unit: a mini TSV (5-10 lemmas) -> correct lemmas, forms, tags;
- [x] unit: optional tags, empty lemma, custom tags, dedup;
- [ ] unit: import report — counters and warnings (no `report.go` yet);
- [x] roundtrip: `ImportTSV -> SaveTo -> Open -> Lookup` identical to a
  direct build;
- [ ] CLI: batch `lookup` with several arguments and via stdin (no batch
  modes yet);
- [x] `go test -race ./...` — green (verified 2026-09-22).

**Manual checks** — the ones that depend on still-open items are not run:
- [ ] `gomorphy import tsv names.tsv -o names.dict --summary` -> `.dat`
  created (the `--summary` flag is not built; the command without it is
  covered by `cmd/gomorphy/import_test.go`);
- [ ] `gomorphy -dict names.dict lookup - < words.txt` -> sections per
  word (no batch modes yet);
- [ ] run the "thematic dictionary" skill through an agent on the example
  "a dictionary of ship-name adjectives" with no OpenCorpora loaded (no
  skill yet).

## Stage 20. Synonym database: groups, tags, sidecar file — PLANNED (scenarios are an open question)

Consumer note (2026-09-24): the lexicon module does not need synonyms for
NER; this stage stays unscheduled.

**Summary**: Alongside the wordform dictionary — a separate database of
**synonyms and derived concepts** (sidecar file `.syn`). Problems it
solves that can't be derived from wordforms:

- "the primary/official concept": axe handle -> axe, refrigerator -> cold;
- "non-canonical form -> official form": Lyosha -> Alexey, Dima ->
  Dmitry, Shurik -> Alexander (the forms may be entirely unrelated by
  Levenshtein distance);
- **reverse query** "all derivatives of a base concept": cold ->
  {refrigerator, ice cream, cooling}; Alexander -> {Sasha, Shura, Shurik, Sanya};
- **m2m**: a lemma can have several interpretations (Lyonya -> both
  Leonid and Alexey) — a "group" (clique) model, not pairs.

It's enough to define links at the **lemma** level (grammatical
derivatives like "Shurikom from Alexander" are worked out in code
through `Lookup`).

**Hypothesis about applicability (to be checked with scenarios)**: a
synonym database alongside the wordform dictionary and word formation —
or on its own — looks promising for text-processing and generation
tasks: synonym substitution, generating variants, normalizing
non-canonical forms. Scenarios are worked out before implementation
(see "Open questions").

### Tagging (needs further design)

Tagging within synonym sections is desirable, and the tag set depends
heavily on the word's domain:

- for names: "official", "colloquial", "affectionate", "coarse";
- for tools: official names, colloquial names, local/dialectal names, etc.

So this stage **doesn't fix a tag set**: tags are supplied by the user
as opaque grammemes (like the thematic dictionaries in Stage 19) and
registered on import.

**Open questions (to work out before implementation, flagged in the plan)**:
- tag semantics: is the set defined by the library or the user; are
  tags mandatory; do they mark the whole group or individual members;
- domain-specificity of tag sets (names vs. tools) — how to model this
  without rigid predefined lists;
- the synonym key: lemma text (simple, but mixes in homonymy) vs.
  `LemmaID` (precise, but tied to a `.dat` version);
- storage: a sidecar `.syn` (updates independently, no need to rebuild
  the large dictionary) vs. a section in `.dat` (rebuild on every
  synonym edit) — sidecar is preferred;
- combined and standalone modes: entry through the existing
  `Lookup`/`Lemmas` when a dictionary is present, or direct key lookups
  with no dictionary.

**Increment** (to be refined once scenarios are worked out):
- package `pkg/synonyms`:
  - a "group" model: lemma -> []group ids; group -> []members +
    optional tags (opaque grammemes);
  - TSV import `group<TAB>lemma[<TAB>tags]` (reusing Stage 19's import
    scheme), dedup, tag auto-registration;
  - serialization into a compact sidecar file (varint arrays, xxh3, mmap);
  - API: `Synonyms(lemma)`, `Derivations(lemma)`; entry through `Lookup`
    when a dictionary is present;
  - CLI: `gomorphy import synonyms <file> -o dict.syn`,
    `gomorphy -dict dict.dat synonyms <word>`, `derivations <word>`;
    a standalone mode with no `-dict`.

**Automated checks (tests)**:
- unit: m2m (Lyonya -> {Leonid, Alexey});
- unit: reverse query "derivatives of a base concept" (cold -> ...);
- unit: import with tags (a nominal set / a domain set), dedup;
- roundtrip: `import -> SaveTo -> Open -> Synonyms/Derivations` identical
  to a direct build;
- integration: wordform -> `Lookup` -> lemma -> `Synonyms`;
- `go test -race ./...` — green.

**Manual checks**:
- an example with names: "Lyosha -> Alexey", "Shurik -> Alexander", the
  reverse "Alexander -> {Sasha, Shura, Shurik, Sanya}";
- an example with tools using a domain tag set;
- standalone mode with no wordform dictionary loaded.

## Windows: native mmap — PLANNED (post-1.0 backlog)

`internal/mmapx` builds on Windows (`go build` doesn't fail — see the
pre-1.0.0 review), but `Open` there returns a "not supported on windows
yet" error: the implementation only reads the file via
`syscall.Mmap`/`MAP_PRIVATE` (Unix-only). Task: add
`internal/mmapx/mmap_windows.go` via `golang.org/x/sys/windows`
(`CreateFileMapping`/`MapViewOfFile`), covering the same contract
(`Open`/`Bytes`/`Len`/`Close`), and verify it on a real Windows machine
(CI or by hand) — cross-compilation with no real test of the mapping
itself isn't enough. Until then, `morphology.OpenBytes` (no mmap) is the
workaround for loading a dictionary on Windows.
