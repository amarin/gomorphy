package tagmap

import "strings"

// tokenizeOpenCorpora splits a dict.xml-style combined tag
// ("NOUN,anim,masc,sing,nomn") into its comma-separated grammeme tokens.
func tokenizeOpenCorpora(tag string) []string {
	if tag == "" {
		return nil
	}
	return strings.Split(tag, ",")
}

// openCorporaTable maps OpenCorpora grammeme names (as produced by
// pkg/morphology/importers/opencorpora's import, TagSet.Name ==
// "opencorpora") to UniMorph features. A representative subset, not
// exhaustive — see docs/en/superpowers/plans/2026-09-17-tag-mapping.md's
// Global Constraints. Grows incrementally as uncovered grammemes are
// found; an uncovered grammeme is not an error (see Bundle.Unmapped).
var openCorporaTable = map[string]Feature{
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
