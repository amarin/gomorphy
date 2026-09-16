// Package opencorpora загружает словарь OpenCorpora из dict.xml:
// извлекает парадигмы из лемм и словоформ, дедуплицирует,
// строит DAWG и возвращает *internal.Dictionary.
package opencorpora

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/amarin/gomorphy/internal/xmlscan"
	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// Progress callback — вызывается periodically (каждые 10s) во время импорта.
// a: processed items, b: total items (0 если unknown).
// Для вывода в stderr: func(a, b int) { fmt.Fprintf(os.Stderr, "...") }
type Progress func(a, b int)

// lemmaEntry accumulates all forms for a single lemma.
type lemmaEntry struct {
	id    uint32
	text  string
	forms []formGrams
}

// formGrams is a single form with its combined grammatical characteristic.
type formGrams struct {
	text  string
	gramm string // combined grammeme string, e.g. "NOUN,anim,masc,sing,nomn"
}

// dawgEntry maps a DAWG key to its (paradigmID << 16) | formIdx value.
type dawgEntry struct {
	key string
	val uint32
}

// paradigmKeyHash builds a dedup key from three parallel per-form arrays
// (prefix IDs, suffix IDs, tag IDs). The concatenation has no separators
// or length prefixes between segments — this is collision-free only
// because the caller always passes three slices of equal length
// (len(lem.forms) each), so total byte length alone determines where
// each segment starts. Do not call this with unequal-length slices.
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

func encodeU16s(u []uint16) []byte {
	if len(u) == 0 {
		return nil
	}
	b := make([]byte, len(u)*2)
	for i, v := range u {
		b[i*2] = byte(v >> 8)
		b[i*2+1] = byte(v)
	}
	return b
}

// suffixShardLimit — вместимость uint16 id-пространства суффиксов одного
// шарда.
const suffixShardLimit = 1 << 16

// shardBuild накапливает состояние (суффиксы, парадигмы, ключи DAWG)
// одного шарда во время ImportFromXML.
type shardBuild struct {
	suffixList     []string
	suffixTexts    map[string]uint16
	paradigmsDedup map[string]uint16 // paradigmKeyHash -> paradigmID
	paradigms      []internal.Paradigm
	dawgEntries    []dawgEntry
}

func newShardBuild() *shardBuild {
	return &shardBuild{
		suffixTexts:    make(map[string]uint16),
		paradigmsDedup: make(map[string]uint16),
	}
}

// ImportFromXML читает dict.xml из r и возвращает *internal.Dictionary с
// заполненными TagSet, Suffixes, Prefixes, Paradigms и Words (DAWG).
//
// Pipeline:
//
//  1. xmlscan собирает все леммы с словоформами.
//  2. Для каждой леммы вычисляется LCP-stem всех словоформ.
//  3. Каждая словоформа → (suffix_id, tag_id) в парадигму текущего шарда.
//     Suffix id адресуется uint16, поэтому леммы делятся на несколько
//     шардов с независимыми id-пространствами, если суффиксов больше,
//     чем помещается в один uint16-диапазон (см. FillOnDemand в shard.go
//     и docs/superpowers/specs/2026-09-14-suffix-sharding-design.md).
//  4. Парадигмы дедуплицируются через map[paradigmKey] → paradigmID,
//     отдельно на каждый шард.
//  5. Строится DAWG из ключей (stem + suffix, value=paraID<<16|formIdx),
//     отдельно на каждый шард.
//
// progress — необязательный callback для вывода прогресса. Вызывается
// периодически с (processed, total), где total — суммарное число ключей
// DAWG по всем шардам и processed растёт непрерывно через границы шардов
// (CLI видит один общий прогресс-бар, а не рестарт на каждом шарде).
func ImportFromXML(r io.Reader, tagSet *internal.TagSet, progress Progress) (*internal.Dictionary, error) {
	if tagSet == nil {
		tagSet = internal.NewTagSet("opencorpora")
	}

	// Phase 1: scan XML and collect lemmas + forms.
	var lemmas []lemmaEntry

	handler := &xmlHandler{
		tagSet: tagSet,
		lemmas: &lemmas,
		err:    nil,
	}

	if err := xmlscan.New(r, handler).Scan(); err != nil {
		return nil, fmt.Errorf("opencorpora: scan: %w", err)
	}
	if handler.err != nil {
		return nil, fmt.Errorf("opencorpora: handler: %w", handler.err)
	}

	// Phase 2: build suffix text -> ID map and extract paradigms, one
	// shard at a time.
	strategy := FillOnDemand{}
	cur := newShardBuild()
	shards := []*shardBuild{cur}

	// Prefixes are shared across ALL shards (unlike suffixes/paradigms,
	// which are per-shard) — see the doc comment on paradigmAffix in
	// pkg/morphology/parse.go. Index 0 is always the empty prefix.
	prefixTexts := map[string]uint16{"": 0}
	prefixList := []string{""}

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
				if len(prefixList) >= 1<<16 {
					return nil, fmt.Errorf("opencorpora: exceeded %d unique prefixes", 1<<16)
				}
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
			if dawgKey != lem.forms[formIdx].text {
				return nil, fmt.Errorf("opencorpora: internal invariant violated: prefix+stem+suffix (%q) != form text (%q) for lemma %q", dawgKey, lem.forms[formIdx].text, lem.text)
			}
			val := uint32(paraID)<<16 | uint32(formIdx)
			cur.dawgEntries = append(cur.dawgEntries, dawgEntry{key: dawgKey, val: val})
		}
	}

	// Phase 3: dedup DAWG entries per shard, then build one DAWG per
	// shard, reporting progress cumulatively across all shards.
	totalEntries := 0
	for _, s := range shards {
		s.dawgEntries = dedupEntries(s.dawgEntries)
		totalEntries += len(s.dawgEntries)
	}

	suffixesPerShard := make([][]string, len(shards))
	paradigmsPerShard := make([][]internal.Paradigm, len(shards))
	wordsPerShard := make([]*internal.DAWG, len(shards))

	processedBefore := 0
	for i, s := range shards {
		dawgKeys := make([]string, len(s.dawgEntries))
		dawgValues := make([]uint32, len(s.dawgEntries))
		for j, e := range s.dawgEntries {
			dawgKeys[j] = e.key
			dawgValues[j] = e.val
		}

		base := processedBefore
		shardProgress := func(processed, _ int) {
			if progress != nil {
				progress(base+processed, totalEntries)
			}
		}
		dawg, err := internal.BuildDAWGWithValuesProgress(dawgKeys, dawgValues, shardProgress)
		if err != nil {
			return nil, fmt.Errorf("opencorpora: build DAWG (shard %d): %w", i, err)
		}

		suffixesPerShard[i] = s.suffixList
		paradigmsPerShard[i] = s.paradigms
		wordsPerShard[i] = dawg
		processedBefore += len(s.dawgEntries)
	}

	dict := internal.NewDictionary(
		"ru",
		tagSet,
		suffixesPerShard,
		prefixList,
		paradigmsPerShard,
		wordsPerShard,
		internal.RussianCharPolicy(),
	)
	dict.Info = &internal.BuildInfo{Source: "opencorpora"}
	if handler.sourceVersion != "" {
		dict.Info.SourceVersion = fmt.Sprintf("%s/%s", handler.sourceVersion, handler.sourceRevision)
	}

	return dict, nil
}

// xmlHandler implements xmlscan.Handler to collect lemmas and forms.
//
// Grammemes for a lemma's headword (<l>) and for each of its forms (<f>)
// are collected into separate buffers (lemGrams, formOwnGrams) because
// they arrive interleaved across many XML events; a form's own <g>
// children are only fully known once </f> closes, so its final tag is
// assembled on OnFormEnd, not on OnForm. See
// docs/superpowers/specs/2026-09-15-opencorpora-tag-fix-design.md.
//
// The field is named formOwnGrams, not formGrams, to avoid colliding in
// spirit with the package's existing formGrams *type* (used below and in
// lem.forms).
type xmlHandler struct {
	tagSet *internal.TagSet
	lemmas *[]lemmaEntry
	err    error

	lemGrams     []string // current lemma's own grammemes (from <l>); persist across all its forms
	formOwnGrams []string // current form's own grammemes (from <f>); reset on every OnForm
	curFormText  string   // current form's text, held between OnForm and OnFormEnd
	inLemmaHead  bool     // true between OnLemma and OnLemmaHeadEnd (i.e. while inside <l>)

	sourceVersion  string // root <dictionary version="..."> attribute
	sourceRevision string // root <dictionary revision="..."> attribute
}

// OnDictionaryRoot copies the root <dictionary> tag's version/revision
// attributes for BuildInfo.SourceVersion (set by ImportFromXML after Scan
// returns) — s.attr's byte slices are scanner-owned and only valid until
// the next handler call, so they must be copied to a string here.
func (h *xmlHandler) OnDictionaryRoot(version, revision []byte) error {
	h.sourceVersion = string(version)
	h.sourceRevision = string(revision)

	return nil
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
// If any Cmp2-tagged form's text does not literally start with "по",
// every form of this lemma falls back to prefix "" and its own full
// text, exactly the pre-fix behavior, and ok is false so the caller
// can detect it. This does happen on the real dict.xml: 3 lemmas
// (недобитее, окологлоточнее, мультипроцессорнее) hit this path,
// because their base word already carries its own prefix
// (недо-/около-/мульти-) and OpenCorpora infixes "по" after it rather
// than prepending it to the whole word (e.g. "недобитее" -> Cmp2 form
// "недопобитее", not "понедобитее") — see
// docs/research/0003-comparative-paradigms-not-merging.md for the full
// trace. The fallback handles these 3 lemmas correctly (no merge
// benefit, no data corruption) — it is not dead code, keep it.
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

// dedupEntries removes exact (key, val) duplicates from dawg entries.
// Entries that share a key but differ in val are NOT duplicates - they are
// distinct homonym readings of the same wordform text (e.g. "кот" as a
// noun vs. "кот" as a verb), and BuildDAWGWithValues embeds val into the
// actual DAWG key (via PayloadSeparator, see dawgbuild.go) precisely so
// such entries coexist; SimilarItems already returns all of them via
// Item.Values. Deduping on key alone would silently drop every homonym
// reading but the first.
func dedupEntries(entries []dawgEntry) []dawgEntry {
	type entryKey struct {
		key string
		val uint32
	}
	seen := make(map[entryKey]bool, len(entries))
	result := make([]dawgEntry, 0, len(entries))
	for _, e := range entries {
		k := entryKey{e.key, e.val}
		if !seen[k] {
			seen[k] = true
			result = append(result, e)
		}
	}
	return result
}

// CompileFromXML wraps ImportFromXML with a fresh TagSet.
func CompileFromXML(r io.Reader, progress Progress) (*internal.Dictionary, error) {
	tagSet := internal.NewTagSet("opencorpora")
	return ImportFromXML(r, tagSet, progress)
}

// CompileFromXMLFile opens path and calls CompileFromXML.
func CompileFromXMLFile(path string, progress Progress) (*internal.Dictionary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opencorpora: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return CompileFromXML(f, progress)
}
