package internal

import "fmt"

// Substitution — a pair of characters interchangeable during search (e.g. е→ё).
type Substitution struct {
	From rune
	To   rune
}

// CharPolicy — a set of character substitutions for search. Empty by
// default (language-neutral); for Russian, the е→ё substitution is
// configured. Substitutions is capped at MaxCharPolicySubstitutions: the
// on-disk "meta" section (EncodeMeta, format.go) stores the count in one
// byte. Build a CharPolicy through the public morphology package (Builder,
// ImportTSV, UniMorphOptions) and the cap is enforced with a wrapped error
// before the dictionary is built; constructing one directly and exceeding
// the cap is only caught later, at SaveTo/ContentHash time, as a panic.
type CharPolicy struct {
	Substitutions []Substitution
}

// MaxCharPolicySubstitutions is the largest number of substitutions a
// CharPolicy may hold — the on-disk "meta" section encodes the count in a
// single byte (EncodeMeta, format.go). ValidateCharPolicy enforces this;
// exceeding it there is a build-time error, not the panic EncodeMeta would
// otherwise raise at save time.
const MaxCharPolicySubstitutions = 255

// ValidateCharPolicy rejects a CharPolicy with more than
// MaxCharPolicySubstitutions substitutions. nil is always valid (no
// substitutions). Callers that accept a caller-supplied CharPolicy before
// building a dictionary (buildFromEntries, unimorph.ImportFromTSV) must
// call this so an oversized policy fails with a wrapped error instead of
// panicking later inside EncodeMeta (SaveTo/ContentHash).
func ValidateCharPolicy(p *CharPolicy) error {
	if p == nil {
		return nil
	}
	if len(p.Substitutions) > MaxCharPolicySubstitutions {
		return fmt.Errorf("internal: charpolicy: %d substitutions exceeds the %d limit", len(p.Substitutions), MaxCharPolicySubstitutions)
	}
	return nil
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
