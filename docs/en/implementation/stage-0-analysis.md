# Stage 0. Repository analysis and preparation — DONE

## Stage contents

- Analysis of the `dict.xml` schema, gathering metrics (see [requirements.md](../requirements.md)).
- Storage structure design ([implementation.md](../implementation.md)).
- Fixing requirements FT1-FT9 ([requirements.md](../requirements.md)).
- Removing stale code from `internal/` and `pkg/` (the index object model,
  binutils serialization, the generic XML parser). Kept: `pkg/common`,
  `pkg/opencorpora` (download/unpack only), `cmd/opencorpora_update`
  (download+unpack).

## Verification

- `go build ./...`, `go vet ./...`, `go test ./...` — green.
- `go mod tidy && go mod vendor` with no extra dependencies.
