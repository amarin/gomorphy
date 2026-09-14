package morphology

import (
	"fmt"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// SaveTo записывает словарь в файл GMOR — единый дисковый формат.
// Секции: meta, tagset, suffixes, prefixes, paradigms, words.dawg,
// prediction-N, probability (если есть).
func (x *Dictionary) SaveTo(path string) error {
	if x == nil || x.d == nil {
		return fmt.Errorf("morphology: nil dictionary")
	}

	flags := internal.FlagNone
	sections := []internal.Section{
		{Name: "meta", Data: internal.EncodeMeta(x.d.Language, x.d.CharPolicy), Flags: flags},
		{Name: "tagset", Data: internal.EncodeTagSet(x.d.TagSet), Flags: flags},
		{Name: "suffixes", Data: internal.EncodeStrings(x.d.Suffixes), Flags: flags},
		{Name: "prefixes", Data: internal.EncodeStrings(x.d.Prefixes), Flags: flags},
		{Name: "paradigms", Data: internal.EncodeParadigms(x.d.Paradigms), Flags: flags},
		{Name: "words.dawg", Data: x.d.Words.Bytes(), Flags: flags},
	}
	for i, pred := range x.d.Prediction {
		if pred == nil {
			continue
		}
		sections = append(sections, internal.Section{
			Name: fmt.Sprintf("prediction-%d", i), Data: pred.Bytes(), Flags: flags,
		})
	}
	if x.d.Probability != nil {
		sections = append(sections, internal.Section{
			Name: "probability", Data: x.d.Probability.Bytes(), Flags: flags,
		})
	}
	return internal.SaveContainer(path, sections)
}
