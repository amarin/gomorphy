package internal

import "encoding/binary"

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

// BuildPrediction rebuilds d.Prediction from d.Words[0] (see
// BuildPredictionFrom). The dictionary must be unsharded and still raw
// (plain-text keys); a sharded dictionary is left untouched (no-op).
func BuildPrediction(d *Dictionary, productive func(tag string) bool) error {
	if d == nil || len(d.Words) != 1 {
		return nil
	}
	var pairs []WordValue
	d.Words[0].Walk(func(word string, values [][]byte) {
		for _, v := range values {
			if len(v) >= 4 {
				pairs = append(pairs, WordValue{Word: word, Value: binary.BigEndian.Uint32(v[:4])})
			}
		}
	})
	var paradigms []Paradigm
	if len(d.Paradigms) > 0 {
		paradigms = d.Paradigms[0]
	}
	pred, err := BuildPredictionFrom(pairs, paradigms, d.TagSet, productive)
	if err != nil {
		return err
	}
	d.Prediction = []*DAWG{pred}
	return nil
}

// BuildPredictionFrom builds the pymorphy2 KnownSuffixAnalyzer prediction
// DAWG for prefix id 0 from raw (word, value) readings resolved against
// paradigms (shard 0) and tagSet. For every reading whose tag is
// productive, the word's last 1..5 runes become suffix keys; readings
// sharing a (suffix, paradigm, form) triple accumulate a count. Each
// triple becomes one payload: count(BE16) + para(BE16) + form(BE16).
func BuildPredictionFrom(pairs []WordValue, paradigms []Paradigm, tagSet *TagSet, productive func(tag string) bool) (*DAWG, error) {
	type predKey struct {
		suffix string
		para   uint16
		form   uint16
	}
	counts := make(map[predKey]int)
	for _, p := range pairs {
		para, form := uint16(p.Value>>16), uint16(p.Value)
		if int(para) >= len(paradigms) || int(form) >= paradigms[para].Len() {
			continue
		}
		tag := ""
		if tagSet != nil {
			tag = tagSet.TagName(paradigms[para].Tag(int(form)))
		}
		if !productive(tag) {
			continue
		}
		rr := []rune(p.Word)
		max := min(predictionMaxSuffix, len(rr))
		for l := 1; l <= max; l++ {
			counts[predKey{suffix: string(rr[len(rr)-l:]), para: para, form: form}]++
		}
	}

	keys := make([]string, 0, len(counts))
	values := make([][]byte, 0, len(counts))
	for k, count := range counts {
		count = min(count, predictionMaxCount)
		buf := make([]byte, 6)
		binary.BigEndian.PutUint16(buf[:2], uint16(count))
		binary.BigEndian.PutUint16(buf[2:4], k.para)
		binary.BigEndian.PutUint16(buf[4:6], k.form)
		keys = append(keys, k.suffix)
		values = append(values, buf)
	}
	return BuildDAWGWithValuesBytes(keys, values)
}
