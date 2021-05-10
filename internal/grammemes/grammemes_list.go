package grammemes

import (
	"fmt"
	"sort"
	"strings"

	"github.com/amarin/binutils"
	"github.com/amarin/gomorphy/internal/grammeme"
	"github.com/amarin/gomorphy/pkg/common"
)

// List хранит список граммем, определённых в индексе.
type List struct {
	index     *grammeme.Index
	grammemes []*grammeme.Grammeme
}

// String возвращает строковое представление списка граммем.
func (g *List) String() string {
	res := make([]string, 0)
	for _, grammeme := range g.grammemes {
		res = append(res, string(grammeme.Name))
	}

	return strings.Join(res, ",")
}

// GrammemeIndex возвращает указатель на использованный индекс.
func (g *List) GrammemeIndex() *grammeme.Index {
	return g.index
}

// NewList создаёт новый список граммем для заданного индекса.
func NewList(index *grammeme.Index, grammemes ...*grammeme.Grammeme) *List {
	return &List{index: index, grammemes: grammemes}
}

// Len возвращает список граммем.
func (g List) Len() int {
	return len(g.grammemes)
}

// Add добавляет граммему в список.
// Возвращает ошибку, если граммема уже в списке.
func (g *List) Add(grammeme *grammeme.Grammeme) error {
	for _, existedGrammeme := range g.grammemes {
		if grammeme.Name == existedGrammeme.Name {
			return NewErrorf("grammeme `%v` already in set", existedGrammeme.Name)
		}
	}

	g.grammemes = append(g.grammemes, grammeme)

	return nil
}

// Slice возвращает список граммем.
func (g List) Slice() []*grammeme.Grammeme {
	return g.grammemes
}

// EqualTo сравнивает список граммем с другим списком.
// Возвращает true если списки граммем используют одинаковый индекс и содержат одинаковый набор граммем,
// независимо от порядка в списке.
func (g *List) EqualTo(another *List) bool {
	// different if indexes differs
	if g.index != another.index {
		return false
	}
	// make grammemes id lists to compare
	var thisGrammemes, anotherGrammemes []byte

	for _, grammeme := range g.grammemes {
		if idx, err := g.index.Idx(grammeme.Name); err != nil {
			panic(NewErrorf("index wrong, grammeme `%v` id not found", grammeme.Name))
		} else {
			thisGrammemes = append(thisGrammemes, idx)
		}
	}

	for _, grammeme := range another.grammemes {
		if idx, err := g.index.Idx(grammeme.Name); err != nil {
			panic(NewErrorf("index wrong, grammeme `%v` id not found", grammeme.Name))
		} else {
			anotherGrammemes = append(anotherGrammemes, idx)
		}
	}
	// different if grammemes count differs
	if len(thisGrammemes) != len(anotherGrammemes) {
		return false
	}
	// there both slices are similar length. Sort slices and compare byte by byte
	sort.Slice(thisGrammemes, func(i int, j int) bool { return thisGrammemes[i] < thisGrammemes[j] })
	sort.Slice(anotherGrammemes, func(i int, j int) bool { return anotherGrammemes[i] < anotherGrammemes[j] })

	for idx := 0; idx < len(thisGrammemes); idx++ {
		if thisGrammemes[idx] != anotherGrammemes[idx] {
			return false
		}
	}

	return true
}

// MarshalBinary сохраняет данные списка граммем в двоичном виде.
// При сохранении в двоичном виде список граммем записывает только идентификаторы граммем в индексе.
// Первый байт записи всегда размер списка.
func (g List) MarshalBinary() (data []byte, err error) {
	buf := binutils.NewEmptyBuffer()

	// write grammeme list len one. One byte enough
	if _, err = buf.WriteUint8(uint8(len(g.grammemes))); err != nil {
		return buf.Bytes(), fmt.Errorf("%w: cant write length byte: %v", common.ErrMarshal, err)
	}

	for _, gInstance := range g.grammemes {
		if grammemeIdx, err := g.index.Idx(gInstance.Name); err != nil {
			return []byte{}, fmt.Errorf("%w: cant detect idx for %v", common.ErrMarshal, gInstance.Name)
		} else if _, err = buf.WriteUint8(grammemeIdx); err != nil {
			return []byte{}, fmt.Errorf("%w: cant write grammeme %v idx", common.ErrMarshal, gInstance.Name)
		}
	}

	return buf.Bytes(), nil
}

// UnmarshalFromBuffer загружает данные списка граммем из бинарного буфера.
// При загрузке бинарных данных список граммем получает размер списка и идентификаторы граммем в индексе.
// Первый байт записи всегда размер списка, затем следуют байты идентификаторов граммем.
// При загрузке списка из буфера список граммем вычитывает необходимое количество байт, оставляя лишние данные в буфере.
func (g *List) UnmarshalFromBuffer(buffer *binutils.Buffer) error {
	if buffer.Len() < binutils.Uint8size {
		return fmt.Errorf("%w: Expected at least %d byte", common.ErrUnmarshal, binutils.Uint8size)
	}

	var listLen, grammemeIdx uint8

	if err := buffer.ReadUint8(&listLen); err != nil {
		return fmt.Errorf("%w: cant read list size", common.ErrUnmarshal)
	}

	for idx := 0; uint8(idx) < listLen; idx++ {
		if err := buffer.ReadUint8(&grammemeIdx); err != nil {
			return fmt.Errorf("%w: cant read grammeme %v idx: %v", common.ErrUnmarshal, idx, err)
		} else if grammeme, err := g.index.ByIdx(grammemeIdx); err != nil {
			return fmt.Errorf("%w: cant take grammeme %v from dict: %v", common.ErrUnmarshal, grammemeIdx, err)
		} else {
			g.grammemes = append(g.grammemes, grammeme)
		}
	}

	return nil
}
