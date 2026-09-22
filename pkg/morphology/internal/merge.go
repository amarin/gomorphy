package internal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
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
	if len(s.suffixes) >= mergeSuffixLimit {
		return 0, fmt.Errorf("shard exceeded %d unique suffixes", mergeSuffixLimit)
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

// MergeOptions configures MergeDictionaries.
type MergeOptions struct {
	Mode MergeMode
	// RebuildPrediction replaces the base's prediction DAWGs with a single
	// prefix-0 DAWG rebuilt from the merged shard 0. Requires a
	// single-shard output (ErrPredictionSharded otherwise) and Productive.
	RebuildPrediction bool
	// Productive filters prediction tags (the engine's productive()).
	Productive func(tag string) bool
}

// ErrPredictionSharded is returned when RebuildPrediction is requested
// but the merged dictionary has more than one shard: the engine resolves
// prediction against shard 0 only.
var ErrPredictionSharded = errors.New("prediction rebuild needs a single-shard output")

// MergeDictionaries merges overlays into base structurally: base ids
// (tags, prefixes, per-shard suffixes and paradigms) are kept verbatim
// and only grow, overlay readings are remapped into them, and only the
// words DAWGs of changed shards are rebuilt. The base's prediction and
// probability therefore stay valid. The result shares no memory with
// the inputs and has no Info.
func MergeDictionaries(base *Dictionary, overlays []*Dictionary, opts MergeOptions) (*Dictionary, error) {
	if base == nil {
		return nil, errors.New("merge: nil base")
	}
	winners, err := decideOverlays(base, overlays, opts.Mode)
	if err != nil {
		return nil, err
	}
	words := make([]string, 0, len(winners))
	for w := range winners {
		words = append(words, w)
	}
	sort.Strings(words)

	m := newMerger(base)
	for _, w := range words {
		win := winners[w]
		for _, r := range win.readings {
			loc, err := m.place(win.overlay, overlays[win.overlay], r)
			if err != nil {
				return nil, err
			}
			s := m.shards[loc.shard]
			s.placed = append(s.placed, WordValue{Word: w, Value: uint32(loc.para)<<16 | uint32(r.form)})
			s.dirty = true
		}
	}
	if opts.RebuildPrediction {
		if len(m.shards) != 1 {
			return nil, ErrPredictionSharded
		}
		if opts.Productive == nil {
			return nil, errors.New("merge: Productive is required to rebuild prediction")
		}
	}

	removed := make(map[string]bool)
	if opts.Mode == MergeReplace {
		for _, w := range words {
			for i, s := range m.shards {
				if i < len(base.Words) && shardHasWord(base.Words[i], base.Alphabet, w) {
					removed[w] = true
					s.dirty = true
				}
			}
		}
	}

	alphabet, changed, err := mergeAlphabet(base, words)
	if err != nil {
		return nil, err
	}

	out := &Dictionary{
		Language:   base.Language,
		TagSet:     m.tagSet,
		Prefixes:   m.prefixes,
		CharPolicy: base.CharPolicy,
		Alphabet:   alphabet,
	}
	var shard0 []WordValue
	for i, s := range m.shards {
		out.Suffixes = append(out.Suffixes, s.suffixes)
		out.Paradigms = append(out.Paradigms, s.paradigms)
		if !s.dirty && !changed && s.base != nil {
			out.Words = append(out.Words, s.base.Clone())
			continue
		}
		pairs, err := shardPairs(s.base, base.Alphabet, removed)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, s.placed...)
		if i == 0 {
			shard0 = pairs
		}
		w, err := buildWordsDAWG(pairs, alphabet)
		if err != nil {
			return nil, fmt.Errorf("merge: shard %d: %w", i, err)
		}
		out.Words = append(out.Words, w)
	}

	if opts.RebuildPrediction {
		if shard0 == nil { // shard 0 was reused verbatim
			if shard0, err = shardPairs(base.Words[0], base.Alphabet, nil); err != nil {
				return nil, err
			}
		}
		pred, err := BuildPredictionFrom(shard0, out.Paradigms[0], out.TagSet, opts.Productive)
		if err != nil {
			return nil, fmt.Errorf("merge: prediction: %w", err)
		}
		out.Prediction = []*DAWG{pred}
	} else {
		for _, p := range base.Prediction {
			out.Prediction = append(out.Prediction, p.Clone())
		}
	}

	if out.Probability, err = mergeProbability(base, overlays, winners, removed); err != nil {
		return nil, fmt.Errorf("merge: probability: %w", err)
	}
	return out, nil
}

// mergeAlphabet picks the output's dense alphabet: the base's when it
// can encode every overlay word (changed=false), otherwise a new one over
// the base runes (or all base words for a non-dense base) plus the
// overlay words — width 1, falling back to width 2.
func mergeAlphabet(base *Dictionary, newWords []string) (Alphabet, bool, error) {
	if da, ok := base.Alphabet.(*DenseAlphabet); ok {
		fits := true
		for _, w := range newWords {
			if _, err := da.Encode(w); err != nil {
				fits = false
				break
			}
		}
		if fits {
			return da, false, nil
		}
		a, err := denseAlphabetFor(append([]string{string(da.Runes())}, newWords...))
		return a, true, err
	}
	corpus := slices.Clone(newWords)
	for _, w := range base.Words {
		if w == nil {
			continue
		}
		var walkErr error
		w.Walk(func(key string, _ [][]byte) {
			word, err := decodeKey(base.Alphabet, key)
			if err != nil && walkErr == nil {
				walkErr = err
			}
			corpus = append(corpus, word)
		})
		if walkErr != nil {
			return nil, false, walkErr
		}
	}
	a, err := denseAlphabetFor(corpus)
	return a, true, err
}

func denseAlphabetFor(corpus []string) (*DenseAlphabet, error) {
	if a, err := NewDenseAlphabet(1, corpus); err == nil {
		return a, nil
	}
	return NewDenseAlphabet(2, corpus)
}

// shardPairs returns a base shard's readings as plain-text pairs,
// skipping removed words. nil w yields nil.
func shardPairs(w *DAWG, a Alphabet, removed map[string]bool) ([]WordValue, error) {
	if w == nil {
		return nil, nil
	}
	var pairs []WordValue
	var walkErr error
	w.Walk(func(key string, vals [][]byte) {
		if walkErr != nil {
			return
		}
		word, err := decodeKey(a, key)
		if err != nil {
			walkErr = err
			return
		}
		if removed[word] {
			return
		}
		for _, v := range vals {
			if len(v) >= 4 {
				pairs = append(pairs, WordValue{Word: word, Value: binary.BigEndian.Uint32(v[:4])})
			}
		}
	})
	return pairs, walkErr
}

// buildWordsDAWG encodes pairs under a and builds a words DAWG.
func buildWordsDAWG(pairs []WordValue, a Alphabet) (*DAWG, error) {
	keys := make([]string, len(pairs))
	vals := make([]uint32, len(pairs))
	for i, p := range pairs {
		k, ok := encodeKey(a, p.Word)
		if !ok {
			return nil, fmt.Errorf("encode %q", p.Word)
		}
		keys[i], vals[i] = k, p.Value
	}
	return BuildDAWGWithValues(keys, vals)
}

// mergeProbability carries the base's p(tag|word) DAWG (keys
// "word:tag"): verbatim when nothing was removed and no overlay supplies
// probability, otherwise filtered (replaced words dropped) and extended
// with the winning overlays' entries, then rebuilt.
func mergeProbability(base *Dictionary, overlays []*Dictionary, winners map[string]*winner, removed map[string]bool) (*DAWG, error) {
	added := make(map[string]uint32)
	for word, win := range winners {
		o := overlays[win.overlay]
		if o.Probability == nil {
			continue
		}
		for _, r := range win.readings {
			key := word + ":" + readingTag(o, r)
			if v := o.Probability.Find(key); v > 0 {
				added[key] = v
			}
		}
	}
	if len(added) == 0 && len(removed) == 0 {
		return base.Probability.Clone(), nil
	}

	kv := make(map[string]uint32)
	if base.Probability != nil {
		base.Probability.WalkValues(func(key string, v uint32) {
			if i := strings.LastIndexByte(key, ':'); i >= 0 && removed[key[:i]] {
				return
			}
			kv[key] = v
		})
	}
	for k, v := range added {
		kv[k] = v
	}
	if len(kv) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	vals := make([]uint32, len(keys))
	for i, k := range keys {
		vals[i] = kv[k]
	}
	return BuildIntDAWG(keys, vals)
}

func readingTag(o *Dictionary, r overlayReading) string {
	if o.TagSet == nil || r.shard >= len(o.Paradigms) || int(r.para) >= len(o.Paradigms[r.shard]) {
		return ""
	}
	p := o.Paradigms[r.shard][r.para]
	if int(r.form) >= p.Len() {
		return ""
	}
	return o.TagSet.TagName(p.Tag(int(r.form)))
}
