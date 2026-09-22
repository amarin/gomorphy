# Builder API + TSV import + `Merge`: building dictionaries from wordform entries

## Context

The library can only *produce* dictionaries through the three importers —
opencorpora (`pkg/morphology/importers/opencorpora/import.go`), pymorphy2
(`pkg/morphology/importers/pymorphy2/import.go`), unimorph — each of which
privately re-implements the same pipeline (intern suffixes/tags → shunt into
shards once the uint16 suffix space fills → dedup paradigms → `BuildDAWGWithValues`
→ `internal.NewDictionary`). There is no public way to assemble a dictionary
from raw wordform entries, `cmd/gomorphy`'s `merge`/`split` are "not yet
implemented" stubs, and `docs/en/todo.md` Stage 19 ("Thematic dictionaries")
has long planned a TSV import path (`lemma<TAB>word[<TAB>tags]`, optional/
opaque tags, auto-lemma, dedup).

Two capabilities requested in this pass — *create a dictionary from scratch*
and *fill it with grammemes*; and *merge dictionaries with an add/replace
conflict policy* — reduce to the **same** core operation: build an
`internal.Dictionary` from a set of wordform-level entries
`(wordform → lemma, tag)`. A break-before-build merge enumeration is already
possible with shipped primitives: `internal.DAWG.Walk` enumerates every
`(word, values)` pair of a `.dat`'s words.dawg (dawg.go:281), `Dictionary.reading`
+ `readingForm` (parse.go:194, 206) reconstruct the `(lemma, tag)` per
reading, and `DenseAlphabet.Decode` (internal/alphabet.go:183) converts dense
encoded keys back to text. `Fuzzy`/`FuzzyTop` (fuzzy.go:34,53) are
alphabet- and shard-aware and depend on neither prediction nor probability,
so they work on any built dictionary out of the box.

Brainstorming decisions locked in this session:

- The Builder API is the core; TSV import and (the far future) an interactive
  console are thin layers over it. **Approach A**: a fresh internal entry →
  dictionary pipeline, the three tested importers left untouched.
- Merge conflict policy is word-level: **add** (only wordforms absent from
  the base get added; words already present are skipped entirely) and
  **replace** (for words in both, the overlay's readings fully replace the
  base's).
- Merge accepts **only compiled `.dat` files** (a TSV→.dat step is explicit;
  `import tsv` builds, `merge` combines).
- Tags for self-made dictionaries are **opaque free-form strings**, auto-
  registered into the TagSet — matching the existing `TagSet.Add` semantics
  and Stage 19. No predeclared grammeme list, no tagmap-normalization table
  for these (normalization stays limited to known sources).
- **Prediction IS rebuildable** from a dictionary's own wordforms+paradigms
  and is rebuilt by the shared pipeline. **Probability is NOT** — it is
  external corpus-frequency data (`p_t_given_w.intdawg`); built dictionaries
  carry none, and the rebuild/carry question enters the roadmap.
- Output dictionaries are **dense 1-byte alphabet by default**, matching the
  existing CLI policy (build.go:38-73; the raw variant stays Go-API-only).

## Decision

A new public `morphology.Builder` collects `(word, lemma, tag)` entries and
builds a `*Dictionary` through a new shared internal pipeline —
`internal.BuildDictionaryFromEntries` (fresh implementation mirroring the
opencorpora importer's proven phases; **no refactor of the existing
importers**). `import_tsv.go` (streaming TSV → Builder) and `merge.go`
(break every `.dat` back into entries via `Walk`+`readingForm`+`Alphabet.Decode`,
apply the add/replace policy, feed the same pipeline) both sit on it.
Prediction is rebuilt for every output dictionary by default.

## Design

### Public API — `pkg/morphology/builder.go` (new)

```go
// BuilderOptions — optional dictionary-level metadata for a built dictionary.
type BuilderOptions struct {
    // Language code for the dictionary's "meta" section (default "ru").
    Language string
    // Source fills BuildInfo.Source; defaults to "builder" (TSV import
    // overrides with "tsv"). Should reflect what the entries came from.
    Source string
}

// NewBuilder returns a Builder that accumulates wordform entries.
func NewBuilder(opts BuilderOptions) *Builder

// AddForm registers one entry: wordform text "word", its lemma "lemma",
// and an opaque grammeme tag "tag". tag may be "" (a reading with no
// grammemes). An empty word is an error (as are whitespace-only words, per
// the TSV trim rule). An empty lemma means the wordform is its own lemma
// (auto-lemma). Case is left to the caller, matching the importers.
func (b *Builder) AddForm(word, lemma, tag string) error

// AddLemma is sugar for AddForm(normal, normal, tag): the lemma is also a
// wordform of itself.
func (b *Builder) AddLemma(normal, tag string) error

// Build assembles the dictionary from all registered entries (deduplicating
// (word, lemma, tag) triples), rebuilding prediction. The result is a fully
// functional *Dictionary (Parse, Lemma, Fuzzy, prediction) usable directly
// or Savable via SaveTo. Build is not idempotent-friendly: it consumes the
// Builder (further AddForm calls after Build are rejected).
func (b *Builder) Build() (*Dictionary, error)
```

`Build` and its consumers never lower-case input, never sort/reshape tags,
never require a paradigm shape — one entry per call is fully general
(a one-form "paradigm" of a single wordform is valid; Russian CharPolicy is
applied via `internal.RussianCharPolicy()`, matching every importer).

### TSV import — `pkg/morphology/import_tsv.go` (new)

Format, exactly as Stage 19 planned:

```
# optional comment line
кошка  кошка  NOUN,femn,sing,nomn
кошка  кошкой NOUN,femn,sing,ablt
дело   дело
дело   делом  NOUN,neut,sing,ablt,...
```

- `lemma<TAB>wordform[<TAB>tags]`, one entry per line; tab-separated.
- Missing lemma → auto-lemma (lemma = wordform). Missing tags → `""`.
- Blank lines and lines whose first non-space byte is `#` are skipped.
- Trimming: leading/trailing space is trimmed from each field (tabs are the
  delimiters; stray spaces around fields are tolerated).
- Anything else — 1 column, or >3 columns — is an error **naming the line
  number**.

```go
// ImportTSV reads TSV entries from r and builds a dictionary (dense
// alphabet, prediction rebuilt) via the same Builder pipeline.
func ImportTSV(r io.Reader, opts BuilderOptions) (*Dictionary, error)
```

A file wrapper lives in the CLI (opens the file, passes the reader) — no
`ImportTSVFile` needed in the library for this pass.

### Merge — `pkg/morphology/merge.go` (new)

```go
// MergeMode — conflict policy applied to every overlay, at word level.
type MergeMode int

const (
    // MergeAdd: entries for wordforms absent from the base are added;
    // wordforms already present in the base are skipped entirely (not even
    // their new readings are merged in).
    MergeAdd MergeMode = iota
    // MergeReplace: for wordforms present in both, ALL base readings for
    // that word are dropped and only the overlay's kept; wordforms unique
    // to either side are preserved.
    MergeReplace
)

// Merge rebuilds a dictionary from base and overlays under mode. Inputs are
// only ever *read* (Walk) — never mutated — so the same *Dictionary may be
// used as both an input and (via SaveTo on the result) a persistence target.
// Any number of overlays may be passed; they apply in order, all under the
// same mode. Output is dense and has prediction rebuilt. BuildInfo of the
// result: Source="merge", SourceVersion/Description copied from base.
func Merge(base *Dictionary, overlays []*Dictionary, mode MergeMode) (*Dictionary, error)
```

Merge enumeration of a single input dictionary:

1. For each shard `i` in `d.d.Words`: `d.d.Words[i].Walk(func(word string,
   vals [][]byte))`.
2. Word key → text: `d.d.Alphabet.Decode([]byte(word))` when
   `d.d.Alphabet != nil`, else `word` as-is.
3. Each `val` (skipping `< 4` bytes, the engine's own guard, parse.go:195):
   `para = BE16(val[:2])`, `form = BE16(val[2:4])`, then the existing
   `d.reading(shard, word, val)` (parse.go:194) → `Reading{Word, Normal,
   Tag}` — the lemma is reconstructed by the already-shipped `readingForm`
   logic (parse.go:206); no new paradigm math.
4. Feed `(Word, Normal, Tag)` into the policy accumulator: an entry map
   keyed by `word`, with the base's words pre-collected into a set for the
   add/replace decision. `MergeAdd` adds overlay entries only for words not
   in the base word set; `MergeReplace` replaces each overlapping word's
   whole reading list with the overlay's.

### Shared internal pipeline — `pkg/morphology/internal/build.go` (new)

`internal.BuildDictionaryFromEntries(opts, entries)` — a fresh implementation
of the phases the opencorpora importer already performs (import.go:196-381),
generalized to arbitrary entries:

1. **Intern**: build prefix/suffix/suffix-shard maps from each entry's
   `word` vs its lemma stem; tags via `TagSet.Add` (dedup by name,
   `ErrTagSetFull` on the 65536 cap — propagated).
2. **Shard**: once a shard's suffix count would exceed `1 << 16`
   (`suffixShardLimit`), open the next shard by the existing
   `FillOnDemand.Boundary` strategy (shard.go) — same as opencorpora. For
   the typical small thematic dictionary this stays 1 shard.
3. **Paradigms**: group entries **by lemma** — one paradigm per lemma, its
   forms being that lemma's entries in insertion order (per-form
   `(suffix_id, tag_id)` in the flat uint16 `internal.Paradigm` arrays — the
   importers' model: opencorpora import.go:236-327 and the near-identical
   unimorph copy). Form 0 of each paradigm **must be the lemma text**: that
   is what `readingForm` (parse.go:206) reconstructs `Normal` from for every
   other form. If no entry of a lemma has `word == lemma`, form 0 is
   synthesized as the lemma text with an empty tag — the unimorph
   `buildForms` pattern (importers/unimorph/import.go:176-194). Identical
   paradigms collapse via a `paradigmKeyHash`-style dedup (import.go:56).
4. **DAWG**: `BuildDAWGWithValues` per shard, value
   `paraID<<16|formIdx`; assemble via `internal.NewDictionary`; then
   `internal.RecompileDense` (dense_recompile.go:20) for the 1-byte dense
   output and `internal.BuildPrediction`.
5. Attach `Info` (`Source` from BuilderOptions / "merge").

The lemmas-splitting and paradigm-merge link logic of the opencorpora
importer (mergeLinkedLemmas, stripCmp2Prefix, lcp) stays there — the builder
works from pre-merged, single-lemma entries and needs none of it.

### Prediction rebuild — `pkg/morphology/internal/prediction.go` (new)

`internal.BuildPrediction(d)` implements pymorphy2's KnownSuffixAnalyzer in
the exact on-disk format the engine already consumes
(`predictForPrefix`, parse.go:146: each DAWG value is ≥6 bytes —
`count(BE16) + para(BE16) + form(BE16)`, filtering non-productive tags via
`productive()`, parse.go:275). One prediction DAWG per prefix id. Engine
constraint honored: predictions always resolve against **shard 0**
(parse.go:144), so `BuildPrediction` is applied only to unsharded
dictionaries (the builder/TSV/small-merge case); a sharded merge output
keeps prediction absent — identical to OpenCorpora dictionaries today.

### CLI — `cmd/gomorphy`

`import.go` and `merge.go` become real commands (mirroring `newBuildCommand`,
build.go:88, flag style `-i/-o`, `configureLogging`, default `-o` paths
under `.data/`):

```
gomorphy import tsv <file> [-o out.dat] [--source name]
gomorphy merge --mode add|replace [-o out.dat] base.dat overlay.dat [overlay.dat …]
```

- `-o` is required for both (`import tsv` has no default domain to fall back
  to, and `merge` must never silently overwrite an input — see below).
- Dense by default (the agreed CLI policy); the raw variant remains
  Go-API-only.
- `--mode` is a required flag (no ambiguous default); modes are
  case-insensitive (`add`/`replace`).
- Minimum two positional args for `merge` (base + ≥1 overlay), validated by
  `Args:`.
- Both report `progress` (the existing `newProgressReporterTo` /
  `isInteractive` pattern) where the pipeline supports it.
- `split` remains a stub in this pass.

## Non-goals

- **Refactoring the opencorpora/pymorphy2/unimorph importers onto the shared
  pipeline** (brainstorming Approach B) — deliberately deferred; the three
  tested importer paths stay untouched.
- **Probability for built dictionaries** — not rebuildable from wordforms;
  built/merged dictionaries carry none (exactly like OpenCorpora dicts
  today). Roadmap TODO: Builder/TagSet frequency input and a merge
  carry-forward policy.
- **Tag normalization** (`tagmap`) for self-made dictionaries — tags are
  opaque; a custom TagSet name is simply absent from the tagmap registry.
- **Batch query modes, the import report, and the Stage 19 agent skill** —
  separate Stage 19 deliverables, out of this pass.
- **Per-overlay merge modes** — one `MergeMode` for the whole call; a
  per-overlay mode array is a later additive refinement.
- **Non-dense CLI output** for `import tsv` / `merge` — CLI is dense-only
  by agreed policy (build.go:38-42).
- **Prediction for sharded merge outputs** — engine resolves prediction
  against shard 0 only (parse.go:144); out of scope (OpenCorpora ships none
  either).
- **`split` command, source-dir merge inputs, interactive console records
  interface account** — not in this pass.
- **Duplicate-word behavior in `AddForm`** — dedup by `(word, lemma, tag)`
  triple at Build, matching `dedupEntries` semantics (import.go:600); a
  duplicate word with a *different* tag/lemma is a second reading (homonymy),
  kept.

## Testing

- **`builder_test.go`** (`pkg/morphology/`): build a small multi-lemma
  dictionary; `SaveTo` → `Open` round-trip; `Parse` returns the expected
  `(Word, Normal, Tag)`/count; auto-lemma sets `Normal == Word`; triple dedup
  collapses exact repeats but keeps homonym readings; empty word form → error;
  `Build` after `AddForm` post-`Build` → error.
- **Prediction** (`prediction_test.go`): a small crafted corpus; `Parse` on
  an out-of-dictionary word with a productive ending returns a predicted
  reading with the right lemma/tag; non-productive tags are excluded.
- **`import_tsv_test.go`**: 3-column / 2-column happy paths; `#` comments and
  blank lines skipped; surrounding spaces trimmed; 1-column and 4-column
  rows → errors naming the line number; result dictionary parses back.
- **`merge_test.go`**:
  - add: overlay-only word added with its own `(lemma, tag)`; a word in both
    keeps *only* base readings (overlay readings for it dropped).
  - replace: a word in both keeps *only* overlay readings; words unique to
    either side preserved; `Prediction` rebuilt and working on the result.
  - multiple overlays applied in order.
  - dense round-trip: two dense inputs → dense output; `Fuzzy` works;
    `Parse` equals the equivalent raw-entries merge.
- **CLI** (`cmd/gomorphy/*_test.go`): fixture dictionaries built via the
  builder; `import tsv` produces a `.dat` that `lookup` reads;
  `merge --mode add|replace` over two fixture files reflects the mode in
  `lookup` output; argument validation (`merge` with <2 args,
  `import tsv` with a bad path).
- `go test ./...` and `go vet ./...` green; `golangci-lint run ./...`
  clean on touched packages (pre-existing unrelated issues excluded).