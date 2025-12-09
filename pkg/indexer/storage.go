package indexer

import (
	"encoding/binary"
	"io"

	"github.com/amarin/gomorphy/internal/storage"
)

// storageConfig возвращает конфигурацию чтения и записи данных индекса.
func (indexer *IndexOf[S, T, I]) storageConfig() *storage.Config {
	var idxSize = Uint8
	return storage.Define(
		*storage.Block("size", storage.StaticBytes([]byte{byte(idxSize)})),
		*storage.ReadWrite("length", indexer.readLength, indexer.writeLength),
		*storage.ReadWrite("items", indexer.readItems, indexer.writeItems),
	)
}

// writeLength записывает длину списка.
func (indexer *IndexOf[S, T, I]) writeLength(w io.Writer) (n int64, err error) {
	var indexSize any = S(0)
	switch indexSize.(type) {
	case uint8:
		if err = binary.Write(w, binary.LittleEndian, byte(len(indexer.idx))); err != nil {
			return 0, err
		}
		return 1, nil
	default:
		return 0, errUnknownSize
	}
}

// readItems читает значение длины списка и инициализирует индекс с заданным количеством элементов.
func (indexer *IndexOf[S, T, I]) readLength(r io.Reader) (n int64, err error) {
	var indexSize any = S(0)

	switch indexSize.(type) {
	case uint8:
		byteValue := byte(0)
		if err = binary.Read(r, binary.LittleEndian, &byteValue); err != nil {
			return 0, err
		}
		indexer.idx = make([]I, int(byteValue))

		return 1, nil
	default:
		return 0, errUnknownSize
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
