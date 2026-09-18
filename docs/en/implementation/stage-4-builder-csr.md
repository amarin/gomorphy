# Stage 4. Builder and CSR structures (internal/build) — DONE

## Stage contents

Increment: programmatic dictionary population (FT8) + in-memory structures.

- Builder: `AddGrammeme`/`AddLemma`/`AddForm`, ancode tables, CSR arrays
  of lemmas/strings, a trie (temporary, struct nodes) -> compiled into CSR
  (`stateOff`/`transitions`/`finals`), posting lists with deduplication, exact-hash.
- The `Build()` method -> an immutable snapshot (values, not pointers).

## Verification (done)

- Builder unit tests: a small dictionary (5-10 lemmas) built by hand;
  checking lookup for every word, duplicate forms, identical texts across
  different lemmas.
- A property test: N random words -> all found; missing ones -> not found.
- `go test ./internal/build -race`.
