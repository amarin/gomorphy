package alphabet

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorage_Length(t *testing.T) {
	alphabet := New()
	t.Run("по умолчанию длина 0", func(t *testing.T) {
		require.Equal(t, 0, alphabet.Length())
	})
	t.Run("добавление 1 символа увеличивает длину на 1", func(t *testing.T) {
		alphabet.Add('a')
		require.Equal(t, 1, alphabet.Length())
	})
	t.Run("добавление 2 символов увеличивает длину на 2", func(t *testing.T) {
		alphabet.Add('b')
		alphabet.Add('c')
		require.Equal(t, 3, alphabet.Length())
	})
}

func TestStorage_Add(t *testing.T) {
	alphabet := New()
	t.Run("по умолчанию нет символа", func(t *testing.T) {
		require.False(t, alphabet.Has('a'))
	})
	t.Run("есть символ после добавления", func(t *testing.T) {
		alphabet.Add('a')
		require.True(t, alphabet.Has('a'))
	})
	t.Run("добавление существующего не изменяет алфавит", func(t *testing.T) {
		lenBeforeAdd := alphabet.Length()
		alphabet.Add('a')
		require.Equal(t, lenBeforeAdd, alphabet.Length())
	})
}

func TestStorage_GetOrCreate(t *testing.T) {
	alphabet := New()
	t.Run("ошибка при запросе отрицательного индекса", func(t *testing.T) {
		_, err := alphabet.GetByIdx(-1)
		require.ErrorIs(t, err, ErrNoRuneAtIndex)
	})
	t.Run("ошибка при запросе индекса за пределами алфавите", func(t *testing.T) {
		_, err := alphabet.GetByIdx(1)
		require.ErrorIs(t, err, ErrNoRuneAtIndex)
	})
	t.Run("добавление если не существует", func(t *testing.T) {
		character := 'a'
		idx := alphabet.GetOrCreate(character)
		require.True(t, alphabet.Has(character))
		require.Equal(t, character, alphabet.MustGetByIdx(idx))
	})
	t.Run("получение существующего символа без добавления", func(t *testing.T) {
		character := 'a'
		require.True(t, alphabet.Has(character))
		currentLen := alphabet.Length()
		_ = alphabet.GetOrCreate(character)
		require.True(t, alphabet.Has(character))
		require.Equal(t, currentLen, alphabet.Length())
	})
}

func TestStorage_Get(t *testing.T) {
	alphabet := New()
	t.Run("ошибка при запросе индекса несуществующего символа", func(t *testing.T) {
		_, err := alphabet.Get('a')
		require.ErrorIs(t, err, ErrNoRuneInAlphabet)
	})
	t.Run("получение индекса существующего символа", func(t *testing.T) {
		alphabet.Add('a')
		idx, err := alphabet.Get('a')
		require.NoError(t, err)
		require.Equal(t, 'a', alphabet.MustGetByIdx(idx))
		require.True(t, alphabet.Has('a'))
		require.Equal(t, 0, idx)
	})
}

func TestStorage_MustGetByIdx(t *testing.T) {
	alphabet := New()
	alphabet.Add('a')
	t.Run("успешное получение символа по индексу", func(t *testing.T) {
		require.Equal(t, 'a', alphabet.MustGetByIdx(0))
	})
	t.Run("паника при запросе отрицательного индекса", func(t *testing.T) {
		require.Panics(t, func() { alphabet.MustGetByIdx(-1) })
	})
	t.Run("паника при запросе индекса за пределами хранилища", func(t *testing.T) {
		require.Panics(t, func() { alphabet.MustGetByIdx(1) })
	})
}

func TestStorage_Alphabet(t *testing.T) {
	alphabet := New()
	t.Run("пустой алфавит возвращает пустую строку", func(t *testing.T) {
		require.Equal(t, "", alphabet.String())
	})
	t.Run("строка алфавита", func(t *testing.T) {
		alphabet.Add('a')
		alphabet.Add('b')
		alphabet.Add('c')
		alphabet.Add('d')
		require.Equal(t, "abcd", alphabet.String())
	})
}

func TestStorage_Reset(t *testing.T) {
	alphabet := New()

	fullEnglishAlpjhabet := "abcdefghijklmnopqrstuvwxyz"
	alphabet.Add('F')
	t.Run("загрузка нового алфавита", func(t *testing.T) {
		require.NoError(t, alphabet.Reset(fullEnglishAlpjhabet))
		require.Equal(t, fullEnglishAlpjhabet, alphabet.String())
		require.False(t, alphabet.Has('F'))
		require.True(t, alphabet.Has('f'))
	})
	t.Run("ошибка при загрузке алфавита с повторами", func(t *testing.T) {
		runeSet := []rune(fullEnglishAlpjhabet)
		for i := 0; i < 100; i++ {
			randomCharIdxToRepeat := rand.Intn(len(runeSet) - 1)
			charToRepeat := string(runeSet[randomCharIdxToRepeat])
			alphabetWithRepeats := fullEnglishAlpjhabet + charToRepeat
			require.Errorf(t, alphabet.Reset(alphabetWithRepeats),
				"ожидалась ошибка при повторе '%s' в алфавите `%s`",
				charToRepeat, alphabetWithRepeats,
			)
		}
	})
}
