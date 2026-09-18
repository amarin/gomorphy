# Pre-1.0.0 code review

Date: 2026-09-14. Scope: the entire codebase (`cmd/`, `internal/`,
`pkg/`), ~4850 lines of production code + ~3025 lines of tests. Overall
test coverage — 65.8% (`go test ./... -cover`); per-package details are
in the "Test coverage" section below.

Some findings were fixed along the way during the review (safe,
mechanical fixes — see "Fixed along the way"); the rest are decided on
a per-finding basis by the user separately, as agreed.

## 🔴 Critical finding (found during follow-up work on this report, 2026-09-14): more suffixes than fit in a uint16

**Files**: `pkg/morphology/importers/opencorpora/import.go` (building
`suffixList`), `pkg/morphology/internal/paradigm.go` (the `Paradigm`
format — suffixes are addressed by `uint16`).

While working on the "an id is assigned with no overflow check" finding
(the "Bugs / edge cases" section below), an explicit check for 65,536
unique suffixes was added — and it **triggered on the real
`dict.xml`**, not just hypothetically:

```
lemmas=391842  suffixes=65835  paradigms=16939  tags=3437
```

`suffixes=65835` is 299 over a `uint16`'s capacity (0..65535). This
report's earlier estimate ("comfortable margin today (actually
~1K/5K/3K)", see the section on id overflow) was wrong — it wasn't
measured against the full `dict.xml`.

**Consequence with no check** (the code before this finding): `sid :=
uint16(len(suffixList))`, on reaching the 65,536th unique suffix
string, silently wraps to `0` — the new suffix gets an id colliding with
the very first registered suffix's id. Every paradigm referencing a
suffix with id >= 65536 (299 suffixes) silently points to the wrong
text. This happens **during an ordinary build of the dictionary from
the current `dict.xml`**, with no special conditions at all — not
future-proofing, an active bug.

**Current state (after the finding)**: an explicit check was added
(`if len(suffixList) >= 1<<16 { return error }`), turning silent data
corruption into a loud error — `gomorphy_build compile` on the current
`dict.xml` now **doesn't build at all**, failing with `too many unique
suffixes (max 65536)`. This is a deliberate tradeoff: an honest error
is better than silent corruption, but the 1.0.0 release currently
can't build the full dictionary from OpenCorpora's official export.

**Not fixed right now** — the approach and grooming this task in
`docs/en/todo.md` is a decision to make (see "Suffixes: widening the
addressing" in the backlog). Options to discuss: a variable-size index
(`uint16`/`uint32` chosen based on the actual data, with a format
flag), automatically splitting the dictionary into several DAWG
segments searched in parallel, or something else.

**Independently** of the already-known critical tag bug for
OpenCorpora wordforms (below) — both findings are release-blocking, but
concern different parts of the import pipeline (form grammemes vs.
suffix addressing).

## 🔴 Critical finding: OpenCorpora wordform tags are scrambled

**Status as of 2026-09-14 (after triaging the report's other
findings)**: not fixed, remains the only open critical release
blocker (the second critical bug, the suffix overflow above, was fixed
via sharding, see
[implementation/code-review-pre-1.0-triage.md](implementation/code-review-pre-1.0-triage.md)
and the self-contained writeup in [todo.md](todo.md)). The line
numbers below have since shifted due to later refactoring (the bug's
own logic didn't change) — check `OnForm`'s current line via
`grep -n "func (h \*xmlHandler) OnForm" pkg/morphology/importers/opencorpora/import.go`
(around line 302 as of 2026-09-14).

**File**: `pkg/morphology/importers/opencorpora/import.go:225-235`
(`xmlHandler.OnForm`) — line numbers as of when the bug was first found

When importing `dict.xml`, a wordform's grammatical tag (`gramm`) is
captured **at the moment** the `<f t="...">` tag opens:

```go
func (h *xmlHandler) OnForm(text []byte) error {
    ...
    gramm := strings.Join(h.curGrams, ",")   // <- a snapshot taken BEFORE this form's <g> tags are parsed
    frm := formGrams{text: string(text), gramm: gramm}
    lem.forms = append(lem.forms, frm)
    h.curForm = &lem.forms[len(lem.forms)-1]
    return nil
}
```

But a form's own `<g v="..."/>` tags come in the XML **after** the
opening `<f t="...">` (the real dict.xml schema: `<f t="ежа"><g
v="sing"/><g v="gent"/></f>`), and `h.curGrams` **isn't reset between
forms within one lemma** (it's only reset in
`OnLemma`/`OnLemmaEnd`, i.e. at lemma boundaries). As a result, every
form gets not its own grammemes, but an accumulated mix of ALL
preceding forms' grammemes in that lemma; the first form gets an empty
tag.

**Confirmed empirically** against a fragment of the real dict.xml
schema (`<lemma><l t="ёж"><g.../></l><f t="ёж"><g v="sing"/><g
v="nomn"/></f><f t="ежа"><g v="sing"/><g v="gent"/></f><f t="ежу"><g
v="sing"/><g v="datv"/></f></lemma>`):

| Form | Should be | Actually got |
|---|---|---|
| ёж (nominative) | `sing,nomn` | `""` (empty) |
| ежа (genitive) | `sing,gent` | `sing,nomn` |
| ежу (dative) | `sing,datv` | `sing,nomn,sing,gent` |

This explains an anomaly noticed earlier in the session: `gomorphy
lookup кота` -> the tag `sing,nomn` (that's the form "кот"'s tag, not
the genitive "кота"'s).

**A second, independent confirmation** (2026-09-14, a manual check
against a real, fully rebuilt `.data/opencorpora/opencorpora.dat`,
after the suffix-overflow fix — i.e. this isn't an artifact of that
bug, but the same tag bug on production data):

```
$ gomorphy -dict .data/opencorpora/opencorpora.dat lookup занудами
занудами  зануда  sing,nomn,sing,gent,sing,datv,sing,accs,sing,ablt,sing,ablt,V-oy,sing,loct,plur,nomn,plur,gent,plur,datv,plur,accs  para#0/23
```

The tag is a concatenation of ~11 preceding forms' grammemes from the
"зануда" paradigm (should be `plur,ablt` — instrumental plural). The
word and lemma are correct, the tag is corrupted by exactly the
mechanism described above — it's just especially visible on a noun
with a large number of forms (sing+plur x nomn/gent/datv/accs/ablt/loct).

**A second, related defect**: the `<l>` element's own grammemes (part
of speech, animacy, gender — shared across all of a lemma's forms,
e.g. `<l t="ёж"><g v="NOUN"/><g v="anim"/><g v="masc"/></l>`)
accumulate into `h.curGrams`, but `OnLemmaEnd` resets `curGrams` before
the first form even gets to them — these grammemes **never make it
into any tag at all**. Hence tags like `sing,nomn` instead of the
expected `NOUN,anim,masc,sing,nomn`.

**Why tests didn't catch this**: `import_test.go` only checks the
result's structure (a non-empty `TagSet`, the word being findable in
the DAWG, `len(Paradigms) <= len(lemmas)`) — no test checks a tag's
content against a specific form when a lemma has 2+ forms.

**Important for a future fix — the test fixture itself doesn't match
the real schema**: `testDictXML` in `import_test.go` writes lemma
grammemes as an attribute — `<l g="NOUN,anim,masc,sing">` — while the
real `dict.xml` stores them as nested elements — `<l t="ёж"><g
v="NOUN"/><g v="anim"/><g v="masc"/></l>` (verified against
`.data/opencorpora/dict.xml`). For the `l` tag,
`internal/xmlscan/dispatch.go` only reads the `t` attribute
(`s.attr("t")`) — the `g` attribute is never read at all. So in the
current fixture, the "lemma grammemes" branch never exercises the real
code path at all (dead relative to the logic under test), and simply
rewriting the fixture to the right schema isn't enough — the format for
merging lemma+form grammemes (see above) needs to be decided first,
and only then should the fixture and regression test be written to
match the final behavior.

**Scope**: affects practically every wordform imported from
OpenCorpora (nouns have ~12 forms, verbs more) — i.e. the library's
main data source. The word and lemma (the text itself) aren't
affected — it's specifically the grammatical tag that's corrupted.

**Not fixed right now**: needs a well-thought-out redesign of
`xmlHandler`'s state (when to move a form's grammemes from "still
accumulating" to "finalized") and a decision on format (whether to
merge `<l>`'s grammemes with `<f>`'s, in what order) — decisions worth
discussing separately, not something to auto-fix as part of the review.

## Fixed along the way (safe, mechanical fixes)

All fixes are in the same commit; `go build ./... && go vet ./... && go
test ./... -race` green (131/131) at every step.

| File | What it was | What was done |
|---|---|---|
| `pkg/opencorpora/loader.go:IsUpdateRequired` | `os.Stat` returns a non-`ErrNotExist` error (e.g. permission denied) -> `fileStat` stays `nil`, but the code continues and crashes with a nil pointer dereference on `fileStat.ModTime()` | An early `return false, err` for any `os.Stat` error except `ErrNotExist` |
| `pkg/opencorpora/loader.go:IsUpdateRequired` | The `http.Head` response was never closed (`response.Body` with no `Close`) — a connection leak | Added `defer response.Body.Close()` |
| `pkg/opencorpora/loader.go:IsDownloadExists` | A `switch` with two branches doing the identical thing (`errors.Is(err, os.ErrNotExist)` and plain `err != nil`) | Collapsed into a plain `if err != nil { return false }` |
| `pkg/opencorpora/loader.go` | Duplicate methods `UnpackedFilePath` (public) and `unpackedFilePath` (private) with identical implementations | The private one removed, all calls go through the public one |
| `pkg/opencorpora/progress.go` | An unused type `Progress` — a namesake of another (real) `Progress` type in `pkg/morphology/importers/opencorpora`, confusing when reading | The file removed (0 references outside the package) |

## Dead code (not removed — a decision for the user)

Found more dead/unused code than is safe to silently remove as part of
"trivial fixes along the way" — with a rationale for each case below.

### The `internal/intern` and `internal/stringsx` packages — never imported at all

```
$ grep -rln "gomorphy/internal/stringsx\|gomorphy/internal/intern" --include="*.go" .
internal/intern/table.go
internal/intern/table_test.go
internal/stringsx/arena_test.go
```

No package outside themselves (and their own tests) imports them — 410
lines (128+150+59+73) of code left over from the removed
CSR-trie/exact-hash implementation (stages 0-10). `internal/xmlscan`
and `internal/mmapx`, from the same "reusable" list (see
`docs/en/todo.md`'s header), are actually used; `intern`/`stringsx`
aren't: the DAWG is self-contained and doesn't need a separate
interning table. A candidate for removal in full.

### `BlockDAWG`/`Block`/`NewBlockDAWG`/`Flatten` — `pkg/morphology/internal/dawg.go:351-403`

```
$ grep -rn "BlockDAWG\|NewBlockDAWG\|\.Flatten(" --include="*.go" .
pkg/morphology/internal/dawg.go   (definitions only, 0 call sites)
```

~53 lines, 0% test coverage, 0 usages. The comment ("a blocked DAWG
variant for faster compilation") points to an abandoned early approach
to the same problem eventually solved a different way by the free-list
allocator (see `docs/implementation/dawg-freelist-optimization.md`). A
candidate for removal.

### `pkg/common.ErrPath` — `pkg/common/utils.go:11`

Declared, never used anywhere (`grep -rn "common.ErrPath"` finds only
the definition).

### `cmd/gomorphy_build/main.go:30` — `const programVersion = "0.1.0"`

The seed this task started from (see `docs/en/todo.md`) — confirming:
never used anywhere. Now that `pkg/morphology.Version` exists (see
`docs/implementation/info-section.md`), it makes sense to either
remove this constant or have it printed by
`gomorphy_build -version`/`--version` (there's no such flag right now).

## Other findings by category

### Bugs / edge cases (not fixed — need a decision on approach)

- **`internal/mmapx/mmap.go`** (the whole file) — uses
  `syscall.Mmap`/`syscall.MAP_PRIVATE` directly, with no build tag.
  This breaks `go build ./...` on Windows entirely (transitively, the
  whole binary, since `pkg/morphology.Open` depends on mmapx). If
  Windows isn't a supported target, this should be documented
  explicitly (`//go:build !windows` + a clear error/alternative
  implementation, or explicitly stating "Unix only" in the README).
- **`internal/xmlscan/scanner.go:fill()` (124-145) + `readTag()`
  (173-204)** — if a single tag/attribute value is longer than the
  buffer (`NewBufferSize` allows a buffer as small as 16 bytes),
  `fill()` can't read more data (`io.ReadFull` on an empty remaining
  buffer returns `0, nil`), and `readTag()` loops forever. Not
  reproducible on the real `dict.xml` (its attributes are short), but
  it's a hang with no diagnostic for a small buffer or pathological input.
- **`internal/xmlscan/scanner.go:Scan()` (95-108)** — the
  `hasPrefix("<?")`/`hasPrefix("<!--")`/`hasPrefix("<!")` checks only
  look at already-buffered bytes, without requesting more data. If a
  buffer read boundary lands right after `<!` (before `--`), a comment
  `<!-- x > y -->` is mistakenly handled as `skipUntil(">")` instead of
  `skipUntil("-->")`, which can cut it off at the first `>` inside the
  comment's text. A narrow case depending on buffer alignment; there
  are no comments in `dict.xml` (not reproduced on real data).
- **`internal/xmlscan/attributes.go:matchEntity` (98-106)** — only
  decodes 5 named XML entities (`&amp; &lt; &gt; &quot; &apos;`);
  numeric ones (`&#39;`, `&#x27;`) aren't supported — they're left in
  the text as-is. There are no numeric entities in the current
  `dict.xml` (`grep -c "&#"` -> 0), but this is a limitation for future sources.
- **`pkg/morphology/internal/dawg.go:splitDAWG` (90-121)** — the
  zero-copy path (`unsafe.Slice` over the mmap bytes) doesn't account
  for byte order: the data is written LittleEndian, but the alias
  assumes the host's native order. On a big-endian platform (Go still
  supports some, e.g. s390x), this would silently produce wrong
  values, not an error. Warrants an explicit comment/build constraint.
- **`pkg/morphology/internal/tagset.go:Add` (16-27)** and
  **`pkg/morphology/importers/opencorpora/import.go:127,140`** (the
  suffix id and paradigm id) — an id is assigned as
  `uint16(len(...))` with no overflow check. Past 65,535 unique
  tags/suffixes/paradigms, ids silently collide, with no error.
  **Fixed during follow-up work on this report** (an explicit check,
  `TagSet.Add` now returns an `error`) — and the check immediately
  triggered on the real `dict.xml` for suffixes (65,835 > 65,536), see
  the critical finding at the top of this file. The "comfortable
  margin today" estimate below was wrong for suffixes — it wasn't
  measured against the full `dict.xml`; for tags (3,437) and paradigms
  (16,939), there really is a margin.
- **`pkg/morphology/parse.go:productive` (223-233)** — the
  "non-productive grammeme" check uses `strings.Contains(tag, g)` on a
  comma-joined tag string, rather than splitting on `,` and comparing
  tokens exactly. Works for the current set of 8 short codes, but it's
  a substring match on an unsplit string — adding a grammeme whose
  name is a substring of another's could cause false positives.
  Simple, cheap to fix (`strings.Split(tag, ",")` + a set membership check).
- **`pkg/morphology/internal/format.go:SaveContainer` (142-199)** —
  writes directly into the target `path` via `os.Create`; if the write
  is interrupted (disk full, process killed), a broken, half-written
  `.dat` is left at the target location. `Open()` will reject it by
  checksum, so there's no silent corruption on read, but a corrupted
  file sitting where a working one is expected is bad UX. The standard
  fix — write to a temp file alongside it and atomically rename
  (`os.Rename`) on success.
- **The CLI's `-o` after a subcommand** (already noted earlier in the
  session, `docs/implementation/stage-17-optimize.md`) — `gomorphy_build
  compile -o path` silently ignores `-o` (the `flag` package stops
  parsing at the first non-flag argument). Notably: **`cmd/gomorphy/main.go`
  already solves exactly this problem** for `import`
  (`findOutFlag`/`stripOutFlag`, lines 136-161) — the pattern just
  wasn't applied consistently across both binaries. Worth either
  factoring out a shared helper or applying the same trick in `cmd/gomorphy_build`.

### Test coverage

Overall: 65.8%. By package (`go test ./... -cover`):

| Package | Coverage | Comment |
|---|---|---|
| `cmd/gomorphy`, `cmd/gomorphy_build` | 0% | Structural: logic mixed with `os.Exit`, see below |
| `internal/mmapx` | 0% | Easy to test (create a temp file, Open/Bytes/Len/Close + error paths) — there just aren't any tests |
| `pkg/common` | 0% | `MakeDomainDataPath`/`DomainFilePath` — no tests |
| `pkg/opencorpora` | 57.3% | Network paths (`DownloadUpdate` 0%) aren't tested — `http.Get`/`http.Head` aren't injectable, nowhere to test without a real network |
| `pkg/morphology/internal` | 79.9% | `BuildDAWGWithValuesProgress` 0% (the real production path!), `BlockDAWG`/`Flatten` 0% (dead code), `HasPayloadChild` 0% |
| `pkg/morphology` | 87.9% | `parse.go`: `paradigmTag` 60%, `paradigm`/`paradigmAffix`/`productive` 66.7% — edge cases (an id not found, `TagSet==nil`) aren't covered |
| `pkg/morphology/importers/opencorpora` | 91.2% | See the critical finding — there's coverage, but not of the right thing (no check on tag correctness) |

**The structural reason for zero CLI coverage**: both `main.go` files
call `os.Exit` directly from logic functions (`findDictXML`,
`runImport`, `initLogging`, every `runXxx` in `cmd/gomorphy_build`),
not only from `main()`. This isn't just a style issue — it physically
prevents writing a unit test without terminating the test process.
Recommendation: logic returns an `error`, `os.Exit` only happens in
`main()`/top-level wrappers.

**`BuildDAWGWithValuesProgress` at 0%, separate from `BuildDAWG`/
`BuildDAWGWithValues`** (which are well covered) — important: this is
the ONLY path that actually builds a DAWG in production (called from
`importers/opencorpora`), yet tests only cover its progress-less twin.
Not a bug by itself (the shared logic is factored into `dawgBuilder`),
but if `BuildDAWGWithValuesProgress` ever diverges from
`buildDAWGWithPayload`, the tests wouldn't notice — see the next section.

### Duplication / structure (candidates for merging functions)

- **`pkg/morphology/internal/dawgbuild.go:buildDAWGWithPayload` (59-92)**
  and **`dawgbuild_progress.go:BuildDAWGWithValuesProgress` (12-72)** —
  the key-insertion loop into `dawgBuilder` (computing the common
  prefix, `closeSuffix`, `appendByte`, marking a leaf) is duplicated
  almost verbatim between the two functions; the only difference is
  the presence of a progress callback. As noted above, the real
  (production) path is only covered by tests through the
  un-duplicated (progress-less) copy. Worth factoring the shared
  insertion loop into one method called from both entry points.
- **`cmd/gomorphy/main.go:runImport` (163-201)** — the `"pymorphy2"`
  and `"opencorpora"` branches are structurally identical
  (open/compile -> error -> SaveTo -> error -> print), differing only
  in which importer is called. Could be collapsed into a shared
  `importAndSave(d *morphology.Dictionary, err error, out string)` function.
- **`pkg/opencorpora/loader.go` vs. `cmd/gomorphy_build/main.go`** —
  both explicitly call `os.Stat`+handle `ErrNotExist` separately in
  several places; not critical, but the pattern could be one shared
  helper function in `pkg/common`.

### Large functions — recommendations for splitting them up

| Function | Lines | Recommendation |
|---|---|---|
| `pkg/morphology/internal/dawgbuild.go:compileImpl` | 83 | Factor the local state (`dic`, `alloc`, `link`, progress counters) into a separate type with methods; make `dfs` one of its methods. Would make it easier to test placement independently of guide building. |
| `cmd/gomorphy_build/main.go:compileAndSave` | ~90 | The progress-printer goroutine (a ticker, rate/ETA calculation, formatting) is a self-contained piece of logic, testable independently of the CLI. Factor it into its own type (e.g. `progressPrinter` with an `Update(processed, total)` method), removing it from `main`. |
| `pkg/morphology/internal/format.go:SaveContainer` | 61 | The natural phases (computing the section layout -> writing the file -> the checksum) could be split into 2-3 named functions — would make adding atomic writes easier (see the finding above). |
| `pkg/morphology/parse.go:predict` | 57 | A triple-nested loop (prefixes x suffix splits x items x values). Factor the inner loop's body into `predictForPrefix(...)` for testing the per-prefix prediction logic independently. |

### Performance (not a bug, but worth knowing)

- **`pkg/morphology/fuzzy.go:maxWordRunes` (209-240)** — a full
  traversal of the DAWG graph (memoized by node, so not exponential,
  but still O(nodes)) runs from scratch **on every call** to
  `FuzzyTop`, instead of being cached on the dictionary. For large
  dictionaries — a noticeable constant cost per call; especially
  relevant given Stage 19's planned batch mode (`fuzzy`/`top` over
  many words from stdin) — there, this would be recomputed for every
  word in the batch. Worth caching (e.g. a `sync.Once` on
  `internal.Dictionary`, or a lazily computed field).

### Comments

Generally good quality where the logic is non-trivial (`fuzzy.go`,
`dawgbuild_freelist.go` — the WHY is explained, not just the WHAT).
Gaps noted:

- `pkg/morphology/internal/dawg.go:offset()` (127-129) — bit magic
  (`(n >> 10) << ((n & extensionBit) >> 6)`) with no comment at its
  definition; the explanation of the same trick only exists next to
  `encodable` in a different file (`dawgbuild.go`). Worth a short
  comment right here too.
- Many exported identifiers with no godoc comments: `pkg/common`
  (`GetDataPath`, `DomainDataPath`, `MakeDomainDataPath`,
  `DomainFilePath`), `internal/stringsx.Arena` and most of its
  methods, `pkg/opencorpora.Error`.
- `pkg/morphology/importers/opencorpora/import.go:197` — the comment
  on the `lGrams` field describes behavior (fallback lemma grammemes
  for a form with no `<g>` of its own) that **isn't implemented** (the
  field is never used anywhere) — misleading. Either implement it, or
  remove the field and the comment (closely tied to the critical
  finding above).

### Other / process

- There's no `.golangci.yml` in the repository — the installed
  `golangci-lint` (2.12.2, built for go1.26) also can't run on a
  module with `go 1.27.1` in `go.mod` (a toolchain version mismatch).
  Couldn't run the automated linter as part of this review — either
  update the linter binary or pin a config with a toolchain-compatible
  version, as part of Stage 18.
- `pkg/opencorpora/const.go:RemoteURL` — `http://opencorpora.org/...`
  (not HTTPS). Downloading the dictionary over an unencrypted,
  unauthenticated channel — if opencorpora.org has an HTTPS mirror,
  worth switching.
- `pkg/opencorpora/errors.go:Error` — an exported variable named
  `Error` with no godoc comment; a vague name for a specific package's
  sentinel error (usually `ErrXxx`).

### Documentation: residual `pkg/dictionary` -> `pkg/morphology` drift

`installation.md`/`cli.md`/`Makefile` have already been fixed (see
commit `04a957d`) — they were actively broken (a nonexistent module
path, a nonexistent binary). The same family of references remains in
6 more places, with varying degrees of staleness:

**Actively misleading (worth fixing first):**
- `docs/library.md:6` — `import ".../pkg/dictionary"` in the code
  example in the main library-usage guide — the path doesn't exist.
- `docs/todo.md:145` (Stage 19's plan) — "`pkg/dictionary` (the
  current facade...)" — for the future `import_tsv.go` implementation,
  names a current package that doesn't exist; should be `pkg/morphology`.
- `docs/implementation.md:108` — the repository-layout diagram for the
  "new implementation" (i.e. the current state) shows
  `cmd/opencorpora_update`, which doesn't exist — the actual name is
  `cmd/gomorphy_build`.

**Low priority (correctly historical narrative, or a case where the
example itself is secondary to the argument):**
- `docs/mcp.md:12,83` — illustrating the "why no MCP" decision, the
  argument doesn't depend on the exact package name.
- `docs/unimorph.md:201` — a reference to the Builder API in the
  analysis of a future source (Stage 16), doesn't block reading it.
- `docs/requirements.md:166,169` — the document deliberately describes
  the state **before** the redesign and the migration plan
  ("Replaced") — historically correct as-is.
- `docs/implementation.md:18,44,52` — the "current implementation
  (stages 0-10)" section is correctly historical; `docs/todo.md:34` —
  likewise, a table stub about the original implementation.

Not fixed — a separate task, per the user's earlier decision.

## What's next

As agreed: this is a report, decisions on each finding are separate.
The one thing worth explicitly calling out is the critical OpenCorpora
tags finding: it affects data correctness, not just code quality, and
ideally should be closed before the 1.0.0 release (or the release
should be explicitly documented as shipping with this known issue).
