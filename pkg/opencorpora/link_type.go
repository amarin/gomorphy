package opencorpora

import (
	"math"
)

// Тип связей между частями речи. Определяет возможный тип преобразования из одной части речи в другую.
// Используется в определении связи между леммами Link
type LinkType struct {
	IdAttr int `xml:"id,attr"`
}

// Сохранение в двоичный формат
func (l LinkType) MarshalBinary() (data []byte, err error) {
	if l.IdAttr > math.MaxUint8 {
		return []byte{0}, NewErrorf("linkType %d byte, %v overflows %v", 1, l.IdAttr, math.MaxInt8)
	} else if l.IdAttr < 0 {
		return []byte{0}, NewErrorf("linkType %d byte, %v overflows %v", 1, l.IdAttr, math.MaxInt8)
	}
	return []byte{uint8(l.IdAttr)}, nil
}

// Загрузка из двоичного формата
func (l *LinkType) UnmarshalBinary(data []byte) error {
	if len(data) != 1 {
		return NewErrorf("linkType %d byte, not %d", 1, len(data))
	}
	l.IdAttr = int(data[0])
	return nil
}
