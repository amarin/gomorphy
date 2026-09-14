package internal

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
func (t *TagSet) Add(name string) uint16 {
	if t.Index == nil {
		t.Index = make(map[string]uint16)
	}
	if id, ok := t.Index[name]; ok {
		return id
	}
	id := uint16(len(t.Tags))
	t.Tags = append(t.Tags, name)
	t.Index[name] = id
	return id
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
