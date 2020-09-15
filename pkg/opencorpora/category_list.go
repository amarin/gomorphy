package opencorpora

import (
	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/internal/grammemes"
)

// CategoryList содержит список категорий. Используется в классификации словоформ.
type CategoryList []*Category

// Получить список категорий из буфера.
// В двоичном виде список категорий содержит длину списка listLen в 1м байте
// и набор элементов списка длиной listLen.
// Данные за пределами длины списка listLen остаются в буфере нетронутыми.
func (c *CategoryList) UnmarshalFromBuffer(buffer *binutils.Buffer) (err error) {
	var listLen uint8

	if buffer.Len() < 1 {
		return NewOpenCorporaError("CategoryList", NewErrorf("Expected at least 1 byte"))
	}

	err = buffer.ReadUint8(&listLen)

	for idx := 0; uint8(idx) < listLen; idx++ {
		category := new(Category)
		if err = category.UnmarshalFromBuffer(buffer); err != nil {
			break
		}

		*c = append(*c, category)
	}

	if err != nil {
		err = WrapOpenCorporaError(err, "CategoryList")
	}

	return err
}

// UnmarshalBinary позволяет распаковать список категорий из последовательности байт.
// В двоичном виде список категорий содержит длину списка listLen в 1м байте
// и набор элементов списка длиной listLen.
// В случае, размер данных в последовательности не соответствует списку категорий, возвращает ошибку.
func (c *CategoryList) UnmarshalBinary(data []byte) error {
	buffer := binutils.NewBuffer(data)
	err := c.UnmarshalFromBuffer(buffer)

	if buffer.Len() > 0 {
		err = WrapOpenCorporaError(err, "CategoryList")
	}

	return err
}

// MarshalBinary позволяет упаковать список категорий в бинарную последовательность.
// В двоичном виде список категорий содержит длину списка listLen в 1м байте
// и набор элементов списка длиной listLen.
// В случае, размер данных в последовательности не соответствует списку категорий, возвращает ошибку.
func (c CategoryList) MarshalBinary() (data []byte, err error) {
	buf := binutils.NewEmptyBuffer()
	// write category list len first. One byte enough
	_, err = buf.WriteUint8(uint8(len(c)))
	for _, category := range c {
		if _, err = buf.WriteObject(category); err != nil {
			break
		}
	}

	if err != nil {
		err = WrapOpenCorporaError(err, "CategoryList")
	}

	return buf.Bytes(), err
}

// GrammemeNames возвращает список имён граммем, заданных в списке категорий.
func (c CategoryList) GrammemeNames() []grammemes.GrammemeName {
	res := make([]grammemes.GrammemeName, 0)
	for _, item := range c {
		res = append(res, item.VAttr)
	}

	return res
}
