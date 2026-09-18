# Stage 1. Format primitives (internal/format) — DONE

## Stage contents

Increment: the encoding package and sectioned container.

- varint/delta codecs (`[]uint32`, `[]uint64`) — encode/decode roundtrip.
- Header: magic `"GMRF"`, version, xxh3 checksum; a section catalog
  (name, offset, size); a writer (sections are appended, catalog at the
  end) and a reader (magic/version/checksum checks, access to a section
  by name).

## Verification (done)

- Roundtrip unit tests for varint/delta on edge cases (0, 1, max
  u32/u64, monotonic/non-monotonic sequences) — `TestAppendDelta*Roundtrip`.
- Writer->reader unit tests: several sections (including empty ones,
  streaming), reading by name, errors on a corrupted checksum/magic/
  version/truncation.
- `go build ./...`, `go vet ./...`, `go test ./internal/format -race` — green.

## Manual verification

A hexdump of a test file — magic `"GMRF"`, version=1, indexOffset=0x1C,
a catalog of 2 sections, an xxh3 trailer — all readable and correct.
