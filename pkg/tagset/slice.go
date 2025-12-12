package tagset

import (
	"errors"
	"io"

	"github.com/RoaringBitmap/roaring/v2"
)

var errNoSuchSet = errors.New("no such set")

type bitmaps []*roaring.Bitmap

func (b *bitmaps) ReadFrom(r io.Reader) (n int64, err error) {
	bitmapBytes := int64(0)

	for idx := range *b {
		(*b)[idx] = roaring.NewBitmap()
		bitmapBytes, err = (*b)[idx].ReadFrom(r)
		n += bitmapBytes

		if err != nil {
			return n, err
		}
	}

	return n, nil
}

func (b *bitmaps) WriteTo(w io.Writer) (n int64, err error) {
	bitmapBytes := int64(0)

	for idx := range *b {
		bitmapBytes, err = (*b)[idx].WriteTo(w)
		n += bitmapBytes

		if err != nil {
			return n, err
		}
	}

	return n, nil
}

func (b *bitmaps) get(tagIds ...uint32) (int, error) {
	target := roaring.BitmapOf(tagIds...)
	for idx, tagset := range *b {
		if target.Equals(tagset) {
			return idx, nil
		}
	}

	return 0, errNoSuchSet
}

func (b *bitmaps) getOrCreate(tagIds ...uint32) (int, error) {
	existedIdx, err := b.get(tagIds...)
	if err == nil {
		return existedIdx, nil
	}

	return b.create(tagIds...), nil
}

// create создаёт новый набор тегов. Возвращает идентификатор созданного набора тегов.
func (b *bitmaps) create(tagIds ...uint32) int {
	newSetIdx := len(*b)
	*b = append(*b, roaring.BitmapOf(tagIds...))

	return newSetIdx
}
