# UniMorph import plan and analysis (Stage 16): draft vs. architecture gaps, design decisions, scope

**Date:** 2026-09-16
**Status:** analysis done, design decisions agreed with the user (Q&A) — see "Decisions made"
**Analyzed by:** Aleksey Marin (a session with an agent)
**Related documents:** [implementation/stage-16-import-unimorph.md](../implementation/stage-16-import-unimorph.md) (the draft), [unimorph.md](../unimorph.md) (format analysis), [implementation/stage-15-import-opencorpora.md](../implementation/stage-15-import-opencorpora.md) (the reference importer), [implementation.md](../implementation.md), [todo.md](../todo.md)

## Context

The task: analyze the Stage 16 draft (importing the UniMorph
dictionary from TSV) and put together a short implementation plan,
listing open questions for further discussion and estimating the
scope of work. The draft `stage-16-import-unimorph.md` refers to the
Builder API (`AddLemma`/`AddForm`), `Options.Mapping`, public
`CompileFromUniMorph*` — some of this no longer exists in the current
codebase; the analysis below cross-checks the draft against the actual
code and records what needs to be rewritten, what needs clarifying,
and what it costs.

Reference data: `unimorph/rus` — 473,482 lines, 28,069 lemmas, 353,004
unique wordforms, license CC-BY-SA 3.0 (the full table is in
[unimorph.md](../unimorph.md) §4).

## Current code snapshot (verified by reading, 2026-09-16)

What's implemented in the repository right now, and what the importer
relies on:

1. **The Builder API (`AddLemma`/`AddForm`) does NOT exist** — neither
   in `pkg/morphology` nor in `internal` (a code grep is empty). The
   reference OpenCorpora importer builds the dictionary directly:
   `lemmaEntry` -> LCP-stem of the forms -> (suffix_id, tag_id) ->
   paradigm dedup -> a DAWG with payload, with sharding by a suffix
   limit (`importers/opencorpora/import.go:112-291`, `shard.go`). The
   Stage 16 draft describes the **old** architecture (Stages 4-10,
   `pkg/dictionary` + Builder); this is also why `unimorph.md` §5.1
   has a stale mention of `pkg/dictionary`.

2. **A "lemma" in the current model is a paradigm's form 0.**
   `readingForm` (`pkg/morphology/parse.go:201-226`): for `form==0`
   the norm is the word itself; for `form!=0` it's
   `prefix₀ + stem + suffix₀`; `LemmaRef.Normal` and `LemmaRef.Tag`
   are taken from form 0 (`lemma.go:15-36`). So the draft's
   "AddLemma(lemma, base grammemes)" maps not to a separate entity but
   to a mandatory form 0 whose form tag `== lemma`.

3. **Tags are opaque strings.** TagSet interns arbitrary names
   (`internal/tagset.go:23-37`, a limit of 65,536). A UniMorph bundle
   can be stored as-is (e.g. `N;ACC;SG`) — the draft's CLI expectation
   (`lookup кота -> N;ACC;SG`) is achieved with no mapping at all.

4. **Language and CharPolicy are hardcoded in the importer.**
   OpenCorpora hardcodes `language="ru"` and
   `internal.RussianCharPolicy()` (`import.go:276-284`). The UniMorph
   importer has no fixed language — it needs an explicit parameter
   (see Q1).

5. **Sharding** (`shard.go`, a limit of 65,536 suffixes per shard) is
   reusable; for `rus` (353K forms) a single shard is almost certainly
   enough, but the mechanism is worth keeping for other languages.

6. **There's no UniMorph data/loader in the repo.** OpenCorpora and
   pymorphy2 have loaders (`pkg/opencorpora`, `pkg/pymorphy`) and data
   in `.data/`; UniMorph has neither (see Q4).

## Gaps between the draft and the actual architecture

1. The "Pipeline. Lemma/Forms" item (`AddLemma`/`AddForm`) is stale —
   rewrite it in terms of "lemma = form 0, forms = the other rows," with
   no Builder.
2. Public API: the draft has `CompileFromUniMorph(r io.Reader)` with no
   options, and `ImportFromTSV(r, opts Options)` with options; a
   language code isn't passed anywhere. The signatures need to be
   reconciled (see Q1).
3. `docs/unimorph.md` §5.1/5.2: references to `pkg/dictionary` and the
   old `pkg/unimorph` are stale; the actual location is
   `pkg/morphology/importers/unimorph/`.
4. The draft's integration tests need a real `rus` file, which the
   project has no source for (Q4).

## Proposed design decisions for the importer

1. **Row format (streaming).** `bufio.Scanner` with an enlarged buffer
   (~1 MB — for compound/hyphenated tokens, as in the draft), split on
   `\n` and trim `\r`. A row is exactly 3 tab-separated fields. Row
   order by lemma isn't guaranteed by contract — accumulation is
   needed: `map[lemma] -> lemmaEntry` (28K entries for `rus` — cheap),
   form order within a lemma = row order.

2. **Lemma = form 0 (the key decision).**
   - If there's a row with `form == lemma`, its bundle becomes form
     0's tag (if there are several such rows with different bundles —
     all are kept: the first as form 0, the rest as ordinary homonym
     forms — a native lemma-form syncretism, handled correctly by
     existing mechanisms).
   - If there's no row with `form == lemma`, synthesize form 0 with the
     lemma's text and an empty tag (`""` is allowed in TagSet). This
     is necessary because otherwise `Lemma()` for the lemma's forms
     would return an arbitrary form instead of the lemma
     (`parse.go:211-215`), and `Parse(lemma)` wouldn't work.
   - **The lemma's text is always included in `lcp()`'s input**
     (technically, in the set of forms used to compute the stem).
     Otherwise, for suppletive groups ("я" vs. "меня", "человек" vs.
     "люди") the stem isn't necessarily a prefix of the lemma, and
     norm reconstruction would break. With LCP over forms+lemma, the
     stem is always a prefix of the lemma -> form 0's suffix is always
     valid.
   - An empty lemma -> lemma := the wordform (agreed in the draft; in
     this model this is trivial: form 0 = the wordform, with its own
     tag).

3. **Opaque tags.** Tag = the bundle verbatim (no reordering,
   `strings.Split(bundle, ";")` is only needed if we ever want
   individual grammemes; for TagSet the string `N;ACC;SG` as-is is
   enough). Feature order is preserved, LGSPEC features aren't
   interpreted. The `Options.Mapping` (FT12/bundle projection) — see
   Q2.

4. **Dedup and syncretism.** There are no exact duplicate rows in
   `rus`; a repeated `(form, bundle)` within a lemma collapses via
   paradigm dedup (`paradigmKeyHash`) and `dedupEntries`; the same text
   with different bundles produces different `(key, val)` entries in
   the DAWG (not a duplicate) — readings are preserved (a regression
   precedent — `TestImportFromXMLHomonymFormsBothSurvive`).

5. **Multi-word/hyphenated tokens.** The library is byte-oriented, and
   LCP is rune-safe (`lcp` in
   `importers/opencorpora/import.go:402-428`); there are no further
   restrictions.

6. **Sharding and Progress.** Reuse the `FillOnDemand` model (a
   copy/extraction of the shared part — see Q3). A progress callback
   is optional (building 473K rows takes seconds; the signature can be
   kept for parity).

## Decisions made (Q&A, 2026-09-16)

The user's answers to the open questions, recorded as decisions.
Where an answer overrode the recommendation, it's marked "↷ overrides
the recommendation."

**Q1. Language code and CharPolicy — decided (↷ partially overrides
the recommendation).** At the first stage, **only Russian** is ever
loaded; the dictionary's download URLs are **static**; there's a
separate constructor for the language. The code is built "with room to
grow" for loading and importing other languages (a very distant
prospect). Consequences:
- `Options.Language` is a mandatory field on the importer; `ru` is
  supported going forward (at the first stage — only `ru`), other
  languages error out.
- CharPolicy is chosen by language: `ru` -> `internal.RussianCharPolicy()`,
  otherwise `nil`.
- CLI: a `-lang` flag (default `ru`) + a language constructor in the
  loader.
- The loader takes the language in its constructor
  (`NewLoader(language, dataPath)`), the URL is built from the
  language, but only `ru` is actually allowed.

**Q2. UniMorph -> OpenCorpora tag mapping — deferred (the
recommendation was accepted).** `Options.Mapping` is **not** in Stage
16's scope. Tags are opaque bundle strings, verbatim (`N;ACC;SG`).
Mapping is a separate increment (+0.5-1.5 days), if it's ever needed.

**Q3. A shared TSV engine — a minimum for Stage 16 (the recommendation
was accepted).** A separate, minimal UniMorph importer for now;
extracting a shared, parameterizable engine happens when Stage 19 is
implemented. Copying shared pieces (`lcp`, `paradigmKeyHash`,
`FillOnDemand`) is acceptable, marked with a comment as an extraction
candidate.

**Q4. Data source for integration tests — add a loader (↷ overrides
the recommendation).** A UniMorph loader modeled on `pkg/pymorphy`
(not a manual download): `pkg/unimorph` with `DomainName="unimorph"`,
a static URL
`https://raw.githubusercontent.com/unimorph/<lang>/master/<lang>`,
downloaded into `.data/unimorph/<lang>` (+0.5 days added to the Stage
5 estimate).

**Q5. Malformed rows — log an error, but don't abort (the
recommendation was accepted).** A row ≠ 3 tab-separated fields -> an
error with the line number is **printed** (logged), and the import
**continues** with the next line (it doesn't abort). An empty bundle
is a valid case (an empty `""` tag in TagSet).

**Q6. "The lemma is always form 0" — confirmed (the recommendation was
accepted).** Synthesize form 0 with the lemma's text and an empty tag
`""` if there's no row with `form == lemma`; the lemma's text is
always included in `lcp()`'s input. For `rus` this adds up to ~28K keys
to the DAWG.

**Q7. Progress/BuildInfo — the recommendation was accepted.**
`BuildInfo.Source="unimorph"`; `SourceVersion` comes from an option
(the static URL / download date); the progress callback is optional
(nil is enough).

## Updated scope estimate (post-Q&A)

| # | Task | Changes from the decisions | Estimate |
|---|---|---|---|
| 1 | Importer `pkg/morphology/importers/unimorph/import.go` | form 0 + synthesis (Q6), the lemma is always in LCP, malformed rows: log and continue (Q5) | 0.5-1 day |
| 2 | Public API | `CompileFromUniMorph(r, opts)`, `Options{Language, CharPolicy}`, no `Mapping` planned in (Q2) | 0.25 day |
| 3 | CLI | `import unimorph <file> -lang ru` (Q1) | 0.25 day |
| 4 | Unit tests | + a test for a synthetic form 0, suppletion, malformed rows -> continue | 0.5-1 day |
| 5 | Integration tests + data | a UniMorph loader (Q4); the real `rus`, cross-checked against OpenCorpora | 0.4-1 day (incl. the loader, +0.5 day) |
| 6 | Loader `pkg/unimorph` | ru only, a static URL, a language constructor (Q1, Q4) | 0.25-0.5 day |
| 7 | Documentation | + record the decisions | 0.25 day |

**Total (excluding mapping): ≈ 2.5-3.5 days** (the loader is added to
the estimate).

## Short implementation plan

1. **Update `docs/implementation/stage-16-import-unimorph.md`** to
   match the actual model: drop the Builder/`AddLemma`/`AddForm`,
   record "lemma = form 0 (including synthetic)",
   `Options.Language`/`CharPolicy`, the CLI `-lang` flag, "malformed
   rows -> log and continue," "bundle verbatim," with no
   `Options.Mapping`.
2. **Implement** tasks 1-3 from the table (the importer package ->
   the public API -> the CLI).
3. **The loader** `pkg/unimorph` (task 6).
4. **Unit tests + a roundtrip** (task 4).
5. **Data and integration tests** (task 5): the loader pulls
   `unimorph/rus`, verify `lookup кота -> N;ACC;SG`, `Lemma`, the ratio
   of paradigms ≤ lemmas, and a cross-check of shared words against the
   OpenCorpora dictionary.
6. **Verification:** `go build ./...`, `go vet ./...`,
   `go test ./pkg/morphology/... -race`, `make test-integration` (with
   UniMorph data).
7. **Documentation** (task 7), mark the stage done in `todo.md` /
   `implementation.md`.

## Limits of applicability

The scope numbers are an expert estimate by analogy with Stage 15
(OpenCorpora), not a measurement. Deliberately out of scope: tag
mapping (deferred, Q2), lexicographic tag sorting, `TagSet` load (28K
lemmas × a small number of unique bundles — far from the 65,536
limit). The sharding estimate for `rus` (1 shard) isn't measured by
volume; the mechanism is reused as a safety net.

## Sources

No third-party sources were used — references are to files and lines
in this repository (see the "Current code snapshot" section) and to
the Stage 16 draft / the format analysis (`unimorph.md`).
