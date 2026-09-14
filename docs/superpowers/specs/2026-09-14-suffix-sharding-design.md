# OpenCorpora import: shard by suffix-id overflow

## Context

During the pre-1.0 code review triage session (see
`docs/code-review-pre-1.0.md`), an explicit overflow guard was added for
suffix/paradigm/tag ids in the OpenCorpora importer (`uint16(len(...))` was
silently wrapping past 65536). That guard immediately fired on the real
`.data/opencorpora/dict.xml`: **65835 unique suffixes, 299 over the uint16
limit (65536)**. Before the guard, this was silent data corruption (the
65536th suffix's id wraps to 0, colliding with the first-ever registered
suffix); now it's a hard, honest build failure — but the real dictionary
currently cannot be compiled at all. This is the second critical,
release-blocking finding, tracked in `docs/todo.md` under "Суффиксы:
расширение адресации".

Tags (3437) and paradigms (16939) are comfortably under the same 65536
ceiling today and are **not** part of this work.

## Decision: shard, don't widen the id type

Two approaches were compared (see the brainstorming conversation this spec
comes from): widening the suffix id from `uint16` to `uint32`, vs.
partitioning lemmas into multiple independent "shards" (each with its own
uint16 suffix-id space starting at 0).

**Decision: shard.** Reasoning:
- The user does not expect frequent/large overflows — a few hundred over a
  65536 budget is the expected shape of the problem, not systematic 2-10x
  growth. A general scaling mechanism (sharding) that only ever needs 2
  shards in the common case is preferable to a permanent doubling of every
  suffix/tag/paradigm id's storage cost, "just in case."
- Sharding requires **no format-breaking type change** and **no public API
  change beyond one additive field** (`Reading.Shard`, see below) — safer
  pre-1.0 than reshaping `Paradigm`'s wire layout.
- Sharding composes with future needs (Этап 19 thematic dictionaries,
  parallel build/load) that a scalar type widening does not address.

Paradigm id was specifically considered and rejected for widening on its
own merits, independent of the shard-vs-widen choice: it is packed into the
DAWG payload's upper 16 bits (`uint32(paraID)<<16 | uint32(formIdx)`,
`import.go`) and exposed as the public `Reading.Para uint16` field.
Widening it would touch the DAWG payload layout and break a public field's
type — a much larger blast radius than suffix/tag id widening would have
been, for a dimension (16939 of 65536) that isn't actually the problem.

### Why not widen anyway, later, as a smaller optional change?

Not rejected — captured as backlog (see "Deferred ideas" below). It's
explicitly *not* part of this shard-based fix; the two are independent
knobs (shard count vs. per-shard id width) that could eventually compose.

## Empirical grounding

Before committing to a specific default sharding algorithm, the real
corpus was measured directly (XML-scan + suffix computation only, ~4-5s,
no DAWG build) rather than guessed at or tested against a synthetic index:

- Suffix-count growth vs. corpus fraction is clearly sub-linear
  (Heaps'-law-shaped): 10% of lemmas already produce 7096 of the eventual
  65835 unique suffixes (10.8%); by 50% of lemmas, 35612 (54%) are already
  registered.
- Average suffix string is ~22.1 bytes (with its length-prefix byte); the
  whole suffix table is ~1.45 MB today — trivially small against the
  ~300 MB full dictionary, so **byte overhead from shard duplication is
  not the concern; shard *count* and *implementation complexity* are.**

Three candidate shard-assignment strategies were simulated on the real
lemma set (391842 lemmas), at the real N=2 boundary and at synthetically
shrunk per-shard caps (to preview N=3..7 behavior on the same real data,
since the corpus doesn't naturally reach that scale yet):

| effective cap | shards needed | round-robin overhead | fill-on-demand overhead | paradigm-grouped overhead |
|---|---|---|---|---|
| 100% (65536, today's real case) | 2 | +5.9% | **+0.59%** | +0.02% |
| 60% | 2 | +5.2% | +5.2% | +0.78% |
| 40% | 3 | +9.5% | +9.5% | +1.13% |
| 30% | 4 | +12.5% | +12.5% | +1.38% |
| 20% | 6–7 | +18.5% | +18.5% | +1.68% |

("Overhead" = extra suffix-id slots needed across all shards vs. the true
global unique count, i.e. duplicate registrations of the same suffix text
in more than one shard.)

Round-robin (each lemma assigned `index % N`) is dominated by both other
strategies at every scale and isn't considered further. Fill-on-demand
(fill shard 0 to its cap before starting shard 1, in corpus order) is
near-free at N=2 (+0.59%) and degrades toward round-robin's overhead as N
grows, because it stops benefiting once every shard is independently
"full" of the same globally-common suffixes. Paradigm-grouped bin-packing
(group lemmas by their paradigm's suffix-set identity, bin-pack paradigms
by descending popularity) stays under 2% overhead even out to 6–7 shards,
because a paradigm *is* the natural unit of suffix reuse — but needs a
two-pass build (compute paradigm groups and their popularity, then
bin-pack) instead of fill-on-demand's single pass.

**Decision: ship fill-on-demand as the only strategy for v1.** It captures
effectively all the achievable benefit at today's real N=2, is a single
pass over already-sorted-by-corpus-order lemmas (simplest to implement and
reason about), and the `ShardingStrategy` interface (below) means
paradigm-grouped can be added later as a second implementation without
touching anything else, if/when a corpus needs N≥3 and the extra ~9%+
overhead starts to matter.

## Design

### What's shared vs. per-shard

- **Shared across all shards**: `TagSet` (grammeme tags — not overflowing,
  and tag ids should stay globally meaningful for tag lookups/parsing
  regardless of which shard a form's paradigm lives in). The same
  `*internal.TagSet` instance is threaded through every shard's build
  pass.
- **Per-shard**: `Suffixes []string`, `Paradigms []Paradigm`, `Words
  *internal.DAWG`. These are exactly the three things whose ids are
  suffix-id-derived (a paradigm's identity depends on its suffix-id
  sequence) or suffix-id-addressed. Each shard gets a fresh, independent
  id space starting at 0.
- `Prefixes []string` stays as it is today (OpenCorpora's own prefixes are
  currently always `nil`/unused; not touched by this work).

### On-disk format

No change to the container mechanics (`SaveContainer`/`OpenContainer`,
`pkg/morphology/internal/format.go`) — it already supports arbitrary named
sections. Shards are encoded as numbered sections instead of the current
single set:

```
tagset                  (shared, unnumbered — unchanged)
suffixes#0  paradigms#0  words.dawg#0
suffixes#1  paradigms#1  words.dawg#1
...
```

A shard count needs to be recorded somewhere readable before the numbered
sections are parsed — either a new small field on `BuildInfo`/the info
section, or the count can be inferred by probing for `suffixes#N`
sections in the catalog until one is missing (simpler, no format
versioning needed, but slightly less explicit). **Left as an
implementation-time choice** — both are compatible with "no `.dat`
compatibility to preserve" (confirmed earlier: project is pre-1.0, no
shipped files exist yet).

Section name budget: `nameSize` is 16 bytes (`format.go`). Longest base
name is `words.dawg` (10 bytes) + `#` + shard index. Comfortably fits
shard indices up to 3–4 digits; not a practical constraint at any shard
count this design anticipates.

A dictionary with no overflow still goes through this path as N=1 (one
shard, section names `suffixes#0`/`paradigms#0`/`words.dawg#0`) — no
special-cased "unsharded" format. This keeps `Open`/`SaveContainer`'s
logic uniform regardless of whether the source dictionary happened to
overflow.

### `ShardingStrategy` (new, extensible)

```go
// ShardingStrategy decides which shard each lemma belongs to, subject to
// each shard's suffix-id space staying within limit.
type ShardingStrategy interface {
    Assign(lemmas []lemmaEntry, limit int) (shardOf []int, numShards int)
}
```

`FillOnDemand` is the first and, for v1, only implementation: walks
`lemmas` in corpus order, accumulating each lemma's suffix strings into
the current shard's working set; once adding a lemma would push the
current shard over `limit` unique suffixes, the current shard is closed
and a new one starts. (This mirrors the `simGreedyFill` simulation used
for the empirical numbers above.)

Adding a second strategy (e.g. paradigm-grouped) later means writing a new
type satisfying this interface — no changes needed to the build pipeline,
container format, or lookup path. Selecting a strategy at CLI level
(`gomorphy_build compile -shard-strategy=...`) is out of scope for this
work (belongs with the already-planned CLI grooming ahead of Stage 18);
v1 hardcodes `FillOnDemand`.

### Build pipeline (`opencorpora.ImportFromXML`)

1. Phase 1 (XML scan → `[]lemmaEntry`) — unchanged.
2. **New**: run the sharding strategy over `lemmas` to get a `shardOf
   []int` assignment.
3. Phase 2 (suffix/paradigm extraction) — same per-lemma logic as today,
   but run once per shard, only over that shard's lemmas, with fresh
   `suffixTexts`/`paradigmsDedup` maps per shard. `tagSet.Add` continues to
   use the single shared `*TagSet`.
4. Phase 3 (DAWG build) — once per shard. v1 builds shards sequentially;
   building shards in parallel goroutines is a straightforward follow-up
   optimization, not required for correctness, and is left for later (the
   per-shard `dawgBuilder` state introduced by the free-list refactor
   doesn't share mutable state across shards, so this should be safe to
   parallelize whenever it's worth doing).
5. Assemble the sharded `Dictionary` (see below) instead of today's single
   `Suffixes`/`Paradigms`/`Words` fields.

### `Dictionary` and lookup

`internal.Dictionary` gains a per-shard grouping instead of single
`Suffixes`/`Paradigms`/`Words` fields — exact shape (e.g. a `Shards
[]Shard` field with `Shard{Suffixes, Paradigms, Words}`) is an
implementation-time detail, not fixed by this spec.

Lookup (`pkg/morphology/parse.go`'s `exact`/`predict`) currently queries a
single `x.d.Words`. With shards, every lookup fans out to all shards
concurrently (one goroutine per shard, per the user's stated preference:
"решается простым параллельным запросом в горутинах к разным словарям и
склейкой результатов") and merges the resulting `[]Reading` slices. Each
shard's own `Suffixes`/`Paradigms` tables resolve that shard's readings
before merging, so the merge step operates on already-fully-resolved
`Reading` values (word/lemma/tag strings), not raw ids.

**`Reading` gains a `Shard int` field** (additive, not breaking): with
multiple shards, `Para`/`Form` alone no longer identify a unique paradigm
(paradigm id 5 in shard 0 and paradigm id 5 in shard 1 are unrelated).
`Word`/`Normal`/`Tag` are unaffected (already fully-resolved strings).
`Shard` is always `0` for the N=1 case, so existing single-shard behavior
and any code that ignores the new field is unaffected.

### Non-goals for this work

- Parallel shard build (noted above as a natural follow-up, not required).
- A second `ShardingStrategy` implementation (paradigm-grouped or
  otherwise) — the interface exists so this can be added later without
  further design work, but v1 ships only `FillOnDemand`.
- CLI selection of sharding strategy.
- Any change to suffix/tag/paradigm id width (uint16 stays).
- pymorphy2 importer changes — pymorphy2's own `paradigms.array` format is
  itself uint16-addressed at the source, so it isn't subject to the same
  overflow risk gomorphy's own OpenCorpora pipeline is; out of scope here.

## Deferred ideas (to `docs/todo.md` backlog, not part of this work)

Recorded per the user's request, not committed to:

- **Adaptive/wide-index build variant for known-large sources.** A
  parallel implementation (or a build-time flag) using `uint32` ids
  instead of sharding, selected when the source is known upfront to be
  large (e.g. an explicit flag, or a cheap pre-pass estimate). Would sit
  alongside the sharding approach rather than replace it — two different
  answers to "what if uint16 isn't enough," chosen per use case.
- **Narrow-index variant (`uint8`) for small/low-cardinality
  dictionaries.** Relevant to Этап 19 thematic/domain dictionaries with a
  small vocabulary and a small custom tag set — could shrink suffix/tag/
  paradigm storage for dictionaries that will never come close to 256
  unique values in any of those dimensions.

## Follow-up noted by the user (unverified, not part of this work)

The user mentioned that once sharding lands, it should become possible to
"fix/verify a bug with lemma loss from OpenCorpora" (formulated in
Russian as "багу с потерей лемм из opencorpora"). This has **not** been
investigated or confirmed in this session — it is not the same as either
of the two already-documented critical bugs (the tag-accumulation bug or
this suffix-overflow bug) unless it turns out to be a symptom of one of
them. Recorded here verbatim as a pointer for after sharding ships; needs
its own investigation before any fix is scoped.

## Testing

(Detailed test plan belongs in the implementation plan, not this spec —
noting the shape here.)

- Unit: `FillOnDemand.Assign` on a synthetic small lemma/suffix set with a
  tiny `limit`, asserting shard boundaries land where expected.
- Unit: sharded `ImportFromXML` roundtrip (build → save → open → lookup)
  on a small multi-shard synthetic fixture, verifying every inserted
  wordform is still found post-sharding and `Reading.Shard` is populated
  correctly.
- Regression: `TestImportFromXMLTooManySuffixes` (added during the review
  triage session) currently asserts a hard error past 65536 unique
  suffixes — this test's expected behavior **changes** once sharding
  lands (it should succeed with 2 shards instead of erroring); update it
  rather than deleting it, so the "overflow is handled, not silently
  ignored" property stays covered.
- Integration: full `.data/opencorpora/dict.xml` compiles successfully
  end-to-end (currently fails) and every word tested in this session's
  manual checks still resolves.
- `go test ./... -race` green throughout, matching this session's
  established verification discipline.

## Risks

- Parallel shard lookup introduces goroutines into a previously
  single-threaded read path — needs `-race` coverage specifically
  exercising concurrent `Parse`/`Lookup` calls across shards, not just
  sequential correctness.
- Two-pass or multi-shard build changes `ImportFromXML`'s memory profile
  (multiple concurrent per-shard maps/builders if shard build is later
  parallelized) — worth a peak-RSS sanity check against the current
  ~4.5 GB figure observed for the unsharded build once shard build lands,
  even though v1 builds shards sequentially.
- The `Reading.Shard` field, while additive, is still a public API surface
  change for the 1.0.0 release — worth flagging explicitly during
  implementation review rather than letting it slip in as an
  implementation detail.
