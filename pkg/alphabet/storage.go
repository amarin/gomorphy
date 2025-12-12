package alphabet

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/amarin/gomorphy/internal/storage"
	"github.com/amarin/gomorphy/pkg/size"
)

const (
	blockName = "alphabet"
	magic     = "ABC"
)

func (s *Alphabet[A]) storageConfig() storage.Config {
	indexSize := s.getIndexSize()

	return *storage.Define(
		*storage.Block("magic", storage.StaticBytes([]byte(magic))),
		*storage.Block("size", indexSize),
		*storage.ReadWrite("len", s.readLen, s.writeLen),
		*storage.ReadWrite("chars", s.readAlphabet, s.writeAlphabet),
	)
}

func (s *Alphabet[A]) ReadFrom(r io.Reader) (n int64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return storage.NewReader(blockName, s.storageConfig()).ReadFrom(r)
}

func (s *Alphabet[A]) WriteTo(w io.Writer) (n int64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return storage.NewWriter(blockName, s.storageConfig()).WriteTo(w)
}

func (s *Alphabet[A]) getIndexSize() (res *size.Index) {
	var (
		sizeDetector any = A(0)
		indexSize    size.Index
	)
	switch sizeDetector.(type) {
	case uint8:
		indexSize = size.Uint8
	case uint16:
		indexSize = size.Uint16
	default:
		return nil
	}

	return &indexSize
}

func (s *Alphabet[A]) writeLen(w io.Writer) (n int64, err error) {
	var (
		indexSize = s.getIndexSize()
	)

	if indexSize == nil {
		return 0, errors.New("index size is nil")
	}

	switch *indexSize {
	case size.Uint8:
		if err = binary.Write(w, binary.LittleEndian, uint8(len(s.characters))); err != nil {
			return 0, err
		}
		n += 1
	case size.Uint16:
		if err = binary.Write(w, binary.LittleEndian, uint16(len(s.characters))); err != nil {
			return 0, err
		}
		n += 2
	default:
		return 0, fmt.Errorf("invalid index size %v", *indexSize)
	}

	return n, nil
}

func (s *Alphabet[A]) readLen(r io.Reader) (n int64, err error) {
	var (
		indexSize = s.getIndexSize()
	)

	if indexSize == nil {
		return 0, errors.New("index size is nil")
	}

	switch *indexSize {
	case size.Uint8:
		var alphabetLen uint8
		if err = binary.Read(r, binary.LittleEndian, &alphabetLen); err != nil {
			return 0, err
		}
		n += 1

		s.index = make(map[rune]A)
		s.characters = make([]rune, alphabetLen)

	case size.Uint16:
		var alphabetLen uint16
		if err = binary.Read(r, binary.LittleEndian, &alphabetLen); err != nil {
			return 0, err
		}

		n += 2

		s.index = make(map[rune]A)
		s.characters = make([]rune, alphabetLen)
	default:
		return 0, fmt.Errorf("invalid index size %v", *indexSize)
	}

	return n, nil
}

func (s *Alphabet[A]) writeAlphabet(w io.Writer) (n int64, err error) {
	n = 0

	for _, character := range s.characters {
		if err = binary.Write(w, binary.LittleEndian, character); err != nil {
			return 0, err
		}
		n += 4
	}

	return n, nil
}

func (s *Alphabet[A]) readAlphabet(r io.Reader) (n int64, err error) {
	var currentRune rune
	n = 0

	for idx := range s.characters {
		if err = binary.Read(r, binary.LittleEndian, &currentRune); err != nil {
			return 0, err
		}
		s.characters[idx] = currentRune
		n += 4
	}

	return n, nil
}
