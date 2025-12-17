package tagset

import (
	"errors"
	"fmt"
	"sync"

	"github.com/amarin/gomorphy/pkg/indexer"
	"github.com/amarin/gomorphy/pkg/size"
	"github.com/amarin/gomorphy/pkg/tag"
)

var errNoSuchSet = errors.New("no such set")

// Set реализует взаимодействие с наборами тегов
type Set[S tagsSize, T tagSetSize] struct {
	tags *indexer.IndexOf[S, tag.Name, *tag.Name]
	mu   *sync.RWMutex
	sets *indexer.IndexOf[S, Tags, *Tags]
}

func NewSet[S tagsSize, T tagSetSize](tagsIndex *indexer.IndexOf[S, tag.Name, *tag.Name]) *Set[S, T] {
	return &Set[S, T]{
		tags: tagsIndex,
		mu:   new(sync.RWMutex),
		sets: indexer.New[S, Tags, *Tags]("tagsInSet"),
	}
}

func (s *Set[S, T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.sets.Len()
}

func (s *Set[S, T]) Size() size.Index {
	var idxSize any = T(0)
	switch idxSize.(type) {
	case uint8:
		return size.Uint8
	case uint16:
		return size.Uint16
	case uint32:
		return size.Uint32
	default:
		return size.Unknown
	}
}
func (s *Set[S, T]) tagIds(tagIds ...S) (res []uint32) {
	res = make([]uint32, len(tagIds))
	for idx, tagId := range tagIds {
		res[idx] = uint32(tagId)
	}

	return res
}

func (s *Set[S, T]) GetTags(tagSetIdx T) ([]*tag.Name, error) {
	tagIndexes, err := s.getTagIndexesById(tagSetIdx)
	if err != nil {
		return nil, err
	}

	res := make([]*tag.Name, len(tagIndexes))
	for idx, tagIdx := range tagIndexes {
		res[idx], err = s.tags.Get(tagIdx)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (s *Set[S, T]) getTagIndexesById(idx T) ([]S, error) {
	if int(idx) >= s.sets.Len() {
		return nil, errNoSuchSet
	}

	tagSet, err := s.sets.Get(S(idx))
	if err != nil {
		return nil, fmt.Errorf("get tag set: %w", err)
	}

	i := 0
	res := make([]S, tagSet.Len())
	for x := range tagSet.EachSet() {
		res[i] = S(x)
		i++
	}

	return res, nil
}

func (s *Set[S, T]) Get(tagIds ...S) (res T, err error) {
	var existedSet, lookupSet *Tags

	s.mu.RLock()
	defer s.mu.RUnlock()

	lookupSet = NewTags(uint(s.tags.Len()))
	for _, tagId := range tagIds {
		lookupSet.Set(uint(tagId))
	}

	for _, idx := range s.sets.Keys() {
		existedSet, err = s.sets.Get(idx)
		switch {
		case err != nil:
			return 0, fmt.Errorf("get tag set: %w", err)
		case existedSet == nil:
			return 0, fmt.Errorf("get tag set: %w", err)
		case existedSet.BitSet.Equal(&lookupSet.BitSet):
			return T(idx), nil
		}
	}

	return 0, errNoSuchSet
}

func (s *Set[S, T]) GetOrCreate(tagIds ...S) (res T, err error) {
	var existedSet, lookupSet *Tags

	s.mu.Lock()
	defer s.mu.Unlock()

	lookupSet = NewTags(uint(s.tags.Len()))
	for _, tagId := range tagIds {
		lookupSet.Set(uint(tagId))
	}

	for _, idx := range s.sets.Keys() {
		existedSet, err = s.sets.Get(idx)
		switch {
		case err != nil:
			return 0, fmt.Errorf("get tag set: %w", err)
		case existedSet == nil:
			return 0, fmt.Errorf("get tag set: %w", err)
		case existedSet.BitSet.Equal(&lookupSet.BitSet):
			return T(idx), nil
		}
	}

	nextId := T(s.sets.Len())
	_, err = s.sets.Add(*lookupSet)
	if err != nil {
		return 0, err
	}

	return nextId, nil
}
