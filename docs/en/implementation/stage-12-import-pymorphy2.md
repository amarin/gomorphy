# Stage 12. PyMorphy2 import

## Stage contents

Loading a PyMorphy2 dictionary from a directory (words.dawg +
paradigms.array + suffixes.json + gramtab-*.json) -> an immutable
`Dictionary`.

### The importer (`pkg/morphology/importers/pymorphy2/`)

Function `ImportFromDir(dir string) (*Dictionary, error)`:

1. **Reading `gramtab-opencorpora-int.json`**: a JSON string array ->
   TagSet. Each string is a tag like `NOUN,anim,masc,sing,nomn`.

2. **Reading `suffixes.json`**: a JSON string array -> `[]string` (the
   suffix pool). The array index = suffix_id in paradigms.

3. **Reading `paradigm-prefixes.json`** (optional): a JSON string array
   -> `[]string` (the prefix pool). If the file is missing —
   `["", "по", "наи"]`.

4. **Reading `paradigms.array`**:
   - uint16 count — the number of paradigms
   - For each paradigm: uint16 len + []uint16 data (len x 3 values:
     suffixes + tags + prefixes)

5. **Reading `words.dawg`**: dictionary uint32[] + guide byte[] -> a
   DAWG. Format: uint32 size -> size x uint32 dictionary -> uint32
   guide_size -> guide_size x 2 bytes of guide.

6. **Optional: `prediction-suffixes-N.dawg`**: prediction DAWGs. Count =
   the number of prefixes (len(prefixes)).

7. **Optional: `p_t_given_w.intdawg`**: a probability DAWG.

### Public API

```go
// In pkg/morphology/
func OpenPyMorphy(dir string) (*Dictionary, error)
```

A wrapper over `pymorphy2.ImportFromDir`. Sets `CharPolicy` to the
Russian one (е→ё) by default.

## Verification (tests)

- Unit test: ImportFromDir with a test directory (a small DAWG + paradigms).
- Unit test: roundtrip — ImportFromDir -> SaveTo -> Open -> identical data.
- Integration test: the full pymorphy2-dicts-ru dictionary:
  - Parse("все") -> >=4 readings
  - Parse("кота") -> a reading with NOUN,anim,masc,sing,gent
  - Parse("кот") -> a reading with NOUN,anim,masc,sing,nomn
- `go test ./pkg/morphology/... -race` — green.

## Manual verification

- `pip install pymorphy2-dicts-ru` -> `gomorphy import pymorphy2 $(python -c "...") -o pymorphy2.dat`
- `gomorphy -dict pymorphy2.dat lookup кота` -> a correct reading
- Compare against the output of `python -c "import pymorphy2; print(pymorphy2.MorphAnalyzer().parse('кота'))"`

## Implementation (the actual API)

Package `pkg/morphology/importers/pymorphy2` (package `pymorphy2`) + the
`pkg/morphology.OpenPyMorphy` wrapper. Format reference — `opennota/morph`'s
morph.go/dict.go/guide.go.

- `pymorphy2.ImportFromDir(dir) (*internal.Dictionary, error)` — reads directly:
  - `gramtab-opencorpora-int.json` -> a JSON string array -> `TagSet`
    (named `opencorpora-int`);
  - `paradigm-prefixes.json` -> `Prefixes`; if missing — `["", "по", "наи"]`
    (as in opennota);
  - `suffixes.json` -> `Suffixes`;
  - `paradigms.array` — `uint16 count` + for each, `uint16 len` +
    `len×uint16` LE -> `NewParadigmFromData` (length must be divisible by 3);
  - `words.dawg` -> `ReadDAWG` (dictionary+guide);
  - optionally `p_t_given_w.intdawg` -> `Probability`; optionally
    `prediction-suffixes-{i}.dawg` for `i < len(prefixes)` -> `Prediction`
    (missing files are skipped);
  - `Language="ru"`, `CharPolicy=RussianCharPolicy()`.
- Internal primitives: `internal.NewParadigmFromData(data []uint16)` —
  accepts pymorphy2's raw data with no copying.
- Test infrastructure: the DAWG fixture builder was factored out of the
  internal package's tests into `pkg/morphology/internal/testdawg`
  (`Build`, `Marshal`) — reused by the importer's unit tests and the
  public wrapper. The fixtures write real pymorphy2 directory files
  (including words with a `0x01` payload, base64-encoded values).
- Notes:
  - `SaveTo`/`Open` roundtrip is deferred to Stage 14 (serialization) —
    see todo.md.
  - The full-dictionary integration test (`//go:build integration`)
    needs `GOMORPHY_PYMORPHY2_DIR`; without it, `t.Skip`. It checks the
    structure (~3K paradigms, ~5K suffixes, ~1K tags, 3 prediction
    DAWGs, probability) and >=4 readings for "все" via `Words.SimilarItems`.
