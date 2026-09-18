# Why paradigms with identical "endings" at identical positions don't merge (the comparative degree)

**Date:** 2026-09-15 (extended the same day, a separate session)
**Status:** the user's two original hypotheses are substantively
**disproven**; the real cause was found (the root is stored inside the
suffix), and a separate **bug** was confirmed along the way (a
byte-wise LCP slice cuts inside a rune -> 61.7% of shard 0's suffixes
are invalid UTF-8).
Addendum 1: a third hypothesis (hoisting a constant tag into the root)
is **partially confirmed** (82.8%/93.2% of cases), the effect on size
is small (4.3%/39.1% of paradigms per shard). Addendum 2: the
root-in-suffix fix mechanism was found, implemented, and verified on
real data — shard 0's paradigms shrank from 17,934 to 3,245 (after the
fix, the dictionary fits in 1 shard instead of 2), the suffix table
shrank by ~94.5%, and the share of invalid-UTF-8 suffixes dropped from
61.7% to 0%.
**Verified by:** Aleksey Marin (asmadews@gmail.com)
**Related documents:** [0002-paradigm-tagset-binary-encoding.md](0002-paradigm-tagset-binary-encoding.md), [2026-09-14-suffix-sharding-design.md](../superpowers/specs/2026-09-14-suffix-sharding-design.md), [stage-17-optimize.md](../implementation/stage-17-optimize.md)

## Context

The compiled dictionary `.data/opencorpora/opencorpora.dat` (2 shards)
contains a suffix section `suffixes-0`/`suffixes-1`: a `[]string`
array, where each paradigm has a corresponding `suffix_id[]` and
`tag_id[]` (see `pkg/morphology/internal/format.go`,
`EncodeParadigms`, `DecodeParadigms`).

While browsing the suffix section, the user noticed **large groups of
paradigms with identical endings at identical positions**, e.g.:

```
ёмкостнее / ёмкостней / поёмкостнее / поёмкостней
абажурнее   / абажурней   / поабажурнее   / поабажурней
бордажнее   / бордажней   / по-...        / ...
боригеннее  / боригенней  / по-...        / ...
```

(the comparative-degree paradigm: `COMP,Qual`, `COMP,Qual,V-ej`,
`COMP,Qual,Cmp2`, `COMP,Qual,Cmp2,V-ej` — 4 forms). Observation: the
"tails" are the same across all groups (`-нее / -ней / по-нее /
по-ней`), and there are many paradigms — why don't they merge into one?

The import/deduplication path (`pkg/morphology/importers/opencorpora/import.go`):

1. For each lemma, `stem := lcp(lem.forms)` (`import.go:136`) — a
   **byte-wise** LCP (`lcp`, `import.go:350`, compares `[]byte`).
2. For each wordform, `suffix = frm.text[len(stem):]`
   (`import.go:143-146`) — the **remainder after the LCP prefix**, not
   an "ending" in the morphological sense.
3. The paradigm `(suffix_id[], tag_id[])` is deduplicated by the key
   `paradigmKeyHash(suffixIDs, tagIDs)` (`import.go:184-195`), without
   prefixes (`import.go:251` — for OpenCorpora the prefix is always
   `"-"`).

Conclusion from 1+2+3: merging is possible **only** when the full list
of suffix ids and the full list of tag ids are identical. IDs uniquely
reflect suffix **strings** (the `suffixes-N` table), not "endings."

## Hypothesis (the user's)

1. **H1 — a merge bug:** the compiler loses/flattens identical
   paradigms; "separate tails of otherwise-identical paradigms" is a
   sign of broken deduplication.
2. **H2 — the tags differ:** the groups don't merge because their
   morphological tags differ (e.g. animacy), and the key hashes both
   suffixes and tags.

## Experiment

**Data:** the actually-built `.data/opencorpora/opencorpora.dat`
(built via `./deploy/gomorphy_build compile`), all sections read via
the package's existing functions (`OpenContainer`, `DecodeStrings`,
`DecodeTagSet`, `DecodeParadigms`).

**Tools:** a temporary `_test.go` in `pkg/morphology/internal`
(`zz_research_paradigm*.test.go`, files not kept in the repository),
calling only the package's exported functions plus a helper
`utf8Valid` (`unicode/utf8.ValidString`).

**Method:**

- A direct check of deduplication integrity (H1): build a
  `(suffix strings -> paradigm)` and `(tag sequence -> paradigm)` map,
  count conflicts of "one key pair -> different ids."
- A direct check of H2: find suffix signatures split across 2+ tag
  sequences (how many, and which ones).
- A byte-wise reproduction of `lcp()` for specific lemmas from
  `dict.xml` (for "ёмкостнее"/"абажурнее"/"бордажнее"/"боригеннее"/
  "яснее") in Python, cross-checked against the actual strings in the
  suffix section: computing `stem=lcp(forms)` and
  `suffix=form[len(stem):]`, checking `utf8.Valid` on the result.

**Reproduction steps:**
```bash
./deploy/gomorphy_build compile
# then — a temporary _test.go in pkg/morphology/internal:
# researchOpen -> (Container, suffixSets, TagSet, paradigms);
# TestResearchDedupPerfect / TestResearchInvalidUtf8Count /
# TestResearchH2TagDiff / TestResearchLargestTagGroups /
# TestResearchFindUserExamples (see the session history)
```

## Results

### H1 rejected: deduplication is perfect (1:1)

`TestResearchDedupPerfect`: every `(suffix-string list, tag list)`
pair maps to exactly one paradigm id:

| Shard | Paradigms | Unique pairs | Conflicts |
|---|---|---|---|
| 0 | 17,934 | 17,934 | 0 |
| 1 | 529 | 529 | 0 |

There are no "flattened/lost" paradigms that a merge bug would have
produced: the set of paradigm ids is bijective with the set of unique
canonical pairs. No duplicate "tail" paradigm was found.

### H2 holds in general, but NOT for the comparative groups

`TestResearchH2TagDiff` / `TestResearchSameSuffixDiffTags`:

| Shard | Suffix signatures | Split by different tags |
|---|---|---|
| 0 | 16,875 | 551 |
| 1 | 249 | 74 |

Real examples of "same suffixes, different tags" (shard 0):

- paradigm 40: suffixes `[ый ого ому ого ий им ом ая ой ой ую...]`,
  tags `ADJF,Qual,…` vs. `ADJF,…` (a qualitative vs. a plain
  adjective);
- paradigm 44: `ADJF,…` vs. `ADJF,Geox,…`;
- paradigm 45: `NOUN,anim,masc,Name,…` vs. `NOUN,anim,masc,…`;
- paradigm 30: `NOUN,inan,masc,Geox,…` vs. `NOUN,inan,masc,…`.

This is **correctly** not merged: the same surface scheme with
different tags. But this **doesn't apply** to the comparative-degree
groups: their tag sequences are **pairwise identical**.

### The real cause: suffix = wordform minus the LCP prefix, the root lives inside the suffix

In `dict.xml`, the comparative degree is a **separate 4-form lemma**
(not part of the base adjective's lemma):

```
LEMMA ёмкостнее  | forms: [ёмкостнее, ёмкостней, поёмкостнее, поёмкостней]
LEMMA яснее      | forms: [яснее, ясней, пояснее, поясней]
LEMMA ёмкостный  | forms: [ёмкостный, ёмкостного, ...]  (27 forms)  — its own lemma
LEMMA ёмкостен   | forms: [ёмкостен, ёмкостна, ёмкостно, ёмкостны] — short form
```

For such a lemma, the LCP of all 4 forms is empty (the "по-" form
diverges from the base form at the very first character), so **each
whole wordform becomes the suffix**:

```
ёмкостнее  → suffix "ёмкостнее"
ёмкостней  → suffix "ёмкостней"
поёмкостнее→ suffix "поёмкостнее"
...
```

The "ending" (`-ее / -ей`) really is the same at the same positions —
but at the **end of each of several different suffix strings, each
starting with the word's root**. Different roots (ёмкостн-, абажурн-,
бордажн-, боригенн-) -> different suffix strings -> different
`suffix_id` -> a different deduplication key. The tags, meanwhile, are
shared — so what results is **one large group of paradigms with an
identical tag list and different suffixes**, exactly what the user is
seeing:

| Tag sequence | Paradigms in shard 0 | Paradigms in shard 1 |
|---|---|---|
| `COMP,Qual / COMP,Qual,V-ej / COMP,Qual,Cmp2 / COMP,Qual,Cmp2,V-ej` | **7,597** | 19 |
| `COMP / COMP,V-ej / COMP,Cmp2 / COMP,Cmp2,V-ej` | **7,136** | 4 |

This is **not a bug**: gomorphy doesn't store "endings," it stores
"suffix = the remainder of the form after the lemma's common prefix."
The comparative group could only be merged by changing the model
(e.g. storing the "по-" prefix separately and subtracting the LCP
prefix at the level of the verb's own forms) — but under the current
"suffix = post-LCP remainder" model, different roots inside the
suffixes make merging impossible by construction, and this matches the
data 1:1.

### A bug found along the way: a byte-wise LCP cuts inside a rune -> 61.7% of shard 0's suffixes are invalid UTF-8

`lcp()` (`import.go:350`) compares `[]byte`, not runes. For lemmas
whose forms diverge **inside the first UTF-8 character**, the LCP
stops at a byte boundary inside the character, and
`frm.text[len(stem):]` produces a string starting with a "half-eaten"
byte.

- "ёмкостнее": `ё = d1 91`, the "по-" form = `п = d0 bf` — the first
  bytes **differ** (`d1 ≠ d0`) -> `LCP = ""` -> the suffixes are
  valid, the table is clean.
- "абажурнее": `а = d0 b0`, the "по-" form = `п = d0 bf` — the first
  bytes **match** (`d0 = d0`), the second byte differs (`b0 ≠ bf`) ->
  `LCP = d0` (1 byte, the middle of a rune) -> `suffix = form[1:]`, the
  suffix has its first UTF-8 byte eaten — the string is **invalid**.
- "бордажнее"/"боригеннее" — the same pattern (б/п in the `d0` block).
- Any word starting with a letter from the `d0` block (а…п) that has a
  "по-"-prefixed form gets an entry that merges correctly at the byte
  level but is invalid as text.

Confirmed by the counts (`TestResearchInvalidUtf8Count`,
`TestResearchInvalidUtf8Paradigms`):

| Shard | Suffixes | Invalid UTF-8 | Invalid share | Paradigms with ≥1 invalid suffix |
|---|---|---|---|---|
| 0 | 65,534 | 40,442 | 61.7% | 10,385 of 17,934 (57.9%) |
| 1 | 688 | 179 | 26.0% | 53 of 529 |

The connection to the large groups (`TestResearchLargestTagGroups`):
in the `COMP,Qual` group (7,597), invalid suffixes occur in 4,760
paradigms — exactly the "абажурнее/бордажнее/боригеннее…" groups
(letters from the `d0` block); the "2837 paradigms" counted earlier in
the first pass is **only the valid subset** of the same group
(7,597 − 4,760 = 2,837 exactly, and "2501" from the first pass =
7,136 − 4,635).

Semantically such suffixes remain correct at the byte level for the
DAWG (the `stem+suffix` key reconstructs the full wordform byte-for-
byte), so `lookup` doesn't break, but **the suffix table in `.dat`
contains garbage strings** that can't be interpreted as text: this is
what the user is seeing, and it breaks any consumer that treats
suffixes as UTF-8 (trimming a word's trailing characters,
normalization, etc.).

## Conclusions

1. **H1 (a merge bug) — disproven.** Paradigm deduplication is 1:1
   (17,934 = 17,934 pairs, 0 conflicts). There are no duplicate
   "tails"; merging always happens when the full suffix-id and tag-id
   lists match.
2. **H2 (tags) — true in general** (551/74 cases), **but doesn't
   explain the observed comparative-degree groups**: there the tag
   sequences are identical, only the suffixes differ.
3. **The real cause of non-merging** is in the storage model: a
   "suffix" in gomorphy isn't an ending, it's the remainder of a
   wordform after the common LCP prefix. For comparative-degree
   lemmas the LCP is empty, so **the word's root lives inside the
   suffix**, different roots -> different suffixes -> different
   paradigms even with shared tags. This behavior is consistent with
   the data and with `paradigmKeyHash`.
4. **A separate bug was found and confirmed**: `lcp()` is byte-wise
   and cuts inside a rune for lemmas whose forms diverge in the first
   character (any word starting with a letter from the `d0` block plus
   a "по-"-prefixed form). Effect: 61.7% of shard 0's suffixes are
   invalid UTF-8, affecting 57.9% of paradigms. Functionally, lookup
   isn't affected (byte-wise reconstruction of `stem+suffix`), but the
   suffix table contains unreadable strings — a necessary consequence
   for future consumers of suffixes as well.
5. **If merging the comparative forms is needed** — that's a format
   change (a "prefix + ending" pair model instead of "LCP + post-LCP
   remainder" for such lemmas / hoisting "по-" into the prefix), a
   separate task outside the scope of this research. Implemented and
   verified — see "Addendum 2" below.
6. **The fix for the bug in item 4** — replace `lcp()` with a
   rune-aware version (truncate the result to a rune boundary), or, if
   deliberately storing byte slices, mark such suffixes as such; but
   the current state (invalid UTF-8 strings with no marking at all) is
   a data defect in `.dat`. Implemented (a rune-aware `lcp()`) — see
   "Addendum 2" below.

## Addendum 1 (2026-09-15, a separate session): a formal check of H2 — the tag delta's constancy

**Status:** the user's hypothesis about "hoisting a constant tag into
the root" is partially confirmed: the structure of the differences in
H2 conflicts allows such a decomposition in most (not all) cases; the
effect on `.dat` size is small and entirely independent from the
root-in-suffix finding (see "Addendum 2" below — that's where the main
lever is).

### Context

The main text above only established one fact: 551/74 suffix
signatures (shard 0/1) are split across 2+ different tag sequences (the
"H2 holds in general..." section). The *structure* of the difference
wasn't checked: is it a constant addition/removal of the same grammeme
on **every** form of the paradigm (a property of the lemma) — that's
exactly what's needed for the idea of "hoist the grammeme into the
root, merge it back in during lookup."

Separately, a fact important for the implementation: `xmlHandler` in
`import.go` already separates grammemes into `lemGrams` (from `<l>`,
persistent for the whole lemma) and `formOwnGrams` (from `<f>`, per
form) — see the struct's fields (`import.go:278-281`) — but
`OnFormEnd` (`import.go:336-347`) immediately flattens them into a
single string (`strings.Join(all, ",")`, line 344) and registers it as
a single `TagSet` id. Keeping them stored separately from the form is
physically possible without reworking the XML scanner — they're
already assembled, just discarded at this step.

### Hypothesis (the user's), H3

If the tag difference in an H2 conflict is structured as the same
grammeme added/removed on **every** form index of the paradigm (i.e.
it's a property of the lemma, not of a specific wordform), then that
grammeme could be stored once per lemma (e.g. attached to the
root/DAWG payload) instead of duplicated across every paradigm variant
— and the final tag assembled on the fly during lookup (lemma tag ∪
form tag), reducing the number of distinct paradigms.

### Experiment

Method: for each suffix signature with 2+ different tag sequences,
take one representative of each unique sequence, compare it pairwise
against a base representative — decompose each tag into grammemes
(`strings.Split(tag, ",")`), compute the symmetric difference
(added/removed) of the grammeme sets at each form index, and check
whether this delta is the same across all indices. Separately — an
upper bound on the savings in paradigm count from merging all H2
groups (ignoring the delta's structure) and from merging only the
"constant" groups.

Tools: the same method as in the main research (a temporary `_test.go`
in `pkg/morphology/internal`, the package's exported functions:
`OpenContainer`, `DecodeTagSet`, `DecodeParadigms`), on the real
`.data/opencorpora/opencorpora.dat`.

### Results

| Shard | H2 conflicts | Constant delta on every form | Not constant |
|---|---|---|---|
| 0 | 551 | 456 (82.8%) | 95 (17.2%) |
| 1 | 74 | 69 (93.2%) | 5 (7.2%) |

Examples of non-constant cases (real, shard 0/1):
- a single form with a one-off tag (`Infr`/`Slng`/`Abbr`) that doesn't
  affect the rest of the paradigm's forms — looks like OpenCorpora
  marking a specific wordform, not the lemma;
- one case in shard 1 where the suffixes **coincidentally matched**
  between two genetically unrelated paradigms with a different case
  order (`Geox,femn,inan` vs. `Init,Name,anim,ms-f`) — this is a
  collision between suffix strings from different objects, not "one
  paradigm + an added tag."

Savings in paradigm count from merging:

| Shard | Total paradigms | Savings (constant groups only) | Upper bound (merging all same-suffix groups, ignoring tags) |
|---|---|---|---|
| 0 | 17,934 | −770 (4.3%) | −1,059 (5.9%) |
| 1 | 529 | −207 (39.1%) | −280 (52.9%) |

### Conclusions

1. **H3 is partially confirmed**: 82.8% (shard 0) / 93.2% (shard 1) of
   H2 conflicts have a lemma-level-constant tag delta — suitable for
   hoisting into a "lemma tag." The remaining 17.2%/7.2% are not (one-
   off marks on individual forms, rare suffix collisions between
   unrelated objects) — an implementation would need a fallback path,
   not a universal exception-free mechanism.
2. **The size effect is small and not "dramatic"**: a reduction in
   paradigm count of 4.3% (shard 0) / 39.1% (shard 1); `paradigms-N` is
   already only ~4% of the file, so in absolute bytes the effect is a
   fraction of a percent of `.dat` (shard 0) and negligible in absolute
   terms (shard 1, the file is already small).
3. **This axis is entirely independent of root-in-suffix** (see
   "Addendum 2"): there the tags are identical and the duplication
   comes from suffixes due to an empty LCP, not from tags — the effect
   there is two orders of magnitude bigger in paradigm count (82% for
   shard 0 vs. 4.3% here).
4. **Integration point for an implementation**: `OnFormEnd`
   (`import.go:341-345`) — don't flatten `lemGrams`/`formOwnGrams` into
   one string, keep them separate. Precondition: an atomic `tagset`
   (already described as backlog candidate #3 in
   [research 0002](0002-paradigm-tagset-binary-encoding.md)) —
   otherwise composing "lemma tag ∪ form tag" on the fly still
   requires materializing the full string on every lookup, losing the
   benefit.
5. **Recommendation**: not a priority for Stage 17 given the current
   numbers. Consider it separately from root-in-suffix (Addendum 2) if
   pursued at all — and even then more as a model cleanup than as a
   compression lever.

## Addendum 2 (2026-09-15, the same session): root-in-suffix — what can be done and what effect it has

See the separate write-up sent to the user in this same session —
briefly: the format already has the necessary mechanism
(`Paradigm.Prefix(form)`, the `prefixes` section), used by the
pymorphy2 importer (`defaultPrefixes = []string{"", "по", "наи"}`,
`pkg/morphology/importers/pymorphy2/import.go:19`), but (before this
fix) unused by the OpenCorpora importer — `prefixes` was filled with
zeros for every form, the index always 0. Verified on the real
`dict.xml` (lemma `поправимее`, id 259490:
`forms=[поправимее, поправимей, попоправимее, попоправимей]`, the
`Cmp2` forms are literally equal to `"по"+base form`): if "по" is
split off using the `Cmp2` grammatical marker **before** computing the
LCP (not after, and not retrofitted on top of the suffix already
computed for `.dat` — retrofitting on top of the old LCP produces
artifacts like a root of `поп` instead of `по`+`поправим`, visible on
this same example), the paradigm collapses to the canonical form
`prefix=["","","по","по"], suffix=["е","й","е","й"]`, identical for
`поправимее`, `яснее`, and any other adjective with this comparative-
degree pattern — regardless of the root, because the root isn't stored
in the paradigm at all, it's reconstructed from the word's literal
text in `words.dawg` via `TrimPrefix`/`TrimSuffix`
(`pkg/morphology/parse.go:198-221`).

**The exact quantitative effect was recomputed on real data** (Task 3
of the research SDD session `2026-09-15-comparative-prefix-split`, the
fix implemented in `stripCmp2Prefix`, `import.go`): the fix was built
(`./deploy/gomorphy_build compile` on the full
`.data/opencorpora/dict.xml`, ~400K lemmas) and the dictionary was
re-measured using the same exported package functions as in the
original research (`OpenContainer`, `DecodeStrings`,
`DecodeParadigms`, `DecodeTagSet`, a temporary `_test.go` in
`pkg/morphology/internal`, removed after measuring).

| Metric | Before the fix | After the fix |
|---|---|---|
| Number of shards | 2 | **1** (after collapsing, everything fits in one shard — the suffix table no longer approaches the `suffixShardLimit`=65536 boundary) |
| Total paradigms, shard 0 | 17,934 | **3,245** |
| Total paradigms, shard 1 | 529 | — (shard 1 no longer exists) |
| Suffixes, shard 0 | 65,534 | **6,080** |
| Invalid-UTF-8 suffixes, shard 0 | 40,442 (61.7%) | **0 (0.0%)** |
| Invalid-UTF-8 suffixes, shard 1 | 179 (26.0%) | — |
| Suffix table bytes, shard 0 | ~1.38 MB | **75,618 bytes (~73.8 KB)** |
| Paradigms in the `COMP,Qual,…` group (4 forms) | 7,597 | **4** |
| Paradigms in the `COMP,…` group (4 forms) | 7,136 | **2** |

Both comparative-degree tag groups (`COMP,Qual,…` and `COMP,…`), which
together produced 14,733 paradigms (≈82% of shard 0's paradigms),
collapsed to 6 paradigms total — confirming the "split off 'по' by the
Cmp2 marker before computing the LCP" mechanism exactly as predicted
in the section above. The outcome matches, in order of magnitude, the
upper bound predicted earlier (94.4% bytes / 82% paradigms for
shard 0): the actual result is a reduction of shard 0's suffix table by
roughly 94.5% (75,618 bytes instead of ~1.38 MB) and a reduction of the
total paradigm count from 18,463 (17,934+529, both shards) to 3,245
(the single remaining shard) — about 82.4%. Along the way, the byte-
wise UTF-8 bug from the main research was also closed: the share of
invalid-UTF-8 suffixes in shard 0 dropped from 61.7% to 0% — 100% of
suffixes are valid UTF-8.

**A separate nuance found during this same measurement, a narrow edge
of the fix's scope:** in the real `dict.xml` there are 3 lemmas
(`недобитее`, `окологлоточнее`, `мультипроцессорнее`) whose base word
already carries its own prefix (недо-/около-/мульти-), and OpenCorpora
infixes "по" after that prefix rather than prepending it to the whole
word (e.g. `недобитее` -> the Cmp2 form `недопобитее`, not
`понедобитее`). For these 3 lemmas out of ~400K, the narrow (`Cmp2`
only, literal "по" prefix only) fix doesn't apply literally — but the
existing fallback path in `stripCmp2Prefix` (`ok=false`) safely
handles them as before (no merging, but no data corruption either);
this is an expected, deliberate limitation of the chosen narrow scope,
not a defect in the fix — see
`TestStripCmp2PrefixRealDictAnomalies` in
`pkg/morphology/importers/opencorpora/real_dict_integration_test.go`.

## Sources

<No third-party sources were used; every reference in the text is to
files and documents in this repository, verified by reading and
running code in this session>
