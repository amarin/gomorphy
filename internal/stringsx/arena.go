// Package stringsx provides packed byte arena with uint32-offset string storage.
package stringsx

import (
	"math"
)

const defaultCapBytes = 1 << 16

type Arena struct {
	data    []byte
	offsets []uint32
}

func New(capBytes, capStrings int) *Arena {
	if capBytes <= 0 {
		capBytes = defaultCapBytes
	}

	if capStrings <= 0 {
		capStrings = 1024
	}

	return &Arena{
		data:    make([]byte, 0, capBytes),
		offsets: make([]uint32, 1, capStrings+1),
	}
}

func (a *Arena) Append(b []byte) uint32 {
	next := uint64(len(a.data)) + uint64(len(b))
	if next > math.MaxUint32 {
		panic("stringsx: arena overflow, max 4GiB supported")
	}

	id := uint32(len(a.offsets)) - 1
	a.data = append(a.data, b...)
	a.offsets = append(a.offsets, uint32(len(a.data)))

	return id
}

func (a *Arena) Get(id uint32) []byte {
	return a.data[a.offsets[id]:a.offsets[id+1]]
}

func (a *Arena) Len() int {
	return len(a.offsets) - 1
}

func (a *Arena) Size() int {
	return len(a.data)
}
