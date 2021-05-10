package opencorpora

import (
	"math"
)

// LinkType определяет тип связей между частями речи.
// Задаёт возможный тип преобразования из одной части речи в другую.
// Используется в определении связи между леммами Link.
type LinkType struct {
	IDAttr int `xml:"id,attr"`
}

// MarshalBinary сохранение в двоичный формат.
func (l LinkType) MarshalBinary() (data []byte, err error) {
	if l.IDAttr > math.MaxUint8 {
		return []byte{0}, NewErrorf("linkType %d byte, %v overflows %v", 1, l.IDAttr, math.MaxInt8)
	} else if l.IDAttr < 0 {
		return []byte{0}, NewErrorf("linkType %d byte, %v overflows %v", 1, l.IDAttr, math.MaxInt8)
	}
	return []byte{uint8(l.IDAttr)}, nil
}

// UnmarshalBinary загрузка из двоичного формата
func (l *LinkType) UnmarshalBinary(data []byte) error {
	if len(data) != 1 {
		return NewErrorf("linkType %d byte, not %d", 1, len(data))
	}
	l.IDAttr = int(data[0])
	return nil
}
