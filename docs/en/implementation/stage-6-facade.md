# Stage 6. Public facade pkg/dictionary (FT7-FT9) — DONE

## Stage contents

Increment: the library API.

- `Open`/`NewEmpty`/`Lookup`/`Lemmas`/`Fuzzy`(a stub until Stage 9)/`SaveTo`/`Builder`
  (Builder is a facade over `build.Builder`: `NewBuilder` + `AddGrammeme`/`AddLemma`/
  `AddForm`/`Compile`; `Dictionary.Builder()` reconstructs the population
  from a snapshot).
- Concurrency: reading from N goroutines; two independent instances at once.

## Verification (done)

- `go test ./pkg/dictionary -race`: concurrent reads, parallel instances.
- A usage example in `example_test.go` (a godoc example with Output).
- No global variables: only immutable-snapshot fields, closing via
  `atomic.Bool` (`ErrClosed`), no package-level state.
