# Stage 16. UniMorph import

> **Status: DONE, 2026-09-19.** Implemented per the 2026-09-16 Q&A
> decisions (see `docs/en/research/0006-unimorph-import-plan.md`) with
> two deltas found during implementation, both recorded below: the
> CLI syntax follows the `build`/`download`/`unpack`/`update <type>`
> shape (the 2026-09-16 CLI redesign, not the `import <type>` shape
> this draft originally assumed), and `gomorphy build unimorph`
> defaults to a dense 1-byte alphabet (a policy that didn't exist yet
> when this draft or the Q&A were written).

## Stage contents

Importing a UniMorph dictionary from a TSV file
(`lemma<TAB>wordform<TAB>bundle`) as an additional data source. Similar
to Stage 15 (OpenCorpora import), but simpler: no XML, lines are parsed
in a stream. The full format analysis is in
[docs/unimorph.md](../unimorph.md).

Reference data: `unimorph/rus` (473,482 rows, 28,069 lemmas, 353,004
wordforms, CC-BY-SA 3.0 license, `github.com/unimorph/rus`).

### Key decisions (fixed in the 2026-09-16 Q&A)

1. **Lemma = form 0 of the paradigm (always).** If a row with
   `form == lemma` exists, its bundle becomes form 0's tag. If not,
   synthesize form 0: text = lemma, tag = `""`. The lemma's text is
   always included in `lcp()`'s input (it always equals form 0's
   text), which ensures the base form is reconstructed correctly for
   suppletive groups ("я"/"меня", "человек"/"люди").
2. **Tags are opaque strings.** A bundle (e.g. `N;ACC;SG`) is stored in
   the TagSet verbatim; reordering and mapping onto OpenCorpora tags
   are **not implemented** in Stage 16 (deferred — see
   [tag-mapping.md](tag-mapping.md)'s "What's left").
3. **Language and CharPolicy.** Only `ru` is accepted
   (`Options.Language`); any other value, including `""`, errors out —
   "mandatory" was read literally, not as "defaults to ru". CharPolicy
   is picked by language: `ru` -> `internal.RussianCharPolicy()`. The
   code is structured with room for future languages, not a promise
   that they work.
4. **Malformed rows (!= 3 fields).** `OnMalformed(lineNumber, text)` is
   called (logging is the caller's choice), but the import continues —
   the current row is skipped. An empty bundle (the third column) is
   allowed (an empty `""` tag).
5. **The data loader** is `pkg/unimorph`, modeled on `pkg/pymorphy`,
   not a manual download.
6. **A repeated `(form, bundle)` row within one lemma is deduplicated**
   at accumulation time (found while implementing — see "What was
   found during implementation" below).

### The importer (`pkg/morphology/importers/unimorph/`)

```go
func ImportFromTSV(
    r io.Reader,
    tagSet *internal.TagSet,   // nil -> NewTagSet("unimorph")
    opts Options,
) (*internal.Dictionary, error)

func CompileFromTSV(r io.Reader, opts Options) (*internal.Dictionary, error)
func CompileFromTSVFile(path string, opts Options) (*internal.Dictionary, error)
```

`Options`:
```go
type Options struct {
    Language      string                            // mandatory, only "ru" accepted
    CharPolicy    *internal.CharPolicy               // nil -> inferred from Language
    OnMalformed   func(lineNumber int, text string)  // nil -> skip silently
    Progress      Progress                           // nil -> no progress reporting
    SourceVersion string                             // nil -> BuildInfo.SourceVersion left empty
}
```

#### Pipeline

1. **Reading.** A `bufio.Scanner` with a 1 MB buffer (for compound/
   hyphenated tokens; `\r` is already stripped by `bufio.ScanLines`); a
   row is exactly 3 `\t`-separated fields. A row with an empty lemma ->
   lemma := wordform. Empty lines are skipped (no error).
2. **Accumulation.** Data is grouped into `map[lemma]*lemmaEntry` (28K
   entries for `rus` — cheap); a lemma's row order isn't guaranteed by
   the file, but the lemma's own *first-appearance line* fixes its
   position in a separate `order []string` slice, so shard assignment
   (and therefore the resulting `.dat`, byte for byte) is deterministic
   across repeated builds of the same input — not dependent on Go's
   randomized map iteration order.
3. **Form 0.** For each lemma, in `order`:
   - Find the row with `text == lemma` -> place it at forms position 0
     (first occurrence). Other rows with the same text are ordinary
     forms (syncretism of the lemma's form).
   - If no row has `text == lemma`, synthesize form 0: `{text: lemma,
     bundle: ""}`.
4. **Stem.** Compute the (rune-safe) LCP over the list of forms
   (including form 0 — whose text always equals the lemma). Each
   form's suffix is `word[len(stem):]`.
5. **Tags.** Each form's bundle -> `tagSet.Add(bundle)` (an opaque
   string; no reordering is performed).
6. **Paradigm dedup.** Suffixes and tags (prefixes are always `""` for
   UniMorph — there's no separable-prefix rule like OpenCorpora's
   Cmp2) are identified by integer IDs. Paradigms are deduplicated
   (`paradigmKeyHash`); sharding is `FillOnDemand` (a limit of 65,536
   suffixes per shard) — both are deliberate copies of the OpenCorpora
   importer's identical helpers (see Q3: a shared TSV engine is
   deferred to Stage 19, not built for a single second consumer).
7. **DAWG.** The key is `stem+suffix` (= the wordform for UniMorph, no
   prefix), the value is `paraID<<16|formIdx`. Built via
   `internal.BuildDAWGWithValuesProgress`.
8. **Assembly.** `internal.NewDictionary("ru", tagSet, suffixes,
   prefixes, paradigms, dawg, charPolicy)`, `Info.Source = "unimorph"`.

#### Handling UniMorph data

| Case | Handling |
|---|---|
| Syncretism (one text with different bundles) | Native: several wordform readings |
| Wordform == lemma | Form 0 gets its bundle; others are ordinary forms |
| Wordform == lemma (several rows) | The first -> form 0; the rest are ordinary forms (lemma-form homonyms) |
| Empty lemma | Lemma := wordform (form 0 = the wordform, tagged with its bundle) |
| Multi-word/hyphenated tokens | The library works with arbitrary bytes |
| `LGSPEC*`/unknown features | An opaque grammeme, not an error |
| Suppletion ("я"/"меня", "человек"/"люди") | Handled correctly: the stem's LCP runs over forms+lemma (form 0 included), the base form = lemma |
| No `form == lemma` row | Form 0 is synthesized: text=lemma, tag=`""`; adds up to ~28K keys to the DAWG |
| A repeated `(form, bundle)` within a lemma | Deduplicated explicitly at accumulation time (`lemmaEntry.addRow`) — see "What was found" below |
| Malformed rows (!= 3 fields) | `OnMalformed` called, row skipped; import continues |
| An empty bundle | An empty `""` tag in the TagSet — allowed |
| Sharding for `rus` | **2 shards** in practice (measured — see "What was found" below), not the 1 shard originally estimated |

### Public API (`pkg/morphology/`)

```go
type UniMorphOptions = unimorph.Options

func CompileFromUniMorph(r io.Reader, opts UniMorphOptions) (*Dictionary, error)
func CompileFromUniMorphFile(path string, opts UniMorphOptions) (*Dictionary, error)
func CompileFromUniMorphDense(r io.Reader, opts UniMorphOptions) (*Dictionary, error)
func CompileFromUniMorphFileDense(path string, opts UniMorphOptions) (*Dictionary, error)
```

Wrappers over `unimorph.CompileFromTSV`; the `Dense` variants recompile
under one shared dense 1-byte alphabet afterward
(`internal.RecompileDense`, see
[pymorphy2-dense-alphabet.md](pymorphy2-dense-alphabet.md)) — the same
source-agnostic mechanism `CompileFromXMLDense`/`OpenPyMorphyDense`
already use, applied here to UniMorph's (up to 2, for `rus`) shards.

### CLI

```bash
gomorphy download unimorph --lang ru      # -> .data/unimorph/ru/data
gomorphy build unimorph -i rus.tsv -o ru-unimorph.dat --lang ru
gomorphy update unimorph --lang ru        # download + unpack (no-op) + build, one step
gomorphy -d ru-unimorph.dat lookup кота   # -> N;ACC;SG
```

`--lang` defaults to `ru` (the only accepted value today) and is
registered on `build`/`download`/`unpack`/`update`, consulted only for
`typ == "unimorph"`. `build unimorph` always produces a dense 1-byte
alphabet, no opt-out flag — the same policy `build
opencorpora`/`build pymorphy` already have (see
[pymorphy2-dense-alphabet.md](pymorphy2-dense-alphabet.md)'s "Agreed
design decisions", item 2). The raw/non-dense variant is Go-API-only
(`morphology.CompileFromUniMorph` directly).

`unpack unimorph` is a no-op that just confirms the download exists —
there is no archive to extract for this source (the raw GitHub file is
already the usable TSV), kept only so every source type goes through
`download`/`unpack`/`build`/`update` uniformly.

## What was found during implementation (2026-09-19)

Two things the 2026-09-16 research/Q&A didn't get exactly right, found
and resolved while implementing (not left as open questions — both are
settled, recorded here for the historical record):

1. **UniMorph's own language codes are ISO 639-3, not gomorphy's
   2-letter codes.** The Q&A's URL template
   (`https://raw.githubusercontent.com/unimorph/<lang>/master/<lang>`)
   didn't specify which convention `<lang>` used, and gomorphy's own
   `-lang ru` naturally suggested "ru" — but UniMorph's GitHub
   organization names every repository after its ISO 639-3 code
   (verified live: `github.com/unimorph/rus`, `github.com/unimorph/eng`,
   `github.com/unimorph/ukr`, etc. — there is no `github.com/unimorph/ru`).
   `pkg/unimorph`'s `isoCodes` map resolves this: gomorphy's own code
   (`"ru"`, used everywhere else in this codebase — `Dictionary.Language`,
   `RussianCharPolicy`, the CLI's `--lang`) maps to UniMorph's
   (`"rus"`), used only to build the download URL. A real download
   during implementation confirmed the mapping: 473,482 rows,
   ~24 MB, matching this document's cited figures exactly.
2. **A repeated `(form, bundle)` row within one lemma isn't actually
   collapsed by paradigm dedup**, contrary to what the original
   research assumed ("a repeated (form, bundle) within a lemma
   collapses via paradigm dedup and dedupEntries"). Paradigm dedup
   (`paradigmKeyHash`) collapses identical whole-paradigm *shapes*
   across different lemmas — it has no effect on two identical rows
   *within* the same lemma's own form list, which would otherwise
   become two distinct form slots (different `formIdx`) with identical
   suffix/tag content, and `dedupEntries` (which dedupes DAWG entries
   by exact `(key, value)`) wouldn't catch it either, since the two
   slots' `value`s differ by `formIdx`. Fixed by deduplicating
   `(text, bundle)` pairs explicitly at accumulation time
   (`lemmaEntry.addRow`), before a lemma's forms ever reach paradigm
   construction — verified with a dedicated regression test
   (`TestImportFromTSV_DuplicateRowWithinLemmaCollapses`).

Real-data measurements (`go test -tags=integration
./pkg/morphology/importers/unimorph/...`, the full `rus` dictionary):
**2 shards** (not the 1 shard the original research estimated —
that estimate predates the within-lemma dedup fix above, among other
implementation details), 6,240 unique paradigms, 102 unique bundles,
~35s import time on this session's machine.

## Verification (tests)

All done — see `pkg/morphology/importers/unimorph/import_test.go`
(unit), `pkg/morphology/importers/unimorph/real_dict_integration_test.go`
(gated, real `rus`), `pkg/unimorph/loader_test.go` +
`loader_integration_test.go`, `pkg/morphology/unimorph_test.go` (public
API, dense variant, `SaveTo`/`Open` round-trip), and
`cmd/gomorphy`'s `build_test.go` / `pkg/morphology/cli_integration_test.go`
(CLI, including through the actual compiled binary).

- Unit: a mini TSV -> correct lemmas and paradigms; a synthetic form 0;
  a real row at `form == lemma` becoming form 0; suppletion; empty
  lemma; malformed rows -> `OnMalformed` + import continues; empty
  bundle allowed; duplicate `(form, bundle)` row collapses; unsupported
  (including empty) language errors; empty input.
- Roundtrip: `CompileFromUniMorphDense -> SaveTo -> Open` gives
  identical `Parse()` results.
- Dense: `CompileFromUniMorphDense` gives identical readings to
  `CompileFromUniMorph` on the same input.
- Integration (gated, `-tags=integration`): the full real `rus` ->
  import succeeds, `"кота"` is found, paradigm/tag counts are sane.
- CLI: `build unimorph` with a real fixture TSV, `--lang` validation,
  and (through the compiled binary) confirming the built `.dat`
  actually carries the dense `"alphabet"` section.
- `go build ./...`, `go vet ./...`, `go test ./... -race` (329/329),
  `golangci-lint run ./...` — all green throughout.

## Manual verification

```bash
gomorphy download unimorph                          # .data/unimorph/ru/data
gomorphy build unimorph -o ru-unimorph.dat           # dense by default
gomorphy -d ru-unimorph.dat lookup кота              # N;ACC;SG
gomorphy build unimorph --lang en -o en.dat          # error: unsupported language "en"
```

## Deliberately not part of this stage

- **Tag mapping** (`pkg/morphology/tagmap`'s `"unimorph"` table) —
  called "trivial" once the importer exists (see
  [tag-mapping.md](tag-mapping.md)'s "What's left"), deliberately kept
  as a separate follow-up per this session's own decision, not bundled
  into Stage 16.
- **A shared TSV import engine** (Stage 19) — `lcp`/`paradigmKeyHash`/
  `FillOnDemand` are copies of the OpenCorpora importer's identical
  helpers, not shared code (Q3).
- **Any language other than `ru`** — the code has room to grow
  (`Options.Language`, `pkg/unimorph.isoCodes`) but nothing beyond `ru`
  is wired up or tested.
