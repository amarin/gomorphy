package opencorpora

type LemmaList []*Lemma

// Search returns Lemma list.
func (l LemmaList) Search(word string) (lemmaList []Lemma) {
	for _, lemmaPtr := range l {
		if lemmaPtr != nil {
			lemma := *lemmaPtr
			if lemma.L.Form == word {
				lemmaList = append(lemmaList, lemma)
			}
		}
	}

	return lemmaList
}
