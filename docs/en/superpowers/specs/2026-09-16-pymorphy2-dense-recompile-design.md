# pymorphy2 `words.dawg` recompile under a dense 1-byte alphabet

## Context

The dense-1-byte DAWG alphabet (`pkg/morphology/internal/alphabet.go`,
harness: `docs/en/superpowers/specs/2026-09-15-dawg-alphabet-harness-design.md`,
numbers: `docs/en/research/0004-dawg-dense-alphabet-with-payload.md`) shrinks
`words.dawg` by ~36% with no read-path compromise, but nothing wires it into
production yet (`Open`/`Parse`/`save.go` all still assume raw UTF-8 keys).
Production wiring was paused mid-design on 2026-09-15/16 (`docs/en/todo.md`,
section "Dense alphabet in production: discussion notes (paused, unresolved)") on
one genuinely open question: whether to keep pymorphy2's `words.dawg` as a
direct UTF-8 passthrough (`pkg/morphology/importers/pymorphy2/import.go:65-69`,
`d.Words = []*internal.DAWG{words}` with zero transformation) or recompile it
through gomorphy's own dense-alphabet pipeline so the whole library is always
1-byte internally.

The recompiler option was set aside as "a genuinely new, large chunk of work
... bigger than adapting the read path to 2 shapes" — the estimate rested on
"read the whole pymorphy2 dictionary" being expensive/risky. That premise was
checked directly (`docs/en/research/0005-pymorphy2-full-dawg-walk-cost.md`):
walking the real pymorphy2 `words.dawg` (3,064,708 word→(para,form) pairs,
downloaded via the new `pkg/pymorphy` loader) with a ~35-line wrapper over
already-exported `DAWG.ForEachChild`/`DAWG.ValuesForIndex` took **570ms**.
Pymorphy2's `Paradigms`/`Suffixes`/`Prefixes` also don't need recomputing
during recompilation — they're read pre-built from
`paradigms.array`/`suffixes.json`/`paradigm-prefixes.json`
(`import.go:59-63`), independent of `words.dawg`'s key encoding — so the
sync-bug risk class that made the *OpenCorpora* path risky (`Suffixes`
computed from the same `stem`/`suffix` vars as the DAWG key,
`opencorpora/import.go:150-234`) doesn't apply here.

With that premise gone, the user chose to proceed with the recompiler,
scoped narrowly: `words.dawg` only (not `Prediction`/`Probability`, both
unexplored and out of scope), taken all the way to a working `Parse()` (not
just a library object nobody can query), but stopping short of `.dat`
serialization and `fuzzy.go` — both separate, already-identified future
steps from the same paused thesis.

## Decision

**Scope: `words.dawg` only, end-to-end through `Parse()`/`SimilarItems`, no
`.dat` serialization, no `fuzzy.go` changes, no `Prediction`/`Probability`
DAWGs.** A new `pymorphy2.RecompileDense` produces an in-memory
`*internal.Dictionary` whose `Words[0]` is dense-1-byte encoded and whose new
`Alphabet` field records the codec; `Parse()` on that in-memory dictionary
must return identical readings to `Parse()` on the raw
`pymorphy2.ImportFromDir` result. Persisting the recompiled dictionary to a
`.dat` file, wiring `Open()` to detect a stored alphabet, and touching
`fuzzy.go`'s rune-based traversal are explicitly deferred — the paused
thesis's own agreed decision #5 (alphabet serializes into `.dat` "like
`prefixes`") still holds, just not implemented by this increment.

**`ImportFromDir` is untouched; recompilation lives in a new function.**
`ImportFromDir` keeps returning the raw UTF-8 passthrough dictionary
unconditionally — existing tests and any code that wants the original
pymorphy2 binary semantics keep working unchanged. `RecompileDense` calls
`ImportFromDir` internally and transforms its result; it does not replace it.

**The DAWG-walk primitive is public and general, not pymorphy2-specific.**
`DAWG.Walk` goes on `internal.DAWG` now (not scoped inside the `pymorphy2`
package), because the same primitive is what a future multi-dict
merge/comparison feature would need regardless of dictionary type or
alphabet decision (raised by the user as the reason to re-check the walk-cost
premise in the first place). Building it as a one-off inside `pymorphy2`
now and promoting it later would just be redone work.

## Design

### 1. `DAWG.Walk` — new public method, `internal/dawg.go`

```go
// Walk visits every (key, values) pair stored in the DAWG, in trie order.
// It reuses ForEachChild (edge traversal) and ValuesForIndex (payload
// enumeration under a PayloadSeparator edge) — the same primitives
// SimilarItems and ValuesForIndex already use for single-key lookups, just
// exhaustively instead of following a caller-given key. See
// docs/en/research/0005-pymorphy2-full-dawg-walk-cost.md for the validated
// approach and real-corpus timing (3,064,708 keys, 570ms).
func (d *DAWG) Walk(fn func(key string, values [][]byte)) {
	var walk func(index uint32, prefix []byte)
	walk = func(index uint32, prefix []byte) {
		d.ForEachChild(index, func(label byte, next uint32) {
			if label == PayloadSeparator {
				fn(string(prefix), d.ValuesForIndex(next))
				return
			}
			walk(next, append(prefix, label))
		})
	}
	walk(0, nil)
}
```

`prefix` is shared/mutated via `append` across sibling branches within one
recursive call; this is safe because `fn` is only ever invoked with a fresh
`string(prefix)` copy at the point of the call (no slice aliasing escapes
past that copy), and DFS fully finishes each child subtree before the next
sibling reuses the same backing array — the same pattern already implicitly
relied on by `ForEachChild`'s own callback-based traversal.

### 2. `Dictionary.Alphabet` — new field, `internal/dictionary.go`

```go
type Dictionary struct {
	Language    string
	TagSet      *TagSet
	Suffixes    [][]string
	Prefixes    []string
	Paradigms   [][]Paradigm
	Words       []*DAWG
	Prediction  []*DAWG
	Probability *DAWG
	CharPolicy  *CharPolicy
	Alphabet    Alphabet // nil = raw UTF-8 keys (today's behavior, unchanged)
	Info        *BuildInfo
}
```

`nil` is the zero value and means "no encoding" — every existing dictionary
(OpenCorpora, raw pymorphy2 via `ImportFromDir`) keeps working with zero
behavior change. `NewDictionary`'s constructor signature is unchanged;
`Alphabet` is set directly on the struct by `RecompileDense` (mirroring how
`Prediction`/`Probability`/`Info` are already set post-construction by
importers, not threaded through `NewDictionary`'s parameter list).

### 3. Alphabet-aware traversal — `internal/similar_items.go`

`SimilarItems` gains a third parameter. `similarItemsRecursive` replaces its
two `d.FollowRune(...)` call sites with a new small helper:

```go
func (d *DAWG) SimilarItems(key string, pol *CharPolicy, alphabet Alphabet) []Item {
	return d.similarItemsRecursive("", []rune(key), 0, pol, alphabet)
}

// followRuneVia follows one rune from index, honoring alphabet (nil = raw
// UTF-8, today's FollowRune behavior unchanged).
func (d *DAWG) followRuneVia(alphabet Alphabet, r rune, index uint32) uint32 {
	if alphabet == nil {
		return d.FollowRune(r, index)
	}
	code, err := alphabet.Encode(string(r))
	if err != nil {
		return 0
	}
	return d.followBytes(code, index)
}
```

Both call sites inside `similarItemsRecursive` (the `CharPolicy` substitution
branch and the main per-rune loop) switch from `d.FollowRune(x, index)` to
`d.followRuneVia(alphabet, x, index)`. Nothing else in the function changes:
`prefix`/`key`/`foundKey` already track the original decoded text
independent of how bytes were matched internally, so the returned `Item.Key`
needs no decode step regardless of which alphabet was used to walk the trie.

### 4. Call sites — `parse.go`

```go
// exactInShard (queries d.Words[shard] — in scope):
items := dawg.SimilarItems(word, x.d.CharPolicy, x.d.Alphabet)

// predictForPrefix (queries d.Prediction[id] — explicitly out of scope):
for _, it := range x.d.Prediction[id].SimilarItems(wordEnd, x.d.CharPolicy, nil) {
```

The `nil` at the `Prediction` call site is deliberate and commented in
code — `Prediction` DAWGs are never touched by this work, so they must
never receive `x.d.Alphabet` even if a future dictionary sets one.

### 5. `pymorphy2.RecompileDense` — new function, `importers/pymorphy2/`

```go
// RecompileDense imports dir like ImportFromDir, then rebuilds Words[0]
// (and only Words[0] — Paradigms/Suffixes/Prefixes/Prediction/Probability
// are copied through unchanged) under a dense 1-byte alphabet built from
// the dictionary's own wordforms. The result's Parse() must return
// identical readings to ImportFromDir's, for any word the source
// dictionary itself resolves.
func RecompileDense(dir string) (*internal.Dictionary, error) {
	d, err := ImportFromDir(dir)
	if err != nil {
		return nil, err
	}

	var words []string
	var keys []string
	var values []uint32
	d.Words[0].Walk(func(word string, vals [][]byte) {
		words = append(words, word)
		for _, v := range vals {
			if len(v) < 4 {
				continue // matches Dictionary.reading's own guard, parse.go:199
			}
			keys = append(keys, word)
			values = append(values, binary.BigEndian.Uint32(v[:4]))
		}
	})

	alphabet, err := internal.NewDenseAlphabet(1, words)
	if err != nil {
		return nil, fmt.Errorf("pymorphy2: recompile: build alphabet: %w", err)
	}

	encodedKeys := make([]string, len(keys))
	for i, k := range keys {
		enc, err := alphabet.Encode(k)
		if err != nil {
			return nil, fmt.Errorf("pymorphy2: recompile: encode %q: %w", k, err)
		}
		encodedKeys[i] = string(enc)
	}

	dense, err := internal.BuildDAWGWithValues(encodedKeys, values)
	if err != nil {
		return nil, fmt.Errorf("pymorphy2: recompile: build DAWG: %w", err)
	}

	d.Words[0] = dense
	d.Alphabet = alphabet
	return d, nil
}
```

The len(v) < 4 guard mirrors the existing tolerance in
`Dictionary.reading` (`parse.go:199`, `if len(value) < 4 { return
Reading{}, false }`) — recompilation must not be stricter than the reader
it feeds.

`BuildDAWGWithValues(keys []string, values []uint32)` already supports
multiple values per identical key string (each `(key[i], values[i])` pair
becomes its own `key + PayloadSeparator + base64(value)` DAWG entry, so a
word with several `(para,form)` pairs naturally produces several entries
sharing the pre-separator prefix) — this is exactly the real `words.dawg`
shape (see `full_dict_integration_test.go`'s "все: items=2 readings=5"), so
no new low-level DAWG-building logic is needed at all.

## Non-goals

- `.dat` serialization of `Dictionary.Alphabet` or the recompiled DAWG —
  paused thesis decision #5, a separate future increment.
- `Open()` support for reading back a dense-encoded `.dat` — depends on the
  above.
- `fuzzy.go` changes — its rune/UTF-8 traversal assumption is untouched;
  fuzzy search against a `RecompileDense` result is not expected to work
  correctly and is not tested by this work.
- `Prediction`/`Probability` DAWGs — different key shapes (word suffixes;
  `"word:tag"` with ASCII grammeme names), unexplored, explicitly deferred
  per the walk-cost research doc's own open item.
- Any change to the `gomorphy_build` CLI or `.dat` compile pipeline (Stage
  21) — this is a Go-API-only increment, matching the design conversation's
  scope decision.
- A general multi-dict API — `DAWG.Walk` is deliberately built as a
  reusable primitive for that future work, but nothing else here (no
  registry, no merge logic, no dictionary-identifier field) is part of it.

## Testing

- **`DAWG.Walk` unit tests** (`internal/dawg_test.go` or new
  `dawg_walk_test.go`): synthetic DAWG (small, hand-built keys with known
  multi-value entries, mirroring `TestDebugStress`'s shape) — walked keys
  and values match exactly what was built, including a key with multiple
  payload values and a key that is a prefix of another key.
- **Alphabet-aware `SimilarItems` unit tests** (`similar_items_test.go`):
  existing tests continue passing with `alphabet: nil` appended; new cases
  build a small `DenseAlphabet`, encode a tiny synthetic DAWG's keys through
  it, and confirm `SimilarItems(word, pol, alphabet)` finds the same
  logical items `SimilarItems` would find on the unencoded version — plus
  one case exercising the `CharPolicy` substitution branch (е→ё) through a
  non-nil alphabet, since that's the one path combining both.
- **Mechanical signature update**: existing direct `SimilarItems(...)`
  callers in test files
  (`pkg/morphology/importers/pymorphy2/import_test.go`,
  `full_dict_integration_test.go`,
  `pkg/morphology/importers/opencorpora/{real_dict_integration_test.go,import_test.go}`,
  `pkg/morphology/internal/{zz_debug_test.go,dawgbuild_test.go,similar_items_test.go}`)
  all pass `nil` as the third argument — no behavior change, compile-fix
  only.
- **`RecompileDense` round-trip integration test** (new,
  `//go:build integration`, gated on `GOMORPHY_PYMORPHY2_DIR` like the
  existing `full_dict_integration_test.go`): import once via
  `ImportFromDir`, once via `RecompileDense`; for a fixed sample of real
  words (a few thousand covering multiple paradigm shapes, plus explicit
  е/ё-substitution cases and at least one word with a non-empty prefix) —
  `Parse(word)` on both must produce the same set of
  `(Word,Normal,Tag,Para,Form)` tuples (ignore `Prob`/`Shard`, which are
  unaffected by this work). Full-3M-word comparison is possible but not the
  default (`RecompileDense`'s `BuildDAWGWithValues` call is a from-scratch
  build at real corpus scale, already documented elsewhere as
  minutes-order) — sample size configurable via env var, same convention as
  `alphabet_compare_integration_test.go`.
- `go test -race ./...` green; `golangci-lint run` clean on touched
  packages.

## Risks

- **`similarItemsRecursive`'s per-rune `alphabet.Encode(string(r))` call
  allocates on every rune of every query** (one-rune string → `[]byte`
  round trip) where `FollowRune` did not. This is the same cost class the
  existing code already pays for `CharPolicy` substitution lookups; not
  expected to be a measurable regression for interactive `Parse()` calls,
  but not benchmarked by this design — if it matters, `Alphabet` could grow
  a `EncodeRune(r rune) ([]byte, bool)` fast path later without touching
  callers.
- **`RecompileDense` is a from-scratch `BuildDAWGWithValues` at full corpus
  scale** — already known to take minutes (harness precedent), not
  seconds. This is a one-time, explicit, offline operation (like compiling
  OpenCorpora's XML today), not a `Parse()`-time cost, but worth calling
  out so nobody expects it to be as fast as the 570ms *walk* it depends on.
- **`DenseAlphabet`'s 254-symbol ceiling (width 1)** is a hard failure mode
  (`NewDenseAlphabet` returns an error, doesn't truncate) — safe by
  construction, but means `RecompileDense` will simply fail loudly if ever
  pointed at a pymorphy2 dictionary for a language whose corpus exceeds 254
  distinct runes. Not a concern for the current Russian dictionary; worth
  a one-line doc note so a future non-Russian pymorphy2 dictionary doesn't
  hit this as a surprise.
