# Stage 7. End-to-end OpenCorpora integration — DONE

## Stage contents

Increment: the full FT7 cycle.

- `dictionary.CompileFromXML(path)`: xmlscan -> Builder -> a snapshot
  (+ `SaveToAtomic`: a temp file + rename).
- `cmd/opencorpora_update`: flags `-l` (download), `-skip-compile`, `-v`;
  compilation runs automatically after unpack, with a time/memory summary.
- `pkg/opencorpora.Update()` triggers compilation again (`loader.Compile()`).

## Verification (done)

- A manual run of `go run ./cmd/opencorpora_update -l` against the real
  `dict.xml`: `opencorpora.dict` (303 MB) built in 13.7s (scan+build+save),
  peak heap ~4 GB.
- Smoke test (`pkg/dictionary/smoke_integration_test.go`): 50
  high-frequency words resolve; control words ("кота" -> кот sing,gent;
  "домами"; homonyms "стекла", "пила", "бежал" with >=2 lemmas each) are
  checked against the dictionary's actual data.

## Note for Stage 8

In `dict.xml`, forms carry only their own `<g>` tags (case/number); POS
and constant tags live on the lemma — when producing an FT2/FT5 result,
the full grammatical characteristic needs to be assembled from the
lemma + the form.
