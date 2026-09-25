// Package opencorpora loads the OpenCorpora dictionary from dict.xml:
// extracts paradigms from lemmas and wordforms, deduplicates them,
// builds a DAWG, and returns a *internal.Dictionary.
package opencorpora

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/amarin/gomorphy/internal/xmlscan"
	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// Progress callback — called while the words DAWG is built (not while
// dict.xml is parsed), roughly every 1% of keys but at least every 1000.
// a: keys processed so far across all shards, b: total keys. nil disables.
// For printing to stderr: func(a, b int) { fmt.Fprintf(os.Stderr, "...") }
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

// xmlLink is one <link from="..." to="..." type="..."/> from dict.xml's
// root <links> section — see mergeLinkedLemmas.
type xmlLink struct {
	from uint32
	to   uint32
	typ  string
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

// excludedLinkTypes are OpenCorpora <link type="..."> ids that connect
// genuinely distinct words and must NOT merge lemmas: 7 NAME-PATR,
// 21 FULL-CONTRACTED, 23 CARDINAL-ORDINAL, 27 ADJF_TEXT-ADJF_NUMBER.
// Every other link type is followed — this matches pymorphy2's own dict
// compiler (pymorphy2/opencorpora_dict/compile.py, _join_lexemes,
// EXCLUDED_LINK_TYPES), verified byte-for-byte against the pymorphy2
// 0.9.1 tag that produced .data/pymorphy's reference dictionary. See
// mergeLinkedLemmas for why this exists.
var excludedLinkTypes = map[string]bool{"7": true, "21": true, "23": true, "27": true}

// mergeLinkedLemmas folds forms of linked lemmas into one another,
// in place, so a paradigm's form[0] (used by readingForm as the "normal
// form") resolves to the linguistically correct lemma instead of an
// arbitrary XML fragment's own headword.
//
// OpenCorpora's dict.xml splits a single lexeme's full paradigm across
// several <lemma> elements — e.g. a verb's infinitive ("ложиться") is
// one <lemma>, its finite/conjugated forms (headword "ложусь", the 1st
// person singular present) are a SEPARATE <lemma>, and its participle
// and gerund forms are separate <lemma> elements too. The only thing
// tying these back into one lexeme is the root <links> section (e.g.
// type="3" INFN-VERB, type="4" INFN-PRTF, type="5" INFN-GRND). Without
// this merge, every finite/participle/gerund form gets normalized to
// its own fragment's headword (e.g. "ложился" -> "ложусь") instead of
// the infinitive — see docs/en/research/0008-opencorpora-link-merge-design.md.
//
// This mirrors pymorphy2's _join_lexemes exactly: for each <link
// from="A" to="B" type="T">, unless T is excluded, B's forms move into
// A (following any earlier move of A itself, so transitive chains
// resolve to their ultimate root), and B is left empty — the existing
// `len(lem.forms) == 0` skip in ImportFromXML's Phase 2 loop then drops
// it, exactly like pymorphy2 keeping only lexemes still non-empty after
// the join.
func mergeLinkedLemmas(lemmas []lemmaEntry, links []xmlLink) {
	byID := make(map[uint32]int, len(lemmas))
	for i, lem := range lemmas {
		byID[lem.id] = i
	}

	// moves[originalToID] = resolved root lemma id it now lives under.
	moves := make(map[uint32]uint32, len(links))
	resolve := func(id uint32) uint32 {
		for {
			next, ok := moves[id]
			if !ok {
				return id
			}
			id = next
		}
	}

	for _, link := range links {
		if excludedLinkTypes[link.typ] {
			continue
		}

		toIdx, ok := byID[link.to]
		if !ok {
			continue
		}

		root := resolve(link.from)

		fromIdx, ok := byID[root]
		if !ok || fromIdx == toIdx {
			continue
		}

		lemmas[fromIdx].forms = append(lemmas[fromIdx].forms, lemmas[toIdx].forms...)
		lemmas[toIdx].forms = nil
		moves[link.to] = root
	}
}

// suffixShardLimit — the capacity of the uint16 id space for one shard's
// suffixes.
const suffixShardLimit = 1 << 16

// shardBuild accumulates the state (suffixes, paradigms, DAWG keys)
// of one shard during ImportFromXML.
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

// ImportFromXML reads dict.xml from r and returns a *internal.Dictionary
// with TagSet, Suffixes, Prefixes, Paradigms, and Words (DAWG) filled in.
//
// Pipeline:
//
//  1. xmlscan collects all lemmas with their wordforms.
//  2. For each lemma, the LCP-stem of all its wordforms is computed.
//  3. Each wordform → (suffix_id, tag_id) goes into the current shard's
//     paradigm. The suffix id is addressed as a uint16, so lemmas are
//     split across several shards with independent id spaces once there
//     are more suffixes than fit in one uint16 range (see FillOnDemand in
//     shard.go and docs/en/superpowers/specs/2026-09-14-suffix-sharding-design.md).
//  4. Paradigms are deduplicated via map[paradigmKey] → paradigmID,
//     separately per shard.
//  5. A DAWG is built from the keys (stem + suffix, value=paraID<<16|formIdx),
//     separately per shard.
//
// progress — an optional callback for reporting progress. Called
// periodically with (processed, total), where total is the combined
// number of DAWG keys across all shards and processed increases
// continuously across shard boundaries (so the CLI sees one overall
// progress bar instead of a restart on every shard).
func ImportFromXML(r io.Reader, tagSet *internal.TagSet, progress Progress) (*internal.Dictionary, error) {
	if tagSet == nil {
		tagSet = internal.NewTagSet("opencorpora")
	}

	// Phase 1: scan XML and collect lemmas + forms + cross-lemma links.
	var lemmas []lemmaEntry
	var links []xmlLink

	handler := &xmlHandler{
		tagSet: tagSet,
		lemmas: &lemmas,
		links:  &links,
		err:    nil,
	}

	if err := xmlscan.New(r, handler).Scan(); err != nil {
		return nil, fmt.Errorf("opencorpora: scan: %w", err)
	}
	if handler.err != nil {
		return nil, fmt.Errorf("opencorpora: handler: %w", handler.err)
	}

	// Phase 1.5: fold linked lemmas' forms together (see
	// mergeLinkedLemmas) so a lexeme split across several <lemma>
	// elements gets one correct normal form instead of one per fragment.
	mergeLinkedLemmas(lemmas, links)

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
// docs/en/superpowers/specs/2026-09-15-opencorpora-tag-fix-design.md.
//
// The field is named formOwnGrams, not formGrams, to avoid colliding in
// spirit with the package's existing formGrams *type* (used below and in
// lem.forms).
type xmlHandler struct {
	tagSet *internal.TagSet
	lemmas *[]lemmaEntry
	links  *[]xmlLink
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

// OnLink records one <link from="..." to="..." type="..."/> from the root
// <links> section, for mergeLinkedLemmas to fold afterwards. Malformed
// from/to attributes are treated as a scan error, same as OnLemma's id.
func (h *xmlHandler) OnLink(from, to, linkType []byte) error {
	fromID, err := strconv.ParseUint(string(from), 10, 32)
	if err != nil {
		return fmt.Errorf("opencorpora: link: bad from=%q: %w", from, err)
	}

	toID, err := strconv.ParseUint(string(to), 10, 32)
	if err != nil {
		return fmt.Errorf("opencorpora: link: bad to=%q: %w", to, err)
	}

	*h.links = append(*h.links, xmlLink{from: uint32(fromID), to: uint32(toID), typ: string(linkType)})

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
// docs/en/research/0003-comparative-paradigms-not-merging.md, section 4/6.
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
// docs/en/superpowers/specs/2026-09-15-comparative-prefix-split-design.md):
// the Cmp2 grammeme marks a comparative-degree form that is literally
// "по" + the corresponding non-Cmp2 form (e.g. lemma "поправимее",
// dict.xml id 259490: Cmp2 form "попоправимее" = "по" + "поправимее").
const cmp2Prefix = "по"

// stripCmp2Prefix separates each form's Cmp2-driven prefix ("по" or
// "") from the text that should feed lcp(), so the shared root never
// ends up split across different suffix strings depending on which
// forms happen to carry that prefix (the root-in-suffix problem from
// docs/en/research/0003-comparative-paradigms-not-merging.md).
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
// docs/en/research/0003-comparative-paradigms-not-merging.md for the full
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
		k := entryKey(e)
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
