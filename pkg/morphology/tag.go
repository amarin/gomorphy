package morphology

import "github.com/amarin/gomorphy/pkg/morphology/internal"

// Grammemes splits a native tag into grammeme tokens. Separators are ',',
// ' ' and ';', which covers OpenCorpora/pymorphy2 ("NOUN,anim,masc,Surn
// sing,ablt") and UniMorph ("N;GEN;SG"). Empty tokens are dropped; nil for
// a tag without tokens.
func Grammemes(tag string) []string {
	var out []string
	for i := 0; ; {
		tok, next := internal.NextGrammeme(tag, i)
		if tok == "" {
			return out
		}
		out = append(out, tok)
		i = next
	}
}

// HasGrammeme reports whether tag contains the grammeme g as a whole token
// (same separators as Grammemes). It does not allocate.
func HasGrammeme(tag, g string) bool {
	if g == "" {
		return false
	}
	for i := 0; ; {
		tok, next := internal.NextGrammeme(tag, i)
		if tok == "" {
			return false
		}
		if tok == g {
			return true
		}
		i = next
	}
}

// POS returns tag's first grammeme — the part of speech in every tag format
// gomorphy imports ("NOUN", "INFN", "N", …) — or "" for an empty tag.
func POS(tag string) string { return internal.FirstGrammeme(tag) }

// HasGrammeme reports whether the reading's tag contains the grammeme g.
func (r Reading) HasGrammeme(g string) bool { return HasGrammeme(r.Tag, g) }
