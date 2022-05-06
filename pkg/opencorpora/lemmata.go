package opencorpora

// Lemmata provides OpenCorpora lemma corpus.
type Lemmata struct {
	Items LemmaList `xml:"lemma"`
}

// Search returns lemma list for specified word or empty list if nothing found.
func (l Lemmata) Search(word string) (lemmaList []Lemma) {
	return l.Items.Search(word)
}
