package morphology

import "github.com/amarin/gomorphy/pkg/morphology/internal"

// Substitution is a pair of runes lookup treats as interchangeable in one
// direction: a query rune From also matches a stored rune To. For Russian
// that is е→ё, so «елка» finds «ёлка» (but «ёлка» does not find «елка»).
//
// It is an alias of the internal type so that every options struct that
// already carries a policy (UniMorphOptions.CharPolicy) accepts it as is.
type Substitution = internal.Substitution

// CharPolicy is a dictionary's set of lookup substitutions. It is stored in
// the dictionary ("meta" section) when the dictionary is built and applied
// by Parse, Lemma, IsKnown, Fuzzy and FuzzyTop. Pre-reform orthography
// (ѣ→е, final ъ) is not a CharPolicy concern: normalize such text before
// calling gomorphy.
type CharPolicy = internal.CharPolicy

// NewCharPolicy returns a policy with the given substitutions.
func NewCharPolicy(subs ...Substitution) *CharPolicy { return internal.NewCharPolicy(subs...) }

// RussianCharPolicy returns the Russian policy: е→ё.
func RussianCharPolicy() *CharPolicy { return internal.RussianCharPolicy() }

// NoCharPolicy returns an explicit empty policy: exact rune matching only.
// Unlike a nil *CharPolicy in options, which means "the language default",
// it disables substitutions.
func NoCharPolicy() *CharPolicy { return internal.NewCharPolicy() }

// defaultCharPolicy is the policy Builder and ImportTSV apply when the
// caller leaves BuilderOptions.CharPolicy nil: е→ё for Russian ("ru", and
// "" which means "ru"), no substitutions for any other language.
func defaultCharPolicy(language string) *CharPolicy {
	switch language {
	case "", "ru":
		return RussianCharPolicy()
	default:
		return NoCharPolicy()
	}
}
