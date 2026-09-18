# Stage 9. FT6 fuzzy search — DONE

## Stage contents

Increment: `Fuzzy(word, k)`.

- A Levenshtein DFA (no transpositions) over the CSR trie; pruned by k;
  results ordered by (distance, word).
- A rune-level metric, not byte-level: multi-byte runes are enumerated as
  byte chains via `utf8.FullRune` (`Snapshot.Edges` yields CSR windows);
  "дом/дым" = 1 substitution, "ежик/ёжик" = 1 substitution.
- `FuzzyTop(word, maxWords)`: a "nearest N words" query via iterative
  distance widening. `walkTrie` was factored out as a shared traversal;
  `FuzzyTop(...,0)` is an exact probe, negative values give
  `ErrInvalidMaxWords`.

## Verification (done)

- Unit tests (`pkg/dictionary/fuzzy_test.go`): "кот" k=1 -> {код, крот},
  k=2 -> +{год, дом}; "ёж/ёжик"; threshold k=0 degenerates to an exact
  lookup (every word in the dictionary); `ErrInvalidMaxDist`/`ErrClosed`
  errors; sorting.
- Integration (`fuzzy_integration_test.go`): "слон→клон", "стул→стол",
  k=0 consistent with `Lookup`; grouping by distance on the full dictionary.
- Benchmark: k=2 on the full dictionary — 981 matches in ~0.7ms
  (`BenchmarkFuzzyK2`), well within (by three orders of magnitude) the <1s budget.
- Review and, if needed, finish the tests (`TestFuzzyTopFullDict` and
  `TestFuzzyTopNearestN`, `TestFuzzyTopZeroIsExactProbe`,
  `TestFuzzyMatchesBruteforceOracle`) (check changed files via git
  status), and confirm the functionality works.
