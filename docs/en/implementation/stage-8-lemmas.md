# Stage 8. FT5 lemma lookup — DONE

## Stage contents

Increment: `Lemmas(word)` -> the base form + readings.

- Implemented on top of postings; values via arenas. The full grammatical
  characteristic is assembled from the lemma + the form:
  `Snapshot.LemmaAncodes` (section `lemmas.ancodes`, format v2),
  `Wordform.Grammemes`/`Ancode` — the combined reading, `LemmaRef` now
  carries the lemma's base grammemes.
- The output semantics were fixed by cross-checking against a reference:
  - `AddLemma` creates an anchor pair "citation -> bare base ancode" for
    resolving base forms via `Lemmas()`; `Lookup` doesn't return such
    pairs (the citationPair filter) — they aren't a dictionary-entry
    reading;
  - a pair shared by several lemmas is expanded in `Lookup` into a
    separate reading per lemma — homonymous readings stay distinguishable
    even when their tag sets coincide ("соляное": 2 pairs / 4 readings).

## Verification (done)

- Unit tests: homonym words ("стекла", "пила") -> >=2 lemmas; a unique
  word -> 1; the merged reading of "кота" = `NOUN,anim,masc,sing,gent`.
- Cross-check against a reference: 100 random wordforms out of 3,065,312
  against an independent two-pass parse of `dict.xml`
  (`pkg/dictionary/reference_integration_test.go`): dense lemma ids, full
  grammemes, and reading counts matched exactly.
