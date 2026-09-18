# Storage redesign rationale (stages 11-18)

> Moved from `docs/todo.md` during the documentation cleanup
> (2026-09-14) — the historical rationale for replacing the internal
> storage format (CSR-trie + exact-hash + pairs) with paradigm + DAWG.
> The decision itself is already implemented; see `docs/en/todo.md` for
> the current status and `implementation/stage-11-*` ... `stage-18-*`
> for each stage's details.

Source material: [pymorphy2 internals](https://pymorphy2.readthedocs.io/en/stable/internals/index.html),
[opennota/morph](https://gitlab.com/opennota/morph) (a Go implementation reading pymorphy2's format).

## PyMorphy2's storage architecture

The key idea: **paradigms + DAWG**.

1. **Paradigms.** Every lemma is split into prefix + stem + suffix. The
   stem is discarded; (prefix, suffix, tag) are encoded as numeric
   indices. The result is an inflection template (a paradigm). For
   Russian: ~3,000 paradigms out of ~400K lexemes. A paradigm is stored
   as `array.array("<H")`: N suffixes + N tags + N prefixes.

2. **The word DAWG.** All words go into a minimized finite automaton.
   Key: `<word>\x00<para_id><form_idx>`. The DAWG merges shared prefixes
   and suffixes. 5 million wordforms take ~7 MB.

3. **Tags and suffixes.** String pools: `suffixes.json` (~5K suffixes),
   `paradigm-prefixes.json` (~3 prefixes), `gramtab-opencorpora-int.json`
   (~1K tags). Stored as JSON string arrays.

4. **Prediction.** Separate DAWGs for 1-5 letter endings -> sets of
   readings. Enables parsing out-of-dictionary words.

5. **Reading it in Go.** The `opennota/morph` library (~500 lines) reads
   pymorphy2's format directly: dictionary+guide arrays for the DAWG,
   binary reads for paradigms.array, JSON for suffixes/tags. е/ё
   handling on the fly.

| Entity | Count | Size |
|---|---|---|
| Paradigms | ~3,000 | ~3-4 MB |
| Suffixes/prefixes/tags | ~6K | ~0.5 MB |
| Words in the DAWG | ~5M | ~7 MB |
| Prediction (3 DAWGs) | — | ~3-4 MB |
| **Total** | | **~15 MB** |

## gomorphy's storage architecture (at the time of analysis, before stages 11-18)

The key idea: **interning + CSR-trie + exact-hash**.

| Entity | Count | On-disk size |
|---|---|---|
| Unique texts (TextData) | 3,065,312 | ~67 MB |
| Exact-hash table | ~4.4M slots x 16 bytes | ~47 MB |
| Pairs (text_id + ancode_id) | 5,393,737 | ~62 MB (raw u32) |
| Trie (CSR) | hundreds of thousands of states | ~30-40 MB |
| Posting lists | 5.4M entries | ~20 MB |
| Lemmas + ancodes | 391K + 876 | ~5 MB |
| **Total on disk** | | **~305 MB** |

## Why there's a 20x gap

| Reason | pymorphy2's savings share | Comment |
|---|---|---|
| Paradigms instead of flat texts | ~40 MB (67->27 MB) | 3K templates instead of 3M texts |
| DAWG instead of CSR-trie | ~15-20 MB | Merging equivalent states |
| Metadata embedded in the DAWG | ~42 MB | PairTexts+PairAncodes not needed |
| No exact-hash | ~47 MB | The DAWG gives O(len) lookup |
| No posting lists | ~20 MB | Metadata lives in DAWG values |

## Why refactoring the current (at the time) code doesn't work

gomorphy's then-current model was **fundamentally different** from pymorphy2's:

1. **Pairs (textID, ancodeID)** are the central storage unit. One text
   can have several ancodes (homonymy). pymorphy2 doesn't need this:
   the DAWG stores `(word -> para_id, form_idx)`, and the tag comes
   from the paradigm.

2. **Posting lists** are attached to trie nodes. pymorphy2 has none:
   the DAWG itself holds `(para_id, form_idx)` as its value.

3. **Exact-hash** is a separate 47 MB table. In pymorphy2, the DAWG
   provides fast lookup with no extra structure.

Incrementally "bolting paradigms onto" the current model wouldn't
deliver the main win (embedding metadata into the graph), and would
instead create a hybrid with none of either model's advantages.

## Why a rewrite is justified

1. **opennota/morph proves out** the DAWG model in Go: ~500 lines,
   reads pymorphy2's format, е/ё handling, prediction. The format is
   simple: dictionary uint32[] + guide byte[].

2. **The XML pipeline is reused.** `internal/xmlscan` already reads
   dict.xml — that part doesn't need rewriting. The new importer takes
   xmlscan's events and builds paradigms + a DAWG.

3. **The CLI adapts.** The interactive mode and commands — 90% of the
   code stays. `import` and multi-source support are added.

4. **FT8 (Builder) is kept.** The `AddGrammeme/AddLemma/AddForm` API
   stays — it's the entry point for programmatic population.
