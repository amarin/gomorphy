# DAWG packing density with a dense label alphabet

**Date:** 2026-09-15
**Status:** the hypothesis (in its original form — permuting the alphabet) was disproven; a separate positive effect was found and confirmed (shrinking the alphabet domain to 1 byte/character)
**Verified by:** Aleksey Marin (asmadews@gmail.com)
**Related documents:** [DAWG build speedup: the free list](../implementation/dawg-freelist-optimization.md), [DAWG minimization fix (a chainSig bug)](../implementation/dawg-minimization-fix.md)

## Context

gomorphy's DAWG is a double-array trie in a dawgdic-compatible format.
Edge labels (`label`) are the raw UTF-8 bytes of a key's
representation, i.e. the label domain is all 256 `byte` values
(0-255), even though only a small subset actually occurs (for Russian
text in UTF-8, each Cyrillic letter is encoded as 2 bytes, so only a
small number of distinct byte values are effectively used).

Double-array edge traversal is implemented in
`pkg/morphology/internal/dawg.go`:

- `FollowByte` (`dawg.go:150`) computes a child's position as
  `next := index ^ off ^ uint32(lbl)` (`dawg.go:155`);
- `ForEachChild` (`dawg.go:245`) walks the guide structure and
  explicitly uses the byte value `0` as the "no child/no next
  sibling" sentinel: `for label != 0 { ... }` (`dawg.go:250`).

Layout construction is in `pkg/morphology/internal/dawgbuild.go`:
`BuildDAWG`/`newDawgBuilder` build a minimized trie (`compileImpl`,
`dawgbuild.go:226`), then `placer.place` (`dawgbuild.go:285`) places
each node in the array `p.dic` (the serialized double-array; written
via `setAt`, `dawgbuild.go:411`) through `unitAt` (`dawgbuild.go:340`).
Finding a free `base` for a node is delegated to `slotAllocator.alloc`
(`pkg/morphology/internal/dawgbuild_freelist.go:127`): free slots are
linked into a doubly-linked free list, and a candidate `base` is
accepted only if `base` itself and every `base^label_i` for the node
being placed's labels are free (`dawgbuild_freelist.go:121-126`, the
doc comment above `alloc`). This is the dawgdic/cedar/Darts technique
(Aoe 1989) — already documented in
[dawg-freelist-optimization.md](../implementation/dawg-freelist-optimization.md).

The current packing density on the full OpenCorpora dictionary
(3,065,312 unique wordforms) hadn't been measured separately from the
overall `.dat` size before this experiment; a previously known fact
(see [dawg-minimization-fix.md](../implementation/dawg-minimization-fix.md))
is that, after the minimization fix, the double-array's density on the
full dictionary was ~51% (two numbers counted before `.dat`
serialization, not in this experiment's terms).

## Hypothesis

If the raw UTF-8 bytes used as edge labels are replaced with a dense,
custom encoding — ids `0..N-1` assigned only to characters that
actually occur (e.g. in decreasing frequency order), keeping byte `0`
reserved — then the double-array's packing density (the fraction of
occupied slots out of the array's size) will increase, because the
free-list allocator will find a compatible `base` faster and with
fewer skipped (reserved but unoccupied) slots for a node with few
outgoing edges.

## Experiment

**Data:** 3,065,312 unique Russian wordforms extracted from the real
OpenCorpora dictionary `.data/opencorpora/dict.xml` (401 MB) via
`grep -o '<f t="[^"]*"' .data/opencorpora/dict.xml`, deduplicated with `sort -u`.

**Tools:** `BuildDAWG`/`newDawgBuilder`+`newPlacer`
(`pkg/morphology/internal/dawgbuild.go`), called directly from a
temporary `_test.go` in `pkg/morphology/internal` (the file isn't kept
in the repository — a one-off experimental script, not a permanent
project tool).

**Measurement method:** for Experiment 1 — the number of slots the
free-list allocator (`slotAllocator`) occupied relative to its
internal capacity after `place`-ing every node. For Experiment 2 — the
length of the actual serialized array, `len(p.dic)`
(`dawgbuild.go`, the same array written via `setAt`), and the fraction
of it that's zero (unused) `uint32` slots; this exact array is what
ends up in the `.dat`.

**Reproduction steps:**
```bash
grep -o '<f t="[^"]*"' .data/opencorpora/dict.xml | sed -E 's/<f t="//;s/"$//' | sort -u > /tmp/wordforms.txt
# then — build a DAWG via BuildDAWG(keys) from a temporary _test.go
# with four byte-alphabet permutation variants (Experiment 1)
# and with 2-byte UTF-8 sequences replaced by 1-byte codes
# by the number of unique code points in the corpus (Experiment 2)
```

### Experiment 1: permuting the byte alphabet (the domain isn't narrowed)

Compared an identity encoding (raw UTF-8 bytes) against four dense
permutations of the same byte domain 0-255: descending frequency,
ascending frequency, byte-value numeric order, a random permutation —
in every case, code `0` is reserved (not assigned to any real byte).

The first attempt (code `0` assigned to the most frequent byte,
`0xd0`) produced a corrupted/undercounted automaton — 690,929 nodes
instead of the expected ~804,264. The cause — a collision with the
"no child/sibling" sentinel in the guide traversal (`ForEachChild`,
`dawg.go:250`: `for label != 0`): assigning `0` to a real character
makes some edges indistinguishable from "no edge". The error was found
from the node-count mismatch and reproduced; the experiment was redone
with codes starting from `1` (`0` stays a reserved sentinel).

### Experiment 2: shrinking the alphabet's domain (2 bytes/character -> 1 byte/character)

On the same set of 3,065,312 keys, the 2-byte UTF-8 sequences of
Cyrillic characters were replaced with single-byte codes: 47 distinct
code points in the corpus -> codes `1..47` (code `0` reserved).

The first measurement mistakenly compared the free-list allocator's
internal capacity (rounded up to a power of two), and both encodings
landed in the same `2^20` bucket — making it look like there was no
difference. The mistake was found and the experiment redone: the
actual serialized array length, `len(p.dic)`, was measured directly,
not the allocator's internal capacity.

## Results

**Experiment 1 (permutations of the byte domain 0-255, after the fix
for code `0`):**

| Variant | Slots occupied | Allocator capacity | Free |
|---|---|---|---|
| identity (raw UTF-8 bytes) | 804,264 | 1,048,576 | 23.30% |
| descending frequency | 804,264 | 1,048,576 | 23.30% |
| ascending frequency | 804,264 | 1,048,576 | 23.30% |
| numeric byte order | 804,264 | 1,048,576 | 23.30% |
| random permutation | 804,264 | 1,048,576 | 23.30% |

All five variants gave an identical number of occupied slots —
804,264 out of 1,048,576 (23.30% free). The permutation had no effect
on density.

**Experiment 2 (narrowing the domain: raw UTF-8 bytes vs. 1
byte/character), measured on the real serialized array `len(p.dic)`:**

| Variant | `uint32` slots (`len(p.dic)`) | Array size | Zero (unused) slots | Fraction zero | Minimized trie nodes |
|---|---|---|---|---|---|
| identity (raw UTF-8 bytes, 2 bytes/character) | 896,480 | 3.42 MB | 350,584 | 39.11% | 804,264 |
| 1 byte/character (47 codes + reserve) | 569,600 | 2.17 MB | 113,481 | 19.92% | 569,547 |

Real array-size savings: ~36.5% (3.42 MB -> 2.17 MB) — larger than the
reduction in the number of nodes in the minimized automaton (804,264
-> 569,547, ~29.2%), because the fraction of "reserved but empty"
slots in the array also dropped (39.11% -> 19.92%).

## Conclusions

The hypothesis in its original form — "permuting/reordering the label
alphabet improves packing density" — is **disproven**: the free-list
allocator (`slotAllocator.alloc`, `dawgbuild_freelist.go:127`) only
requires that `base` and every `base^label_i` be compatible for a
given node's own label set; a label's numeric value has no effect on
how many slots get skipped while searching for a suitable `base` —
collisions are determined purely by the graph's topology (its
branching structure), not by code values. All five permutations of
the fixed byte domain 0-255 on the same graph gave an identical
result — 804,264 occupied slots out of 1,048,576.

The real lever is **not permutation, but narrowing the alphabet's
domain itself**: switching from per-byte UTF-8 encoding (2 bytes per
character for Cyrillic) to a true "1 byte = 1 character" alphabet
reduces both the number of automaton nodes (~29.2%) and, more
strongly, the actual serialized array's size (~36.5%), because the
fraction of reserved-but-unused value slots in the array is also lower
with a smaller label domain.

**Limits of applicability:** only checked on the full OpenCorpora
dictionary (3,065,312 wordforms, 47 unique Cyrillic+Latin+punctuation
code points found in the corpus). The effect wasn't checked on other
languages/corpora with a different number of unique characters, or
with alphabets where UTF-8 takes 1 or 3+ bytes per character — the
magnitude of the win (36.5%) is specific to Cyrillic's 2-byte domain
and isn't guaranteed to carry over linearly to other cases.

**A production rollout would need:** a format-compatible solution,
since `pkg/morphology/internal/dawg.go` (`ReadDAWG`) can read
pymorphy2's original `words.dawg`, where labels are raw UTF-8 bytes.
Switching to a dense 1-byte label alphabet either needs a format
version marker (the alphabet's encoding is stored in the file and
interpreted variably on read), or must only apply to dictionaries
gomorphy itself builds "from scratch" (OpenCorpora/UniMorph import),
not to a direct import of pymorphy2's original `.dawg` files without
re-encoding. Also a mandatory correctness condition (found during the
experiment): code `0` in any such encoding must stay reserved and
never be assigned to a real character, since the guide traversal
(`ForEachChild`, `dawg.go:250`) treats byte `0` as the "no
child/sibling" sentinel.

## Sources

<No third-party sources were used beyond what's already documented in
the project; the free-list allocator technique and its origin
(dawgdic/cedar/Darts, Aoe 1989) are already recorded and weren't
re-verified in this session — see
[dawg-freelist-optimization.md](../implementation/dawg-freelist-optimization.md)>
