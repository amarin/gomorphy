package internal

import "encoding/binary"

// predictionMaxSuffix is the longest suffix key (in runes) the prediction
// DAWG indexes, matching the engine's suffixSplits(word, 5) lookup in
// pkg/morphology/parse.go.
const predictionMaxSuffix = 5

// predictionMaxCount caps the attested-reading count stored in a prediction
// payload: the field is a uint16 in the on-disk format.
const predictionMaxCount = 0xffff

// BuildPrediction rebuilds the pymorphy2 KnownSuffixAnalyzer prediction DAWG
// (d.Prediction) from the raw dictionary wordforms in d.Words[0].
//
// For every reading (wordform → paradigm, form) whose tag is productive, the
// wordform's last 1..5 runes become suffix keys; readings sharing a
// (suffix, paradigm, form) triple accumulate an attestation count. Each
// distinct triple becomes one payload leaf: count(BE16) + para(BE16) +
// form(BE16), attached to its suffix key via BuildDAWGWithValuesBytes — the
// exact shape parse.go's predictForPrefix consumes.
//
// The dictionary must be unsharded and still raw (plain-text wordform keys),
// so BuildPrediction has to run before RecompileDense; a sharded dictionary
// (len(d.Words) != 1) is left untouched (no-op). productive is injected
// because it lives in the engine package (pkg/morphology/parse.go) and
// internal must not import it.
func BuildPrediction(d *Dictionary, productive func(tag string) bool) error {
	if d == nil || len(d.Words) != 1 {
		return nil
	}

	var paradigms []Paradigm
	if len(d.Paradigms) > 0 {
		paradigms = d.Paradigms[0]
	}

	type predKey struct {
		suffix string
		para   uint16
		form   uint16
	}
	counts := make(map[predKey]int)

	d.Words[0].Walk(func(word string, values [][]byte) {
		for _, v := range values {
			if len(v) < 4 {
				continue
			}
			para := binary.BigEndian.Uint16(v[:2])
			form := binary.BigEndian.Uint16(v[2:4])
			if int(para) >= len(paradigms) || int(form) >= paradigms[para].Len() {
				continue
			}

			tag := ""
			if d.TagSet != nil {
				tag = d.TagSet.TagName(paradigms[para].Tag(int(form)))
			}
			if !productive(tag) {
				continue
			}

			rr := []rune(word)
			max := predictionMaxSuffix
			if max > len(rr) {
				max = len(rr)
			}
			for l := 1; l <= max; l++ {
				counts[predKey{suffix: string(rr[len(rr)-l:]), para: para, form: form}]++
			}
		}
	})

	keys := make([]string, 0, len(counts))
	values := make([][]byte, 0, len(counts))
	for k, count := range counts {
		if count > predictionMaxCount {
			count = predictionMaxCount
		}
		buf := make([]byte, 6)
		binary.BigEndian.PutUint16(buf[:2], uint16(count))
		binary.BigEndian.PutUint16(buf[2:4], k.para)
		binary.BigEndian.PutUint16(buf[4:6], k.form)
		keys = append(keys, k.suffix)
		values = append(values, buf)
	}

	pred, err := BuildDAWGWithValuesBytes(keys, values)
	if err != nil {
		return err
	}
	d.Prediction = []*DAWG{pred}
	return nil
}
