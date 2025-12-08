package node

import (
	"errors"
	"fmt"
	"io"
)

const magicMimePrefix = "dag"

type writerFunc func(w io.Writer) (n int64, err error)

func (graph *Graph[A, M]) WriteTo(w io.Writer) (n int64, err error) {
	n = 0
	count := int64(0)
	for _, part := range []writerFunc{
		graph.writeMagic,
	} {
		count, err = part(w)
		n += count

		if err != nil {
			return n, err
		}
	}

	return n, nil
}

type readerFunc func(r io.Reader) (n int64, err error)

func (graph *Graph[A, M]) ReadFrom(r io.Reader) (n int64, err error) {
	n = 0
	count := int64(0)
	for _, part := range []readerFunc{
		graph.readMagic,
	} {
		count, err = part(r)
		n += count

		if err != nil {
			return n, err
		}
	}

	return n, nil
}

func (graph *Graph[A, M]) BinaryWriteTo(writer io.Writer) error {
	if _, err := writer.Write([]byte(magicMimePrefix)); err != nil {
		return err
	}

	alphabetString := graph.withAlphabet.String()
	if _, err := fmt.Fprintf(writer, "%dZ", len(alphabetString)); err != nil {
		return err
	}
	if _, err := writer.Write([]byte(alphabetString)); err != nil {
		return err
	}

	for _, node := range graph.nodes {
		if _, err := node.WriteTo(writer); err != nil {
			return err
		}
	}

	return nil
}

func (graph *Graph[A, M]) writeMagic(w io.Writer) (n int64, err error) {
	count, err := w.Write([]byte(magicMimePrefix))
	return int64(count), err
}

var ErrWrongMagic = errors.New("wrong magic")

func (graph *Graph[A, M]) readMagic(r io.Reader) (n int64, err error) {
	magic := []byte("   ")
	count, err := r.Read(magic)
	if err != nil {
		return int64(count), err
	}
	if string(magic) != magicMimePrefix {
		return int64(count), ErrWrongMagic
	}
	return int64(count), nil
}
