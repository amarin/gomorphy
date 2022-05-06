package dag

import (
	"fmt"
)

// Node implements container methods as well as node specific rune fetcher and parent resolver.
type Node interface {
	fmt.Stringer

	// TagSets returns node TagSet.
	TagSets() []TagSet

	// AddTagSet adds a new TagSet to node data.
	AddTagSet(newTagSet ...TagName) error

	// Word returns sequence of characters from root upto current node wrapped into string.
	Word() string
}

// Index defines main dictionary interface.
type Index interface {
	// AddRunes adds runes sequence into index.
	// Returns final node filled with node data or error if add caused error.
	AddRunes([]rune) (Node, error)

	// AddString adds string word into index.
	// Returns final node or error if add caused error.
	AddString(word string) (node Node, err error)

	// FetchRunes lookups runes sequence in container.
	// If found returns final node or error if not found.
	FetchRunes([]rune) (Node, error)

	// FetchString lookups string in container.
	// If found returns final node or error if not found.
	FetchString(word string) (Node, error)

	// TagID returns index of grammeme specified by name and parent name.
	TagID(name TagName, parent TagName) TagID
}
