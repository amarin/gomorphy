package alphabet

import (
	"math"
	"sync"

	"github.com/amarin/gomorphy/pkg/size"
)

// Alphabet реализует компактное хранение используемого алфавита.
type Alphabet[A size.Alphabet] struct {
	maxIdx A // Максимальный индекс символа

	mu         *sync.RWMutex // Защита параллельного доступа для атрибутов ниже
	characters []rune        // Все символы алфавита и их порядок
	index      map[rune]A    // Индексы символов
}

// New создаёт новое хранилище алфавита с типизированным интерфейсом.
// Принимает и возвращает индексы с типом, соответствующим размеру хранилища.
func New[A size.Alphabet]() *Alphabet[A] {
	var (
		resolveIdx any = A(0)
		maxIdx     A
	)

	switch resolveIdx.(type) {
	case uint8:
		maxIdx = A(math.MaxUint8)
	case uint16:
		maxIdx = A(math.MaxUint8)
	}

	return &Alphabet[A]{
		mu:         new(sync.RWMutex),
		characters: []rune{},
		index:      map[rune]A{},
		maxIdx:     maxIdx,
	}
}

// Length возвращает длину алфавита в символах.
func (s *Alphabet[A]) Length() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.characters)
}

// Add добавляет символ в алфавит, если такого символа ещё нет.
// Возвращает индекс добавленного или существующего символа.
func (s *Alphabet[A]) Add(char rune) (A, error) {
	s.mu.RLock()
	existedIndex, alreadyExists := s.index[char]
	s.mu.RUnlock()

	if alreadyExists {
		return existedIndex, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	nextIdx := len(s.characters)
	if nextIdx > int(s.maxIdx) { // нет места для нового символа
		return 0, ErrOverflow
	}

	s.characters = append(s.characters, char)
	charIndex := A(nextIdx)
	s.index[char] = charIndex

	return charIndex, nil
}

// MustAdd добавляет символ в алфавит, если такого символа ещё нет.
// Возвращает индекс добавленного или существующего символа.
// Паникует при переполнении ёмкости алфавита
func (s *Alphabet[A]) MustAdd(char rune) A {
	index, err := s.Add(char)
	if err != nil {
		panic(err)
	}
	return index
}

// Has возвращает true если символ есть в алфавите
func (s *Alphabet[A]) Has(char rune) bool {
	s.mu.RLock()
	_, exists := s.index[char]
	s.mu.RUnlock()

	return exists
}

// MustGetByIdx получает символ по заданному индексу в алфавите.
// Паникует если символ не найден.
func (s *Alphabet[A]) MustGetByIdx(idx A) rune {
	character, err := s.GetByIdx(idx)
	if err != nil {
		panic(err)
	}

	return character
}

// GetByIdx получает символ по заданному индексу в алфавите.
// Возвращает ошибку если символ по индексу не найден.
func (s *Alphabet[A]) GetByIdx(idx A) (rune, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idxLen := len(s.characters)
	if idx < 0 || int(idx) >= idxLen {
		return 0, ErrOverflow
	}

	return s.characters[idx], nil
}

// GetOrCreate получает индекс символа в алфавите.
// Добавляет символ, если его не было в индексе.
// Возвращает ошибку, если требуется добавить символ, а алфавит уже заполнен.
func (s *Alphabet[A]) GetOrCreate(char rune) (A, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if idx, alreadyExists := s.index[char]; alreadyExists {
		return idx, nil
	}

	nextIdx := len(s.characters)
	if nextIdx > int(s.maxIdx) { // нет места для нового символа
		return 0, ErrOverflow
	}

	s.characters = append(s.characters, char)
	s.index[char] = A(nextIdx)

	return A(nextIdx), nil
}

// Get получает индекс символа в алфавите.
// Возвращает ошибку, если символа нет в алфавите.
func (s *Alphabet[A]) Get(char rune) (A, error) {
	s.mu.RLock()
	idx, alreadyExists := s.index[char]
	s.mu.RUnlock()

	if alreadyExists {
		return idx, nil
	}

	return 0, ErrNoRuneInAlphabet
}

// String возвращает алфавит одной строкой.
func (s *Alphabet[A]) String() string {
	s.mu.RLock()
	characters := s.characters
	s.mu.RUnlock()

	return string(characters)
}

// Reset заполняет алфавит из заданной строки. Удаляет любые существовавшие символы.
// Индексы символов в алфавите будут соответствовать индексам символа в строке.
// Символы в строке не должны повторяться.
func (s *Alphabet[A]) Reset(alphabetString string) error {
	newChars := []rune(alphabetString)
	newIndex := map[rune]A{}

	for idx, char := range newChars {
		if idx > int(s.maxIdx) {
			return ErrOverflow
		}
		newIndex[char] = A(idx)
	}

	if len(newChars) != len(newIndex) {
		return ErrRepeatedCharacters
	}

	s.mu.Lock()
	s.characters = newChars
	s.index = newIndex
	s.mu.Unlock()

	return nil
}
