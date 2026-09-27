package morphology

import "strings"

// Forms returns every form of r's lexeme — the paradigm r.Para in r.Shard,
// with the stem derived from r.Word and r.Form — in paradigm order (form 0
// is the lemma). r must be a reading of this dictionary (from Parse,
// ParseAppend or Forms); a reading that does not fit its paradigm yields
// nil. Every returned Reading carries r's Para, Shard, Dict and Predicted,
// Normal = form 0, Prob = 0. For a predicted reading the forms are
// generated from the predicted paradigm and are as much a guess as the
// reading itself.
func (x *Dictionary) Forms(r Reading) []Reading {
	if x == nil || x.d == nil || r.Word == "" {
		return nil
	}
	para, ok := x.paradigm(r.Shard, r.Para)
	if !ok || int(r.Form) >= para.Len() {
		return nil
	}
	prefix, suffix := x.paradigmAffix(r.Shard, para, int(r.Form))
	if !strings.HasPrefix(r.Word, prefix) || !strings.HasSuffix(r.Word[len(prefix):], suffix) {
		return nil
	}
	stem := r.Word[len(prefix) : len(r.Word)-len(suffix)]

	p0, s0 := x.paradigmAffix(r.Shard, para, 0)
	normal := p0 + stem + s0
	out := make([]Reading, 0, para.Len())
	for i := 0; i < para.Len(); i++ {
		p, s := x.paradigmAffix(r.Shard, para, i)
		out = append(out, Reading{
			Word:      p + stem + s,
			Normal:    normal,
			Tag:       x.paradigmTag(para, i),
			Para:      r.Para,
			Form:      uint16(i),
			Shard:     r.Shard,
			Dict:      r.Dict,
			Predicted: r.Predicted,
		})
	}
	return out
}
