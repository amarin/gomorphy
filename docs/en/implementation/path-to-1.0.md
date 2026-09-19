# Path to version 1.0.0

The release checklist fixed on 2026-09-14 and extended on 2026-09-15/16,
kept here as a historical record now that every item is done. See
[todo.md](../todo.md) for what's tracked as still open (post-1.0.0
backlog); at the time of writing, nothing blocks the 1.0.0 release
itself.

1. **Format: groundwork for extensible compression + info section** —
   DONE.
2. **Pre-1.0.0 code review: findings triage + both critical bugs**
   (suffix overflow via sharding, corrupted OpenCorpora wordform tags
   including the paradigm dedup bug) — DONE, see
   [code-review-pre-1.0-triage.md](code-review-pre-1.0-triage.md).
3. **Fix for the root-in-suffix issue with the comparative degree**
   (`Cmp2`/"по-") + a rune-safe `lcp()` — DONE, see
   [0003-comparative-paradigms-not-merging.md](../research/0003-comparative-paradigms-not-merging.md),
   [2026-09-15-comparative-prefix-split-design.md](../superpowers/specs/2026-09-15-comparative-prefix-split-design.md),
   [2026-09-15-comparative-prefix-split.md](../superpowers/plans/2026-09-15-comparative-prefix-split.md).
   Real effect: shard 0's paradigms shrank from 17,934 to 3,245, the
   dictionary now fits in 1 shard instead of 2, and the share of
   invalid UTF-8 suffixes dropped from 61.7% to 0%.
4. **Support for several dictionaries open at once (multi-dict)** —
   DONE 2026-09-16, see [multi-dict.md](multi-dict.md).
5. **Production rollout of the dense 1-byte DAWG alphabet** — DONE, for
   pymorphy2 on 2026-09-16, generalized to OpenCorpora on 2026-09-19,
   see [pymorphy2-dense-alphabet.md](pymorphy2-dense-alphabet.md)
   (its own backlog — `.dat` serialization, `fuzzy.go`,
   Prediction/Probability, the 2-byte read path, the CLI default — is
   also fully closed, see that document's "Backlog" section).
6. **CLI grooming + redesign** — DONE 2026-09-16 — a single cobra-based
   `gomorphy` binary (`cmd/gomorphy_build` removed), commands
   `lookup`/`lemmas`/`fuzzy`/`top`/`cli`/`download`/`unpack`/`build`/`update`/
   `version`/`merge`(stub)/`split`(stub), `-d/--dictionary` always
   goes through `MultiDictionary`, `$GOMORPHY_DICTIONARY`, `-v/-l`
   logging. See
   [2026-09-16-cli-redesign-design.md](../superpowers/specs/2026-09-16-cli-redesign-design.md),
   [2026-09-16-cli-redesign.md](../superpowers/plans/2026-09-16-cli-redesign.md), and
   [pymorphy-source-and-cli.md](pymorphy-source-and-cli.md).
7. **Stage 18 (documentation, tests, metrics)** — DONE 2026-09-17, see
   [stage-18-finalize.md](stage-18-finalize.md). The CLI part was
   closed earlier, in item 6. The agent skill and `examples/` were
   deliberately split out of this stage, see "Dictionary usage skill +
   examples" in [todo.md](../todo.md).
8. **Release 1.0.0** — the remaining step; not done as of this writing.
