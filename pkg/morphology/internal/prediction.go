package internal

import (
	"encoding/binary"
	"fmt"
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

// BuildPrediction rebuilds d.Prediction (one DAWG, prefix id 0) from the
// words of every shard of d and sets d.PredictionSharded. Keys are decoded
// through d.Alphabet, so d may be raw or dense. A dictionary with no shards
// is left untouched.
func BuildPrediction(d *Dictionary, productive func(tag string) bool) error {
	if d == nil || len(d.Words) == 0 {
		return nil
	}
	pairs := make([][]WordValue, len(d.Words))
	for s, w := range d.Words {
		p, err := shardPairs(w, d.Alphabet, nil)
		if err != nil {
			return fmt.Errorf("prediction: shard %d: %w", s, err)
		}
		pairs[s] = p
	}
	pred, sharded, err := BuildPredictionFrom(pairs, d.Paradigms, d.TagSet, productive)
	if err != nil {
		return err
	}
	d.Prediction = []*DAWG{pred}
	d.PredictionSharded = sharded
	return nil
}

// BuildPredictionFrom builds the pymorphy2 KnownSuffixAnalyzer prediction
// DAWG for prefix id 0. pairs[s] are shard s's raw (word, value) readings,
// resolved against paradigms[s] and tagSet. For every reading whose tag is
// productive, the word's last 1..5 runes become suffix keys; readings
// sharing a (suffix, paradigm, form, shard) key accumulate a count. With
// one shard each key becomes count(BE16) + para(BE16) + form(BE16); with
// more, shard(BE16) is appended and sharded is true.
func BuildPredictionFrom(pairs [][]WordValue, paradigms [][]Paradigm, tagSet *TagSet, productive func(tag string) bool) (*DAWG, bool, error) {
	type predKey struct {
		suffix            string
		para, form, shard uint16
	}
	sharded := len(pairs) > 1
	counts := make(map[predKey]int)
	for s, shard := range pairs {
		if s >= len(paradigms) {
			break
		}
		ps := paradigms[s]
		for _, p := range shard {
			para, form := uint16(p.Value>>16), uint16(p.Value)
			if int(para) >= len(ps) || int(form) >= ps[para].Len() {
				continue
			}
			tag := ""
			if tagSet != nil {
				tag = tagSet.TagName(ps[para].Tag(int(form)))
			}
			if !productive(tag) {
				continue
			}
			rr := []rune(p.Word)
			max := min(predictionMaxSuffix, len(rr))
			for l := 1; l <= max; l++ {
				counts[predKey{suffix: string(rr[len(rr)-l:]), para: para, form: form, shard: uint16(s)}]++
			}
		}
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
