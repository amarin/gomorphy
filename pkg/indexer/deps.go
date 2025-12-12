//go:generate mockgen -source=${GOFILE} -destination=mocks_test.go -package=${GOPACKAGE}
package indexer

import "io"

type indexSize interface {
	uint8 | uint16 | uint32
}

type item[T any] interface {
	io.ReaderFrom
	io.WriterTo
	*T
}
