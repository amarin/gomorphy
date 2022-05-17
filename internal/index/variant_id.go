package index

import (
	"github.com/amarin/gomorphy/pkg/storage"
)

// VariantSubID wraps storage.ID16 to represent TagSetIDCollection ID in VariantsTable.
type VariantSubID storage.ID16

// ID16 returns storage.ID16 of VariantSubID value.
// It's a simple wrapper over storage.ID16(VariantSubID).
func (variantSubID VariantSubID) ID16() storage.ID16 {
	return storage.ID16(variantSubID)
}

func (variantSubID VariantSubID) Add(value int) VariantSubID {
	return VariantSubID(int(variantSubID) + value)
}

// CollectionTableNumber represents table number in VariantsIndex.
type CollectionTableNumber storage.ID16

func (collectionTableNumber CollectionTableNumber) Add(addition int) CollectionTableNumber {
	return CollectionTableNumber(int(collectionTableNumber) + addition)
}

func (collectionTableNumber CollectionTableNumber) ID16() storage.ID16 {
	return storage.ID16(collectionTableNumber)
}

// VariantID makes an VariantID from CollectionTableNumber using specified VariantSubID value.
func (collectionTableNumber CollectionTableNumber) VariantID(indexInTable VariantSubID) VariantID {
	return VariantID(storage.Combine16(collectionTableNumber.ID16(), indexInTable.ID16()))
}

// VariantID represents TagSet collection ID in VariantsIndex.
type VariantID storage.ID32

func (variantID VariantID) TableNum() (res CollectionTableNumber) {
	return CollectionTableNumber(storage.ID32(variantID).Upper16())
}

func (variantID VariantID) CollectionTableID() (res VariantSubID) {
	return VariantSubID(storage.ID32(variantID).Lower16())
}
