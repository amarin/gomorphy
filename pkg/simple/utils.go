package simple

import (
	"errors"
	"fmt"
	"math"

	"github.com/amarin/gomorphy/pkg/size"
)

func resolveSize(maxValue int) size.Index {
	switch {
	case maxValue < math.MaxUint8:
		return size.Uint8
	case maxValue < math.MaxUint16:
		return size.Uint16
	case maxValue < math.MaxUint32:
		return size.Uint32
	default:
		return size.Unknown
	}
}

var (
	ErrUnexpectedMorphemesSize    = errors.New("unexpected morphemes size")
	ErrUnexpectedTagsSize         = errors.New("unexpected tags size")
	ErrUnexpectedSizesCombination = errors.New("unexpected sizes combination")
)

func Create(maxCharacters, maxMorphemes, maxTags int) (Interface, error) {
	var alphabetSize, morphemesSize, tagsSize size.Index
	alphabetSize = resolveSize(maxCharacters)

	switch s := resolveSize(maxMorphemes); s {
	case size.Uint16:
		morphemesSize = s
	case size.Uint32:
		morphemesSize = s
	default:
		return nil, fmt.Errorf(
			"%w: expected %d...%d",
			ErrUnexpectedMorphemesSize,
			math.MaxUint8+1, math.MaxUint16,
		)
	}

	switch s := resolveSize(maxTags); s {
	case size.Uint8:
		tagsSize = s
	case size.Uint16:
		tagsSize = s
	default:
		return nil, fmt.Errorf(
			"%w: expected %d...%d",
			ErrUnexpectedTagsSize,
			math.MaxUint16+1, math.MaxUint32,
		)
	}

	switch {
	case alphabetSize == size.Uint8 && morphemesSize == size.Uint16 && tagsSize == size.Uint8:
		// 255 символов, 65535 потенциально возможных слов, 255 тегов, 65535 видов слов (комбинаций тегов)
		return newIndex[uint8, uint16, uint8, uint16](), nil

	case alphabetSize == size.Uint8 && morphemesSize == size.Uint16 && tagsSize == size.Uint16:
		// 255 символов, 65535 потенциально возможных слов, 65535 тегов, 65535 видов слов (комбинаций тегов)
		return newIndex[uint8, uint16, uint16, uint16](), nil

	case alphabetSize == size.Uint8 && morphemesSize == size.Uint32 && tagsSize == size.Uint8:
		// 255 символов, 4294967295 потенциально возможных слов, 255 тегов, 65535 видов слов (комбинаций тегов)
		return newIndex[uint8, uint32, uint8, uint16](), nil

	case alphabetSize == size.Uint8 && morphemesSize == size.Uint32 && tagsSize == size.Uint16:
		// 255 символов, 4294967295 потенциально возможных слов, 65535 тегов, 65535 видов слов (комбинаций тегов)
		return newIndex[uint8, uint32, uint16, uint16](), nil

	// 16 бит на алфавит, до 65535 возможных символов
	case alphabetSize == size.Uint16 && morphemesSize == size.Uint16 && tagsSize == size.Uint8:
		// 65535 символов, 65535 потенциально возможных слов, 255 тегов, 65535 видов слов (комбинаций тегов)
		return newIndex[uint16, uint16, uint8, uint16](), nil
	case alphabetSize == size.Uint16 && morphemesSize == size.Uint16 && tagsSize == size.Uint16:
		// 65535 символов, 65535 потенциально возможных слов, 65535 тегов, 65535 видов слов (комбинаций тегов)
		return newIndex[uint16, uint16, uint16, uint16](), nil
	case alphabetSize == size.Uint16 && morphemesSize == size.Uint32 && tagsSize == size.Uint8:
		// 65535 символов, 4294967295 потенциально возможных слов, 255 тегов, 65535 видов слов (комбинаций тегов)
		return newIndex[uint16, uint32, uint8, uint16](), nil
	case alphabetSize == size.Uint16 && morphemesSize == size.Uint32 && tagsSize == size.Uint16:
		// 65535 символов, 4294967295 потенциально возможных слов, 65535 тегов, 65535 видов слов (комбинаций тегов)
		return newIndex[uint16, uint32, uint16, uint16](), nil
	}

	return nil, fmt.Errorf(
		"%w: alphabet %s morphemes %s tags %s",
		ErrUnexpectedSizesCombination,
		alphabetSize.String(),
		morphemesSize.String(),
		tagsSize.String(),
	)
}
