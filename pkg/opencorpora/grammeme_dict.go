package opencorpora

import (
	"strings"

	"github.com/amarin/binutils"
)

// Список граммем содержит определение всех грамматических категорий
type Grammemes struct {
	Grammeme []*Grammeme `xml:"grammeme"`
}

func (x Grammemes) String() string {
	var stringItems []string
	for _, g := range x.Grammeme {
		stringItems = append(stringItems, g.String())
	}
	return "Grammemes(" + strings.Join(stringItems, ",") + ")"
}

// Marshal binary Grammemes data
func (x Grammemes) MarshalBinary() (data []byte, err error) {
	var res []byte
	for idx, grammeme := range x.Grammeme {
		if d, err := grammeme.MarshalBinary(); err != nil {
			return res, WrapOpenCorporaErrorf(err, "grammeme %d", idx)
		} else {
			res = append(res, d...)
		}
	}
	return res, nil
}

// Unmarshal binary Grammemes data
func (x *Grammemes) UnmarshalBinary(data []byte) error {
	buffer := binutils.NewBuffer(data)
	idx := 0
	for {
		grammeme := new(Grammeme)
		if err := grammeme.UnmarshalFromBuffer(buffer); err != nil {
			return WrapOpenCorporaErrorf(err, "grammeme %d error", idx)
		}
		x.Grammeme = append(x.Grammeme, grammeme)
		if buffer.Len() <= 0 {
			break
		}
	}
	return nil
}

// // Контейнер хранения глобального списка граммем
// var OpenCorporaGrammemesList Grammemes
//
// // при инициализации модуля система создаёт пустой контейнер хранения глобального списка граммем
// func init() {
// 	OpenCorporaGrammemesList = Grammemes{make([]*Grammeme, 0)}
// }
