# Stage 14. Serialization: a unified on-disk format

## Stage contents

Defining and implementing a unified file format for storing a
dictionary. The format is compatible with pymorphy2 (words.dawg +
paradigms.array + strings) or is an extension of it with additional
sections.

### File format (`pkg/morphology/internal/format.go`)

```
┌──────────────────────────────────┐
│  Header                          │
│  magic  "GMOR"  4 bytes          │
│  version    u32                  │
│  checksum   xxh3-64  8 bytes     │
├──────────────────────────────────┤
│  Section catalog                 │
│  count      u16                  │
│  entries:                          │
│    name     [16]byte             │
│    offset   u64                  │
│    size     u64                  │
│    flags    u8  (compression, etc.) │
├──────────────────────────────────┤
│  Data sections                   │
│  meta, tagset, suffixes,         │
│  prefixes, paradigms,            │
│  words.dawg, prediction-N,       │
│  probability                      │
└──────────────────────────────────┘
```

### Sections

- **meta**: language, lemma/form/paradigm counts, format version.
- **tagset**: a JSON array of grammeme name strings.
- **suffixes**: varint-length-prefixed strings (the suffix pool).
- **prefixes**: varint-length-prefixed strings (the prefix pool).
- **paradigms**: a uint16 array (N suffixes + N tags + N prefixes per paradigm).
- **words.dawg**: dictionary uint32[] + guide byte[] (as in pymorphy2).
- **prediction-N**: prediction DAWGs (optional).
- **probability**: a probability DAWG (optional).

### Writing (`pkg/morphology/save.go`)

```go
func (d *Dictionary) SaveTo(path string) error
```

1. Serializing the components:
   - tagset -> JSON
   - suffixes -> varint-length-prefixed strings
   - prefixes -> varint-length-prefixed strings
   - paradigms -> a uint16 array
   - words.dawg -> raw bytes
2. Writing the header + catalog + sections.
3. Optionally: zstd-compressing the cold sections (suffixes, prefixes,
   tagset, paradigms). A `flag_compressed` flag in the catalog.

### Reading (`pkg/morphology/open.go`)

```go
func Open(path string) (*Dictionary, error)
```

1. mmap the file, read the header and catalog.
2. For each section:
   - If the compressed flag is set — decompress (zstd) into memory.
   - If not compressed — alias the mmap region (zero-copy).
3. Parse the components: tagset from JSON, suffixes/prefixes from
   varint-encoded data, paradigms from a uint16 array, the DAWG from
   dictionary+guide.
4. Return an immutable `Dictionary`.

### Compatibility with pymorphy2's format

To read directly from a pymorphy2 directory (with no conversion) —
`OpenPyMorphy(dir)` reads pymorphy2's format files directly (Stage
12). The `GMOR` format is an extension with additional sections
(meta, prediction, probability).

## Verification (tests)

- Unit test: roundtrip ImportFromDir -> SaveTo -> Open -> identical Parse results.
- Unit test: the format version is written and read correctly.
- Unit test: a corrupted file -> a meaningful error (magic mismatch, checksum).
- Unit test: a compressed section decompresses correctly.
- Unit test: an mmap alias for words.dawg (zero-copy).
- `go test ./pkg/morphology/... -race` — green.

## Manual verification

- Compare the `.dat` size against the pymorphy2 directory's size.
- `gomorphy -dict pymorphy2.dat lookup кота` -> identical to a direct load.
- Loading via mmap: `time gomorphy -dict pymorphy2.dat lookup кота`.

## Implementation (result)

- `pkg/morphology/internal/format.go` — the GMOR container: a header
  (magic `GMOR` | version u32 | checksum xxh3-64), a catalog
  (count u16; entry: name[16] | offset u64 | size u64 | flags u8),
  sections. Section offsets are aligned to 8 bytes (0 mod 8), so
  words.dawg's unit array (offset+4) is 4-byte aligned and can be
  aliased from mmap with no copying. Checks: magic, version, checksum,
  the catalog (bounds, duplicate names, truncation).
- Section codecs: `EncodeMeta`/`DecodeMeta` (language + CharPolicy),
  `EncodeTagSet`/`DecodeTagSet` (JSON {name,tags}),
  `EncodeStrings`/`DecodeStrings` (uvarint-length-prefixed),
  `EncodeParadigms`/`DecodeParadigms` (u32 count; len + u16[] per paradigm).
- `internal.DAWG.Bytes()` / `internal.ParseDAWG` — the words.dawg
  stream format; `ParseDAWG` aliases dictionary/guide when aligned
  (zero-copy mmap), otherwise copies. `Paradigm.Data()` for serialization.
- `pkg/morphology.SaveTo` and `Open` (+ `Dictionary.Close`, the mmap
  region's lifecycle). `Open` holds the mmap until `Close`.
- The CLI (`cmd/gomorphy`) switched over to `pkg/morphology`:
  `import pymorphy2 <dir> -o <file.dat>`, `-dict <file.dat>` +
  lookup/lemmas/fuzzy/top; the `-o` flag after the subcommand is parsed
  by hand (the flag package stops at the first non-flag). An end-to-end
  CLI test builds the binary and runs import->lookup->lemmas->fuzzy.

### Deviations from the plan

- **zstd isn't wired up** in this stage: the `FlagCompressed` flag in
  the catalog is provisioned for, and reading a compressed section
  returns a meaningful error. Real compression comes in Stage 17
  (narrowing types + zstd), which adds the codec
  (klauspost/compress) for this.
