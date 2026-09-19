// Package unimorph imports a UniMorph TSV dictionary
// (lemma<TAB>wordform<TAB>bundle) into gomorphy's internal format:
// extracts paradigms from lemma/form/bundle rows, deduplicates them,
// builds a DAWG, and returns a *internal.Dictionary. Simpler than the
// OpenCorpora importer (no XML, no cross-lemma links) but follows the
// same paradigm/suffix/DAWG construction shape. See
// docs/en/research/0006-unimorph-import-plan.md (design) and
// docs/en/implementation/stage-16-import-unimorph.md.
package unimorph

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// Progress callback — called periodically during import. a: processed
// items, b: total items (0 if unknown).
type Progress func(a, b int)

// Options configures ImportFromTSV/CompileFromTSV.
type Options struct {
	// Language is gomorphy's own language code (e.g. "ru", not
	// UniMorph's own ISO 639-3 repository naming — see pkg/unimorph).
	// Mandatory: only "ru" is accepted today (see docs/en/research/
	// 0006-unimorph-import-plan.md, Q1); the code is structured with
	// room to grow, not a promise that other languages work yet.
	Language string

	// CharPolicy overrides the language-inferred default (nil means
	// infer from Language: "ru" -> internal.RussianCharPolicy()).
	CharPolicy *internal.CharPolicy

	// OnMalformed is called for each row that isn't exactly 3
	// tab-separated fields, with its 1-based line number and raw text.
	// The row is skipped either way; nil means skip silently.
	OnMalformed func(lineNumber int, text string)

	// Progress reports import progress; nil means no reporting.
	Progress Progress

	// SourceVersion, if non-empty, is recorded in the resulting
	// Dictionary's BuildInfo.SourceVersion (see docs/en/research/
	// 0006-unimorph-import-plan.md, Q7). UniMorph's TSV carries no
	// version metadata of its own — this always comes from the caller
	// (e.g. pkg/unimorph.Loader recording a download date/URL).
	SourceVersion string
}

// formBundle is a single row's wordform and its UniMorph feature bundle
// (e.g. "N;ACC;SG"), stored verbatim — see Options.Language's doc
// comment on why tags aren't reordered or mapped onto another schema.
type formBundle struct {
	text   string
	bundle string
}

// lemmaEntry accumulates all rows for a single lemma, plus a dedup set
// so a repeated (form, bundle) row within the same lemma is collapsed
// (see docs/en/research/0006-unimorph-import-plan.md, "Dedup and
// syncretism").
type lemmaEntry struct {
	text  string
	forms []formBundle
	seen  map[string]bool // "text\x00bundle" -> already appended
}

func (e *lemmaEntry) addRow(text, bundle string) {
	key := text + "\x00" + bundle
	if e.seen[key] {
		return
	}
	if e.seen == nil {
		e.seen = make(map[string]bool)
	}
	e.seen[key] = true
	e.forms = append(e.forms, formBundle{text: text, bundle: bundle})
}

// dawgEntry maps a DAWG key to its (paradigmID << 16) | formIdx value.
type dawgEntry struct {
	key string
	val uint32
}

// paradigmKeyHash builds a dedup key from two parallel per-form arrays
// (suffix IDs, tag IDs — UniMorph paradigms have no prefix, always id 0).
// A copy of pkg/morphology/importers/opencorpora's identical helper (see
// shard.go's doc comment on why this isn't shared yet).
func paradigmKeyHash(sk, tk []uint16) string {
	if len(sk) == 0 && len(tk) == 0 {
		return ""
	}
	buf := make([]byte, 0, len(sk)*2+len(tk)*2)
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

// lcp computes the longest common prefix of texts, trimmed back to the
// nearest valid UTF-8 rune boundary. A copy of
// pkg/morphology/importers/opencorpora's identical helper.
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

// suffixShardLimit — the capacity of the uint16 id space for one shard's
// suffixes.
const suffixShardLimit = 1 << 16

// shardBuild accumulates the state (suffixes, paradigms, DAWG keys) of
// one shard during ImportFromTSV.
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

// buildForms assembles a lemma's final form list with the lemma always
// at form 0 (see docs/en/research/0006-unimorph-import-plan.md, Q6): the
// first row whose text equals the lemma becomes form 0 (any further
// such rows are ordinary lemma-form homonyms, left in place); if no row
// has text == lemma, form 0 is synthesized with the lemma's own text and
// an empty tag. Either way, form 0's text always equals the lemma text,
// so lcp() over the returned forms' texts always includes it.
func buildForms(e *lemmaEntry) []formBundle {
	for i, f := range e.forms {
		if f.text != e.text {
			continue
		}
		if i == 0 {
			return e.forms
		}
		out := make([]formBundle, 0, len(e.forms))
		out = append(out, f)
		out = append(out, e.forms[:i]...)
		out = append(out, e.forms[i+1:]...)
		return out
	}
	out := make([]formBundle, 0, len(e.forms)+1)
	out = append(out, formBundle{text: e.text, bundle: ""})
	out = append(out, e.forms...)
	return out
}

// ImportFromTSV reads a UniMorph TSV (lemma<TAB>wordform<TAB>bundle,
// one row per line) from r and returns a *internal.Dictionary with
// TagSet, Suffixes, Prefixes (always [""]), Paradigms, and Words (DAWG)
// filled in.
//
// Pipeline:
//
//  1. Rows are read with a streaming scanner and accumulated by lemma
//     text (row order within the file isn't guaranteed to group a
//     lemma's rows together).
//  2. For each lemma, form 0 is fixed as the lemma itself (buildForms),
//     then the LCP-stem of all its forms (including form 0) is computed.
//  3. Each form -> (suffix_id, tag_id) goes into the current shard's
//     paradigm, sharded like the OpenCorpora importer (FillOnDemand,
//     shard.go).
//  4. Paradigms are deduplicated via map[paradigmKey] -> paradigmID,
//     separately per shard.
//  5. A DAWG is built from the keys (= the wordform text,
//     value=paraID<<16|formIdx), separately per shard.
//
// tagSet, if nil, is a fresh internal.NewTagSet("unimorph").
func ImportFromTSV(r io.Reader, tagSet *internal.TagSet, opts Options) (*internal.Dictionary, error) {
	if opts.Language != "ru" {
		return nil, fmt.Errorf("unimorph: unsupported language %q (only \"ru\" is supported)", opts.Language)
	}
	charPolicy := opts.CharPolicy
	if charPolicy == nil {
		charPolicy = internal.RussianCharPolicy()
	}
	if tagSet == nil {
		tagSet = internal.NewTagSet("unimorph")
	}

	// Phase 1: read rows, accumulate by lemma text. order preserves each
	// lemma's first-appearance line order, so shard assignment (and
	// therefore the resulting .dat) is deterministic across runs of the
	// same input, not dependent on Go's randomized map iteration order.
	var order []string
	byLemma := make(map[string]*lemmaEntry)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20) // 1MB max line: compound/hyphenated tokens
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			if opts.OnMalformed != nil {
				opts.OnMalformed(lineNum, line)
			}
			continue
		}
		lemma, form, bundle := fields[0], fields[1], fields[2]
		if lemma == "" {
			lemma = form
		}
		e, ok := byLemma[lemma]
		if !ok {
			e = &lemmaEntry{text: lemma}
			byLemma[lemma] = e
			order = append(order, lemma)
		}
		e.addRow(form, bundle)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("unimorph: scan: %w", err)
	}

	// Phase 2: build suffix text -> ID map and extract paradigms, one
	// shard at a time.
	strategy := FillOnDemand{}
	cur := newShardBuild()
	shards := []*shardBuild{cur}

	// Prefixes are always empty for UniMorph (no prefix-splitting rule
	// like OpenCorpora's Cmp2) but shared-across-shards to match the
	// internal.Dictionary contract other importers use.
	prefixList := []string{""}

	for _, lemmaText := range order {
		forms := buildForms(byLemma[lemmaText])

		stemInput := make([]string, len(forms))
		for i, f := range forms {
			stemInput[i] = f.text
		}
		stem := lcp(stemInput)

		newInLemma := make(map[string]bool)
		for _, text := range stemInput {
			suffix := ""
			if len(stem) < len(text) {
				suffix = text[len(stem):]
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

		for _, f := range forms {
			suffix := ""
			if len(stem) < len(f.text) {
				suffix = f.text[len(stem):]
			}

			sid, ok := cur.suffixTexts[suffix]
			if !ok {
				if len(cur.suffixList) >= suffixShardLimit {
					return nil, fmt.Errorf("unimorph: shard %d exceeded %d unique suffixes despite sharding strategy", len(shards)-1, suffixShardLimit)
				}
				sid = uint16(len(cur.suffixList))
				cur.suffixTexts[suffix] = sid
				cur.suffixList = append(cur.suffixList, suffix)
			}
			suffixIDs = append(suffixIDs, sid)

			tid, err := tagSet.Add(f.bundle)
			if err != nil {
				return nil, fmt.Errorf("unimorph: %w", err)
			}
			tagIDs = append(tagIDs, tid)
		}

		prefixIDs := make([]uint16, len(forms)) // always 0 ("")

		hash := paradigmKeyHash(suffixIDs, tagIDs)
		paraID, ok := cur.paradigmsDedup[hash]
		if !ok {
			if len(cur.paradigms) >= 1<<16 {
				return nil, fmt.Errorf("unimorph: shard %d exceeded 65536 unique paradigms", len(shards)-1)
			}
			paraID = uint16(len(cur.paradigms))
			cur.paradigmsDedup[hash] = paraID

			para := internal.NewParadigm(suffixIDs, tagIDs, prefixIDs)
			cur.paradigms = append(cur.paradigms, para)
		}

		for formIdx, f := range forms {
			val := uint32(paraID)<<16 | uint32(formIdx)
			cur.dawgEntries = append(cur.dawgEntries, dawgEntry{key: f.text, val: val})
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
			if opts.Progress != nil {
				opts.Progress(base+processed, totalEntries)
			}
		}
		dawg, err := internal.BuildDAWGWithValuesProgress(dawgKeys, dawgValues, shardProgress)
		if err != nil {
			return nil, fmt.Errorf("unimorph: build DAWG (shard %d): %w", i, err)
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
		charPolicy,
	)
	dict.Info = &internal.BuildInfo{Source: "unimorph"}
	if opts.SourceVersion != "" {
		dict.Info.SourceVersion = opts.SourceVersion
	}

	return dict, nil
}

// dedupEntries removes exact (key, val) duplicates from dawg entries. A
// copy of pkg/morphology/importers/opencorpora's identical helper — see
// its doc comment for why entries sharing a key but differing in val
// (homonyms) must NOT be deduped.
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

// CompileFromTSV wraps ImportFromTSV with a fresh TagSet.
func CompileFromTSV(r io.Reader, opts Options) (*internal.Dictionary, error) {
	tagSet := internal.NewTagSet("unimorph")
	return ImportFromTSV(r, tagSet, opts)
}

// CompileFromTSVFile opens path and calls CompileFromTSV.
func CompileFromTSVFile(path string, opts Options) (*internal.Dictionary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unimorph: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return CompileFromTSV(f, opts)
}
