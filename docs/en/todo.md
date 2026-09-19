# Roadmap

Stages 0-15 are done. Details for each stage are in
[implementation.md](implementation.md) and the individual files under
[implementation/](implementation/); the same place holds the storage
redesign rationale (stages 11-18: CSR-trie + exact-hash + pairs ->
paradigms + DAWG) and unplanned but significant findings and features
discovered along the way (DAWG build speedup, DAWG minimization fix,
critical pre-1.0 review bugs, multi-dict, the dense DAWG alphabet for
pymorphy2, the pymorphy2 source in the CLI, tag mapping between
dictionaries).

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
| — Dense 1-byte DAWG alphabet for pymorphy2 `words.dawg` | DONE (partial, see remaining backlog below) | [implementation/pymorphy2-dense-alphabet.md](implementation/pymorphy2-dense-alphabet.md) |
| — pymorphy2 source (`pkg/pymorphy`) + integration into the `gomorphy` CLI | DONE | [implementation/pymorphy-source-and-cli.md](implementation/pymorphy-source-and-cli.md) |
| — Universal tag mapping between dictionaries (`pkg/morphology/tagmap`, native -> universal) | DONE (partial, see remaining backlog below) | [implementation/tag-mapping.md](implementation/tag-mapping.md) |
| — Stage 18 (documentation + tests + godoc audit; the CLI part was closed separately, see above) | DONE 2026-09-17 | [implementation/stage-18-finalize.md](implementation/stage-18-finalize.md) |

## Unfinished/future stages

### Stage 16. UniMorph import — NOT STARTED

Import a UniMorph dictionary from TSV. Full plan in
[implementation/stage-16-import-unimorph.md](implementation/stage-16-import-unimorph.md)
(ready to implement, the plan hasn't changed). Not on the path to 1.0.0
(see below) — prioritized after release.

### Stage 17 — remainder: zstd compression + two candidates from the format analysis

Narrowing ID types, format groundwork for compression, and the density
analysis of DAWG packing/`tagset` encoding are already done (see the
table above,
[docs/research/0001-dawg-alphabet-density.md](research/0001-dawg-alphabet-density.md)
and [docs/research/0002-paradigm-tagset-binary-encoding.md](research/0002-paradigm-tagset-binary-encoding.md)).
What's left — separate future tasks, **not blocking 1.0.0**, in priority
order:

1. zstd for the cold suffixes/prefixes/tagset/paradigms sections
   (`klauspost/compress`, maximum compression level).
2. A dense one-byte label alphabet for `words.dawg` (instead of
   per-byte UTF-8) — already implemented for pymorphy2 as a separate
   feature, see [implementation/pymorphy2-dense-alphabet.md](implementation/pymorphy2-dense-alphabet.md)
   and "Dense alphabet: remaining backlog" below — not started for the
   OpenCorpora build.
3. A grammeme dictionary + index lists instead of JSON for the `tagset`
   section — found a ~78.2% effect on the section (~1.09% of the whole
   file) on a real dictionary, with no change to the hot read path.
   Smaller effect than item 2, but with no open applicability
   questions — a low-risk implementation. Details in
   [implementation/stage-17-optimize.md](implementation/stage-17-optimize.md#tagset-encoding-analysis--done-a-backlog-candidate-exists).

Details and what's already done are in
[implementation/stage-17-optimize.md](implementation/stage-17-optimize.md).

### Dense alphabet: remaining backlog (not blocking 1.0.0)

For pymorphy2 `words.dawg`, the dense 1-byte alphabet is implemented
(see the table above). The full breakdown of what's done and what's
left is in
[implementation/pymorphy2-dense-alphabet.md](implementation/pymorphy2-dense-alphabet.md).
A short summary of what remains, in priority order:

1. ~~Serializing `Alphabet` into `.dat` + support in `Open()`~~ — DONE
   2026-09-19: a new `"alphabet"` section, written/read like the
   optional `probability` section; `OpenPyMorphyDense` -> `SaveTo` ->
   `Open` now round-trips with identical `Parse`/`Lemma` results. See
   [implementation/pymorphy2-dense-alphabet.md](implementation/pymorphy2-dense-alphabet.md).
2. `fuzzy.go` — doesn't work for dense dictionaries (`nil`); the
   traversal needs to be rethought for fixed width.
3. `Prediction`/`Probability` DAWGs — not investigated under a dense
   alphabet.
4. A 2-byte alphabet in the read path — not carried into `Open`/`Parse`;
   only needed by a real multilingual consumer for whom multi-dict
   (see [implementation/multi-dict.md](implementation/multi-dict.md)) doesn't fit.
5. CLI (`gomorphy`) — a dense dictionary can only be obtained via the Go API.

### Universal tag mapping: remaining backlog (not blocking 1.0.0)

The `native -> universal` direction is implemented (see the table
above). Two items found during implementation remain separate future
tasks — details in
[implementation/tag-mapping.md](implementation/tag-mapping.md):

1. **`Unmap` (universal -> native)** — needed for dictionary export,
   see "Dictionary export" below; deliberately deferred until a
   concrete consumer appears (the ambiguity of the reverse mapping
   needs to be resolved against a specific exporter).
2. **`TagSet.Name` is unreachable from `pkg/morphology`'s public API** —
   `tagmap.Map` needs a `dictName`, and there's currently no way to get
   one from outside the package (`Dictionary` doesn't export `TagSet`,
   and `BuildInfo.Source` isn't a substitute — their values diverge).
   Doesn't block anything right now (`tagmap` is a leaf package nobody
   uses yet), but it's the first item for any real consumer of
   `tagmap.Map` or for export.

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

## Path to version 1.0.0

Release order (fixed on 2026-09-14, extended on 2026-09-15/16):

1. ~~Format: groundwork for extensible compression + info section~~ — DONE.
2. ~~Pre-1.0.0 code review + findings triage + both critical bugs
   (suffix overflow via sharding, corrupted OpenCorpora wordform tags
   including the paradigm dedup bug)~~ —
   DONE, see [implementation/code-review-pre-1.0-triage.md](implementation/code-review-pre-1.0-triage.md).
3. ~~Fix for the root-in-suffix issue with the comparative degree
   (`Cmp2`/"по-") + a rune-safe `lcp()`~~ — DONE, see
   [docs/research/0003-comparative-paradigms-not-merging.md](research/0003-comparative-paradigms-not-merging.md),
   [docs/superpowers/specs/2026-09-15-comparative-prefix-split-design.md](superpowers/specs/2026-09-15-comparative-prefix-split-design.md),
   [docs/superpowers/plans/2026-09-15-comparative-prefix-split.md](superpowers/plans/2026-09-15-comparative-prefix-split.md).
   Real effect: shard 0's paradigms shrank from 17,934 to 3,245, the
   dictionary now fits in 1 shard instead of 2, and the share of
   invalid UTF-8 suffixes dropped from 61.7% to 0%.
4. ~~Support for several dictionaries open at once (multi-dict)~~ —
   DONE 2026-09-16, see [implementation/multi-dict.md](implementation/multi-dict.md).
5. ~~Production rollout of the dense 1-byte DAWG alphabet~~ — for
   pymorphy2 `words.dawg`, DONE 2026-09-16, see
   [implementation/pymorphy2-dense-alphabet.md](implementation/pymorphy2-dense-alphabet.md).
   The remainder (`.dat` serialization of the alphabet, `fuzzy.go`,
   `Prediction`/`Probability`) is backlog, not blocking 1.0.0, see
   "Dense alphabet: remaining backlog" above.
6. ~~CLI grooming + redesign~~ — DONE 2026-09-16 — a single cobra-based
   `gomorphy` binary (`cmd/gomorphy_build` removed), commands
   `lookup`/`lemmas`/`fuzzy`/`top`/`cli`/`download`/`unpack`/`build`/`update`/
   `version`/`merge`(stub)/`split`(stub), `-d/--dictionary` always
   goes through `MultiDictionary`, `$GOMORPHY_DICTIONARY`, `-v/-l`
   logging. See
   `docs/superpowers/specs/2026-09-16-cli-redesign-design.md`,
   `docs/superpowers/plans/2026-09-16-cli-redesign.md`, and
   [implementation/pymorphy-source-and-cli.md](implementation/pymorphy-source-and-cli.md).
7. ~~Stage 18 (documentation, tests, metrics)~~ — DONE 2026-09-17,
   see [implementation/stage-18-finalize.md](implementation/stage-18-finalize.md).
   The CLI part was closed earlier, in item 6. The agent skill and
   `examples/` were deliberately split out of this stage, see "Dictionary
   usage skill + examples" below.
8. **Release 1.0.0 — the next step, nothing blocks it.**

Everything else (the zstd implementation from Stage 17, Stage 16, Stage
19, Stage 20, the remaining dense-alphabet work, the remaining tag-mapping
work, dictionary export, the skill + examples, Universal Dependencies) is
backlog after 1.0.0, to be prioritized and refined separately before each
task starts.

### Dictionary usage skill + `examples/` — NOT STARTED

Split out of Stage 18 (2026-09-17) as new content with its own design,
not a documentation fix:

- `skills/use-dictionary/SKILL.md` — an agent skill: `lookup`/
  `lemmas`/`fuzzy`/`top`, working with several `.dat` files via `-d`,
  interactive mode (`gomorphy cli`). Versioned together with the
  library, copied into the agent's configuration. (Not to be confused
  with the "thematic dictionary" skill from Stage 19 — this one is
  about using a dictionary, that one is about building one.)
- `examples/` — minimal Go examples for each `pkg/morphology` entry
  point (`Open`/`OpenPyMorphy`/`CompileFromXMLFile`/
  `Parse`/`MultiDictionary`).

### Universal Dependencies as a data source — open questions, including a conflict with the already-implemented tag mapping

Exploratory research: [docs/research/0007-universal-dependencies-import-plan.md](research/0007-universal-dependencies-import-plan.md).
Three ways to use UD: (A) a full importer into `.dat` — not
recommended (low marginal value, licensing-confusion risk for some
Russian treebanks); (B) a reference corpus for checking `Parse()`'s
accuracy; (C) UD FEATS' ready-made schema as the target for universal
tag mapping between dictionaries.

**A conflict found (not noticed until today's cross-check of the
documents)**: the research's recommendation is to start with variant C,
taking **UD FEATS** as the target mapping schema. The `pkg/morphology/tagmap`
implemented on 2026-09-17 (see [implementation/tag-mapping.md](implementation/tag-mapping.md))
uses the **UniMorph** schema, not UD FEATS — a decision made in a
separate brainstorming session on 2026-09-17 without cross-checking
this research (which wasn't accounted for at the time). That's not
necessarily a mistake — UniMorph has its own grounds too (its format
was already analyzed, and it was already referenced in
`docs/unimorph.md` §5.4) — but the choice between the two universal-tag
schemas was effectively made by default, not deliberately. An open
question before any further work on tag mapping or dictionary export:
whether to switch the schema to UD FEATS, keep UniMorph, or whether it
doesn't matter in practice (both are fixed external standards, and
converting between them isn't any harder than the original task).

The research's other open questions (Q1-Q5: which treebank to use as
the reference for variant B, manual vs. programmatic mapping, how to
store test CoNLL-U files, whether a separate CoNLL-U reader is needed)
are unresolved, awaiting discussion with the user on whether to take on
UD at all.

### Comparison with alternative Go implementations — NOT STARTED

There are at least three other Go projects for Russian morphological
analysis:

- [jus1d/gomorphy](https://github.com/jus1d/gomorphy)
- [AlexMaxy/gomorphy](https://github.com/AlexMaxy/gomorphy)
- [SteosOfficial/SteosMorphy](https://github.com/SteosOfficial/SteosMorphy)

Task: analyze these implementations (architecture, dictionary data
source, API coverage — exact lookup/lemmas/fuzzy/prediction, storage
format and its size, performance if benchmarks exist, maintenance
activity, license) and produce a pros/cons/differences comparison
against gomorphy. Result: a table/section in the README and/or a
separate document (`docs/en/comparison.md`?), which the README's
"Related projects" section already links to.

## Stage 19. Thematic dictionaries: TSV import, CLI batches, skills, MCP decision — PLANNED

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

**Increment**:
- `pkg/morphology` (the current facade, FT8 Builder is kept in the new
  implementation too):
  - `import_tsv.go`: `ImportTSV(r io.Reader, opts Options) (*Dictionary, error)`,
    `ImportTSVFile(path string, opts Options) (*Dictionary, error)` —
    streaming `bufio.Scanner` reads, splitting on `\t` into 2-3 fields ->
    `AddLemma`/`AddForm`, grammeme auto-registration, dedup.
  - `report.go`: an import report — lemma/form counts, lemmas without
    tags, warnings about diverging stems (LCP) and anomalously short
    paradigms (a substitute for external verification in offline
    scenarios).
- CLI `cmd/gomorphy`:
  - `gomorphy import tsv <file> -o out.dict [--summary]`;
  - **batch query modes**: `lookup`/`lemmas`/`fuzzy` accept several
    words as arguments or read from stdin one word per line (the `-`
    argument); results are sectioned by word. Batching closes the main
    source of token bloat for an agent (see [docs/mcp.md](mcp.md), §Tokens).
- Skill `skills/thematic-dictionary/SKILL.md` — "thematic dictionary"
  (authored as a `SKILL.md` in the repository, versioned together with
  the library, copied into the agent's configuration): how to prepare a
  TSV for arbitrary vocabulary (nouns, ship/region names that inflect
  like adjectives), paradigm tables, custom tags, working standalone
  without OpenCorpora, verification via batch `lookup` and the import
  report. (The "using the gomorphy dictionary" skill is in Stage 18.)
- MCP decision: [docs/mcp.md](mcp.md) — record the intent not to
  implement a built-in MCP server, with the reasoning (no token
  savings, overhead, already covered by CLI batching).

**Automated checks (tests)**:
- unit: a mini TSV (5-10 lemmas) -> correct lemmas, forms, tags;
- unit: optional tags, empty lemma, custom tags, dedup;
- unit: import report — counters and warnings;
- roundtrip: `ImportTSV -> SaveTo -> Open -> Lookup` identical to a
  direct build;
- CLI: batch `lookup` with several arguments and via stdin;
- `go test -race ./...` — green.

**Manual checks**:
- `gomorphy import tsv names.tsv -o names.dict --summary` -> `.dat` created;
- `gomorphy -dict names.dict lookup - < words.txt` -> sections per word;
- run the "thematic dictionary" skill through an agent on the example
  "a dictionary of ship-name adjectives" with no OpenCorpora loaded.

## Stage 20. Synonym database: groups, tags, sidecar file — PLANNED (scenarios are an open question)

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
itself isn't enough.
