package opencorpora

type LemmaList []*Lemma

// Search возвращает полный список лемм.
func (l LemmaList) Search(word string) (lemmaList []Lemma) {
	for _, lemmaPtr := range l {
		if lemmaPtr != nil {
			lemma := *lemmaPtr
			if lemma.L.Form.String() == word {
				lemmaList = append(lemmaList, lemma)
			}
		}
	}

	return lemmaList
}
