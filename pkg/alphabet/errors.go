package alphabet

import "errors"

var (
	// ErrNoRuneInAlphabet сигнализирует об ошибке поиска символа в индексе.
	ErrNoRuneInAlphabet = errors.New("no rune in alphabet")

	// ErrNoRuneAtIndex сигнализирует об ошибке поиска символа по индексу
	ErrNoRuneAtIndex = errors.New("no rune at index")

	// ErrRepeatedCharacters сигнализирует о повторяющихся символах в алфавите
	ErrRepeatedCharacters = errors.New("repeated characters")

	// ErrOverflow сигнализирует о переполнении алфавита
	// (например, при попытке добавления символа за пределами ёмкости)
	ErrOverflow = errors.New("overflow")
)
