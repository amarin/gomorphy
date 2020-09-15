package opencorpora

import (
	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/internal/grammemes"
)

// Контейнер для представления грамматической категории OpenCorpora.
// Содержит граммему грамматической категории.
type Category struct {
	VAttr grammemes.GrammemeName `xml:"v,attr"`
}

// Строковое представление грамматической категории.
func (x Category) String() string {
	return x.VAttr.String()
}

// MarshalBinary позволяет представить категорию в виде байтовой строки.
func (x Category) MarshalBinary() (data []byte, err error) {
	return x.VAttr.MarshalBinary()
}

// UnmarshalFromBuffer позволяет получить байтовую строку категории из буфера.
func (x *Category) UnmarshalFromBuffer(buffer *binutils.Buffer) (err error) {
	return buffer.ReadObjectBytes(&x.VAttr, 4)
}

// UnmarshalBinary позволяет распаковать байтовую строку.
func (x *Category) UnmarshalBinary(data []byte) error {
	if len(data) != 4 {
		return WrapOpenCorporaErrorf(nil, "expected %d bytes, bot %d", 4, len(data))
	}
	return x.UnmarshalFromBuffer(binutils.NewBuffer(data))
}
