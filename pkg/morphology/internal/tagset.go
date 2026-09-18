package internal

import "errors"

// ErrTagSetFull is returned by TagSet.Add once 65536 unique tags (the
// uint16 id space) are already registered.
var ErrTagSetFull = errors.New("tagset: too many unique tags (max 65536)")

// TagSet is a dictionary's set of grammatical tags: id <-> name.
type TagSet struct {
	Name  string
	Tags  []string          // id -> tag name
	Index map[string]uint16 // tag name -> id
}

// NewTagSet creates an empty TagSet with its index table ready.
func NewTagSet(name string) *TagSet {
	return &TagSet{Name: name, Index: make(map[string]uint16)}
}

// Add registers a tag by name, returning its id. Deduplicated by name.
// Returns ErrTagSetFull once all 65536 uint16 values are already taken.
func (t *TagSet) Add(name string) (uint16, error) {
	if t.Index == nil {
		t.Index = make(map[string]uint16)
	}
	if id, ok := t.Index[name]; ok {
		return id, nil
	}
	if len(t.Tags) >= 1<<16 {
		return 0, ErrTagSetFull
	}
	id := uint16(len(t.Tags))
	t.Tags = append(t.Tags, name)
	t.Index[name] = id
	return id, nil
}

// ID returns a tag's id by name.
func (t *TagSet) ID(name string) (uint16, bool) {
	id, ok := t.Index[name]
	return id, ok
}

// TagName returns a tag's name by id ("" for a nonexistent id).
func (t *TagSet) TagName(id uint16) string {
	if int(id) < len(t.Tags) {
		return t.Tags[id]
	}
	return ""
}
