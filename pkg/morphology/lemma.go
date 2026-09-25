package morphology

// LemmaRef — a reference to a lemma (base form): text, tag of paradigm
// form 0.
type LemmaRef struct {
	Normal string // lemma (base form)
	Tag    string // tag of the lemma (paradigm form 0)
	Para   uint16 // paradigm id — unique only together with Shard
	Shard  int    // dictionary shard index; always 0 for unsharded dictionaries
	Dict   int    // dictionary index in MultiDictionary; always 0 for Dictionary.Lemma directly
	// Predicted is true when the lemma comes from predicted readings (the
	// word is absent from the dictionary). A single Dictionary.Parse never
	// mixes predicted and dictionary readings, and MultiDictionary.Lemma
	// never merges refs across dictionaries, so every reading behind one
	// LemmaRef shares this value.
	Predicted bool
}

// Lemma returns the word's lemmas based on its readings. Deduplicated by
// the (Normal, Tag) pair — the same form with different tags (homonyms) is
// kept. Returns nil if the word is not found.
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
		out = append(out, LemmaRef{Normal: r.Normal, Tag: tag, Para: r.Para, Shard: r.Shard, Predicted: r.Predicted})
	}
	return out
}
