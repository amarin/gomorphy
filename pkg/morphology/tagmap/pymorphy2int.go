package tagmap

import "strings"

// tokenizeOpenCorporaInt splits a pymorphy2 gramtab-opencorpora-int tag
// ("NOUN,anim,masc sing,nomn" — lemma-part and form-changing part
// separated by a space, each comma-joined) into a flat grammeme token
// list; the structural space is discarded, both parts are just
// grammemes to the mapping table. A tag with a single group (no space)
// is still split by comma.
func tokenizeOpenCorporaInt(tag string) []string {
	if tag == "" {
		return nil
	}
	var tokens []string
	for _, group := range strings.Fields(tag) {
		tokens = append(tokens, strings.Split(group, ",")...)
	}
	return tokens
}

// openCorporaIntTable maps pymorphy2's gramtab-opencorpora-int grammeme
// names (TagSet.Name == "opencorpora-int") to UniMorph features.
// Deliberately not shared with openCorporaTable (see this task's doc
// comment in the implementation plan) even though its content is
// currently identical — the two sources are independent inputs to this
// package and are kept independently editable.
var openCorporaIntTable = map[string]Feature{
	"NOUN": {DimPartOfSpeech, "N"},
	"VERB": {DimPartOfSpeech, "V"},
	"INFN": {DimPartOfSpeech, "V"},
	"ADJF": {DimPartOfSpeech, "ADJ"},
	"ADVB": {DimPartOfSpeech, "ADV"},
	"NPRO": {DimPartOfSpeech, "PRO"},
	"NUMR": {DimPartOfSpeech, "NUM"},

	"anim": {DimAnimacy, "ANIM"},
	"inan": {DimAnimacy, "INAN"},

	"nomn": {DimCase, "NOM"},
	"gent": {DimCase, "GEN"},
	"datv": {DimCase, "DAT"},
	"accs": {DimCase, "ACC"},
	"ablt": {DimCase, "INS"},
	"loct": {DimCase, "ESS"},
	"voct": {DimCase, "VOC"},

	"sing": {DimNumber, "SG"},
	"plur": {DimNumber, "PL"},

	"masc": {DimGender, "MASC"},
	"femn": {DimGender, "FEM"},
	"neut": {DimGender, "NEUT"},

	"pres": {DimTense, "PRS"},
	"past": {DimTense, "PST"},
	"futr": {DimTense, "FUT"},

	"perf": {DimAspect, "PFV"},
	"impf": {DimAspect, "IPFV"},

	"indc": {DimMood, "IND"},
	"impr": {DimMood, "IMP"},

	"actv": {DimVoice, "ACT"},
	"pssv": {DimVoice, "PASS"},

	"1per": {DimPerson, "1"},
	"2per": {DimPerson, "2"},
	"3per": {DimPerson, "3"},
}
