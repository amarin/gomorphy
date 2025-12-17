package words

import (
	"io"

	"github.com/RoaringBitmap/roaring/v2"

	"github.com/amarin/gomorphy/pkg/size"
)

type TagSets[S size.TagSetSize] struct {
	*roaring.Bitmap
}

func NewTagSets[S size.TagSetSize](ids ...S) *TagSets[S] {
	t := &TagSets[S]{
		Bitmap: roaring.NewBitmap(),
	}
	if len(ids) > 0 {
		t.AddMany(storageIds(ids...))
	}

	return t
}

func storageIds[S size.TagSetSize](tagIds ...S) (res []uint32) {
	res = make([]uint32, len(tagIds))
	for idx, tagId := range tagIds {
		res[idx] = uint32(tagId)
	}

	return res
}

func (t *TagSets[S]) ReadFrom(r io.Reader) (n int64, err error) {
	t.Bitmap = roaring.NewBitmap()
	return t.Bitmap.ReadFrom(r)
}

func (t *TagSets[S]) WriteTo(w io.Writer) (n int64, err error) {
	return t.Bitmap.WriteTo(w)
}
