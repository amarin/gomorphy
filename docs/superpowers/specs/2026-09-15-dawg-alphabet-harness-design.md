# DAWG dense-alphabet comparison harness

## Context

`docs/research/0001-dawg-alphabet-density.md` found that recoding `words.dawg`'s
edge labels from raw UTF-8 bytes (2 bytes/character for Cyrillic) to a dense
1-byte-per-character alphabet shrinks the serialized DAWG array by ~36.5% and
the minimized trie's node count by ~29.2%, measured on 3,065,312 real
OpenCorpora wordforms **without** the payload suffix that production keys
actually carry (`word + PayloadSeparator + base64(uint32 value)`,
`internal/dawgbuild.go:43-53`).

A same-day follow-up spike (this session, throwaway `_test.go`, not committed)
re-measured with realistic payload-bearing keys on a 102,178-word sample:
36.2% size reduction — the effect holds essentially unchanged. The spike also
surfaced an unexpected, unexplained slowdown: building a DAWG from
payload-bearing keys took far longer than the bare-word case implied by
research 0001 (102K keys: ~121s for the raw-UTF-8 variant, ~41s for the dense
variant — both far slower per-key than research 0001's un-timed but evidently
fast 3M-key no-payload build). Attempts with **sequential** `uint32` payload
values (0, 1, 2, ...) failed to complete within 15 minutes even at 510K and
3.06M keys; switching to random (non-sequential) values let a 102K-key sample
complete. The cause was not isolated — plausibly sequential big-endian values
give alphabetically-adjacent words (which already share trie prefixes) an
artificial extra correlation in their payload suffixes that interacts badly
with the free-list allocator's collision search — but this is a hypothesis,
not a confirmed root cause.

## Why the core DAWG engine needs no changes

The on-disk dawgdic-compatible unit format (`internal/dawg.go:11-20`) reserves
exactly bits 0-7 of each `uint32` unit for the edge label — an 8-bit field is
a hard structural constraint of this format, not a gomorphy choice, and
changing it would break compatibility with `ReadDAWG`/`ParseDAWG` reading
original pymorphy2 `words.dawg` files (raw UTF-8 labels), an existing
requirement research 0001 already flagged.

This constraint turns out not to matter: `FollowByte`/`Follow`
(`internal/dawg.go:150-181`) already walk a key **one raw byte at a time** —
UTF-8 itself is already "a multi-byte alphabet on top of a 1-byte engine."
A 2-byte-per-symbol dense alphabet needs no wider label field; it is simply
two sequential 1-byte trie edges per logical character, invisible to
`dict`/`guide`. The free-list allocator (`internal/dawgbuild_freelist.go`)
confirms this: `slotAllocator.hint` is a fixed `[256]uint32` keyed by a single
label byte, and `fits`/`alloc` operate on a node's actual `labels []byte`
(bounded by real per-node fan-out, never by alphabet size). Alphabet width is
therefore a pure **codec layer at the key-construction boundary** — it does
not touch `dict []uint32`, `guide []byte`, `placer`, or `slotAllocator` at
all.

## Decision

**Scope: harness only, no production wiring.** Build a pluggable `Alphabet`
codec and a repeatable comparison harness inside `pkg/morphology/internal`;
make zero changes to `import.go`, `parse.go`, `open.go`, `save.go`, or the
`.dat` format. Whether/how to ship a chosen alphabet as a real build option
(CLI flag, format version marker, `Open()` support) is an explicit follow-up
decision after this harness produces numbers — not part of this work.

**Widths: 1 and 2 bytes only, no 4-byte variant.** A 2-byte dense alphabet
already addresses up to 65,536 distinct symbols — more than the entire
Unicode BMP, and orders of magnitude above what any realistic corpus
(including a future multi-language UniMorph import) would need. A 4-byte
variant would only matter for an alphabet size that essentially never occurs
in this domain, and for ASCII-heavy text it could be **worse** than UTF-8
itself (4 bytes fixed vs. UTF-8's 1 byte for ASCII). Not worth the added
surface for a harness whose job is to inform a real decision.

**The harness must use realistic, payload-bearing keys.** Research 0001's
own methodology (bare words) undersells the real question — payload is what
production `words.dawg` actually carries, and the spike showed it can matter
for both size (slightly) and build time (a lot, unexplained). Excluding it
would answer a question nobody's actually asking.

## Design

### `Alphabet` interface

New file `pkg/morphology/internal/alphabet.go`:

```go
// Alphabet encodes a string (a DAWG key's word portion) into a byte
// sequence suitable for the DAWG engine, and decodes it back. The engine
// itself is alphabet-agnostic (see the design doc this implements) - an
// Alphabet only changes how many raw bytes represent one logical
// character, never the double-array/guide format.
type Alphabet interface {
	// Name identifies the alphabet for logging/comparison output, e.g.
	// "identity", "dense-1", "dense-2".
	Name() string

	// Encode converts s into the byte sequence to use as DAWG key
	// material. Returns an error if s contains a rune the alphabet
	// cannot represent (e.g. not present in a DenseAlphabet's corpus).
	Encode(s string) ([]byte, error)

	// Decode is Encode's inverse: reconstructs the original string from
	// a byte sequence previously produced by Encode. Returns an error on
	// malformed input (e.g. a length not divisible by a fixed width, or
	// a code with no known rune).
	Decode(b []byte) (string, error)
}
```

### `IdentityAlphabet`

Raw UTF-8 passthrough — today's actual behavior, and the only correct choice
for reading original pymorphy2 `words.dawg` files (out of scope to change,
mentioned only as the harness's baseline arm):

```go
type IdentityAlphabet struct{}

func (IdentityAlphabet) Name() string { return "identity" }
func (IdentityAlphabet) Encode(s string) ([]byte, error) { return []byte(s), nil }
func (IdentityAlphabet) Decode(b []byte) (string, error) { return string(b), nil }
```

### `DenseAlphabet`

```go
// DenseAlphabet maps each rune in a fixed corpus to a code of exactly
// width bytes (1 or 2), reserving code 0 (the DAWG engine's
// guide-traversal sentinel, internal/dawg.go's ForEachChild) and code 1
// (PayloadSeparator, internal/dawg.go:23) so a dense-coded character byte
// sequence can never be confused with either. Codes are assigned in
// ascending rune order for determinism (same corpus -> same codec, byte
// for byte, run to run).
type DenseAlphabet struct {
	width  int // 1 or 2
	codeOf map[rune]uint16
	runeOf []rune // runeOf[code-2] == r for codeOf[r] == code (codes start at 2)
}

// NewDenseAlphabet builds a DenseAlphabet from every distinct rune found
// across corpus, encoding each into width bytes (1: codes 2..255, up to
// 254 distinct runes; 2: codes 2..64517, up to 64516 (254*254) distinct
// runes). Returns an error if corpus contains more distinct runes than
// width can address.
func NewDenseAlphabet(width int, corpus []string) (*DenseAlphabet, error)
```

`Encode`: for each rune in `s` (via `range s`, which already decodes UTF-8),
look up `codeOf[r]`; error if absent; for width 1, append the code as a
single byte; for width 2, append it as a base-254 two-digit encoding where
each digit is offset by +2, so both bytes individually stay in [2,255] -
never `0x00` (the DAWG guide-traversal sentinel) or `0x01`
(`PayloadSeparator`), even though a naive big-endian 16-bit split would let
the high byte fall to `0x00` for any code <= 255. `Decode`: read
`width`-byte chunks, look up `runeOf[code]`; error if `len(b)` isn't a
multiple of `width`, a chunk contains a byte < 2, or a code has no
mapping.

### Harness

New file `pkg/morphology/internal/alphabet_compare_integration_test.go`,
`//go:build integration` (needs the real `.data/opencorpora/dict.xml`, same
convention as `internal/xmlscan/integration_test.go` and this session's
`real_dict_integration_test.go`):

- Load real wordforms the same way research 0001 did: shell out to
  `grep -o '<f t="[^"]*"' dict.xml | sed ... | sort -u` (documented as a
  reproduction step, matching the existing convention of not hiding corpus
  extraction inside Go code) or reuse an already-extracted word list at a
  configurable path (env var, default `/tmp/wordforms.txt`).
- Configurable sample size (env var, e.g. `GOMORPHY_ALPHABET_BENCH_SAMPLE`,
  default a few hundred thousand — full-corpus runs take minutes per arm and
  are opt-in, not the default for a quick `go test` invocation).
- For each of `IdentityAlphabet{}`, `dense-1 := NewDenseAlphabet(1, sample)`,
  `dense-2 := NewDenseAlphabet(2, sample)`: build realistic payload-bearing
  keys (`alphabet.Encode(word) + PayloadSeparator + base64(random uint32)`,
  random values per the spike's finding that sequential ones may be
  pathological), call `BuildDAWG` directly (bypassing `BuildDAWGWithValues`,
  which always uses raw UTF-8 for the word portion), time the build, and
  report slot count / byte size / wall time via `t.Logf`.
- Print a results table at the end (all three arms), so the comparison is
  legible in one `-v` run instead of scattered log lines.

### Testing

- `alphabet_test.go` (plain unit tests, not integration-tagged):
  - `IdentityAlphabet`: `Encode`/`Decode` roundtrip on ASCII, Cyrillic,
    empty string.
  - `DenseAlphabet` width 1 and width 2: roundtrip on a small synthetic
    corpus; reserved codes 0 and 1 never appear in `codeOf`'s value set
    (assert directly on the constructed map, not just black-box behavior);
    `NewDenseAlphabet(1, corpus)` returns an error (not silent truncation
    or wraparound) when `corpus` has more than 254 distinct runes; same for
    width 2 at 64,516; `Decode` on a byte sequence whose length isn't a
    multiple of `width` returns an error, not a panic or silent
    misalignment.
  - Determinism: two `NewDenseAlphabet(1, sameCorpus)` calls produce
    byte-identical `codeOf` assignments.

## Non-goals

- Any change to `import.go`, `parse.go`, `open.go`, `save.go`, or the `.dat`
  binary format — this is explicitly deferred to a future decision.
- A CLI flag, format version marker, or `Open()`-time alphabet detection —
  same reason.
- A 4-byte alphabet variant — see Decision above.
- Explaining or fixing the sequential-value build-time anomaly from the
  spike — the harness uses random values specifically to sidestep it, not
  to diagnose it. If the harness's own timing numbers turn out to still be
  surprising, that becomes a separate follow-up, not part of this work.
- Any interaction with `CharPolicy` (е/ё substitution, `similar_items.go`)
  or query-side encoding for `Parse()`/`Lemma()` — irrelevant while nothing
  outside the harness changes.

## Risks

- Full-corpus harness runs are slow (minutes per arm, three arms) — the
  configurable sample size exists specifically so a routine `go test
  -tags=integration` doesn't take 10+ minutes by default; a full-corpus run
  is an intentional, explicit, opt-in invocation when someone wants the
  final numbers.
- The build-time anomaly from the spike is not understood, only worked
  around (random values). If it reappears in the harness even with random
  values at larger sample sizes, that is real signal worth reporting
  alongside the density numbers — the harness's job is to surface it, not
  suppress it.
- `DenseAlphabet`'s error-on-overflow behavor (more distinct runes than the
  width can address) is a real constraint for width 1 on multi-language
  corpora; the harness will hit this immediately if ever pointed at
  something larger than the current ~45-47-symbol OpenCorpora corpus — this
  is expected and desirable (fail loud, not silently wrap runes onto
  colliding codes).
