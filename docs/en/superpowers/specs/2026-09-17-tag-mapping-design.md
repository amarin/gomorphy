# Universal tag mapping: `pkg/morphology/tagmap`

## Context

`docs/todo.md`'s "Универсальный маппинг тегов между словарями" section
(raised 2026-09-16 during the multi-dict design, left unresolved) flagged
that different dictionary sources already describe the same grammatical
meaning with different tag syntax, even though it's formally the same
grammeme set: `pymorphy2` (`gramtab-opencorpora-int.json`) uses
`"NOUN,anim,masc sing,nomn"` (a space before the form-changing part), while
the OpenCorpora importer (`pkg/morphology/importers/opencorpora/import.go`)
always joins grammemes with commas, no spaces. `TagSet.Name`
(`"opencorpora"` vs `"opencorpora-int"`) already records this convention
difference as metadata but does nothing to resolve it.

This blocks two things independently, both real needs surfaced this
session (see `docs/research/0008-dictionary-export-feasibility.md`):

1. `morphology.MultiDictionary.Parse`/`Lemma` (already shipped,
   `docs/superpowers/specs/2026-09-16-multi-dict-design.md`) returns
   `Reading.Tag`/`LemmaRef.Tag` as-is, tagged with the source dictionary's
   index but never normalized — a caller aggregating results from an
   OpenCorpora dictionary and a pymorphy2 dictionary in the same
   `MultiDictionary` cannot compare their tags for the same grammatical
   meaning without doing this mapping itself.
2. A future dictionary-export feature (`0008-dictionary-export-feasibility.md`)
   needs the same normalization to translate a tag from one dictionary's
   native format into another's — this spec covers only the direction
   needed today (native → universal); the reverse (universal → native,
   needed for export) is an explicit non-goal below.

## Decision

**A new standalone package, `pkg/morphology/tagmap`, independent of
`internal.TagSet`.** `TagSet` stays a generic string-interning table with
no knowledge of tag syntax or semantics; `tagmap` owns both. This keeps
`TagSet`'s existing responsibility (interning, `Add`/`ID`/`TagName`) from
mixing with tag semantics, and lets `tagmap` be imported by callers that
never touch `internal` at all (it's a `pkg/morphology/...` subpackage, not
`pkg/morphology/internal/...`).

**The universal tag is a UniMorph feature bundle, not a new bespoke
schema.** `docs/unimorph.md` already analyzed the UniMorph Schema (23
dimensions, feature codes like `NOM`, `SG`, `MASC`) as part of the planned
(not yet started) Stage 16 UniMorph importer, and its §5.4 already
sketches a partial `UniMorph → OpenCorpora` grammeme mapping table in the
other direction. Reusing that schema means a future UniMorph importer
needs no separate mapping at all (its bundles are already in the target
schema) and this package doesn't invent a second competing taxonomy.

**Features are canonically ordered by a fixed dimension order, not
source order.** The whole point of normalizing is that two tags expressing
the same grammatical meaning from different sources compare equal
(`slices.Equal` on `Bundle.Features`) regardless of how each source
happened to order its own grammemes. `Bundle.Features` is therefore always
sorted by `Dimension` (see Design below), not by input order.

**Unknown tokens are preserved as-is, not rejected.** A token with no
entry in a source's mapping table (an opaque grammeme from a future
thematic-dictionary import, Stage 19, or simply a grammeme this table
hasn't been extended to cover yet) goes into `Bundle.Unmapped` rather than
producing an error. `Map` only returns `ok=false` for a structurally
unregistered `dictName` (a caller bug — asking for a source the package
doesn't know about at all), never for tag content it can't fully map. This
mirrors the project's repeated experience of two tag-corruption bugs
(`paradigmKeyHash` dedup, `OnFormEnd` grammeme accumulation, both in
`docs/todo.md`) — a mapping package that silently drops or errors on
partial data would be the same class of risk in a new place.

**Two independent per-source mapping tables (`opencorpora`,
`opencorpora-int`), not one shared table.** Even though both sources
describe the same underlying OpenCorpora grammeme set, their token
spellings are not assumed to be identical without verification — the
tokenizers differ by construction (see below), and collapsing them into
one table on an unverified assumption risks exactly the kind of silent
mismatch this package exists to prevent. If, during implementation,
`gramtab-opencorpora-int.json` turns out to use identical token spellings
to `dict.xml`'s grammemes, the two tables will simply have identical
content — an acceptable duplication, not a design flaw to avoid pre-emptively.

**Scope: `native → universal` only. No `Unmap` (universal → native) in
this increment.** The reverse direction is needed for export, but export
has no concrete consumer yet (see `0008-dictionary-export-feasibility.md`'s
recommended ordering — pymorphy2-round-trip export first, universal export
blocked on this very mapping). Building `Unmap` now would mean designing
around an ambiguity (one universal bundle can correspond to more than one
native tag spelling) with no real caller to validate the choice against.
This is deferred to whenever export is actually scoped.

**No change to `MultiDictionary`, `Reading`, or `LemmaRef`.** Callers that
want a universal tag call `tagmap.Map(dictName, tag)` themselves, passing
`dict.TagSet.Name` and the tag string. This avoids computing a bundle on
every `Parse`/`Lemma` call when the caller doesn't need one, and avoids
threading a new public type through `internal` → public API just for this.

## Design

### Data model (`pkg/morphology/tagmap/bundle.go`)

```go
package tagmap

// Dimension is one of the UniMorph Schema's dimensions of meaning
// (Sylak-Glassman 2016), restricted to the ones with at least one feature
// actually used by a mapping table in this package — not a hardcoded
// enumeration of all 23 dimensions from the spec, most of which have no
// use yet. New dimensions are added here only when a mapping table needs
// them.
type Dimension uint8

const (
	DimPartOfSpeech Dimension = iota
	DimAnimacy
	DimCase
	DimNumber
	DimGender
	DimTense
	DimAspect
	DimMood
	DimVoice
	DimPerson
	// extended as mapping tables grow; order here fixes canonical sort
	// order for Bundle.Features (see dimensionOrder below).
)

// Feature is a single UniMorph feature value within a Dimension (e.g.
// {DimCase, "NOM"}).
type Feature struct {
	Dim   Dimension
	Value string
}

// Bundle is a native tag normalized into UniMorph terms. Features is
// sorted by Dimension (declaration order above) so that two Bundles
// built from different sources compare equal via slices.Equal whenever
// they express the same grammatical meaning, independent of the source
// tag's own token order. Unmapped holds, verbatim, every input token that
// had no entry in the source's mapping table — never merged into
// Features, never dropped.
type Bundle struct {
	Features []Feature
	Unmapped []string
}
```

### Per-source tokenizers and tables (`pkg/morphology/tagmap/opencorpora.go`, `pkg/morphology/tagmap/pymorphy2int.go`)

```go
// tokenizeOpenCorpora splits a dict.xml-style combined tag
// ("NOUN,anim,masc,sing,nomn") into its comma-separated grammeme tokens.
func tokenizeOpenCorpora(tag string) []string

// tokenizeOpenCorporaInt splits a pymorphy2 gramtab-opencorpora-int tag
// ("NOUN,anim,masc sing,nomn" — lemma-part and form-changing part
// separated by a space, each comma-joined) into the same flat token list,
// discarding the structural space (both parts are just grammemes to the
// mapping table).
func tokenizeOpenCorporaInt(tag string) []string

var openCorporaTable = map[string]Feature{ /* ... */ }
var openCorporaIntTable = map[string]Feature{ /* ... */ }
```

Each table starts with a representative subset (parts of speech, case,
number, gender — enough for the cross-dictionary integration test below)
and grows incrementally as gaps are found; it is not required to cover
every grammeme in either source's full grammeme inventory from the first
commit (see Non-goals).

### Public API (`pkg/morphology/tagmap/tagmap.go`)

```go
package tagmap

// sources maps a TagSet.Name to its tokenizer + table.
var sources = map[string]source{
	"opencorpora":     {tokenize: tokenizeOpenCorpora, table: openCorporaTable},
	"opencorpora-int": {tokenize: tokenizeOpenCorporaInt, table: openCorporaIntTable},
}

// Map normalizes tag (as found in a Dictionary's TagSet) into a Bundle,
// using dictName (the dictionary's TagSet.Name) to pick the tokenizer and
// mapping table. ok is false only when dictName is not a source this
// package knows about at all — not when tag contains tokens the table
// doesn't cover (those land in Bundle.Unmapped, see Decision above).
func Map(dictName, tag string) (Bundle, bool) {
	src, known := sources[dictName]
	if !known {
		return Bundle{}, false
	}
	var b Bundle
	for _, tok := range src.tokenize(tag) {
		if f, ok := src.table[tok]; ok {
			b.Features = append(b.Features, f)
		} else {
			b.Unmapped = append(b.Unmapped, tok)
		}
	}
	sortByDimension(b.Features) // stable sort by Dimension per dimensionOrder
	return b, true
}
```

If a single native tag produces two tokens mapping into the same
`Dimension` (not expected from either of the two current sources, but not
structurally prevented), the later one silently wins during the sort/merge
step — documented behavior, not a bug, since neither current source is
expected to trigger it; a future source that does should be caught by
that source's own integration test, not by a generic panic here.

## Non-goals

- `Unmap` (universal → native) — deferred until a concrete exporter needs
  it (see Decision above and `0008-dictionary-export-feasibility.md`).
- Any change to `MultiDictionary`, `Reading`, `LemmaRef`, or `Dictionary` —
  this package is called by application code on top of the existing
  public API, not wired into it.
- A mapping table for a future UniMorph importer (Stage 16) — trivial
  (identity, since UniMorph bundles are already in the target schema) but
  not written until that importer exists.
- Exhaustive grammeme coverage for `opencorpora`/`opencorpora-int` — tables
  grow incrementally; an uncovered token is a normal, non-error outcome
  (`Bundle.Unmapped`), not a blocking gap.
- CLI wiring — no command surfaces `tagmap` in this increment.

## Testing

- Unit, per tokenizer: representative real tag strings (including
  multi-form OpenCorpora tags cited in
  `docs/implementation/code-review-pre-1.0-triage.md`) → expected token
  lists, including the `opencorpora-int` space-separated two-part case.
- Unit, per table: a representative subset of grammemes (case, number,
  gender, part of speech) → expected `Feature`.
- Unit: a token absent from the table → lands in `Bundle.Unmapped`, not an
  error, and does not appear in `Bundle.Features`.
- Unit: an unregistered `dictName` → `ok == false`.
- Unit: canonical ordering — two synthetic tags with the same grammemes in
  different input order → identical `Bundle.Features` (`slices.Equal`).
- Integration: using real OpenCorpora and real pymorphy2 dictionary
  fixtures (or the real downloaded corpora already used by other
  integration tests in this repo), pick word forms with known matching
  grammar and assert `tagmap.Map` produces equal `Bundle.Features` for both
  sources — the one cross-dictionary check this increment can validate
  without an exporter to test against.
- `go test ./... -race` green; `golangci-lint run ./...` clean on the new
  package.
