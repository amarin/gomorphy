package dictionary

import (
	"github.com/amarin/gomorphy/internal/build"
)

// NewBuilder creates an empty Builder for programmatic dictionary creation
// (FT8). Compile produces an immutable Dictionary.
type Builder struct {
	inner *build.Builder
}

// NewBuilder returns a fresh empty Builder.
func NewBuilder() *Builder {
	return &Builder{inner: build.NewBuilder()}
}

// AddGrammeme registers a grammeme name, returning its dense id.
func (b *Builder) AddGrammeme(name string) (uint32, error) {
	return b.inner.AddGrammeme(name)
}

// AddLemma adds a lemma with its citation form and base-form grammemes.
func (b *Builder) AddLemma(text string, grammemes ...string) (int, error) {
	return b.inner.AddLemma(text, grammemes...)
}

// AddForm attaches a wordform to a lemma returned earlier by AddLemma.
func (b *Builder) AddForm(lemma int, text string, grammemes ...string) error {
	return b.inner.AddForm(lemma, text, grammemes...)
}

// Compile builds the immutable Dictionary snapshot. The Builder stays usable
// for further additions and recompiles.
func (b *Builder) Compile() *Dictionary {
	return &Dictionary{snap: b.inner.Build()}
}
