package tagset

import "github.com/bits-and-blooms/bitset"

type Tags struct {
	bitset.BitSet
}

func NewTags(l uint) *Tags {
	return &Tags{*bitset.New(l)}
}
