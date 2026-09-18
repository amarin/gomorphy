package internal

// Substitution — a pair of characters interchangeable during search (e.g. е→ё).
type Substitution struct {
	From rune
	To   rune
}

// CharPolicy — a set of character substitutions for search. Empty by
// default (language-neutral); for Russian, the е→ё substitution is configured.
type CharPolicy struct {
	Substitutions []Substitution
}

// NewCharPolicy creates a CharPolicy from substitution pairs.
func NewCharPolicy(subs ...Substitution) *CharPolicy {
	return &CharPolicy{Substitutions: subs}
}

// RussianCharPolicy — the policy for Russian: е→ё.
func RussianCharPolicy() *CharPolicy {
	return NewCharPolicy(Substitution{From: 'е', To: 'ё'})
}

// Substitute returns the replacement for rune r, if the policy has one.
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
