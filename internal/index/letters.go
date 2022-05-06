package index

import "github.com/amarin/gomorphy/pkg/dag"

// Letter maps letter associated with item ID.
type Letter struct {
	ID     dag.ID
	Letter rune
}
