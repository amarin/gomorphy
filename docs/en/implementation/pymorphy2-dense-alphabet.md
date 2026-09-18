# A dense 1-byte DAWG alphabet: production rollout for pymorphy2's `words.dawg`

Discussed 2026-09-15/16, right after merging the `dawg-alphabet-harness`
branch into `master`. Variant A (a pymorphy2 recompiler), for
`words.dawg` only, was decided on and implemented on 2026-09-16.
Design:
[2026-09-16-pymorphy2-dense-recompile-design.md](../superpowers/specs/2026-09-16-pymorphy2-dense-recompile-design.md).
Plan: [2026-09-16-pymorphy2-dense-recompile.md](../superpowers/plans/2026-09-16-pymorphy2-dense-recompile.md)
(6 tasks, subagent-driven-development, a final review + 1 fix wave).

## What had already been established (measured, not in question)

- The dense 1-byte DAWG alphabet: -36.2% `words.dawg` size and a 3x
  faster build on real data with a payload, with no tradeoffs. The
  dense 2-byte alphabet: works correctly (after finding and fixing a
  bug with the `0x00` sentinel byte), but on the current Russian corpus
  (46 characters) it only gives ~2.4% — nearly useless for a single
  language. Details and numbers:
  [docs/research/0004-dawg-dense-alphabet-with-payload.md](../research/0004-dawg-dense-alphabet-with-payload.md).
- `Alphabet`/`IdentityAlphabet`/`DenseAlphabet` already existed in
  `pkg/morphology/internal/alphabet.go` (a harness) — but nothing was
  wired up to `import.go`/`open.go`/`save.go`/`parse.go`/`fuzzy.go`/
  the `.dat` format.

## Agreed design decisions for the production rollout

1. The alphabet table is shared across the whole dictionary (like
   `prefixes`), not per-shard.
2. CLI (`gomorphy`) — no new alphabet flags at all. Always: a dense
   1-byte alphabet, Russian order by default, extend mode (see item 4).
3. The library (Go API, embedding) — full flexibility: your own base
   alphabet (order, given directly as `[]rune` in code, not a file),
   choice of 1- or 2-byte width, choice of extend/strict mode.
4. Extending the alphabet is **append-only**: the base order fixes the
   first codes, new characters found while extending are appended **at
   the end**, with no resorting. Important: the current
   `NewDenseAlphabet` in `alphabet.go` sorts all the corpus's runes by
   code point before assigning codes — this is **incompatible** with
   append mode (resorting would shift already-issued codes) and would
   require reworking the table-building logic, not just adding an
   option — relevant when implementing item 5 of the backlog below.
5. The alphabet table is serialized into the `.dat` in full (a new
   section, similar to `prefixes`), so `Open()` can reconstruct the
   exact codec regardless of whether it's the default, custom, or
   extended on the fly. **Not implemented** — see "Remaining backlog" below.
6. Alphabet overflow (in the CLI — extend hitting the 254-character
   ceiling for 1-byte width) — an honest, detailed compile error linking
   to the documentation, not a silent failure or truncation.
7. The 2-byte alphabet stays an experimental, library-only feature —
   **not carried into the read path** (`Open`/`Parse`/`Lemma`/
   `fuzzy.go`) at all. A 2-byte dictionary can be built (`BuildDAWG` +
   `DenseAlphabet{width:2}`, as the harness already does), but
   `Open()`/`Parse()` won't understand it — if a real consumer shows
   up, that's a separate future task.
8. Multilinguality — via parallel dictionaries (each its own file, its
   own alphabet), not one dictionary with a wide alphabet — see
   [multi-dict.md](multi-dict.md).

## The read-path complexity found (the main reason to pause before deciding)

Rolling this out isn't an isolated change to `import.go`/`open.go`/
`save.go`, but at least 3 interconnected subsystems that must use one
encoding in sync:

1. `words.dawg` — build/lookup/read.
2. `d.Suffixes`/`d.Prefixes` — computed from the **same**
   `stem`/`suffix`/`prefix` variables as the DAWG key
   (`import.go:150-234`), but these are independent paths. If the DAWG
   key is encoded while the tables are left raw, `TrimPrefix`/`TrimSuffix`
   in `parse.go:207-209` (reconstructing the base form) would break
   almost everywhere except forms with an empty prefix (where the bug
   would be masked by accident). The same class of silent data
   corruption that has already happened twice in this project
   (`paradigmKeyHash`, see
   [code-review-pre-1.0-triage.md](code-review-pre-1.0-triage.md)) —
   the implementation needs an explicit regression test for exactly
   this synchronization.
3. `fuzzy.go` — the entire fuzzy-search apparatus (rune-level
   Levenshtein, `utf8.DecodeRune`/`FullRune`, counting runes by
   continuation bytes) is built on the assumption that "a DAWG edge is
   part of UTF-8". Under fixed width it should **get simpler** (no need
   to fuss over multi-byte runes), but that's a rethink of the
   traversal structure, not a cosmetic change.
4. An unresolved open question: does the dense alphabet extend to
   `prediction-N.dawg`/`probability.dawg`? For OpenCorpora, `Prediction`
   currently isn't populated at all (probably not relevant);
   `Probability` wasn't checked.

## Discussed, but not adopted as the only path: a full pymorphy2 recompiler

A proposal — remove direct support for raw pymorphy2 `words.dawg`
files (at the time, `pkg/morphology/importers/pymorphy2/import.go`
simply aliased pymorphy2's ready-made binary artifacts as-is, with no
transformation at all) and instead rebuild pymorphy2 dictionaries
through the same pipeline as OpenCorpora.

- The effort estimate for the "read the whole `words.dawg`" step turned
  out to be inflated: a full traversal of a real pymorphy2 dictionary
  (3,064,708 wordform->(para,form) pairs) via the already-exported
  `DAWG.ForEachChild`+`DAWG.ValuesForIndex` took 570 ms in ~35 lines of
  code. Details:
  [docs/research/0005-pymorphy2-full-dawg-walk-cost.md](../research/0005-pymorphy2-full-dawg-walk-cost.md).
- The narrow variant A was chosen: a recompiler for `words.dawg` only
  (not `Prediction`/`Probability`, which remain unexplored).

## What's implemented (2026-09-16)

- `internal.DAWG.Walk(fn func(key string, values [][]byte))` — a
  general full-DAWG-traversal primitive (not tied to pymorphy2; it also
  turned out useful for multi-dict).
- `internal.Dictionary.Alphabet` — a new field (nil by default = raw
  UTF-8, existing dictionaries' behavior unchanged).
  `internal.DAWG.SimilarItems` gained a third parameter, `alphabet`.
- `pymorphy2.RecompileDense(dir)` — imports the dictionary via
  `ImportFromDir`, then rebuilds only `Words[0]` under a dense 1-byte
  alphabet built from the dictionary's own wordforms;
  `Paradigms`/`Suffixes`/`Prefixes`/`Prediction`/`Probability`/`TagSet`
  are copied unchanged.
- `morphology.OpenPyMorphyDense(dir)` — a public entry point (modeled
  on `OpenPyMorphy`). `Parse()` on the result gives the same readings
  as `OpenPyMorphy` — verified on a fixture roundtrip and on a real
  dictionary (3,064,708 words, 25.7s to rebuild+compare).
- **Explicit guards against silent data corruption** (found by the
  final review of the whole branch, the same risk class as
  `paradigmKeyHash`): `SaveTo` now returns an error for a dictionary
  with a non-empty `Alphabet` (without this, saving+reopening silently
  lost the codec and produced wrong readings — reproduced by the review
  on real data: "кот" 1->2 readings, "все" 5->4). `Fuzzy`/`FuzzyTop`
  return `nil` for such a dictionary (without the guard, dense codes
  would accidentally decode as "valid" UTF-8 garbage instead of an error).

## Remaining backlog (doesn't block 1.0.0, separate future tasks)

Deliberately not done in the 2026-09-16 pass:

1. **Serializing `Alphabet` into the `.dat` and support in `Open()`** —
   item 5 of the agreed decisions above remains unimplemented; so
   `SaveTo` currently just refuses a dictionary with a dense alphabet,
   rather than saving it. Without this, `OpenPyMorphyDense` is useless
   for persistent storage — a rebuild is needed on every run.
2. **`fuzzy.go`** — still assumes the DAWG's bytes are UTF-8; disabled
   for dense dictionaries (returns `nil`), not rethought for fixed
   width (item 3 of the read-path complexity above).
3. **`Prediction`/`Probability` DAWGs** — not investigated (a different
   key format: word suffixes and `"word:tag"` with ASCII grammemes), item 4 above.
4. **The 2-byte alphabet in the read path** — deliberately not carried
   in (`Open`/`Parse`/`Lemma`/`fuzzy.go` don't understand it), see item
   7 above; only relevant if a real multilingual consumer shows up for
   whom multi-dict (see [multi-dict.md](multi-dict.md)) somehow doesn't fit.
5. **CLI** (`gomorphy`) is untouched — this was a Go-API-only increment;
   there's no way to get a dense dictionary from the CLI without writing code.
