# OpenCorpora import: split comparative-degree "по-" prefix from suffix

## Context

`docs/research/0003-comparative-paradigms-not-merging.md` investigated why
paradigms for comparative-degree adjectives (`яснее`/`ясней`/`пояснее`/
`поясней`-style, 4 forms: `COMP[,Qual]` / `[...],V-ej` / `[...],Cmp2` /
`[...],Cmp2,V-ej`) never merge despite sharing identical tag sequences.
Root cause: for these lemmas the byte-LCP of all 4 form texts is empty or
near-empty (the `Cmp2`-tagged forms carry a literal `"по"` prefix that the
other two forms don't), so under gomorphy's "suffix = form text minus
LCP-prefix" model, the word's own root ends up *inside* the suffix string.
Different roots → different suffix strings → different paradigms, even
though the tag sequences are pairwise identical.

Measured on the real `.data/opencorpora/opencorpora.dat` (shard 0):

- These two tag groups (`COMP,Qual,...` and `COMP,...`) account for
  **14,735 of 17,934 paradigms (82%)**.
- They account for **94.4% of the suffix-table bytes** (1.31 MB of 1.38 MB).
- `suffixes-N` is ~9% and `paradigms-N` is ~4% of the whole ~16 MB file.

Separately, the same research found a **byte-vs-rune bug** in `lcp()`
(`pkg/morphology/importers/opencorpora/import.go:350`, byte comparison,
no rune-boundary awareness): for lemmas whose forms diverge mid-rune (any
Cyrillic letter in the а…п block with a `Cmp2` form), the LCP cut lands
inside a UTF-8 character, producing invalid-UTF-8 suffix strings — 61.7%
of shard 0's suffixes (40,442 of 65,534), 26.0% of shard 1's (179 of 688).
Of the paradigms with at least one invalid suffix, **9,395 belong to the
two comparative-degree tag groups above and 990 do not** — the two bugs
overlap but are not the same population.

Both bugs are being fixed together in this plan (see "Decision" below):
the "по-" split fix, done correctly, structurally avoids the byte-cut
issue for every comparative-degree lemma it touches (no more raw-byte LCP
across forms that differ at their very first character), and a
rune-safe `lcp()` closes the remaining ~990 paradigms outside the
comparative-degree groups for free.

## Prior art already in the codebase

gomorphy's on-disk format already has everything needed — it's simply
unused by the OpenCorpora importer:

- `internal.Paradigm` stores three **parallel** arrays per form:
  suffix ID, tag ID, and **prefix ID** (`internal/paradigm.go`,
  `NewParadigm(suffixes, tags, prefixes []uint16)`).
- `internal.Dictionary.Prefixes` is a string table, decoded from a
  `prefixes` section in the `.dat` file (`open.go:100-104`), read at
  lookup time by `paradigmAffix` (`parse.go:236-245`,
  `strAt(x.d.Prefixes, para.Prefix(form))`).
- The **pymorphy2** importer already populates this correctly:
  `pkg/morphology/importers/pymorphy2/import.go:19`,
  `defaultPrefixes = []string{"", "по", "наи"}` — pymorphy2's own
  dictionaries split comparative/superlative prefixes off as real
  prefixes, and gomorphy just loads that table as-is.
- The OpenCorpora importer never does this: `import.go:193`,
  `prefixes := make([]uint16, len(lem.forms))` — always index 0 (empty
  prefix), and `import.go:251` passes `nil` for the dictionary-level
  `Prefixes` table entirely.

Verified against real `dict.xml`: lemma `поправимее` (id 259490) has
`forms = [поправимее, поправимей, попоправимее, попоправимей]`; the two
`Cmp2`-tagged forms are literally `"по"` + the corresponding non-`Cmp2`
form. This is the only lemma-internal separable-prefix pattern present in
OpenCorpora's `dict.xml` — checked, and the superlative grammeme `Supr`
(e.g. `абстрактнейший`) is always its own fully-declined lemma, not a
`Cmp2`-style prefixed variant of another lemma, so it needs no special
handling here.

## Decision

**Scope: narrow, `Cmp2`-driven only** — not a generic configurable-prefix
mechanism. `Cmp2` → `"по"` is the only pattern present in OpenCorpora's
`dict.xml`; a generic table+detector adds complexity for hypothetical
future cases (e.g. a UniMorph importer with a different pattern) that
would get its own scoped fix when/if it actually arises.

**Mechanism: strip the known prefix from raw form text *before* computing
LCP — not retrofitted onto an already-computed suffix.** Two approaches
were compared:

- **A (chosen).** In the per-lemma loop (`import.go:131` onward, which
  runs after XML scanning — `frm.gramm` is already the fully-assembled
  tag string for each form), classify each form by
  `strings.Contains(frm.gramm, "Cmp2")`. For `Cmp2` forms, strip the
  literal `"по"` prefix (4 bytes) from the form text *before* it enters
  LCP computation. Compute LCP over the stripped texts. Suffix = stripped
  text minus that LCP. Prefix = `"по"` or `""` per form.
- **B (rejected).** Keep computing LCP over raw form text as today, then
  try to detect and strip a `"по"` prefix from the *resulting suffix
  string*. Empirically shown broken in this session's own investigation:
  for lemma `поправимее` (root itself starts with `"по"`), the current
  byte-LCP already consumes `"поп"` as shared prefix across all 4 raw
  forms (base `поправимее` vs `Cmp2` form `попоправимее` — first 3
  characters match: `п`,`о`,`п`), so the stored suffix for the `Cmp2`
  form is `"оправимее"`, not `"по"+"правимее"` — retrofitting prefix
  detection on top of this gives garbage. Order of operations matters;
  the prefix must be peeled off the source text first.

**Error handling: fail-fast is not enough; fall back per-lemma.** If a
`Cmp2`-tagged form's text does not literally start with `"по"` (an
anomaly this session did not observe in the real `dict.xml`, but the
importer must not assume it), don't abort the whole import — fall back to
current (no-prefix-split) behavior for *that lemma only*, and increment a
counter. A real-data regression test asserts this counter is exactly 0
against the current `dict.xml`; a future `dict.xml` update that
introduces such a case will fail that test visibly instead of silently
producing a wrong paradigm.

## Design

### `lcp()` becomes rune-safe

`pkg/morphology/importers/opencorpora/import.go:350`. Keep the existing
byte-comparison loop (no behavior change for the common case — it already
returns 0 for genuinely different first characters), but if the resulting
cut point is not a valid rune boundary, trim back to the last one
(`utf8.RuneStart(b)` on each preceding byte, or equivalent). This is a
general fix, independent of the prefix-split logic below, and applies to
every lemma the importer processes — not just comparative-degree ones.

### Prefix classification and stripping

New logic in the per-lemma loop, replacing the current unconditional
`stem := lcp(lem.forms)`:

```
for each form in lem.forms:
    hasCmp2 := strings.Contains(form.gramm, "Cmp2")
    if hasCmp2:
        if !strings.HasPrefix(form.text, "по"):
            anomalyCount++
            hasCmp2 = false // fall back for this lemma below
    strippedText[i] = hasCmp2 ? form.text[len("по"):] : form.text
    formPrefix[i] = hasCmp2 ? "по" : ""

if anomalyCount > 0 for this lemma:
    // fallback: treat as if no form has Cmp2 (all formPrefix = "")
    strippedText := original form texts unchanged

stem := runeSafeLCP(strippedText)
for each form:
    suffix[i] = strippedText[i][len(stem):]
    prefixID[i] = registerPrefix(formPrefix[i])   // new prefixTexts map, like suffixTexts
```

### Dictionary-level `Prefixes` table

`x.d.Prefixes` is built the same way `suffixes-N` already is (a
`map[string]uint16` accumulated across the whole import, converted to a
`[]string` at the end) instead of the current `nil`. Per the existing
`parse.go` contract, this table is **shared across all shards** — no
per-shard bookkeeping needed. In practice this table will contain just
`["", "по"]` for real OpenCorpora data (plus whatever the anomaly
fallback never needs, since it reuses the empty-prefix entry).

### `paradigmKeyHash` includes prefix IDs

`import.go:184`. Signature changes from
`paradigmKeyHash(suffixIDs, tagIDs []uint16) string` to
`paradigmKeyHash(prefixIDs, suffixIDs, tagIDs []uint16) string`, hashing
all three arrays. This is not optional: once `prefixID` actually varies
(0 vs 1) instead of always being 0, two paradigms with identical
suffix+tag sequences but different prefix sequences must **not**
collapse — this is the same failure mode as the already-fixed
`paradigmKeyHash` bug in `docs/todo.md` ("баг 2": `n = copy(...)` instead
of `n += copy(...)` silently dropping tag bytes from the dedup key).

### Read path — unchanged

`pkg/morphology/parse.go` (`paradigmAffix`, `readingForm`) needs no
changes. It already generically composes `prefix + stem + suffix` and
already correctly handles non-empty prefixes for pymorphy2 dictionaries —
this fix only changes what the OpenCorpora importer *writes*, not how
anything is *read*.

## Non-goals

- A generic/configurable prefix-detection mechanism (table beyond `["",
  "по"]`, or detection not driven by the `Cmp2` grammeme) — explicitly
  out of scope per the "narrow" decision above. If a future importer
  (e.g. UniMorph, Stage 16) needs a different pattern, it gets its own
  scoped design.
- Handling `Supr` (superlative) specially — verified or that grammeme
  always marks a fully independent lemma in `dict.xml`, not a
  `Cmp2`-style prefixed variant; no change needed there.
- The separate H2 tag-hoisting idea from `docs/research/0003-...md`
  ("Дополнение 1") — investigated and explicitly deprioritized in that
  document (small effect, unrelated axis); not part of this plan.
- Any change to `words.dawg` construction, DAWG minimization, or the
  DAWG-alphabet-density backlog item (`docs/research/0001-...md`) — this
  plan only changes the `suffixes-N`/`paradigms-N`/`prefixes` sections
  and the importer logic that produces them.
- zstd compression, tagset atomic-grammeme encoding, or any other Stage
  17 backlog item — unrelated, untouched by this plan.

## Testing

(Exact test list belongs in the implementation plan; noting shape here.)

- `lcp()`: unit tests for the rune-safety fix, specifically the documented
  byte-block-collision case (а…п block letter + a form that diverges at
  the first character) — no more mid-rune cuts; a case where the cut was
  already correct (letters outside that block) is unchanged.
- `testDictXML` fixture gains a comparative-degree lemma with `Cmp2`
  forms (based on the real `поправимее` example) — regression test
  asserts the resulting `prefix`/`suffix`/`tag` per form.
- `paradigmKeyHash`: regression test with two lemmas sharing a
  suffix+tag sequence but differing prefix sequence — must produce two
  distinct paradigms, not one.
- Anomaly fallback: a fixture lemma with a `Cmp2` form that does *not*
  start with `"по"` — import must not fail, must fall back for that
  lemma only, and the anomaly counter must reflect it.
- End-to-end regression test resolving specific real comparative-degree
  words (e.g. `яснее`, `абажурнее`, `поправимее`) through
  `words.dawg` → paradigm → normal form / tag, verifying output is
  unchanged from pre-fix behavior (same failure mode as the precedent
  `paradigmKeyHash` bug: word/lemma text right, tag or normalization
  silently wrong).
- Real-data verification (`./deploy/gomorphy_build compile` against
  `.data/opencorpora/dict.xml`): anomaly counter is exactly 0; record the
  actual post-fix paradigm/suffix counts for shard 0/1 (closing the
  "not recomputed" caveat left open in
  `docs/research/0003-comparative-paradigms-not-merging.md`,
  "Дополнение 2") — update that document with the real numbers once
  known.
- `go test ./... -race` green.

## Risks

- `paradigmKeyHash`'s signature change touches every call site in
  `import.go` — small in count, but this exact function was the site of
  a previous silent-data-corruption bug; the new prefix-inclusion test
  above is the direct mitigation, not just a nice-to-have.
- The real-data end-to-end counts (paradigms/suffixes after the fix)
  are not known ahead of implementation — this session's own attempt to
  simulate them by retrofitting on top of the existing `.dat` produced
  incorrect numbers (see "Decision" / Approach B). They must be measured
  post-implementation, not predicted.
- `TagSet`/`Suffixes`/`Prefixes` content changes (new prefix strings,
  fewer/different suffix strings) — acceptable pre-1.0 (no `.dat`
  compatibility to preserve), same as the precedent tag-fix spec noted
  for its own changes.
