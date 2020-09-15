package opencorpora

import (
	"fmt"
	"strings"

	"github.com/amarin/gomorphy/pkg/categories"
)

// Лемма OpenCorpora представляет основу части речи с присущими ему словоформами
type Lemma struct {
	// Уникальный идентификатор слова
	IdAttr uint32 `xml:"id,attr"`
	// Номер ревизии
	RevAttr uint32 `xml:"rev,attr"`
	// Базовая форма
	L WordForm `xml:"l"`
	// Другие формы этой-же части речи
	F WordFormList `xml:"f"`
}

// // Загрузить двоичные данные леммы из буфера, оставляя остальные данные нетронутыми
// func (x *Lemma) UnmarshalFromBuffer(buffer *binutils.Buffer) (err error) {
// 	err = buffer.ReadUint32(&x.IdAttr, err)
// 	err = buffer.ReadUint32(&x.RevAttr, err)
// 	err = buffer.UnmarshalObject(&x.L, err)
// 	err = buffer.UnmarshalObject(&x.F, err)
// 	if err != nil {
// 		err = WrapOpenCorporaError(err, "Lemma")
// 	}
// 	return err
// }
//
// // Сохранить лемму в двоичном формате
// func (x Lemma) MarshalBinary() (data []byte, err error) {
// 	buffer := binutils.NewEmptyBuffer()
// 	_, err = buffer.WriteUint32(x.IdAttr, err)
// 	_, err = buffer.WriteUint32(x.RevAttr, err)
// 	_, err = buffer.WriteObject(x.L, err)
// 	_, err = buffer.WriteObject(x.F, err)
// 	if err != nil {
// 		err = WrapOpenCorporaError(err, "Lemma")
// 	}
// 	return buffer.Bytes(), err
// }

// // Загрузить двоичные данные леммы
// func (x *Lemma) UnmarshalBinary(data []byte) error {
// 	return x.UnmarshalFromBuffer(binutils.NewBuffer(data))
// }

// Строковое представление леммы. Реализует интерфейс Stringer()
func (x Lemma) String() string {
	str := make([]string, 0)
	for _, item := range x.F {
		str = append(str, item.String())
	}
	return "Lemma(" + fmt.Sprintf("%v", x.IdAttr) + "," + fmt.Sprintf("%v", x.RevAttr) + "," + x.L.String() + ", " + strings.Join(str, ",")
}

// Получить часть речи
func (x Lemma) POS() categories.POS {
	for _, g := range x.L.G {
		if pos := categories.KnownPoses.ByString(g.VAttr.String()); pos != nil {
			return *pos
		}
	}
	panic("Unknown part of speech " + x.L.String())
}

// Установить часть речи
func (x Lemma) SetPOS(pos categories.POS) {
	panic("implement me")
}
