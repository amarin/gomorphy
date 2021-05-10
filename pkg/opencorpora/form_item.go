package opencorpora

import (
	"strings"

	"github.com/amarin/binutils"
	"github.com/amarin/gomorphy/internal/grammeme"

	"github.com/amarin/gomorphy/internal/text"
	"github.com/amarin/gomorphy/pkg/categories"
	"github.com/amarin/gomorphy/pkg/words"
)

// WordForm задаёт структуру словоформы. Содержит текстовое представление и набор грамматических категорий.
type WordForm struct {
	Form text.RussianText `xml:"t,attr"`
	G    CategoryList     `xml:"g"`
}

func (x WordForm) MarshalBinary() (data []byte, err error) {
	buffer := binutils.NewEmptyBuffer()
	if _, err = buffer.WriteObject(x.Form); err != nil {
		return []byte{}, WrapOpenCorporaErrorf(err, "cant write word text")
	} else if _, err = buffer.WriteObject(x.G); err != nil {
		return []byte{}, WrapOpenCorporaErrorf(err, "cant write category list")
	}

	return buffer.Bytes(), err
}

func (x *WordForm) UnmarshalFromBuffer(buffer *binutils.Buffer) (err error) {
	if err = buffer.ReadObject(&x.Form); err != nil {
		return WrapOpenCorporaErrorf(err, "cant read word text")
	} else if err = buffer.ReadObject(&x.G); err != nil {
		return WrapOpenCorporaErrorf(err, "cant read category list")
	}

	return err
}
func (x *WordForm) UnmarshalBinary(data []byte) error {
	return x.UnmarshalFromBuffer(binutils.NewBuffer(data))
}

// ---------------------------------------------------------------------------------------------------------------
// Общие функции
// ---------------------------------------------------------------------------------------------------------------
// Получить текстовое представление.
// Реализует Stringer
func (x WordForm) String() string {
	str := make([]string, 0)
	for _, item := range x.G {
		str = append(str, item.String())
	}
	return "WordForm(" + x.Form.String() + "," + strings.Join(str, ",") + ")"
}

// Получить набор категорий из возможного набора
// Если не найдено, возвращает nil
func (x WordForm) GetTagsFromSet(namesSet []string) []*Category {
	var resultTags []*Category
	for _, g := range x.G {
		for _, name := range namesSet {
			if name == string(g.VAttr) {
				resultTags = append(resultTags, g)
			}
		}
	}
	return resultTags
}

// ---------------------------------------------------------------------------------------------------------------
// Категория падежа
// ---------------------------------------------------------------------------------------------------------------
// Получить тэг падежа. Если не найдено, возвращает nil
func (x WordForm) getCaseG() *Category {
	caseTags := x.GetTagsFromSet(categories.CaseStrings)
	if len(caseTags) == 1 {
		return caseTags[0]
	} else if len(caseTags) > 1 {
		panic(WrapOpenCorporaErrorf(nil, "WordForm: Multiple case tags: %v", x))
	}
	return nil
}

// Получить падеж
func (x WordForm) GetCase() (categories.Case, error) {
	if caseTag := x.getCaseG(); caseTag != nil {
		if c, ok := categories.MapStringToCase[string(caseTag.VAttr)]; ok {
			return c, nil
		}
	}
	return "", WrapOpenCorporaErrorf(nil, "WordForm: %s has no case", x)
}

// Установить падеж
func (x *WordForm) SetCase(c categories.Case) {
	if caseTag := x.getCaseG(); caseTag != nil {
		caseTag.VAttr = grammeme.Name(c)
	} else {
		x.G = append(x.G, &Category{VAttr: grammeme.Name(c)})
	}
}

// ---------------------------------------------------------------------------------------------------------------
// Числа
// ---------------------------------------------------------------------------------------------------------------
// Получить тэг числа. Если не найдено, возвращает nil
func (x WordForm) getNumberG() *Category {
	caseTags := x.GetTagsFromSet(categories.NumberStrings)
	if len(caseTags) == 1 {
		return caseTags[0]
	} else if len(caseTags) > 1 {
		panic(WrapOpenCorporaErrorf(nil, "WordForm: multiple number tags: %v", x))
	}
	return nil
}

// Получить число
func (x WordForm) GetNumber() (categories.Number, error) {
	if caseTag := x.getNumberG(); caseTag != nil {
		if c, ok := categories.MapStringToNumber[string(caseTag.VAttr)]; ok {
			return c, nil
		}
	}
	return "", WrapOpenCorporaErrorf(nil, "WordForm: %s has no case", x)
}

// Установить число
func (x *WordForm) SetNumber(c categories.Number) {
	if caseTag := x.getNumberG(); caseTag != nil {
		caseTag.VAttr = grammeme.Name(c)
	} else {
		x.G = append(x.G, &Category{VAttr: grammeme.Name(c)})
	}
}

func (x *WordForm) Word(index *grammeme.Index) (*words.Word, error) {
	word := words.NewWord(index, x.Form)
	for _, category := range x.G {
		if grammeme, err := index.ByName(category.VAttr); err != nil {
			return nil, WrapOpenCorporaErrorf(err, "cant find grammeme `%v` in index", category.VAttr)
		} else if err = word.Grammemes().Add(grammeme); err != nil {
			return nil, WrapOpenCorporaErrorf(err, "cant add grammeme `%v` to word", category.VAttr)
		}
	}
	return word, nil
}
