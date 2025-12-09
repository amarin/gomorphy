package storager

import "io"

// Segment задаёт функции чтения и записи сегмента данных,
// а также имя сегмента для логирования чтения и записи.
type Segment struct {
	name       string
	readerFunc readerFunc
	writerFunc writerFunc
}

func NewSegment(name string, reader readerFunc, writer writerFunc) *Segment {
	return &Segment{
		name:       name,
		readerFunc: reader,
		writerFunc: writer,
	}
}

func (s *Segment) Read(r io.Reader) (n int64, err error) {
	return s.readerFunc(r)
}

func (s *Segment) Write(w io.Writer) (n int64, err error) {
	return s.writerFunc(w)
}

func (s *Segment) Reader(f readerFunc) *Segment {
	s.readerFunc = f
	return s
}

func (s *Segment) Writer(f writerFunc) *Segment {
	s.writerFunc = f
	return s
}
