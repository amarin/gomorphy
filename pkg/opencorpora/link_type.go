package opencorpora

// LinkType определяет тип связей между частями речи.
// Задаёт возможный тип преобразования из одной части речи в другую.
// Используется в определении связи между леммами Link.
type LinkType struct {
	IDAttr int `xml:"id,attr"`
}
