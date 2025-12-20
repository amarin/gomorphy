package tagset

import (
	"encoding/binary"
	"io"

	"github.com/amarin/gomorphy/internal/storage"
	"github.com/amarin/gomorphy/pkg/indexer"
)

const storageName = "tagSets"

func (s *Set[A, W]) ReadFrom(r io.Reader) (n int64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return storage.NewReader(storageName, s.storageConfig()).ReadFrom(r)
}

func (s *Set[A, W]) WriteTo(w io.Writer) (n int64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return storage.NewWriter(storageName, s.storageConfig()).WriteTo(w)
}

func (s *Set[A, W]) storageConfig() storage.Config {
	return *storage.Define(
		*storage.ReadWrite("len", s.readLen, s.writeLen),
		*storage.ReadWrite("len", s.readSets, s.writeSets),
	)
}

func (s *Set[A, W]) writeSets(w io.Writer) (n int64, err error) {
	bitmapBytes := int64(0)
	for _, idx := range s.sets.Keys() {
		nextSet := s.sets.MustGet(idx)
		bitmapBytes, err = nextSet.WriteTo(w)
		n += bitmapBytes

		if err != nil {
			return n, err
		}
	}

	return n, nil
}

func (s *Set[A, W]) readSets(r io.Reader) (n int64, err error) {
	bitmapBytes := int64(0)

	for _, idx := range s.sets.Keys() {
		nextSet := NewTags(uint(s.tags.Len()))
		bitmapBytes, err = nextSet.ReadFrom(r)
		n += bitmapBytes
		if err != nil {
			return n, err
		}
		if err = s.sets.Set(idx, *nextSet); err != nil {
			return n, err
		}
	}

	return n, nil
}

func (s *Set[A, W]) writeLen(w io.Writer) (int64, error) {
	selLen := s.sets.Len()
	err := binary.Write(w, binary.LittleEndian, selLen)
	if err != nil {
		return 0, err
	}
	return int64(binary.Size(selLen)), nil
}

func (s *Set[A, W]) readLen(r io.Reader) (int64, error) {
	var selLen int
	err := binary.Read(r, binary.LittleEndian, &selLen)
	if err != nil {
		return 0, err
	}

	s.sets = indexer.New[A, Tags, *Tags]("tagsInSet")
	return int64(binary.Size(selLen)), nil
}
