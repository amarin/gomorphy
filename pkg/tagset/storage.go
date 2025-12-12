package tagset

import (
	"encoding/binary"
	"io"

	"github.com/RoaringBitmap/roaring/v2"

	"github.com/amarin/gomorphy/internal/storage"
)

const storageName = "tagSets"

func (s *Set[S, T]) ReadFrom(r io.Reader) (n int64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return storage.NewReader(storageName, s.storageConfig()).ReadFrom(r)
}

func (s *Set[S, T]) WriteTo(w io.Writer) (n int64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return storage.NewWriter(storageName, s.storageConfig()).WriteTo(w)
}

func (s *Set[S, T]) storageConfig() storage.Config {
	return *storage.Define(
		*storage.ReadWrite("len", s.readLen, s.writeLen),
		*storage.ReadWrite("len", s.readSets, s.writeSets),
	)
}

func (s *Set[S, T]) writeSets(w io.Writer) (n int64, err error) {
	bitmapBytes := int64(0)
	for idx := range s.sets {
		bitmapBytes, err = s.sets[idx].WriteTo(w)
		n += bitmapBytes

		if err != nil {
			return n, err
		}
	}

	return n, nil
}

func (s *Set[S, T]) readSets(r io.Reader) (n int64, err error) {
	bitmapBytes := int64(0)

	for idx := range s.sets {
		s.sets[idx] = roaring.NewBitmap()
		bitmapBytes, err = s.sets[idx].ReadFrom(r)
		n += bitmapBytes

		if err != nil {
			return n, err
		}
	}

	return n, nil
}

func (s *Set[S, T]) writeLen(w io.Writer) (int64, error) {
	selLen := len(s.sets)
	err := binary.Write(w, binary.LittleEndian, s.Len())
	if err != nil {
		return 0, err
	}
	return int64(binary.Size(selLen)), nil
}

func (s *Set[S, T]) readLen(r io.Reader) (int64, error) {
	var selLen int
	err := binary.Read(r, binary.LittleEndian, &selLen)
	if err != nil {
		return 0, err
	}

	s.sets = make([]*roaring.Bitmap, selLen)
	return int64(binary.Size(selLen)), nil
}
