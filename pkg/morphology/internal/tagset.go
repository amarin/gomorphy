package internal

import "errors"

// ErrTagSetFull is returned by TagSet.Add once 65536 unique tags (the
// uint16 id space) are already registered.
var ErrTagSetFull = errors.New("tagset: too many unique tags (max 65536)")

// TagSet — набор грамматических тегов словаря: id ↔ имя.
type TagSet struct {
	Name  string
	Tags  []string          // id → имя тега
	Index map[string]uint16 // имя тега → id
}

// NewTagSet создаёт пустой TagSet с готовой таблицей индексов.
func NewTagSet(name string) *TagSet {
	return &TagSet{Name: name, Index: make(map[string]uint16)}
}

// Add добавляет тег по имени, возвращает его id. Дедупликация по имени.
// Возвращает ErrTagSetFull, если все 65536 значений uint16 уже заняты.
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

// ID возвращает id тега по имени.
func (t *TagSet) ID(name string) (uint16, bool) {
	id, ok := t.Index[name]
	return id, ok
}

// TagName возвращает имя тега по id ("" для несуществующего id).
func (t *TagSet) TagName(id uint16) string {
	if int(id) < len(t.Tags) {
		return t.Tags[id]
	}
	return ""
}
