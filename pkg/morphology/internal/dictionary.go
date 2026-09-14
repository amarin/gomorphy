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
	Info        *BuildInfo
}

// NewDictionary собирает Dictionary из компонентов. Prediction, Probability
// и Info остаются nil; Prediction/Probability заполняются импортёрами при
// чтении prediction-файлов, Info — импортёром (обычно только Source) или
// SaveTo (BuiltAt/LibraryVersion — при каждом сохранении).
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
