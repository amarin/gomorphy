package opencorpora

import (
	"github.com/amarin/binutils"
)

// Список словоформ
type WordFormList []*WordForm

func (w WordFormList) MarshalBinary() (data []byte, err error) {
	buffer := binutils.NewEmptyBuffer()
	for _, form := range w {
		if _, err = buffer.WriteObject(form); err != nil {
			break
		}
	}
	if err != nil {
		err = NewOpenCorporaError("WordForm", err)
	}
	return buffer.Bytes(), err
}

func (w *WordFormList) UnmarshalFromBuffer(buffer *binutils.Buffer) error {
	for {
		if buffer.Len() > 0 {
			form := new(WordForm)
			if err := form.UnmarshalFromBuffer(buffer); err != nil {
				return NewOpenCorporaError("WordForm", err)
			}
			*w = append(*w, form)
		} else {
			break
		}
	}
	return nil
}

func (w *WordFormList) UnmarshalBinary(data []byte) error {
	return w.UnmarshalFromBuffer(binutils.NewBuffer(data))
}
