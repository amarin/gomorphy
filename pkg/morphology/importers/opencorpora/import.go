// Package opencorpora загружает словарь OpenCorpora из dict.xml:
// извлекает парадигмы из лемм и словоформ, дедуплицирует,
// строит DAWG и возвращает *internal.Dictionary.
package opencorpora

import (
	"fmt"
	"io"
	"os"
	"strings"

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

func paradigmKeyHash(sk []uint16, tk []uint16) string {
	if len(sk) == 0 && len(tk) == 0 {
		return ""
	}
	var buf [1024]byte
	n := 0
	n = copy(buf[n:], encodeU16s(sk))
	n = copy(buf[n:], encodeU16s(tk))
	return string(buf[:n])
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

// ImportFromXML читает dict.xml из r и возвращает *internal.Dictionary с
// заполненными TagSet, Suffixes, Prefixes, Paradigms и Words (DAWG).
//
// Pipeline:
//
//	1. xmlscan собирает все леммы с словоформами.
//	2. Для каждой леммы вычисляется LCP-stem всех словоформ.
//	3. Каждая словоформа → (suffix_id, tag_id) в парадигму.
//	4. Парадигмы дедуплицируются через map[paradigmKey] → paradigmID.
//	5. Строится DAWG из ключей (stem + suffix, value=paraID<<16|formIdx).
//
// progress — необязательный callback для вывода прогресса.
// Вызывается каждые 10 секунд: (processed_keys, total_keys) при сборке DAWG.
func ImportFromXML(r io.Reader, tagSet *internal.TagSet, progress Progress) (*internal.Dictionary, error) {
	if tagSet == nil {
		tagSet = internal.NewTagSet("opencorpora")
	}

	// Phase 1: scan XML and collect lemmas + forms.
	var lemmas []lemmaEntry

	handler := &xmlHandler{
		tagSet:  tagSet,
		lemmas:  &lemmas,
		curForm: nil,
		err:     nil,
	}

	if err := xmlscan.New(r, handler).Scan(); err != nil {
		return nil, fmt.Errorf("opencorpora: scan: %w", err)
	}
	if handler.err != nil {
		return nil, fmt.Errorf("opencorpora: handler: %w", handler.err)
	}

	// Phase 2: build suffix text → ID map and extract paradigms.
	var suffixList []string
	suffixTexts := make(map[string]uint16)

	// paradigmsDedup: paradigmKey hash → paradigmID
	paradigmsDedup := make(map[string]uint16)
	var paradigms []internal.Paradigm

	// DAWG entries: (stem + suffix) → (paraID << 16) | formIdx
	var dawgEntries []dawgEntry

	for _, lem := range lemmas {
		if len(lem.forms) == 0 {
			continue
		}

		stem := lcp(lem.forms)

		var suffixIDs []uint16
		var tagIDs []uint16

		for _, frm := range lem.forms {
			suffix := ""
			if len(stem) < len(frm.text) {
				suffix = frm.text[len(stem):]
			}

			sid, ok := suffixTexts[suffix]
			if !ok {
				sid = uint16(len(suffixList))
				suffixTexts[suffix] = sid
				suffixList = append(suffixList, suffix)
			}
			suffixIDs = append(suffixIDs, sid)

			tid := tagSet.Add(frm.gramm)
			tagIDs = append(tagIDs, tid)
		}

		hash := paradigmKeyHash(suffixIDs, tagIDs)
		paraID, ok := paradigmsDedup[hash]
		if !ok {
			paraID = uint16(len(paradigms))
			paradigmsDedup[hash] = paraID

			prefixes := make([]uint16, len(lem.forms))
			para := internal.NewParadigm(suffixIDs, tagIDs, prefixes)
			paradigms = append(paradigms, para)
		}

		for formIdx := 0; formIdx < len(lem.forms); formIdx++ {
			suffix := ""
			if len(stem) < len(lem.forms[formIdx].text) {
				suffix = lem.forms[formIdx].text[len(stem):]
			}
			dawgKey := stem + suffix
			val := uint32(paraID)<<16 | uint32(formIdx)
			dawgEntries = append(dawgEntries, dawgEntry{key: dawgKey, val: val})
		}
	}

	// Deduplicate DAWG entries.
	dawgEntries = dedupEntries(dawgEntries)

	dawgKeys := make([]string, len(dawgEntries))
	dawgValues := make([]uint32, len(dawgEntries))
	for i, e := range dawgEntries {
		dawgKeys[i] = e.key
		dawgValues[i] = e.val
	}

	// Phase 3: build DAWG with progress callback.
	// This is the longest phase (sort + insert 3M+ keys + compile).
	dawg, err := internal.BuildDAWGWithValuesProgress(dawgKeys, dawgValues, progress)
	if err != nil {
		return nil, fmt.Errorf("opencorpora: build DAWG: %w", err)
	}

	dict := internal.NewDictionary(
		"ru",
		tagSet,
		suffixList,
		nil, // Prefixes: OpenCorpora lemmas carry their own prefixes
		paradigms,
		dawg,
		internal.RussianCharPolicy(),
	)
	dict.Info = &internal.BuildInfo{Source: "opencorpora"}

	return dict, nil
}

// xmlHandler implements xmlscan.Handler to collect lemmas and forms.
type xmlHandler struct {
	tagSet    *internal.TagSet
	lemmas    *[]lemmaEntry
	curForm   *formGrams
	err       error
	curGrams  []string
	lGrams    []string // gramms from <l g="..."> — used as fallback for <f> without <g>
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

func (h *xmlHandler) OnLemmaEnd() error {
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

// lcp computes the longest common prefix of all form texts.
func lcp(forms []formGrams) string {
	if len(forms) == 0 {
		return ""
	}
	if len(forms) == 1 {
		return forms[0].text
	}
	prefix := forms[0].text
	for i := 1; i < len(forms); i++ {
		max := len(prefix)
		if len(forms[i].text) < max {
			max = len(forms[i].text)
		}
		j := 0
		for j < max && prefix[j] == forms[i].text[j] {
			j++
		}
		prefix = prefix[:j]
		if prefix == "" {
			return ""
		}
	}
	return prefix
}

// dedupEntries removes duplicate keys from dawg entries.
func dedupEntries(entries []dawgEntry) []dawgEntry {
	seen := make(map[string]bool)
	result := make([]dawgEntry, 0, len(entries))
	for _, e := range entries {
		if !seen[e.key] {
			seen[e.key] = true
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
	defer f.Close()
	return CompileFromXML(f, progress)
}
