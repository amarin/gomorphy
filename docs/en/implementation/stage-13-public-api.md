# Stage 13. Public API: Parse, Lemma, Fuzzy

## Stage contents

Implementing the public API on top of Stage 11's `Dictionary`: Parse
(exact lookup + prediction), Lemma (base form), Fuzzy (fuzzy search).
Adapting the CLI.

### Parse (`pkg/morphology/parse.go`)

```go
func (d *Dictionary) Parse(word string) []Reading
```

Logic:
1. `d.Words.find(word)` -> if found, get `(para_id, form_idx)`.
   The DAWG value is encoded as `(para_id << 16) | form_idx`.
2. For each `(para_id, form_idx)`:
   - `paradigm = d.Paradigms[para_id]`
   - `suffix_id = paradigm.Suffix(form_idx)`
   - `tag_id = paradigm.Tag(form_idx)`
   - `prefix_id = paradigm.Prefix(form_idx)`
   - `normal = d.Prefixes[prefix_id] + stem + d.Suffixes[0]` (the base form)
   - `tags = d.TagSet.Names(tag_id)`
   - Probability: if `d.Probability != nil` -> a lookup by `(para_id, form_idx)`
3. If the word isn't found and prediction DAWGs are available:
   - For each `d.Prediction[i]`:
     - `suffix = word[len(word)-i:]` (the last i characters)
     - `d.Prediction[i].find(suffix)` -> a set of `(para_id, form_idx)`
     - The same computations as above

### Lemma (`pkg/morphology/lemma.go`)

```go
func (d *Dictionary) Lemma(word string) []LemmaRef
```

Logic:
1. A DAWG lookup -> `(para_id, form_idx)` for every occurrence.
2. The base form: `d.Prefixes[paradigm.Prefix(para_id)] + stem + d.Suffixes[0]`.
3. The base form's tags: `d.TagSet.Names(paradigm.Tag(para_id, 0))`.
4. Deduplicated by the base form's text.

### Fuzzy (`pkg/morphology/fuzzy.go`)

```go
func (d *Dictionary) Fuzzy(word string, maxDist int) []FuzzyMatch
func (d *Dictionary) FuzzyTop(word string, maxWords int) []FuzzyMatch
```

Logic (a joint traversal of the DAWG and a Levenshtein DFA):
1. A Levenshtein DFA: `[]int` — the current distance column-by-column (byte-wise).
2. A recursive traversal: for every DAWG transition `(byte -> target_state)`:
   - Compute the distance accounting for substitution/insertion/deletion.
   - If the current distance is <= maxDist, continue the recursion.
   - If the DAWG node is final and the distance is <= maxDist, add a result.
3. A rune-level metric (so "ё/е" correctly counts as 1 substitution).
4. `FuzzyTop` — iteratively widens maxDist starting from 1 until enough
   results are found.

### CLI (`cmd/gomorphy/main.go`)

New commands:
```
gomorphy -dict <path> lookup <word>      exact reading
gomorphy -dict <path> lemma <word>       base form
gomorphy -dict <path> fuzzy <word> <k>   fuzzy search
gomorphy -dict <path> top <word> <n>     top N fuzzy matches
```

Interactive mode: `parse`, `lemma`, `fuzzy`, `top`, `import`, `help`.

## Verification (tests)

- Unit test: `Parse("кота")` -> >=1 reading with NOUN,anim,masc,sing,gent.
- Unit test: `Parse("все")` -> >=4 readings (pronoun, verb, etc.).
- Unit test: `Lemma("кота")` -> "кот".
- Unit test: `Fuzzy("кот", 1)` -> contains "кота", "коты", etc.
- Unit test: `FuzzyTop("кот", 3)` -> the 3 nearest words.
- Unit test: prediction for an out-of-dictionary word (a prediction DAWG).
- Integration test: 100 random words -> all found.
- `go test ./pkg/morphology/... -race` — green.

## Manual verification

- `gomorphy -dict pymorphy2.dat lookup кота` -> a correct reading.
- `gomorphy -dict pymorphy2.dat lemma котам` -> "кот".
- `gomorphy -dict pymorphy2.dat fuzzy кот 1` -> a list of words.
- Interactive mode: every command works.

## Implementation (the actual API)

Deviations from the spec (the opennota/morph + pymorphy2 model):

- `Parse` does **not** build normal by blindly gluing `stem+suffix[0]`;
  instead, from the form: `normal = TrimPrefix(word, prefix[form])`,
  then `TrimSuffix(..., suffix[form])`, then `normal = prefix[0]+stem+suffix[0]`;
  for `form == 0`, normal = the word. words.dawg's payload is `2×uint16 BE`
  `(para_id, form_idx)` (the spec incorrectly said
  `(para_idx<<16)|form_idx`).
- Probability: the key is `word + ":" + tag` (the form's grammemes), the
  value is `p_t_given_w / 1e6`; sorted with `sort.SliceStable`
  descending, only when a nonzero probability is present (zero-value
  ties keep the dictionary's own order).
- Prediction mirrors pymorphy2's `KnownSuffixAnalyzer`: prediction
  DAWG values are 6 bytes BE `(count, para, form)`; the word's suffixes
  are tried (up to 5 runes, longest to shortest), with non-productive
  grammemes (NUMR, NPRO, PRED, PREP, CONJ, PRCL, INTJ, Apro) breaking
  once totalCount>1.
- `Lemma` — via `Parse`, the base form is form 0 of the paradigm;
  deduplicated by `(Normal, Tag)`, homonyms (`кот` NOUN/VERB) are kept.
- `Fuzzy` — a banded DP (refactored from `pkg/dictionary/fuzzy.go`), a
  rune-level metric, deduplicated by word, sorted by `(dist, word)`. A
  node's terminal status is checked via the guide
  (`DAWG.HasPayloadChild`), not by an exploratory FollowByte: the probe
  approach catches double-array layout collisions (spurious 0x01 edges
  on prefixes in testdawg).
- `FuzzyTop` — iteratively widens the distance from 0 up to the upper
  bound `len(query) + maxRunes` (maxRunes comes from a DAG traversal
  deduplicated by node at max depth; without dedup, shared suffixes
  would be counted exponentially).
- Public API: a new type `morphology.Dictionary{d *internal.Dictionary}`,
  `OpenPyMorphy` returns `*Dictionary` (the spec/Stage 12 said
  `*internal.Dictionary`), `Dictionary.Language()` was added.
- CLI adaptation (`import`, `-dict`, interactive mode) is **deferred to
  Stage 14**: `-dict` opens a compiled `.dat` (SaveTo/Open), which
  doesn't exist yet in Stage 13; a mock CLI integration over the full
  dictionary once SaveTo exists.
- The roundtrip/binary search over `p_t_given_w` (the prob DAWG is
  accessed via `Find`) is deferred to Stage 14 together with `.dat` encoding.

Files created:
- `pkg/morphology/dictionary.go`, `open.go` (+`open_test.go`) — the wrapper.
- `pkg/morphology/parse.go` (+`parse_test.go`), `lemma.go` (+`lemma_test.go`),
  `fuzzy.go` (+`fuzzy_test.go`), `fixture_test.go` (buildFixture).
- `internal`: `PayloadSeparator` (exported), `DAWG.ForEachChild`,
  `DAWG.HasPayloadChild`.
