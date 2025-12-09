package indexer

import (
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type fake rune

func (f *fake) ReadFrom(r io.Reader) (n int64, err error) {
	var target rune
	if err = binary.Read(r, binary.BigEndian, &target); err != nil {
		return 0, err
	}

	*f = fake(target)

	return 4, nil
}

func (f *fake) WriteTo(w io.Writer) (n int64, err error) {
	if err = binary.Write(w, binary.BigEndian, rune(*f)); err != nil {
		return 0, err
	}

	return 4, nil
}

func newFake() *fake {
	return (*fake)(new(rune))
}

func TestNew(t *testing.T) {
	t.Run("конструктор заполняет имя", func(t *testing.T) {
		i := New[uint8, fake]("any")
		require.Equal(t, "any", i.name)
	})
	t.Run("конструктор заполняет индекс", func(t *testing.T) {
		i := New[uint8, fake]("any")
		require.NotNil(t, i.idx)
	})
	t.Run("конструктор заполняет семафор", func(t *testing.T) {
		i := New[uint8, fake]("any")
		require.NotNil(t, i.mu)
	})
}

func TestIndexOf_Add_Get(t *testing.T) {
	i := New[uint8, fake]("any")
	t.Run("1й элемент", func(t *testing.T) {
		initialElem := fake('a')
		idx, err := i.Add(initialElem)
		require.NoError(t, err)
		require.Equal(t, uint8(0), idx)
		require.Equal(t, 1, i.Len())
		elemPtr, err := i.Get(idx)
		require.NoError(t, err)
		require.NotNil(t, elemPtr)
		require.Equal(t, initialElem, *elemPtr)
	})

	t.Run("2й элемент", func(t *testing.T) {
		initialElem := fake('b')
		idx, err := i.Add(initialElem)
		require.NoError(t, err)
		require.Equal(t, uint8(1), idx)
		require.Equal(t, 2, i.Len())
		elemPtr, err := i.Get(idx)
		require.NoError(t, err)
		require.NotNil(t, elemPtr)
		require.Equal(t, initialElem, *elemPtr)
	})
}
