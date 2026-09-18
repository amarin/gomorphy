# The cost of a full walk over pymorphy2's `words.dawg` for recompiling to a dense alphabet

**Date:** 2026-09-16
**Status:** the hypothesis is confirmed
**Verified by:** Aleksey Marin (a session with Claude Code)
**Related documents:** [0004-dawg-dense-alphabet-with-payload.md](0004-dawg-dense-alphabet-with-payload.md), the "Dense alphabet in production: discussion notes (paused, unresolved)" section in [../todo.md](../todo.md)

## Context

While discussing rolling the dense 1-byte DAWG alphabet out to
production (2026-09-15/16), one of the two options — "drop direct
support for pymorphy2's raw `words.dawg` and recompile pymorphy2
dictionaries through the same pipeline as OpenCorpora" — was deferred
from immediate implementation with the note: "this is genuinely a
large new chunk of work (a full-fledged 'pymorphy2 -> gomorphy-native
recompiler'), bigger than adapting the read path itself for 2 forms."
No decision on the path was made, specifically because of this
effort estimate.

The key step of such a recompiler is to read the **entire** pymorphy2
dictionary (every "wordform -> (paradigm_id, form_id)" pair from
`words.dawg`), in order to run each wordform's text through
`DenseAlphabet.Encode` and rebuild the DAWG. Before this session, there
was no place in the codebase that walked `words.dawg` in full — only
targeted queries via `SimilarItems` (`pkg/morphology/parse.go:87`) for
a specific word.

The current pymorphy2 import (`pkg/morphology/importers/pymorphy2/import.go:59-69`)
reads `Paradigms`/`Suffixes`/`Prefixes` ready-made from
`paradigms.array`/`suffixes.json`/`paradigm-prefixes.json` — these
tables aren't computed from word text (unlike the OpenCorpora import,
where `Suffixes`/`Prefixes` are built via LCP from the same variables
as the DAWG key, `pkg/morphology/importers/opencorpora/import.go:150-234`).
In `words.dawg` the key is literally the wordform's text, and the
payload is 4 bytes big-endian (`para uint16`, `form uint16`), see
`pkg/morphology/parse.go:199-206` (`Dictionary.reading`).

`pkg/morphology/internal/dawg.go` already has a private `completer`
type (`start`/`next`, dawg.go:290-354) — a sub-automaton walker over
the `guide`. It's used in the already-exported `DAWG.ValuesForIndex`
(dawg.go:231-239) and indirectly confirmed by `TestDebugStress`
(`zz_debug_stress_test.go`), which builds a DAWG from 33 keys (11 words
× 3 payload variants) and walks the **whole** tree from the root
(`c.start(0, "")` + a `c.next()` loop), logging the number of keys
found. There's also already `DAWG.ForEachChild` (dawg.go:245-258, a
walk over a node's outgoing edges via the guide) — meaning all the
primitives needed for a full walk are already in production and
already individually test-covered, just not assembled into one full-
walk function.

## Hypothesis

If a walk over the whole `words.dawg` (enumerating every "wordform ->
payload" pair) is written on top of the already-existing exported
methods `DAWG.ForEachChild` and `DAWG.ValuesForIndex`, it will be
orders of magnitude cheaper than the "large new chunk of work"
estimate from the 2026-09-15/16 discussion — because (a) the walking
mechanism itself is already implemented and tested, and the wrapper
over it is trivial (~15-20 lines), and (b) for pymorphy2 there's no
need to recompute `Paradigms`/`Suffixes`/`Prefixes` (unlike for
OpenCorpora), since the DAWG key is independent of those tables (it's
just the wordform's own text). Consequence: a full walk of a real
pymorphy2 dictionary (~3M wordforms) should finish in seconds, not
minutes/hours, and shouldn't require new infrastructure like holding
the entire dictionary in memory beyond structures already being read.

## Experiment

**Data:**
- The real pymorphy2-dicts-ru dictionary, version
  `2.4.417127.4579844`, wheel
  `pymorphy2_dicts_ru-2.4.417127.4579844-py2.py3-none-any.whl`
  (8,212,425 bytes), downloaded from PyPI
  (`https://files.pythonhosted.org/packages/3a/79/.../pymorphy2_dicts_ru-2.4.417127.4579844-py2.py3-none-any.whl`)
  and unpacked via the new `pkg/pymorphy.Loader` (see below) into
  `.data/pymorphy/data/`. The `words.dawg` file — 7,360,520 bytes.
- Additionally (for a separate synthetic pass, see "Deviation from the
  plan" below): the OpenCorpora wordform list, 3,065,312 lines,
  extracted via
  `grep -o '<f t="[^"]*"' .data/opencorpora/dict.xml | sed -E 's/<f t="//;s/"$//' | sort -u`
  (the same command documented in the doc comment of
  `pkg/morphology/internal/alphabet_compare_integration_test.go`).

**Tools:** a throwaway test inside the `pkg/morphology/internal`
package (access is only needed to the already-exported `ReadDAWG`,
`DAWG.ForEachChild`, `DAWG.ValuesForIndex`, `PayloadSeparator` — so the
same walk is reproducible from the `pkg/morphology/importers/pymorphy2`
importer package too, with no access to unexported details). The test
wasn't kept in the tree (only used for this experiment).

**Measurement method:** a recursive walk from the root (`index=0`):
`ForEachChild` at every node; an edge labeled `PayloadSeparator` marks
the end of a wordform, and `ValuesForIndex` on the destination node
gives every payload value for that key; otherwise — recurse deeper,
appending the label byte to the accumulated prefix. Each payload was
decoded as `(para, form)` from two big-endian `uint16`s, the same way
`Dictionary.reading` does. The whole walk's time was measured
(`time.Since`), the number of keys found was counted, and the maximum
`para`/`form` values seen were recorded as a rough consistency check
against `paradigms.array`.

**Reproduction steps:**
```bash
# Fetching the real dictionary (download+unpack steps; see pkg/pymorphy):
go run <a temporary program calling pymorphy.NewLoader("").Sync(false)>
# → .data/pymorphy/data/words.dawg (7,360,520 bytes)

# Walking words.dawg (a throwaway test in pkg/morphology/internal, using
# only ReadDAWG/ForEachChild/ValuesForIndex/PayloadSeparator):
GOMORPHY_WALK_SPIKE_DAWG="$(pwd)/.data/pymorphy/data/words.dawg" \
  go test -tags=integration ./pkg/morphology/internal/ \
  -run TestScratchWalkRealPymorphy2 -v -timeout 120s
```

**Deviation from the plan (an honest record of a methodological
error):** the first pass measured the wrong thing — a synthetic test
built a DAWG "from scratch" out of 3,065,312 OpenCorpora keys with a
random `uint32` payload via `BuildDAWG`, in order to then walk the
resulting tree. The build didn't finish within the 10-minute timeout
(it hit deep recursion in `placer.place`, `dawgbuild.go:316` — a known
property of the placement algorithm at large N, unrelated to the
walking question). This measured the cost of **building** a DAWG from
scratch, not the cost of **walking** an already-built file — and
walking is exactly the operation the recompiler needs (it reads an
already-compiled pymorphy2 `words.dawg`, not a newly-built one). This
pass was discarded, and replaced with a direct walk of the real
`words.dawg` with no rebuilding at all (see above).

## Results

| Variant | Keys | Walk time | Method |
|---|---|---|---|
| synthetic, `TestDebugStress` (a pre-existing test) | 33 (11 words × 3 payload variants) | not measured separately, the `completer` found 33/33 | `completer.start(0,"")`+`next()` |
| **real pymorphy2 `words.dawg`** | **3,064,708** | **570.6 ms** | `ForEachChild`+`ValuesForIndex` (a ~35-line wrapper) |

Additional consistency checks on the real data:
- `payloads shorter than 4 bytes = 0` — no payload turned out
  corrupted/truncated when decoded as `(para,form)`.
- `max para id seen = 3455` — consistent with the "full dictionary:
  ~3000 paradigms" expectation from `TestFullDictStructure`
  (`pkg/morphology/importers/pymorphy2/full_dict_integration_test.go:30`)
  and with the size of `paradigms.array` (857,102 bytes).
- `max form id seen = 232` — a plausible upper bound on the number of
  forms in a single paradigm.
- Separately (with the already-existing importer, unchanged) on this
  same downloaded dictionary: `TestFullDictStructure`/
  `TestFullDictVseReadings` passed green, `Parse("все")` gave 2 keys
  found / 5 readings — confirming that `.data/pymorphy/data/` is fit
  for real further use, not just for this spike.

For scale comparison: the OpenCorpora wordlist on the same machine —
3,065,312 unique wordforms (within 0.02% of the number of keys in
pymorphy2's `words.dawg`) — both dictionaries are the same order of
magnitude, so the numbers found aren't a toy, unrepresentative case.

## Conclusions

The hypothesis was confirmed: a full walk of the real pymorphy2
`words.dawg` (~3.06M "wordform -> (para,form)" pairs) takes **570 ms**,
using only already-existing, individually-tested code (`ForEachChild`,
`ValuesForIndex`), with nothing built from scratch and no access to
unexported details of the `internal` package. This is orders of
magnitude cheaper than the "large new chunk of work" phrasing from the
2026-09-15/16 pause implied — at least for the "read the whole
dictionary" step, which was the source of that estimate.

What this changes in the effort estimate for a
pymorphy2->gomorphy-native recompiler, specifically for `words.dawg`:
1. Walking is not a risk and not a separate task — it's ~30-40 lines
   reusing the existing API.
2. `Paradigms`/`Suffixes`/`Prefixes` aren't recomputed (unlike the
   OpenCorpora path) — meaning there's no risk of the "DAWG key and
   suffix/prefix table desync" class of bug that's already bitten the
   project twice (`paradigmKeyHash`, see the project's roadmap memory).
3. Work left unmeasured by this experiment: (a) run each found
   wordform through `DenseAlphabet.Encode`, (b) rebuild via
   `BuildDAWGWithValues` with the same payload, (c) serialize the
   alphabet table into `.dat`, (d) a round-trip regression — compare
   `Parse()` before/after on a sample of words. The rebuild itself
   (`BuildDAWG`) on 3M keys is already separately documented by the
   harness (`alphabet_compare_integration_test.go`) as "takes minutes"
   — this isn't an open question about walking, and isn't a discovery
   of this session.
4. The question of `prediction-suffixes-N.dawg` (3,095,048 + 32,264 +
   1,544 bytes) and `p_t_given_w.intdawg` (2,884,616 bytes) remains
   open: the same walker technically applies to them (they're also a
   `DAWG`), but their keys aren't the same kind of thing as wordform
   text (word suffixes and `"word:tag"` respectively) — the decision
   of whether to extend the dense alphabet to them needs separate
   consideration, not covered by this experiment.

**Limits of applicability:** measured on one machine, one run, with no
allocation/GC profiling. The number may not be stable under a
different architecture/machine load, but the order of magnitude
(hundreds of milliseconds, not minutes) is unambiguous enough to take
the "this is expensive" argument off the table when deciding on the
pymorphy2 path.

The test data was obtained with new code (`pkg/pymorphy.Loader`, the
download+unpack steps per the "Stage 21" plan in `todo.md`), which
stays in the tree and isn't part of the experiment itself — the
experiment itself used only already-existing internal `DAWG`
primitives.

## Sources

No third-party sources were used beyond what's referenced above: files
in this repository, and the PyPI JSON API
(`https://pypi.org/pypi/pymorphy2-dicts-ru/json`, queried directly in
this session).
