# Comparative-degree prefix split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop comparative-degree adjective paradigms (`COMP[,Qual]`/`Cmp2` forms) from exploding into one paradigm per lexical root, by splitting the `"по-"` prefix off `Cmp2`-tagged forms before computing the stem — and along the way, make the stem computation rune-safe so it never again produces invalid-UTF-8 suffix strings.

**Architecture:** In `pkg/morphology/importers/opencorpora/import.go`'s per-lemma loop, classify each form by its `Cmp2` grammeme, strip a literal `"по"` prefix from `Cmp2` forms before LCP, and register a per-form prefix ID (`""` or `"по"`) alongside the existing suffix/tag IDs. `paradigmKeyHash` grows a third argument (prefix IDs) so paradigms no longer collapse across different prefix sequences. The on-disk format and the read path (`pkg/morphology/parse.go`) already support this (prefix/suffix/tag triples per form) — this plan only changes what the OpenCorpora importer *writes*.

**Tech Stack:** Go (stdlib `strings`, `unicode/utf8`), existing `github.com/stretchr/testify` for fixture-based tests, plain table-driven tests for white-box unit tests (matching `shard_test.go`'s existing style).

**Spec:** [docs/superpowers/specs/2026-09-15-comparative-prefix-split-design.md](../specs/2026-09-15-comparative-prefix-split-design.md)

## Global Constraints

- Narrow scope: only the `Cmp2` grammeme → literal `"по"` prefix. No generic/configurable prefix table.
- `Cmp2` form text that does not literally start with `"по"` must not fail the whole import — fall back to no-prefix-split for that lemma only, and this must be independently unit-testable.
- `paradigmKeyHash` must hash prefix IDs, suffix IDs, and tag IDs together — this is the same class of bug as the precedent fix in `docs/todo.md` ("баг 2": `n = copy(...)` instead of `n += copy(...)`).
- No changes to `pkg/morphology/parse.go` or any other read-path code — it already handles non-empty prefixes correctly (proven by the pymorphy2 importer).
- No `.dat` backward compatibility to preserve (pre-1.0, no released format).
- `go test ./... -race` must stay green after every task.

---

## Task 1: Rune-safe `lcp()`, operating on `[]string`

**Files:**
- Create: `pkg/morphology/importers/opencorpora/internal_test.go`
- Modify: `pkg/morphology/importers/opencorpora/import.go:6-14` (imports), `:136` (call site), `:349-373` (`lcp` function)

**Interfaces:**
- Produces: `func lcp(texts []string) string` — longest common prefix of `texts`, trimmed back to the nearest valid UTF-8 rune boundary. Replaces the old `func lcp(forms []formGrams) string`.

- [ ] **Step 1: Write the failing test**

Create `pkg/morphology/importers/opencorpora/internal_test.go`:

```go
package opencorpora

import (
	"testing"
	"unicode/utf8"
)

func TestLCP(t *testing.T) {
	cases := []struct {
		name  string
		texts []string
		want  string
	}{
		{"empty", nil, ""},
		{"single", []string{"кот"}, "кот"},
		{"identical", []string{"дом", "дом"}, "дом"},
		{"simple shared prefix", []string{"кот", "кота"}, "кот"},
		{"no shared prefix", []string{"кот", "мышь"}, ""},
		{
			// "абажурнее" (а = d0 b0) vs "побажурнее" (п = d0 bf): the
			// first byte of both matches (d0), the second byte differs
			// (b0 vs bf). A naive byte-level LCP would cut at 1 byte,
			// splitting the multi-byte characters "а"/"п" in half. The
			// correct LCP is "" — the first character differs.
			name:  "byte-boundary collision does not produce a mid-rune cut",
			texts: []string{"абажурнее", "побажурнее"},
			want:  "",
		},
		{
			// "поправимее" vs "попоправимее": the shared prefix is the
			// three whole characters "поп" (6 bytes) — the 7th byte
			// starts a new, different character in both texts, so no
			// trimming is needed here; this case guards against an
			// over-eager fix that trims valid boundaries too.
			name:  "shared multi-rune prefix stays intact",
			texts: []string{"поправимее", "попоправимее"},
			want:  "поп",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lcp(tc.texts)
			if got != tc.want {
				t.Errorf("lcp(%v) = %q, want %q", tc.texts, got, tc.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("lcp(%v) = %q is not valid UTF-8", tc.texts, got)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/morphology/importers/opencorpora/... -run TestLCP -v`
Expected: compile error — `lcp` still takes `[]formGrams`, not `[]string` (this counts as the "red" state: the test as written cannot pass against the current code).

- [ ] **Step 3: Change `lcp()`'s signature and make it rune-safe**

In `pkg/morphology/importers/opencorpora/import.go`, replace the `lcp` function (currently lines 349-373):

```go
// lcp computes the longest common prefix of texts, trimmed back to the
// nearest valid UTF-8 rune boundary so the result (and therefore every
// "suffix = text minus this prefix") is always valid UTF-8. A raw
// byte-level cut can otherwise land inside a multi-byte character when
// two texts share a lead byte but differ in its continuation byte (e.g.
// any Cyrillic letter in the а-п block compared against "по") — see
// docs/research/0003-comparative-paradigms-not-merging.md, section 4/6.
func lcp(texts []string) string {
	if len(texts) == 0 {
		return ""
	}
	if len(texts) == 1 {
		return texts[0]
	}
	n := len(texts[0])
	for i := 1; i < len(texts); i++ {
		max := n
		if len(texts[i]) < max {
			max = len(texts[i])
		}
		j := 0
		for j < max && texts[0][j] == texts[i][j] {
			j++
		}
		n = j
		if n == 0 {
			return ""
		}
	}
	for n > 0 && n < len(texts[0]) && !utf8.RuneStart(texts[0][n]) {
		n--
	}
	return texts[0][:n]
}
```

Add `"unicode/utf8"` to the import block (`import.go:6-14`):

```go
import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/amarin/gomorphy/internal/xmlscan"
	"github.com/amarin/gomorphy/pkg/morphology/internal"
)
```

- [ ] **Step 4: Update the call site**

In `ImportFromXML`, replace line 136 (`stem := lcp(lem.forms)`) with:

```go
		texts := make([]string, len(lem.forms))
		for i, f := range lem.forms {
			texts[i] = f.text
		}
		stem := lcp(texts)
```

(This is a temporary, throwaway extraction — Task 2 replaces it with `stripCmp2Prefix`'s output.)

- [ ] **Step 5: Run the new test to verify it passes**

Run: `go test ./pkg/morphology/importers/opencorpora/... -run TestLCP -v`
Expected: PASS, all subtests green.

- [ ] **Step 6: Run the full existing test suite to check for regressions**

Run: `go test ./pkg/morphology/... -race`
Expected: PASS, no regressions (the byte-boundary fix does not change behavior for any case in the existing `testDictXML` fixture — verified by inspection: none of its lemmas hit the byte-collision case).

- [ ] **Step 7: Commit**

```bash
git add pkg/morphology/importers/opencorpora/import.go pkg/morphology/importers/opencorpora/internal_test.go
git commit -m "opencorpora: make lcp() rune-safe, operate on []string

Fixes the byte-vs-rune LCP bug from docs/research/0003-comparative-paradigms-not-merging.md
(section 4/6): a raw byte-level cut could land inside a multi-byte
UTF-8 character, producing invalid suffix strings for any lemma whose
forms diverge on the first character within the same lead byte (e.g.
letters in the а-п block vs по-). lcp() now trims its result back to
the nearest valid rune boundary."
```

---

## Task 2: Split the `Cmp2` prefix before computing the stem

**Files:**
- Modify: `pkg/morphology/importers/opencorpora/import.go` (`paradigmKeyHash`, main loop in `ImportFromXML`, new `stripCmp2Prefix` function)
- Modify: `pkg/morphology/importers/opencorpora/internal_test.go` (new tests)
- Modify: `pkg/morphology/importers/opencorpora/import_test.go` (new fixture + tests)

**Interfaces:**
- Consumes: `formGrams{text, gramm string}` (existing type), `lcp(texts []string) string` (Task 1).
- Produces: `func stripCmp2Prefix(forms []formGrams) (stemInput []string, prefixes []string, ok bool)`; `func paradigmKeyHash(pk, sk, tk []uint16) string` (signature change — was `(sk, tk []uint16)`).

- [ ] **Step 1: Write the failing fixture-based test (paradigm merge)**

In `pkg/morphology/importers/opencorpora/import_test.go`, add a new fixture constant and test, after `testDictXML`'s declaration (after line 63):

```go
// comparativeDictXML has two comparative-degree adjectives ("яснее",
// "плотнее") with different roots but the same tag pattern (COMP,Qual /
// COMP,Qual,V-ej / COMP,Qual,Cmp2 / COMP,Qual,Cmp2,V-ej) — real
// OpenCorpora data shows this pattern accounts for 82% of shard 0's
// paradigms (docs/research/0003-comparative-paradigms-not-merging.md).
// The Cmp2-tagged forms are literally "по" + the corresponding
// non-Cmp2 form, matching the real dict.xml lemma "поправимее" (id
// 259490) verified in that document.
const comparativeDictXML = `<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="ADJF">прилагательное</grammeme>
  <grammeme id="COMP">сравнит. степень</grammeme>
  <grammeme id="Qual">качественное</grammeme>
  <grammeme id="Cmp2">по-сравнит.</grammeme>
  <grammeme id="V-ej">форма на -ей</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text="яснее">
   <l t="яснее"><g v="COMP"/><g v="Qual"/></l>
   <f t="яснее"/>
   <f t="ясней"><g v="V-ej"/></f>
   <f t="пояснее"><g v="Cmp2"/></f>
   <f t="поясней"><g v="Cmp2"/><g v="V-ej"/></f>
  </lemma>
  <lemma id="2" text="плотнее">
   <l t="плотнее"><g v="COMP"/><g v="Qual"/></l>
   <f t="плотнее"/>
   <f t="плотней"><g v="V-ej"/></f>
   <f t="поплотнее"><g v="Cmp2"/></f>
   <f t="поплотней"><g v="Cmp2"/><g v="V-ej"/></f>
  </lemma>
 </lemmata>
</dictionary>`

// TestImportFromXMLComparativeParadigmsMerge guards against the
// root-in-suffix duplication documented in
// docs/research/0003-comparative-paradigms-not-merging.md: "яснее" and
// "плотнее" share suffix/tag/prefix structure once "по" is split off
// as a real prefix, and must collapse into ONE paradigm.
func TestImportFromXMLComparativeParadigmsMerge(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(comparativeDictXML), nil)
	require.NoError(t, err)
	require.Len(t, d.Paradigms, 1)

	assert.Equal(t, 1, len(d.Paradigms[0]),
		"яснее и плотнее делят один паттерн словоизменения и должны схлопнуться в одну парадигму, несмотря на разные корни")

	assert.Contains(t, d.Prefixes, "по", "по-приставка Cmp2-форм должна попасть в таблицу префиксов")

	for _, word := range []string{"яснее", "ясней", "пояснее", "поясней", "плотнее", "плотней", "поплотнее", "поплотней"} {
		items := d.Words[0].SimilarItems(word, d.CharPolicy)
		assert.Greater(t, len(items), 0, "%q должно быть найдено", word)
	}
}
```

Also add, right after `tagsForWord` (after line 168), a sibling helper and a normal-form regression test:

```go
// normalFormsForWord resolves word to every normalized (citation) form
// it can produce, replicating exactly what
// (*morphology.Dictionary).readingForm does (pkg/morphology/parse.go):
// decode (paraID, formIdx) from the DAWG payload, then
// norm = prefix(form0) + TrimSuffix(TrimPrefix(word, prefix(form)), suffix(form)) + suffix(form0).
// This exercises the read path with real (non-empty) prefixes, which
// tagsForWord alone does not.
func normalFormsForWord(t *testing.T, d *internal.Dictionary, shard int, word string) []string {
	t.Helper()
	require.Less(t, shard, len(d.Words))

	strAt := func(list []string, id uint16) string {
		if int(id) < len(list) {
			return list[id]
		}
		return ""
	}

	var norms []string
	for _, it := range d.Words[shard].SimilarItems(word, d.CharPolicy) {
		if it.Key != word {
			continue
		}
		for _, v := range it.Values {
			require.Len(t, v, 4)
			paraID := binary.BigEndian.Uint16(v[:2])
			formIdx := binary.BigEndian.Uint16(v[2:4])
			require.Less(t, int(paraID), len(d.Paradigms[shard]))
			para := d.Paradigms[shard][paraID]
			require.Less(t, int(formIdx), para.Len())

			ownPrefix := strAt(d.Prefixes, para.Prefix(int(formIdx)))
			ownSuffix := strAt(d.Suffixes[shard], para.Suffix(int(formIdx)))
			stem := strings.TrimSuffix(strings.TrimPrefix(word, ownPrefix), ownSuffix)

			p0 := strAt(d.Prefixes, para.Prefix(0))
			s0 := strAt(d.Suffixes[shard], para.Suffix(0))
			norms = append(norms, p0+stem+s0)
		}
	}
	return norms
}

// TestImportFromXMLComparativeParadigmsNormalFormPerLemma guards
// against the shared paradigm silently mixing up which lemma a word
// belongs to (the same failure shape as the precedent
// paradigmKeyHash bug covered by TestImportFromXMLWordResolvesOwnLemmaTag):
// "поясней" and "поплотней" share one paradigm after this fix, but
// each must still resolve to its OWN lemma's normal form.
func TestImportFromXMLComparativeParadigmsNormalFormPerLemma(t *testing.T) {
	d, err := opencorpora.CompileFromXML(strings.NewReader(comparativeDictXML), nil)
	require.NoError(t, err)
	require.Len(t, d.Words, 1)

	assert.Contains(t, normalFormsForWord(t, d, 0, "поясней"), "яснее",
		"поясней должно нормализоваться к 'яснее', не к 'плотнее'")
	assert.Contains(t, normalFormsForWord(t, d, 0, "поплотней"), "плотнее",
		"поплотней должно нормализоваться к 'плотнее', не к 'яснее'")
}
```

- [ ] **Step 2: Run the new tests to verify they fail**

Run: `go test ./pkg/morphology/importers/opencorpora/... -run TestImportFromXMLComparative -v`
Expected: FAIL — `TestImportFromXMLComparativeParadigmsMerge` fails on `assert.Equal(t, 1, len(d.Paradigms[0]))` (currently produces 2 separate paradigms, since nothing strips `"по"` yet) and on `assert.Contains(t, d.Prefixes, "по")` (currently `d.Prefixes` is `nil`).

- [ ] **Step 3: Write `stripCmp2Prefix` and its unit tests**

Append to `pkg/morphology/importers/opencorpora/internal_test.go`:

```go
func TestStripCmp2Prefix(t *testing.T) {
	cases := []struct {
		name          string
		forms         []formGrams
		wantStemInput []string
		wantPrefixes  []string
		wantOK        bool
	}{
		{
			name: "no Cmp2 forms",
			forms: []formGrams{
				{text: "кот", gramm: "NOUN,anim,masc,sing,nomn"},
				{text: "кота", gramm: "NOUN,anim,masc,sing,gent"},
			},
			wantStemInput: []string{"кот", "кота"},
			wantPrefixes:  []string{"", ""},
			wantOK:        true,
		},
		{
			name: "Cmp2 forms strip по",
			forms: []formGrams{
				{text: "яснее", gramm: "COMP,Qual"},
				{text: "ясней", gramm: "COMP,Qual,V-ej"},
				{text: "пояснее", gramm: "COMP,Qual,Cmp2"},
				{text: "поясней", gramm: "COMP,Qual,Cmp2,V-ej"},
			},
			wantStemInput: []string{"яснее", "ясней", "яснее", "ясней"},
			wantPrefixes:  []string{"", "", "по", "по"},
			wantOK:        true,
		},
		{
			name: "Cmp2 form without по prefix falls back for the whole lemma",
			forms: []formGrams{
				{text: "яснее", gramm: "COMP,Qual"},
				{text: "СЛОМАНО", gramm: "COMP,Qual,Cmp2"},
			},
			wantStemInput: []string{"яснее", "СЛОМАНО"},
			wantPrefixes:  []string{"", ""},
			wantOK:        false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stemInput, prefixes, ok := stripCmp2Prefix(tc.forms)
			if ok != tc.wantOK {
				t.Errorf("ok = %v, want %v", ok, tc.wantOK)
			}
			if len(stemInput) != len(tc.wantStemInput) {
				t.Fatalf("stemInput = %v, want %v", stemInput, tc.wantStemInput)
			}
			for i := range stemInput {
				if stemInput[i] != tc.wantStemInput[i] {
					t.Errorf("stemInput[%d] = %q, want %q", i, stemInput[i], tc.wantStemInput[i])
				}
			}
			if len(prefixes) != len(tc.wantPrefixes) {
				t.Fatalf("prefixes = %v, want %v", prefixes, tc.wantPrefixes)
			}
			for i := range prefixes {
				if prefixes[i] != tc.wantPrefixes[i] {
					t.Errorf("prefixes[%d] = %q, want %q", i, prefixes[i], tc.wantPrefixes[i])
				}
			}
		})
	}
}

func TestParadigmKeyHashIncludesPrefix(t *testing.T) {
	sk := []uint16{0, 1}
	tk := []uint16{5, 6}
	h1 := paradigmKeyHash([]uint16{0, 0}, sk, tk)
	h2 := paradigmKeyHash([]uint16{0, 1}, sk, tk)
	if h1 == h2 {
		t.Errorf("paradigmKeyHash must differ when prefix IDs differ (same suffix+tag IDs): got equal hashes for %v vs %v", h1, h2)
	}
}
```

Run: `go test ./pkg/morphology/importers/opencorpora/... -run 'TestStripCmp2Prefix|TestParadigmKeyHashIncludesPrefix' -v`
Expected: compile error — `stripCmp2Prefix` does not exist yet, `paradigmKeyHash` still takes 2 arguments.

Now implement. In `pkg/morphology/importers/opencorpora/import.go`, replace `paradigmKeyHash` (currently lines 40-48):

```go
func paradigmKeyHash(pk, sk, tk []uint16) string {
	if len(pk) == 0 && len(sk) == 0 && len(tk) == 0 {
		return ""
	}
	buf := make([]byte, 0, len(pk)*2+len(sk)*2+len(tk)*2)
	buf = append(buf, encodeU16s(pk)...)
	buf = append(buf, encodeU16s(sk)...)
	buf = append(buf, encodeU16s(tk)...)
	return string(buf)
}
```

Add `stripCmp2Prefix` right after `lcp` (after the function added in Task 1):

```go
// cmp2Prefix is the only lemma-internal separable prefix present in
// OpenCorpora's dict.xml (verified against the real file — see
// docs/superpowers/specs/2026-09-15-comparative-prefix-split-design.md):
// the Cmp2 grammeme marks a comparative-degree form that is literally
// "по" + the corresponding non-Cmp2 form (e.g. lemma "поправимее",
// dict.xml id 259490: Cmp2 form "попоправимее" = "по" + "поправимее").
const cmp2Prefix = "по"

// stripCmp2Prefix separates each form's Cmp2-driven prefix ("по" or
// "") from the text that should feed lcp(), so the shared root never
// ends up split across different suffix strings depending on which
// forms happen to carry that prefix (the root-in-suffix problem from
// docs/research/0003-comparative-paradigms-not-merging.md).
//
// If any Cmp2-tagged form's text does not literally start with "по"
// (an anomaly not observed against the real dict.xml — see the
// real_dict_integration_test.go check in Task 3 — but not assumed
// impossible), every form of this lemma falls back to prefix "" and
// its own full text, exactly the pre-fix behavior, and ok is false so
// the caller can detect and count it.
func stripCmp2Prefix(forms []formGrams) (stemInput []string, prefixes []string, ok bool) {
	stemInput = make([]string, len(forms))
	prefixes = make([]string, len(forms))
	ok = true
	for i, f := range forms {
		if !strings.Contains(f.gramm, "Cmp2") {
			stemInput[i] = f.text
			continue
		}
		if !strings.HasPrefix(f.text, cmp2Prefix) {
			ok = false
			break
		}
		prefixes[i] = cmp2Prefix
		stemInput[i] = f.text[len(cmp2Prefix):]
	}
	if !ok {
		for i, f := range forms {
			stemInput[i] = f.text
			prefixes[i] = ""
		}
	}
	return stemInput, prefixes, ok
}
```

Run: `go test ./pkg/morphology/importers/opencorpora/... -run 'TestStripCmp2Prefix|TestParadigmKeyHashIncludesPrefix' -v`
Expected: PASS.

- [ ] **Step 4: Wire `stripCmp2Prefix` and the prefix table into the main loop**

In `ImportFromXML`, replace the whole Phase 2 loop body. First, insert the shared prefix-table accumulator right after `shards := []*shardBuild{cur}` (currently line 129) and before `for _, lem := range lemmas {`:

```go
	// Prefixes are shared across ALL shards (unlike suffixes/paradigms,
	// which are per-shard) — see the doc comment on paradigmAffix in
	// pkg/morphology/parse.go. Index 0 is always the empty prefix.
	prefixTexts := map[string]uint16{"": 0}
	prefixList := []string{""}
```

Then replace the entire loop body (currently lines 131-207, from `for _, lem := range lemmas {` through its closing `}`) with:

```go
	for _, lem := range lemmas {
		if len(lem.forms) == 0 {
			continue
		}

		stemInput, formPrefixes, _ := stripCmp2Prefix(lem.forms)
		stem := lcp(stemInput)

		// How many suffixes would this lemma add to the CURRENT shard if
		// placed there? Dedup within the lemma's own forms too, so a
		// lemma reusing one suffix across several forms counts once.
		newInLemma := make(map[string]bool)
		for _, si := range stemInput {
			suffix := ""
			if len(stem) < len(si) {
				suffix = si[len(stem):]
			}
			if _, ok := cur.suffixTexts[suffix]; !ok {
				newInLemma[suffix] = true
			}
		}

		if strategy.Boundary(len(cur.suffixTexts), len(newInLemma), suffixShardLimit) {
			cur = newShardBuild()
			shards = append(shards, cur)
		}

		var prefixIDs []uint16
		var suffixIDs []uint16
		var tagIDs []uint16

		for i, frm := range lem.forms {
			suffix := ""
			if len(stem) < len(stemInput[i]) {
				suffix = stemInput[i][len(stem):]
			}

			pid, ok := prefixTexts[formPrefixes[i]]
			if !ok {
				pid = uint16(len(prefixList))
				prefixTexts[formPrefixes[i]] = pid
				prefixList = append(prefixList, formPrefixes[i])
			}
			prefixIDs = append(prefixIDs, pid)

			sid, ok := cur.suffixTexts[suffix]
			if !ok {
				if len(cur.suffixList) >= suffixShardLimit {
					return nil, fmt.Errorf("opencorpora: shard %d exceeded %d unique suffixes despite sharding strategy", len(shards)-1, suffixShardLimit)
				}
				sid = uint16(len(cur.suffixList))
				cur.suffixTexts[suffix] = sid
				cur.suffixList = append(cur.suffixList, suffix)
			}
			suffixIDs = append(suffixIDs, sid)

			tid, err := tagSet.Add(frm.gramm)
			if err != nil {
				return nil, fmt.Errorf("opencorpora: %w", err)
			}
			tagIDs = append(tagIDs, tid)
		}

		hash := paradigmKeyHash(prefixIDs, suffixIDs, tagIDs)
		paraID, ok := cur.paradigmsDedup[hash]
		if !ok {
			if len(cur.paradigms) >= 1<<16 {
				return nil, fmt.Errorf("opencorpora: shard %d exceeded 65536 unique paradigms", len(shards)-1)
			}
			paraID = uint16(len(cur.paradigms))
			cur.paradigmsDedup[hash] = paraID

			para := internal.NewParadigm(suffixIDs, tagIDs, prefixIDs)
			cur.paradigms = append(cur.paradigms, para)
		}

		for formIdx := 0; formIdx < len(lem.forms); formIdx++ {
			suffix := ""
			if len(stem) < len(stemInput[formIdx]) {
				suffix = stemInput[formIdx][len(stem):]
			}
			dawgKey := formPrefixes[formIdx] + stem + suffix
			val := uint32(paraID)<<16 | uint32(formIdx)
			cur.dawgEntries = append(cur.dawgEntries, dawgEntry{key: dawgKey, val: val})
		}
	}
```

(`dawgKey` is now `prefix + stem + suffix` instead of `stem + suffix` — for non-`Cmp2` forms `formPrefixes[formIdx]` is `""` so this is unchanged; for `Cmp2` forms it reconstructs the original literal word text, e.g. `"по" + "ясн" + "ей" = "поясней"`.)

Finally, replace the `internal.NewDictionary` call (currently lines 247-255):

```go
	dict := internal.NewDictionary(
		"ru",
		tagSet,
		suffixesPerShard,
		prefixList,
		paradigmsPerShard,
		wordsPerShard,
		internal.RussianCharPolicy(),
	)
```

- [ ] **Step 5: Run the fixture-based tests to verify they pass**

Run: `go test ./pkg/morphology/importers/opencorpora/... -run TestImportFromXMLComparative -v`
Expected: PASS — both `TestImportFromXMLComparativeParadigmsMerge` and `TestImportFromXMLComparativeParadigmsNormalFormPerLemma`.

- [ ] **Step 6: Run the full existing test suite to check for regressions**

Run: `go test ./pkg/morphology/... -race -v 2>&1 | tail -100`
Expected: PASS, including all pre-existing tests in `import_test.go` (`TestImportFromXMLParadigmsDedup` must still assert exactly 4 — `testDictXML` itself is untouched by this task; the new lemmas live in the separate `comparativeDictXML` fixture).

- [ ] **Step 7: Commit**

```bash
git add pkg/morphology/importers/opencorpora/import.go pkg/morphology/importers/opencorpora/internal_test.go pkg/morphology/importers/opencorpora/import_test.go
git commit -m "opencorpora: split Cmp2 (\"по-\") prefix off before computing stem

Fixes the root-in-suffix paradigm explosion from
docs/research/0003-comparative-paradigms-not-merging.md: comparative-degree
adjective paradigms no longer duplicate per lexical root, because the
Cmp2 grammeme's literal \"по\" prefix is now split off before LCP
instead of ending up inside the suffix string. paradigmKeyHash now
hashes prefix IDs too, so paradigms with the same suffix+tag sequence
but different prefixes correctly stay distinct."
```

---

## Task 3: Real-data verification

**Files:**
- Create: `pkg/morphology/importers/opencorpora/real_dict_integration_test.go`

**Interfaces:**
- Consumes: `xmlHandler`, `lemmaEntry`, `formGrams`, `stripCmp2Prefix` (Task 2), `CompileFromXML` (existing exported function).

- [ ] **Step 1: Write the anomaly-count integration test**

Create `pkg/morphology/importers/opencorpora/real_dict_integration_test.go`:

```go
//go:build integration

// Integration tests here require the real OpenCorpora dict.xml at
// .data/opencorpora/dict.xml (see docs/todo.md / Makefile's
// test-integration target). Run explicitly:
//
//	go test -tags=integration ./pkg/morphology/importers/opencorpora/... -v
package opencorpora

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amarin/gomorphy/internal/xmlscan"
	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

const (
	realDictEnvVar      = "GOMORPHY_DICT_XML"
	realDictDefaultPath = ".data/opencorpora/dict.xml"
)

func openRealDict(t *testing.T) *os.File {
	t.Helper()

	path := os.Getenv(realDictEnvVar)
	if path == "" {
		path = findRealDictXML()
		if path == "" {
			t.Skipf("dict.xml not found (%s to override)", realDictEnvVar)
		}
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open dict %q: %v", path, err)
	}
	return f
}

// findRealDictXML walks up from the working dir looking for
// .data/opencorpora/dict.xml, so this test passes both from the
// package dir and from the repo root (same approach as
// internal/xmlscan/integration_test.go).
func findRealDictXML() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, realDictDefaultPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// TestStripCmp2PrefixRealDictHasNoAnomalies verifies the assumption
// the narrow Cmp2-only fix depends on: every Cmp2-tagged form in the
// real dict.xml literally starts with "по". If a future dict.xml
// update breaks this, the fallback in stripCmp2Prefix still produces
// a correct (if non-merged) paradigm — but this test existing means
// that regression is caught loudly instead of silently reducing the
// fix's effect.
func TestStripCmp2PrefixRealDictHasNoAnomalies(t *testing.T) {
	f := openRealDict(t)
	defer func() { _ = f.Close() }()

	tagSet := internal.NewTagSet("opencorpora")
	var lemmas []lemmaEntry
	handler := &xmlHandler{tagSet: tagSet, lemmas: &lemmas}
	if err := xmlscan.New(f, handler).Scan(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if handler.err != nil {
		t.Fatalf("handler: %v", handler.err)
	}

	anomalies := 0
	var examples []string
	for _, lem := range lemmas {
		if _, _, ok := stripCmp2Prefix(lem.forms); !ok {
			anomalies++
			if len(examples) < 10 {
				examples = append(examples, lem.text)
			}
		}
	}
	if anomalies != 0 {
		t.Errorf("real dict.xml has %d lemmas where a Cmp2 form does not start with \"по\" "+
			"(examples: %v) — the narrow Cmp2-only fix assumed this doesn't happen; "+
			"see docs/superpowers/specs/2026-09-15-comparative-prefix-split-design.md",
			anomalies, examples)
	}
}

// TestImportFromXMLRealDictComparativeWordsResolve spot-checks known
// comparative-degree words from
// docs/research/0003-comparative-paradigms-not-merging.md against the
// real compiled dictionary: each word must be found, and — critically
// — must resolve to ITS OWN normal form even though it now shares a
// paradigm with unrelated lemmas (same regression shape as
// TestImportFromXMLComparativeParadigmsNormalFormPerLemma, on real
// data instead of a fixture).
func TestImportFromXMLRealDictComparativeWordsResolve(t *testing.T) {
	f := openRealDict(t)
	defer func() { _ = f.Close() }()

	d, err := CompileFromXML(f, nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	strAt := func(list []string, id uint16) string {
		if int(id) < len(list) {
			return list[id]
		}
		return ""
	}

	cases := []struct {
		word       string
		wantNormal string
	}{
		{"яснее", "яснее"},
		{"ясней", "яснее"},
		{"пояснее", "яснее"},
		{"поясней", "яснее"},
		{"абажурнее", "абажурнее"},
		{"поабажурнее", "абажурнее"}, // "по" + full base word, per docs/research/0003-...md's own example
		{"поправимее", "поправимее"},
		{"попоправимее", "поправимее"},
	}

	for _, tc := range cases {
		found := false
		for shard := range d.Words {
			for _, it := range d.Words[shard].SimilarItems(tc.word, d.CharPolicy) {
				if it.Key != tc.word {
					continue
				}
				for _, v := range it.Values {
					if len(v) != 4 {
						continue
					}
					paraID := binary.BigEndian.Uint16(v[:2])
					formIdx := binary.BigEndian.Uint16(v[2:4])
					if int(paraID) >= len(d.Paradigms[shard]) {
						continue
					}
					para := d.Paradigms[shard][paraID]
					if int(formIdx) >= para.Len() {
						continue
					}
					found = true

					ownPrefix := strAt(d.Prefixes, para.Prefix(int(formIdx)))
					ownSuffix := strAt(d.Suffixes[shard], para.Suffix(int(formIdx)))
					stem := strings.TrimSuffix(strings.TrimPrefix(tc.word, ownPrefix), ownSuffix)

					p0 := strAt(d.Prefixes, para.Prefix(0))
					s0 := strAt(d.Suffixes[shard], para.Suffix(0))
					norm := p0 + stem + s0
					if norm != tc.wantNormal {
						t.Errorf("%q resolved to normal form %q, want %q", tc.word, norm, tc.wantNormal)
					}
				}
			}
		}
		if !found {
			t.Errorf("%q not found in any shard", tc.word)
		}
	}
}
```

- [ ] **Step 2: Run the integration tests**

Run: `go test -tags=integration ./pkg/morphology/importers/opencorpora/... -run 'TestStripCmp2PrefixRealDictHasNoAnomalies|TestImportFromXMLRealDictComparativeWordsResolve' -v`
Expected: PASS. If `TestStripCmp2PrefixRealDictHasNoAnomalies` fails, stop and investigate the listed example lemmas before proceeding — this would mean the narrow fix's core assumption is wrong for some real lemmas.

- [ ] **Step 3: Record the real post-fix paradigm/suffix counts**

This step has no scripted assertion — the exact counts are not known ahead of implementation (see the spec's "Risks" section). Run:

```bash
./deploy/gomorphy_build compile
```

Then, using the same technique as the original research (`pkg/morphology/internal`'s exported `OpenContainer`/`DecodeStrings`/`DecodeParadigms`/`DecodeTagSet` functions, in a throwaway `_test.go` — see `docs/research/0003-comparative-paradigms-not-merging.md`'s own "Эксперимент" section for the exact method), record:

- Total paradigms and suffix-table bytes for shard 0/1, before vs. after.
- Confirm the two `COMP`/`COMP,Qual` tag groups now collapse to a small number of paradigms (not 14,735).
- Confirm `utf8.ValidString` is true for 100% of suffixes in both shards (closing the byte-cut bug's UTF-8-validity claim from the original research).

- [ ] **Step 4: Update the research document with real numbers**

Edit `docs/research/0003-comparative-paradigms-not-merging.md`, "Дополнение 2" section — replace the paragraph starting "Точный количественный эффект (сколько именно парадигм схлопнется после фикса) не пересчитан..." with the actual measured before/after numbers from Step 3, in the same table style used elsewhere in that document.

- [ ] **Step 5: Commit**

```bash
git add pkg/morphology/importers/opencorpora/real_dict_integration_test.go docs/research/0003-comparative-paradigms-not-merging.md
git commit -m "opencorpora: real-data verification for the Cmp2 prefix-split fix

Adds integration tests against the real dict.xml (anomaly count,
specific comparative-degree word resolution) and records the actual
post-fix paradigm/suffix counts in the research document, closing the
'not recomputed' caveat left open when the fix was designed."
```

---

## Task 4: Update the roadmap

**Files:**
- Modify: `docs/todo.md`

**Interfaces:** none (documentation only).

- [ ] **Step 1: Insert this fix into the pre-1.0.0 path**

In `docs/todo.md`, section "## Путь к версии 1.0.0", replace:

```markdown
Порядок до релиза (зафиксирован 2026-09-14):

1. ~~Формат: задел под расширяемое сжатие + секция info~~ — ВЫПОЛНЕНО.
2. ~~Ревью кода перед 1.0.0 + разбор находок~~ — ВЫПОЛНЕНО, см. ниже.
   ~~Суффиксы больше, чем вмещает uint16~~ — ВЫПОЛНЕНО (шардирование);
   ~~искажённые теги OpenCorpora-словоформ~~ — ВЫПОЛНЕНО, включая баг
   схлопывания парадигм в дедупликации (`paradigmKeyHash`), найденный
   финальным ревью всей ветки. Обе критические находки код-ревью
   закрыты.
3. Этап 18, но сначала — груминг CLI-команд (есть отдельные идеи,
   уточняются с пользователем до начала этапа).
4. Релиз 1.0.0.
```

with:

```markdown
Порядок до релиза (зафиксирован 2026-09-14, дополнен 2026-09-15):

1. ~~Формат: задел под расширяемое сжатие + секция info~~ — ВЫПОЛНЕНО.
2. ~~Ревью кода перед 1.0.0 + разбор находок~~ — ВЫПОЛНЕНО, см. ниже.
   ~~Суффиксы больше, чем вмещает uint16~~ — ВЫПОЛНЕНО (шардирование);
   ~~искажённые теги OpenCorpora-словоформ~~ — ВЫПОЛНЕНО, включая баг
   схлопывания парадигм в дедупликации (`paradigmKeyHash`), найденный
   финальным ревью всей ветки. Обе критические находки код-ревью
   закрыты.
3. Фикс корня-в-суффиксе для сравнительной степени (`Cmp2`/«по-») +
   руно-безопасный `lcp()` — см.
   [docs/research/0003-comparative-paradigms-not-merging.md](research/0003-comparative-paradigms-not-merging.md),
   [docs/superpowers/specs/2026-09-15-comparative-prefix-split-design.md](superpowers/specs/2026-09-15-comparative-prefix-split-design.md),
   [docs/superpowers/plans/2026-09-15-comparative-prefix-split.md](superpowers/plans/2026-09-15-comparative-prefix-split.md).
   Не блокировало 1.0.0 по своей природе (data-hygiene + рычаг сжатия,
   не корректность lookup), но решено сделать до релиза, пока формат
   `.dat` ещё не выпущен и версионирование не требуется.
4. Этап 18, но сначала — груминг CLI-команд (есть отдельные идеи,
   уточняются с пользователем до начала этапа).
5. Релиз 1.0.0.
```

- [ ] **Step 2: Commit**

```bash
git add docs/todo.md
git commit -m "docs: add comparative-prefix-split fix to the pre-1.0.0 path"
```
