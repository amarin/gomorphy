# Universal tag mapping between dictionaries (`pkg/morphology/tagmap`)

Implemented 2026-09-17 (the native -> universal direction). The full
architecture and decision rationale is in the spec:
[2026-09-17-tag-mapping-design.md](../superpowers/specs/2026-09-17-tag-mapping-design.md).
The export feasibility assessment that led to this task:
[docs/research/0008-dictionary-export-feasibility.md](../research/0008-dictionary-export-feasibility.md).
Implementation plan (6 tasks, subagent-driven-development, 1 task-level
fix round + 1 final-review fix wave):
[2026-09-17-tag-mapping.md](../superpowers/plans/2026-09-17-tag-mapping.md).

## Context

Raised on 2026-09-16 while designing multi-dict (see
[multi-dict.md](multi-dict.md)): different dictionaries already
describe grammemes differently, even though it's formally the same
set. Example: pymorphy2 (`gramtab-opencorpora-int.json`) uses
`"NOUN,anim,masc sing,nomn"` (a space before the inflecting part),
while the OpenCorpora importer always joins a tag with commas and no
spaces — the same meaning, different strings. `TagSet.Name`
(`"opencorpora"` vs. `"opencorpora-int"`) recorded this difference in
convention as metadata, but did nothing to eliminate it.

## Decisions made

1. The mapping table is a separate package, `pkg/morphology/tagmap`,
   not part of `TagSet` and not external configuration: `TagSet` stays
   a generic string interner, with no knowledge of tag syntax or semantics.
2. A universal tag is an independent **UniMorph** feature set (not
   tied to OpenCorpora or any other existing set), in a canonical
   dimension order (`tagmap.Dimension`) independent of the source
   tag's token order — this is what makes `Bundle.Features` from two
   different sources comparable via `slices.Equal`.
3. Opaque tags and uncovered grammemes — a single rule: kept as-is, in
   `Bundle.Unmapped`, no error. `tagmap.Map` returns `ok=false` only
   for an entirely unregistered `dictName` (a structural error in the
   calling code), never for a tag's content.
4. Two **independent** tables (`opencorpora`, `opencorpora-int`), not
   one shared table — deliberately, so as not to rely on the unverified
   assumption that the two sources' tokens are spelled identically
   (they actually do coincide right now, but this is checked
   separately for each source, not derived logically).
5. Direction — only `native -> universal` in this increment. `Unmap`
   (universal -> native, needed for export) is explicitly out of
   scope, a separate future task, see "What's left" below.
6. `MultiDictionary`/`Reading`/`LemmaRef` are unchanged. The calling
   code calls `tagmap.Map(dictName, reading.Tag)` itself whenever it
   needs a universal tag.

## What's implemented

- `tagmap.Bundle`/`tagmap.Feature`/`tagmap.Dimension` — the data model.
- `tokenizeOpenCorpora`/`openCorporaTable`,
  `tokenizeOpenCorporaInt`/`openCorporaIntTable` — tokenizers and
  mapping tables for the two sources; a representative subset of
  grammemes (part of speech, animacy, case, number, gender, tense,
  aspect, mood, voice, person) — not exhaustive coverage, growing as
  uncovered tokens are found.
- `tagmap.Map(dictName, tag string) (Bundle, bool)` — the sole public
  entry point.
- An integration test against real dictionaries (OpenCorpora `dict.xml`
  + a real pymorphy2 dictionary): the word "кот" (NOUN, nominative,
  singular) normalizes to an identical `Bundle.Features` from both
  tables, with no uncovered tokens.

## A known gap, found by the final review — CLOSED 2026-09-19

`dictName` (`TagSet.Name`) used to be **unreachable from
`pkg/morphology`'s public API**: `Dictionary` didn't export `TagSet`
or its name, and `BuildInfo.Source` isn't a substitute (for pymorphy2,
`Source == "pymorphy2"`, but `TagSet.Name == "opencorpora-int"`). A
real external consumer of `tagmap.Map` holding a
`*morphology.Dictionary` (especially one opened via `Open(path)`,
where it's unknown which importer built it) couldn't get the right
`dictName` without out-of-band knowledge.

Fixed with two small accessors mirroring the existing `Info`/`DictInfo`
pattern: `Dictionary.TagSetName() string` and
`MultiDictionary.DictTagSetName(i int) string` (nil-safe, "" when
absent/out of range). Typical real usage:
`tagmap.Map(multi.DictTagSetName(reading.Dict), reading.Tag)`. Verified
against real dictionaries — both sources report exactly the `dictName`
`tagmap.Map` already recognizes
(`TestDictionary_TagSetName`, `TestMultiDictionary_DictTagSetName`,
`pkg/morphology/tagsetname_test.go`).

## What's left (separate future tasks, not part of this increment)

1. **`Unmap` (universal -> native)** — needed to export dictionaries
   back into pymorphy2/OpenCorpora-compatible formats (see
   `0008-dictionary-export-feasibility.md`). Ambiguous by construction
   (one universal tag can correspond to several native-tag variants) —
   resolving the ambiguity needs to be decided against a concrete
   consumer (an exporter), not ahead of time. **Deliberately still not
   started** (2026-09-19 decision): building it without a concrete
   exporter to design against would mean guessing at the ambiguity
   resolution, with real rework risk once dictionary export actually
   starts (see `todo.md`, "Dictionary export").
2. **A table for the UniMorph importer (Stage 16)** — trivial (its
   bundle is already in the target schema), but not written since the
   importer itself doesn't exist yet.
3. **`Dimension.String()`** — for readable diagnostic messages when
   tags diverge between dictionaries (`Dimension` currently prints as
   a `uint8`).
