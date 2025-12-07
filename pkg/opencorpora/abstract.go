package opencorpora

import (
	"io"

	"github.com/amarin/gomorphy/pkg/dag"
	"github.com/amarin/gomorphy/pkg/tag"
)

// Index defines main dictionary interface.
type Index interface {
	// AddRunes adds runes sequence into index.
	// Returns final node filled with node data or error if add caused error.
	AddRunes([]rune) (dag.Node, error)

	// AddString adds string word into index.
	// Returns final node or error if add caused error.
	AddString(word string) (node dag.Node, err error)

	// FetchRunes lookups runes sequence in container.
	// If found returns final node or error if not found.
	FetchRunes([]rune) (dag.Node, error)

	// FetchString lookups string in container.
	// If found returns final node or error if not found.
	FetchString(word string) (dag.Node, error)

	// TagID returns index of grammeme specified by name and parent name.
	TagID(name tag.Name, parent tag.Name) dag.TagID
}

type BinaryWriterTo interface {
	BinaryWriteTo(writer io.Writer) error
}

type mainIndex interface {
	WordsCount() int
	NodesCount() int
	Optimize()
	BinaryWriteTo(writer io.Writer) error
}

type SimpleIndex interface {
	mainIndex
	Add(word string, tags ...any) (int, error)
}
