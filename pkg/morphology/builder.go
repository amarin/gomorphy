package morphology

import (
	"errors"
	"fmt"
	"strings"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// Sentinel errors of the public Builder API.
var (
	// ErrNoEntries is returned by Builder.Build when no entries have been
	// registered.
	ErrNoEntries = errors.New("morphology: builder: no entries")

	// ErrBuilderClosed is returned by Builder.AddForm (and repeated Build
	// calls) once the Builder has been consumed by a Build.
	ErrBuilderClosed = errors.New("morphology: builder is closed")
)

// builderTagSetName is the default TagSet name for dictionaries built
// through the public Builder.
const builderTagSetName = "builder"

// BuilderOptions — optional dictionary-level metadata for a built dictionary.
type BuilderOptions struct {
	// Language code for the dictionary's "meta" section (default "ru").
	Language string

	// Source fills BuildInfo.Source; defaults to "builder" (TSV import
	// defaults to "tsv" when empty). Should reflect what the entries came
	// from.
	Source string

	// CharPolicy is the lookup substitution policy stored in the built
	// dictionary. nil means the language default: е→ё for "ru" (and for an
	// empty Language, which means "ru"), no substitutions for any other
	// language. Use NoCharPolicy to disable substitutions explicitly, or
	// RussianCharPolicy to get е→ё for another language. A policy with
	// more than 255 substitutions is rejected by Build/ImportTSV with a
	// wrapped error — see CharPolicy's doc comment.
	CharPolicy *CharPolicy
}

// Builder accumulates (word, lemma, tag) triples and builds an immutable
// *Dictionary via Build. A Builder is single-use: once Build is called,
// further AddForm calls are rejected with ErrBuilderClosed.
type Builder struct {
	opts    BuilderOptions
	entries []internal.BuildEntry // insertion order, deduped
	seen    map[internal.BuildEntry]bool
	closed  bool
}

// NewBuilder returns a Builder that accumulates wordform entries.
func NewBuilder(opts BuilderOptions) *Builder {
	if opts.Language == "" {
		opts.Language = "ru"
	}
	if opts.Source == "" {
		opts.Source = "builder"
	}
	return &Builder{opts: opts}
}

// AddForm registers one entry: wordform text "word", its lemma "lemma",
// and an opaque grammeme tag "tag". tag may be "" (a reading with no
// grammemes). An empty or whitespace-only word is an error; otherwise word
// and lemma are not trimmed (unlike ImportTSV). An empty lemma means the
// wordform is its own lemma (auto-lemma). word and lemma are lower-cased
// (strings.ToLower): Parse lower-cases its input, so a mixed-case form
// would otherwise be unreachable. tag is stored verbatim.
func (b *Builder) AddForm(word, lemma, tag string) error {
	if b.closed {
		return ErrBuilderClosed
	}
	if strings.TrimSpace(word) == "" {
		return fmt.Errorf("morphology: builder: word must not be empty")
	}
	if lemma == "" {
		lemma = word
	}
	entry := internal.BuildEntry{Word: strings.ToLower(word), Lemma: strings.ToLower(lemma), Tag: tag}
	if b.seen == nil {
		b.seen = make(map[internal.BuildEntry]bool)
	}
	if b.seen[entry] {
		return nil
	}
	b.seen[entry] = true
	b.entries = append(b.entries, entry)
	return nil
}

// AddLemma is sugar for AddForm(normal, normal, tag): the lemma is also a
// wordform of itself.
func (b *Builder) AddLemma(normal, tag string) error {
	return b.AddForm(normal, normal, tag)
}

// Build assembles the dictionary from all registered entries, deduplicating
// (word, lemma, tag) triples by insertion order and rebuilding prediction.
// The result is a fully functional *Dictionary (Parse, Lemma, Fuzzy,
// prediction) usable directly or Savable via SaveTo. Build consumes the
// Builder even when it fails (for example, on a CharPolicy over the
// 255-substitution limit): later AddForm and Build calls return
// ErrBuilderClosed. Build with no entries, or on a nil Builder, returns
// ErrNoEntries and leaves the Builder open.
func (b *Builder) Build() (*Dictionary, error) {
	if b == nil || len(b.entries) == 0 {
		return nil, ErrNoEntries
	}
	if b.closed {
		return nil, ErrBuilderClosed
	}
	b.closed = true
	return buildFromEntries(b.opts, b.entries, builderTagSetName)
}

// buildFromEntries assembles an immutable *Dictionary from wordform entries
// through the shared internal pipeline — BuildDictionaryFromEntries (raw),
// BuildPrediction, RecompileDense (dense by default) — and stamps the
// BuildInfo. It is the shared post-process helper underneath
// Builder.Build, ImportTSV, and Merge.
func buildFromEntries(opts BuilderOptions, entries []internal.BuildEntry, tagSetName string) (*Dictionary, error) {
	language := opts.Language
	if language == "" {
		language = "ru" // ImportTSV does not default Language; NewBuilder does
	}
	policy := opts.CharPolicy
	if policy == nil {
		policy = defaultCharPolicy(language)
	}
	if err := internal.ValidateCharPolicy(policy); err != nil {
		return nil, fmt.Errorf("morphology: build: %w", err)
	}
	d, err := internal.BuildDictionaryFromEntries(internal.BuildOptions{
		Language:   language,
		CharPolicy: policy,
		TagSetName: tagSetName,
	}, entries)
	if err != nil {
		return nil, fmt.Errorf("morphology: build: %w", err)
	}

	if err := internal.BuildPrediction(d, productive); err != nil {
		return nil, fmt.Errorf("morphology: build prediction: %w", err)
	}

	if err := internal.RecompileDense(d); err != nil {
		return nil, fmt.Errorf("morphology: compile dense: %w", err)
	}

	source := opts.Source
	if source == "" {
		source = tagSetName
	}
	d.Info = &internal.BuildInfo{Source: source}

	return &Dictionary{d: d}, nil
}
