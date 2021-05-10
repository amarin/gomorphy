package words

// Word object stores together word text and its grammemes.

import (
	"fmt"

	"github.com/amarin/binutils"
	"github.com/amarin/gomorphy/internal/grammeme"

	"github.com/amarin/gomorphy/internal/grammemes"
	"github.com/amarin/gomorphy/internal/text"
	"github.com/amarin/gomorphy/pkg/common"
)

// Word object stores together word text and its grammemes.
type Word struct {
	text      text.RussianText
	grammemes *grammemes.List
}

// Text returns word text.
func (e *Word) Text() text.RussianText {
	return e.text
}

// GrammemesIndex returns used grammemes index.
// Implements GrammemeIndexer.
func (e Word) GrammemesIndex() *grammeme.Index {
	return e.grammemes.GrammemeIndex()
}

// Grammemes returns pointer to list of word grammemes.
func (e Word) Grammemes() *grammemes.List {
	return e.grammemes
}

// NewWord creates new instance of Word.
// Takes grammemes index, word text and variable length list of grammemes to append to instance.
func NewWord(index *grammeme.Index, wordText text.RussianText, grammemes ...*grammeme.Grammeme) *Word {
	return &Word{text: wordText, grammemes: index.NewList(grammemes...)}
}

// MarshalBinary creates new binary word representation joined together with its grammemes.
// Stores grammemes as indexes of used grammemes index to save used place.
// Used to store words as bytes sequences.
func (e Word) MarshalBinary() (data []byte, err error) {
	buffer := binutils.NewEmptyBuffer()
	if _, err = buffer.WriteObject(e.text); err != nil {
		return buffer.Bytes(), fmt.Errorf("%w: cant marshal text: %v", common.ErrUnmarshal, err)
	} else if _, err = buffer.WriteObject(e.grammemes); err != nil {
		return buffer.Bytes(), fmt.Errorf("%w: cant marshal grammemes: %v", common.ErrUnmarshal, err)
	}

	return buffer.Bytes(), err
}

// UnmarshalFromBuffer takes required amount of bytes from buffer to restore word data.
// Используется при загрузке бинарных словарей..
func (e *Word) UnmarshalFromBuffer(buffer *binutils.Buffer) (err error) {
	if err = buffer.ReadObject(&e.text); err != nil {
		return fmt.Errorf("%w: cant unmarshal text: %v", common.ErrUnmarshal, err)
	} else if err = buffer.ReadObject(e.grammemes); err != nil {
		return fmt.Errorf("%w: cant unmarshal grammemes: %v", common.ErrUnmarshal, err)
	}

	return err
}

// EqualsTo compares current word with another one.
// Returns true if both words have same texts and grammemes sets (not depending of grammemes order).
// Returns false if either word texts differs or grammemes sets contains different grammemes.
// Also returns false if grammemes indexes differs even if grammemes sets equials.
func (e Word) EqualsTo(another *Word) bool {
	if e.text != another.text {
		return false
	} else if !e.grammemes.EqualTo(another.grammemes) {
		return false
	}

	return true
}
