package indexer

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/amarin/gomorphy/internal/storage"
	"github.com/amarin/gomorphy/pkg/size"
)

const (
	storageBlockSizeName   = "indexSize"
	storageBlockLengthName = "length"
	storageBlockItemsName  = "items"
)

var errUnexpectedIndexSize = errors.New("unexpected index indexSize")

// storageConfig возвращает конфигурацию чтения и записи данных индекса.
func (indexer *IndexOf[S, T, I]) storageConfig() *storage.Config {
	var idxSize = indexer.Size()

	return storage.Define(
		*storage.Block(storageBlockSizeName, &idxSize),
		*storage.ReadWrite(storageBlockLengthName, indexer.readLength, indexer.writeLength),
		*storage.ReadWrite(storageBlockItemsName, indexer.readItems, indexer.writeItems),
	)
}

// writeLength записывает длину списка.
func (indexer *IndexOf[S, T, I]) writeLength(w io.Writer) (n int64, err error) {
	switch s := indexer.Size(); s {
	case size.Uint8:
		if err = binary.Write(w, binary.LittleEndian, uint8(len(indexer.idx))); err != nil {
			return 0, err
		}
		return s.BytesCount(), nil
	case size.Uint16:
		if err = binary.Write(w, binary.LittleEndian, uint16(len(indexer.idx))); err != nil {
			return 0, err
		}
		return s.BytesCount(), nil
	default:
		return 0, errUnexpectedIndexSize
	}
}

// readItems читает значение длины списка и инициализирует индекс с заданным количеством элементов.
func (indexer *IndexOf[S, T, I]) readLength(r io.Reader) (n int64, err error) {
	switch s := indexer.Size(); s {
	case size.Uint8:
		var innerSize uint8
		if err = binary.Read(r, binary.LittleEndian, &innerSize); err != nil {
			return 0, err
		}
		indexer.idx = make([]I, innerSize)

		return s.BytesCount(), nil
	case size.Uint16:
		var innerSize uint16
		if err = binary.Read(r, binary.LittleEndian, &innerSize); err != nil {
			return 0, err
		}
		indexer.idx = make([]I, innerSize)

		return s.BytesCount(), nil
	default:
		return 0, errUnexpectedIndexSize
	}
}

// writeLength записывает элементы списка.
func (indexer *IndexOf[S, T, I]) writeItems(w io.Writer) (int64, error) {
	totalBytes := int64(0)
	for _, elem := range indexer.idx {
		itemBytes, err := elem.WriteTo(w)
		totalBytes += itemBytes
		if err != nil {
			return totalBytes, err
		}
	}

	return totalBytes, nil
}

// readItems читает элементы списка.
func (indexer *IndexOf[S, T, I]) readItems(r io.Reader) (int64, error) {
	totalBytes := int64(0)

	for idx := range indexer.idx {
		indexer.idx[idx] = new(T)
		itemBytes, err := indexer.idx[idx].ReadFrom(r)
		totalBytes += itemBytes

		if err != nil {
			return totalBytes, err
		}
	}

	return totalBytes, nil
}

// WriteTo записывает индекс в заданный io.Writer.
func (indexer *IndexOf[S, T, I]) WriteTo(w io.Writer) (n int64, err error) {
	indexer.mu.Lock()
	defer indexer.mu.Unlock()

	return storage.NewWriter(indexer.name, *indexer.storageConfig()).WriteTo(w)
}

// ReadFrom загружает индекс из заданного io.Reader.
func (indexer *IndexOf[S, T, I]) ReadFrom(r io.Reader) (n int64, err error) {
	indexer.mu.Lock()
	defer indexer.mu.Unlock()

	return storage.NewReader(indexer.name, *indexer.storageConfig()).ReadFrom(r)
}
