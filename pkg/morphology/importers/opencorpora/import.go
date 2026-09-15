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

	// Phase 2: build suffix text -> ID map and extract paradigms, one
	// shard at a time.
	strategy := FillOnDemand{}
	cur := newShardBuild()
	shards := []*shardBuild{cur}

	for _, lem := range lemmas {
		if len(lem.forms) == 0 {
			continue
		}

		stem := lcp(lem.forms)

		// How many suffixes would this lemma add to the CURRENT shard if
		// placed there? Dedup within the lemma's own forms too, so a
		// lemma reusing one suffix across several forms counts once.
		newInLemma := make(map[string]bool)
		for _, frm := range lem.forms {
			suffix := ""
			if len(stem) < len(frm.text) {
				suffix = frm.text[len(stem):]
			}
			if _, ok := cur.suffixTexts[suffix]; !ok {
				newInLemma[suffix] = true
			}
		}

		if strategy.Boundary(len(cur.suffixTexts), len(newInLemma), suffixShardLimit) {
			cur = newShardBuild()
			shards = append(shards, cur)
		}

		var suffixIDs []uint16
		var tagIDs []uint16

		for _, frm := range lem.forms {
			suffix := ""
			if len(stem) < len(frm.text) {
				suffix = frm.text[len(stem):]
			}

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

		hash := paradigmKeyHash(suffixIDs, tagIDs)
		paraID, ok := cur.paradigmsDedup[hash]
		if !ok {
			if len(cur.paradigms) >= 1<<16 {
				return nil, fmt.Errorf("opencorpora: shard %d exceeded 65536 unique paradigms", len(shards)-1)
			}
			paraID = uint16(len(cur.paradigms))
			cur.paradigmsDedup[hash] = paraID

			prefixes := make([]uint16, len(lem.forms))
			para := internal.NewParadigm(suffixIDs, tagIDs, prefixes)
			cur.paradigms = append(cur.paradigms, para)
		}

		for formIdx := 0; formIdx < len(lem.forms); formIdx++ {
			suffix := ""
			if len(stem) < len(lem.forms[formIdx].text) {
				suffix = lem.forms[formIdx].text[len(stem):]
			}
			dawgKey := stem + suffix
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
		nil, // Prefixes: OpenCorpora lemmas carry their own prefixes
		paradigmsPerShard,
		wordsPerShard,
		internal.RussianCharPolicy(),
	)
	dict.Info = &internal.BuildInfo{Source: "opencorpora"}

	return dict, nil
}

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
	defer func() { _ = f.Close() }()
	return CompileFromXML(f, progress)
}
