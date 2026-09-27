# NER support for lexicon (1.2.0, 1.3.0)

## Context

A new module, **lexicon** (`~/dev/mine/lexicon`, spec
`docs/specs/2026-09-24-lexicon-design.md`), builds text analysis and
dictionary NER on top of gomorphy. Its first host is genodex (search
normalizer P1, NER E9). genodex's P1 plan listed two blocking requests
("Task 0": `Reading.Predicted`, `OpenBytes`) and two desirable ones
(content hash without `info`, `CharPolicy` in `BuilderOptions`). A review
of gomorphy 1.1.0 source for NER needs found further gaps, some of which
were bugs rather than missing features.

Full design and rationale:
[2026-09-24-ner-support-design.md](../superpowers/specs/2026-09-24-ner-support-design.md).
That spec splits the work into two releases. **1.2.0** (items A-F, below)
is the small, mostly-fixes release that unblocked lexicon/genodex.
**1.3.0** (items G-L, below) adds tag helpers, lexeme access (`Forms`/
`Inflect`), `Parse` performance work, Builder homonym grouping, the
2-byte alphabet fallback, and ending-based prediction for sharded
(OpenCorpora/UniMorph) dictionaries.

## 1.2.0

**A. Known-word flag.** `Reading.Predicted` and `LemmaRef.Predicted` are
new `bool` fields: `true` when a reading came from ending-based suffix
prediction rather than an actual dictionary entry, `false` for a real
dictionary reading. `Dictionary.IsKnown(word)` and
`MultiDictionary.IsKnown(word)` report dictionary membership directly —
lower-cased, `CharPolicy` applied, exact lookup only, never predicting.
Before this, `Parse` silently fell back from exact lookup to prediction
and callers had no way to tell a dictionary reading from a guess; a small
NER dictionary would "predict" a lemma for almost any word, which is
exactly wrong for a component whose job is to flag known names. The CLI
`lookup` command appends `(predicted)` next to predicted readings so
manual dictionary inspection shows the same distinction.

**B. In-memory open.** `OpenBytes(data []byte) (*Dictionary, error)`
opens a GMOR-format dictionary directly from a byte slice — typically one
embedded with `//go:embed` — with the same checksum verification as
`Open`, but no mmap. `data` is not copied and must outlive the
`Dictionary`; DAWG sections that land on a 4-byte boundary are used
zero-copy, misaligned ones (which `//go:embed` gives no guarantee
against) are copied. Because it does not use mmap, `OpenBytes` works on
Windows, where `Open` is not supported. This closes the gap that made it
impossible to ship a gomorphy dictionary inside a Windows binary without
writing it to a temp file first.

**C. Case consistency.** `Builder.AddForm`/`AddLemma` and `ImportTSV` now
lower-case the word and lemma columns with `strings.ToLower` — the same
function `Parse` already uses on its input. Before this fix, a form
registered as «Москва» was stored verbatim; `Parse("москва")` would
"find" it, but only via ending prediction (a real dictionary reading was
never reachable, since nothing in the DAWG matched the lower-cased query
exactly). `Fuzzy`/`FuzzyTop` also lower-case their query now, so
`Fuzzy("Москва", 0)` finds the same stored `москва` that `Fuzzy("москва",
0)` does. Tags are opaque strings and are never lower-cased or
normalized by this change. This is filed as "Fixed" in the CHANGELOG, not
"Changed": the old behavior was a bug, not a documented contract.

**D. Public CharPolicy.** `CharPolicy` and `Substitution` are now exported
as type aliases of the internal types (`type CharPolicy =
internal.CharPolicy`), plus constructors `NewCharPolicy(subs...)`,
`RussianCharPolicy()` (е→ё) and `NoCharPolicy()` (explicit "no
substitutions"). `BuilderOptions` gained a `CharPolicy *CharPolicy`
field; `nil` picks a default by language — е→ё for `Language` "ru" (and
for an empty `Language`, which already meant "ru" elsewhere in the
codebase), no substitutions for any other language. Because `CharPolicy`
is now a real exported type, `UniMorphOptions.CharPolicy` (an alias of
`unimorph.Options`) becomes settable from outside the module for the
first time, with no type change at all. `Fuzzy`/`FuzzyTop` apply the
dictionary's `CharPolicy` in the Levenshtein walk: a substitution pair
now costs 0 instead of 1, matching how `Parse`/`SimilarItems` already
treat it. The substitution stays directional (query rune matches a
different stored rune), so a query "ё" against a stored "е" still costs
1 — `Fuzzy` and `Parse` agree.

**E. Content hash.** `Dictionary.ContentHash() string` returns a stable
xxh3-128 hex digest of the dictionary's content sections, explicitly
excluding `info` (`BuiltAt`, `LibraryVersion`, `Source`…). Re-saving an
unchanged dictionary, or opening it via `Open`/`OpenBytes`, keeps the
digest; two dictionaries with equal `ContentHash` parse every word
identically. This lets a host (lexicon's hot-reload path) detect a
genuinely new dictionary file without being fooled by `SaveTo` always
stamping a fresh `BuiltAt`. Each section is framed as `name, 0x00, len,
data` before hashing, so moving bytes between adjacent sections cannot
produce a colliding digest. The first call re-encodes every section
through the same path `SaveTo` uses (for a large dictionary, that copies
the words DAWG); the result is cached with `sync.Once`, so repeated calls
are cheap.

**F. Toolchain and lifecycle.** `go.mod`'s `go` directive is lowered from
`1.27.1` to `1.25.0`, per the owner's standing policy "current Go minus
two minor versions" (Go 1.27 is current), with an added `toolchain
go1.27.1` line for development builds. No dependency was downgraded — see
"Findings" below. `Dictionary.Close`'s doc comment now spells out the
lifecycle rule explicitly: it must not be called while other goroutines
may still call methods on the dictionary (directly or through a
`MultiDictionary`), because in-flight `Parse`/`Lemma`/`IsKnown`/
`Fuzzy`/`FuzzyTop`/`ContentHash` calls read the mmap region, and
unmapping it under them crashes the process with SIGSEGV/SIGBUS rather
than a recoverable panic. Values already returned (`Reading`, `LemmaRef`,
`FuzzyMatch`, `BuildInfo` and all their strings) are documented as
independent copies that stay valid after `Close` — verified, not just
asserted (see "Findings" below and `TestReturnedStringsSurviveClose`).

## 1.3.0

**G. Tag helpers.** `Grammemes(tag string) []string`, `HasGrammeme(tag, g
string) bool` (allocation-free), `POS(tag string) string` (first
grammeme) and `Reading.HasGrammeme(g string) bool` split a native tag
into grammeme tokens on `,`, ` ` (space) and `;` — the separators that
cover every tag format gomorphy imports: OpenCorpora/pymorphy2's
`"NOUN,anim,masc,Surn sing,ablt"` and UniMorph's `"N;GEN;SG"`. Tags stay
native strings: `pkg/morphology/tagmap` is not extended with these
tokens, since UniMorph Schema has no `Name`/`Surn`/`Patr`/`Geox`
dimensions to map them onto — `Surn`, `Name`, `Patr`, `Geox` are plain
grammeme tokens like any other, not `tagmap`-known. This gives lexicon
(and any other caller) a name-entity-adjacent grammeme check
(`HasGrammeme(r.Tag, "Surn")`) without hand-rolling a splitter over every
tag format gomorphy supports.

**H. Lexeme access.** `Dictionary.Forms(r Reading) []Reading` returns
every form of `r`'s lexeme — the paradigm `r.Para` in `r.Shard`, stem
derived from `r.Word` and `r.Form` — in paradigm order (form 0 is the
lemma); `Dictionary.Inflect(r Reading, want ...string) []Reading` returns
the subset of `Forms(r)` whose tags contain every grammeme in `want`,
best match first (fewest grammemes differing from `r.Tag`, the symmetric
difference of the two grammeme sets, ties by paradigm order). Both are
mirrored on `*MultiDictionary`, dispatching by `r.Dict`. This implements
the `Forms`/`Inflect` sketch that `docs/en/todo.md` carried since
1.1.0-era planning; the spec's `want []string` became `want ...string`
(variadic) to match the design doc, not the older sketch — see D-14.
Both take an already-resolved `Reading` rather than a bare word string,
reusing `Parse`'s own homonym disambiguation instead of duplicating it.
Forms of a predicted reading are generated from the predicted paradigm
and keep `Predicted == true` on every result — as much a guess as the
source reading, not special-cased. `Prob` is always 0 on generated forms
(probabilities describe the parsed word, not a generated one).

**I. Parse performance.** `Parse`/`ParseAppend` were rewritten end to end
(`pkg/morphology/parse.go`): single-shard dictionaries (pymorphy2,
Builder, ImportTSV, and now most OpenCorpora/UniMorph imports) are
searched inline, with no goroutine/`WaitGroup`/channel; multi-shard
dictionaries keep one goroutine per shard, each appending into its own
slice as before. Runes are encoded into a small stack buffer via a new
concrete `(*DenseAlphabet).EncodeRune(r) ([2]byte, int, bool)` instead of
`Encode(string(r))` (an interface method would have made the buffer
escape — see D-15). Payload values decode into a fixed array instead of
through a base64 string allocation. Probability lookup builds its key in
a byte buffer instead of `key+":"+tag` string concatenation. The new
`func (x *Dictionary) ParseAppend(dst []Reading, word string) []Reading`
lets a caller reuse one result slice across many `Parse` calls instead of
letting each call allocate its own; `Parse` is now `x.ParseAppend(nil,
word)`. New benchmarks (`BenchmarkParseKnown`, `BenchmarkParsePredicted`,
`BenchmarkLemma`, `BenchmarkIsKnown`, `BenchmarkFuzzyTop`,
`BenchmarkParseAppend`) run against both the fixture dictionary and a
real `.dat` selected via `GOMORPHY_BENCH_DICT`. Numbers: see
"Performance" below. `Parse` results are unchanged in content and order
for every existing dictionary file — every snapshot test
(`readingsSnapshot`, `semanticSnapshot`, `TestMergeRealDictionary`) stays
green, and `LookupEach`'s walk order matches `SimilarItems`'s (straight
path before substitution branches, at the cost of walking the straight
path twice — see D-16).

**J. Builder homonyms.** `Builder.AddForm`/`AddLemma` (and `ImportTSV`,
which builds on `Builder`) now group entries into lexemes by lemma
**and** part-of-speech class, not lemma text alone: «знать» tagged NOUN
and «знать» tagged INFN now become two separate paradigms instead of one
lexeme mixing a noun and a verb's forms. The class table folds
INFN/VERB/PRTF/PRTS/GRND into one "verb" class, ADJF/ADJS/COMP into one
"adjective" class (OpenCorpora-style tags), and V/V.PTCP/V.CVB/V.MSDR
into one class for UniMorph-style bundles with the POS token first
(`internal.POSClass`) — not the raw first grammeme, which would have
split every OpenCorpora verb/adjective into several paradigms (see
D-13). An entry with an empty or POS-less tag joins the lemma's first
lexeme, unchanged from before. This is always on, no `BuilderOptions`
field (owner decision 2026-09-24, Q1) — dictionaries rebuilt from
homonymous input differ from 1.2.x output; already-built `.dat` files are
unaffected, since the DAWG bytes on disk don't change by themselves.

**K. Builder 2-byte alphabet fallback.** `RecompileDense` (shared by
`Builder`, `ImportTSV`, `Merge`, and every `…Dense` importer —
`CompileFromXMLDense`, `CompileFromUniMorphDense`, `OpenPyMorphyDense`,
see D-18) now chooses a 2-byte alphabet width when the dictionary's
distinct-rune count exceeds width 1's 254-rune capacity, reusing the
fallback logic `Merge` already had, instead of returning a hard error.
`TestRecompileDenseAlphabetOverflowReturnsError`, which asserted the old
error, is rewritten to assert the new 2-byte dictionary works correctly
instead.

**L. Prediction for sharded dictionaries.** Before 1.3.0, `CompileFromXML*`
and `CompileFromUniMorph*` built dictionaries with no ending-based
prediction: an out-of-dictionary word made `Parse` return `nil`, so H's
`Forms`/`Inflect` had nothing to work from for OpenCorpora/UniMorph
imports. The blocker was sharding, not the source: a dictionary with more
than 65536 suffixes splits into shards (`FillOnDemand`, 65536 suffix ids
per shard) — today the UniMorph import has 2 shards, the OpenCorpora
import only one (`words.dawg-0`) — while `internal.BuildPrediction`
handled one shard only and `predictForPrefix` resolved every prediction
against shard 0.

*Format.* A sharded prediction is one DAWG per paradigm prefix id, in a
new section named `pred-sharded-P` — not `prediction-sharded-P`, because
GMOR catalog entries hold section names in a fixed 16-byte field and the
longer name doesn't fit (found during implementation, not anticipated by
the spec text — see D-23). Its key is a word suffix of
max(rune length of the form's suffix, 1)..5 runes (see *Key lengths*
below); its value is `count(BE16) | para(BE16) | form(BE16) | shard(BE16)` — 8 bytes
instead of the 6-byte `prediction-N` value single-shard dictionaries
still use. `internal.Dictionary` gained a `PredictionSharded bool` field
that tracks which value width is in play; `Open`/`OpenBytes` set it from
the section name found, `SaveTo` picks the section name from it. A file
holds either `prediction-N` or `pred-sharded-N`, never both.
Single-shard dictionaries (pymorphy2, Builder, ImportTSV, and the
OpenCorpora import) store prediction as 6-byte `prediction-N`, readable by
1.2.x — so a 1.3.0 OpenCorpora `.dat` predicts under 1.2.x too;
multi-shard ones (UniMorph) as `pred-sharded-N`, which 1.2.x skips. The
section is genuinely new, not an 8-byte reinterpretation of the old name,
because gomorphy 1.2.x must keep opening a new multi-shard `.dat` without
prediction rather than misreading 8-byte values as if they were 6-byte
shard-0 ones — an unknown section name is the only way to make that
safe.

*Building and lookup.* `internal.BuildPrediction`/`BuildPredictionFrom`
were generalized to accept dense dictionaries too (keys decoded through
the alphabet), since `shardPairs` needed to feed pairs that carry a
shard; the "must still be raw (pre-`RecompileDense`)" precondition this
removed is recorded as D-20. `predictForPrefix` lost its
`predictionShard = 0` constant: an 8-byte value takes its shard from
`v[6:8]`, a 6-byte value still means shard 0, and a shard at or past the
dictionary's shard count is skipped like an invalid `para` already is.
The predicted `Reading` carries that `Shard`, so H's `Forms`/`Inflect`
work on a predicted reading from any shard unchanged. `CompileFromXML*`
and `CompileFromUniMorph*` now build prediction by default, before
`RecompileDense`, through the shared `finishCompiled` helper (D-19: the
importers themselves can't build prediction — `productive()` lives in
`pkg/morphology`, which imports the importer packages, so the
`morphology.CompileFrom*` wrappers do it instead). Opt-out:
`XMLOptions.NoPrediction` via the new `CompileFromXMLWithOptions(r,
XMLOptions{Progress, Dense, NoPrediction})` that the four
`CompileFromXML*` functions wrap, `UniMorphOptions.NoPrediction`, or CLI
`gomorphy build opencorpora|unimorph --no-prediction`; `pymorphy2` has no
such flag and rejects `--no-prediction` with an error, since its
prediction comes from its own source files, not from a `BuildPrediction`
call (D-21). `MergeWithOptions`'s `RebuildPrediction` no longer needs a
single-shard result: it now goes through the generalized
`BuildPrediction` over every output shard, so `ErrPredictionSharded` is
never returned any more (kept exported, marked `Deprecated`).

*Key lengths (ruling R14, found by the final review).* The first
implementation indexed every reading under the word's last 1..5 runes,
including keys shorter than the form's own suffix. An unknown word
matching such a key got a (paradigm, form) whose suffix it does not end
with: `readingForm`'s `TrimSuffix` was a no-op and the lemma became word +
lemma suffix («зя» → «зяезти», «ся» → «сяться» on OpenCorpora, «ю» →
«юный» on UniMorph, «глокая» → «глокаявысокий» with unpruned prediction),
and `Forms` returned nil for it. Like pymorphy2's compiler, a reading now
yields keys of max(len(form suffix), 1)..5 runes, none when its form
suffix is longer than 5 runes, and none when its form has a non-empty
paradigm prefix (OpenCorpora «по»/«наи» comparatives: the prefix-0 DAWG
must only predict words that need no prefix). This applies to every
prediction build — Builder/ImportTSV/`Merge` too, so their prediction
bytes differ from 1.2.x (the format does not). `predictForPrefix` also
skips a value whose form prefix/suffix the candidate word does not
start/end with; this cleans Builder/merged files written before the fix
without a format change and leaves pymorphy2 files unchanged (their keys
already follow the rule; Parse output for 33 unknown words on
`pymorphy.dat` is identical before and after).

*Pruning.* The first real-dictionary measurement (Task 12 Step 5)
tripped the spec's 50%-growth gate: unpruned prediction grew
`opencorpora.dat` from 10,669,636 to 24,112,684 bytes (+126.0%) and
`unimorph.dat` from 11,085,089 to 18,556,233 bytes (+67.4%), and an
unknown UniMorph word (`бутявкающий`) took up to ~340 µs to predict
(853 KB allocated per call) — short endings collect thousands of
`(paradigm, form)` candidates without a cap. Per an owner decision made
2026-09-27, mid-implementation (D-24), `CompileFromXML*`/
`CompileFromUniMorph*` now prune prediction the way pymorphy2's own
dictionary compiler does, via `internal.ImportPredictionPruning`
(`MinParadigmPopularity: 3`, `MinEndingFreq: 2`, `MaxFormsPerClass: 1`):
readings of a paradigm used by fewer than 3 lemmas don't feed prediction,
a suffix key attested by fewer than 2 readings is dropped, and per suffix
and part-of-speech (the tag's first grammeme) only the single most
attested `(paradigm, form, shard)` survives. `Builder`, `ImportTSV` and
`Merge` keep unpruned prediction (`internal.BuildPrediction`'s zero-value
`PredictionPruning` keeps everything, proven identical to the unpruned
build by `TestBuildPredictionPrunedZeroIsUnpruned`):
pruning by lemma-paradigm popularity would erase all prediction from the
small, thematic dictionaries those two are typically used for. The
thresholds are internal constants, not a public option (YAGNI) —
a `Merge` with `RebuildPrediction` over an OpenCorpora/UniMorph base
therefore still rebuilds unpruned, large prediction; noted as backlog in
`docs/en/todo.md`. The pruning approximates pymorphy2 rather than
reproducing it: gomorphy counts ending frequency after the productive-tag
filter, and its per-class candidates are (paradigm, form) rather than
pymorphy2's (form suffix, tag, prefix), so byte-parity with pymorphy2's
prediction files is not expected.

After pruning, both real-dictionary measurements moved well inside the
gates: `opencorpora.dat` 10,669,636 → 14,887,124 bytes (+39.5%, was
+126.0%), `unimorph.dat` 11,085,089 → 12,564,297 bytes (+13.3%, was
+67.4%) — remeasured after R14 (14,895,196 / 12,579,657 bytes before it);
build time with pruned prediction was ≈119 s for OpenCorpora and
≈51 s for UniMorph (Apple M4 Pro); parsing an unknown word with pruned
prediction took ≈1.0-1.1 µs for OpenCorpora («бутявкающий»,
«шмуклерами») and ≈1.3-1.5 µs for UniMorph («бутявкающий», «глокая») —
down from UniMorph's unpruned ≈340 µs/853 KB per call, and comfortably
under the spec's 50 µs prediction target. `productive()` itself is
unchanged: UniMorph `rus` has exactly five parts of speech (`N`, `ADJ`,
`V`, `V.PTCP`, `V.CVB`), all open classes, so nothing gets filtered —
a UniMorph-specific nonproductive list would have nothing to remove.

**Performance.** `go test ./pkg/morphology/ -bench . -benchmem -count=5`
(Go 1.25, arm64, Apple M4 Pro); "before" is the pre-Task-9 branch state,
"after" is post-1.3.0. Medians of 5 reruns.

Fixture dictionary (`benchDict`):

| Benchmark | ns/op before → after | B/op before → after | allocs/op before → after |
|---|---:|---:|---:|
| ParseAppend (new) | — | — | 175 → **0** |
| ParseKnown | 1121 → **201** | 592 → **96** | 24 → **1** |
| ParseKnownYo | 1238 → **249** | 664 → **112** | 30 → **3** |
| ParsePredicted | 1714 → **345** | 904 → **104** | 32 → **2** |
| Lemma | 1253 → **240** | 688 → **192** | 26 → **3** |
| IsKnown | 283 → **221** | 256 → **224** | 18 → **10** |
| FuzzyTop | 14085 → **14000** | 5912 → **5912** | 151 → **151** (unaffected — doesn't call `Parse`) |

Real pymorphy dictionary (`.data/pymorphy/pymorphy.dat`):

| Benchmark | ns/op before → after | B/op before → after | allocs/op before → after |
|---|---:|---:|---:|
| ParseKnown/кота | 1330 → **437** | 880 → **272** | 26 → **2** |
| ParseKnown/стали | 2215 → **1300** | 2816 → **1489** | 42 → **10** |
| ParseKnown/ежик | 1359 → **275** | 672 → **112** | 34 → **3** |
| Lemma | 2609 → **1600** | 3472 → **2145** | 49 → **17** |
| FuzzyTop | ~52ms → **~51ms** | 37858671 → **37858700** | 7743 → **7743** (unaffected) |

`ns/op` and `allocs/op` improved everywhere `Parse` is involved;
`FuzzyTop`, which never calls `Parse`, is unchanged — confirming no
regression outside the touched code paths. `FuzzyTop` on the real
pymorphy dictionary (~51-52 ms/op, ~37.9 MB/op) is a known, pre-existing
cost — a full Levenshtein-DFA walk over the whole DAWG — left unchanged
by this release and worth keeping in mind alongside the sub-microsecond
`Parse`/`Lemma` numbers above.

Real OpenCorpora and UniMorph dictionaries, built **with pruned
prediction** (`ParseKnown` unaffected by pruning; `ParsePredicted` is the
number pruning targeted):

| dictionary | ParseKnown/кота | ParseKnown/стали | ParseKnown/ежик | ParsePredicted/бутявкающий | ParsePredicted/глокая |
|---|---:|---:|---:|---:|---:|
| opencorpora | ~282 ns | ~758 ns | ~274 ns | ~1049 ns (**~1.0 µs**) | ~1066 ns (**~1.1 µs**) |
| unimorph | ~1732 ns | ~2407 ns | ~1783 ns | ~1435 ns (**~1.4 µs**) | ~1246 ns (**~1.2 µs**) |

(Medians of 5 runs, remeasured after R14.)

`TestParseAllocs` (`pkg/morphology/parse_alloc_test.go`, fixture
dictionary, no `-race`):

| word | `ParseAppend` allocs | `Parse` allocs | bound (`ParseAppend`) | bound (`Parse`) | 1.1.0 baseline (`Parse`) |
|---|---:|---:|---:|---:|---:|
| кот | 0 | 1 | 1 | 2 | 22 |
| кота | 0 | 1 | 1 | 2 | 24 |
| бота (predicted) | 2 | 3 | 3 | 4 | 44 |

**Spec targets.** `docs/en/implementation.md`'s Metrics table targets
"Parse (exact) < 10 µs" and "Parse (prediction) < 50 µs" both hold,
comfortably: every measured `Parse`/`ParseKnown` number above, across the
fixture and all three real dictionaries (pymorphy, OpenCorpora, UniMorph)
is under 1.3 µs; every measured `ParsePredicted` number is under 1.5 µs
after pruning. Pruning (above) was exactly what kept UniMorph prediction
inside the 50 µs target — the pre-pruning measurement (~340 µs) would
have missed it by nearly 7x.

### Discrepancies with the spec (found while planning)

- **D-12 (J).** The spec's test case «стекло» (NOUN) / «стекло» (VERB,
  past neut of «стечь») does not exercise the change: the verb form's
  lemma is «стечь», not «стекло», so the Builder already puts it into a
  different paradigm. The plan tests «знать» (NOUN «знать» vs INFN
  «знать»), which really shares the lemma text.
- **D-13 (J).** "Group by `(lemma, POS(tag of the lemma form))`": an
  entry carries only its own tag, and grouping by the raw first grammeme
  would split every OpenCorpora verb into INFN/VERB/PRTF/PRTS/GRND
  paradigms and adjectives into ADJF/ADJS/COMP. The plan groups by a
  **POS class** (`internal.POSClass`, with a fold table for OpenCorpora
  and UniMorph) and defines what POS-less entries do.
- **D-14 (H).** `docs/en/todo.md` sketched `Inflect(r Reading, want
  []string)`; the spec says `want ...string`. The plan follows the spec.
- **D-15 (I).** "`EncodeRune` API on the alphabet": adding an interface
  method that writes into a caller buffer would make the buffer escape
  (dynamic call); the plan adds a concrete
  `(*DenseAlphabet).EncodeRune(r) ([2]byte, int, bool)` and a type switch
  in `followRuneVia`.
- **D-16 (I).** A naive callback version of `SimilarItems` changes result
  order (substitution branches before the straight match). `Parse`
  results must not change, so `LookupEach` walks the straight path
  first, then the branches — at the cost of following the straight path
  twice.
- **D-17 (I).** Benchmarks use `b.Loop()` (Go 1.24; allowed by the `go
  1.25.0` directive from 1.2.0, owner decision 2026-09-24, answer 15 —
  the earlier 1.22 plan forbade it). `AllocsPerRun` bounds are still
  skipped under `-race` (the race detector changes allocation counts);
  `b.Loop` does not change that.
- **D-18 (K).** `RecompileDense` is shared with `CompileFromXMLDense`,
  `CompileFromUniMorphDense` and `OpenPyMorphyDense`; the fallback
  therefore also changes them (from an error to a working 2-byte
  dictionary). The spec only mentions Builder/ImportTSV. The existing
  test `TestRecompileDenseAlphabetOverflowReturnsError` asserts the old
  error and is rewritten.
- **D-19 (L).** The spec says the importers build prediction. They
  cannot: `productive` lives in `pkg/morphology`, which imports the
  importer packages. Prediction is built by the `morphology.CompileFrom*`
  wrappers (`finishCompiled`); `unimorph.Options.NoPrediction` is a field
  of the importer's options that only those wrappers honour, and is
  documented as such.
- **D-20 (L).** Generalizing `BuildPrediction` through `shardPairs` makes
  it accept dense dictionaries too (keys are decoded through the
  alphabet); the "must still be raw" precondition goes away. `Merge`'s
  rebuild re-reads all output shards instead of special-casing a reused
  shard 0.
- **D-21 (L).** `pymorphy` has no `--no-prediction`: its prediction comes
  from the source files. The CLI rejects the flag for `pymorphy` with an
  error rather than silently ignoring it.
- **D-22 (L).** Examples and CLI tests built on OpenCorpora XML fixtures
  via `CompileFromXML` now get prediction; `MultiDictionary` examples may
  print extra predicted readings. Task 12 updates their outputs instead
  of opting out.
- **D-23 (L).** The spec's own format sketch named the new section
  `prediction-sharded-P`; GMOR catalog entries hold section names in a
  fixed 16-byte field, which the longer name overflows, so it was
  renamed to `pred-sharded-P` during implementation. `global-constraints.md`
  and the design spec were both updated to the short name.
- **D-24 (L).** The spec's own "Measurements" note said the default-on
  decision would be revisited if a real dictionary grew by more than
  ~50%; Task 12 Step 5 tripped that gate (OpenCorpora +126.0%, UniMorph
  +67.4%). Rather than flip prediction to opt-in, the owner decided
  (2026-09-27) to prune it like pymorphy2's own compiler and keep
  default-on if the pruned numbers came in under the gate — they did
  (+39.5% / +13.3% after R14), so default-on shipped as originally planned, just
  pruned.
- **D-25 (I).** `internal.DAWG.LookupEach`'s decode buffer is a
  `sync.Pool`-backed fixed array rather than a stack array: a slice
  handed to an opaque `func([]byte, ...)` callback always escapes to the
  heap (the compiler can't prove the callback doesn't retain it), so a
  stack buffer would allocate on every call anyway; pooling keeps the
  common case at 0 allocations instead.
- **D-26 (I).** The completer's DAWG-walk step (used by fuzzy completion)
  and `LookupEach`'s straight-path walk were unified into one shared
  `internal.DAWG.nextTerminal` helper instead of keeping two independent
  walk implementations, once it became clear both needed the same
  "advance one unit, report if it's a terminal" logic.
- **D-27 (J).** Two internal merge tests used placeholder tags `"N"`/`"G"`
  for their entries; once Builder started grouping by POS class (J) those
  placeholders no longer round-tripped through a real POS-class lookup
  correctly, so they were switched to real-looking tags `"N,nomn"`/
  `"N,gent"` that carry an actual class.

### Testing (1.3.0)

Per the spec's Testing section for items G-L:

- **G** — `pkg/morphology/tag_test.go`, `pkg/morphology/internal/tag_test.go`:
  `Grammemes`/`HasGrammeme`/`POS` against OpenCorpora- and
  UniMorph-shaped tags, including the `"Surn sing"` space-separated case;
  `Reading.HasGrammeme` delegates correctly; empty-tag edge cases.
- **H** — `pkg/morphology/lexeme_test.go` plus doc examples: `Forms`
  returns the whole paradigm in form order with form 0 as the lemma;
  `Inflect` ranks by symmetric grammeme difference with paradigm-order
  tie-break; a mismatched `Reading` yields `nil`; a predicted `Reading`'s
  forms stay `Predicted == true`; `MultiDictionary` dispatch by
  `r.Dict`, including an out-of-range index.
- **I** — `pkg/morphology/bench_test.go`, `pkg/morphology/parse_alloc_test.go`,
  `pkg/morphology/internal/lookup_each_test.go`: `TestParseAppendMatchesParse`
  (equivalence across single- and multi-shard dictionaries),
  `TestParseAllocs` (allocation bounds, see the table above),
  `LookupEach` walk order equivalence to `SimilarItems`.
- **J** — `pkg/morphology/builder_test.go`, `pkg/morphology/internal/build_test.go`:
  «знать» NOUN/INFN yields two paradigms; a POS-less entry joins the
  lemma's existing lexeme; the OpenCorpora/UniMorph POS-class fold table.
- **K** — `pkg/morphology/internal/dense_recompile_test.go`,
  `pkg/morphology/builder_internal_test.go`: a >254-rune alphabet
  recompiles successfully at width 2 for `Builder`/`ImportTSV`/`Merge`
  and every `…Dense` importer; the old overflow-error test is replaced.
- **L** — `pkg/morphology/internal/prediction_test.go`,
  `pkg/morphology/internal/merge_test.go`,
  `pkg/morphology/prediction_sharded_test.go`,
  `pkg/morphology/compile_prediction_test.go`,
  `cmd/gomorphy/build_test.go`: sharded prediction round-trips through
  `SaveTo`/`Open`; a 1.2.x-shaped reader skips an unknown
  `pred-sharded-P` section without error; `CompileFromXML*`/
  `CompileFromUniMorph*` predict by default and honor `NoPrediction`;
  `--no-prediction` CLI flag, including its rejection for `pymorphy`;
  `MergeWithOptions{RebuildPrediction: true}` on a multi-shard result;
  all four pruning rules (`TestBuildPredictionPrunedParadigmPopularity`,
  `...EndingFreq`, `...MaxFormsPerClass`, and
  `TestBuildPredictionPrunedZeroIsUnpruned` proving `Builder`/`ImportTSV`/
  `Merge` stay unpruned); R14 key lengths
  (`TestBuildPredictionKeysContainFormSuffix`,
  `pkg/morphology/prediction_affix_test.go`: short unknown words on a
  Builder dictionary and the runtime guard on a hand-injected bad value).

## Findings

Four things were checked while planning this release; the resulting tasks
rely on them rather than re-deriving them.

1. **Aliasing audit.** Every code path that hands a string to the caller
   was traced: `internal.DecodeStrings` copies (`string(data[p:p+n])`);
   `DecodeTagSet`/`DecodeBuildInfo` go through `encoding/json` (copies);
   `DecodeAlphabet` copies into `[]rune`; `SimilarItems` builds
   `Item.Key` from the query runes, never from the DAWG, and its values
   come from base64 decoding (an allocation); `Fuzzy` builds
   `FuzzyMatch.Word` via `string(f.path)` or `Alphabet.Decode` (copies).
   The only aliasing in the whole package is `internal.ParseDAWG`'s
   `unsafe.Slice` over the DAWG unit array (plus its `guide` sub-slice) —
   read by every in-flight query, which is exactly why an unsynchronized
   `Close` crashes the process, but nothing ever returned to a caller
   points into that memory. `TestReturnedStringsSurviveClose`
   (`pkg/morphology/lifecycle_test.go`) locks this in with a regression
   test that collects strings from every query method and re-checks them
   after `Close`.
2. **«Москва» was already "found", just as a guess.** A Builder
   dictionary containing only «Москва» returned a reading for
   `Parse("москва")` even on 1.1.0 — but it was a *prediction* (suffix
   match from the prediction DAWG, indistinguishable from a real
   dictionary hit before item A shipped). This is why the fix in item C
   needed item A's `Predicted` flag to be testable at all: the tests
   added in `case_test.go` assert `Predicted == false` for the
   lower-cased form, not merely "returns something".
3. **The `go` directive policy and the dependency table.** With the owner
   policy "current Go minus two minor versions" resolving to `go 1.25.0`,
   the dependency floor was checked directly rather than assumed. As of
   planning, `go.mod`'s per-dependency `go` lines were:
   `golang.org/x/sys v0.47.0` → `go 1.25.0` (the highest; it enters only
   transitively, through `cmd/gomorphy` → `golang.org/x/term v0.31.0`,
   which itself declares `go 1.23.0`), `github.com/klauspost/cpuid/v2
   v2.4.0` → `go 1.24.0`, `github.com/zeebo/xxh3 v1.1.0` → `go 1.22.0`.
   `1.25.0` therefore needs no dependency downgrades, and
   `strings.SplitSeq` (introduced in 1.24, already used in `parse.go`)
   stays usable. A scratch-copy check (`go mod edit -go=1.25.0
   -toolchain=go1.27.1 && go mod tidy`) confirmed the requirement set is
   unchanged and `go vet`/`go build` stay clean; forcing `-go=1.24.0` and
   re-tidying is raised straight back to `1.25.0` by `x/sys`. The
   technical floor after downgrading transitive requirements would be
   `1.22` (set by `xxh3`) — recorded for reference, not used, since the
   policy is "current minus two", not "lowest possible".
4. **Builder output is deterministic.** Building the same entries twice
   and saving both produced byte-identical sections except `info`
   (checked across 20 runs in a scratch test): `BuildPredictionFrom`
   iterates a Go map internally, but `buildDAWGWithPayload` sorts keys
   before encoding, so map iteration order never leaks into the on-disk
   bytes. This is what makes `ContentHash`'s "rebuild the same entries →
   equal hash" test (`content_hash_test.go`) sound rather than flaky.

## Deviations from the spec

The spec's original design sketch differed from what shipped in these
ways (all found or resolved during planning, before implementation):

- **D-1 (E).** The spec described `ContentHash` as "cheap for `Open`ed
  dictionaries (sections already located)". `Dictionary` does not
  actually retain the `internal.Container` after `Open` (only `d` and
  `mm` are kept), so the hash always re-encodes sections, including a
  full copy of `words.dawg`, regardless of how the dictionary was
  opened. The shipped version caches the result with `sync.Once` instead.
  A zero-copy fast path for `Open`ed dictionaries is possible in
  principle but would need to produce the identical digest for
  in-memory dictionaries too — deferred as open question Q5.
- **D-2 (D).** The spec proposed `UniMorphOptions.CharPolicy` become
  `*morphology.CharPolicy` "(converted to the internal type)". That is
  not implementable while `UniMorphOptions` is an alias of
  `unimorph.Options`, because `unimorph` cannot import `morphology`
  (import cycle). The shipped design instead exposes `CharPolicy` and
  `Substitution` as **type aliases** of the internal types, which makes
  the existing field usable from outside with no type change at all —
  strictly more compatible than the spec's original plan.
- **D-3 (C).** The spec's problem statement said a form added as
  «Москва» is "never found". In fact it is returned, but as an
  indistinguishable prediction (see Finding 2 above). The fix is the
  same either way; the tests were written to assert `Predicted == false`
  specifically, not just non-empty output.
- **D-4 (D).** "`nil` → language default (ru: е→ё)" reads, on its own, as
  if the pre-1.2.0 behavior (е→ё for every language) stays for "ru" and
  changes only for others. What actually changed: before 1.2.0 the
  Builder applied е→ё for *every* language unconditionally
  (`internal/build.go:259-262`); 1.2.0 makes that conditional on
  `Language` being `"ru"` or empty. This is a real behavior change for
  non-"ru" Builder dictionaries with no explicit `CharPolicy`, listed
  under "Changed" in the CHANGELOG. Confirmed with the owner 2026-09-24.
- **D-5 (F).** Superseded by the owner's 2026-09-24 decision: the `go`
  directive follows "current Go minus two minor versions" →
  `go 1.25.0`, not the technical floor. No dependency downgrades were
  needed (see Finding 3); `strings.SplitSeq` stays. `go.mod` had no
  `toolchain` line before this release; `toolchain go1.27.1` was added.
- **D-6 (F).** The spec raised an "aliasing concern" as something to
  verify. It came back clean (Finding 1): no string returned to a caller
  aliases the mmap-backed mapping, only the internal DAWG unit arrays
  do. No code change was needed, only the regression test
  (`TestReturnedStringsSurviveClose`).
- **D-7 (D).** The spec left the direction of a substitution pair's
  zero-cost match unspecified. The shipped `Fuzzy` keeps the policy
  directional — a query "е" matches a stored "ё" at cost 0, but a query
  "ё" against a stored "е" still costs 1 — consistent with how
  `Parse`/`SimilarItems` already treated it before this release.
- **D-8 (D).** Not called out in the spec's own Compatibility section:
  the existing `TestFuzzyRuneMetricYo` asserted е/ё distance 1; after
  item D it is 0. This is a real behavior change, listed under
  "Changed" in the CHANGELOG.
- **D-9 (A).** The spec's open question G2 ("what does
  `LemmaRef.Predicted` mean when a lemma mixes exact and predicted
  readings?") turned out to be moot on the current code:
  `MultiDictionary.Lemma` concatenates per-dictionary results without
  cross-dictionary dedup, and a single `Dictionary.Parse` call returns
  either only exact or only predicted readings for a given word — never
  both — so every reading behind one `LemmaRef` always has the same
  `Predicted` value today. The documented rule ("true only if every
  reading behind it is predicted") is kept for the future case where
  cross-dictionary dedup is added.
- **D-10 (E).** The spec said "xxh3-128 over the section payloads"
  without specifying framing. The shipped hash frames each section as
  `name, 0x00, len, data` specifically so that moving bytes across a
  section boundary (e.g. shrinking one section and growing the next by
  the same amount) cannot produce the same digest as the original.
- **D-11 (C).** The spec said `ImportTSV` "inherits" the lower-casing
  fix by virtue of calling `Builder.AddForm`. It does not: `ImportTSV`
  builds `internal.BuildEntry` values itself and never calls `AddForm`,
  so it needed its own `strings.ToLower` calls on the wordform and lemma
  columns — done alongside the `Builder` fix.
- **D-12 (D).** `NewCharPolicy(subs...)` is not named in the spec's own
  API sketch — only `RussianCharPolicy()` and `NoCharPolicy()` are. It
  was added by the implementation plan as the general constructor behind
  both (`RussianCharPolicy`/`NoCharPolicy` are thin wrappers over it) and
  kept in the shipped API: a harmless, symmetrical constructor that a
  caller needs anyway to build a custom (non-Russian, non-empty)
  `CharPolicy` from outside the module, now that `CharPolicy` is a public
  type alias (D-2).

## Final review round (2026-09-25)

A whole-branch review of 1.2.0 after `release: 1.2.0` (commit `616e448`)
found and fixed eight issues, none changing the GMOR format or the
public names fixed by the spec:

1. **UniMorph mixed-case unreachability.** `unimorph.ImportFromTSV`
   stored lemma/wordform text verbatim; a mixed-case UniMorph row (e.g.
   «Аббас») was unreachable by `Parse`/`Fuzzy`/`IsKnown`, all of which
   lower-case their query. Fixed with `strings.ToLower` on the lemma and
   wordform fields, consistent with `Builder`/`ImportTSV` (spec item C);
   tags are left untouched. The OpenCorpora importer does not have this
   bug — its `t=` attributes (the actual DAWG keys) are already
   lower-case in the source format; the `pymorphy2` importer reads
   pre-built binary DAWGs with no string-level entry point to fix. See
   the CHANGELOG's 1.2.0 "Fixed" section for the rebuild note.
2. **Windows install docs overstated the gap.** `docs/*/installation.md`
   said Windows "is not supported"; aligned with README.md's actual
   behavior (`Open` returns a runtime error there, but the code compiles)
   and added that `OpenBytes` works on Windows today as the workaround.
3. **`ContentHash`'s `""` cases were undocumented, and an oversized
   `CharPolicy` panicked instead of erroring.** `ContentHash`'s doc
   comment now states both cases that return `""` (nil dictionary;
   an internal encoding failure). A `CharPolicy` with more than
   `internal.MaxCharPolicySubstitutions` (255) substitutions used to
   reach a `panic` inside `EncodeMeta` only at `SaveTo`/`ContentHash`
   time; it is now rejected earlier, at build time, with a wrapped
   error (`internal.ValidateCharPolicy`, called from
   `buildFromEntries` and `unimorph.ImportFromTSV`). The `EncodeMeta`
   panic stays as a last-resort invariant guard.
5. **`LemmaRef.Predicted` wording unified.** `pkg/morphology/lemma.go`'s
   doc comment now says "true when every reading behind this lemma was
   predicted", matching `docs/en/library.md` and D-9 above.
6. **CLI docs now say the `(predicted)` marker is an optional 6th
   column** (`docs/en/cli.md`, `docs/ru/cli.md`): a dictionary reading's
   line has 5 tab-separated fields, a predicted one has 6.
7. **Two doc-comment lines re-wrapped** to the surrounding ~75-column
   width: `Parse`'s doc comment (`parse.go`) and `MultiDictionary.Close`'s
   (`multidict.go`).
8. **`docs/ru/library.md`'s `MultiDictionary` method list was missing
   `IsKnown`** (present in `docs/en/library.md` and in the actual API,
   `known.go`); added.

## Testing

Per the spec's Testing section for items A–F:

- **A** — `pkg/morphology/known_test.go`: a dictionary word yields
  `Predicted=false`/`IsKnown=true`; an absent-but-predictable word yields
  `Predicted=true`/`IsKnown=false`; Builder dictionaries included;
  `MultiDictionary.IsKnown` mixes both.
- **B** — `pkg/morphology/open_bytes_test.go`: `OpenBytes` parses
  identically to `Open` on the same bytes; corrupted bytes fail the
  checksum; a 1-byte-misaligned buffer still opens correctly (copy path).
- **C** — `pkg/morphology/case_test.go`: a Builder dictionary seeded with
  «Москва» is found by both `Parse("москва")` and `Parse("Москва")` with
  `Predicted == false`; `Fuzzy("Москва", 0)` finds the stored `москва`.
- **D** — `pkg/morphology/char_policy_test.go` and
  `pkg/morphology/fuzzy_test.go`: `NoCharPolicy()` makes «елка» miss
  «ёлка»; the language default finds it for `Language` `""`/`"ru"` and
  misses it for any other language (both `Builder` and `ImportTSV`); an
  explicit `RussianCharPolicy()` overrides the language default; `Fuzzy`
  reports е/ё distance 0 under the Russian policy.
- **E** — `pkg/morphology/content_hash_test.go`: save → reopen →
  `ContentHash` unchanged; rebuilding the same entries yields the same
  hash (relies on Finding 4's determinism); changing one form changes
  the hash.
- **F** — Task 10 Step 6 (toolchain: `go build ./...`/`go vet ./...`
  clean at `go 1.25.0`) plus
  `pkg/morphology/lifecycle_test.go`'s
  `TestReturnedStringsSurviveClose` (Finding 1's no-aliasing guarantee).
