package grammemes

import (
	"encoding/hex"
	"math"

	"github.com/amarin/binutils"
)

const (
	binaryPrefix = "GIdx"
)

var (
	binaryPrefixBytesCount = len([]byte(binaryPrefix)) // nolint:gochecknoglobals
	indexBinaryPrefixBytes = []byte(binaryPrefix)      // nolint:gochecknoglobals
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
func (x Index) Idx(name GrammemeName) (uint8, error) {
	for idx, grammeme := range x.knownGrammemes {
		if grammeme.Name == name {
			return uint8(idx), nil
		}
	}

	return 0, NewErrorf("grammeme name `%v` not found", name)
}

// ByIdx позволяет получить граммему из индекса по известному идентификатору.
// Возвращает ошибку, если идентификатор в индексе не найден.
func (x Index) ByIdx(requiredIdx uint8) (*Grammeme, error) {
	for idx, grammeme := range x.knownGrammemes {
		if uint8(idx) == requiredIdx {
			return &grammeme, nil
		}
	}

	return nil, NewErrorf("grammeme %v not found", requiredIdx)
}

// MarshalBinary упаковывает индекс граммем в двоичную последовательность.
// В двоичном виде список граммем содержит длину списка listLen в 1м байте.
// и набор элементов списка длиной listLen.
// В случае, размер данных в последовательности не соответствует списку категорий, возвращает ошибку.
func (x Index) MarshalBinary() (data []byte, err error) {
	buf := binutils.NewEmptyBuffer()
	// write grammeme list len first. One byte enough
	if _, err = buf.WriteUint8(uint8(len(x.knownGrammemes))); err != nil {
		return buf.Bytes(), WrapErrorf(err, "cant write length byte")
	}

	for idx, grammeme := range x.knownGrammemes {
		if _, err = buf.WriteObject(grammeme); err != nil {
			return buf.Bytes(), WrapErrorf(err, "cant write grammeme %d", idx)
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
		return NewErrorf("Expected at least %d byte", minBytes)
	}
	// read prefix from buffer
	if err := buffer.ReadBytes(&prefix, binaryPrefixBytesCount); err != nil {
		return WrapErrorf(err, "expected prefix %v", binaryPrefix)
	} else if string(prefix) != binaryPrefix {
		return NewErrorf(
			"expected prefix %v not %v",
			hex.EncodeToString(indexBinaryPrefixBytes),
			hex.EncodeToString(prefix),
		)
	}

	if err := buffer.ReadUint8(&listLen); err != nil {
		return WrapErrorf(err, "cant take buffer len")
	}

	grammemesInFile := make([]Grammeme, listLen)
	for idx := 0; uint8(idx) < listLen; idx++ {
		grammeme := new(Grammeme)
		if err := grammeme.UnmarshalFromBuffer(buffer); err != nil {
			return WrapErrorf(err, "cant unmarshal grammeme")
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
		return WrapErrorf(err, "cant unmarshal from buffer")
	} else if buffer.Len() > 0 {
		return NewErrorf("extra %d bytes in bytes array", buffer.Len())
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
			return NewErrorf("grammeme `%v` already exists", existedGrammeme.Name)
		} else if existedGrammeme.Alias == grammeme.Alias {
			return NewErrorf("grammeme alias `%v` already exists", existedGrammeme.Alias)
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
			return NewErrorf("parent `%v`not found", grammeme.ParentAttr)
		}
	}
	// prevent grammemes numbering overflow
	if len(x.knownGrammemes) == math.MaxUint8 {
		return NewErrorf("too many grammemes, max expected %v", math.MaxUint8)
	}

	x.knownGrammemes = append(x.knownGrammemes, grammeme)

	return nil // no errors
}

// ByName находит граммему в индексе по имени.
// Имя граммемы не должно совпадать с именем уже существующей граммемы.
// Группа (родительская граммема) должна быть уже добавлена (за исключением граммем не имеющих группы).
// Кириллическая аббревиатура наименования не должна совпадать с аббревиатурой уже существующей граммемы.
func (x *Index) ByName(name GrammemeName) (*Grammeme, error) {
	for _, existedGrammeme := range x.knownGrammemes {
		if existedGrammeme.Name == name {
			return &existedGrammeme, nil
		}
	}

	return nil, NewErrorf("grammeme `%v` not found", name) // no such name
}

// NewList создаёт новый список граммем привязанных к индексу.
func (x *Index) NewList(grammemes ...*Grammeme) *List {
	return NewList(x, grammemes...)
}
