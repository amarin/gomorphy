# Comparison with Other Go Morphological Analyzers

Three other Go libraries do Russian morphological analysis. This
document compares them against gomorphy (this project) on
architecture, dictionary sourcing, API surface, storage/size,
performance, license, and maintenance activity, so a reader choosing
between them (or wondering what gomorphy still lacks) doesn't have to
read four READMEs and four codebases to find out.

Repositories compared:

- [jus1d/gomorphy](https://github.com/jus1d/gomorphy)
- [AlexMaxy/gomorphy](https://github.com/AlexMaxy/gomorphy)
- [SteosOfficial/SteosMorphy](https://github.com/SteosOfficial/SteosMorphy)

**Methodology.** Based on each repository's README, source (public
function signatures), `go.mod`, and GitHub repository metadata, read
directly on 2026-09-19 — not on secondhand descriptions. Size and
metadata figures are snapshots as of that date and will drift; treat
them as orientation, not a live comparison. gomorphy's own figures
(`.dat` sizes) were measured the same day by rebuilding each source
fresh with `gomorphy build`.

## At a glance

| | **gomorphy** (this project) | [jus1d/gomorphy](https://github.com/jus1d/gomorphy) | [AlexMaxy/gomorphy](https://github.com/AlexMaxy/gomorphy) | [SteosOfficial/SteosMorphy](https://github.com/SteosOfficial/SteosMorphy) |
|---|---|---|---|---|
| Dictionary sources | pymorphy2, OpenCorpora `dict.xml`, UniMorph TSV — three, pluggable, built with the CLI/library from data you fetch yourself | OpenCorpora v0.92 via pymorphy3, one fixed snapshot, embedded at compile time | OpenCorpora (a pymorphy3 DAWG dump dated 22.05.2026), one fixed snapshot, copied alongside the binary | One custom, team-curated dictionary, embedded in the repo |
| Storage format | Own sectioned `GMOR` format; mmap-backed; dense 1-byte DAWG alphabet by default | pymorphy2/3's own `dawgdic` (`words.dawg` + `paradigms.array`), `go:embed`, loaded fully into memory | Same `dawgdic` format, plus prediction-suffix DAWGs and a probability `intdawg`; `go:embed` | Own flat node/edge trie format, real `mmap` (`edsrzf/mmap-go`), zero-copy reads |
| Dictionary size | ~10 MB (OpenCorpora), ~15.7 MB (pymorphy2, incl. prediction+probability), ~11 MB (UniMorph) | ~8.8 MB added to the binary | ~15.9 MB dictionary directory, shipped alongside the binary | ~444.6 MB embedded in the repository (10 chunked blobs, reassembled at build time) |
| Exact lookup | `Parse` — every reading, sorted by probability | `Tag` — a single best tag only, no full reading list | `Parse` — every reading, with probability and an analyzer trace (`MethodsStack`) | `Analyze` — every reading (`[]*Parsed`) |
| Lemma | `Lemma` | not exposed directly (`WordForms()[0]` is presumably the lemma, but this isn't documented as guaranteed) | `parse.Normalized()` | `Parsed.Lemma` field |
| All wordforms of a lemma | not a dedicated API | `WordForms` | `parse.Lexeme()` (handles suppletion) | `Analyze`'s 2nd return value (handles suppletion) |
| Fuzzy/typo search | `Fuzzy`, `FuzzyTop` — a Levenshtein automaton joined with the DAWG traversal | not implemented | not implemented | not implemented |
| Prediction for unknown words | yes — an ending-based prediction DAWG for pymorphy2 dictionaries and for dictionaries built with Builder/ImportTSV (not for OpenCorpora or UniMorph imports); readings carry a Predicted flag | not found in the public API | yes — the full pymorphy2 heuristic stack (by-analogy, by-hyphen, by-shape, abbreviations, unknown) | yes — its own suffix-based OOV predictor (`ParsePredicted`/`Predict`) |
| Inflection / form generation | not implemented | only phrase-level adjective-noun agreement (`PhraseFormsConcordant`), not a general per-word inflect call | `parse.Inflect`/`InflectVar`, `MakeAgreeWithNumber` | `Inflect`/`InflectList` |
| Multiple dictionaries at once | `MultiDictionary` | no — one embedded dictionary | no — one dictionary directory | no — one embedded dictionary |
| Universal tag mapping across sources | `pkg/morphology/tagmap` (native tags -> a UniMorph-schema feature bundle) | n/a — single source | n/a — single source | n/a — single source |
| Built-in batch/concurrent processing | not built in (calls are already sub-microsecond; callers loop themselves) | no | no | yes — `ParseList`/`InflectList`, a CPU-core-sized worker pool |
| CLI tool | yes — `gomorphy` (`lookup`/`lemmas`/`fuzzy`/`top`/`cli`/`download`/`unpack`/`build`/`update`) | no (library only) | no (library only) | no (library only; a CGo/Python binding is offered instead) |
| External Go dependencies | `cobra`, `readline`, `testify`, `xxh3`, `x/term`, `amarin/logging` (+ indirect) | zero | zero | one direct (`edsrzf/mmap-go`) + one indirect |
| Code license | MIT | MIT | MIT | Apache-2.0 |
| Dictionary data license | not bundled — you fetch your own dictionary, under that source's own license (OpenCorpora CC BY-SA 4.0, `pymorphy2-dicts-ru`'s own PyPI license, UniMorph CC BY-SA 3.0) | CC BY-SA 4.0 (OpenCorpora), stated explicitly, embedded in the binary | CC BY-SA 4.0 (OpenCorpora), stated explicitly, shipped alongside the binary | not stated in the repository |
| Repo activity (as of 2026-09-19) | — | created 2026-02-26, last push 2026-09-17, 0 stars | created 2026-08-08, last push 2026-08-27, 0 stars, 1 fork | created 2025-09-03, last push 2026-06-25, 24 stars, 3 forks, 2 open issues |
| Published benchmarks | none published; internal design targets only (e.g. exact `Parse` < 10 µs, see [implementation.md](implementation.md)) | none | none | README claims ~20 µs/word ("tens of thousands words/sec"); a `go test -bench=.` target exists but no published numbers |

## Architecture and storage

The four projects split into two real architectural families:

- **`dawgdic`-format ports** (jus1d/gomorphy, AlexMaxy/gomorphy): both
  read pymorphy2/pymorphy3's own binary format (`words.dawg` +
  `paradigms.array`, plus, for AlexMaxy, prediction-suffix DAWGs and a
  probability `intdawg`) via `go:embed`, so the dictionary lives fully
  in the binary and fully in Go's heap at runtime — no mmap, no
  streaming. AlexMaxy is explicitly a fork of jus1d's DAWG-reading code
  extended into a much fuller pymorphy2 API port.
- **Custom mmap-backed formats** (gomorphy, SteosMorphy): both designed
  their own on-disk format and read it via real `mmap`
  (`internal/mmapx` here, `edsrzf/mmap-go` there) for zero-copy loads.
  gomorphy additionally recompiles every source under a shared dense
  1-byte DAWG alphabet by default (see
  [pymorphy2-dense-alphabet.md](implementation/pymorphy2-dense-alphabet.md)),
  which is most of why its `.dat` files land smaller than the
  `dawgdic` ports' embedded data despite covering the same underlying
  OpenCorpora word list.

SteosMorphy's ~444.6 MB embedded dictionary is an outlier by design,
not an inefficiency artifact visible from the outside — its README
claims 120,000+ lemmas and 5,300,000+ wordforms with 198 tags, a much
larger and more finely tagged vocabulary than the OpenCorpora-derived
data the other three projects (including gomorphy) share. Its size and
gomorphy's aren't measuring the same underlying dictionary, so they
aren't directly comparable the way jus1d/AlexMaxy/gomorphy's
OpenCorpora-derived numbers are.

## Dictionary sourcing: fixed snapshot vs. build-your-own

jus1d/gomorphy, AlexMaxy/gomorphy, and SteosMorphy each ship **one
fixed, pre-built dictionary** embedded in (or alongside) the binary —
simple to use, but the snapshot ages with the repository and there's
no supported way to point the library at a different or newer source.
gomorphy instead ships **no dictionary data at all**: `gomorphy
download/unpack/build <type>` fetches and compiles pymorphy2,
OpenCorpora, or UniMorph data yourself, and the same is available from
the Go API (`CompileFromXMLFile`, `OpenPyMorphy`,
`CompileFromUniMorphFile`). This is a real trade-off, not a strict
improvement: gomorphy needs a build/download step (network access, or
a pre-fetched file) before first use, where the other three work out
of the box with `go get`.

## API coverage

Exact lookup, lemma resolution, and OOV prediction are table stakes —
every project here has some form of all three (jus1d's `Tag` is the
partial exception: it returns only the single best tag, not gomorphy's
full sorted reading list). Past that, each project has chosen a
different differentiator:

- **gomorphy**: fuzzy/typo search (`Fuzzy`/`FuzzyTop`, a genuine
  Levenshtein-automaton-over-DAWG implementation) and multiple
  dictionaries open at once (`MultiDictionary`) — neither exists in
  any of the three alternatives.
- **AlexMaxy/gomorphy**: the most complete pymorphy2 Python-API parity
  of the four — `Inflect`/`InflectVar`, `MakeAgreeWithNumber`,
  `Lexeme`, `Normalized`, an analyzer method-stack trace, Cyrillic tag
  conversion (`Lat2Cyr`). If what you need is "reproduce pymorphy2's
  Python behavior in Go," this is the closest match.
- **SteosMorphy**: built-in concurrent batch processing
  (`ParseList`/`InflectList`, worker-pool-parallel across
  `runtime.NumCPU()`), plus a Python binding via CGo for the same
  compiled dictionary.
- **jus1d/gomorphy**: the smallest surface of the four (three exported
  functions total), plus one feature none of the others has —
  phrase-level adjective-noun agreement (`PhraseFormsConcordant`).

None of the three alternatives implement **inflection/generation** the
way gomorphy doesn't either — gomorphy has no "give me this word in
the genitive plural" API at all, while AlexMaxy and SteosMorphy both
do. None of the three implement gomorphy's **tag mapping across
dictionary sources** (`pkg/morphology/tagmap`) or **fuzzy search** —
but that's expected, since none of them support more than one
dictionary source in the first place, so there's nothing to map
between.

## Performance

None of the four projects publish a reproducible, apples-to-apples
benchmark against the others — every number here is either an internal
design target or an unverified README claim, not a controlled
comparison:

- gomorphy's own targets (not third-party-verified) are documented in
  [implementation.md](implementation.md)'s metrics table: exact
  `Parse` < 10 µs, prediction < 50 µs, DAWG build ~24 s for the full
  3.06M-lemma OpenCorpora set.
- SteosMorphy's README claims ~20 µs/word ("tens of thousands words
  per second") and ships a `go test -bench=.` target, but no benchmark
  output is published in the repository.
- jus1d/gomorphy and AlexMaxy/gomorphy publish no performance figures
  at all.

Take any specific number here — including gomorphy's own — as a
starting point for your own benchmark, not a settled fact.

## License

All four projects use a permissive OSS license for their Go code (MIT:
gomorphy, jus1d, AlexMaxy; Apache-2.0: SteosMorphy) — no copyleft
concerns on the code itself, in any of them.

The dictionary **data** is a separate question, and this is where
gomorphy's "don't embed the dictionary" architecture has a real
licensing consequence: jus1d and AlexMaxy both embed OpenCorpora data
directly in their binaries and both explicitly flag the CC BY-SA 4.0
attribution+ShareAlike obligation this creates for anyone distributing
a binary built with their library. SteosMorphy embeds ~444.6 MB of its
own dictionary without stating its license anywhere in the repository
— worth checking directly with that project before depending on it in
anything you plan to distribute. gomorphy sidesteps the question
structurally: since no dictionary data ships with the library or CLI,
gomorphy itself carries no CC BY-SA (or other) data-license
obligation — but whichever dictionary *you* download and build with
`gomorphy build` carries its own source's license (OpenCorpora, CC BY-SA
4.0; UniMorph, CC BY-SA 3.0; `pymorphy2-dicts-ru`, its own PyPI
license), same as it would for any of these libraries.

## Maintenance activity

As of 2026-09-19: jus1d/gomorphy (created 2026-02-26, last push
2026-09-17, 0 stars) and AlexMaxy/gomorphy (created 2026-08-08, last
push 2026-08-27, 0 stars, 1 fork, an explicit partial fork of
jus1d/gomorphy) both read as small, single-purpose, recent projects
with minimal external visibility so far. SteosMorphy (created
2025-09-03, last push 2026-06-25, 24 stars, 3 forks, 2 open issues) is
the most established of the three by visible community signal, though
its README has unfinished sections (the table of contents links to
"Architecture and memory usage," "Thread safety," "Dependencies and
environment," and "Grammeme reference" sections that don't exist in
the current README).

## Summary

If you need **fuzzy/typo-tolerant lookup**, **more than one dictionary
source or open dictionary at once**, or a **CLI you can script without
writing Go**, none of the three alternatives currently offer any of
that — gomorphy is the only option here. If you need **word
inflection/generation** (producing a specific grammatical form of a
word, not just parsing one), AlexMaxy/gomorphy or SteosMorphy are
closer fits than gomorphy today. If you want the dictionary to work
immediately with no download/build step, all three alternatives beat
gomorphy on that specific point, at the cost of a fixed, aging
snapshot embedded in the binary.
