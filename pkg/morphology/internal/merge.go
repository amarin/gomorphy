package internal

import (
	"encoding/binary"
	"fmt"
	"slices"
)

// MergeMode is MergeDictionaries' word-level conflict policy; values
// mirror the public morphology.MergeMode one for one.
type MergeMode int

const (
	// MergeAdd takes an overlay word only if neither the base nor an
	// earlier overlay has it.
	MergeAdd MergeMode = iota
	// MergeReplace lets an overlay word's readings replace whatever the
	// word had so far (base or earlier overlay) — the last overlay wins.
	MergeReplace
)

// overlayReading is one reading of an overlay wordform, in the overlay's
// own id space.
type overlayReading struct {
	shard      int
	para, form uint16
}

// winner is the overlay whose readings a word gets in the output.
type winner struct {
	overlay  int
	readings []overlayReading
}

// decideOverlays folds overlays over the base in order and returns, for
// every word an overlay supplies to the output, the winning overlay and
// its readings. Merge(b,[o1,o2]) == Merge(Merge(b,[o1]),[o2]).
func decideOverlays(base *Dictionary, overlays []*Dictionary, mode MergeMode) (map[string]*winner, error) {
	winners := make(map[string]*winner)
	for oi, o := range overlays {
		words, err := wordReadings(o)
		if err != nil {
			return nil, fmt.Errorf("overlay %d: %w", oi, err)
		}
		for word, rs := range words {
			if mode == MergeAdd {
				if _, taken := winners[word]; taken || hasWord(base, word) {
					continue
				}
			}
			winners[word] = &winner{overlay: oi, readings: rs}
		}
	}
	return winners, nil
}

// wordReadings enumerates every wordform of d (decoded to plain text)
// with its readings, across all shards.
func wordReadings(d *Dictionary) (map[string][]overlayReading, error) {
	out := make(map[string][]overlayReading)
	for s, w := range d.Words {
		if w == nil {
			continue
		}
		var walkErr error
		w.Walk(func(key string, vals [][]byte) {
			if walkErr != nil {
				return
			}
			word, err := decodeKey(d.Alphabet, key)
			if err != nil {
				walkErr = err
				return
			}
			for _, v := range vals {
				if len(v) < 4 {
					continue
				}
				out[word] = append(out[word], overlayReading{
					shard: s,
					para:  binary.BigEndian.Uint16(v[:2]),
					form:  binary.BigEndian.Uint16(v[2:4]),
				})
			}
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	return out, nil
}

// decodeKey turns a stored words.dawg key into plain text.
func decodeKey(a Alphabet, key string) (string, error) {
	if a == nil {
		return key, nil
	}
	word, err := a.Decode([]byte(key))
	if err != nil {
		return "", fmt.Errorf("decode DAWG key: %w", err)
	}
	return word, nil
}

// encodeKey turns plain text into a words.dawg key; ok is false when the
// alphabet can't encode it.
func encodeKey(a Alphabet, word string) (string, bool) {
	if a == nil {
		return word, true
	}
	b, err := a.Encode(word)
	if err != nil {
		return "", false
	}
	return string(b), true
}

// hasWord reports whether word is an exact key of any of d's shards (no
// CharPolicy substitution).
func hasWord(d *Dictionary, word string) bool {
	for _, w := range d.Words {
		if shardHasWord(w, d.Alphabet, word) {
			return true
		}
	}
	return false
}

// shardHasWord reports whether word is an exact key of one words DAWG.
func shardHasWord(w *DAWG, a Alphabet, word string) bool {
	if w == nil || word == "" {
		return false
	}
	key, ok := encodeKey(a, word)
	if !ok {
		return false
	}
	idx := w.Follow(key, 0)
	return idx != 0 && w.HasPayloadChild(idx)
}

// mergeSuffixLimit caps a merge target shard's suffix count before a new
// shard is opened. A var (not suffixShardLimit directly) so tests can
// lower it.
var mergeSuffixLimit = suffixShardLimit

// mergeShard is one output shard under construction: the base shard's
// suffixes/paradigms verbatim (ids stable) plus whatever the overlays
// append, and the overlay readings placed into it.
type mergeShard struct {
	suffixes  []string
	suffixIdx map[string]uint16 // lazily built text → id
	paradigms []Paradigm
	paraIdx   map[string]uint16 // lazily built paradigm data key → id
	base      *DAWG             // the base shard's words DAWG; nil for an appended shard
	placed    []WordValue       // overlay readings targeting this shard
	dirty     bool              // the words DAWG must be rebuilt
}

type paraRef struct {
	overlay, shard int
	para           uint16
}

type paraLoc struct {
	shard int
	para  uint16
}

// merger holds the output id spaces while overlay paradigms are remapped
// into them.
type merger struct {
	tagSet    *TagSet
	prefixes  []string
	prefixIdx map[string]uint16
	shards    []*mergeShard
	cache     map[paraRef]paraLoc
}

func newMerger(base *Dictionary) *merger {
	m := &merger{
		tagSet:    cloneTagSet(base.TagSet),
		prefixes:  slices.Clone(base.Prefixes),
		prefixIdx: make(map[string]uint16, len(base.Prefixes)),
		cache:     make(map[paraRef]paraLoc),
	}
	if len(m.prefixes) == 0 {
		m.prefixes = []string{""}
	}
	for i, p := range m.prefixes {
		if _, ok := m.prefixIdx[p]; !ok {
			m.prefixIdx[p] = uint16(i)
		}
	}
	for i, w := range base.Words {
		s := &mergeShard{base: w}
		if i < len(base.Suffixes) {
			s.suffixes = slices.Clone(base.Suffixes[i])
		}
		if i < len(base.Paradigms) {
			s.paradigms = slices.Clone(base.Paradigms[i])
		}
		m.shards = append(m.shards, s)
	}
	if len(m.shards) == 0 {
		m.shards = []*mergeShard{{}}
	}
	return m
}

func cloneTagSet(t *TagSet) *TagSet {
	if t == nil {
		return NewTagSet("")
	}
	out := NewTagSet(t.Name)
	out.Tags = slices.Clone(t.Tags)
	for i, name := range out.Tags {
		if _, ok := out.Index[name]; !ok {
			out.Index[name] = uint16(i)
		}
	}
	return out
}

func (s *mergeShard) suffixIndex() map[string]uint16 {
	if s.suffixIdx == nil {
		s.suffixIdx = make(map[string]uint16, len(s.suffixes))
		for i, text := range s.suffixes {
			if _, ok := s.suffixIdx[text]; !ok {
				s.suffixIdx[text] = uint16(i)
			}
		}
	}
	return s.suffixIdx
}

func (s *mergeShard) internSuffix(text string) (uint16, error) {
	idx := s.suffixIndex()
	if id, ok := idx[text]; ok {
		return id, nil
	}
	if len(s.suffixes) >= suffixShardLimit {
		return 0, fmt.Errorf("shard exceeded %d unique suffixes", suffixShardLimit)
	}
	id := uint16(len(s.suffixes))
	s.suffixes = append(s.suffixes, text)
	idx[text] = id
	return id, nil
}

func (s *mergeShard) paradigmIndex() map[string]uint16 {
	if s.paraIdx == nil {
		s.paraIdx = make(map[string]uint16, len(s.paradigms))
		for i, p := range s.paradigms {
			k := string(encodeU16s(p.Data()))
			if _, ok := s.paraIdx[k]; !ok {
				s.paraIdx[k] = uint16(i)
			}
		}
	}
	return s.paraIdx
}

func (m *merger) internPrefix(text string) (uint16, error) {
	if id, ok := m.prefixIdx[text]; ok {
		return id, nil
	}
	if len(m.prefixes) >= 1<<16 {
		return 0, fmt.Errorf("too many prefixes (max %d)", 1<<16)
	}
	id := uint16(len(m.prefixes))
	m.prefixes = append(m.prefixes, text)
	m.prefixIdx[text] = id
	return id, nil
}

// targetShard returns the shard a paradigm with the given suffix texts
// goes to: the last shard while it has room, otherwise a new empty one.
func (m *merger) targetShard(sufTexts []string) int {
	last := len(m.shards) - 1
	t := m.shards[last]
	idx := t.suffixIndex()
	fresh := make(map[string]bool)
	for _, s := range sufTexts {
		if _, ok := idx[s]; !ok {
			fresh[s] = true
		}
	}
	if len(t.suffixes)+len(fresh) <= mergeSuffixLimit && len(t.paradigms) < paradigmLimit {
		return last
	}
	m.shards = append(m.shards, &mergeShard{})
	return last + 1
}

// place remaps the overlay paradigm behind r into the output id spaces
// (form-for-form: suffix text, prefix text and tag name → output ids),
// deduplicating against the target shard's paradigms, and returns where
// it landed. The form index is preserved.
func (m *merger) place(oi int, o *Dictionary, r overlayReading) (paraLoc, error) {
	ref := paraRef{overlay: oi, shard: r.shard, para: r.para}
	if loc, ok := m.cache[ref]; ok {
		return loc, nil
	}
	if r.shard >= len(o.Paradigms) || int(r.para) >= len(o.Paradigms[r.shard]) {
		return paraLoc{}, fmt.Errorf("overlay %d: shard %d: paradigm %d out of range", oi, r.shard, r.para)
	}
	p := o.Paradigms[r.shard][r.para]
	var oSuffixes []string
	if r.shard < len(o.Suffixes) {
		oSuffixes = o.Suffixes[r.shard]
	}

	n := p.Len()
	sufTexts := make([]string, n)
	tagIDs := make([]uint16, n)
	prefIDs := make([]uint16, n)
	for i := 0; i < n; i++ {
		sufTexts[i] = stringAt(oSuffixes, p.Suffix(i))
		tag := ""
		if o.TagSet != nil {
			tag = o.TagSet.TagName(p.Tag(i))
		}
		tid, err := m.tagSet.Add(tag)
		if err != nil {
			return paraLoc{}, err
		}
		tagIDs[i] = tid
		pid, err := m.internPrefix(stringAt(o.Prefixes, p.Prefix(i)))
		if err != nil {
			return paraLoc{}, err
		}
		prefIDs[i] = pid
	}

	shard := m.targetShard(sufTexts)
	t := m.shards[shard]
	sufIDs := make([]uint16, n)
	for i, text := range sufTexts {
		id, err := t.internSuffix(text)
		if err != nil {
			return paraLoc{}, fmt.Errorf("shard %d: %w", shard, err)
		}
		sufIDs[i] = id
	}

	para := NewParadigm(sufIDs, tagIDs, prefIDs)
	key := string(encodeU16s(para.Data()))
	idx := t.paradigmIndex()
	id, ok := idx[key]
	if !ok {
		if len(t.paradigms) >= paradigmLimit {
			return paraLoc{}, fmt.Errorf("shard %d exceeded %d unique paradigms", shard, paradigmLimit)
		}
		id = uint16(len(t.paradigms))
		t.paradigms = append(t.paradigms, para)
		idx[key] = id
	}
	loc := paraLoc{shard: shard, para: id}
	m.cache[ref] = loc
	return loc, nil
}

// stringAt returns ar[i], or "" when i is out of range (mirrors the
// engine's strAt in pkg/morphology/parse.go).
func stringAt(ar []string, i uint16) string {
	if int(i) < len(ar) {
		return ar[i]
	}
	return ""
}
