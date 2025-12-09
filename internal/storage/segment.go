package storage

import "io"

// DataSegment задаёт функции чтения и записи сегмента данных,
// а также имя сегмента для логирования чтения и записи.
type DataSegment struct {
	name       string
	readerFunc ReaderFromFunc
	writerFunc WriterToFunc
}

// ReadWrite создаёт новый экземпляр обработчика сегмента данных.
func ReadWrite(name string, reader ReaderFromFunc, writer WriterToFunc) *DataSegment {
	return &DataSegment{
		name:       name,
		readerFunc: reader,
		writerFunc: writer,
	}
}

// Block создаёт экземпляр обработчика сегмента данных из готовой реализации Segment.
func Block(name string, impl Segment) *DataSegment {
	return &DataSegment{
		name:       name,
		readerFunc: impl.ReadFrom,
		writerFunc: impl.WriteTo,
	}
}

// Read читает данные сегмента.
func (s *DataSegment) Read(r io.Reader) (n int64, err error) {
	return s.readerFunc(r)
}

// Write записывает данные сегмента.
func (s *DataSegment) Write(w io.Writer) (n int64, err error) {
	return s.writerFunc(w)
}
