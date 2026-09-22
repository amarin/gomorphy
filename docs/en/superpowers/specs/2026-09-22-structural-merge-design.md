# Structural `Merge`: id-preserving dictionary merge

## Context

Stage 19 shipped `morphology.Merge` (spec
[2026-09-21-builder-tsv-merge-design.md](2026-09-21-builder-tsv-merge-design.md)).
It breaks every input into `(word, lemma, tag)` triples and rebuilds the
result with the Builder pipeline (`buildFromEntries`). That works for small
thematic dictionaries built with `Builder`/`ImportTSV`. A review on
2026-09-22 ran it on the real bundled dictionaries (base + a 3-line TSV
overlay, `--mode replace`) and found it unusable on a real base:

| Base | Time | Peak RSS | Size before → after |
|---|---|---|---|
| `pymorphy.dat` | 138 s | 2.8 GB | 16 MB → 41 MB |
| `opencorpora.dat` | 141 s | 2.7 GB | 10 MB → 41 MB |

Review findings this spec fixes:

1. **Prediction is lost.** Regrouping by lemma text computes a fresh LCP
   stem for every lemma and throws away pymorphy2's prefix paradigms
   (`наи-`, `по-`). Suppletive and prefixed forms get whole-word suffixes, the
   suffix space overflows 65 536 and the output is sharded (6 shards seen).
   `BuildPrediction` is a no-op on sharded output, so `Parse("бутявкающий")`
   gives 5 readings on the base and `no readings` after the merge.
2. **Probability is lost.** `p_t_given_w` is dropped, so reading order
   changes (`стали`: `стать` came first, `сталь` does now), and so does the
   "best" reading.
3. **Tag normalization breaks.** The output TagSet is named `merge`, which
   `tagmap.Map` doesn't know, so `tagmap` stops working for the whole merged
   dictionary.
4. **Size and speed.** The flattened paradigms explode the structure: size
   ×2.5–4, time ~140 s, RSS ~2.8 GB.
5. **Multiple overlays behave inconsistently.** `replace`: the first overlay
   replaces the base and later overlays *append* (they should replace again).
   `add`: a word missing from the base collects readings from *every* overlay
   (after O1 adds it, O2 should skip it). Both contradict the original spec's
   "overlays apply in order", and the `MergeReplace` godoc contradicts
   itself.
6. **Docs are stale.** `docs/ru/cli.md` still calls `merge` unimplemented.
   `docs/ru/library.md`, `README.md` and `CHANGELOG.md` don't mention
   Builder/TSV/Merge. The EN docs promise "prediction rebuilt".

Measurements that shaped the design (2026-09-22, Apple Silicon):

| Dictionary | TagSet | Shards | Suffixes (shard 0) | Paradigms | Prefixes | Prediction | Probability | (word,value) pairs | Walk | Words-DAWG rebuild |
|---|---|---|---|---|---|---|---|---|---|---|
| pymorphy.dat | opencorpora-int | 1 | 16 311 | 3 456 | 3 | 3 DAWGs | yes | 5 140 055 | 0.7 s | 23.6 s |
| opencorpora.dat | opencorpora | 1 | 19 375 | 3 559 | 2 | — | — | 5 141 231 | 0.5 s | 23.3 s |
| unimorph.dat | unimorph | 2 | 65 526 / 37 373 | 3 932 / 2 308 | 1 | — | — | 473 517 | 0.06 s | 21.3 s |

The native structures are small: a few thousand paradigms and tens of
thousands of suffixes. Only the words DAWG is big, and rebuilding it
(~23 s) is the unavoidable floor for any merge that touches a shard.

## Decision

Replace the entries-based merge with a **structural, id-preserving merge**.
The base's TagSet, Prefixes, and each shard's Suffixes and Paradigms are
copied **verbatim, ids unchanged**, and grow append-only. Overlay readings
are **remapped** into that id space: each paradigm is translated
form-for-form (suffix text, prefix text, tag name → output ids), then
deduplicated against the target shard's existing paradigms. Because base ids
never change, the base's prediction DAWGs (values are `(count, para, form)`
in shard 0) and probability DAWG (keys `word:tag`) **stay valid as-is**, and
the TagSet keeps the base's name, so `tagmap` keeps working. Only the words
DAWGs of shards whose contents changed are rebuilt.

## Design

### Semantics

- **Word identity** is the exact stored wordform key. There is no
  CharPolicy substitution, so `ёж` and `еж` are different words.
- **Fold over overlays** (fixes #5). `Merge(b, [o1, o2], m)` gives the same
  result as `Merge(Merge(b, [o1], m), [o2], m)`:
  - `MergeAdd`: an overlay word is taken only if it's absent from the base
    **and** no earlier overlay supplied it.
  - `MergeReplace`: an overlay word's readings replace whatever the word had
    so far (base or an earlier overlay); **the last overlay wins**.
- A word taken from an overlay keeps **all** of that overlay's readings for
  it, with lemma and tag reconstructed exactly as the overlay would (same
  paradigm shape, remapped ids).
- In replace mode a replaced word loses **all** of its base readings in
  **every** shard, plus its base probability entries.

### Compatibility checks (public layer)

- `Language` must match between the base and every overlay. Otherwise
  `ErrIncompatibleDictionaries`.
- TagSet vocabularies: when base and overlay TagSet names differ **and both
  are known to `tagmap`** (`opencorpora`, `opencorpora-int`, `unimorph`),
  return `ErrIncompatibleDictionaries`. Mixing two known vocabularies under
  one name would make `tagmap` silently misread one of them. Opaque names
  (`builder`, `tsv`, anything else) are always allowed: their tags are
  appended verbatim, and `tagmap.Map` reports unknown tokens in
  `Bundle.Unmapped`, which is its documented behavior.
- The output TagSet **name is the base's** (fixes #3). A new exported helper,
  `tagmap.Known(name) bool`, backs the check.

### Output layout

- `TagSet`: a deep copy of the base's, with overlay tags appended through
  `TagSet.Add` (dedup by name; `ErrTagSetFull` propagates).
- `Prefixes`: a copy of the base's, with new prefix texts appended (uint16
  cap, error on overflow).
- `Suffixes[i]` / `Paradigms[i]`: a copy of the base's shard `i`; overlay
  suffixes and paradigms are appended to the **target shard**.
- **Target shard**: the last shard. If a paradigm's new suffix texts would
  push it past `mergeSuffixLimit` (= `suffixShardLimit`, 65 536; a `var` so
  tests can lower it), or the shard already holds `paradigmLimit`
  paradigms, a new empty shard is appended and becomes the target. The
  engine allows a word's readings to span shards (`exact` queries every
  shard).
- Paradigm dedup key within a shard: `(suffixIDs, tagIDs, prefixIDs)`. The
  prefix ids are included, unlike `build.go`'s `paradigmKeyHash`, because
  pymorphy2 paradigms do carry prefixes.
- Overlay readings are placed in **sorted word order** for determinism.
- `CharPolicy`, `Language`: the base's. `Info`: `Source="merge"`, plus the
  base's `SourceVersion`/`Description` (unchanged from today).

### Alphabet

The output is always dense (CLI policy):

- If the base has a `*DenseAlphabet` that can encode every overlay word it
  takes, **reuse it**.
- Otherwise build a new one from the base alphabet's runes
  (`DenseAlphabet.Runes()`, new) plus the overlay words' runes. Try width 1
  and fall back to width 2. Every shard is then rebuilt.
- If the base is raw (no alphabet), build one from all base words plus the
  overlay words, and rebuild every shard.

### Words DAWGs: reuse vs rebuild

A shard is **dirty** if it's the target of at least one placed reading,
holds a replaced word, or the alphabet changed. Clean shards reuse the
base's DAWG (cloned, see below). Dirty shards are rebuilt: walk the base
shard, decode keys, drop replaced words, add the placed overlay pairs,
encode with the output alphabet, then `BuildDAWGWithValues`.

### Independence from inputs

`Open` mmaps a `.dat`, and `ParseDAWG` aliases the mapped bytes for the
words, prediction and probability DAWGs (paradigms and strings are
already copied on decode). Anything carried over from an input is
therefore **deep-copied** with a new `(*DAWG).Clone()`, so the result stays
valid after the inputs are `Close`d. That's a contract, and it's tested.

### Prediction (fixes #1)

- **Default: carry the base's prediction DAWGs verbatim** (cloned). They stay
  correct because shard-0 paradigm ids, prefix ids and tag ids never change.
  Overlay words don't contribute. Replaced base words may still count
  towards suffix frequencies, which is harmless and documented.
- **Opt-in `RebuildPrediction`**: build a fresh prefix-0 prediction DAWG
  from the merged shard 0 with `BuildPredictionFrom` (a refactor of
  `BuildPrediction` that takes raw `(word, value)` pairs, so there's no
  second DAWG build). Useful for merging thematic dictionaries with each
  other. It returns `ErrPredictionSharded` if the output has more than one
  shard. Note that it replaces the base's per-prefix DAWGs with a single
  prefix-0 DAWG.
- The CLI exposes it as `gomorphy merge --rebuild-prediction`.

### Probability (fixes #2)

- **Fast path**: nothing removed from the base and no overlay supplies
  probability → the base's DAWG is cloned verbatim (or stays nil).
- Otherwise rebuild: `WalkValues` over the base DAWG, drop keys whose word
  (`key[:LastIndexByte(key, ':')]`) was replaced, and add
  `overlay.Probability.Find(word+":"+tag)` (when > 0) for every reading a
  winning overlay supplied. Build the result with the new
  `BuildIntDAWG(keys, values)`.
- That needs two DAWG primitives that don't exist yet:
  - `BuildIntDAWG`: a dawgdic value DAWG. The builder's `dbNode` gets a
    `value uint32` that is part of the minimization signature, and `place`
    writes `isLeafBit|value` into the value unit. Values must be `< 1<<31`.
  - `(*DAWG).WalkValues(fn(key, value))`: enumerates the leaf nodes.

### Public API

```go
// Merge is unchanged in shape: Merge(base, overlays, mode) ==
// MergeWithOptions(base, overlays, MergeOptions{Mode: mode}).
func Merge(base *Dictionary, overlays []*Dictionary, mode MergeMode) (*Dictionary, error)

type MergeOptions struct {
    Mode              MergeMode
    RebuildPrediction bool
}
func MergeWithOptions(base *Dictionary, overlays []*Dictionary, opts MergeOptions) (*Dictionary, error)

var ErrIncompatibleDictionaries = errors.New("morphology: merge: incompatible dictionaries")
var ErrPredictionSharded        = internal.ErrPredictionSharded
```

`ErrNoEntries` is still returned when the output has no words at all.
`collectEntries` and `mergeTagSetName` are deleted. Builder/`ImportTSV`
keep the entries pipeline, since it's the right tool for building from
scratch.

### Internal layout

- `internal/merge.go` (new): `MergeDictionaries(base, overlays, MergeOptions)`
  with decision → placement → alphabet → shard rebuild → probability →
  prediction.
- `internal/dawgbuild.go`: `BuildIntDAWG`, and a value in `dbNode`.
- `internal/dawg.go`: `WalkValues`, `Clone`, `Empty`.
- `internal/alphabet.go`: `(*DenseAlphabet).Runes`.
- `internal/prediction.go`: `WordValue`, `BuildPredictionFrom`.
- `tagmap/tagmap.go`: `Known`.

## Acceptance criteria

- Synthetic tests: fold semantics for 2+ overlays in both modes;
  pymorphy2-shape fixture base (prediction + probability + `opencorpora-int`)
  merged with a Builder overlay keeps the TagSet name, base `Para` ids,
  `Prob` and OOV prediction; a replaced word loses its probability; overlay
  probability is carried; a Latin overlay word extends the alphabet; a clean
  shard is reused byte-for-byte; target-shard overflow opens a new shard;
  the result survives `Close` of mmap'd inputs; language/tagset mismatch →
  `ErrIncompatibleDictionaries`; `RebuildPrediction` on sharded output →
  `ErrPredictionSharded`.
- Real-data gated test (`GOMORPHY_MERGE_BASE=<.dat>`): add-mode merge of a
  small overlay. For 20 000 sampled base words and a list of OOV words,
  `Parse` results are **identical** (Word, Normal, Tag, Para, Prob, order);
  `TagSetName` is unchanged; output size ≤ base × 1.02.
- On `pymorphy.dat`: merge time ≤ 40 s, peak RSS ≤ 1.5 GB (reported in the
  write-up; the test only logs time).
- `go test ./...`, `go vet ./...`, `golangci-lint run ./...` green.

## Non-goals

- Tag conversion between known vocabularies (e.g. unimorph → opencorpora-int
  through tagmap). It's rejected, not converted.
- Per-overlay modes.
- Recounting base prediction frequencies after replace.
- Changing Builder/`ImportTSV`, or fixing Builder's lemma-LCP suffix
  explosion on large inputs (a separate concern: it only matters when
  building a big dictionary from scratch).
- `split`.
