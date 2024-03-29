package index

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/amarin/binutils"

	"github.com/amarin/gomorphy/pkg/storage"
)

// TagSetIDCollection stores a list of TagSetID identifiers of TagSet stored in TagSetIndex.
type TagSetIDCollection []TagSetID

func (t TagSetIDCollection) GoString() string {
	stringsList := make([]string, len(t))
	for idx, ts := range t {
		stringsList[idx] = strconv.FormatUint(uint64(ts), 16)
	}

	return "[" + strings.Join(stringsList, ",") + "]"
}

// BinaryWriteTo writes TagSetIndex data using specified binutils.BinaryWriter instance.
// Returns error if happens or nil.
// Implements binutils.BinaryWriterTo.
func (t TagSetIDCollection) BinaryWriteTo(writer *binutils.BinaryWriter) (err error) {
	if err = writer.WriteUint8(uint8(len(t))); err != nil {
		return fmt.Errorf("%w: %v", Error, err)
	}

	for _, tagSetTable := range t {
		if err = writer.WriteUint32(storage.ID32(tagSetTable).Uint32()); err != nil {
			return fmt.Errorf("%w: %v", Error, err)
		}
	}

	return nil
}

// BinaryReadFrom reads TagSetIndex data using specified binutils.BinaryReader instance.
// Returns error if happens or nil.
// Implements binutils.BinaryReaderFrom.
func (t *TagSetIDCollection) BinaryReadFrom(reader *binutils.BinaryReader) (err error) {
	var (
		tagSetIndexLen uint8
		currentUint32  uint32
	)

	if tagSetIndexLen, err = reader.ReadUint8(); err != nil {
		return fmt.Errorf("%w: read: tagset: %v", Error, err)
	}

	*t = make(TagSetIDCollection, tagSetIndexLen)
	for idx := 0; idx < int(tagSetIndexLen); idx++ {
		if currentUint32, err = reader.ReadUint32(); err != nil {
			return fmt.Errorf("%w: read: tagset: %v", Error, err)
		}
		(*t)[idx] = TagSetID(currentUint32)
	}

	return nil
}

// Len returns length of TagSetIDCollection. Implements sort.Interface.
func (t TagSetIDCollection) Len() int {
	return len(t)
}

// Less returns true i-th element of TagSetIDCollection is fewer than j-th. Implements sort.Interface.
func (t TagSetIDCollection) Less(i, j int) bool {
	return t[i] < t[j]
}

// Swap swaps i-th and j-th elements of array in place. Implements sort.Interface.
func (t TagSetIDCollection) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// EqualTo compares TagSetIDCollection with another one.
// Returns true if both sets are contains the same ID8 elements or false otherwise.
// Note: both sets should be sorted before compare.
func (t TagSetIDCollection) EqualTo(another TagSetIDCollection) bool {
	if t.Len() != another.Len() { // fast non-equal if length differs.
		return false
	}

	for idx := 0; idx < t.Len(); idx++ {
		if t[idx] != another[idx] { // nok if own ids[i] != another ids[i]
			return false
		}
	}

	return true
}

// Has returns true if TagSetIDCollection contains specified TagSetID.
func (t TagSetIDCollection) Has(searchTagSetID TagSetID) bool {
	for _, tsID := range t {
		if tsID == searchTagSetID {
			return true
		}
	}

	return false
}

// Add makes a new TagSetIDCollection having all TagSetID's from original plus specified additional TagSetID.
// Result is sorted and ready to place into VariantsIndex.
// If original TagSetIDCollection already contains specified element,
// resulting TagSetIDCollection will be the same as original one.
func (t TagSetIDCollection) Add(additionalTagSetID TagSetID) TagSetIDCollection {
	if t.Has(additionalTagSetID) {
		return t
	}

	res := make(TagSetIDCollection, len(t)+1)
	copy(res, t)
	res[len(t)] = additionalTagSetID
	sort.Sort(res)

	return res
}
