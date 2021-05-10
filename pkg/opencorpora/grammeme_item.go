package opencorpora

import (
	"github.com/amarin/binutils"
	"github.com/amarin/gomorphy/internal/grammeme"
)

// Граммема OpenCorpora является описанием грамматической категории.
type Grammeme struct {
	// Ссылка принадлежности к обобщающей категории
	ParentAttr grammeme.Name `xml:"parent,attr"`
	// Наименование категории. Аббревиатура от латинского или англоязычного наименования. Всегда 4 символа
	Name grammeme.Name `xml:"name"`
	// Кириллическая аббревиатура наименования.
	Alias string `xml:"alias"`
	// Полное наименование на русском языке
	Description string `xml:"description"`
}

func (g Grammeme) String() string {
	return "Grammeme(" + string(g.Name) + "/" + string(g.ParentAttr) + "," + g.Alias + "," + g.Description + ")"
}

func (g *Grammeme) UnmarshalFromBuffer(buffer *binutils.Buffer) (err error) {
	if err = buffer.ReadObject(&g.Name); err != nil {
		return WrapOpenCorporaErrorf(err, "cant read grammeme name")
	} else if err = buffer.ReadObject(&g.ParentAttr); err != nil {
		return WrapOpenCorporaErrorf(err, "cant read parent grammeme name")
	} else if err = buffer.ReadString(&g.Alias); err != nil {
		return WrapOpenCorporaErrorf(err, "cant read alias")
	} else if err = buffer.ReadString(&g.Description); err != nil {
		return WrapOpenCorporaErrorf(err, "cant read description")
	}

	return err
}

// Байтовое представление граммемы
// Все строковые параметры записываются как строки, завершённые нулевым байтом
func (g Grammeme) MarshalBinary() ([]byte, error) {
	buffer := binutils.NewEmptyBuffer()

	if _, err := buffer.WriteObject(&g.Name); err != nil {
		return []byte{}, WrapOpenCorporaErrorf(err, "cant write name")
	} else if _, err = buffer.WriteObject(&g.ParentAttr); err != nil {
		return []byte{}, WrapOpenCorporaErrorf(err, "cant parent name")
	} else if _, err = buffer.WriteString(g.Alias); err != nil {
		return []byte{}, WrapOpenCorporaErrorf(err, "cant alias")
	} else if _, err = buffer.WriteString(g.Description); err != nil {
		return []byte{}, WrapOpenCorporaErrorf(err, "cant description")
	}

	return buffer.Bytes(), nil
}

// Загрузить байтовое представление
func (g *Grammeme) UnmarshalBinary(data []byte) error {
	return g.UnmarshalFromBuffer(binutils.NewBuffer(data))
}
