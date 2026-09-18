# Stage 5. Compiler and file loader — DONE

## Stage contents

Increment: `SaveTo`/`Open` using Stage 1's format (FT3, FT4).

- Serializing a snapshot into sections: delta/varint for offsets and postings.
- An mmap reader (`internal/mmapx`): `Open(path)` -> a Dictionary snapshot.

## Verification (done)

- Roundtrip test: `Build` -> `SaveTo` -> `Open` -> `DeepEqual` of the
  snapshots + equal lookups over a control word set (a golden set with
  expected grammars).
- A corrupted-file test: truncation, checksum corruption -> meaningful errors.
- A size measurement on the full `dict.xml`: 303 MB (logged in the test;
  no hard assertions until it stabilizes). Dominant sections:
  `texts.data` ~67 MB, `exact` ~47 MB, `pairs raw` ~62 MB — candidates
  for compression in an FT4 optimization (Stage 10+). Save ~1.0s, Open ~0.3s.
