package alphabet

import (
	"errors"
	"sync"
)

var (
	// ErrNoRuneInAlphabet сигнализирует об ошибке поиска символа в индексе.
	ErrNoRuneInAlphabet = errors.New("no rune in alphabet")

	// ErrNoRuneAtIndex сигнализирует об ошибке поиска символа по индексу
	ErrNoRuneAtIndex = errors.New("no rune at index")

	// ErrRepeatedCharacters сигнализирует о повторяющихся символах в алфавите
	ErrRepeatedCharacters = errors.New("repeated characters")
)

// Storage реализует компактное хранение используемого алфавита.
type Storage struct {
	mutex      *sync.RWMutex
	characters []rune
	index      map[rune]int
}

// New создаёт новое хранилище алфавита.
func New() *Storage {
	return &Storage{
		mutex:      new(sync.RWMutex),
		characters: []rune{},
		index:      map[rune]int{},
	}
}

// Length возвращает длину алфавита в символах.
func (s *Storage) Length() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return len(s.characters)
}

// Add добавляет символ в алфавит, если такого символа ещё нет.
func (s *Storage) Add(char rune) {
	s.mutex.RLock()
	_, alreadyExists := s.index[char]
	s.mutex.RUnlock()

	if alreadyExists {
		return
	}

	s.mutex.Lock()
	s.characters = append(s.characters, char)
	charIndex := len(s.characters) - 1
	s.index[char] = charIndex
	s.mutex.Unlock()
}

// Has возвращает true если символ есть в алфавите
func (s *Storage) Has(char rune) bool {
	s.mutex.RLock()
	_, exists := s.index[char]
	s.mutex.RUnlock()

	return exists
}

// MustGetByIdx получает символ по заданному индексу в алфавите.
// Паникует если символ не найден.
func (s *Storage) MustGetByIdx(idx int) rune {
	character, err := s.GetByIdx(idx)
	if err != nil {
		panic(err)
	}

	return character
}

// GetByIdx получает символ по заданному индексу в алфавите.
// Возвращает ошибку если символ по индексу не найден.
func (s *Storage) GetByIdx(idx int) (rune, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	idxLen := len(s.characters)

	if idx < 0 || idx >= idxLen {
		return 0, ErrNoRuneAtIndex
	}

	return s.characters[idx], nil
}

// GetOrCreate получает индекс символа в алфавите. Добавляет символ, если его не было в индексе.
func (s *Storage) GetOrCreate(char rune) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if idx, alreadyExists := s.index[char]; alreadyExists {
		return idx
	}

	s.characters = append(s.characters, char)
	charIndex := len(s.characters) - 1
	s.index[char] = charIndex

	return charIndex
}

// Get получает индекс символа в алфавите. Возвращает ошибку, если символа нет в алфавите.
func (s *Storage) Get(char rune) (int, error) {
	s.mutex.RLock()
	idx, alreadyExists := s.index[char]
	s.mutex.RUnlock()

	if alreadyExists {
		return idx, nil
	}

	return -1, ErrNoRuneInAlphabet
}

// String возвращает алфавит одной строкой.
func (s *Storage) String() string {
	s.mutex.RLock()
	characters := s.characters
	s.mutex.RUnlock()

	return string(characters)
}

// Reset заполняет алфавит из заданной строки. Удаляет любые существовавшие символы.
// Индексы символов в алфавите будут соответствовать индексам символа в строке.
// Символы в строке не должны повторяться.
func (s *Storage) Reset(alphabetString string) error {
	newChars := []rune(alphabetString)
	newIndex := map[rune]int{}

	for idx, char := range newChars {
		newIndex[char] = idx
	}

	if len(newChars) != len(newIndex) {
		return ErrRepeatedCharacters
	}

	s.mutex.Lock()
	s.characters = newChars
	s.index = newIndex
	s.mutex.Unlock()

	return nil
}
