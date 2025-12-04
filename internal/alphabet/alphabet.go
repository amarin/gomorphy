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
