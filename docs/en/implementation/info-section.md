# The info section: dictionary build diagnostics — DONE

> Moved from `docs/todo.md` during the documentation cleanup (2026-09-14).
> Not part of Stage 17's original plan — added as a result of the
> discussion about format groundwork for compression (see
> [stage-17-optimize.md](stage-17-optimize.md)).

## Motivation

A new optional `info` section in the GMOR format — not needed for
`Parse`/`Lemma`/`Fuzzy` to work, but it answers the question "when and
with what was this `.dat` built", which came up explicitly during the
DAWG minimization fix (see
[dawg-minimization-fix.md](dawg-minimization-fix.md)): `Version` (the
format) doesn't change for years, but build behavior can, and there was
previously nothing to tell the two apart after the fact.

## What's implemented

`pkg/morphology/internal/buildinfo.go`, `pkg/morphology/buildinfo.go`,
`pkg/morphology/version.go`:

- `BuildInfo{BuiltAt, LibraryVersion, Source, SourceVersion, Author,
  Description, SourceURL}` — a JSON section, every field optional, the
  section as a whole optional (old files and dictionaries built by hand
  through the Builder API without `SaveTo` still open as before —
  `Dictionary.Info()` returns `nil`).
- `BuiltAt`/`LibraryVersion` are set by `SaveTo` itself on every save
  (without mutating the source `Dictionary` — it's immutable); any
  values an importer set for these two fields beforehand are overwritten.
- `Source` is filled in by the importers: `"opencorpora"`
  (`importers/opencorpora`), `"pymorphy2"` (`importers/pymorphy2`).
- `LibraryVersion` comes from the new `pkg/morphology.Version = "0.1.0"`
  (the first time this constant exists in the repository — it was never
  needed anywhere before). It's a manually maintained string; switching
  to a value embedded at build time (ldflags/VCS info) is a separate
  change during the CLI grooming (see `docs/en/todo.md`, Stage 18).
- `Dictionary.Info() *BuildInfo` — a public accessor for reading it.

## Deliberately not done now

Considered during design; some items deliberately deferred, some rejected:

- **`SourceVersion` isn't filled in for OpenCorpora**: `dict.xml` has
  `<dictionary version="0.92" revision="417257">`, but `internal/xmlscan`
  currently doesn't surface the root tag's attributes
  (`dispatch.go`: `case t.is("dictionary"): s.section = sectOther` — the
  tag is only recognized as a section marker, its attributes aren't
  extracted). Needed: a new `Handler` method (e.g.
  `OnDictionaryMeta(version, revision []byte)`) + a branch in
  `dispatch()` + reading it in `opencorpora.ImportFromXML`. A small but
  separate task — as importers "learn" to parse their source's version
  (the same applies to future `pymorphy2`/`unimorph` sources).
- **A stable content hash** (independent of the build date/library
  version, to compare two differently-built `.dat` files for identical
  linguistic data) — considered and rejected: decided the
  (`BuiltAt`, `SourceVersion`) pair is enough.
- **Per-section checksums** — considered and rejected: the whole file
  is already checked with one xxh3 on `Open`; per-section checksums add
  nothing for this format.
- **Tooling to check for/fetch a fresher dictionary version** via
  `SourceURL` — deliberately only a field-level placeholder for now.
  The forward-looking scenario: `SourceURL` is just a link for
  manual/agent-driven download today; in the future, its own small
  standard for a version listing (similar to yum/pip index files: JSON
  enumerating the dictionary's available versions by URL), but that's a
  separate feature with its own contract (what counts as a "new
  version," how to compare), not just a format field — to be designed
  separately once a concrete public resource with dictionary versions
  exists.
- No CLI flag fills in `Author`/`Description`/`SourceURL` yet — that's
  part of the CLI grooming before Stage 18.

## A finding for a future code review

`cmd/gomorphy_build/main.go` already has `const programVersion =
"0.1.0"`, but never uses it — dead code, found in passing while adding
`LibraryVersion`. Left untouched (out of scope for this change) — see
`docs/en/todo.md`, the "Pre-1.0.0 code review" section.
