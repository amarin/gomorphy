package index

import (
	"fmt"
	"sync"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/dag"
)

// Items stores items ordered by their ID.
type Items struct {
	items  []Item
	mu     *sync.Mutex
	nextID dag.ID
}

func NewItems() *Items {
	return &Items{
		items:  make([]Item, 1),
		mu:     new(sync.Mutex),
		nextID: 1,
	}
}

// BinaryReadFrom reads ItemIndex data from specified binutils.BinaryReader instance.
// Implements binutils.BinaryReaderFrom.
func (items *Items) BinaryReadFrom(reader *binutils.BinaryReader) (n int64, err error) {
	var (
		itemBytes   int64
		itemListLen uint32
	)

	n = 0

	if itemListLen, err = reader.ReadUint32(); err != nil {
		return n, err
	}

	items.nextID = dag.ID(itemListLen)
	items.items = make([]Item, int(itemListLen))
	for idx := 0; idx < int(itemListLen); idx++ {
		itemBytes, err = items.items[idx].BinaryReadFrom(reader)
		n += itemBytes
		if err != nil {
			return n, err
		}
	}

	return n, nil
}

// BinaryWriteTo writes ItemIndex data using supplied binutils.BinaryWriter.
// Implements binutils.BinaryWriterTo.
func (items Items) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteUint32(uint32(items.nextID)); err != nil {
		return fmt.Errorf("%w: items: write size: %v", Error, err)
	}
	indexToWrite := items.items[:items.nextID]
	for _, item := range indexToWrite {
		err = item.BinaryWriteTo(writer)
		if err != nil {
			return err
		}
	}

	return nil
}

// NextID returns next ID to use when creating new Item.
func (items Items) NextID() dag.ID {
	return items.nextID
}

// Extend adds elements
func (items *Items) Extend(count int) {
	items.items = append(items.items, make([]Item, count)...)
}

func (items *Items) NewChild(parentID dag.ID, letter rune) Item {
	items.mu.Lock() // lock items to ensure nextID will not be taken by another goroutine

	nextID := items.NextID()
	currentLen := len(items.items)
	// Extend strategy: 64, 128, 256, 512, 1024, 2048, 4096, 8192
	maxExtendPower := 13
	minExtendPower := 5

	if currentLen <= int(nextID) {
		extendTo := 2 ^ minExtendPower

		for i := maxExtendPower; i >= minExtendPower; i-- {
			extendLimit := 2 ^ i
			if currentLen > extendLimit {
				extendTo = extendLimit
				break
			}
		}
		items.Extend(extendTo)
	}

	items.nextID += 1
	items.mu.Unlock() // free items as nextID successfully acquired and list extended

	items.items[nextID].ID = nextID
	items.items[nextID].Parent = parentID
	items.items[nextID].Letter = letter
	items.items[nextID].Variants = 0 // start with empty collection at 0 index

	return items.items[nextID]
}

// Get gets item by its ID.
func (items *Items) Get(id dag.ID) *Item {
	intID := int(id)
	if intID == 0 {
		return nil
	}

	if intID >= int(items.NextID()) {
		return nil
	}

	return &items.items[intID]
}

func (items Items) filter(filterFunc func(Item) bool) Items {
	res := make([]Item, 0)
	res = append(res, Item{})

	for _, item := range items.items {
		if filterFunc(item) && item.ID != 0 {
			res = append(res, item)
		}
	}

	return Items{items: res, mu: new(sync.Mutex), nextID: dag.ID(len(res) + 1)}
}
