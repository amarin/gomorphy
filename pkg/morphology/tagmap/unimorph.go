package tagmap

import "strings"

// tokenizeUniMorph splits a UniMorph bundle ("N;NOM;SG") into its
// semicolon-separated feature tokens.
func tokenizeUniMorph(tag string) []string {
	if tag == "" {
		return nil
	}
	return strings.Split(tag, ";")
}

// unimorphTable maps UniMorph feature tokens (as produced by
// pkg/morphology/importers/unimorph's import, TagSet.Name ==
// "unimorph") to Bundle features. Near-identity: a UniMorph bundle's
// tokens already spell the same feature values openCorporaTable/
// openCorporaIntTable normalize *onto* (this package's Bundle is
// itself the UniMorph Schema — see the package doc comment), so this
// table only needs to sort each token into its Dimension.
//
// A representative subset, not exhaustive — mirrors the sibling
// tables' philosophy (see their doc comments): grows as uncovered
// tokens are found. Left deliberately unmapped, landing in
// Bundle.Unmapped rather than a guessed-at Dimension: compound POS
// markers observed in the real rus data (V.PTCP "participle", V.CVB
// "converb/gerund", NFIN "infinitive" as a form marker rather than a
// tense/mood value) and language-specific features (LGSPEC1, e.g. the
// rus reflexive "-ся" marker) — none of this package's existing
// Dimensions (part of speech, animacy, case, number, gender, tense,
// aspect, mood, voice, person) represents verb finiteness/form, and
// inventing one wasn't needed for this table to be useful.
var unimorphTable = map[string]Feature{
	"N":   {DimPartOfSpeech, "N"},
	"V":   {DimPartOfSpeech, "V"},
	"ADJ": {DimPartOfSpeech, "ADJ"},
	"ADV": {DimPartOfSpeech, "ADV"},
	"PRO": {DimPartOfSpeech, "PRO"},
	"NUM": {DimPartOfSpeech, "NUM"},

	"ANIM": {DimAnimacy, "ANIM"},
	"INAN": {DimAnimacy, "INAN"},

	"NOM": {DimCase, "NOM"},
	"GEN": {DimCase, "GEN"},
	"DAT": {DimCase, "DAT"},
	"ACC": {DimCase, "ACC"},
	"INS": {DimCase, "INS"},
	"ESS": {DimCase, "ESS"},
	"VOC": {DimCase, "VOC"},

	"SG": {DimNumber, "SG"},
	"PL": {DimNumber, "PL"},

	"MASC": {DimGender, "MASC"},
	"FEM":  {DimGender, "FEM"},
	"NEUT": {DimGender, "NEUT"},

	"PRS": {DimTense, "PRS"},
	"PST": {DimTense, "PST"},
	"FUT": {DimTense, "FUT"},

	"PFV":  {DimAspect, "PFV"},
	"IPFV": {DimAspect, "IPFV"},

	"IND": {DimMood, "IND"},
	"IMP": {DimMood, "IMP"},

	"ACT":  {DimVoice, "ACT"},
	"PASS": {DimVoice, "PASS"},

	"1": {DimPerson, "1"},
	"2": {DimPerson, "2"},
	"3": {DimPerson, "3"},
}
