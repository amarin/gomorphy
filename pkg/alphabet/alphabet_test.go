package alphabet

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorage_Length(t *testing.T) {
	t.Run("алфавит uint8", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint8]()
		t.Run("по умолчанию длина 0", func(t *testing.T) {
			require.Equal(t, 0, alphabet.Length())
		})
		t.Run("добавление 1 символа увеличивает длину на 1", func(t *testing.T) {
			newIdx, err := alphabet.Add('a')
			require.NoError(t, err)
			require.Equal(t, uint8(0), newIdx)
			require.Equal(t, 1, alphabet.Length())
		})
		t.Run("добавление 2 символов увеличивает длину на 2", func(t *testing.T) {
			newIdx, err := alphabet.Add('b')
			require.NoError(t, err)
			require.Equal(t, uint8(1), newIdx)
			newIdx, err = alphabet.Add('c')
			require.NoError(t, err)
			require.Equal(t, uint8(2), newIdx)
			require.Equal(t, 3, alphabet.Length())
		})
	})
	t.Run("алфавит uint16", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint16]()
		t.Run("по умолчанию длина 0", func(t *testing.T) {
			require.Equal(t, 0, alphabet.Length())
		})
		t.Run("добавление 1 символа увеличивает длину на 1", func(t *testing.T) {
			newIdx, err := alphabet.Add('a')
			require.NoError(t, err)
			require.Equal(t, uint16(0), newIdx)
			require.Equal(t, 1, alphabet.Length())
		})
		t.Run("добавление 2 символов увеличивает длину на 2", func(t *testing.T) {
			newIdx, err := alphabet.Add('b')
			require.NoError(t, err)
			require.Equal(t, uint16(1), newIdx)
			newIdx, err = alphabet.Add('c')
			require.NoError(t, err)
			require.Equal(t, uint16(2), newIdx)
			require.Equal(t, 3, alphabet.Length())
		})
	})
}

func TestStorage_Add(t *testing.T) {
	t.Run("алфавит uint8", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint8]()
		t.Run("по умолчанию нет символа", func(t *testing.T) {
			require.False(t, alphabet.Has('a'))
		})
		t.Run("есть символ после добавления", func(t *testing.T) {
			newIdx, err := alphabet.Add('a')
			require.NoError(t, err)
			require.Equal(t, uint8(0), newIdx)
			require.True(t, alphabet.Has('a'))
		})
		t.Run("добавление существующего не изменяет алфавит и не возвращает ошибок", func(t *testing.T) {
			lenBeforeAdd := alphabet.Length()
			newIdx, err := alphabet.Add('a')
			require.NoError(t, err)
			require.Equal(t, uint8(lenBeforeAdd-1), newIdx)
			require.Equal(t, lenBeforeAdd, alphabet.Length())
		})
	})
	t.Run("алфавит uint16", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint16]()
		t.Run("по умолчанию нет символа", func(t *testing.T) {
			require.False(t, alphabet.Has('a'))
		})
		t.Run("есть символ после добавления", func(t *testing.T) {
			newIdx, err := alphabet.Add('a')
			require.NoError(t, err)
			require.Equal(t, uint16(0), newIdx)
			require.True(t, alphabet.Has('a'))
		})
		t.Run("добавление существующего не изменяет алфавит и не возвращает ошибок", func(t *testing.T) {
			lenBeforeAdd := alphabet.Length()
			newIdx, err := alphabet.Add('a')
			require.NoError(t, err)
			require.Equal(t, uint16(lenBeforeAdd-1), newIdx)
			require.Equal(t, lenBeforeAdd, alphabet.Length())
		})
	})
}

func TestStorage_GetOrCreate(t *testing.T) {
	t.Run("алфавит uint8", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint8]()
		t.Run("ошибка при запросе индекса за пределами алфавите", func(t *testing.T) {
			_, err := alphabet.GetByIdx(1)
			require.ErrorIs(t, err, ErrOverflow)
		})
		t.Run("добавление если не существует", func(t *testing.T) {
			character := 'a'
			idx, err := alphabet.GetOrCreate(character)
			require.NoError(t, err)
			require.True(t, alphabet.Has(character))
			require.Equal(t, character, alphabet.MustGetByIdx(idx))
		})
		t.Run("получение существующего символа без добавления", func(t *testing.T) {
			character := 'a'
			require.True(t, alphabet.Has(character))
			currentLen := alphabet.Length()
			_, err := alphabet.GetOrCreate(character)
			require.NoError(t, err)
			require.True(t, alphabet.Has(character))
			require.Equal(t, currentLen, alphabet.Length())
		})
	})
	t.Run("алфавит uint16", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint16]()
		t.Run("ошибка при запросе индекса за пределами алфавите", func(t *testing.T) {
			_, err := alphabet.GetByIdx(1)
			require.ErrorIs(t, err, ErrOverflow)
		})
		t.Run("добавление если не существует", func(t *testing.T) {
			character := 'a'
			idx, err := alphabet.GetOrCreate(character)
			require.NoError(t, err)
			require.True(t, alphabet.Has(character))
			require.Equal(t, character, alphabet.MustGetByIdx(idx))
		})
		t.Run("получение существующего символа без добавления", func(t *testing.T) {
			character := 'a'
			require.True(t, alphabet.Has(character))
			currentLen := alphabet.Length()
			_, err := alphabet.GetOrCreate(character)
			require.NoError(t, err)
			require.True(t, alphabet.Has(character))
			require.Equal(t, currentLen, alphabet.Length())
		})
	})
}

func TestStorage_Get(t *testing.T) {
	t.Run("uint8", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint8]()
		t.Run("ошибка при запросе индекса несуществующего символа", func(t *testing.T) {
			_, err := alphabet.Get('a')
			require.ErrorIs(t, err, ErrNoRuneInAlphabet)
		})
		t.Run("получение индекса существующего символа", func(t *testing.T) {
			addIdx, err := alphabet.Add('a')
			require.NoError(t, err)
			require.Equal(t, uint8(0), addIdx)
			getIdx, err := alphabet.Get('a')
			require.NoError(t, err)
			require.Equal(t, addIdx, getIdx)

			require.Equal(t, 'a', alphabet.MustGetByIdx(addIdx))
			require.True(t, alphabet.Has('a'))
		})
		t.Run("получение индекса дополнительного символа", func(t *testing.T) {
			addIdx, err := alphabet.Add('b')
			require.NoError(t, err)
			require.Equal(t, uint8(1), addIdx)
			getIdx, err := alphabet.Get('b')
			require.NoError(t, err)
			require.Equal(t, addIdx, getIdx)

			require.Equal(t, 'b', alphabet.MustGetByIdx(addIdx))
			require.True(t, alphabet.Has('b'))
		})
	})
	t.Run("uint16", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint16]()
		t.Run("ошибка при запросе индекса несуществующего символа", func(t *testing.T) {
			_, err := alphabet.Get('a')
			require.ErrorIs(t, err, ErrNoRuneInAlphabet)
		})
		t.Run("получение индекса существующего символа", func(t *testing.T) {
			addIdx, err := alphabet.Add('a')
			require.NoError(t, err)
			require.Equal(t, uint16(0), addIdx)
			getIdx, err := alphabet.Get('a')
			require.NoError(t, err)
			require.Equal(t, addIdx, getIdx)

			require.Equal(t, 'a', alphabet.MustGetByIdx(addIdx))
			require.True(t, alphabet.Has('a'))
		})
		t.Run("получение индекса дополнительного символа", func(t *testing.T) {
			addIdx, err := alphabet.Add('b')
			require.NoError(t, err)
			require.Equal(t, uint16(1), addIdx)
			getIdx, err := alphabet.Get('b')
			require.NoError(t, err)
			require.Equal(t, addIdx, getIdx)

			require.Equal(t, 'b', alphabet.MustGetByIdx(addIdx))
			require.True(t, alphabet.Has('b'))
		})
	})
}

func TestStorage_MustGetByIdx(t *testing.T) {
	t.Run("uint8", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint8]()
		addIdx, err := alphabet.Add('a')
		require.NoError(t, err)
		require.Equal(t, uint8(0), addIdx)
		t.Run("успешное получение символа по индексу", func(t *testing.T) {
			require.Equal(t, 'a', alphabet.MustGetByIdx(0))
		})
		t.Run("паника при запросе индекса за пределами хранилища", func(t *testing.T) {
			require.Panics(t, func() { alphabet.MustGetByIdx(1) })
		})
	})
	t.Run("uint16", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint16]()
		addIdx, err := alphabet.Add('a')
		require.NoError(t, err)
		require.Equal(t, uint16(0), addIdx)
		t.Run("успешное получение символа по индексу", func(t *testing.T) {
			require.Equal(t, 'a', alphabet.MustGetByIdx(0))
		})
		t.Run("паника при запросе индекса за пределами хранилища", func(t *testing.T) {
			require.Panics(t, func() { alphabet.MustGetByIdx(1) })
		})
	})
}

func TestStorage_Alphabet(t *testing.T) {
	t.Run("uint8", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint8]()
		t.Run("пустой алфавит возвращает пустую строку", func(t *testing.T) {
			require.Equal(t, "", alphabet.String())
		})
		t.Run("строка алфавита", func(t *testing.T) {
			alphabet.MustAdd('a')
			alphabet.MustAdd('b')
			alphabet.MustAdd('c')
			alphabet.MustAdd('d')
			require.Equal(t, "abcd", alphabet.String())
		})
	})
	t.Run("uint16", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint16]()
		t.Run("пустой алфавит возвращает пустую строку", func(t *testing.T) {
			require.Equal(t, "", alphabet.String())
		})
		t.Run("строка алфавита", func(t *testing.T) {
			alphabet.MustAdd('a')
			alphabet.MustAdd('b')
			alphabet.MustAdd('c')
			alphabet.MustAdd('d')
			require.Equal(t, "abcd", alphabet.String())
		})
	})
}

func TestStorage_Reset(t *testing.T) {
	t.Run("uint8", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint8]()
		alphabet.MustAdd('F')

		resetWith := "abcdefghijklmnopqrstuvwxyz"

		t.Run("загрузка нового алфавита", func(t *testing.T) {
			require.NoError(t, alphabet.Reset(resetWith))
			require.Equal(t, resetWith, alphabet.String())
			require.False(t, alphabet.Has('F'))
			require.True(t, alphabet.Has('f'))
		})
		t.Run("ошибка при загрузке алфавита с повторами", func(t *testing.T) {
			runeSet := []rune(resetWith)
			for i := 0; i < 100; i++ {
				randomCharIdxToRepeat := rand.Intn(len(runeSet) - 1)
				charToRepeat := string(runeSet[randomCharIdxToRepeat])
				alphabetWithRepeats := resetWith + charToRepeat
				require.Errorf(t, alphabet.Reset(alphabetWithRepeats),
					"ожидалась ошибка при повторе '%s' в алфавите `%s`",
					charToRepeat, alphabetWithRepeats,
				)
			}
		})
	})
	t.Run("uint16", func(t *testing.T) {
		t.Parallel()
		alphabet := New[uint16]()
		alphabet.MustAdd('F')

		resetWith := "abcdefghijklmnopqrstuvwxyz"

		t.Run("загрузка нового алфавита", func(t *testing.T) {
			require.NoError(t, alphabet.Reset(resetWith))
			require.Equal(t, resetWith, alphabet.String())
			require.False(t, alphabet.Has('F'))
			require.True(t, alphabet.Has('f'))
		})
		t.Run("ошибка при загрузке алфавита с повторами", func(t *testing.T) {
			runeSet := []rune(resetWith)
			for i := 0; i < 100; i++ {
				randomCharIdxToRepeat := rand.Intn(len(runeSet) - 1)
				charToRepeat := string(runeSet[randomCharIdxToRepeat])
				alphabetWithRepeats := resetWith + charToRepeat
				require.Errorf(t, alphabet.Reset(alphabetWithRepeats),
					"ожидалась ошибка при повторе '%s' в алфавите `%s`",
					charToRepeat, alphabetWithRepeats,
				)
			}
		})
	})
}
