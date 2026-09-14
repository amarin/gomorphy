package morphology

// LemmaRef — ссылка на начальную форму (лемму): текст, тег формы 0 парадигмы.
type LemmaRef struct {
	Normal string // начальная форма
	Tag    string // тег начальной формы (форма 0 парадигмы)
	Para   uint16 // id парадигмы — уникален только вместе с Shard
	Shard  int    // индекс шарда словаря; всегда 0 для нешардированных словарей
}

// Lemma возвращает начальные формы слова по его разборам. Дедупликация по
// паре (Normal, Tag) — одна и та же форма с разными тегами (омонимы)
// сохраняется. Возвращает nil, если слово не найдено.
func (x *Dictionary) Lemma(word string) []LemmaRef {
	readings := x.Parse(word)
	if len(readings) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(readings))
	out := make([]LemmaRef, 0, len(readings))
	for _, r := range readings {
		para, ok := x.paradigm(r.Shard, r.Para)
		if !ok {
			continue
		}
		tag := x.paradigmTag(para, 0)
		key := r.Normal + "\x00" + tag
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, LemmaRef{Normal: r.Normal, Tag: tag, Para: r.Para, Shard: r.Shard})
	}
	return out
}
