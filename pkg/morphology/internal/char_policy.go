package internal

// Substitution — пара подменяемых символов поиска (например, е→ё).
type Substitution struct {
	From rune
	To   rune
}

// CharPolicy — набор подмен символов для поиска. По умолчанию пуст
// (нейтрален к языку); для русского настраивается подмена е→ё.
type CharPolicy struct {
	Substitutions []Substitution
}

// NewCharPolicy создаёт CharPolicy из пар подстановок.
func NewCharPolicy(subs ...Substitution) *CharPolicy {
	return &CharPolicy{Substitutions: subs}
}

// RussianCharPolicy — политика для русского языка: е→ё.
func RussianCharPolicy() *CharPolicy {
	return NewCharPolicy(Substitution{From: 'е', To: 'ё'})
}

// Substitute возвращает замену для руны r, если она есть в политике.
func (p *CharPolicy) Substitute(r rune) (rune, bool) {
	if p == nil {
		return r, false
	}
	for _, s := range p.Substitutions {
		if s.From == r {
			return s.To, true
		}
	}
	return r, false
}
