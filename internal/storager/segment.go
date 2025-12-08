package storager

import "io"

type writerFunc func(w io.Writer) (n int64, err error)
type readerFunc func(r io.Reader) (n int64, err error)

type Segment struct {
	read  readerFunc
	write writerFunc
}

func (s *Segment) Read(r io.Reader) (n int64, err error) {
	return s.read(r)
}

func (s *Segment) Write(w io.Writer) (n int64, err error) {
	return s.write(w)
}

func (s *Segment) Reader(f readerFunc) *Segment {
	s.read = f
	return s
}

func (s *Segment) Writer(f writerFunc) *Segment {
	s.write = f
	return s
}
