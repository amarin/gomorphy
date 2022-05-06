package index

import (
	"sort"

	"github.com/amarin/gomorphy/pkg/dag"
)

// Child provides Child from parent to child and vise versa.
type Child struct {
	Parent dag.ID
	ID     dag.ID
}

// ChildList stores Items children.
type ChildList []Child

// Len returns length of ChildList. Implements sort.Interface.
func (childrenList ChildList) Len() int {
	return len(childrenList)
}

// Less reports whether the element with index i must sort before the element with index j. Implements sort.Interface.
func (childrenList ChildList) Less(i, j int) bool {
	return childrenList[i].Parent < childrenList[j].Parent
}

// Swap swaps elements in i and j positions. Implements sort.Interface.
func (childrenList ChildList) Swap(i, j int) {
	childrenList[i], childrenList[j] = childrenList[j], childrenList[i]
}

func (childrenList ChildList) Get(id dag.ID) *Child {
	return &childrenList[id]
}

func (childrenList ChildList) Sort() chan<- Child {
	sort.Sort(childrenList)
	res := make(chan Child)

	go func() {
		previousParent := -1
		for idx, rel := range childrenList {
			if int(rel.Parent) != previousParent {
				res <- Child{
					Parent: rel.Parent,
					ID:     dag.ID(idx),
				}
			}
		}
	}()

	return res
}
