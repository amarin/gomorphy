//go:generate mockgen -source=${GOFILE} -destination=mocks_test.go -package=${GOPACKAGE}
package indexer

import "io"

type size interface {
	~uint8
}

type item[T any] interface {
	io.ReaderFrom
	io.WriterTo
	*T
}

//type itemConstructor[T any] func() *T
