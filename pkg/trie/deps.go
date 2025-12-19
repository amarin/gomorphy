//go:generate mockgen -source=${GOFILE} -destination=mocks_test.go -package=${GOPACKAGE}
package trie

import (
	"io"

	"github.com/amarin/gomorphy/pkg/size"
)

type alphabetInterface[A size.Alphabet] interface {
	io.ReaderFrom
	io.WriterTo

	// GetOrCreate возвращает индекс символа в алфавите;
	// Добавляет символ в алфавит при необходимости.
	// Возвращает ошибку, если символа не было в алфавите и не удалось добавить
	GetOrCreate(char rune) (A, error)

	// Get возвращает индекс символа в алфавите или ошибку, если символ в алфавите не найден.
	Get(char rune) (A, error)

	String() string
}
