package indexer

import (
	"errors"
	"math"
	"sync"
)

var (
	ErrOverflow       = errors.New("overflow")
	ErrUnexpectedSize = errors.New("unexpected indexer size")
	ErrNotFound       = errors.New("not found")
)

// IndexOf реализует индекс элементов.
type IndexOf[S size, T any, I item[T]] struct {
	name string

	mu  *sync.RWMutex
	idx []I
}

// New создаёт новых индекс.
func New[S size, T any, I item[T]](name string) *IndexOf[S, T, I] {
	return &IndexOf[S, T, I]{
		mu:   new(sync.RWMutex),
		idx:  make([]I, 0),
		name: name,
	}
}

// Len возвращает длину списка элементов в индексе.
func (indexer *IndexOf[S, T, I]) Len() int {
	indexer.mu.RLock()
	defer indexer.mu.RUnlock()

	return len(indexer.idx)
}

// Add Добавляет элемент в конец списка.
func (indexer *IndexOf[S, T, I]) Add(value T) (S, error) {
	var indexSize any = S(0)

	indexer.mu.Lock()
	defer indexer.mu.Unlock()

	nextIdx := len(indexer.idx)

	switch indexSize.(type) {
	case uint8:
		if nextIdx > math.MaxUint8-1 {
			return 0, ErrOverflow
		}
		indexer.idx = append(indexer.idx, &value)
		return S(nextIdx), nil
	default:
		return 0, ErrUnexpectedSize
	}
}

// MustAdd Добавляет элемент в конец списка. Паникует, если не удалось добавить.
func (indexer *IndexOf[S, T, I]) MustAdd(value T) S {
	idx, err := indexer.Add(value)
	if err != nil {
		panic(err)
	}

	return idx
}

// Get получает элемент из списка по индексу.
func (indexer *IndexOf[S, T, I]) Get(idx S) (*T, error) {
	indexer.mu.RLock()
	defer indexer.mu.RUnlock()

	currentLen := len(indexer.idx)
	if int(idx)+1 > currentLen {
		return nil, ErrNotFound
	}

	return indexer.idx[idx], nil
}

// MustGet получает элемент из списка. Паникует, если не удалось получить.
func (indexer *IndexOf[S, T, I]) MustGet(idx S) T {
	elem, err := indexer.Get(idx)
	if err != nil {
		panic(err)
	}

	return *elem
}
