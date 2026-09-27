package morphology

import (
	"cmp"
	"slices"
	"strings"
)

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

// Inflect returns the forms of r's lexeme (see Forms) whose tags contain
// every grammeme in want, best first: fewest grammemes differing from
// r.Tag (size of the symmetric difference of the two grammeme sets), then
// paradigm order. With no want it returns every form, r's own first.
// Grammemes are native tokens (see Grammemes), e.g. "gent", "plur" or
// "GEN". nil when no form matches.
func (x *Dictionary) Inflect(r Reading, want ...string) []Reading {
	forms := x.Forms(r)
	if len(forms) == 0 {
		return nil
	}
	src := Grammemes(r.Tag)

	type candidate struct {
		reading Reading
		diff    int
	}
	var cands []candidate
	for _, f := range forms {
		if hasAllGrammemes(f.Tag, want) {
			cands = append(cands, candidate{f, grammemeDiff(src, Grammemes(f.Tag))})
		}
	}
	if len(cands) == 0 {
		return nil
	}
	slices.SortStableFunc(cands, func(a, b candidate) int { return cmp.Compare(a.diff, b.diff) })
	out := make([]Reading, len(cands))
	for i, c := range cands {
		out[i] = c.reading
	}
	return out
}

func hasAllGrammemes(tag string, want []string) bool {
	for _, g := range want {
		if !HasGrammeme(tag, g) {
			return false
		}
	}
	return true
}

// grammemeDiff is the size of the symmetric difference of two grammeme
// lists treated as sets.
func grammemeDiff(a, b []string) int {
	n := 0
	for _, g := range a {
		if !slices.Contains(b, g) {
			n++
		}
	}
	for _, g := range b {
		if !slices.Contains(a, g) {
			n++
		}
	}
	return n
}

// Forms returns every form of r's lexeme from the dictionary r came from
// (r.Dict); nil if r.Dict is out of range. See Dictionary.Forms.
func (m *MultiDictionary) Forms(r Reading) []Reading {
	if r.Dict < 0 || r.Dict >= len(m.dicts) {
		return nil
	}
	return m.dicts[r.Dict].Forms(r)
}

// Inflect is Dictionary.Inflect on the dictionary r came from (r.Dict);
// nil if r.Dict is out of range.
func (m *MultiDictionary) Inflect(r Reading, want ...string) []Reading {
	if r.Dict < 0 || r.Dict >= len(m.dicts) {
		return nil
	}
	return m.dicts[r.Dict].Inflect(r, want...)
}
