package grammeme

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	"github.com/amarin/binutils"
	"github.com/amarin/gomorphy/pkg/common"
)

const (
	binaryPrefix = "GIdx"
)

var (
	binaryPrefixBytesCount = len([]byte(binaryPrefix)) // nolint:gochecknoglobals
	indexBinaryPrefixBytes = []byte(binaryPrefix)      // nolint:gochecknoglobals

	ErrAlreadyExists = errors.New("already exists")
	ErrNotFound      = errors.New("not found")
	ErrIndexOverflow = errors.New("idx overflow")
)

// Index реализует индекс известных граммем и предоставляет уникальные идентификаторы граммем в индексе.
type Index struct {
	knownGrammemes []Grammeme
}

// NewIndex создаёт новый индекс с заданными граммемами.
func NewIndex(grammemes ...Grammeme) *Index {
	return &Index{knownGrammemes: grammemes}
}

// Slice возвращает список граммем в индексе.
func (x Index) Slice() []Grammeme {
	return x.knownGrammemes
}

// Len возвращает количество граммем в индексе.
func (x Index) Len() int {
	return len(x.knownGrammemes)
}

// Idx позволяет получить уникальный идентификатор граммемы в индексе.
// Возвращает ошибку, если граммема в индексе не найдена.
func (x Index) Idx(name Name) (uint8, error) {
	for idx, grammeme := range x.knownGrammemes {
		if grammeme.Name == name {
			return uint8(idx), nil
		}
	}

	return 0, fmt.Errorf("%w: grammeme name `%v`", ErrNotFound, name)
}

// ByIdx позволяет получить граммему из индекса по известному идентификатору.
// Возвращает ошибку, если идентификатор в индексе не найден.
func (x Index) ByIdx(requiredIdx uint8) (*Grammeme, error) {
	for idx, grammeme := range x.knownGrammemes {
		if uint8(idx) == requiredIdx {
			return &grammeme, nil
		}
	}

	return nil, fmt.Errorf("%w: grammeme idx %v", ErrNotFound, requiredIdx)
}

// MarshalBinary упаковывает индекс граммем в двоичную последовательность.
// В двоичном виде список граммем содержит длину списка listLen в 1м байте.
// и набор элементов списка длиной listLen.
// В случае, размер данных в последовательности не соответствует списку категорий, возвращает ошибку.
func (x Index) MarshalBinary() (data []byte, err error) {
	buf := binutils.NewEmptyBuffer()
	// write grammeme list len first. One byte enough
	if _, err = buf.WriteUint8(uint8(len(x.knownGrammemes))); err != nil {
		return buf.Bytes(), fmt.Errorf("%w: cant write length byte: %v", common.ErrMarshal, err)
	}

	for idx, grammeme := range x.knownGrammemes {
		if _, err = buf.WriteObject(grammeme); err != nil {
			return buf.Bytes(), fmt.Errorf("%w: cant write grammeme %d", common.ErrMarshal, idx)
		}
	}

	data = buf.Bytes()
	// Добавить префикс типа.
	return append(indexBinaryPrefixBytes, data...), nil
}

// UnmarshalFromBuffer загружает индекс граммем из двоичного буфера.
// В двоичном виде список граммем содержит длину списка listLen в 1м байте
// и набор элементов списка длиной listLen.
// Данные за пределами длины списка listLen остаются в буфере нетронутыми.
func (x *Index) UnmarshalFromBuffer(buffer *binutils.Buffer) error {
	var (
		listLen uint8
		prefix  []byte
	)
	// read min prefix + 1 byte of list len
	minBytes := binutils.Uint8size + binaryPrefixBytesCount
	// error if no expected lines len
	if buffer.Len() < minBytes {
		return fmt.Errorf("%w: expected at least %d byte", common.ErrUnmarshal, minBytes)
	}
	// read prefix from buffer
	if err := buffer.ReadBytes(&prefix, binaryPrefixBytesCount); err != nil {
		return fmt.Errorf("%w: expected prefix %v", common.ErrUnmarshal, binaryPrefix)
	} else if string(prefix) != binaryPrefix {
		return fmt.Errorf("%w: expected prefix %v not %v",
			common.ErrUnmarshal,
			hex.EncodeToString(indexBinaryPrefixBytes),
			hex.EncodeToString(prefix),
		)
	}

	if err := buffer.ReadUint8(&listLen); err != nil {
		return fmt.Errorf("%w: cant take buffer len: %v", common.ErrUnmarshal, err)
	}

	grammemesInFile := make([]Grammeme, listLen)
	for idx := 0; uint8(idx) < listLen; idx++ {
		grammeme := new(Grammeme)
		if err := grammeme.UnmarshalFromBuffer(buffer); err != nil {
			return fmt.Errorf("%w: cant unmarshal grammeme: %v", common.ErrUnmarshal, err)
		}
		grammemesInFile[idx] = *grammeme
	}
	x.knownGrammemes = append(x.knownGrammemes, grammemesInFile...)

	return nil
}

// UnmarshalBinary распаковывает бинарные данные индекса из бинарной последовательности.
// В двоичном виде список граммем содержит длину списка listLen в 1м байте
// и набор элементов списка длиной listLen.
// В случае, если размер данных в последовательности не соответствует списку граммем, возвращает ошибку.
func (x *Index) UnmarshalBinary(data []byte) error {
	buffer := binutils.NewBuffer(data)
	if err := x.UnmarshalFromBuffer(buffer); err != nil {
		return fmt.Errorf("%w: cant unmarshal from buffer: %v", common.ErrUnmarshal, err)
	} else if buffer.Len() > 0 {
		return fmt.Errorf("%w: extra %d bytes in bytes array", common.ErrUnmarshal, buffer.Len())
	}

	return nil // no errors
}

// Add добавляет новую граммему в индекс.
// Имя граммемы не должно совпадать с именем уже существующей граммемы.
// Группа (родительская граммема) должна быть уже добавлена (за исключением граммем не имеющих группы).
// Кириллическая аббревиатура наименования не должна совпадать с аббревиатурой уже существующей граммемы.
func (x *Index) Add(grammeme Grammeme) error {
	for _, existedGrammeme := range x.knownGrammemes {
		if existedGrammeme.Name == grammeme.Name {
			return fmt.Errorf("%w: grammeme name `%v`", ErrAlreadyExists, existedGrammeme.Name)
		} else if existedGrammeme.Alias == grammeme.Alias {
			return fmt.Errorf("%w: grammeme alias `%v`", ErrAlreadyExists, existedGrammeme.Alias)
		}
	}

	if grammeme.ParentAttr.String() != "" {
		parentFound := false

		for _, existedGrammeme := range x.knownGrammemes {
			if existedGrammeme.Name == grammeme.ParentAttr {
				parentFound = true
				break
			}
		}

		if !parentFound {
			return fmt.Errorf("%w: parent `%v` not found", ErrNotFound, grammeme.ParentAttr)
		}
	}
	// prevent grammemes numbering overflow
	if len(x.knownGrammemes) == math.MaxUint8 {
		return fmt.Errorf("%w: too many grammemes, max expected %v", ErrIndexOverflow, math.MaxUint8)
	}

	x.knownGrammemes = append(x.knownGrammemes, grammeme)

	return nil // no errors
}

// ByName находит граммему в индексе по имени.
// Имя граммемы не должно совпадать с именем уже существующей граммемы.
// Группа (родительская граммема) должна быть уже добавлена (за исключением граммем не имеющих группы).
// Кириллическая аббревиатура наименования не должна совпадать с аббревиатурой уже существующей граммемы.
func (x *Index) ByName(name Name) (*Grammeme, error) {
	for _, existedGrammeme := range x.knownGrammemes {
		if existedGrammeme.Name == name {
			return &existedGrammeme, nil
		}
	}

	return nil, fmt.Errorf("%w: grammeme `%v` not found", ErrNotFound, name) // no such name
}
