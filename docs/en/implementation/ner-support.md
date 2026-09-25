# NER support for lexicon (1.2.0)

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
That spec splits the work into two releases; this document covers
**1.2.0** — the small, mostly-fixes release that unblocks lexicon/genodex.
1.3.0 (tag helpers, lexeme access, `Parse` performance, Builder homonym
grouping, 2-byte alphabet fallback) is separate, not part of this release.

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
