package text

import (
	"github.com/amarin/binutils"
	"golang.org/x/text/encoding/charmap"
)

// RussianText содержит слово или текст на русском языке.
type RussianText string

// NewRussianText создаёт слово или текст из строки.
func NewRussianText(text string) RussianText {
	return RussianText(text)
}

// String возвращает представление текста в строковом типе.
func (w RussianText) String() string {
	return string(w)
}

// Len возвращает длину строки в символах.
func (w RussianText) Len() int {
	return len([]rune(w))
}

// MarshalBinary кодирует русский текст в байтовую строку в кодировке KOI8R.
func (w RussianText) MarshalBinary() (data []byte, err error) {
	data, err = EncodeString(string(w), charmap.KOI8R)
	if err != nil {
		return nil, WrapErrorf(err, "cant encode string")
	}

	return append(data, 0), nil
}

// UnmarshalFromBuffer позволяет загрузить текст в байтовой строке в кодировке KOI8R из буфера.
func (w *RussianText) UnmarshalFromBuffer(buffer *binutils.Buffer) error {
	var currentByte uint8

	encodedBytes := make([]byte, 0)

	for {
		if err := buffer.ReadUint8(&currentByte); err != nil {
			return WrapErrorf(err, "cant read next byte")
		}

		if currentByte == 0 {
			break
		}

		encodedBytes = append(encodedBytes, currentByte)
	}

	decodedString, err := DecodeBytes(encodedBytes, charmap.KOI8R)
	if err != nil {
		return WrapErrorf(err, "character map")
	}

	*w = RussianText(decodedString)

	return nil
}

// UnmarshalBinary распаковывает значение из байтовой строки в кодировке KOI8R.
func (w *RussianText) UnmarshalBinary(data []byte) error {
	return w.UnmarshalFromBuffer(binutils.NewBuffer(data))
}
