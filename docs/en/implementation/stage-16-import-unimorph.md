# Stage 16. UniMorph import

> **Status:** the draft was updated with Q&A decisions (2026-09-16).
> Implementation hasn't started yet. See also
> `docs/en/research/0006-unimorph-import-plan.md` — the full analysis,
> effort estimate, and detailed rationale.

## Stage contents

Importing a UniMorph dictionary from a TSV file
(`lemma<TAB>wordform<TAB>bundle`) as an additional data source. Similar
to Stage 15 (OpenCorpora import), but simpler: no XML, lines are parsed
in a stream. The full format analysis is in
[docs/unimorph.md](../unimorph.md).

Reference data: `unimorph/rus` (473,482 lines, 28,069 lemmas, 353,004
wordforms, CC-BY-SA 3.0 license).

### Key decisions (fixed in the 2026-09-16 Q&A)

1. **Lemma = form 0 of the paradigm (always).** If a line with
   `form == lemma` exists, its bundle becomes form 0's tag. If not,
   synthesize form 0: text = lemma, tag = `""`. The lemma's text is
   always included in `lcp()`'s input, which ensures the base form is
   reconstructed correctly for suppletive groups ("я"/"меня",
   "человек"/"люди").
2. **Tags are opaque strings.** A bundle (e.g. `N;ACC;SG`) is stored in
   the TagSet verbatim; reordering and mapping onto OpenCorpora tags
   **are not implemented** in Stage 16 (deferred, Q2).
3. **Language and CharPolicy.** In the first pass — only `ru`
   (`Options.Language`); other languages produce an error. CharPolicy
   is picked by language: `ru` -> `internal.RussianCharPolicy()`,
   otherwise nil. The code is structured with room for future languages (Q1).
4. **Malformed lines (!= 3 fields).** The error is reported (logged),
   but the import continues — the current line is skipped (Q5). An
   empty bundle (the third column) is allowed (an empty `""` tag).
5. **The data loader** is modeled on `pkg/pymorphy`, in a `pkg/unimorph`
   package (Q4), not a manual download.

### The importer (`pkg/morphology/importers/unimorph/`)

```go
func ImportFromTSV(
    r io.Reader,
    tagSet *internal.TagSet,   // nil -> NewTagSet("unimorph")
    opts Options,
) (*internal.Dictionary, error)
```

`Options`:
```go
type Options struct {
    Language    string                     // "ru" by default
    CharPolicy *internal.CharPolicy        // nil -> inferred from Language
    OnMalformed func(lineNumber int, text string) // nil -> silently skip
    Progress    Progress                   // nil -> no progress reporting
}
```

#### Pipeline

1. **Reading.** A `bufio.Scanner` with a 1 MB buffer (for compound/
   hyphenated tokens); a line is exactly 3 `\t`-separated fields, `\r`
   trimmed from the 3rd field. A line with an empty lemma -> lemma :=
   wordform. Empty lines are skipped (no error).
2. **Accumulation.** Data is grouped into `map[lemma]*lemmaEntry` (28K
   entries for `rus` — cheap; line order per lemma isn't guaranteed,
   the map's arbitrary key order is used).
3. **Form 0.** Once all lines are loaded, for each lemma:
   - Find the line with `text == lemma` -> place it at forms position 0
     (first occurrence). Other lines with the same text are ordinary
     forms (syncretism of the lemma's form).
   - If no line has `text == lemma`, synthesize form 0: `{text: lemma,
     gramm: ""}`.
4. **Stem.** Compute the (rune-safe) LCP over the list of forms
   (including form 0). Each form's suffix is `word[len(stem):]`.
5. **Tags.** Each form's bundle -> `tagSet.Add(bundle)` (an opaque
   string; no reordering is performed).
6. **Paradigm dedup.** Suffixes, tags, and prefixes (all `""` for
   UniMorph) are identified by integer IDs. Paradigms are deduplicated
   (`paradigmKeyHash`); sharding is `FillOnDemand` (a limit of 65,536
   suffixes per shard).
7. **DAWG.** The key is `prefix+stem+suffix` (= the wordform for
   UniMorph), the value is `paraID<<16|formIdx`. Built via
   `BuildDAWGWithValuesProgress`.
8. **Assembly.** `internal.NewDictionary(language, tagSet, suffixes,
   prefixes, paradigms, dawg, charPolicy)` -> `SaveTo`.

#### Handling UniMorph data

| Case | Handling |
|---|---|
| Syncretism (one text with different bundles) | Native: several wordform readings |
| Wordform == lemma | Form 0 gets its bundle; others are ordinary forms |
| Wordform == lemma (several lines) | The first -> form 0; the rest are ordinary forms (lemma-form homonyms) |
| Empty lemma | Lemma := wordform (form 0 = the wordform, tagged with its bundle) |
| Multi-word/hyphenated tokens | The library works with arbitrary bytes |
| `LGSPEC*`/unknown features | An opaque grammeme, not an error |
| Suppletion ("я"/"меня", "человек"/"люди") | Handled correctly: the stem's LCP runs over forms+lemma (form 0 included), the base form = lemma |
| No `form == lemma` line | Form 0 is synthesized: text=lemma, tag=`""`; adds ~28K keys to the DAWG |
| A repeated `(form, bundle)` within a lemma | Deduplicated by paradigm (pairs are interned) |
| Malformed lines (!= 3 fields) | OnMalformed + the line is skipped; import continues |
| An empty bundle | An empty `""` tag in the TagSet — allowed |
| 1 shard for `rus` (353K forms) | The sharding mechanism is used as a safeguard; it kicks in for larger languages |

### Public API (`pkg/morphology/`)

```go
type UniMorphOptions = unimorph.Options

func CompileFromUniMorph(r io.Reader, opts UniMorphOptions) (*Dictionary, error)
func CompileFromUniMorphFile(path string, opts UniMorphOptions) (*Dictionary, error)
```

Wrappers over `unimorph.CompileFromTSV`; Language from opts is the
dictionary's language (only `"ru"` for now).

### CLI

```bash
gomorphy download unimorph             # download rus into .data/unimorph/
gomorphy import unimorph <file> -lang ru -o ru-unimorph.dat
gomorphy -dict ru-unimorph.dat lookup кота  # -> N;ACC;SG
```

The `-lang` flag (default `ru`). The `download/unpack/build/unpack` CLI
commands for the `unimorph` type mirror pymorphy/opencorpora.

`CompileFromUniMorphFile` reads a TSV file and builds a Dictionary. A
wrapper over `unimorph.ImportFromTSV`.

### CLI

```bash
gomorphy import unimorph <rus.tsv> -o ru-unimorph.dat
gomorphy -dict ru-unimorph.dat lookup кота
```

## Verification (tests)

- Unit test: a mini TSV (5-10 lemmas) -> correct lemmas and paradigms.
- Unit test: a synthetic form 0 (the lemma never occurs as a form) ->
  `Parse(lemma)`, `Lemma(form)` returns the lemma, tag `""`.
- Unit test: roundtrip ImportFromTSV -> SaveTo -> Open -> identical Parse.
- Unit test: syncretism — one wordform with several bundles -> all readings.
- Unit test: suppletion ("я"/"меня", "человек"/"люди") -> base form = the lemma.
- Unit test: empty lemma -> lemma == the wordform.
- Unit test: malformed lines (!= 3 fields) -> OnMalformed is called, import continues.
- Unit test: `(form, bundle)` and paradigm dedup.
- Integration test: the entire `rus` -> `Lookup`/`Lemmas` on sample
  words (`кота` -> `N;ACC;SG`, lemma `кот`).
- Integration test: paradigms < lemmas (for `rus`: 28,068 < 28,069).
- Integration test: cross-checking words shared with the OpenCorpora dictionary.
- `go test ./pkg/morphology/... -race` — green.

## Manual verification

- `gomorphy download unimorph` -> a `rus` file in `.data/unimorph/`.
- `gomorphy import unimorph <file> -lang ru -o ru-unimorph.dat` -> the file is created.
- `gomorphy -dict ru-unimorph.dat lookup кота` -> `N;ACC;SG`.
- A different language (e.g. `eng`): `gomorphy -dict en.dat lookup cats` —
  in Stage 16, `-lang` only accepts `ru` (anything else is an error);
  groundwork for other languages is left in the signatures (Q1).
- Cross-checking words shared with the OpenCorpora dictionary.
