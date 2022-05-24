package index

import (
	"github.com/amarin/gomorphy/pkg/storage"
)

// TagSetSubID represents 0-based ID of TagSet item in TagSetTable.
// It's a simple wrapper over storage.ID16 type.
type TagSetSubID storage.ID16

// ID16 returns TagSetSubID instance value as storage.ID16 value.
func (tagSetID TagSetSubID) ID16() storage.ID16 {
	return storage.ID16(tagSetID)
}

// TagSetTableNumber provides 0-based TagSetTable number in TagSetIndex.
// It's a simple wrapper over storage.ID16 type.
type TagSetTableNumber storage.ID16

// Add makes new TagSetTableNumber instance having value of original TagSetTableNumber incremented by specified value.
func (tagSetTableNumber TagSetTableNumber) Add(increment int) TagSetTableNumber {
	return TagSetTableNumber(int(tagSetTableNumber) + increment)
}

// ID16 returns TagSetTableNumber instance value as storage.ID16 value.
func (tagSetTableNumber TagSetTableNumber) ID16() storage.ID16 {
	return storage.ID16(tagSetTableNumber)
}

// Int returns TagSetTableNumber instance value as int.
func (tagSetTableNumber TagSetTableNumber) Int() int {
	return int(tagSetTableNumber)
}

// TagSetID generates TagSetID value TagSetTableNumber value as upper uint16 part and specified subID as lower.
func (tagSetTableNumber TagSetTableNumber) TagSetID(id TagSetSubID) TagSetID {
	return TagSetID(storage.Combine16(tagSetTableNumber.ID16(), id.ID16()))
}

// TagSetID provides unique 0-based TagSet ID in TagSetIndex.
type TagSetID storage.ID32

// TagSetSubID returns lower uint16 part of TagSetID as TagSetSubID.
func (tagSetID TagSetID) TagSetSubID() TagSetSubID {
	return TagSetSubID(storage.ID32(tagSetID).Lower16())
}

// TagSetTableNumber returns upper uint16 part of TagSetID as TagSetTableNumber.
func (tagSetID TagSetID) TagSetTableNumber() TagSetTableNumber {
	return TagSetTableNumber(storage.ID32(tagSetID).Upper16())
}
