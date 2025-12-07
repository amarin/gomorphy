package tag

import (
	"errors"
	"fmt"

	"github.com/amarin/binutils"
)

const (
	// EmptyTagName defines constant tag name to replace empty strings and distinct unfilled and empty cases.
	EmptyTagName Name = "----"

	tagNameLen = 4
)

var (
	// Error indicates grammar-related errors.
	Error = errors.New("grammar")

	// ErrName указывает на ошибки с именем тега
	ErrName = errors.New("tag name")

	errRead  = fmt.Errorf("%w: read", ErrName)
	errWrite = fmt.Errorf("%w: write", ErrName)

	emptyTagNameBytes = []byte{0x20, 0x20, 0x20, 0x20}
)

// Name представляет имя тега. Использует 4 4 ASCII символа для именования.
type Name string

func (g *Name) BinaryReadFrom(reader *binutils.BinaryReader) (err error) {
	var tagNameBytes []byte
	if tagNameBytes, err = reader.ReadBytesCount(tagNameLen); err != nil {
		return errors.Join(errRead, err)
	}

	if err = g.UnmarshalBinary(tagNameBytes); err != nil {
		return err
	}

	return nil
}

// BinaryWriteTo writes Name data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (g *Name) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	var nameBytes []byte

	if nameBytes, err = g.MarshalBinary(); err != nil {
		return err
	}

	if err = writer.WriteBytes(nameBytes); err != nil {
		return errors.Join(errWrite, err)
	}

	return nil
}

// String возвращает имя в формате строки. Реализует интерфейс fmt.Stringer.
func (g *Name) String() string {
	return string(*g)
}

// MarshalBinary возвращает байтовое представление имени.
// Всегда возвращает 4 байта. EmptyTagName возвращает 4 байта, соответствующие символу пробела.
func (g *Name) MarshalBinary() (data []byte, err error) {
	if len(*g) == 0 || *g == EmptyTagName {
		return []byte("    "), nil
	}

	res := []byte(*g)

	if len(res) != tagNameLen {
		return []byte{}, fmt.Errorf("%w: expect %d bytes, %d generated", ErrName, tagNameLen, len(res))
	}

	return res, nil
}

// UnmarshalBinary извлекает значение имени из байтового представления.
// Требует, чтобы на входе было ровно 4 байта.
func (g *Name) UnmarshalBinary(nameBytes []byte) error {
	if len(nameBytes) != tagNameLen {
		return fmt.Errorf("%w: expect %d bytes, not %d", ErrName, tagNameLen, len(nameBytes))
	}

	isEmpty := false
	for idx := 0; idx < len(emptyTagNameBytes); idx++ {
		isEmpty = nameBytes[idx] == emptyTagNameBytes[idx]
		if !isEmpty {
			break
		}
	}

	if isEmpty || string(nameBytes) == "    " {
		*g = EmptyTagName
	} else {
		*g = Name(nameBytes)
	}

	return nil
}
