# Stage 15. OpenCorpora import — DONE

## Stage contents

Importing the OpenCorpora dictionary (`dict.xml`) by extracting
paradigms from lemmas. Reuses `internal/xmlscan` to read the XML. The
result is the same internal format (paradigm + a DAWG with a payload).

### The importer (`pkg/morphology/importers/opencorpora/`)

Function `ImportFromXML(r io.Reader, tagSet *TagSet) (*Dictionary, error)`:

#### Pipeline

1. **Scanning the XML** via `xmlscan`:
   - Event `Grammeme`: collecting grammeme names (optional, used for
     `<grammeme>` with `<name>`).
   - Event `Lemma`: the start of a lemma (id, text).
   - Event `Form`: a wordform (text from the `t` attribute, grammemes
     from `<g>` tags).

2. **Extracting paradigms** (the core algorithm):
   ```
   For each lemma:
     Collect all forms: forms = [{text, gramm}, ...]
     Compute the stem: stem = LCP(forms.map(_.text))
     For each form:
       suffix = form.text[len(stem):]
       tag_id = tagSet.ID(form.gramm)  // the combined tag string
       paradigm.add(suffix_id, tag_id, 0)  // prefix = 0 (empty)
     Deduplicate the paradigm (hash -> paradigmID)
     Store the pair (stem, paradigmID)
   ```

3. **Building the DAWG**:
   ```
   For each (stem, paradigmID):
     For each form of the paradigm (form_idx):
       key = stem + suffix[form_idx] + "\x01" + base64(para_id << 16 | form_idx)
   BuildDAWGWithValues(keys, values)
   ```

4. **Returning a Dictionary** with Suffixes, Prefixes (nil), Paradigms,
   and Words populated.

#### Paradigm deduplication

A paradigm is an ordered list of `(suffix_id, tag_id)` (prefix = 0 for
OpenCorpora). The hash is computed by serializing the suffix and tag
bytes. Result: ~391K lemmas -> ~3K unique paradigms (for the full dict.xml).

#### TagSet

Tags are collected from the `<g v="...">` tags on each form. Every
unique `gramm` (a combined string like `"NOUN,anim,masc,sing,nomn"`) is
added to the TagSet via `Add()`.

### Public API

```go
// In pkg/morphology/
func CompileFromXML(r io.Reader) (*Dictionary, error)
func CompileFromXMLFile(path string) (*Dictionary, error)
```

`CompileFromXMLFile` opens `dict.xml`, parses the grammemes, and builds
the Dictionary. `CompileFromXML` is a wrapper over
`opencorpora.ImportFromXML`.

### CLI

```bash
gomorphy import opencorpora dict.xml -o oc.dat
gomorphy -dict oc.dat lookup кота
```

## Verification (done)

- Unit test: a small XML (3 lemmas) -> paradigms extracted correctly.
- Unit test: the paradigm count <= the lemma count (dedup works).
- Unit test: stem = LCP of all forms (checking the "" suffix = index 0).
- Unit test: the DAWG via SimilarItems finds every wordform.
- Unit test: roundtrip — DAWG values have the correct structure (4 bytes).
- Unit test: lemmas with no wordforms are ignored.
- `go test ./pkg/morphology/importers/opencorpora/... -count=1` — green.

## Manual verification

- `gomorphy import opencorpora dict.xml -o oc.dat` -> the file is created.
- `gomorphy -dict oc.dat lookup кота` -> a correct reading.

## Implementation (result)

- `pkg/morphology/importers/opencorpora/import.go` — the full importer:
  an xmlscan Handler -> LCP stem -> a suffix ID map -> paradigm dedup -> DAWG.
- `internal.BuildDAWGWithValues` — an extension of the dawgdic builder
  to store values as a payload (key = `word\x01<base64_value>`).
- `pkg/morphology/open.go` — `CompileFromXML` and `CompileFromXMLFile` added.
- `cmd/gomorphy/main.go` — the `import opencorpora <xml> -o <out.dat>` subcommand.

### Deviations from the plan

- **Suffixes vs. paradigms**: in the original plan, suffixes were a
  separate pool, but the actual implementation stores suffix texts in
  `Dictionary.Suffixes` as an ordered list built from a `suffixTexts`
  map. This is compatible with the pymorphy2 importer.
- **Prefixes**: OpenCorpora doesn't store prefixes in paradigms (lemmas
  already contain the full text), so `Prefixes = nil`.
