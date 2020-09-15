package grammemes

// Grammeme is a structured definition of grammatical category.
// It combines base grammeme name with relation from child to parent.
// Also adds alias for national lang abbreviation and description string.

import (
	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/internal/text"
)

// Grammeme implements storage for grammatical category structure data..
type Grammeme struct {
	ParentAttr  GrammemeName     // Parent grammeme name.
	Name        GrammemeName     // Grammeme name.
	Alias       text.RussianText // Localized grammeme name.
	Description text.RussianText // Grammeme description.
}

// NewGrammeme makes new grammeme with required parent, name, alias and description.
func NewGrammeme(parent GrammemeName, name GrammemeName, alias text.RussianText, desc text.RussianText) *Grammeme {
	return &Grammeme{ParentAttr: parent, Name: name, Alias: alias, Description: desc}
}

// String returns string representation of grammeme. Implements Stringer.
func (g Grammeme) String() string {
	return "Grammeme{" + g.ParentAttr.String() + "," + g.Name.String() + "," +
		g.Alias.String() + "," + g.Description.String() + "}"
}

// UnmarshalFromBuffer takes required bytes fro buffer to unmarshal binary grammeme data.
func (g *Grammeme) UnmarshalFromBuffer(buffer *binutils.Buffer) error {
	var err error

	if err = buffer.ReadObjectBytes(&g.Name, 4); err != nil {
		return WrapErrorf(err, "cant read name 4 bytes")
	} else if err = buffer.ReadObjectBytes(&g.ParentAttr, 4); err != nil {
		return WrapErrorf(err, "cant read parent 4 bytes")
	} else if err = buffer.ReadObject(&g.Alias); err != nil {
		return WrapErrorf(err, "cant read alias")
	} else if err = buffer.ReadObject(&g.Description); err != nil {
		return WrapErrorf(err, "cant read description")
	}

	return nil
}

// MarshalBinary makes binary grammeme data.
// Все строковые параметры записываются как строки, завершённые нулевым байтом.
func (g Grammeme) MarshalBinary() ([]byte, error) {
	var err error

	buffer := binutils.NewEmptyBuffer()
	if _, err = buffer.WriteObject(&g.Name); err != nil {
		return buffer.Bytes(), WrapErrorf(err, "cant write name")
	} else if _, err = buffer.WriteObject(&g.ParentAttr); err != nil {
		return buffer.Bytes(), WrapErrorf(err, "cant write parent")
	} else if _, err = buffer.WriteObject(g.Alias); err != nil {
		return buffer.Bytes(), WrapErrorf(err, "cant write alias")
	} else if _, err = buffer.WriteObject(g.Description); err != nil {
		return buffer.Bytes(), WrapErrorf(err, "cant write description")
	}

	return buffer.Bytes(), nil
}

// UnmarshalBinary распаковывает байтовую строку в данные граммемы.
func (g *Grammeme) UnmarshalBinary(data []byte) error {
	return g.UnmarshalFromBuffer(binutils.NewBuffer(data))
}
