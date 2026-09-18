# Feasibility of exporting dictionaries into the pymorphy2 and OpenCorpora formats

**Date:** 2026-09-17
**Status:** analysis done (by reading code, no experiments); no scope decisions were made
**Analyzed by:** Aleksey Marin (a session with Claude Code)
**Related documents:** [../todo.md](../todo.md) (the "Universal tag mapping between dictionaries" section), [../implementation/stage-15-import-opencorpora.md](../implementation/stage-15-import-opencorpora.md), [../implementation/stage-12-import-pymorphy2.md](../implementation/stage-12-import-pymorphy2.md), [0005-pymorphy2-full-dawg-walk-cost.md](0005-pymorphy2-full-dawg-walk-cost.md)

## Context

The task: assess how feasible it is to export gomorphy's internal
dictionary (`internal.Dictionary`) back into the original formats:
pymorphy2's binary format (`words.dawg` + `paradigms.array` +
`suffixes.json` + `paradigm-prefixes.json` +
`gramtab-opencorpora-int.json`) and OpenCorpora's XML format
(`dict.xml`). At the time of the analysis the codebase only has import
in both directions (`pkg/morphology/importers/{pymorphy2,opencorpora}`)
— there's no export path in any form, and the task isn't mentioned in
any `docs/implementation/` file or in `docs/todo.md`.

## Current code snapshot (verified by reading, 2026-09-17)

The low-level serialization primitives needed for export **already
exist and are already round-trip tested** — export won't need new
binary infrastructure:

1. `internal.DAWG.Bytes()` (`pkg/morphology/internal/dawg.go:61`) —
   serializes a DAWG back into the same binary wire format
   (dictionary+guide) that `ReadDAWG` reads; covered by
   `TestBuildDAWGRoundtrip`. The format matches pymorphy2's
   `words.dawg` (`ReadDAWG` already reads real pymorphy2 dictionaries,
   see [0005-pymorphy2-full-dawg-walk-cost.md](0005-pymorphy2-full-dawg-walk-cost.md)).
2. `internal.Paradigm.Data()` (`pkg/morphology/internal/paradigm.go:57`)
   — the inverse conversion of a paradigm to `[]uint16`, symmetric with
   `NewParadigmFromData`, which reads pymorphy2's `paradigms.array`
   (`importers/pymorphy2/import.go:167-195`).
3. `Suffixes`/`Prefixes`/`TagSet.Tags` — ordinary `[]string`, trivially
   serialized to JSON (the `suffixes.json`/`paradigm-prefixes.json`/
   `gramtab-opencorpora-int.json` format is a flat JSON array of
   strings).

In short: for a dictionary whose `d.Words`/`d.Paradigms`/`d.Suffixes`/
`d.Prefixes` are shaped like pymorphy2's (a single shard, no dense
alphabet), exporting to the pymorphy2 format is mostly wiring together
already-ready pieces, not new engineering.

Three independent obstacles make exporting an **arbitrary** dictionary
(chiefly one imported from OpenCorpora) to the pymorphy2 format
non-trivial:

1. **The tag format differs syntactically.** The OpenCorpora importer
   assembles a tag as comma-separated grammemes with no spaces
   (`importers/opencorpora/import.go`, `gramm` in `OnFormEnd`);
   pymorphy2 (`gramtab-opencorpora-int.json`) uses a space before the
   form-specific part: `"NOUN,anim,masc sing,nomn"`. Same meaning,
   different syntax — already recorded in `todo.md` ("Universal tag
   mapping between dictionaries — NOT DESIGNED") as a fact, but with
   no resolution.
2. **Suffix sharding has no inverse path.** The sharding mechanism
   (`importers/opencorpora/shard.go`) was introduced specifically
   because of a `uint16` overflow in the per-dictionary suffix count
   (see `implementation/code-review-pre-1.0-triage.md`). Merging
   several shards back into a single pymorphy2 `suffixes.json` risks
   recreating the very overflow that forced sharding in the first
   place.
3. **The dense alphabet.** Dictionaries rebuilt via `RecompileDense`
   currently can't even have their own `SaveTo` serialize them
   (`Alphabet` non-nil -> an error, see the "pymorphy2 recompiler for
   the dense alphabet" section in `todo.md`). Exporting such a
   dictionary into someone else's format is either forbidden for the
   same reason, or requires first converting it back into a raw UTF-8
   DAWG.

Exporting to OpenCorpora XML runs into a separate, more fundamental
problem — **information loss during import**, not just syntax:

4. **A form's tag, after import, is an inseparable merge.** `OnFormEnd`
   combines the `<l>` (lemma) grammemes and the specific `<f>` (form)
   grammemes into a single tag string per form (see the "Critical bug:
   corrupted OpenCorpora wordform tags" fix in `todo.md`). After
   import, recovering which part of the tag belonged to the lemma and
   which to the specific form is impossible without heuristics: this
   is deliberate import architecture (one flat tag per form), not a
   side effect of a bug.
5. **Schema metadata isn't stored.** `internal.Dictionary` doesn't
   carry OpenCorpora's numeric lemma id, `<links>`, comments, or
   revisions — exporting to a valid `dict.xml` would require either
   generating fake ids, or accepting that the result isn't the same
   dictionary in another file, but a structurally similar surrogate.

## Conclusions

| Direction | Assessment | Main obstacle |
|---|---|---|
| pymorphy2 export, a dictionary of **pymorphy2 origin** (round-trip) | low risk, all the binary infrastructure already exists | none |
| pymorphy2 export, a dictionary of **any origin** (chiefly OpenCorpora) | blocked | tag mapping (see below) + sharding + the dense alphabet |
| OpenCorpora XML export | not recommended as a reversible operation | irreversible lemma+form tag merge on import; missing schema metadata |

Recommended order of work, if export is ever implemented:
1. pymorphy2 export for dictionaries of pymorphy2 origin — a
   standalone task, blocked by nothing in the list above.
2. Universal export (any dictionary -> pymorphy2) — blocked by the
   lack of a tag mapping; it isn't worth starting before that mapping
   is designed: otherwise export would silently produce
   incorrect/unrecognizable pymorphy2 tags on output (the same class of
   risk that's already bitten the project twice —
   `paradigmKeyHash`, grammeme accumulation in `OnFormEnd`).
3. OpenCorpora XML export — if ever needed, specify it explicitly as
   "export to OpenCorpora-style XML for human/third-party-tool
   reading," not as a round-trip; items 4-5 above make an exact
   round-trip unimplementable without changing what import currently
   preserves.

A direct consequence: mapping tags between dictionary formats is a
shared dependency both for universal export and for multi-dict
(`morphology.MultiDictionary`, where `Reading.Tag` is currently handed
out as-is, with no normalization across dictionaries from different
sources) — it's worth designing as a standalone task, not only "for
the sake of export."

## Sources

No third-party sources were used — the analysis is based only on
reading this repository's code (paths and lines given above) and on
the existing `docs/todo.md`, `docs/implementation/`, `docs/research/`
documents.
