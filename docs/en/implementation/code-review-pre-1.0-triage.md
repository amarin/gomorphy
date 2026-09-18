# Pre-1.0.0 code review: findings triage and the suffix-overflow fix

Two related but separate pieces of work, both done on 2026-09-14: a
full item-by-item triage of the findings in
[code-review-pre-1.0.md](../code-review-pre-1.0.md), and — a separate
task uncovered along the way — a fix for the critical suffix overflow
via sharding.

## Pre-1.0.0 code review — findings triage

Full review report: [code-review-pre-1.0.md](../code-review-pre-1.0.md).

During the review's first pass (before the triage), safe mechanical
findings were fixed along the way: a nil-panic on an `os.Stat` error, a
`response.Body` leak, a duplicate method, two dead types — 131/131
tests green.

An item-by-item triage of the report's findings (a separate session
right after the review) closed almost all the non-critical items:

- **Documentation drift** — `pkg/dictionary`->`pkg/morphology`
  everywhere it was actively misleading.
- **Dead code** — all 4 candidates: `internal/intern`+`internal/stringsx`
  in their entirety, `BlockDAWG`/`Block`/`NewBlockDAWG`/`Flatten`,
  `pkg/common.ErrPath`, `cmd/gomorphy_build`'s `programVersion`
  (replaced with a working `-version` flag).
- **Edge-case bugs** — the Windows build of `internal/mmapx` (a build
  tag + a clear error instead of a compile failure), `internal/xmlscan`
  hanging on a small buffer (a defensive check), a buffer boundary
  splitting a `<!--` comment (a bug + a regression test), numeric XML
  entities (`&#39;`/`&#x27;`), id overflow in tagset/suffix/paradigm
  (see below — suffixes turned out to be a real case, not a
  hypothetical one), a non-atomic `SaveContainer` write (a temp file + rename).
- **Duplication** — a shared `insertKeys` for
  `buildDAWGWithPayload`/`BuildDAWGWithValuesProgress` (closed a blind
  spot in test coverage — the real production DAWG-build path had only
  been tested through an undeduplicated copy), a shared `importAndSave`
  for the `runImport` branches in `cmd/gomorphy`.
- **Splitting up large functions** — `compileAndSave`->`progressPrinter`,
  `SaveContainer`-> 3 named phases, `compileImpl`->`placer`,
  `predict`->`predictForPrefix`.
- **`.golangci.yml`** — was missing entirely; added, plus 17 of 18
  findings from the linter's first real run were fixed (13 errcheck, 3
  staticcheck, 1 unused) — the one finding left untouched
  (`import.go`'s `lGrams`) was deliberately not fixed, since it's tied
  to the critical tag bug (see below).
- **Other** — HTTP->HTTPS for `pkg/opencorpora.RemoteURL`,
  `opencorpora.Error`->`ErrOpenCorpora` with godoc, godoc for `pkg/common`.

Every step was checked with `go build`+`go vet`+`go test -race`
(147/147 tests, finally).

**Both of the review's critical bugs were deliberately left untouched**
as part of this triage — the corrupted OpenCorpora wordform tags and
the suffix id-space overflow; the user made the per-item decision
separately. Both are now closed — suffixes below, tags in a separate
section, "Critical bug: corrupted OpenCorpora wordform tags," closer to
the end of this file.

## Critical finding: more suffixes than fit in a uint16 — FIXED

Found while working on the "an id is assigned with no overflow check"
finding: an added explicit check for 65,536 unique suffixes tripped on
the real `dict.xml` — `lemmas=391842 suffixes=65835 paradigms=16939
tags=3437`. `suffixes=65835` is 299 over a `uint16`'s capacity
(0..65535). Before the check, this silently corrupted ~299 suffix ids
(the overflow wraps to 0, colliding with the first registered suffix);
after it, a proper build error — but `gomorphy_build compile` on the
full `dict.xml` didn't build at all.

**Solution**: sharding, not widening the id type. Rationale (after
brainstorming with an estimate on real data — the suffix count clearly
grows sub-linearly, +5.9%/+0.6%/+0.02% overhead for
round-robin/fill-on-demand/paradigm-grouped at N=2, growing but staying
low for paradigm-grouped even at N=6-7): overflow is expected to be
rare and small, and widening the id type would have meant permanently
raising it for all data for a rare case. Sharding doesn't change the
`.dat` format (named sections with a shard number,
`suffixes-N`/`paradigms-N`/`words.dawg-N`, similar to the already
existing `prediction-N`) and barely touches the public API (only the
additive `Reading.Shard`/`LemmaRef.Shard int`). v1 has a single
strategy, `FillOnDemand` (fill a shard to capacity), behind a
`ShardingStrategy` interface, so other strategies (paradigm-grouped,
etc.) can be added later without reworking the build/lookup pipeline.

Full spec (including a measurement table and alternatives):
[2026-09-14-suffix-sharding-design.md](../superpowers/specs/2026-09-14-suffix-sharding-design.md).
Implementation plan (6 tasks + subagent-driven-development, including
findings from the final review — missing coverage for N>1 shards,
ambiguous CLI output, stale format docs, all closed in a fix wave):
[2026-09-14-suffix-sharding.md](../superpowers/plans/2026-09-14-suffix-sharding.md).

**Result**: `gomorphy_build compile` on the full
`.data/opencorpora/dict.xml` successfully builds into 2 shards (~65s).
Verified twice — when the branch was merged into `master`, and when
the user manually rebuilt it
(`bin/gomorphy_build compile -o .data/opencorpora/opencorpora.dat`).

**Deferred, not part of this work** (see the spec's "Deferred ideas"):
an adaptive/wide variant (`uint32` indices) for dictionaries known to
be huge — a separate implementation/flag alongside sharding, not a
replacement for it; a narrow variant (`uint8` indices) for small,
low-cardinality dictionaries — relevant for Stage 19.

**Follow-up, needs a separate investigation**: the user mentioned a
possible "loss of lemmas from opencorpora," not identified and not
necessarily related to either of the two critical bugs — not
investigated, needs a separate session before any fix.

## Critical bug: corrupted OpenCorpora wordform tags — FIXED

The review's second critical bug (see above). Two independent bugs,
both closed within the same branch: the first via a redesign of how a
form's tag is assembled, the second by the final review of the whole
branch before merging.

### Bug 1: a form's tag was assembled as a snapshot "at open time," not after its grammemes were parsed

**File**: `pkg/morphology/importers/opencorpora/import.go`,
`xmlHandler` (methods `OnForm`/`OnFormEnd`/`OnLemma`/`OnLemmaHeadEnd`).

**The mechanism that existed** (at the time of the finding, before the
fix): `h.curGrams` was only reset at a lemma boundary
(`OnLemma`/`OnLemmaEnd` — the `OnLemmaEnd` method has since been
renamed `OnLemmaHeadEnd`, see below), not between forms within one
lemma. A form's tag was captured as a snapshot of `curGrams` **at the
moment `<f t="...">` opened** — before that form's own `<g v="..."/>`
tags were parsed (they come after the opening tag in the XML). Result:
every form got not its own grammemes, but an accumulated mix of all
preceding forms' grammemes in that lemma; the first form got an empty
tag. Plus a second, related defect: the `<l>` element's own grammemes
(part of speech, animacy, gender) never made it into any form's tag at
all — `OnLemmaEnd` reset `curGrams` before the first form even got to them.

**Confirmed twice, independently**:
1. A crafted repro against a fragment of the real schema
   (`ёж`/`ежа`/`ежу`) — the full "should be / actually got" table is in
   [code-review-pre-1.0.md](../code-review-pre-1.0.md).
2. A manual check against a real, fully rebuilt
   `.data/opencorpora/opencorpora.dat` (after the sharding fix):
   `gomorphy lookup занудами` -> the tag
   `sing,nomn,sing,gent,sing,datv,sing,accs,sing,ablt,sing,ablt,V-oy,sing,loct,plur,nomn,plur,gent,plur,datv,plur,accs`
   instead of the correct `plur,ablt` — a concatenation of grammemes
   from ~11 preceding forms of the "зануда" paradigm, clearly
   demonstrating the accumulation mechanism on real data with a large
   number of forms. The word's and lemma's text weren't corrupted —
   only the tag.

**Scope**: essentially the entire OpenCorpora-imported dictionary
(nouns have ~12 forms, verbs more).

**Why tests didn't catch this before the finding**: at the time of the
finding, `import_test.go` didn't check the tag's content against a
specific form when a lemma had 2+ forms. On top of that, the
`testDictXML` fixture itself didn't match the real schema: lemma
grammemes were written as an attribute (`<l g="NOUN,anim,masc,sing">`),
whereas the real `dict.xml` stores them as nested elements
(`<l t="ёж"><g v="NOUN"/>...</l>`), and `internal/xmlscan/dispatch.go`
only reads the `t` attribute for the `l` tag — the `g` attribute is
never read. So fixing the tag first required deciding the format for
merging lemma and form grammemes (order, dedup), and only then
rewriting the fixture and the regression test to match the final
behavior — which is what was done (see "The fix" below).

**Not investigated, but mentioned by the user**: a possible "loss of
lemmas from opencorpora" — not identified, not necessarily related to
this bug or the suffix bug (the same open follow-up as above).

**The fix** (process: brainstorming -> spec -> plan ->
subagent-driven-development, modeled on the suffix fix): see
[2026-09-15-opencorpora-tag-fix-design.md](../superpowers/specs/2026-09-15-opencorpora-tag-fix-design.md)
and [2026-09-15-opencorpora-tag-fix.md](../superpowers/plans/2026-09-15-opencorpora-tag-fix.md).
The bug's root cause: `</l>` (closing the lemma's headword) mistakenly
fired an event treated as "end of lemma," wiping the lemma's collected
grammemes before even one form was parsed; the event was renamed
(`OnLemma` -> `OnLemmaHeadEnd`), and assembling the form's tag was
moved to `OnFormEnd` (called after all of a form's `<g>` tags are
parsed). The `testDictXML` fixture was rewritten to match the real
schema (`<g v="..."/>` as nested elements), and a regression test,
`TestImportFromXMLFormTagsCombineLemmaAndOwnGrammemes`, was added,
checking the final tag string for every form, not just the absence of empty tags.

### Bug 2: paradigm deduplication ignored tags (`paradigmKeyHash`)

Found by the final review of the whole branch — already **after** bug
1 had been fixed and marked "FIXED" — while tracing the fix down to
real wordform lookups. This bug existed in the code before this
branch too (not introduced by bug 1's fix), but it directly undid bug
1's fix in real-world readings, so it was fixed in the same branch
before merging.

**File**: `pkg/morphology/importers/opencorpora/import.go`,
`paradigmKeyHash`.

**Mechanism**: the function built a paradigm-dedup key by concatenating
the encoded suffix ids (`sk`) and tag ids (`tk`) bytes into a shared
buffer, but computed the final length as `n = copy(...)` instead of
`n += copy(...)` on the second line — the second assignment overwrote
the byte counter instead of accumulating it. Since for the call site in
`ImportFromXML`, `len(sk) == len(tk)` always holds, both assignments
were numerically equal, and `buf[:n]` silently returned only the suffix
bytes — the tag bytes were physically written into the buffer at the
right offset, but never made it into the returned string. Result:
`paradigmsDedup[hash]` — the map deciding whether to collapse two
lemmas' paradigms into one — was keyed only on the suffix id set,
completely ignoring the tag ids. Two lemmas with the same suffix set
but different grammemes (e.g. one NOUN,anim, another NOUN,inan, both
with suffixes `""`,`"а"`) collapsed into one stored paradigm, and the
second lemma's forms silently got the first one's tags.

**Scope**: any pair of OpenCorpora lemmas with a matching wordform
suffix set but different lemma or form grammemes — not rare among
nouns that differ only in animacy, gender, etc. while sharing the same
declension paradigm.

**Why tests didn't catch this**: `TestImportFromXMLParadigmsDedup` only
checked `len(d.Paradigms[0]) <= 3` — such a check misses exactly the
excessive-collapse case (this bug's symptom), rather than catching it.
The form-tags regression test
(`TestImportFromXMLFormTagsCombineLemmaAndOwnGrammemes`, added when bug
1 was fixed) didn't catch this either: it checks that the right tag
strings are registered in `d.TagSet.Tags`, but not that a specific
*word*, when looked up, resolves to *its own* tag — which is exactly
what the paradigm-dedup bug breaks. The form-tag computation itself
(`gramm` in `OnFormEnd`), which bug 1's fix addressed, was already
correct by this point — a later pipeline step was broken.

**The fix**: `paradigmKeyHash` was rewritten to `n += copy(...)`
(accumulating instead of overwriting); while at it, the fixed
`[1024]byte` buffer was replaced with an unbounded one
(`make([]byte, 0, ...)` + `append`), so a buffer overflow with a very
large number of forms in a lemma couldn't silently repeat the same bug
class in the future. The `testDictXML` fixture gained a lemma "дом"
(inanimate) with the same suffix set as the lemma "кот" (a noun,
animate), but different grammemes — exactly the collision the old hash
function missed. `TestImportFromXMLParadigmsDedup` was replaced with an
exact check on the paradigm count (`assert.Equal`, not
`LessOrEqual` — an exact count catches both insufficient and excessive
dedup). An end-to-end regression test,
`TestImportFromXMLWordResolvesOwnLemmaTag`, was added, resolving a word
through the DAWG payload -> paradigm -> tag (rather than just checking
that a tag string exists somewhere in `TagSet`) and checking that "дом"
gets its own tag (`inan`), never "кот"'s tag (`anim`).

Both critical findings from the pre-1.0.0 code review are closed.
