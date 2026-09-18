# Stage 2. String interning (internal/intern, internal/stringsx) — DONE

## Stage contents

Increment: a string arena + an interning table with no per-token allocations.

- `stringsx.Arena`: appending bytes, boundaries via uint32 offsets, `Get(i) []byte`.
- `intern.Table`: a `[]byte` -> id hash table, open addressing, pre-sized, rehashing;
  method `Intern(b []byte) (id uint32, existed bool)`, `Get(id) []byte`.

## Verification (done)

- Arena/Intern unit tests: duplicates get one id; unique inputs get new
  ids; Get correctness after the table grows (a rehash at 50k inserts
  against an expected 4); similar words get different ids.
- Intern benchmark on 1M strings: repeated inserts — 63 ns/op,
  **0 allocs/op** (requirement FT1 met);
  building the full unique dictionary of 1M strings — 71 ms, 23
  allocations (amortized arena growth and rehashing).
- `go test ./internal/intern ./internal/stringsx -bench .` — green,
  the full `-race` run across all packages green.
