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

type Size interface {
	~uint8
}

type Indexer[S Size, T any] struct {
	mu  sync.RWMutex
	idx []T
}

func New[S Size, T any]() *Indexer[S, T] {
	return &Indexer[S, T]{
		mu:  sync.RWMutex{},
		idx: make([]T, 0),
	}
}

func (indexer *Indexer[S, T]) Len() int {
	indexer.mu.RLock()
	defer indexer.mu.RUnlock()

	return len(indexer.idx)
}

func (indexer *Indexer[S, T]) Append(value T) (S, error) {
	var indexSize any = S(0)

	indexer.mu.Lock()
	defer indexer.mu.Unlock()

	nextIdx := len(indexer.idx)

	switch indexSize.(type) {
	case uint8:
		if nextIdx > math.MaxUint8-1 {
			return 0, ErrOverflow
		}
		indexer.idx = append(indexer.idx, value)
		return S(nextIdx), nil
	default:
		return 0, ErrUnexpectedSize
	}
}

func (indexer *Indexer[S, T]) Get(idx S) (*T, error) {
	indexer.mu.RLock()
	defer indexer.mu.RUnlock()

	currentLen := len(indexer.idx)
	if int(idx)+1 > currentLen {
		return nil, ErrNotFound
	}

	return &indexer.idx[idx], nil
}
