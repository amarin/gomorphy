package internal

// isGrammemeSep reports whether c separates grammemes in a native tag:
// ',' and ' ' (OpenCorpora/pymorphy2, e.g. "NOUN,anim,masc sing,nomn") and
// ';' (UniMorph, e.g. "N;GEN;SG").
func isGrammemeSep(c byte) bool { return c == ',' || c == ' ' || c == ';' }

// NextGrammeme returns the first grammeme token of tag at or after byte
// offset i (leading separators are skipped) and the offset just past it.
// tok is "" once the tag is exhausted. It does not allocate: tok is a
// substring of tag.
func NextGrammeme(tag string, i int) (tok string, next int) {
	for i < len(tag) && isGrammemeSep(tag[i]) {
		i++
	}
	j := i
	for j < len(tag) && !isGrammemeSep(tag[j]) {
		j++
	}
	return tag[i:j], j
}

// FirstGrammeme returns tag's first grammeme — the part of speech in every
// tag format gomorphy imports — or "" for an empty tag.
func FirstGrammeme(tag string) string {
	tok, _ := NextGrammeme(tag, 0)
	return tok
}

// posClasses folds parts of speech that belong to one lexeme into one
// class: an OpenCorpora verb lexeme mixes INFN, VERB, PRTF, PRTS and GRND
// forms, an adjective lexeme ADJF, ADJS and COMP; UniMorph verb lexemes mix
// V, V.PTCP, V.CVB and V.MSDR.
var posClasses = map[string]string{
	"INFN": "VERB", "PRTF": "VERB", "PRTS": "VERB", "GRND": "VERB",
	"ADJS": "ADJF", "COMP": "ADJF",
	"V.PTCP": "V", "V.CVB": "V", "V.MSDR": "V",
}

// POSClass returns the part-of-speech class of tag: its first grammeme,
// with the forms of one lexeme folded together (see posClasses). "" for an
// empty tag.
func POSClass(tag string) string {
	pos := FirstGrammeme(tag)
	if c, ok := posClasses[pos]; ok {
		return c
	}
	return pos
}
