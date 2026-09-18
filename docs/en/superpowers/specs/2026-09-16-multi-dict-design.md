# Multi-dict: `morphology.MultiDictionary`

## Context

`docs/en/todo.md`'s "Multi-dict: known context and open questions" section
(raised 2026-09-15, resolved into this spec 2026-09-16) established: the
library has no registry or aggregation across multiple open dictionaries.
`pkg/morphology/open.go`'s `Open`/`OpenPyMorphy`/`OpenPyMorphyDense`/
`CompileFromXML*` each return an independent `*Dictionary` with its own
lifecycle; nothing connects several of them, and `Reading`
(`pkg/morphology/parse.go:14-22`) / `LemmaRef` (`pkg/morphology/lemma.go:4-9`)
carry only `Shard int` — an index *within* one dictionary, not a
dictionary/language identifier. `docs/en/todo.md` also flagged that `Shard`'s
own design (deliberately left extensible for future multi-dict use, see
`docs/en/superpowers/specs/2026-09-14-suffix-sharding-design.md`) establishes
the *pattern* to follow — a small int field, not a design to literally
reuse, since `Shard` already means something else (the shard within one
dictionary's own sharded suffix table) and a single `Reading` needs both
pieces of information at once when it comes from a multi-dict lookup.

The originally-cited motivating scenario was per-language dictionaries
(one dictionary per language, each with its own dense alphabet, instead of
one dictionary with a wide 2-byte alphabet over a combined corpus — see
`docs/en/research/0004-dawg-dense-alphabet-with-payload.md` and the "Dense
alphabet in production" thesis in `todo.md`). The brainstorming session that
produced this spec confirmed the actual scope is wider: any combination of
N dictionaries the caller wants opened together — including two
dictionaries of the *same* language (a main dictionary plus a custom/
supplementary one, for example), not just one-per-language. This rules out
using language code as the sole identifier.

A prerequisite gap was found and closed in the same session: `BuildInfo`
(`pkg/morphology/buildinfo.go`) has had a `SourceVersion` field since it was
introduced, documented as "the source data's version/revision," but **no
importer ever populated it** — `Source` was always set
(`"pymorphy2"`/`"opencorpora"`), `SourceVersion` was always empty. Fixed in
two prep commits on `master` before this spec: `internal/xmlscan` gained
`Handler.OnDictionaryRoot(version, revision []byte)`, wired into
`opencorpora`'s importer as `Dictionary.Info.SourceVersion =
"<version>/<revision>"` from `dict.xml`'s root `<dictionary version="..."
revision="...">` tag; `pymorphy2`'s importer now reads the dictionary's own
`meta.json` (format: an array of `[key, value]` pairs, not a plain
object — pymorphy2's own on-disk convention, verified against the real
`pymorphy2-dicts-ru==2.4.417127.4579844` corpus, `source_version` "0.92" /
`source_revision` "417127") into the same field, same `"<version>/<revision>"`
format. Both leave `SourceVersion` empty (not an error) when the
attribute/key is absent, matching `BuildInfo`'s existing "all fields
optional" contract.

## Decision

**A new wrapper type, `morphology.MultiDictionary`, not a set of free
functions.** It holds `[]*Dictionary` (dictionaries the caller already
opened through the existing `Open`/`OpenPyMorphy`/etc. entry points — the
wrapper does not open anything itself) and becomes the single point for
`Parse`/`Lemma`/`Close` across the whole set, mirroring how `*Dictionary`
itself is already the single point for one dictionary's lifecycle.

**Dictionary identity on a `Reading`/`LemmaRef` is a small int index
(`Dict int`), not a string.** Same shape as the existing `Shard int` —
cheap to carry on every returned reading (there can be many). The
human-readable composite identity (`Source`/`SourceVersion`, the "identifier
from these fields" the user proposed) is looked up separately, by index,
through `MultiDictionary.DictInfo(i int) *BuildInfo` — not duplicated onto
every `Reading`. A dictionary with no `Info` section (hand-built via the
Builder API, never `SaveTo`'d) makes `DictInfo` return `nil` for that
index; the caller decides the fallback (e.g. display the index alone). No
uniqueness is enforced across dictionaries' composite identities — two
dictionaries can share the same `Source`/`SourceVersion` (e.g. the same
pymorphy2 corpus opened twice under different paths); the int index alone
is what disambiguates their readings, always.

**Overlap between dictionaries: return everything, tagged, no merge logic.**
When a word resolves in more than one dictionary, `MultiDictionary.Parse`
returns every reading from every dictionary that found the word, each
tagged with its `Dict` index, concatenated in registration order. No
priority-by-open-order, no dedup, no cross-dictionary sorting beyond what
each individual `Dictionary.Parse` already does internally (probability
sort within its own shards). This is deliberately the simplest, most
information-preserving default — the alternative (picking one dictionary's
result and discarding the rest) throws away information a caller might
need, and any smarter merge policy can be layered on top later without
breaking this contract, since nothing is lost by returning everything now.

**Go-API only, no CLI wiring in this increment.** `cmd/gomorphy` keeps its
single `-dict` flag. Same scoping decision already made for the pymorphy2
dense-alphabet work (`docs/en/superpowers/specs/2026-09-16-pymorphy2-dense-recompile-design.md`) — matching precedent, not re-litigated here.

## Design

### `Reading`/`LemmaRef` — new field

```go
// pkg/morphology/parse.go
type Reading struct {
	Word   string
	Normal string
	Tag    string
	Para   uint16
	Form   uint16
	Shard  int
	Dict   int     // NEW: index into the MultiDictionary that produced this
	                // reading (see MultiDictionary.DictInfo); always 0 for a
	                // reading from a plain Dictionary.Parse call.
	Prob   float64
}
```

```go
// pkg/morphology/lemma.go
type LemmaRef struct {
	Normal string
	Tag    string
	Para   uint16
	Shard  int
	Dict   int // NEW: same meaning as Reading.Dict
}
```

Both are additive fields — every existing call site that constructs a
`Reading`/`LemmaRef` today does so with keyed struct literals (verified:
`parse.go`'s `readingForm`, `lemma.go`'s `Lemma`), so the new field's zero
value (`0`) requires no changes to existing construction sites; only
`MultiDictionary.Parse`/`Lemma` (below) ever set it to something else.

### `MultiDictionary` — new file `pkg/morphology/multidict.go`

```go
package morphology

import (
	"errors"
	"sync"
)

// MultiDictionary — a set of independently opened dictionaries, queried as
// a single whole. Each *Dictionary in the set retains its own lifecycle
// (mmap etc.) — MultiDictionary itself opens or imports nothing, it only
// aggregates Parse/Lemma and owns closing the whole set at once.
type MultiDictionary struct {
	dicts []*Dictionary
}

// NewMultiDictionary wraps already-open dictionaries into a single set.
// The order of dicts fixes the indexing of Reading.Dict/LemmaRef.Dict and
// the order in which Parse/Lemma results are concatenated — both always
// follow registration order and are never re-sorted.
func NewMultiDictionary(dicts ...*Dictionary) *MultiDictionary {
	return &MultiDictionary{dicts: dicts}
}

// Len returns the number of dictionaries in the set.
func (m *MultiDictionary) Len() int { return len(m.dicts) }

// DictInfo returns the diagnostic metadata of the dictionary at index i
// (the same index carried by Reading.Dict/LemmaRef.Dict), or nil if the
// index is out of range, or that dictionary has no info section (see
// Dictionary.Info).
func (m *MultiDictionary) DictInfo(i int) *BuildInfo {
	if i < 0 || i >= len(m.dicts) {
		return nil
	}
	return m.dicts[i].Info()
}

// Parse parses word across all dictionaries in the set in parallel (one
// goroutine per dictionary — the same pattern Dictionary.exact already
// uses for shards within a single dictionary). The result is the
// concatenation of each dictionary's Parse in the set's registration
// order, with Reading.Dict set; there is no sorting or deduplication
// across dictionaries beyond what each Dictionary.Parse already does
// internally. Returns nil if no dictionary produced any readings.
func (m *MultiDictionary) Parse(word string) []Reading {
	results := make([][]Reading, len(m.dicts))

	var wg sync.WaitGroup
	for i, d := range m.dicts {
		wg.Add(1)
		go func(i int, d *Dictionary) {
			defer wg.Done()
			readings := d.Parse(word)
			for j := range readings {
				readings[j].Dict = i
			}
			results[i] = readings
		}(i, d)
	}
	wg.Wait()

	var out []Reading
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}

// Lemma parses word across all dictionaries in the set and returns their
// base forms. Same registration-order-concatenation semantics as Parse.
func (m *MultiDictionary) Lemma(word string) []LemmaRef {
	results := make([][]LemmaRef, len(m.dicts))

	var wg sync.WaitGroup
	for i, d := range m.dicts {
		wg.Add(1)
		go func(i int, d *Dictionary) {
			defer wg.Done()
			refs := d.Lemma(word)
			for j := range refs {
				refs[j].Dict = i
			}
			results[i] = refs
		}(i, d)
	}
	wg.Wait()

	var out []LemmaRef
	for _, r := range results {
		out = append(out, r...)
	}
	return out
}

// Close closes every dictionary in the set (Dictionary.Close is a no-op
// for dictionaries not opened via Open), aggregating all errors via
// errors.Join. After Close the set must not be used, nor its dictionaries.
func (m *MultiDictionary) Close() error {
	var errs []error
	for _, d := range m.dicts {
		if err := d.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
```

`Parse`/`Lemma`'s per-dictionary goroutine fan-out nests inside each
`Dictionary.Parse`'s own per-shard fan-out (`parse.go`'s `exact`) — this is
ordinary nested goroutine use, not a new concurrency pattern; no shared
mutable state crosses dictionary boundaries (each `results[i]` slot is
written by exactly one goroutine, matching `exact`'s existing
`results[shard]` pattern one level up).

## Non-goals

- CLI wiring (`cmd/gomorphy`'s `-dict` flag, output format for a
  multi-dict identifier) — explicit non-goal per the Decision above, a
  separate future increment.
- Any merge/priority/dedup policy across dictionaries' overlapping results
  — the "return everything, tagged" default is deliberately simple and can
  be layered on top later without breaking callers, since nothing is
  discarded now.
- Enforcing unique `Source`/`SourceVersion` identities across a
  `MultiDictionary`'s members — allowed to collide; the int index always
  disambiguates.
- Any change to how a single `*Dictionary` behaves standalone — `Reading`/
  `LemmaRef`'s new `Dict` field is zero-valued and unused outside
  `MultiDictionary`, so `Dictionary.Parse`/`Lemma` called directly are
  unaffected.
- 2-byte DAWG alphabet, `.dat` serialization of `Alphabet`, `fuzzy.go` — all
  separate, already-tracked non-goals from the pymorphy2 dense-recompile
  work, untouched here too.

## Testing

- **`Reading`/`LemmaRef.Dict` field**: no dedicated test needed beyond what
  `MultiDictionary`'s own tests already exercise — it's a passive field
  nothing but `MultiDictionary` ever sets.
- **`MultiDictionary` unit tests** (`pkg/morphology/multidict_test.go`,
  reusing `fixture_test.go`'s `buildFixtureDir`/`stdWords` helpers to build
  two or more small synthetic pymorphy2 fixture directories with
  deliberately different word sets, opened via `OpenPyMorphy`):
  - A word present in only one dictionary of the set — `Parse`/`Lemma`
    return readings/refs tagged with that dictionary's index only.
  - A word present in two dictionaries of the set (e.g. the same word
    added to both fixtures) — both dictionaries' readings come back,
    tagged with their respective indices, concatenated in registration
    order (assert the `Dict` sequence, not just total count).
  - A word present in none — `nil`.
  - `DictInfo(i)` returns the right `Source` (and `SourceVersion`, using a
    fixture whose `meta.json` sets it — see the pymorphy2 prep commit's own
    test for the pattern) for each registered index; an out-of-range index
    returns `nil`.
  - `Len()` matches the number of dictionaries passed to
    `NewMultiDictionary`.
  - `Close()` on a set built from non-mmap (`OpenPyMorphy`) dictionaries
    returns `nil` (each member's `Close` is a no-op, already covered by
    `Dictionary.Close`'s own tests). `errors.Join`'s aggregation itself is
    stdlib and not re-tested here — no need to fabricate a failing
    `Dictionary.Close` just to exercise it.
  - Zero-dictionary `MultiDictionary` (`NewMultiDictionary()`) — `Parse`/
    `Lemma` return `nil`, `Close` returns `nil`, `Len()` is `0`. Not a
    contrived edge case: it is what a caller gets from filtering an empty
    or all-failed-to-open list before construction.
- `go test ./... -race` and `go build -tags=integration ./...` green;
  `golangci-lint run ./...` clean on touched packages (the same 2
  pre-existing, unrelated issues elsewhere in the repo are not this
  work's concern).
