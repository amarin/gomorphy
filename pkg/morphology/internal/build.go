package internal

import (
	"fmt"
	"unicode/utf8"
)

// BuildEntry is a single raw wordform triple fed to
// BuildDictionaryFromEntries. The lemma names the paradigm group the word
// form belongs to; a form whose Word equals the lemma is the paradigm's
// entry point.
type BuildEntry struct {
	Word  string
	Lemma string
	Tag   string
}

// BuildOptions configures BuildDictionaryFromEntries. The zero value
// produces a Russian dictionary (Language "ru", RussianCharPolicy,
// TagSet named "builder") with no progress reporting.
type BuildOptions struct {
	// Language is the dictionary's language code. Empty means "ru".
	Language string

	// CharPolicy overrides the language-inferred default (nil means
	// RussianCharPolicy()).
	CharPolicy *CharPolicy

	// TagSetName names the fresh TagSet the pipeline creates. Empty means
	// "builder".
	TagSetName string

	// Progress reports building progress, when non-nil, with cumulative
	// (processed, total) counts in DAWG-key units: processed is the
	// number of keys accounted for so far, aggregated across shard
	// boundaries, and total is the combined number of keys across all
	// shards (after per-shard dedup). The pair is informational only —
	// there is no total==0 end-of-run marker, and processed is not
	// guaranteed to reach total (each shard's DAWG-compile phase
	// rescales node counts onto the key axis, so the final reported
	// pair can be below (total, total), and intermediate values are not
	// monotonic). Treat the successful return of
	// BuildDictionaryFromEntries as the completion signal.
	Progress func(processed, total int)
}

// buildForm is one form within a lemma group.
type buildForm struct {
	text string
	tag  string
}

// lemmaGroup accumulates all forms of one lemma. Grouping is by lemma
// text in insertion order, so the output (shard assignment, paradigm and
// suffix ids, DAWG contents) is deterministic across runs of the same
// input rather than dependent on Go's randomized map iteration order.
type lemmaGroup struct {
	text  string
	forms []buildForm
	seen  map[buildFormKey]bool // "text\x00tag" -> already appended
}

type buildFormKey struct {
	text string
	tag  string
}

func (g *lemmaGroup) add(text, tag string) {
	key := buildFormKey{text: text, tag: tag}
	if g.seen[key] {
		return
	}
	if g.seen == nil {
		g.seen = make(map[buildFormKey]bool)
	}
	g.seen[key] = true
	g.forms = append(g.forms, buildForm{text: text, tag: tag})
}

// dawgEntry maps a DAWG key to its (paradigmID << 16) | formIdx value.
type dawgEntry struct {
	key string
	val uint32
}

// paradigmKeyHash builds a dedup key from two parallel per-form arrays
// (suffix IDs, tag IDs — the raw pipeline's paradigms have no prefix,
// always id 0). The concatenation has no separators or length prefixes
// between segments — this is collision-free only because the caller
// always passes two slices of equal length (len(forms) each), so total
// byte length alone determines where each segment starts. Do not call
// this with unequal-length slices.
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

// suffixShardLimit — the capacity of the uint16 id space for one shard's
// suffixes.
const suffixShardLimit = 1 << 16

// paradigmLimit — the capacity of the uint16 id space for one shard's
// paradigms. A variable (not a const) so tests can lower it and exercise
// the overflow path on a small corpus.
var paradigmLimit = 1 << 16

// ShardingStrategy decides, while lemma groups are processed in input
// order, when the current shard should close and a new one start. Each
// shard's suffix strings get their own independent id space (starts at 0,
// addressed as uint16), so no shard may hold more than limit unique
// suffixes. A copy of the importers' identical helper — see
// docs/en/superpowers/specs/2026-09-14-suffix-sharding-design.md.
type ShardingStrategy interface {
	// Boundary reports whether the CURRENT shard should close before a
	// lemma that would add newSuffixes new, not-yet-seen suffix strings
	// to it is placed — given the shard already holds currentCount
	// unique suffixes and no shard may exceed limit.
	Boundary(currentCount, newSuffixes, limit int) bool
}

// FillOnDemand packs lemmas into the current shard until adding the next
// lemma's new suffixes would exceed limit, then starts a new shard. A
// shard is never closed while still empty, so a single lemma needing
// more than limit suffixes on its own still gets a shard to itself
// rather than looping forever.
type FillOnDemand struct{}

func (FillOnDemand) Boundary(currentCount, newSuffixes, limit int) bool {
	return currentCount > 0 && currentCount+newSuffixes > limit
}

// shardBuild accumulates the state (suffixes, paradigms, DAWG keys) of
// one shard during BuildDictionaryFromEntries.
type shardBuild struct {
	suffixList     []string
	suffixTexts    map[string]uint16
	paradigmsDedup map[string]uint16 // paradigmKeyHash -> paradigmID
	paradigms      []Paradigm
	dawgEntries    []dawgEntry
}

func newShardBuild() *shardBuild {
	return &shardBuild{
		suffixTexts:    make(map[string]uint16),
		paradigmsDedup: make(map[string]uint16),
	}
}

// buildForms assembles a lemma group's final form list with the lemma
// always at form 0: the first form whose text equals the lemma becomes
// form 0 (any further such forms are ordinary lemma-form homonyms, left
// in place); if no form has text == lemma, form 0 is synthesized with
// the lemma's own text and an empty tag. Either way, form 0's text
// always equals the lemma text, so lcp() over the returned forms' texts
// always includes it — this is what makes every other form's normal form
// (readingForm's prefix₀+stem+suffix₀ reconstruction) come out as the
// lemma.
func buildForms(g *lemmaGroup) []buildForm {
	lemma := g.text
	for i, f := range g.forms {
		if f.text != lemma {
			continue
		}
		if i == 0 {
			return g.forms
		}
		out := make([]buildForm, 0, len(g.forms))
		out = append(out, f)
		out = append(out, g.forms[:i]...)
		out = append(out, g.forms[i+1:]...)
		return out
	}
	out := make([]buildForm, 0, len(g.forms)+1)
	out = append(out, buildForm{text: lemma, tag: ""})
	out = append(out, g.forms...)
	return out
}

// lcp computes the longest common prefix of texts, trimmed back to the
// nearest valid UTF-8 rune boundary so the result (and therefore every
// "suffix = text minus this prefix") is always valid UTF-8. A raw
// byte-level cut can otherwise land inside a multi-byte character. A copy
// of the importers' identical helper.
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

// BuildDictionaryFromEntries builds an immutable raw *Dictionary from a
// list of (word, lemma, tag) triples — the shared low-level pipeline
// underneath the public Builder API.
//
// Pipeline:
//
//  1. Entries are grouped by lemma text in insertion order (an empty
//     lemma falls back to the entry's own word).
//  2. For each lemma, form 0 is fixed as the lemma itself (buildForms),
//     then the LCP-stem of all its forms (including form 0) is computed.
//  3. Each form -> (suffix_id, tag_id) goes into the current shard's
//     paradigm, sharded like the importers (FillOnDemand,
//     suffixShardLimit, see
//     docs/en/superpowers/specs/2026-09-14-suffix-sharding-design.md).
//     Suffixes are interned per shard in a uint16 id space; tags via the
//     pipeline's own fresh TagSet (ErrTagSetFull propagates at the 65536
//     tag cap).
//  4. Paradigms are deduplicated via map[paradigmKey] -> paradigmID,
//     separately per shard.
//  5. A DAWG is built from the keys (= the wordform text,
//     value=paraID<<16|formIdx), separately per shard.
//
// The returned dictionary is raw: it has no Alphabet, Prediction, or
// Info — callers that need them (e.g. the public Builder) add them after
// this returns.
func BuildDictionaryFromEntries(opts BuildOptions, entries []BuildEntry) (*Dictionary, error) {
	language := opts.Language
	if language == "" {
		language = "ru"
	}
	charPolicy := opts.CharPolicy
	if charPolicy == nil {
		charPolicy = RussianCharPolicy()
	}
	tagSetName := opts.TagSetName
	if tagSetName == "" {
		tagSetName = "builder"
	}
	tagSet := NewTagSet(tagSetName)

	// Phase 1: group entries by lemma text, preserving first-appearance
	// order so the resulting .dat is deterministic across runs of the same
	// input.
	var order []string
	byLemma := make(map[string]*lemmaGroup)
	for _, e := range entries {
		lemma := e.Lemma
		if lemma == "" {
			lemma = e.Word
		}
		g, ok := byLemma[lemma]
		if !ok {
			g = &lemmaGroup{text: lemma}
			byLemma[lemma] = g
			order = append(order, lemma)
		}
		g.add(e.Word, e.Tag)
	}

	// Phase 2: build suffix text -> ID map and extract paradigms, one
	// shard at a time.
	strategy := FillOnDemand{}
	cur := newShardBuild()
	shards := []*shardBuild{cur}

	// Prefixes are always empty for the raw pipeline (no prefix-splitting
	// rule like OpenCorpora's Cmp2) but shared-across-shards to match the
	// internal.Dictionary contract other importers use.
	prefixList := []string{""}

	for _, lemma := range order {
		g := byLemma[lemma]
		forms := buildForms(g)

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
					return nil, fmt.Errorf("builder: shard %d exceeded %d unique suffixes despite sharding strategy", len(shards)-1, suffixShardLimit)
				}
				sid = uint16(len(cur.suffixList))
				cur.suffixTexts[suffix] = sid
				cur.suffixList = append(cur.suffixList, suffix)
			}
			suffixIDs = append(suffixIDs, sid)

			tid, err := tagSet.Add(f.tag)
			if err != nil {
				return nil, fmt.Errorf("builder: %w", err)
			}
			tagIDs = append(tagIDs, tid)
		}

		prefixIDs := make([]uint16, len(forms)) // always 0 ("")

		hash := paradigmKeyHash(suffixIDs, tagIDs)
		paraID, ok := cur.paradigmsDedup[hash]
		if !ok {
			if len(cur.paradigms) >= paradigmLimit {
				return nil, fmt.Errorf("builder: shard %d exceeded %d unique paradigms", len(shards)-1, paradigmLimit)
			}
			paraID = uint16(len(cur.paradigms))
			cur.paradigmsDedup[hash] = paraID

			para := NewParadigm(suffixIDs, tagIDs, prefixIDs)
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
	paradigmsPerShard := make([][]Paradigm, len(shards))
	wordsPerShard := make([]*DAWG, len(shards))

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
		dawg, err := BuildDAWGWithValuesProgress(dawgKeys, dawgValues, shardProgress)
		if err != nil {
			return nil, fmt.Errorf("builder: build DAWG (shard %d): %w", i, err)
		}

		suffixesPerShard[i] = s.suffixList
		paradigmsPerShard[i] = s.paradigms
		wordsPerShard[i] = dawg
		processedBefore += len(s.dawgEntries)
	}

	return NewDictionary(
		language,
		tagSet,
		suffixesPerShard,
		prefixList,
		paradigmsPerShard,
		wordsPerShard,
		charPolicy,
	), nil
}

// dedupEntries removes exact (key, val) duplicates from dawg entries. A
// copy of the importers' identical helper: entries that share a key but
// differ in val are distinct homonym readings (e.g. "кот" as a noun vs.
// "кот" as a verb) and must NOT be deduped.
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
