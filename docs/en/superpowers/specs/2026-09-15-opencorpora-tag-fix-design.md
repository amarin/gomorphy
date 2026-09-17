# OpenCorpora import: fix corrupted word-form tags

## Context

The pre-1.0 code review (`docs/code-review-pre-1.0.md`) found that every
word-form tag produced by the OpenCorpora importer is wrong: instead of its
own grammemes, a form gets an accumulated mixture of grammemes from
preceding forms of the same lemma (and the lemma's own grammemes never
reach any form at all). This is the one remaining critical,
release-blocking finding tracked in `docs/todo.md` under "Критический
баг: искажённые теги OpenCorpora-словоформ". Confirmed twice
independently: a crafted repro against a real-schema fragment (ёж/ежа/ежу)
and a manual lookup against a fully rebuilt `.data/opencorpora/opencorpora.dat`
(`занудами` → an 11-form grammeme concatenation instead of `plur,ablt`).
Scope affects essentially the whole OpenCorpora-imported dictionary (nouns
~12 forms, verbs more).

## Root cause

`internal/xmlscan/dispatch.go`'s `closeTag` fires `Handler.OnLemmaEnd()`
when the `<l>` element (the lemma's headword, inside `<lemma>...</lemma>`)
closes — not when `<lemma>` itself closes (which has no handler callback
at all). `pkg/morphology/importers/opencorpora/import.go`'s `xmlHandler`
treated `OnLemmaEnd` as "clear all accumulated grammemes", which fires
**before any form is parsed**, not after all of them are. Sequence per
lemma, as implemented before this fix:

1. `<l t="…">` → `OnLemma` clears `curGrams`.
2. `<g v="X"/>` inside `<l>` → `OnGrammemeRef` appends to `curGrams` (the
   lemma's own grammemes: part of speech, animacy, gender, …).
3. `</l>` → `OnLemmaEnd` **wipes `curGrams`** — the lemma grammemes just
   collected are discarded before the first form is even seen.
4. First `<f t="…">` → `OnForm` snapshots `curGrams` (now empty) as this
   form's tag. Its own `<g>` children, parsed *after* this snapshot, are
   appended to `curGrams` too late to count for this form.
5. Each subsequent `<f>` snapshots whatever accumulated from *all prior
   forms'* own grammemes (never cleared between forms) — an ever-growing
   mixture, and the lemma's own grammemes are gone for good.

One mechanism explains all three symptoms already recorded in
`docs/todo.md`: the first form's empty tag, the lemma-level grammemes
never appearing anywhere, and the accumulating mixture in later forms.

## Decision: fix inside `xmlHandler` only, rename the misleading event

Two approaches were compared:

- **A — fix `xmlHandler` only.** No change to `xmlscan.Handler`'s
  contract or `dispatch.go`'s wiring; `xmlHandler` stops misinterpreting
  what already fires on `</l>` close. Single-file diff, zero risk to the
  rest of the streaming XML scanner.
- **B — A, plus split `OnLemmaEnd` into an honestly-named `</l>`-close
  event and a newly-wired true `</lemma>`-close event.** Same runtime
  behavior as A; additionally fixes the misleading name that contributed
  to this bug in the first place, at the cost of touching the `Handler`
  interface and its two test doubles (`internal/xmlscan/scanner_test.go`,
  `internal/xmlscan/integration_test.go`).

**Decision: A, with one piece of B folded in** — rename the interface
method from `OnLemmaEnd` to `OnLemmaHeadEnd` (it fires on `</l>` close;
`</lemma>` still has no dedicated event and doesn't need one — the next
lemma's `OnLemma` already resets all per-lemma state). This is a
mechanical rename, not a new event, so it stays small: interface + 2 call
sites in `dispatch.go` + both test doubles + `xmlHandler`. Every touched
method gets a doc comment stating exactly when it fires and what it does,
per the user's explicit request — the absence of that is part of what let
the original miswiring go unnoticed.

## Design

### `xmlHandler` state (`pkg/morphology/importers/opencorpora/import.go`)

`curForm *formGrams` and `lGrams []string` are removed — both write-only,
never read anywhere (confirmed by grep before writing this spec; `lGrams`
was already flagged as dead code from a prior attempt at this fix).

```go
type xmlHandler struct {
    tagSet *internal.TagSet
    lemmas *[]lemmaEntry
    err    error

    lemGrams    []string // current lemma's own grammemes (from <l>); persist across all its forms
    formOwnGrams []string // current form's own grammemes (from <f>); reset on every OnForm
    curFormText string   // current form's text, held between OnForm and OnFormEnd
    inLemmaHead bool     // true between OnLemma and OnLemmaHeadEnd (i.e. while inside <l>)
}
```

(`formOwnGrams`, not `formGrams` — the package already has a type named
`formGrams` (`type formGrams struct{ text, gramm string }`, used as
`lem.forms []formGrams`); a same-named field would be legal Go but
needlessly confusing next to it.)

### Event flow

| XML event | Handler method | Effect |
|---|---|---|
| `<l t="…">` opens | `OnLemma(id, text)` | Append new `lemmaEntry`. Reset `lemGrams=nil`, `formOwnGrams=nil`. Set `inLemmaHead=true`. |
| `<g v="X"/>` inside `<l>` | `OnGrammemeRef(v)` | `inLemmaHead==true` ⇒ append to `lemGrams`. |
| `</l>` closes | `OnLemmaHeadEnd()` | Set `inLemmaHead=false`. **Does not clear `lemGrams`** — this is the actual fix. |
| `<f t="…">` opens | `OnForm(text)` | Reset `formOwnGrams=nil`, set `curFormText=text`. Tag is **not** built here (own `<g>` children haven't been parsed yet). |
| `<g v="Y"/>` inside `<f>` | `OnGrammemeRef(v)` | `inLemmaHead==false` ⇒ append to `formOwnGrams`. |
| `</f>` closes | `OnFormEnd()` | Build `gramm = strings.Join(append(append([]string{}, lemGrams...), formOwnGrams...), ",")` (lemma grammemes first, then the form's own, in XML declaration order — no sorting, no dedup, per the user's explicit answer). Append `formGrams{text: curFormText, gramm}` (the existing `formGrams` *type*, unrelated to the removed field name above) to the current lemma's `forms`. |

The key structural change from the current code: **the tag is assembled
on `OnFormEnd`, not `OnForm`** — it must wait until the form's own `<g>`
children have actually been parsed. `lemGrams` is deliberately never
cleared between forms of the same lemma (only on the next `OnLemma`),
since it represents grammemes shared by every form of that lemma.

### `xmlscan` changes

- `scanner.go`: rename `Handler.OnLemmaEnd() error` to
  `OnLemmaHeadEnd() error`; doc comment: "fires when `<l>` (the lemma's
  headword element) closes — not when the enclosing `<lemma>` closes,
  which has no dedicated event; call sites needing true end-of-lemma
  behavior should reset on the next `OnLemma` instead."
- `dispatch.go`: rename the 2 call sites (`dispatch()`'s self-closing
  `<l>` branch, `closeTag()`'s `case t.is("l")` branch) with a one-line
  comment reiterating the same "fires on `</l>`, not `</lemma>`" point at
  the call site.
- `scanner_test.go`, `integration_test.go`: rename the method on whatever
  test double(s) implement `Handler` to keep them satisfying the
  interface.

### Test fixture

`import_test.go`'s `testDictXML` currently writes lemma grammemes as an
attribute — `<l g="NOUN,anim,masc,sing">` — which does not match the real
schema (nested elements: `<l t="ёж"><g v="NOUN"/><g v="anim"/>…</l>`,
confirmed against `.data/opencorpora/dict.xml`). Since
`internal/xmlscan/dispatch.go` only ever reads `<l>`'s `t` attribute and
its nested `<g>` children, the attribute-based fixture was never
exercising the real code path for lemma grammemes at all. Rewrite it to
the real nested-element form as part of this fix.

## Non-goals

- The separately-mentioned "possible loss of lemmas from OpenCorpora" —
  raised again by the user in this session and explicitly confirmed as
  out of scope: "это отдельная история, здесь мы разбираемся только с
  граммемами". Not investigated here; stays a pointer in `docs/todo.md`
  for its own future investigation, same status as before this spec.
- Splitting a real `</lemma>`-close event out of `OnLemma`/next-lemma
  reset (the extra part of Approach B beyond the rename) — no current
  need for it; `OnLemma` already resets all per-lemma state correctly for
  the next lemma.
- Any change to `TagSet.Add`'s dedup-by-exact-string-equality behavior
  (`pkg/morphology/internal/tagset.go`) — declaration-order concatenation
  is sufficient per the user's answer; no canonicalization/sorting is
  being introduced.
- Grammeme merge order alternatives (sorted, deduped, tagged by source) —
  explicitly rejected by the user in favor of plain declaration-order
  concatenation with no source tracking.

## Testing

(Exact test list belongs in the implementation plan; noting shape here.)

- Regression: a lemma with 2+ forms — each form gets its *own* tag
  (lemma grammemes + that form's own grammemes), not the previous form's;
  the first form is not empty; lemma-level grammemes (POS/animacy/gender)
  appear in every form's tag.
- Self-closing `<l .../>` with no `<g>` children — degenerate case,
  `lemGrams` stays empty, forms get only their own grammemes.
- Optional: replay the ёж/ежа/ежу and занудами cases from
  `docs/code-review-pre-1.0.md` directly, since both already have a
  documented "expected" tag to assert against.
- `internal/xmlscan` tests green after the `OnLemmaHeadEnd` rename.
- `go test ./... -race` green.

## Risks

- The fixture rewrite (attribute → nested `<g>` elements) changes what
  the existing importer unit tests actually exercise; any test currently
  passing *because* the fixture never hit the lemma-grammeme code path at
  all needs re-checking, not just the tag-content assertions.
- `TagSet` is dedup'd by exact string equality (`tagset.go`); this fix
  changes what strings get produced (previously-wrong tags won't match
  whatever a hypothetical downstream consumer may have already keyed off
  of) — acceptable pre-1.0 (no compatibility to preserve), but worth
  flagging since it's the same file space this session's DAWG-density
  research (`docs/research/0001-dawg-alphabet-density.md`) also touches
  conceptually (tag strings feed `BuildDAWG` keys downstream).
