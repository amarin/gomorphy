package morphology

import (
	"fmt"
	"time"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// SaveTo записывает словарь в файл GMOR — единый дисковый формат.
// Секции: meta, info, tagset, suffixes, prefixes, paradigms, words.dawg,
// prediction-N, probability (если есть).
func (x *Dictionary) SaveTo(path string) error {
	if x == nil || x.d == nil {
		return fmt.Errorf("morphology: nil dictionary")
	}

	// info — копия x.d.Info (если импортёр её заполнил, например Source),
	// с BuiltAt/LibraryVersion, проставленными заново при каждом
	// сохранении; сам x.d не мутируется (Dictionary иммутабелен).
	info := internal.BuildInfo{}
	if x.d.Info != nil {
		info = *x.d.Info
	}
	info.BuiltAt = time.Now().UTC()
	info.LibraryVersion = Version

	// Сжатие пока не реализовано (см. docs/todo.md, "Этап 17") — все секции
	// пишутся как есть. words.dawg всегда останется CompressionNone: она
	// алиасится из mmap без копирования, а сжатая секция требует полной
	// декомпрессии в память при загрузке.
	const noCompression = internal.CompressionNone
	sections := []internal.Section{
		{Name: "meta", Data: internal.EncodeMeta(x.d.Language, x.d.CharPolicy), Flags: noCompression},
		{Name: "info", Data: internal.EncodeBuildInfo(&info), Flags: noCompression},
		{Name: "tagset", Data: internal.EncodeTagSet(x.d.TagSet), Flags: noCompression},
		{Name: "suffixes", Data: internal.EncodeStrings(x.d.Suffixes), Flags: noCompression},
		{Name: "prefixes", Data: internal.EncodeStrings(x.d.Prefixes), Flags: noCompression},
		{Name: "paradigms", Data: internal.EncodeParadigms(x.d.Paradigms), Flags: noCompression},
		{Name: "words.dawg", Data: x.d.Words.Bytes(), Flags: internal.CompressionNone},
	}
	for i, pred := range x.d.Prediction {
		if pred == nil {
			continue
		}
		sections = append(sections, internal.Section{
			Name: fmt.Sprintf("prediction-%d", i), Data: pred.Bytes(), Flags: noCompression,
		})
	}
	if x.d.Probability != nil {
		sections = append(sections, internal.Section{
			Name: "probability", Data: x.d.Probability.Bytes(), Flags: noCompression,
		})
	}
	return internal.SaveContainer(path, sections)
}
