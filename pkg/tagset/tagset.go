package tagset

import (
	"sync"

	"github.com/RoaringBitmap/roaring/v2"

	"github.com/amarin/gomorphy/pkg/indexer"
	"github.com/amarin/gomorphy/pkg/size"
	"github.com/amarin/gomorphy/pkg/tag"
)

// Set реализует взаимодействие с наборами тегов
type Set[S tagsSize, T tagSetSize] struct {
	tags *indexer.IndexOf[S, tag.Tag, *tag.Tag]
	mu   *sync.RWMutex
	sets bitmaps
}

func NewSet[S tagsSize, T tagsSize](tagsIndex *indexer.IndexOf[S, tag.Tag, *tag.Tag]) *Set[S, T] {
	return &Set[S, T]{
		tags: tagsIndex,
		mu:   new(sync.RWMutex),
		sets: make([]*roaring.Bitmap, 0),
	}
}

func (s *Set[S, T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.sets)
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
	res = make([]uint32, len(s.sets))
	for idx, tagId := range tagIds {
		res[idx] = uint32(tagId)
	}

	return res
}

func (s *Set[S, T]) Get(tagIds ...S) (res T, err error) {
	var existed int

	s.mu.RLock()
	defer s.mu.RUnlock()

	if existed, err = s.sets.get(s.tagIds(tagIds...)...); err != nil {
		return 0, err
	}

	return T(existed), nil
}

func (s *Set[S, T]) GetOrCreate(tagIds ...S) (res T, err error) {
	var existed int
	s.mu.RLock()
	defer s.mu.RUnlock()

	if existed, err = s.sets.getOrCreate(s.tagIds(tagIds...)...); err != nil {
		return 0, err
	}

	return T(existed), nil
}
