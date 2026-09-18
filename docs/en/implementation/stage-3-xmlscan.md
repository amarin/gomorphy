# Stage 3. dict.xml scanner (internal/xmlscan) — DONE

## Stage contents

Increment: a fixed-schema streaming byte scanner.

- A scanner over `io.Reader`: events `OnGrammeme(parent,name)`, `OnLemma(id,text)`,
  `OnForm(text)`, `OnGrammemeRef(v)`; attributes as slices of the internal buffer.
- Handling buffer boundaries (carrying over incomplete tokens).

## Verification (done)

- Unit tests on mini fixtures (XML strings): event order, attribute values.
- A unit test for buffer truncation (a small 64-byte buffer size).
- Integration smoke test: the first 10k lemmas of the real `dict.xml` parse
  with no errors (`go test -tags integration` or a manual utility run).
- Comparing the event count against the reference: 391,842 lemmas /
  5,533,109 lines on the full file.
