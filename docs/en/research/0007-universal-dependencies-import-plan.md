# Universal Dependencies (universaldependencies.org) as a data source for gomorphy: applicability analysis and plan

**Date:** 2026-09-16
**Status:** exploratory research done (format, the composition of the Russian data,
licenses); the decision on which use case to pursue is an open question for the user
**Analyzed by:** Aleksey Marin (a session with an agent)
**Related documents:** [0006-unimorph-import-plan.md](0006-unimorph-import-plan.md)
(a reference for structure and for the "lemma = form 0" model),
[unimorph.md](../unimorph.md) (UniMorph's format/licenses — for comparison),
[todo.md](../todo.md) §"Stage 19" (a TSV engine for topical dictionaries,
opaque tags — a reusable mechanism) and §"Universal tag mapping
between dictionaries" (an open question that UD could give a ready-made answer to),
[implementation/stage-15-import-opencorpora.md](../implementation/stage-15-import-opencorpora.md)
(the reference importer)

## Context

The user asked for research into universaldependencies.org (UD) from
the angle of its data format, the composition of its Russian
dictionaries/corpora, and whether gomorphy could add downloading and
importing Russian-language data from that site. Unlike Stage 16
(UniMorph), there's no draft stage for UD — this is exploration from
scratch: understanding how UD's data differs from the already-
supported sources (OpenCorpora, the UniMorph plan) and whether an
importer is even worth building, versus "it's just a different format
for the same thing."

Research into the site and the GitHub repositories of the
`UniversalDependencies` organization was done live (see "Sources"
below); what follows is a summary relevant to the decision.

## 1. Data format: CoNLL-U

UD treebanks are distributed in **CoNLL-U** format — line-based TSV,
10 columns per word token: `ID FORM LEMMA UPOS XPOS FEATS HEAD DEPREL
DEPS MISC`. Rows are grouped into sentences (a blank line is the
separator), and each sentence is preceded by `# sent_id = ...` and
`# text = ...` comments.

Three fields are relevant to morphology:
- **LEMMA** — the wordform's lemma (as in OpenCorpora/UniMorph);
- **UPOS** — a universal part of speech, a fixed list of ~17 tags
  (NOUN, VERB, ADJ, ADP, ...), the same across all UD languages;
- **FEATS** — a list of grammatical features `Attr=Val`, `|`-separated,
  e.g. `Case=Nom|Number=Sing|Gender=Masc`; features are drawn, where
  possible, from UD's **universal cross-language inventory** (unlike
  XPOS, a language-specific tagset that's often just `_`/unused — in
  the Russian treebanks XPOS is mostly "not available" or "automatic,"
  with no meaningful data there).

**The key structural difference from OpenCorpora/UniMorph:** CoNLL-U
is an annotation of **connected text** (a corpus with a syntax tree),
not a lexicon. A row is a specific word occurrence in a specific
sentence, not a "lemma -> all forms of its paradigm" entry. The same
lemma appears in a treebank in exactly the forms that happened to
occur in the source texts — typically a small fraction of the full
paradigm (whereas for OpenCorpora/UniMorph, the unit of storage is the
whole lexeme's paradigm). This difference drives the entire analysis
below.

## 2. The composition of Russian data on UD

The `UniversalDependencies` organization on GitHub holds 5 repositories
for Russian (confirmed by pulling the repository list and reading each
README, as of the 2.18 release, May 2026):

| Treebank | Size | Genre | License | Notes |
|---|---|---|---|---|
| **UD_Russian-SynTagRus** | ~1.4M tokens (the largest; 66K+ sentences, manually annotated + 409K words added 2015-2020) | fiction, popular science, news 1960-2016 | **CC BY-NC-SA 4.0** ⚠️ | Source: the Russian National Corpus; the highest quality (manual annotation at every level) |
| **UD_Russian-GSD** | ~16K tokens | Wikipedia | CC BY-SA 4.0 | Annotated by Google; originally NC, the restriction was lifted in 2019 |
| **UD_Russian-Taiga** | ~64K tokens (train+dev+test, v2.9) | blogs, social media, reviews, poetry, news, wiki | CC BY-SA 4.0 | Auto-annotated with UDPipe + manual review |
| **UD_Russian-PUD** | 1000 sentences (a parallel corpus) | news/wiki | CC BY-SA 3.0 | Part of a multilingual parallel set, the entire treebank is a test split |
| **UD_Russian-Poetry** | small (not precisely measured) | Russian poetry, 19th-21st centuries | CC BY-SA 4.0 | Based on the National Corpus's Poetry sub-corpus plus additional versification markup in MISC |

Total for Russian: 3,455K word occurrences (an aggregate from the UD
homepage, matching the order of magnitude of the table above).

**License risk:** the largest and highest-quality source (SynTagRus)
is **CC BY-NC-SA**, i.e. incompatible with commercial use of a
derived dictionary. The other four are CC BY-SA (more permissive, but
require attribution and share-alike). Mixing sources into one compiled
dictionary without tracking provenance row-by-row would legally bind
the resulting dictionary to the strictest license among the sources
included — the same principle already recorded for UniMorph in
[unimorph.md:274](../unimorph.md).

## 3. Applicability to gomorphy's architecture

A direct analogy with Stages 15/16 (lemma -> full paradigm -> a DAWG
with payload) works poorly, for three reasons:

1. **Incomplete paradigms.** Only the forms of a lemma that were
   actually observed are extracted from the corpus. For a
   morphological analyzer this matters a lot — the whole point of a
   dictionary is usually paradigm completeness (from any form, learn
   the lemma and every other form). A dictionary built from UD would
   have 1-3 forms for many lemmas instead of 10-20.
2. **License heterogeneity by source** — merging several treebanks
   would require tracking provenance at the (lemma, form, tag) level,
   or explicitly dropping SynTagRus in favor of compiling only from
   CC BY-SA sources.
3. **The tag format isn't opaque, it's structured and universal** —
   unlike UniMorph/topical dictionaries, FEATS is `Attr=Val|Attr=Val`,
   not a single bundle string. Feeding it into the existing TSV engine
   (Stage 19, [todo.md:552](../todo.md)) as an opaque grammeme string
   is technically possible (take the whole FEATS as one opaque tag),
   but it loses UD data's main value — the structure and cross-
   language standardization of its features.

That said, the format is technically **suitable for parsing and
fitting into a TSV-like pipeline**: `LEMMA<TAB>FORM<TAB>UPOS+FEATS` is
almost literally the Stage 19 format (`lemma<TAB>form[<TAB>tags]`),
except that here the tags aren't opaque — they carry semantics that
would be a shame to throw away.

## 4. Options for use (to choose between)

### Option A — a full-fledged dictionary source (following Stages 15/16)

Aggregate every (lemma, form) pair found across the treebank(s) ->
compile into `.dat` via the existing pipeline (LCP stem, paradigm
dedup, sharding), serializing FEATS as a verbatim opaque tag
(similar to a UniMorph bundle) or as a sorted `Attr=Val|...` list.

- **Pro:** a minimal increment, reuses all of Stage 16's experience.
- **Con:** incomplete paradigms (see §3.1) — as a standalone dictionary
  for `Parse()`/`Lookup()` this is noticeably worse than
  OpenCorpora/UniMorph on coverage; its value as an "additional"
  dictionary (by analogy with how UniMorph is positioned,
  [unimorph.md:265](../unimorph.md)) is also questionable — it doesn't
  add vocabulary missing from OpenCorpora, it just covers the same
  vocabulary worse. There's a licensing risk if SynTagRus is chosen.

### Option B — a reference (disambiguated) test corpus

Don't compile a dictionary; use CoNLL-U directly as a set of
verification examples: for each sentence, tokens with homonymy already
resolved (FEATS refers to a specific usage in context, not to every
possible reading of the form). Run `gomorphy Parse()` on FORM and
compare against the reference LEMMA/UPOS/FEATS — an accuracy metric
for the analyzer on real text, which doesn't exist right now (the
project's unit/integration tests check against the source
dictionaries, not against independently annotated text).

- **Pro:** closes a real gap (context-based homonymy resolution is
  outside the current analyzer's scope, but reading-ranking quality
  can be measured); the data doesn't need to be compiled into `.dat` —
  it's read on the fly in tests/benchmarks; SynTagRus's license (NC)
  isn't a problem here — it's test data, not part of the distribution.
- **Con:** requires mapping UPOS+FEATS -> gomorphy's internal tag
  (the OpenCorpora set) for comparison — not trivially 1:1 (see
  option C).

### Option C — a source of a canonical "universal" tagset

Use UPOS+FEATS as the target schema for the open "universal tag
mapping between dictionaries" task recorded in [todo.md](../todo.md)
(the "Universal tag mapping between dictionaries — NOT DESIGNED"
section, raised by the user on 2026-09-16). One of the open questions
there is "The format of a 'universal tag' — is it its own independent
set, or does it reduce to one of the existing sets?" UD FEATS is
already a ready-made, documented, established cross-language feature
inventory (not invented for gomorphy, but a standing standard with
versioned guidelines), which directly answers that question with one
of the options.

- **Pro:** doesn't require importing any data at all — only the
  dictionary of features/values (`Case`, `Number`, `Gender`,
  `Animacy`, `Voice`, ... and their values) is used as the mapping's
  target schema; this closes an architectural question, not just adds
  "one more dictionary." It gives option B for free as a side effect
  (once the mapping exists, comparison against a reference corpus
  comes with it).
- **Con:** this is a design task (the OpenCorpora<->UD and
  UniMorph<->UD feature-pair mappings are incomplete and in places
  ambiguous — as already noted for UniMorph->FT12 in
  [unimorph.md §5.4]), not an importer increment; larger in scope than
  options A/B.

## 5. Recommendation

Don't start with option A. A full-fledged UD dictionary importer would
deliver low marginal value (see §4.A) with a real risk of license
confusion — this is what distinguishes UD from UniMorph, where the
"additional source" decision was justified (there, it's a lexicon, not
a corpus). A reasonable sequence:

1. First — **option C**, as a continuation of the universal tag
   mapping question the user already raised (it's needed for
   multi-dict anyway, independent of UD) — but this is a separate
   design task, requiring brainstorming, as already noted in
   todo.md.
2. **Option B** — a cheap increment on top of option C (the same
   mapping is used both for a tag-conversion report and for
   comparison against a reference); it can also be done without C, by
   accepting a simplified, partial mapping just for the test's purposes.
3. **Option A** — don't do it, until a concrete scenario shows up for
   which OpenCorpora/UniMorph aren't enough and UD lemmas/forms are
   specifically needed.

## 6. Open questions (for discussion with the user)

**Q1.** Do you agree with the "C -> B, defer A" priority from §5, or is
option A's value underestimated (e.g. for genre diversity — blogs/
poetry — that OpenCorpora doesn't have)?

**Q2.** If options B/C are in scope: is the mapping meant to be a
**manual table** (an OpenCorpora grammeme <-> a UD Attr=Val) or a
**programmatic** derivation (heuristics on name patterns)? A manual
table is smaller in scope (UPOS has ~17 values, FEATS has on the order
of 30 attributes), but needs linguistic review.

**Q3.** For option B — which treebank(s) should serve as the
reference? All signs point to SynTagRus (bigger and more reliable, but
its NC license is acceptable for test data, not for distribution
further down the chain if the tests ever become a public example/
fixture with a download dependency), or GSD/Taiga (permissive, but
noticeably smaller)?

**Q4.** Where should CoNLL-U files physically live for testing — by
analogy with Q4 in [0006](0006-unimorph-import-plan.md) (a manual
download into `.data/ud/` + `make test-integration`), or should a
loader be built?

**Q5.** Is a CoNLL-U parser even needed as a separate, reusable
component (`pkg/.../importers/conllu` or similar), even if options B/C
don't require compiling `.dat` — i.e. is only a sentence/token reader
needed, not a Builder pipeline?

## 7. Work-scope estimate (draft, pending a decision on Q1-Q5)

| # | Task | Option | Estimate |
|---|---|---|---|
| 1 | A CoNLL-U reader (a sentence/token scanner, no dictionary compilation) | B, C | 0.5 day |
| 2 | An OpenCorpora grammeme <-> UD UPOS/FEATS mapping table + application code | C | 1-2 days (linguistic review is the main time risk) |
| 3 | A test harness "Parse() vs. reference CoNLL-U" + an accuracy-metrics report | B (after C) | 0.5-1 day |
| 4 | An importer into `.dat`, following Stage 16 (if Q1 decides A is needed) | A | 1-1.5 days (reuses the bulk of Stage 16's code) |
| 5 | Documentation (`docs/`, `todo.md`) | all | 0.25 day |

**Total if choosing "B+C" (the §5 recommendation): ≈ 2-3.5 days.**
**Additionally if A is added: +1-1.5 days.**

## 8. Limits of applicability

The scope numbers are an expert estimate by analogy with Stages 15/16,
not a measurement. Not done: downloading and parsing real `.conllu`
files (only the site's README and format spec were read), evaluating
the quality of any ready-made UPOS<->OpenCorpora mapping tables from
third-party projects (they may already exist publicly — not checked),
comparing against other reference-text sources for option B (e.g. a
manual sample from the National Corpus outside UD). The decision on
options A/B/C and open questions Q1-Q5 haven't been made — they're
waiting on discussion with the user.

## Sources

- [universaldependencies.org](https://universaldependencies.org) —
  the homepage (the list of languages, the number of treebanks/tokens for Russian)
- [universaldependencies.org/format.html](https://universaldependencies.org/format.html)
  — the CoNLL-U format specification
- The `UniversalDependencies` GitHub organization, the repository list
  filtered by "Russian" — 5 repositories
- The README.md of each of the five repositories (SynTagRus, GSD,
  Taiga, PUD, Poetry) — size, genre, license, the "Machine-readable
  metadata" block
- Locally: [unimorph.md](../unimorph.md),
  [0006-unimorph-import-plan.md](0006-unimorph-import-plan.md),
  [todo.md](../todo.md) (Stage 19, the section on universal tag
  mapping), [implementation/stage-15-import-opencorpora.md](../implementation/stage-15-import-opencorpora.md)
