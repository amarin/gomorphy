package internal

import (
	"encoding/binary"
	"fmt"
	"slices"
	"unicode/utf8"
)

// predictionMaxSuffix is the longest suffix key (in runes) the prediction
// DAWG indexes, matching the engine's suffixSplits(word, 5) lookup in
// pkg/morphology/parse.go.
const predictionMaxSuffix = 5

// predictionMaxCount caps the attested-reading count stored in a prediction
// payload: the field is a uint16 in the on-disk format.
const predictionMaxCount = 0xffff

// WordValue is one raw words.dawg reading: a plain-text wordform and its
// (paradigm<<16 | form) value.
type WordValue struct {
	Word  string
	Value uint32
}

// PredictionPruning trims a prediction DAWG the way pymorphy2's dictionary
// compiler does. The zero value keeps everything.
type PredictionPruning struct {
	// MinEndingFreq drops suffix keys attested by fewer readings.
	MinEndingFreq int
	// MinParadigmPopularity ignores paradigms used by fewer lemmas
	// (form-0 readings).
	MinParadigmPopularity int
	// MaxFormsPerClass keeps, per suffix and part of speech (the tag's
	// first grammeme), only this many most attested (paradigm, form)
	// entries; 0 keeps all.
	MaxFormsPerClass int
}

// ImportPredictionPruning is pymorphy2's compile default, used for the
// OpenCorpora and UniMorph imports.
var ImportPredictionPruning = PredictionPruning{MinEndingFreq: 2, MinParadigmPopularity: 3, MaxFormsPerClass: 1}

// BuildPrediction rebuilds d.Prediction (one DAWG, prefix id 0) from the
// words of every shard of d and sets d.PredictionSharded. Keys are decoded
// through d.Alphabet, so d may be raw or dense. A dictionary with no shards
// is left untouched. Equivalent to BuildPredictionPruned with the zero
// PredictionPruning (no pruning).
func BuildPrediction(d *Dictionary, productive func(tag string) bool) error {
	return BuildPredictionPruned(d, productive, PredictionPruning{})
}

// BuildPredictionPruned is like BuildPrediction, but trims the resulting
// prediction DAWG according to p (see PredictionPruning). The zero
// PredictionPruning keeps everything, matching BuildPrediction.
func BuildPredictionPruned(d *Dictionary, productive func(tag string) bool, p PredictionPruning) error {
	if d == nil || len(d.Words) == 0 {
		return nil
	}
	pairs := make([][]WordValue, len(d.Words))
	for s, w := range d.Words {
		wp, err := shardPairs(w, d.Alphabet, nil)
		if err != nil {
			return fmt.Errorf("prediction: shard %d: %w", s, err)
		}
		pairs[s] = wp
	}
	pred, sharded, err := buildPrediction(pairs, d.Paradigms, d.Suffixes, d.Prefixes, d.TagSet, productive, p)
	if err != nil {
		return err
	}
	d.Prediction = []*DAWG{pred}
	d.PredictionSharded = sharded
	return nil
}

// BuildPredictionFrom builds the pymorphy2 KnownSuffixAnalyzer prediction
// DAWG for prefix id 0, with no pruning. pairs[s] are shard s's raw (word,
// value) readings, resolved against paradigms[s], suffixes[s], prefixes
// and tagSet. For every reading whose tag is productive and whose form
// prefix is empty, the word's last max(len(form suffix), 1)..5 runes
// become suffix keys (a form suffix longer than 5 runes gives none, as in
// pymorphy2's compiler); readings sharing a (suffix, paradigm, form,
// shard) key accumulate a count. With one shard each key becomes
// count(BE16) + para(BE16) + form(BE16); with more, shard(BE16) is
// appended and sharded is true.
func BuildPredictionFrom(pairs [][]WordValue, paradigms [][]Paradigm, suffixes [][]string, prefixes []string, tagSet *TagSet, productive func(tag string) bool) (*DAWG, bool, error) {
	return buildPrediction(pairs, paradigms, suffixes, prefixes, tagSet, productive, PredictionPruning{})
}

// predKey identifies one prediction payload: an attested (paradigm, form)
// reading of a shard, keyed by the wordform suffix it was seen under.
type predKey struct {
	suffix            string
	para, form, shard uint16
}

// buildPrediction is the shared implementation behind BuildPredictionFrom
// (p is the zero PredictionPruning) and BuildPredictionPruned. See
// PredictionPruning and the design doc (spec L, "Pruning") for the rules.
//
// Key lengths follow pymorphy2's compiler (ruling R14): a reading yields
// keys of max(len(form suffix), 1)..predictionMaxSuffix runes, so every key
// ends with the form's suffix and readingForm can strip it; a reading whose
// form suffix is longer than predictionMaxSuffix yields none. Readings
// whose form has a non-empty paradigm prefix (OpenCorpora «по»/«наи»
// comparatives) are skipped: this DAWG serves prefix id 0, and a word
// predicted through it must not need a prefix.
func buildPrediction(pairs [][]WordValue, paradigms [][]Paradigm, suffixes [][]string, prefixes []string, tagSet *TagSet, productive func(tag string) bool, p PredictionPruning) (*DAWG, bool, error) {
	sharded := len(pairs) > 1

	var skip map[[2]uint16]bool
	if p.MinParadigmPopularity > 0 {
		popularity := make(map[[2]uint16]int)
		for s, shard := range pairs {
			for _, wv := range shard {
				para, form := uint16(wv.Value>>16), uint16(wv.Value)
				if form != 0 {
					continue
				}
				popularity[[2]uint16{uint16(s), para}]++
			}
		}
		skip = make(map[[2]uint16]bool)
		for k, n := range popularity {
			if n < p.MinParadigmPopularity {
				skip[k] = true
			}
		}
	}

	counts := make(map[predKey]int)
	var endingFreq map[string]int
	if p.MinEndingFreq > 0 {
		endingFreq = make(map[string]int)
	}
	for s, shard := range pairs {
		if s >= len(paradigms) {
			break
		}
		ps := paradigms[s]
		var ss []string
		if s < len(suffixes) {
			ss = suffixes[s]
		}
		for _, wv := range shard {
			para, form := uint16(wv.Value>>16), uint16(wv.Value)
			if int(para) >= len(ps) || int(form) >= ps[para].Len() {
				continue
			}
			if skip != nil && skip[[2]uint16{uint16(s), para}] {
				continue
			}
			tag := ""
			if tagSet != nil {
				tag = tagSet.TagName(ps[para].Tag(int(form)))
			}
			if !productive(tag) {
				continue
			}
			if stringAt(prefixes, ps[para].Prefix(int(form))) != "" {
				continue
			}
			minLen := max(utf8.RuneCountInString(stringAt(ss, ps[para].Suffix(int(form)))), 1)
			if minLen > predictionMaxSuffix {
				continue
			}
			rr := []rune(wv.Word)
			maxLen := min(predictionMaxSuffix, len(rr))
			for l := minLen; l <= maxLen; l++ {
				suffix := string(rr[len(rr)-l:])
				counts[predKey{suffix: suffix, para: para, form: form, shard: uint16(s)}]++
				if endingFreq != nil {
					endingFreq[suffix]++
				}
			}
		}
	}

	if endingFreq != nil {
		for k := range counts {
			if endingFreq[k.suffix] < p.MinEndingFreq {
				delete(counts, k)
			}
		}
	}

	if p.MaxFormsPerClass > 0 {
		counts = prunePredictionClasses(counts, paradigms, tagSet, p.MaxFormsPerClass)
	}

	width := 6
	if sharded {
		width = 8
	}
	keys := make([]string, 0, len(counts))
	values := make([][]byte, 0, len(counts))
	for k, count := range counts {
		count = min(count, predictionMaxCount)
		buf := make([]byte, width)
		binary.BigEndian.PutUint16(buf[0:2], uint16(count))
		binary.BigEndian.PutUint16(buf[2:4], k.para)
		binary.BigEndian.PutUint16(buf[4:6], k.form)
		if sharded {
			binary.BigEndian.PutUint16(buf[6:8], k.shard)
		}
		keys = append(keys, k.suffix)
		values = append(values, buf)
	}
	pred, err := BuildDAWGWithValuesBytes(keys, values)
	return pred, sharded, err
}

// prunePredictionClasses applies PredictionPruning.MaxFormsPerClass: keys
// are grouped by (suffix, FirstGrammeme(tag of para/form)) and each group
// keeps its max entries with the highest count, ties broken by the lowest
// (shard, para, form).
func prunePredictionClasses(counts map[predKey]int, paradigms [][]Paradigm, tagSet *TagSet, max int) map[predKey]int {
	type classKey struct {
		suffix string
		pos    string
	}
	groups := make(map[classKey][]predKey, len(counts))
	for k := range counts {
		tag := ""
		if tagSet != nil && int(k.shard) < len(paradigms) {
			ps := paradigms[k.shard]
			if int(k.para) < len(ps) && int(k.form) < ps[k.para].Len() {
				tag = tagSet.TagName(ps[k.para].Tag(int(k.form)))
			}
		}
		ck := classKey{suffix: k.suffix, pos: FirstGrammeme(tag)}
		groups[ck] = append(groups[ck], k)
	}

	pruned := make(map[predKey]int, len(counts))
	for _, keys := range groups {
		slices.SortFunc(keys, func(a, b predKey) int {
			if ca, cb := counts[a], counts[b]; ca != cb {
				return cb - ca
			}
			if a.shard != b.shard {
				return int(a.shard) - int(b.shard)
			}
			if a.para != b.para {
				return int(a.para) - int(b.para)
			}
			return int(a.form) - int(b.form)
		})
		if len(keys) > max {
			keys = keys[:max]
		}
		for _, k := range keys {
			pruned[k] = counts[k]
		}
	}
	return pruned
}
