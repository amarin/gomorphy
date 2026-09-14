package internal

// Dictionary — иммутабельный снимок словаря.
type Dictionary struct {
	Language    string
	TagSet      *TagSet
	Suffixes    []string
	Prefixes    []string
	Paradigms   []Paradigm
	Words       *DAWG
	Prediction  []*DAWG
	Probability *DAWG
	CharPolicy  *CharPolicy
}

// NewDictionary собирает Dictionary из компонентов. Prediction и Probability
// остаются nil; заполняются импортёрами при чтении prediction-файлов.
func NewDictionary(
	language string,
	tagSet *TagSet,
	suffixes, prefixes []string,
	paradigms []Paradigm,
	words *DAWG,
	charPolicy *CharPolicy,
) *Dictionary {
	return &Dictionary{
		Language:   language,
		TagSet:     tagSet,
		Suffixes:   suffixes,
		Prefixes:   prefixes,
		Paradigms:  paradigms,
		Words:      words,
		CharPolicy: charPolicy,
	}
}
