# Usage scenarios

> Russian version: [docs/ru/scenarios.md](../ru/scenarios.md).

What gomorphy is for, task by task: what each scenario solves, which calls
do it, and a runnable example. [library.md](library.md) and
[cli.md](cli.md) are the reference for every call named here; this page is
the "why" and "which one".

Each scenario states the version it is available since and, when behaviour
changed later, a short **History** — a per-feature excerpt of
[CHANGELOG.md](../../CHANGELOG.md), which stays the source of truth.

Examples: `go run ./examples/<name>` from the repository root
([examples/](../../examples/README.md)); `ExampleXxx` functions are in
`pkg/morphology/example_test.go` and render on pkg.go.dev.

| # | Scenario | Since | Example |
|---|---|---|---|
| 1 | [Morphological analysis of a word](#1-morphological-analysis-of-a-word) | 1.0.0 | [opencorpora](../../examples/opencorpora/main.go) |
| 2 | [Normalize words to lemmas for search and indexing](#2-normalize-words-to-lemmas-for-search-and-indexing) | 1.0.0 | `ExampleDictionary_Lemma` |
| 3 | [Tell a dictionary word from a guess](#3-tell-a-dictionary-word-from-a-guess) | 1.2.0 | [ner](../../examples/ner/main.go), `ExampleDictionary_IsKnown` |
| 4 | [Dictionary-based named-entity lookup](#4-dictionary-based-named-entity-lookup) | 1.1.0 | [ner](../../examples/ner/main.go) |
| 5 | [A thematic dictionary from a TSV file](#5-a-thematic-dictionary-from-a-tsv-file) | 1.1.0 | [importtsv](../../examples/importtsv/main.go) |
| 6 | [Extend or override a base dictionary](#6-extend-or-override-a-base-dictionary) | 1.1.0 | [merge](../../examples/merge/main.go) |
| 7 | [Query several dictionaries at once](#7-query-several-dictionaries-at-once) | 1.0.0 | [multidict](../../examples/multidict/main.go) |
| 8 | [Typos and suggestions](#8-typos-and-suggestions) | 1.0.0 | [typos](../../examples/typos/main.go) |
| 9 | [е/ё and other character substitutions](#9-её-and-other-character-substitutions) | 1.2.0 | `ExampleNewCharPolicy` |
| 10 | [A dictionary for a language other than Russian](#10-a-dictionary-for-a-language-other-than-russian) | 1.1.0 | `ExampleNoCharPolicy` |
| 11 | [Ship a dictionary inside the binary; Windows](#11-ship-a-dictionary-inside-the-binary-windows) | 1.2.0 | [embed](../../examples/embed/main.go) |
| 12 | [Cache results by dictionary content](#12-cache-results-by-dictionary-content) | 1.2.0 | [contenthash](../../examples/contenthash/main.go) |
| 13 | [Compare tags across dictionaries](#13-compare-tags-across-dictionaries) | 1.0.0 | [tagmap](../../examples/tagmap/main.go) |
| 14 | [Inspect a dictionary by hand](#14-inspect-a-dictionary-by-hand) | 1.0.0 | CLI `lookup`, `cli` |
| 15 | [Download source dictionaries from Go code](#15-download-source-dictionaries-from-go-code) | 1.0.0 | [pymorphy](../../examples/pymorphy/main.go) |
| 16 | [Word forms: decline or conjugate a word](#16-word-forms-decline-or-conjugate-a-word) | 1.3.0 | [inflect](../../examples/inflect/main.go) |
| 17 | [Work with grammemes without parsing tag strings](#17-work-with-grammemes-without-parsing-tag-strings) | 1.3.0 | `ExampleHasGrammeme` |

## 1. Morphological analysis of a word

**Task.** Get every grammatical reading of a wordform: part of speech,
case, number, gender, the lemma.

**How.** `Dictionary.Parse(word)` returns `[]Reading` (`Word`, `Normal`,
`Tag`, `Prob`, …). The input is lower-cased. Readings of a
pymorphy2-sourced dictionary are sorted by probability (`Prob`); other
sources have `Prob == 0`. From the shell: `gomorphy lookup -d <dict.dat>
<word>`.

```go
d, err := morphology.Open(".data/pymorphy/pymorphy.dat")
// ...
for _, r := range d.Parse("стали") {
    fmt.Println(r.Normal, r.Tag, r.Prob)
}
```

In a pymorphy2 dictionary, and in one built with `Builder`/`ImportTSV`
(or merged onto such a base), an unknown word still gets readings, guessed
from its ending — see [scenario 3](#3-tell-a-dictionary-word-from-a-guess).
Since 1.3.0, OpenCorpora and UniMorph imports predict too, by default
(pruned like pymorphy2's own dictionary compiler); use `IsKnown` to tell a
dictionary reading from a guess, or opt out at build time with
`XMLOptions.NoPrediction`/`UniMorphOptions.NoPrediction` or
`gomorphy build opencorpora|unimorph --no-prediction`, in which case
`Parse` still returns `nil` for an unknown word.

Parsing many words in a loop (e.g. every token of a document)? Reuse a
result buffer with `Dictionary.ParseAppend(dst, word)` — it appends to
`dst` instead of allocating a fresh slice per call.

```go
var buf []morphology.Reading
for _, word := range tokens {
    buf = d.ParseAppend(buf[:0], word)
    // ... use buf ...
}
```

**Available since:** 1.0.0.

**History:**
- 1.2.0 — `Reading.Predicted` marks guessed readings; CLI `lookup` prints
  `(predicted)` next to them.
- 1.3.0 — OpenCorpora and UniMorph imports build prediction by default too
  (previously pymorphy2/`Builder`/`ImportTSV` only); `--no-prediction`
  opts out.
- 1.3.0 — `Dictionary.ParseAppend` reuses a caller-supplied slice.

## 2. Normalize words to lemmas for search and indexing

**Task.** Map «кота», «котом», «коты» to one key «кот» — for a search
index, word counts, or matching a query against text.

**How.** `Dictionary.Lemma(word)` returns one `LemmaRef` per homonym
(`Normal`, the lemma's own `Tag`). Several results mean the word is
ambiguous («стали» → «сталь» and «стать»); either index all of them or
pick by `Tag`. CLI: `gomorphy lemmas`.

```go
for _, l := range d.Lemma("кота") {
    fmt.Println(l.Normal) // кот
}
```

**Available since:** 1.0.0.

**History:**
- 1.2.0 — `LemmaRef.Predicted` is true when every reading behind the lemma
  was guessed; skip such lemmas if the index must hold dictionary words
  only.
- 1.3.0 — OpenCorpora and UniMorph imports build prediction by default too,
  so `Lemma` can return a `Predicted` lemma for these sources as well;
  `--no-prediction` opts out.

## 3. Tell a dictionary word from a guess

**Task.** Know whether a word really is in the dictionary. `Parse`
guesses readings for unknown words from their endings, and a guessed
reading looks like any other — for a small dictionary almost every word
"parses". (This concerns every dictionary source with prediction:
pymorphy2, `Builder`/`ImportTSV`/`Merge`, and, since 1.3.0, OpenCorpora
and UniMorph imports.)

**How.** `Dictionary.IsKnown(word)` — exact lookup only, never predicts
(lower-cased, `CharPolicy` applied, like `Parse`). Or check
`Reading.Predicted` / `LemmaRef.Predicted` on results you already have.
`MultiDictionary.IsKnown` is true if any member knows the word.

```go
d.IsKnown("кота")               // true
d.IsKnown("бота")               // false
d.Parse("бота")[0].Predicted    // true — a guess
```

**Example:** [ner](../../examples/ner/main.go), `ExampleDictionary_IsKnown`.

**Available since:** 1.2.0. Before 1.2.0 a guess could not be told from a
dictionary reading.

**History:**
- 1.3.0 — matters for OpenCorpora and UniMorph too, since their imports
  now predict by default (see [scenario 1](#1-morphological-analysis-of-a-word)).

## 4. Dictionary-based named-entity lookup

**Task.** Find known names (people, places, organisations, product
names) in text in any grammatical form: «в Москве», «памятник Пушкина».

**How.** Build a names dictionary with `Builder` (or `ImportTSV` —
[scenario 5](#5-a-thematic-dictionary-from-a-tsv-file)): register every
form with its lemma and an entity-type tag of your own (`GEO`, `PERSON`,
…; tags are opaque strings). Then, per token: `IsKnown` says whether it is
a name, `Parse` gives the lemma and tag. Words are lower-cased on build,
so «Москве» in text finds the stored «москве». To look up names and
common words together, put both dictionaries into a `MultiDictionary`
([scenario 7](#7-query-several-dictionaries-at-once)) and use
`Reading.Dict` to see which one answered.

```go
b := morphology.NewBuilder(morphology.BuilderOptions{Language: "ru", Source: "names"})
b.AddForm("Москве", "Москва", "GEO,loct")
names, err := b.Build()
// ...
if names.IsKnown(token) {
    r := names.Parse(token)[0] // r.Normal == "москва", r.Tag == "GEO,loct"
}
```

**Example:** [ner](../../examples/ner/main.go).

**Available since:** 1.1.0 (`Builder`); reliable since 1.2.0.

**History:**
- 1.2.0 — `IsKnown` and `Predicted` added: before, a tiny names dictionary
  "recognised" almost any word via ending prediction.
- 1.2.0 — `Builder`/`ImportTSV` lower-case words and lemmas. A name added
  as «Москва» by 1.1.0 was reachable only as a guess. **Rebuild names
  dictionaries built by 1.1.0 from capitalised input.**

## 5. A thematic dictionary from a TSV file

**Task.** Build a dictionary of your own vocabulary — domain terms, names,
slang — with no internet access and no base dictionary.

**How.** Prepare `lemma<TAB>wordform[<TAB>tags]` rows, then either
`gomorphy import tsv words.tsv -o words.dat` or
`morphology.ImportTSV(r, BuilderOptions{Language: "ru"})`. Blank lines and
`#` comments are skipped; an empty lemma makes the wordform its own lemma;
prediction is built from your own words. The result round-trips through
`SaveTo`/`Open` like any dictionary.

```
# lemma	wordform	tags
фрегат	фрегат	NOUN,nomn
фрегат	фрегата	NOUN,gent
```

**Example:** [importtsv](../../examples/importtsv/main.go),
`ExampleImportTSV`; CLI: [cli.md — `import`](cli.md#import--build-a-dat-from-a-wordform-tsv).

**Available since:** 1.1.0.

**History:**
- 1.1.0 — a row with an empty wordform column is an error (was silently
  accepted).
- 1.2.0 — the wordform and lemma columns are lower-cased; rebuild TSV
  dictionaries built by 1.1.0 from mixed-case input.
- 1.2.0 — the default е→ё substitution applies only to `Language` "ru"
  (see [scenario 9](#9-её-and-other-character-substitutions)).
- 1.2.1 — a stream with no entries is an error (`ErrNoEntries`), as for
  `Builder`; it used to give an empty dictionary.
- 1.3.0 — entries are grouped into lexemes by lemma **and** part-of-speech
  class: a homonym like «знать» NOUN (nobility) and «знать» INFN (to know)
  becomes two lemmas instead of one mixed paradigm. A dictionary rebuilt
  from homonymous input differs from 1.2.0 output; existing files are
  unaffected.

## 6. Extend or override a base dictionary

**Task.** Add missing words to a full dictionary (pymorphy2, OpenCorpora),
or replace wrong readings, and keep one `.dat` file.

**How.** `morphology.Merge(base, overlays, MergeAdd)` adds overlay words
the base lacks; `MergeReplace` makes an overlay word's readings replace the
existing ones (the last overlay wins). CLI: `gomorphy merge --mode
add|replace -o merged.dat base.dat overlay.dat`. The base keeps its
paradigms, tag-set name (so `tagmap` still works), probabilities and
prediction. For the alternative that keeps files separate, see
[scenario 7](#7-query-several-dictionaries-at-once).

**Example:** [merge](../../examples/merge/main.go), `ExampleMerge`,
`ExampleMergeWithOptions`.

**Available since:** 1.1.0.

**History:**
- 1.1.0 — the merge is structural (Stage 19.1): earlier builds of the
  feature rebuilt the output from scratch and lost prediction,
  probabilities and the tag-set name.
- 1.3.0 — `MergeOptions.RebuildPrediction`/`merge --rebuild-prediction`
  works for a merged dictionary with any number of shards (was
  single-shard only).

## 7. Query several dictionaries at once

**Task.** Use a base dictionary and your own additions together without
merging them into one file — e.g. to update them independently, or to know
which dictionary a reading came from.

**How.** `morphology.NewMultiDictionary(d1, d2, …)` over already-open
dictionaries. `Parse`/`Lemma`/`Fuzzy` concatenate the members' results in
registration order; `Reading.Dict` (and `LemmaRef.Dict`,
`FuzzyMatch.Dict`) is the member's index. CLI: repeat `-d`.

**Example:** [multidict](../../examples/multidict/main.go),
`ExampleNewMultiDictionary`.

**Available since:** 1.0.0.

**History:**
- 1.2.0 — `MultiDictionary.IsKnown`.
- 1.3.0 — `MultiDictionary.Forms`/`Inflect`, dispatching to the member
  dictionary a reading came from (`Reading.Dict`); see
  [scenario 16](#16-word-forms-decline-or-conjugate-a-word).

## 8. Typos and suggestions

**Task.** Find what the user meant: «кто» → «кот», spelling suggestions,
lenient search.

**How.** `Fuzzy(word, maxDist)` returns every dictionary word within
Levenshtein distance `maxDist` (in runes); `FuzzyTop(word, n)` returns the
`n` nearest words, widening the distance as needed. Both are sorted by
(distance, word). CLI: `gomorphy fuzzy`, `gomorphy top`.

```go
d.Fuzzy("кот", 1)     // кот 0, кит 1, код 1, …
d.Fuzzy("Елка", 0)    // ёлка 0
d.FuzzyTop("катк", 2) // the 2 nearest words
```

**Example:** [typos](../../examples/typos/main.go),
`ExampleDictionary_Fuzzy`, `ExampleDictionary_FuzzyTop`.

**Available since:** 1.0.0.

**History:**
- 1.2.0 — the query is lower-cased: a capital letter no longer costs an
  edit («Кот» did not match «кот» at distance 0).
- 1.2.0 — the dictionary's `CharPolicy` applies: for Russian, е in the
  query matches a stored ё at distance 0 (was 1). Thresholds tuned on the
  old metric may need adjusting.

## 9. е/ё and other character substitutions

**Task.** Find «ёлка» when the text says «елка» — Russian text usually
drops the dots — or define similar equivalences for another alphabet.

**How.** A dictionary's `CharPolicy` lists one-way substitutions: a query
rune `From` also matches a stored rune `To`. For Russian the default is
е→ё: «елка» finds «ёлка», but «ёлка» does not find «елка». The policy is
stored in the dictionary and used by `Parse`, `Lemma`, `IsKnown`, `Fuzzy`
and `FuzzyTop`. Set it when building: `BuilderOptions.CharPolicy` or
`UniMorphOptions.CharPolicy` — `RussianCharPolicy()`, `NoCharPolicy()`, or
`NewCharPolicy(Substitution{From: 'е', To: 'ё'}, …)` (at most 255
substitutions).

```go
opts := morphology.BuilderOptions{
    Language:   "ru",
    CharPolicy: morphology.NoCharPolicy(), // е and ё are different letters
}
```

**Example:** `ExampleNewCharPolicy`, `ExampleNoCharPolicy`.

**Available since:** 1.2.0 (public API; Russian dictionaries applied е→ё
before, without a way to change it).

**History:**
- 1.2.0 — `nil` `CharPolicy` means "by language": е→ё for "ru" and for an
  empty `Language`, nothing for other languages (before, every built
  dictionary got е→ё).
- 1.2.0 — more than 255 substitutions is an error at build time (it used
  to panic later, in `SaveTo`).

## 10. A dictionary for a language other than Russian

**Task.** Use gomorphy's lookup, lemmas and fuzzy search for another
language.

**How.** Build the dictionary yourself with `Builder` or `ImportTSV`
([scenario 5](#5-a-thematic-dictionary-from-a-tsv-file)) and set
`BuilderOptions.Language` (e.g. "uk"). The Russian е→ё substitution is not
applied; pass your own `CharPolicy` if the language needs one
([scenario 9](#9-её-and-other-character-substitutions)). The UniMorph
importer accepts only `Language: "ru"` today. `Merge` requires every input
to have the same language.

**Available since:** 1.1.0.

**History:**
- 1.2.0 — non-Russian dictionaries no longer get е→ё by default.
- 1.3.0 — `Builder`/`ImportTSV` (and the `*Dense` importers) fall back to a
  2-byte alphabet instead of failing when the wordforms use more than 254
  distinct characters — languages with large alphabets or mixed scripts
  are no longer capped at 254.

## 11. Ship a dictionary inside the binary; Windows

**Task.** Distribute one executable with the dictionary inside — no data
file next to it — or run on Windows, where `Open` (mmap) is not supported.

**How.** Embed a `.dat` with `//go:embed` and open it with
`morphology.OpenBytes(data)`: same format and checksum check as `Open`, no
mmap, no copy of `data` (keep it alive as long as the dictionary; a
package-level variable does). `Close` is a no-op.

```go
//go:embed names.dat
var namesDat []byte

d, err := morphology.OpenBytes(namesDat)
```

**Example:** [embed](../../examples/embed/main.go), `ExampleOpenBytes`.

**Available since:** 1.2.0.

## 12. Cache results by dictionary content

**Task.** Cache something computed from a dictionary (a derived index,
query results) and invalidate it only when the words actually change —
not on every rebuild or re-save.

**How.** `Dictionary.ContentHash()` is a digest of the content sections
that ignores the `info` section (build time, library version, source), so
it stays the same across `SaveTo`/`Open`/`OpenBytes` and rebuilds from the
same input. It does change with the encoding: the same words under a
different alphabet or `CharPolicy` hash differently. The first call is
computed and cached.

**Example:** [contenthash](../../examples/contenthash/main.go),
`ExampleDictionary_ContentHash`.

**Available since:** 1.2.0.

## 13. Compare tags across dictionaries

**Task.** Readings from different sources describe the same form in
different notations: OpenCorpora `NOUN,anim,masc,sing,gent`, UniMorph
`N;GEN;SG`. You need to compare them — e.g. to check that two dictionaries
agree on case and number.

**How.** `tagmap.Map(d.TagSetName(), reading.Tag)` normalizes a native tag
into a UniMorph feature bundle; compare bundles, or single dimensions
(`tagmap.DimCase`, `tagmap.DimNumber`, …) when one source marks features
the other lacks. `tagmap.Known(name)` tells whether a tag set can be
normalized at all (a `Builder` dictionary with your own tags cannot).

**Example:** [tagmap](../../examples/tagmap/main.go), `tagmap.ExampleMap`,
`ExampleMultiDictionary_DictTagSetName`.

**Available since:** 1.0.0.

**History:**
- 1.1.0 — `tagmap.Known`.

## 14. Inspect a dictionary by hand

**Task.** Look into a dictionary while building or debugging one: what a
word parses to, whether it is a real entry, which dictionary answered.

**How.** `gomorphy lookup`/`lemmas`/`fuzzy`/`top -d <dict.dat> <word>`,
or the interactive console `gomorphy cli -d <dict.dat>`. Repeat `-d` to
query several dictionaries; the `para#<dict>/…` column shows which one
answered. See [cli.md](cli.md).

**Available since:** 1.0.0.

**History:**
- 1.2.0 — `lookup` ends a guessed reading's line with an extra
  `(predicted)` column.

## 15. Download source dictionaries from Go code

**Task.** Fetch and unpack pymorphy2, OpenCorpora or UniMorph source data
from your own program, as `gomorphy download`/`unpack` do.

**How.** `pymorphy.NewLoader(dir)`, `opencorpora.NewLoader(dir)` or
`unimorph.NewLoader("ru", dir)`, then `loader.Sync(false)`, then compile
from `loader.UnpackedDirPath()` / `UnpackedFilePath()` — see
[library.md — Fetching source data](library.md#fetching-source-data).
`Sync(false)` checks for a newer release and refreshes the local copy;
`Sync(true)` stays offline and uses what is on disk. A failed download
keeps the previous data. The loaders need no logging setup: without
`logging.Init` they log nothing; assign the exported `Logger` field to use
your own logger.

**Example:** [pymorphy](../../examples/pymorphy/main.go) (reads an
already-downloaded directory).

**Available since:** 1.0.0.

**History:**
- 1.2.1 — the loaders no longer panic when the program never called
  `logging.Init` (only the `gomorphy` CLI did).
- 1.2.1 — `Sync` unpacks a new download over the old copy (the old data
  used to stay), files go under the given data path (not `./.data`),
  downloads are atomic and check the HTTP status, and `Sync(true)` makes
  no network requests.

## 16. Word forms: decline or conjugate a word

**Task.** Given a word, produce another grammatical form of the same
lexeme — plural, a different case, a different tense — e.g. to inflect
«кот» into «котов» (plural genitive) for a sentence template.

**How.** `Dictionary.Parse(word)` a reading, then `Dictionary.Forms(r)`
for every form of its lexeme (form 0 is the lemma) or
`Dictionary.Inflect(r, grammemes...)` for the forms that contain every
grammeme in `grammemes`, best match first (fewest grammemes differing
from `r`'s own tag). Both work the same on a `Predicted` reading: the
forms come from the predicted paradigm and are as much a guess as the
reading itself — check `Reading.Predicted` if that matters. On a
`MultiDictionary`, use the equivalent methods, which dispatch to the
member dictionary the reading came from (`Reading.Dict`).

```go
r := d.Parse("кот")[0]
for _, f := range d.Inflect(r, "plur", "gent") {
    fmt.Println(f.Word) // котов
}
```

**Example:** [inflect](../../examples/inflect/main.go),
`ExampleDictionary_Forms`, `ExampleDictionary_Inflect`.

**Available since:** 1.3.0.

## 17. Work with grammemes without parsing tag strings

**Task.** Check a single grammatical feature of a tag — is it genitive?
plural? what part of speech? — without splitting the native tag string by
hand and without hard-coding its separators.

**How.** `Grammemes(tag)` splits a native tag into its grammeme tokens;
separators are `,`, space and `;`, which covers every tag format gomorphy
imports (OpenCorpora/pymorphy2 `"NOUN,anim,masc,Surn sing,ablt"`, UniMorph
`"N;GEN;SG"`). `HasGrammeme(tag, g)` checks one grammeme without
allocating; `Reading.HasGrammeme(g)` is the same check on a reading's own
tag. `POS(tag)` returns the tag's first grammeme — the part of speech in
every format gomorphy imports. These read the native tag as-is; to
compare a grammeme across dictionary sources that use different tag
vocabularies, normalize with `tagmap` first
([scenario 13](#13-compare-tags-across-dictionaries)).

```go
r := d.Parse("кота")[0]
morphology.Grammemes(r.Tag)     // ["NOUN" "anim" "masc" "sing" "gent"]
r.HasGrammeme("gent")           // true
morphology.POS(r.Tag)           // "NOUN"
```

**Example:** `ExampleHasGrammeme`, [inflect](../../examples/inflect/main.go).

**Available since:** 1.3.0.
