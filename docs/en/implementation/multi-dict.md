# Multi-dict: `morphology.MultiDictionary`

Implemented 2026-09-16. Spec: [2026-09-16-multi-dict-design.md](../superpowers/specs/2026-09-16-multi-dict-design.md).
Plan: [2026-09-16-multi-dict.md](../superpowers/plans/2026-09-16-multi-dict.md).

## Context

Raised on 2026-09-15 while discussing the production rollout of the
dense 1-byte DAWG alphabet
([docs/research/0004-dawg-dense-alphabet-with-payload.md](../research/0004-dawg-dense-alphabet-with-payload.md)):
for texts in different languages, each language could use its own
dictionary with its own alphabet, instead of one large dictionary with
a wide (2-byte) alphabet over a combined corpus. Checked: the library
did **not** support this at the time the task was raised — only
"technically nothing stops you from opening `Dictionary` N times by hand."

**Known context (verified by reading the code on 2026-09-15, before
implementation):**
- There was no registry/manager for several dictionaries.
  `pkg/morphology/open.go` (`Open`, `OpenPyMorphy`, `CompileFromXML*`) —
  every call gives an independent `*Dictionary` with its own mmap;
  nothing stopped you from opening several, but the library didn't link
  them together.
- `Reading` (`parse.go:14-22`) and `LemmaRef` (`lemma.go:4-9`) only
  carried `Shard int` — a shard index **within one** dictionary, not a
  dictionary/language identifier.
- The CLI accepted exactly one `-dict <path>`.
- The `Shard` field on `Reading`/`LemmaRef` was designed to be
  extensible with future multi-dict support specifically in mind (see
  [2026-09-14-suffix-sharding-design.md](../superpowers/specs/2026-09-14-suffix-sharding-design.md)) —
  architecturally anticipated, but not implemented.

A prerequisite was closed along the way: `BuildInfo.SourceVersion` used
to not be filled in by any importer — now both fill it in (OpenCorpora
from `dict.xml`'s root tag, pymorphy2 from the dictionary's own
`meta.json`).

## Decisions made

1. A dictionary's identifier is **not a string**, but `Dict int` on
   `Reading`/`LemmaRef` (analogous to `Shard`), an index in the order
   the dictionary was registered in `MultiDictionary`. The composite
   human-readable identifier (`Source`/`SourceVersion`) is looked up by
   the same index via `MultiDictionary.DictInfo(i)`, not duplicated
   onto every `Reading`.
2. The API is a separate wrapper type, `morphology.MultiDictionary`
   (`NewMultiDictionary(dicts ...*Dictionary)`, `Parse`, `Lemma`,
   `Close`, `DictInfo`, `Len`). The dictionaries themselves are opened
   as before (`Open`/`OpenPyMorphy`/...); the wrapper doesn't open
   anything itself.
3. Overlapping results — **everything, tagged**: `Parse`/`Lemma` return
   readings from every dictionary the word was found in, each tagged
   with `Dict`, in registration order. No priorities, dedup, or sorting
   across dictionaries — a deliberately simple, nothing-lost default; a
   smarter policy can be layered on top later without breaking the contract.
4. CLI — out of scope for this increment (Go API only), same as the
   dense alphabet for pymorphy2 (see
   [pymorphy2-dense-alphabet.md](pymorphy2-dense-alphabet.md)).
   `cmd/gomorphy` stays with a single `-dict`.
5. Relationship with the 2-byte DAWG alphabet — the split is fixed:
   multi-dict (implemented) is the default path for "the alphabet
   doesn't fit in 1 byte"; the 2-byte dense alphabet remains an
   experimental library-only feature with no read-path/CLI support —
   not solving the same problem twice.

Every open question raised when this task started was resolved and
implemented within the same session/spec — there's no separate "still
to do" for multi-dict itself. The known growth points (CLI flags, a
cross-dictionary priority policy) are explicitly out of scope, see above.
