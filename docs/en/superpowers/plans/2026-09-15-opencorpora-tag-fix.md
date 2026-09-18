# OpenCorpora Tag-Corruption Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the critical, release-blocking bug where every OpenCorpora word-form gets a corrupted grammatical tag (empty first form, lemma-level grammemes never applied, later forms accumulating a mixture of previous forms' grammemes).

**Architecture:** Root cause is in `internal/xmlscan`: the `</l>` (lemma headword) close event is misleadingly named `OnLemmaEnd` and is misinterpreted by the OpenCorpora importer's `xmlHandler` as "clear everything," which fires before any of the lemma's forms are parsed. The fix has two parts: (1) a pure rename of the misleading event to `OnLemmaHeadEnd` across `internal/xmlscan` and its callers, with no behavior change; (2) a redesign of `xmlHandler`'s grammeme bookkeeping so a form's own `<g>` children (only known once `</f>` closes) are combined with the lemma's own grammemes (from `<l>`) in XML declaration order, assembled on `OnFormEnd` instead of prematurely on `OnForm`.

**Tech Stack:** Go 1.27, `github.com/stretchr/testify` (assert/require, already a dependency).

**Spec:** [docs/en/superpowers/specs/2026-09-15-opencorpora-tag-fix-design.md](../specs/2026-09-15-opencorpora-tag-fix-design.md)

## Global Constraints

- `go build ./...` must succeed after every task (no task may leave the repo non-compiling), matching this project's existing convention (`docs/en/todo.md`, "Formatting requirements").
- `go test ./... -race` must be green after every task.
- No change to `xmlscan.Handler`'s method set beyond the `OnLemmaEnd`→`OnLemmaHeadEnd` rename — no new event, no signature change (per spec's "Non-goals").
- Grammeme merge order is lemma grammemes first, then the form's own, in plain XML declaration order — no sorting, no dedup, no source tracking (per spec's "Decision").
- Out of scope: the separately-tracked "possible loss of lemmas from OpenCorpora" investigation (`docs/en/todo.md`) — do not touch it in this work.

---

### Task 1: Rename `OnLemmaEnd` to `OnLemmaHeadEnd` (pure rename, no behavior change)

**Files:**
- Modify: `internal/xmlscan/scanner.go` (interface method + doc comment)
- Modify: `internal/xmlscan/dispatch.go:59`, `internal/xmlscan/dispatch.go:109` (2 call sites)
- Modify: `internal/xmlscan/scanner_test.go:33` (`recorder` test double)
- Modify: `internal/xmlscan/integration_test.go:85` (`counter` test double)
- Modify: `pkg/morphology/importers/opencorpora/import.go:296` (`xmlHandler`'s method — name only, body unchanged in this task)

**Interfaces:**
- Produces: `xmlscan.Handler.OnLemmaHeadEnd() error` — replaces `OnLemmaEnd() error` in the interface and every implementer. Task 2 relies on this name existing.

This task is a mechanical, repo-wide rename with **zero behavior change** — every implementer's method body stays exactly as it is today, only the name changes. There is no new test to write; the existing test suite is the safety net (a missed rename anywhere fails to compile, since `Handler` is an interface).

- [ ] **Step 1: Rename the interface method in `scanner.go`**

In `internal/xmlscan/scanner.go`, replace:

```go
type Handler interface {
	OnGrammeme(parent []byte, name []byte) error
	OnGrammemeRef(value []byte) error
	OnLemma(id uint32, text []byte) error
	OnLemmaEnd() error
	OnForm(text []byte) error
	OnFormEnd() error
}
```

with:

```go
type Handler interface {
	OnGrammeme(parent []byte, name []byte) error
	OnGrammemeRef(value []byte) error
	OnLemma(id uint32, text []byte) error
	// OnLemmaHeadEnd fires when <l> (the lemma's headword element) closes
	// — not when the enclosing <lemma> closes, which has no dedicated
	// event of its own. Implementations needing true end-of-lemma
	// behavior should reset their state on the next OnLemma call instead.
	OnLemmaHeadEnd() error
	OnForm(text []byte) error
	OnFormEnd() error
}
```

- [ ] **Step 2: Rename the two call sites in `dispatch.go`**

In `internal/xmlscan/dispatch.go`, inside `dispatch()`'s `case t.is("l")` branch, replace:

```go
		if t.selfClosing {
			s.inWord = false

			return s.h.OnLemmaEnd()
		}
```

with:

```go
		if t.selfClosing {
			s.inWord = false

			// </l> (self-closing <l/> here), not </lemma> — see Handler.OnLemmaHeadEnd's doc comment.
			return s.h.OnLemmaHeadEnd()
		}
```

In `closeTag()`'s `case t.is("l")` branch, replace:

```go
	case t.is("l"):
		if s.section == sectLemmata && s.inWord {
			s.inWord = false

			return s.h.OnLemmaEnd()
		}
```

with:

```go
	case t.is("l"):
		if s.section == sectLemmata && s.inWord {
			s.inWord = false

			// </l> closes here, not </lemma> — see Handler.OnLemmaHeadEnd's doc comment.
			return s.h.OnLemmaHeadEnd()
		}
```

- [ ] **Step 3: Rename the test double in `scanner_test.go`**

Replace:

```go
func (r *recorder) OnLemmaEnd() error { r.events = append(r.events, "lemma-end"); return nil }
```

with:

```go
func (r *recorder) OnLemmaHeadEnd() error { r.events = append(r.events, "lemma-end"); return nil }
```

(The recorded event string stays `"lemma-end"` — it's a test-only label, not part of the fix; renaming it too would just churn every `wantEvents()`/`want` literal in this file for no behavioral reason.)

- [ ] **Step 4: Rename the test double in `integration_test.go`**

Replace:

```go
func (c *counter) OnLemmaEnd() error { return nil }
```

with:

```go
func (c *counter) OnLemmaHeadEnd() error { return nil }
```

- [ ] **Step 5: Rename the method in `xmlHandler` (name only — body unchanged in this task)**

In `pkg/morphology/importers/opencorpora/import.go`, replace:

```go
func (h *xmlHandler) OnLemmaEnd() error {
	h.curForm = nil
	h.curGrams = nil
	return nil
}
```

with:

```go
func (h *xmlHandler) OnLemmaHeadEnd() error {
	h.curForm = nil
	h.curGrams = nil
	return nil
}
```

(This keeps the *old, still-buggy* behavior for now — Task 2 redesigns this method's body. The point of this task is solely that the whole repo still compiles and every existing test still passes after the rename.)

- [ ] **Step 6: Verify the whole repo still builds**

Run: `go build ./...`
Expected: no output, exit 0. If `pkg/morphology/importers/opencorpora` fails to compile with "missing method OnLemmaHeadEnd" (or similar), Step 5 was missed or misspelled.

- [ ] **Step 7: Verify existing tests are unaffected**

Run: `go test ./internal/xmlscan/... ./pkg/morphology/importers/opencorpora/... -race -v`
Expected: all PASS, identical to pre-rename output (this is a pure rename — no test's expected values change).

- [ ] **Step 8: Commit**

```bash
git add internal/xmlscan/scanner.go internal/xmlscan/dispatch.go internal/xmlscan/scanner_test.go internal/xmlscan/integration_test.go pkg/morphology/importers/opencorpora/import.go
git commit -m "$(cat <<'EOF'
xmlscan: rename OnLemmaEnd to OnLemmaHeadEnd (no behavior change)

The event actually fires on </l> (the lemma's headword element)
closing, not on </lemma> (which has no event at all) — the old name
misled the OpenCorpora importer into treating it as "end of lemma,
clear everything," which is the root cause of the tag-corruption bug
fixed in the next commit. Pure rename here: every implementer's
method body is unchanged, existing tests are unaffected.

See docs/en/superpowers/specs/2026-09-15-opencorpora-tag-fix-design.md.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Redesign `xmlHandler` grammeme bookkeeping (the actual fix)

**Files:**
- Modify: `pkg/morphology/importers/opencorpora/import.go:105-317` (`ImportFromXML`'s handler construction + the whole `xmlHandler` type and its methods)
- Modify: `pkg/morphology/importers/opencorpora/import_test.go` (fixture rewrite + new regression test)

**Interfaces:**
- Consumes: `xmlscan.Handler.OnLemmaHeadEnd` (from Task 1).
- Produces: no change to any exported symbol — `xmlHandler` is unexported, `ImportFromXML`'s signature is unchanged. Downstream (`internal.TagSet.Tags`, already exported) now contains correct tag strings instead of corrupted ones.

- [ ] **Step 1: Rewrite `testDictXML`'s lemma-grammeme encoding to match the real schema**

In `pkg/morphology/importers/opencorpora/import_test.go`, replace the `testDictXML` constant's `<lemmata>` section:

```go
 <lemmata>
  <lemma id="1" text="кот">
   <l g="NOUN,anim,masc,sing">
    <f t="кот">
     <g v="nomn"/>
    </f>
    <f t="кота">
     <g v="gent"/>
    </f>
   </l>
  </lemma>
  <lemma id="2" text="кот">
   <l g="VERB,impf,trans">
    <f t="кот"/>
   </l>
  </lemma>
  <lemma id="3" text="мышь">
   <l g="NOUN,fem,sing,anim">
    <f t="мышь">
     <g v="nomn"/>
    </f>
    <f t="мыши">
     <g v="gent"/>
    </f>
   </l>
  </lemma>
 </lemmata>
```

with the real schema's structure — `<l>` and `<f>` as siblings under `<lemma>`, lemma grammemes as nested `<g>` elements (not an attribute), matching `internal/xmlscan/scanner_test.go`'s `sampleDict` and the real `.data/opencorpora/dict.xml`:

```go
 <lemmata>
  <lemma id="1" text="кот">
   <l t="кот"><g v="NOUN"/><g v="anim"/><g v="masc"/><g v="sing"/></l>
   <f t="кот">
    <g v="nomn"/>
   </f>
   <f t="кота">
    <g v="gent"/>
   </f>
  </lemma>
  <lemma id="2" text="кот">
   <l t="кот"><g v="VERB"/><g v="impf"/><g v="trans"/></l>
   <f t="кот"/>
  </lemma>
  <lemma id="3" text="мышь">
   <l t="мышь"><g v="NOUN"/><g v="fem"/><g v="sing"/><g v="anim"/></l>
   <f t="мышь">
    <g v="nomn"/>
   </f>
   <f t="мыши">
    <g v="gent"/>
   </f>
  </lemma>
 </lemmata>
```

- [ ] **Step 2: Rewrite the inline fixture in `TestImportFromXMLNoForms`**

In the same file, inside `TestImportFromXMLNoForms`, replace:

```go
 <lemmata>
  <lemma id="1" text="пустая">
   <l g="NOUN">
   </l>
  </lemma>
  <lemma id="2" text="есть">
   <l g="NOUN">
    <f t="есть">
     <g v="nomn"/>
    </f>
   </l>
  </lemma>
 </lemmata>
```

with:

```go
 <lemmata>
  <lemma id="1" text="пустая">
   <l t="пустая"><g v="NOUN"/></l>
  </lemma>
  <lemma id="2" text="есть">
   <l t="есть"><g v="NOUN"/></l>
   <f t="есть">
    <g v="nomn"/>
   </f>
  </lemma>
 </lemmata>
```

- [ ] **Step 3: Write the new failing regression test**

Add to `pkg/morphology/importers/opencorpora/import_test.go` (e.g. after `TestImportFromXMLStemLCP`):

```go
// TestImportFromXMLFormTagsCombineLemmaAndOwnGrammemes guards against the
// tag-corruption bug documented in docs/code-review-pre-1.0.md and fixed
// per docs/en/superpowers/specs/2026-09-15-opencorpora-tag-fix-design.md:
// each form's tag must be its lemma's own grammemes plus its own — not
// empty, not a previous form's, not an accumulating mixture.
func TestImportFromXMLFormTagsCombineLemmaAndOwnGrammemes(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(testDictXML), nil)
	require.NoError(t, err)
	require.NotNil(t, d.TagSet)

	want := []string{
		"NOUN,anim,masc,sing,nomn", // lemma "кот" (noun), form "кот"
		"NOUN,anim,masc,sing,gent", // lemma "кот" (noun), form "кота"
		"VERB,impf,trans",          // lemma "кот" (verb), form "кот" (no own grammemes)
		"NOUN,fem,sing,anim,nomn",  // lemma "мышь", form "мышь"
		"NOUN,fem,sing,anim,gent",  // lemma "мышь", form "мыши"
	}
	for _, tag := range want {
		assert.Contains(t, d.TagSet.Tags, tag, "tag %q must be registered", tag)
	}

	for _, tag := range d.TagSet.Tags {
		assert.NotEqual(t, "", tag, "no form should get an empty tag")
	}
}
```

- [ ] **Step 4: Run the new test and confirm it fails against the current (unfixed) handler**

Run: `go test ./pkg/morphology/importers/opencorpora/... -run TestImportFromXMLFormTagsCombineLemmaAndOwnGrammemes -v`

Expected: **FAIL**. Tracing the current (pre-fix) `xmlHandler` logic against the new fixture: `OnLemmaHeadEnd` (still has the old body from Task 1) wipes `curGrams` right after `<l>`'s own `<g>` refs are collected and before any `<f>` is parsed, so lemma 1's first form ("кот") gets tag `""` instead of `"NOUN,anim,masc,sing,nomn"`, and its second form ("кота") gets tag `"nomn"` (leaked from the first form's own `<g>`, appended too late) instead of `"NOUN,anim,masc,sing,gent"`. The `assert.Contains` checks fail and/or the `assert.NotEqual(t, "", tag, ...)` check fails on the empty first-form tag.

- [ ] **Step 5: Replace `xmlHandler`'s struct and methods**

In `pkg/morphology/importers/opencorpora/import.go`, replace the whole block from the `xmlHandler` type definition through `OnFormEnd` (currently):

```go
// xmlHandler implements xmlscan.Handler to collect lemmas and forms.
type xmlHandler struct {
	tagSet   *internal.TagSet
	lemmas   *[]lemmaEntry
	curForm  *formGrams
	err      error
	curGrams []string
	// lGrams is declared but never populated or read — it predates the fix
	// for the OpenCorpora tag bug (docs/code-review-pre-1.0.md), where
	// lemma-level grammemes never reach any form's tag. Whoever designs that
	// fix should decide whether a field like this is still needed.
	lGrams []string
}

func (h *xmlHandler) OnGrammeme(_ []byte, name []byte) error {
	return nil
}

func (h *xmlHandler) OnGrammemeRef(value []byte) error {
	if len(value) > 0 {
		h.curGrams = append(h.curGrams, string(value))
	}
	return nil
}

func (h *xmlHandler) OnLemma(id uint32, text []byte) error {
	lem := lemmaEntry{id: id, text: string(text)}
	*h.lemmas = append(*h.lemmas, lem)
	h.curForm = nil
	h.curGrams = nil
	return nil
}

func (h *xmlHandler) OnLemmaHeadEnd() error {
	h.curForm = nil
	h.curGrams = nil
	return nil
}

func (h *xmlHandler) OnForm(text []byte) error {
	if h.lemmas == nil || len(*h.lemmas) == 0 {
		return nil
	}
	lem := &(*h.lemmas)[len(*h.lemmas)-1]
	gramm := strings.Join(h.curGrams, ",")
	frm := formGrams{text: string(text), gramm: gramm}
	lem.forms = append(lem.forms, frm)
	h.curForm = &lem.forms[len(lem.forms)-1]
	return nil
}

func (h *xmlHandler) OnFormEnd() error {
	h.curForm = nil
	return nil
}
```

with:

```go
// xmlHandler implements xmlscan.Handler to collect lemmas and forms.
//
// Grammemes for a lemma's headword (<l>) and for each of its forms (<f>)
// are collected into separate buffers (lemGrams, formOwnGrams) because
// they arrive interleaved across many XML events; a form's own <g>
// children are only fully known once </f> closes, so its final tag is
// assembled on OnFormEnd, not on OnForm. See
// docs/en/superpowers/specs/2026-09-15-opencorpora-tag-fix-design.md.
type xmlHandler struct {
	tagSet *internal.TagSet
	lemmas *[]lemmaEntry
	err    error

	lemGrams     []string // current lemma's own grammemes (from <l>); persist across all its forms
	formOwnGrams []string // current form's own grammemes (from <f>); reset on every OnForm. Named
	// formOwnGrams, not formGrams, to avoid colliding in spirit with the
	// package's existing formGrams *type* (used below and in lem.forms).
	curFormText string // current form's text, held between OnForm and OnFormEnd
	inLemmaHead bool   // true between OnLemma and OnLemmaHeadEnd (i.e. while inside <l>)
}

func (h *xmlHandler) OnGrammeme(_ []byte, name []byte) error {
	return nil
}

// OnGrammemeRef records one <g v="..."/> reference, routing it to the
// lemma's own grammemes while inside <l> (inLemmaHead) or to the current
// form's own grammemes while inside <f>.
func (h *xmlHandler) OnGrammemeRef(value []byte) error {
	if len(value) == 0 {
		return nil
	}
	if h.inLemmaHead {
		h.lemGrams = append(h.lemGrams, string(value))
	} else {
		h.formOwnGrams = append(h.formOwnGrams, string(value))
	}
	return nil
}

// OnLemma fires when <l> (the lemma's headword) opens: starts a new
// lemmaEntry and resets both grammeme buffers for it.
func (h *xmlHandler) OnLemma(id uint32, text []byte) error {
	lem := lemmaEntry{id: id, text: string(text)}
	*h.lemmas = append(*h.lemmas, lem)
	h.lemGrams = nil
	h.formOwnGrams = nil
	h.inLemmaHead = true
	return nil
}

// OnLemmaHeadEnd fires when </l> closes (see xmlscan.Handler's doc comment
// — this is not the end of the enclosing <lemma>). It only flips
// inLemmaHead off; lemGrams is deliberately NOT cleared here, since every
// form of this lemma still needs it.
func (h *xmlHandler) OnLemmaHeadEnd() error {
	h.inLemmaHead = false
	return nil
}

// OnForm fires when <f> opens: starts a fresh per-form grammeme buffer.
// The form's tag is not built here — its own <g> children haven't been
// parsed yet at this point; see OnFormEnd.
func (h *xmlHandler) OnForm(text []byte) error {
	h.formOwnGrams = nil
	h.curFormText = string(text)
	return nil
}

// OnFormEnd fires when </f> closes: the form's own grammemes are now fully
// known, so this is where the final tag is assembled — lemma grammemes
// first, then this form's own, in XML declaration order (no sorting, no
// dedup, per the design spec) — and appended to the current lemma's forms.
func (h *xmlHandler) OnFormEnd() error {
	if h.lemmas == nil || len(*h.lemmas) == 0 {
		return nil
	}
	lem := &(*h.lemmas)[len(*h.lemmas)-1]
	all := make([]string, 0, len(h.lemGrams)+len(h.formOwnGrams))
	all = append(all, h.lemGrams...)
	all = append(all, h.formOwnGrams...)
	gramm := strings.Join(all, ",")
	lem.forms = append(lem.forms, formGrams{text: h.curFormText, gramm: gramm})
	return nil
}
```

- [ ] **Step 6: Update `ImportFromXML`'s handler construction**

In the same file, inside `ImportFromXML`, replace:

```go
	handler := &xmlHandler{
		tagSet:  tagSet,
		lemmas:  &lemmas,
		curForm: nil,
		err:     nil,
	}
```

with:

```go
	handler := &xmlHandler{
		tagSet: tagSet,
		lemmas: &lemmas,
		err:    nil,
	}
```

- [ ] **Step 7: Run the new test again and confirm it passes**

Run: `go test ./pkg/morphology/importers/opencorpora/... -run TestImportFromXMLFormTagsCombineLemmaAndOwnGrammemes -v`
Expected: PASS.

- [ ] **Step 8: Run the full package test suite**

Run: `go test ./pkg/morphology/... -race -v`
Expected: all PASS, including every test in `import_test.go` that was already there before this task (`TestImportFromXMLBasic`, `TestImportFromXMLParadigmsDedup`, `TestImportFromXMLStemLCP`, `TestImportFromXMLDAWGContains`, `TestImportFromXMLRoundtrip`, `TestImportFromXMLNoForms`, `TestImportFromXMLShardsOnSuffixOverflow`, `TestImportFromXMLPropertyTest`).

- [ ] **Step 9: Commit**

```bash
git add pkg/morphology/importers/opencorpora/import.go pkg/morphology/importers/opencorpora/import_test.go
git commit -m "$(cat <<'EOF'
opencorpora: fix corrupted word-form tags

Each form's tag is now its lemma's own grammemes (from <l>) plus its
own (from <f>), in XML declaration order — assembled on OnFormEnd,
once the form's own <g> children are actually known, instead of
prematurely on OnForm. Removes the dead curForm/lGrams fields.

Fixes the critical, release-blocking bug where the first form of
every lemma got an empty tag, lemma-level grammemes never reached any
form, and later forms accumulated a growing mixture of earlier forms'
own grammemes (docs/code-review-pre-1.0.md). Test fixtures updated to
match the real dict.xml schema (<l> and <f> as siblings, lemma
grammemes as nested <g> elements, not an attribute) — the old
attribute-based fixture never exercised the real lemma-grammeme code
path at all.

See docs/en/superpowers/specs/2026-09-15-opencorpora-tag-fix-design.md.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Full-suite verification and `docs/en/todo.md` status update

**Files:**
- Modify: `docs/en/todo.md` (mark the bug entry resolved)

**Interfaces:**
- Consumes: nothing new — this task only verifies and documents.

- [ ] **Step 1: Run the entire test suite**

Run: `go test ./... -race`
Expected: all PASS, no skips beyond the usual `-tags integration`-gated ones.

- [ ] **Step 2: Run `go build ./...` and `go vet ./...`**

Run: `go build ./... && go vet ./...`
Expected: no output, exit 0.

- [ ] **Step 3 (manual, optional): replay the documented real-world repro**

If a full rebuilt `.data/opencorpora/opencorpora.dat` is available (per
`docs/en/todo.md`'s existing build instructions), manually re-run the exact
repro from the critical-bug finding:

```bash
gomorphy -dict .data/opencorpora/opencorpora.dat lookup занудами
```

Expected: a plausible single-form tag (containing `plur` and `ablt`, not an
11-form concatenation of grammemes from unrelated forms). This step is
manual/optional verification against the real corpus, not part of the
automated test suite — skip it if the full dictionary isn't built in this
environment; Task 2's unit-level regression test already covers the
mechanism directly.

- [ ] **Step 4: Update `docs/en/todo.md`**

In `docs/en/todo.md`, find the section starting with:

```markdown
### Critical bug: corrupted OpenCorpora wordform tags — PLANNED (release-blocking)
```

Change the heading to:

```markdown
### Critical bug: corrupted OpenCorpora wordform tags — FIXED
```

At the end of that section (after the existing "Not investigated, but
mentioned by the user" paragraph), add:

```markdown
**Fix**: see [superpowers/specs/2026-09-15-opencorpora-tag-fix-design.md](superpowers/specs/2026-09-15-opencorpora-tag-fix-design.md)
and [superpowers/plans/2026-09-15-opencorpora-tag-fix.md](superpowers/plans/2026-09-15-opencorpora-tag-fix.md).
The bug's root cause — `</l>` (the lemma headword's closing tag) mistakenly
fired an event that was treated as "end of lemma" and wiped the lemma's
already-collected grammemes before a single form had been parsed; the
event was renamed (`OnLemmaHeadEnd`), and building the form's tag was
moved to `OnFormEnd`.
Both critical pre-1.0.0 code-review findings are now closed — all that's
left is CLI grooming before Stage 18.
```

Also update the row for this item in the "Path to version 1.0.0" section
(item 2) — change:

```markdown
   ~~Suffixes exceeding uint16's capacity~~ — DONE (sharding),
   see below. **One critical bug remains — corrupted OpenCorpora
   wordform tags; a decision is needed before release.**
```

to:

```markdown
   ~~Suffixes exceeding uint16's capacity~~ — DONE (sharding);
   ~~corrupted OpenCorpora wordform tags~~ — DONE. Both
   critical code-review findings are closed.
```

- [ ] **Step 5: Commit**

```bash
git add docs/en/todo.md
git commit -m "$(cat <<'EOF'
docs: mark OpenCorpora tag-corruption bug fixed

Both critical findings from the pre-1.0 code review are now closed
(suffix sharding, tag corruption). Only CLI grooming remains before
Stage 18 and the 1.0.0 release.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review Notes

- **Spec coverage:** every section of the design spec maps to a task —
  Context/Root cause → Task 2's fixture+test (demonstrates the bug
  mechanically); Decision (rename) → Task 1; Design (event flow table,
  `xmlHandler` fields/methods) → Task 2 Steps 5-6; `xmlscan` changes →
  Task 1; test fixture → Task 2 Step 1-2; Non-goals → Global Constraints
  and left untouched; Testing → Task 2 Steps 3-4,7-8 and Task 3 Step 3;
  Risks (fixture previously not exercising the real path, `TagSet` dedup
  by string equality) → addressed by Task 2's fixture rewrite and by
  Global Constraints stating no canonicalization is introduced.
- **Placeholder scan:** no TBD/TODO; every step has literal code or an
  exact command and expected output.
- **Type consistency:** `xmlHandler` field names (`lemGrams`,
  `formOwnGrams`, `curFormText`, `inLemmaHead`) are identical between
  Task 2 Step 3's test rationale, Step 5's implementation, and the design
  spec's table — no drift. Caught and fixed during this self-review:
  the field was initially named `formGrams`, colliding in spirit with
  the package's existing `formGrams` type (`lem.forms []formGrams`) —
  legal Go, but exactly the kind of misleading-naming this fix is
  supposed to eliminate, not reintroduce. Renamed to `formOwnGrams`
  throughout both this plan and the design spec.
