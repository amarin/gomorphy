# Builder API + TSV import + `Merge` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship three user-facing capabilities on top of one new shared
pipeline: (1) a public `morphology.Builder` that assembles a dictionary from
raw `(word, lemma, tag)` entries, (2) a TSV import path
(`gomorphy import tsv` / `morphology.ImportTSV`), and (3) `gomorphy merge` /
`morphology.Merge` combining `.dat` dictionaries with a word-level
`add`/`replace` conflict policy. Every output dictionary is fully
functional — dense 1-byte alphabet, prediction rebuilt, `Parse`/`Fuzzy`
working, `SaveTo`-able.

**Architecture:** The three features share one core operation — build an
`internal.Dictionary` from wordform-level entries. That core lands as a new
internal pipeline `internal.BuildDictionaryFromEntries` (a fresh
implementation mirroring the opencorpora importer's proven phases — intern
suffixes/tags → shard → dedup paradigms → `BuildDAWGWithValues` →
`internal.NewDictionary`), which stays **raw** (no alphabet, no prediction,
no Info). A tiny unexported helper in `pkg/morphology` (next to the new
`builder.go`) post-processes every output: `internal.BuildPrediction` →
`internal.RecompileDense` → `Info`. This split keeps `internal` free of the
`productive()` predicate, which lives in `pkg/morphology` (parse.go:275).

**Tech Stack:** Go (existing toolchain), `github.com/stretchr/testify`
(already vendored). CLI via the existing `spf13/cobra` setup in
`cmd/gomorphy`.

**Spec:** [docs/en/superpowers/specs/2026-09-21-builder-tsv-merge-design.md](../specs/2026-09-21-builder-tsv-merge-design.md)

## Global Constraints

- **Do NOT refactor the three importers** (opencorpora, pymorphy2, unimorph)
  onto the new pipeline. They stay untouched; this is an explicit spec
  non-goal.
- **TSV column order is lemma-first**: `lemma<TAB>word[<TAB>tags]` (the
  spec's example is lemma-first after its fix). Never swap the order.
- **`internal` must not import `pkg/morphology`** (import cycle: the
  `productive` predicate is a `pkg/morphology` function, parse.go:274-285).
  Therefore `internal.BuildPrediction` takes the predicate as a parameter:
  `BuildPrediction(d *Dictionary, productive func(tag string) bool) error`,
  and callers pass `morphology`'s own `productive`.
- **Pipeline order matters**: `BuildPrediction` must run on the **raw**
  dict (plain-text wordform keys) **before** `RecompileDense` — prediction
  DAWGs are never densified (parse.go:154-158 comment, and
  `RecompileDense`'s own doc says Prediction is left untouched,
  dense_recompile.go:12-14).
- **Prediction is only built for unsharded dictionaries** (`len(d.Words)==1`).
  `BuildPrediction` no-ops otherwise (the engine resolves predictions against
  shard 0 only, parse.go:143-145).
- **Every built/merged/TSV dictionary carries no probability** (spec
  non-goal — same as OpenCorpora dicts today). Probability stays a roadmap
  item in todo.md Stage 19.
- **Builder dedup is by `(word, lemma, tag)` triple**, preserving insertion
  order (store an ordered slice + a dedup map — never rely on map iteration
  order; determinism matters for reproducible `.dat` files). A duplicate
  word with a *different* tag/lemma is a second reading (homonymy), kept.
- **Output color**: dictionary text is **dense 1-byte alphabet** for all three
  features (agreed CLI policy, build.go:38-42); the raw variant stays
  Go-API-only and is not exposed in this pass.
- **Public API names** are exactly as in the spec: `BuilderOptions`,
  `NewBuilder`, `AddForm(word, lemma, tag)`, `AddLemma(normal, tag)`,
  `Build()`, `ImportTSV(r, opts)`, `MergeMode` (`MergeAdd`/`MergeReplace`),
  `Merge(base, overlays, mode)`.
- **Errors are wrapped with context** (`fmt.Errorf("...: %w", err)`), matching
  the repo's existing `open.go`/`save.go` style. New sentinel errors:
  `ErrBuilderClosed` and `ErrNoEntries`.
- **CLI**: `-o` is required for both `import tsv` and `merge` (spec:
  "must never silently overwrite an input"); `--mode` is required and
  validated case-insensitively (`add`/`replace`); `merge` takes minimum 2
  positional args (base + ≥1 overlay) via `Args:`.
- **Existing CLI test to update in Task 7**: `cmd/gomorphy/merge_test.go`
  asserts "not yet implemented" — replace with real-command behavior tests.
- Run `go test ./... -race` after every task; it must stay green. Run
  `golangci-lint run ./...` on touched packages before each task's final
  commit. `go vet ./...` must stay clean.

---

## Task 1: `internal.BuildDictionaryFromEntries` (raw pipeline)

**Files:**
- Create: `pkg/morphology/internal/build.go`
- Test: `pkg/morphology/internal/build_test.go`

**Interfaces:**
- Produces:
  - `type Entry struct { Word, Lemma, Tag string }`
  - `type BuildOptions struct { Language string; CharPolicy *CharPolicy; TagSetName string; Progress func(processed, total int) }`
  - `func BuildDictionaryFromEntries(opts BuildOptions, entries []Entry) (*Dictionary, error)`
    — returns a **raw** dictionary: `Prefixes == [""]`, `Suffixes`/`Paradigms`/
    `Words` parallel per shard, `TagSet` fresh (`NewTagSet(opts.TagSetName)`,
    default `"builder"`), **no** `Alphabet`, **no** `Prediction`, no `Info`.
    `RecompileDense`/`BuildPrediction` are the caller's job (Task 4's helper).

The pipeline, modeled on `importers/opencorpora/import.go`'s phases and the
near-identical unimorph copy (`importers/unimorph/import.go`):

1. **Normalize entries**: `Lemma == ""` → `Lemma = Word` (auto-lemma).
2. **Shard grouping** by lemma with `FillOnDemand` (`shard.go`)
   and `suffixShardLimit = 1 << 16` — copied locally like the other
   importers do (a copy, not a shared import; see shard.go's existing
   comment on why compilers duplicating suffixes/tag helpers).
3. **Per lemma** (insertion order, tracked by a `byLemma`-style structure):
   - form 0 **must be the lemma text**: if no entry has `Word == Lemma`,
     synthesize `{Word: Lemma, Tag: ""}` as form 0 (the unimorph `buildForms`
     pattern, importers/unimorph/import.go:176-194). `readingForm`
     (parse.go:206) reconstructs `Normal` from form-0's affixes, so this is
     what makes every other form's lemma come out right.
   - `lcp` stem = longest common prefix of all form texts (a local copy of
     the existing `lcp`, trimmed to UTF-8 rune boundary).
   - each form → `(suffix_id, tag_id)`; suffixes interned per shard in
     uint16 id space (`ErrTagSetFull` propagated from `TagSet.Add`, cap
     checked like unimorph import.go:315-320); tags via `opts.TagSet`.
   - `prefixes` array all zeros (`""`), one paradigm per lemma; dedup
     identical paradigms via a `paradigmKeyHash`-style function (a local
     copy of opencorpora import.go's `paradigmKeyHash`) → `paraID`.
   - `(paraID<<16|formIdx)` DAWG values, key = form text.
4. **DAWG per shard**: `internal.BuildDAWGWithValues` (the existing 4-byte
   uint32-value builder, dawgbuild.go:44 — exactly the words.dawg payload
   shape `reading` expects, parse.go:194-201). Progress cumulative across
   shards via `BuildDAWGWithValuesProgress` (dawgbuild_progress.go) when
   `opts.Progress != nil`.
5. **Assemble** via `internal.NewDictionary(language, tagSet, suffixes,
   prefixList, paradigms, words, charPolicy)`. `CharPolicy` defaults to
   `internal.RussianCharPolicy()` when nil; `Language` defaults `"ru"`.

Errors to produce and test: `ErrTagSetFull` propagation at the 65536 tag cap
and 65536-paradigm cap (mirror unimorph import.go:336-338); invalid entry
word empty is handled earlier (Task 4) so the internal pipeline accepts any
non-empty entries.

- [ ] **Step 1: Write the failing tests**
  `internal/build_test.go` (package `internal`): a small 2-lemma corpus
  (`кошка`/`кошка`…`кошкой`, `дело`/`дело`/`делом`) asserts: `Prefixes ==
  [""]`; one shard; each wordform resolves via `DAWG.SimilarItems` to the
  right `(paraID, form)`; form-0-is-lemma synthesis (a lemma whose wordforms
  never include the lemma text gets a synthesized form 0 with `Tag: ""`);
  auto-lemma (empty lemma in an `Entry` → lemma = word); paradigm dedup
  (two lemmas with identical `(suffix_id, tag_id)` lists share one
  `paraID`, verified through `SimilarItems` values); insertion-order
  determinism (two identical runs produce byte-identical `.dat` via
  `EncodeParadigms`/`EncodeStrings`); `ErrTagSetFull` is hard to hit — skip
  (covered by existing tagset tests), but the 65536-unique-paradigms guard
  error can be unit-tested by setting the cap low via a test-only variable
  if one exists, else by `1<<16` theory only.
- [ ] **Step 2: Implement `internal/build.go`**
- [ ] **Step 3: Verify** — `go test ./pkg/morphology/internal/ -race` green;
  `golangci-lint run ./pkg/morphology/internal/...` clean.

---

## Task 2: `internal.BuildDAWGWithValuesBytes` + `internal.BuildPrediction`

**Files:**
- Edit: `pkg/morphology/internal/dawgbuild.go` (add
  `BuildDAWGWithValuesBytes`)
- Create: `pkg/morphology/internal/prediction.go`
- Test: `pkg/morphology/internal/prediction_test.go`

**Interfaces:**
- Produces:
  - `func BuildDAWGWithValuesBytes(keys []string, values [][]byte) (*DAWG, error)`
    — like `BuildDAWGWithValues` (dawgbuild.go:44) but takes an
    arbitrary-length byte payload per key; each key becomes
    `key + PayloadSeparator + base64(value)` (the exact shape
    `SimilarItems` decodes, dawg.go:281 / similar_items.go:63). Reimplement
    the existing `BuildDAWGWithValues` as a thin wrapper over it
    (4-byte BE uint32 → bytes) so the payload rule lives in exactly one
    place; the existing dawgbuild tests must stay green.
  - `func BuildPrediction(d *Dictionary, productive func(tag string) bool) error`
    — pymorphy2's KnownSuffixAnalyzer rebuilt from `d`'s own wordforms, in
    the exact on-disk format the engine already consumes (parse.go:146-190):
    each value is ≥6 bytes — `count(BE16) + para(BE16) + form(BE16)`. Sets
    `d.Prediction = []*DAWG{pred}` (one entry — `Prefixes == [""]` for every
    dict this pipeline produces, so engine index 0 is correct). No-op
    (returns nil, leaves `d.Prediction` nil) when `len(d.Words) != 1`.

Algorithm (mirrors `predictForPrefix`'s inverse):
1. For each wordform walked from `d.Words[0].Walk` (the raw dict — must be
   called **before** densification), for each of its values
   (≥4 bytes: `para = BE16`, `form = BE16`, via the engine's own guard
   parse.go:195), skip if `!productive(d.TagSet.TagName(para.Tag(form)))`.
2. For each suffix length L in 1..5 over the wordform's **runes**
   (`suffixSplits(wordform, 5)` shape, parse.go:291-305): key = last L
   runes of the wordform; accumulate `count` per `(key, para, form)`,
   `count` capped at 65535.
3. One `(key, value=6-byte)` pair per `(key, para, form)` accumulation;
   `BuildDAWGWithValuesBytes` to `pred`.

- [ ] **Step 1: Write the failing tests**
  `dawgbuild` additions in the existing `dawgbuild_test.go`: multi-byte
  payload roundtrip via `SimilarItems`; several values per key land as
  separate payload leaves (homonym-style).
  `internal/prediction_test.go` (package `internal`): build the small
  Task-1 dict manually (or via `BuildDictionaryFromEntries`), run
  `BuildPrediction` with a `productive` returning false for one constructed
  tag; assert: `d.Prediction` has length 1; a known productive suffix
  resolves via `pred.SimilarItems(suffix, RussianCharPolicy(), nil)` to the
  expected `(count, para, form)` triples; the non-productive tag's
  `(suffix, para, form)` is absent; `count` of 2 for a suffix twice present;
  keys are rune-correct (a 5-rune word produces suffix keys of lengths 1..5);
  multi-value keys return ≥2 values. No-op on a 2-shard dict.
- [ ] **Step 2: Implement both**
- [ ] **Step 3: Verify** — `go test ./pkg/morphology/internal/ -race`; lint.

---

## Task 3: `internal.BuildPrediction` integration — prediction works end-to-end

(Optional split of Task 2's verification into its own task to keep each
commit reviewable. If Tasks 1–2 land together cleanly, merge this into Task 2
and drop the empty task.)

**Files:**
- Test: `pkg/morphology/internal/prediction_integration_test.go`

- [ ] **Step 1: Write the failing test** — build the Task-1 dict via
  `BuildDictionaryFromEntries`, `BuildPrediction(d, productive)`, and
  `RecompileDense(d)`; assert a predicted reading is decodable by the public
  `parse`-equivalent math: `readingForm(0, wordStart+suffixKey, para, form)`
  recovers `Normal` == the lemma. This proves the para/form ids written by
  `BuildPrediction` line up with the paradigms/affixes `readingForm` uses.
- [ ] **Step 2: Implement** — no library code expected; this task exists to
  make the cross-check visible before public API work starts.
- [ ] **Step 3: Verify** — `go test ./pkg/morphology/internal/ -race`.

---

## Task 4: public `morphology.Builder` + shared post-process helper

**Files:**
- Create: `pkg/morphology/builder.go`
- Test: `pkg/morphology/builder_test.go`

**Interfaces** (exactly the spec):
- `type BuilderOptions struct { Language string; Source string }` —
  `Language` default `"ru"`; `Source` default `"builder"` (overridden to
  `"tsv"` by `ImportTSV`, `"merge"` by `Merge`), fills `BuildInfo.Source`.
- `func NewBuilder(opts BuilderOptions) *Builder`
- `func (b *Builder) AddForm(word, lemma, tag string) error` — empty or
  whitespace-only `word` is an error (new sentinel `ErrNoEntries` is **not**
  used here; just an error naming the problem). `lemma == ""` → auto-lemma.
- `func (b *Builder) AddLemma(normal, tag string) error` — `AddForm(normal,
  normal, tag)`.
- `func (b *Builder) Build() (*Dictionary, error)` — dedup `(word, lemma,
  tag)` triples by insertion order; zero entries → `ErrNoEntries`; then run
  the shared helper (below); mark the Builder closed (further `AddForm`s →
  `ErrBuilderClosed`).
- unexported `func buildFromEntries(opts BuilderOptions, entries []Entry, tagSetName string) (*Dictionary, error)`:
  1. `internal.BuildDictionaryFromEntries(internal.BuildOptions{Language,
     CharPolicy: nil, TagSetName: tagSetName}, entries)` (raw).
  2. `internal.BuildPrediction(d, productive)` (no-ops if sharded).
  3. `internal.RecompileDense(d)`.
  4. `d.Info = &internal.BuildInfo{Source: opts.Source}` (or the tagSetName
     default when empty).
  5. wrap into `&Dictionary{d: d}`.
  This helper is reused by `import_tsv.go` (Task 5) and `merge.go` (Task 6).

- [ ] **Step 1: Write the failing tests**
  `pkg/morphology/builder_test.go` (package `morphology_test`): build a small
  multi-lemma dictionary → `SaveTo` → `Open` round-trip; `Parse` returns the
  expected `(Word, Normal, Tag)`; auto-lemma gives `Normal == Word`; triple
  dedup collapses exact repeats but keeps a homonym reading (same word,
  different tag); `AddLemma`; empty-word `AddForm` → error; `Build` on empty
  → `ErrNoEntries`; `AddForm` after `Build` → `ErrBuilderClosed`; output is
  dense (`d.d.Alphabet != nil`) and prediction non-empty (`d.d.Prediction`
  length 1); `Fuzzy`/`FuzzyTop` (fuzzy.go:34,53) return plausible results.
- [ ] **Step 2: Implement `builder.go`**
- [ ] **Step 3: Verify** — `go test ./pkg/morphology/ -race`; lint both
  packages.

---

## Task 5: `morphology.ImportTSV` + `morphology.BuilderOptions` (TSV format)

**Files:**
- Create: `pkg/morphology/import_tsv.go`
- Test: `pkg/morphology/import_tsv_test.go`

**Interface** (spec):
- `func ImportTSV(r io.Reader, opts BuilderOptions) (*Dictionary, error)`
  — builds via the same shared helper with `Source` forced to `"tsv"`.

Format rules (spec "TSV import" section):
- `lemma<TAB>word[<TAB>tags]`, one entry per line; tab-separated.
- Missing lemma → auto-lemma (`lemma = word`); missing tags → `""`.
- Blank lines and lines whose first non-space byte is `#` are skipped.
- Leading/trailing space trimmed from each field.
- 1 column or >3 columns → error **naming the line number**.
- Streaming scan with `bufio.Scanner` + the same 1MB max-line buffer as
  unimorph (importers/unimorph/import.go:236-237).

- [ ] **Step 1: Write the failing tests**
  `pkg/morphology/import_tsv_test.go`: a 3-column happy path (lemma-first)
  parses back via `Parse`; 2-column path auto-lemmas; `#` comments and blank
  lines skipped; surrounding spaces trimmed; 1-column and 4-column rows →
  errors naming the line number; result is dense with prediction rebuilt;
  `SaveTo`/`Open` round-trip.
- [ ] **Step 2: Implement `import_tsv.go`**
- [ ] **Step 3: Verify** — `go test ./pkg/morphology/ -race`; lint.

---

## Task 6: `morphology.Merge` (add/replace)

**Files:**
- Create: `pkg/morphology/merge.go`
- Test: `pkg/morphology/merge_test.go`

**Interface** (spec):
- `type MergeMode int` with `MergeAdd`, `MergeReplace`.
- `func Merge(base *Dictionary, overlays []*Dictionary, mode MergeMode) (*Dictionary, error)`
  — inputs read-only (only walked), never mutated; result is dense with
  prediction rebuilt; `Info`: `Source == "merge"`, `SourceVersion`
  and `Description` copied from `base.d.Info`; `Language`/`CharPolicy` from
  base.

Enumeration of one input dictionary (spec §Merge):
1. For each shard `i` in `d.d.Words`: `d.d.Words[i].Walk(func(word string, vals [][]byte))`.
2. Word → text: `d.d.Alphabet.Decode([]byte(word))` when
   `d.d.Alphabet != nil`, else the walked `word` as-is. (Merge must accept
   both dense and raw inputs.)
3. Per value (skip `< 4` bytes, the engine's own guard, parse.go:195):
   `d.reading(shard, word, val)` (parse.go:194) → `Reading{Word, Normal,
   Tag}` — lemma reconstructed by the already-shipped `readingForm`.
4. Accumulate into an entry map keyed by `word`:
   - base: every word's readings collected.
   - overlays, in order, under `mode`:
     - `MergeAdd`: overlay readings kept only for words **not** in the base
       word set (a word present in base is skipped entirely, even its new
       readings).
     - `MergeReplace`: for words in both, the overlay's readings fully
       replace the base's; words unique to either side preserved.
   - overlay-vs-overlay: overlays just accumulate (later overlays' readings
     for the same word are appended; a later `replace` semantic across
     overlays is out of scope — spec non-goal).
5. Feed all resulting `(Word, Normal, Tag)` entries into the shared
   post-process helper (Task 4) → dense + prediction rebuilt (prediction
   only if the merge output is 1 shard).

Determinism: overlay application order == argument order; Walk order is
deterministic (double-array DFS); the final entry list order must be stable
(sort by word, then insertion, or preserve a deterministic build order; the
helper's paradigm grouping is by lemma in entry order — document the chosen
order in a comment).

- [ ] **Step 1: Write the failing tests**
  `pkg/morphology/merge_test.go` (package `morphology_test`):
  - Build dicts via the public Builder (Task 4) — no importers needed.
  - add: overlay-only word added with its own `(lemma, tag)`; a word in both
    keeps **only** base readings (overlay readings for it dropped).
  - replace: a word in both keeps **only** overlay readings; words unique to
    either side preserved; prediction rebuilt and verified via `Parse` on an
    out-of-dictionary word.
  - multiple overlays applied in order.
  - dense round-trip: two dense Builder-produced inputs → dense output
    (`d.d.Alphabet != nil`); `Fuzzy` works; `Parse` on the merge result
    equals `Parse` on the equivalent raw-entries Builder dict.
  - inputs not mutated: base's `Parse` output unchanged after `Merge`.
- [ ] **Step 2: Implement `merge.go`**
- [ ] **Step 3: Verify** — `go test ./pkg/morphology/ -race`; lint.

---

## Task 7: CLI `import tsv` + `merge`

**Files:**
- Create: `cmd/gomorphy/import.go`
- Rewrite: `cmd/gomorphy/merge.go` (replace the stub, main.go:28 entry stays)
- Edit: `cmd/gomorphy/merge_test.go` (drop the "not yet implemented" test),
  add `cmd/gomorphy/import_test.go`
- Edit: `cmd/gomorphy/main.go` only if a new command needs registration
  (`newImportCommand` added next to `newMergeCommand(), newSplitCommand()`,
  main.go:28)

Commands (mirror `newBuildCommand`, build.go:88; use `configureLogging`,
`cmd.OutOrStdout()`, and existing `-i/-o` flag style):

```
gomorphy import tsv <file> [-o out.dat] [--source name]
gomorphy merge --mode add|replace [-o out.dat] base.dat overlay.dat [overlay.dat …]
```

- `import` parent command with a `tsv` subcommand (`Args: ExactArgs(1)`).
  Opens `<file>`, `morphology.ImportTSV(f, BuilderOptions{Source: source})`,
  `SaveTo(-o)`, prints `saved <output>\n` (the build.go:84 pattern).
- `merge`: `Args: cobra.MinimumNArgs(2)`; `-o` required; `--mode` required,
  validated case-insensitively against `add`/`replace`; open each
  `base.dat`/`overlay.dat` via `morphology.Open` (tracking `.Close()`s),
  `morphology.Merge(base, overlays, mode)`, `SaveTo(-o)`, print `saved ...`.
- Both: if `-o` empty → error naming that `-o` is required (spec).
- `-d/--dictionary` global flags are irrelevant here (no lookup) — no wiring.

- [ ] **Step 1: Write the failing tests**
  Fixture `.dat`s built through the public `morphology.Builder` + `SaveTo`
  (the `buildFixtureDat`-style helper in dict_test.go:44 but with the
  Builder — keep the XML fixture for `opencorpora`-based tests untouched).
  - `import_test.go`: `import tsv <file> -o out.dat` produces a `.dat` that
    `lookup`/`morphology.Open` reads; `--source custom` lands in
    `BuildInfo.Source`; missing `-o` → error; bad input path → error;
    malformed TSV row → error naming the line.
  - `merge_test.go` (rewrite): `merge --mode add -o out.dat a.dat b.dat`
    reflects add semantics in `lookup` output (or in `Open`+`Parse`);
    `--mode replace` likewise; <2 args → cobra `Args` error; missing/unknown
    `--mode` → error; `-o` same as a `.dat` input → rejected (must not
    silently overwrite an input).
- [ ] **Step 2: Implement both commands**
- [ ] **Step 3: Verify** — `go test ./cmd/gomorphy/ -race`; `go build ./...`;
  `golangci-lint run ./cmd/gomorphy/...`. Smoke-test the real binaries:
  `go run ./cmd/gomorphy import tsv -o /tmp/g.dat <sample>` and
  `go run ./cmd/gomorphy merge --mode replace -o /tmp/m.dat <base> <overlay>`,
  then `go run ./cmd/gomorphy -d /tmp/g.dat lookup <word>`.

---

## Task 8: docs + full verification

**Files:**
- Edit: `docs/cli.md` — add `import tsv` and `merge` to the command
  reference (flags, examples, the "-o required" rule).
- Edit: `docs/en/todo.md` — tick the Stage 19 boxes this pass delivers
  (TSV import, Builder API, merge) and the lifecycle note; leave the
  probability and tag-normalization items open.
- Create: `docs/en/implementation/stage-19-builder-tsv-merge.md` — a short
  implementation note (what shipped, the raw→post-process split, the
  BuildPrediction-on-raw-then-RecompileDense order, the injected
  `productive` rationale, merge add/replace semantics, links to the design
  spec and tests).
- Update the `pkg/morphology` package doc comment in `dictionary.go:1-4`
  if it still says UniMorph import is "planned but not yet implemented"
  (it now is, and Builder/Merge/TSV exist).

- [ ] **Step 1: Write the docs** per the repo's doc conventions.
- [ ] **Step 2: Full verification** — `go build ./... && go vet ./... &&
  go test ./... -race`; `golangci-lint run ./...`; confirm the split
  command still reports "not yet implemented" (stub unchanged).
- [ ] **Step 3: Self-review** the diff against the spec's Testing section —
  every bullet has a corresponding test; re-read the spec for any missed
  item (merge BuildInfo copying, `Source` overrides, dense-by-default).