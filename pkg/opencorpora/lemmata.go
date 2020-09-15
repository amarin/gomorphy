package opencorpora

// Lemmata содержит полный корпус слов OpenCorpora.
type Lemmata struct {
	Items LemmaList `xml:"lemma"`
}

// Search позволяет найти вхождения слова в корпус.
// Возвращает список найденных вхождений или пустой список, если вхождений не найдено.
func (l Lemmata) Search(word string) (lemmaList []Lemma) {
	return l.Items.Search(word)
}
