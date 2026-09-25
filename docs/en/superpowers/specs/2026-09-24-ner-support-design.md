# NER support: known-word flag, in-memory open, case/ё consistency, tag helpers, lexeme forms, Parse performance

## Context

A new module, **lexicon** (`~/dev/mine/lexicon`, spec
`docs/specs/2026-09-24-lexicon-design.md`), builds text analysis and
dictionary NER on top of gomorphy. Its first host is genodex (search
normalizer P1, NER E9). genodex's P1 plan already listed two blocking
requests ("Task 0": `Reading.Predicted`, `OpenBytes`) and two desirable ones
(content hash without `info`, `CharPolicy` in `BuilderOptions`). A review of
gomorphy 1.1.0 source for NER needs found more gaps, some of which are bugs
rather than features.

Scope criterion (the project's own): add API only where the caller **has no
workaround**, or where the data already lives in the format and only access
is missing. Convenience wrappers and domain logic (aliases, entities, rules,
hot reload orchestration) stay out — they belong to lexicon.

Findings driving this spec (verified on 1.1.0 source and real `.dat` files):

- `Parse` silently falls back to prediction (`exact` → `predict`); a caller
  cannot tell a dictionary reading from a guess. `хлѣбъ` in the pymorphy
  dictionary yields four `Fixd` readings indistinguishable from real ones.
  Prediction exists for pymorphy2 and Builder/TSV dictionaries (every built
  dictionary gets `BuildPrediction`), so a small user dictionary "predicts" a
  lemma for almost any word.
- `Parse` lower-cases its input; `Builder.AddForm`/`AddLemma` and `ImportTSV`
  store words verbatim, so a form added as «Москва» is never found as a
  dictionary reading — `Parse` returns it only as an indistinguishable prediction.
  `Fuzzy`/`FuzzyTop` lower-case nothing and ignore `CharPolicy` (е/ё cost one
  edit; «Москва» finds nothing, «москва» does).
- `CharPolicy` is an `internal` type: `UniMorphOptions.CharPolicy` can only be
  `nil` from outside; `BuilderOptions` has no field (Builder defaults to
  Russian е→ё).
- `Open` verifies an xxh3 checksum of the whole file and mmaps it; there is no
  way to open a dictionary embedded with `//go:embed` without writing it to
  disk.
- `SaveTo` writes `BuiltAt` into the `info` section, so re-saving identical
  content changes the file hash; hosts that version dictionary sets by file
  hash re-index needlessly.
- Tags are opaque strings; `tagmap` maps POS/case/number/… but puts
  `Name/Surn/Patr/Geox/Orgn/Abbr` into `Unmapped`. Hosts parse tag strings
  themselves (`"NOUN,anim,masc,Sgtm,Surn sing,ablt"`, `"N;GEN;SG"`).
- No lexeme access: `Forms`/`Inflect` are planned (todo, "after 1.0") but the
  paradigm tables are not reachable from outside.
- `Parse` spawns one goroutine per shard even for unsharded dictionaries and
  allocates per rune (`alphabet.Encode(string(r))`), per value (base64 decode)
  and per probability key; there are no benchmarks.
- Builder groups entries by lemma text only: homonymous lemmas of different
  parts of speech («знать» noun / infinitive) merge into one paradigm.
- Builder/ImportTSV build only a 1-byte dense alphabet and fail hard beyond
  254 distinct runes, while `Merge` already falls back to 2 bytes.
- `go.mod` requires `go 1.27.1`; consumers (genodex 1.26.4, nerman 1.22) must
  upgrade their toolchain just to import the library.

## Decision

Two releases.

**1.2.0 — unblock lexicon/genodex (small, mostly fixes):**

- A. `Reading.Predicted` + `LemmaRef.Predicted` + `Dictionary.IsKnown` /
  `MultiDictionary.IsKnown`.
- B. `OpenBytes(data []byte)`.
- C. Consistent case handling: Builder/ImportTSV lower-case; Fuzzy lower-cases.
- D. Public `CharPolicy`: exported type, `BuilderOptions.CharPolicy`,
  `UniMorphOptions` usable from outside, applied in `Fuzzy`.
- E. `Dictionary.ContentHash()`.
- F. Minimal `go` directive; document `Close` lifecycle.

**1.3.0 — quality of life for NER (natural extensions of the core):**

- G. Tag helpers: `HasGrammeme`, `Grammemes`, `POS`.
- H. Lexeme access: `Forms`, `Inflect` (the planned todo item).
- I. `Parse` performance: single-shard fast path, fewer allocations,
  `ParseAppend`, benchmarks.
- J. Builder keeps homonymous lemmas of different POS apart.
- K. Builder/ImportTSV fall back to a 2-byte alphabet.

**Not doing** (belongs to lexicon, or no consumer): public DAWG / step
iteration API, phrase dictionaries, synonyms (Stage 20) as a NER dependency,
dedup in `MultiDictionary`, reference-counted `Close`, prediction for
OpenCorpora/UniMorph imports, probability building, hyphen/shape heuristics,
Windows mmap (stays in backlog), MCP (rejected earlier).

## Design

### A. Known-word flag — `pkg/morphology/parse.go`, `lemma.go`, `multidict.go`

```go
type Reading struct {
    // … existing fields …
    // Predicted is true when the reading was produced by suffix prediction
    // (the word is absent from the dictionary), false for dictionary readings.
    Predicted bool
}
type LemmaRef struct {
    // … existing fields …
    Predicted bool // true when every reading behind this lemma was predicted
}
// IsKnown reports whether word (lower-cased, CharPolicy applied — same
// lookup as Parse) has at least one dictionary reading. It never predicts.
func (x *Dictionary) IsKnown(word string) bool
func (m *MultiDictionary) IsKnown(word string) bool
```

- `predictForPrefix` sets `Predicted = true`; `exact` leaves it false.
- `IsKnown` runs only the exact path (no allocation of readings beyond what
  `SimilarItems` needs; a dedicated early-exit walk is acceptable).
- CLI `lookup` prints `(predicted)` next to predicted readings.

### B. In-memory open — `pkg/morphology/open.go`

```go
// OpenBytes opens a dictionary in gomorphy binary format (as written by
// SaveTo) from data. data is not copied and must outlive the Dictionary;
// Close releases nothing. The checksum is verified like Open.
func OpenBytes(data []byte) (*Dictionary, error)
```

- Shares the container parser with `Open`; the mmap layer becomes one source of
  bytes among two. Works on Windows (no mmap involved).
- Alignment: DAWG sections are zero-copy when aligned; `//go:embed` data has
  no alignment guarantee, so misaligned sections are copied (as `Open` already
  does for misaligned files). Document it.

### C. Case consistency — `builder.go`, `import_tsv.go`, `fuzzy.go`

- `Builder.AddForm(word, lemma, tag)` and `AddLemma(normal, tag)` lower-case
  `word` and `lemma` (`strings.ToLower`). `ImportTSV` does not go through
  `AddForm` and lower-cases on its own.
- `Fuzzy`, `FuzzyTop` (and `MultiDictionary` variants) lower-case the query.
- Behaviour change for dictionaries built from mixed-case input: previously
  unreachable forms become reachable. CHANGELOG: "Fixed", not "Changed".
- Stored readings keep whatever case the dictionary has (always lower after
  this change for built dictionaries).

### D. Public CharPolicy — new `pkg/morphology/char_policy.go`

```go
type Substitution struct{ From, To rune }
type CharPolicy struct{ Substitutions []Substitution }
func RussianCharPolicy() *CharPolicy     // е→ё
func NoCharPolicy() *CharPolicy          // explicit "none" (nil keeps language default)

type BuilderOptions struct {
    Language string
    Source   string
    CharPolicy *CharPolicy // nil → language default: "ru" or "" → е→ё, other → none
}
```

- Default by language (owner decision 2026-09-24, G4): with `CharPolicy` nil,
  Builder and ImportTSV apply е→ё only for `Language` "ru" (an empty
  `Language` means "ru"); any other language gets no substitutions. Today every
  built dictionary gets е→ё regardless of language — a behaviour change for
  non-"ru" built dictionaries (CHANGELOG "Changed").

- `CharPolicy`, `Substitution` are exported as **type aliases** of the internal
  types (`UniMorphOptions` is an alias of `unimorph.Options`, and `unimorph`
  cannot import `morphology` — import cycle). The existing
  `UniMorphOptions.CharPolicy` field becomes usable from outside with no type
  change. The substitution stays one-way, like `Parse` (е in the query matches ё
  in the dictionary).
- `Fuzzy` applies the dictionary's policy: a substitution pair costs 0 in the
  Levenshtein row (the automaton already walks runes; the policy adds an
  alternative edge).
- Pre-reform orthography (ѣ→е, final ъ) is **not** modelled as CharPolicy: it
  is text normalization done by lexicon before calling gomorphy, and final-ъ
  removal is not a substitution.

### E. Content hash — `pkg/morphology/content_hash.go`

```go
// ContentHash returns a stable hex digest of the dictionary's content
// sections, excluding "info" (BuiltAt, Source…). Two dictionaries with equal
// ContentHash parse every word identically.
func (x *Dictionary) ContentHash() string
```

- xxh3-128 over the section payloads in container order, skipping `info`;
  each section framed by name and length. `Dictionary` does not keep the
  container, so the hash re-encodes sections through the `SaveTo` encoder and is
  cached (`sync.Once`).

### F. Toolchain and lifecycle

- Set the `go` directive by the policy **"current Go minus two minor versions"**
  (owner decision G3, 2026-09-24): with Go 1.27 current → `go 1.25.0`, plus a
  `toolchain` line for development. At 1.25 no dependency downgrades are needed
  (`x/sys` via the CLI's `x/term` requires 1.25) and `strings.SplitSeq` (1.24)
  stays. The directive is revisited on each minor Go release. Planning found the
  technical floor is 1.22 (set by `xxh3`) — not used, by policy.
- `Close` doc comment: it must not be called while other goroutines may still
  call methods on the Dictionary — in-flight `Parse`/`Fuzzy` read the mapping,
  and unmapping it crashes the process. Returned strings are expected to be
  copies that stay valid (verify during planning; if any string aliases the
  mapping, copy it). Callers that swap dictionaries at runtime (lexicon) retire the old
  one only after in-flight calls finish.
- Optional (decide during planning, see open question): move `cmd/gomorphy`
  into its own module so consumers' `go.sum` do not list cobra/readline/zap.

### G. Tag helpers — `pkg/morphology/tag.go` (1.3.0)

```go
// Grammemes splits a native tag into grammeme tokens: separators ',', ' ',
// ';' (covers OpenCorpora/pymorphy2 "NOUN,anim,masc,Surn sing,ablt" and
// UniMorph "N;GEN;SG").
func Grammemes(tag string) []string
func HasGrammeme(tag, g string) bool     // no allocation
func POS(tag string) string              // first grammeme
func (r Reading) HasGrammeme(g string) bool
```

- Native tokens only; `tagmap` is not extended with Name/Surn/Patr/Geox
  (UniMorph Schema has no such dimensions).

### H. Lexeme access — `pkg/morphology/lexeme.go` (1.3.0)

```go
// Forms returns every form of r's lexeme (paradigm of r.Para in r.Shard,
// stem derived from r.Word and r.Form), in paradigm order.
func (x *Dictionary) Forms(r Reading) []Reading
// Inflect returns forms of r's lexeme whose tags contain all grammemes in
// want, best (fewest extra differences from r.Tag) first.
func (x *Dictionary) Inflect(r Reading, want ...string) []Reading
// MultiDictionary: dispatch by r.Dict.
```

- Predicted readings: forms are generated from the predicted paradigm and keep
  `Predicted = true`.
- Implements the existing todo item; a short design note replaces the sketch
  there.

### I. Parse performance (1.3.0)

- Single-shard dictionaries: no goroutine/WaitGroup.
- Encode runes into a stack buffer (`EncodeRune` API on the alphabet) instead
  of `Encode(string(r))`.
- Decode payload values without base64 string allocations (decode into a
  fixed array).
- Probability lookup without `key+":"+tag` concatenation (byte buffer).
- `func (x *Dictionary) ParseAppend(dst []Reading, word string) []Reading`.
- Benchmarks: `BenchmarkParseKnown`, `BenchmarkParsePredicted`,
  `BenchmarkLemma`, `BenchmarkFuzzyTop` on the test dictionaries; numbers
  recorded in the implementation write-up. Targets from
  `docs/en/implementation.md` (Parse < 10 µs, prediction < 50 µs) become
  checked facts.

### J. Builder homonyms (1.3.0)

- Group entries by `(lemma, POS(tag of the lemma form))` instead of lemma text
  alone; entries with empty or POS-less tags keep today's behaviour.
- Group by a **part-of-speech class** (VERB/INFN/PRTF/PRTS/GRND → verb,
  ADJF/ADJS/COMP → adjective …), not the raw first grammeme, or every verb would
  split. `AddForm` documents it; a test with «знать» (NOUN) and «знать» (INFN)
  yields two paradigms.
- Always on, no option (owner decision 2026-09-24, G5): every Builder/ImportTSV
  build groups by (lemma, POS class).

### K. Builder 2-byte alphabet fallback (1.3.0)

- `RecompileDense` chooses width 2 when the rune set exceeds 254, reusing the
  logic `Merge` already has; the hard error disappears.

## Compatibility

- A, B, E, G, H, I (`ParseAppend`) are additive.
- C changes results only for mixed-case built dictionaries and mixed-case Fuzzy
  queries — previously broken cases.
- D changes the type of `UniMorphOptions.CharPolicy` (was unusable from
  outside); callers passing `nil` compile unchanged. Built dictionaries with a
  non-"ru" `Language` and no explicit policy no longer get е→ё.
- J changes paradigm grouping for built dictionaries with homonymous lemmas —
  new `.dat` files differ; existing files open unchanged.
- The binary format does not change (no format version bump).

## Testing

- A: a word present in the dictionary → `Predicted=false`, `IsKnown=true`; an
  absent word with prediction → `Predicted=true`, `IsKnown=false`; Builder
  dictionaries included; MultiDictionary mixes both.
- B: `OpenBytes(os.ReadFile(p))` parses identically to `Open(p)`; corrupted
  bytes → checksum error; misaligned buffer (offset by 1) works.
- C: Builder with «Москва» → `Parse("москва")` and `Parse("Москва")` find it;
  `Fuzzy("Москва", 0)` finds «москва».
- D: `BuilderOptions{CharPolicy: NoCharPolicy()}` → «елка» does not find «ёлка»;
  default with `Language` "" or "ru" → finds; default with another language
  (Builder and ImportTSV) → does not find; an explicit `RussianCharPolicy()`
  wins over the language; Fuzzy distance ё/е = 0 under Russian policy.
- E: save → reopen → `ContentHash` equal; rebuild same entries → equal; change
  one form → different.
- G–K per section; I via benchmarks and an allocation test
  (`testing.AllocsPerRun` bound).

## Documentation

- `docs/en/library.md`: new API; `docs/ru/library.md` minimal subset.
- `docs/en/todo.md`: mark Forms/Inflect design as done in 1.3.0; add the
  lexicon consumer note to Stage 20 (synonyms not needed by it).
- `docs/en/comparison.md`: fix the claim that OpenCorpora dictionaries have
  prediction.
- CHANGELOG per release; implementation write-up per project convention.

## Open questions

- G1. RESOLVED 2026-09-24: single module, `cmd/gomorphy` stays inside.
- G2. `LemmaRef.Predicted` semantics when a lemma has both exact and predicted
  readings — cannot happen today (Parse returns either exact or predicted), but
  `MultiDictionary.Lemma` can mix dictionaries; proposed: `true` only if all
  readings behind it are predicted.
- G3. RESOLVED 2026-09-24: policy "current Go minus two minor versions" (now 1.25).
- G4. RESOLVED 2026-09-24: Builder/ImportTSV default CharPolicy is chosen by
  language — е→ё only for "ru" (empty `Language` = "ru"), no substitutions for
  other languages (section D).
- G5. RESOLVED 2026-09-24: POS-class homonym grouping (J) is on by default with
  no option to turn it off.
