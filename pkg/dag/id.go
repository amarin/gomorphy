package dag

import (
	"github.com/amarin/gomorphy/pkg/storage"
)

// ID represents node ID.
type ID storage.ID32

// IDList provides simple list of items.
type IDList []ID
