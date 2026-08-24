// Package build provides the mutable dictionary Builder (FT8) and compiles
// accumulated data into an immutable CSR-based runtime Snapshot.
//
// The Builder is intentionally simple and allocation-tolerant: plain maps and
// node-struct tries serve as scratch space, Build() then compiles everything
// into flat value-only arrays suitable for zero-copy loading (stage 5).
package build

import (
	"errors"
	"strconv"
	"strings"

	"github.com/amarin/gomorphy/internal/intern"
	"github.com/amarin/gomorphy/internal/stringsx"
)

var (
	ErrEmptyText     = errors.New("build: empty text")
	ErrUnknownLemma  = errors.New("build: unknown lemma")
	ErrEmptyGrammeme = errors.New("build: empty grammeme name")
)

// Builder accumulates grammemes, lemmas and wordforms (FT8).
// It is NOT safe for concurrent use. Build() returns an immutable Snapshot;
// the Builder stays usable afterwards for further additions and rebuilds.
type Builder struct {
	texts   *stringsx.Arena
	textIdx *intern.Table

	grammemes []string          // id -> name
	gramIdx   map[string]uint32 // name -> id

	ancodeIdx  map[string]uint32 // canonical "id,id,id" -> ancode id
	ancodeKeys [][]uint32        // ancode id -> grammeme ids (ascending)

	lemmaTexts []uint32 // dense lemma id -> text id
	// lemmaAncodes holds the base-form ancode of each lemma (dense id).
	lemmaAncodes []uint32

	pairIdx     map[uint64]uint32 // (textID<<32|ancodeID) -> pair id
	pairTexts   []uint32
	pairAncodes []uint32
	attPair     []uint32 // attach log: pair id per attach call
	attLemma    []uint32 // attach log: lemma per attach call

	nodes []*tnode // text id -> terminal trie node, nil until first insert
	root  *tnode
}

// NewBuilder creates an empty Builder.
func NewBuilder() *Builder {
	return NewBuilderWithCapacity(1<<16, 1<<16, 1<<18)
}

// NewBuilderWithCapacity pre-sizes internal structures for expected volumes.
func NewBuilderWithCapacity(texts, lemmas, pairs int) *Builder {
	arena := stringsx.New(64*texts, texts)

	return &Builder{
		texts:      arena,
		textIdx:    intern.New(arena, texts),
		gramIdx:    make(map[string]uint32),
		ancodeIdx:  make(map[string]uint32),
		pairIdx:    make(map[uint64]uint32, pairs),
		root:       &tnode{kids: make(map[byte]*tnode)},
		lemmaTexts: make([]uint32, 0, lemmas),
	}
}

// AddGrammeme registers a grammeme by name and returns its dense id.
func (b *Builder) AddGrammeme(name string) (uint32, error) {
	if name == "" {
		return 0, ErrEmptyGrammeme
	}

	if id, ok := b.gramIdx[name]; ok {
		return id, nil
	}

	id := uint32(len(b.grammemes))
	b.grammemes = append(b.grammemes, name)
	b.gramIdx[name] = id

	return id, nil
}

// AddLemma adds a lemma together with its base-form word. Grammemes describe
// the base-form ancode and may be empty. Returns the dense lemma id.
func (b *Builder) AddLemma(text string, grammemes ...string) (int, error) {
	if text == "" {
		return 0, ErrEmptyText
	}

	ancode, err := b.ensureAncode(grammemes)
	if err != nil {
		return 0, err
	}

	lemma := len(b.lemmaTexts)
	textID := b.internText(text)

	b.lemmaTexts = append(b.lemmaTexts, textID)
	b.lemmaAncodes = append(b.lemmaAncodes, ancode)

	if err := b.attachPair(textID, ancode, uint32(lemma)); err != nil {
		return 0, err
	}

	return lemma, nil
}

// AddForm attaches a wordform to a lemma previously returned by AddLemma.
func (b *Builder) AddForm(lemma int, text string, grammemes ...string) error {
	if text == "" {
		return ErrEmptyText
	}

	if lemma < 0 || lemma >= len(b.lemmaTexts) {
		return ErrUnknownLemma
	}

	textID := b.internText(text)

	ancode, err := b.ensureAncode(grammemes)
	if err != nil {
		return err
	}

	return b.attachPair(textID, ancode, uint32(lemma))
}

func (b *Builder) internText(text string) uint32 {
	id, _ := b.textIdx.Intern([]byte(text))

	for int(id) >= len(b.nodes) {
		b.nodes = append(b.nodes, nil)
	}

	return id
}

// ensureAncode resolves or creates the ancode for a grammeme name list.
// Grammeme order is significant (OpenCorpora treats ordered sets as distinct);
// the empty set is a valid ancode and gets its own id.
func (b *Builder) ensureAncode(names []string) (uint32, error) {
	if len(names) > 255 {
		return 0, errors.New("build: too many grammemes in one ancode")
	}

	var (
		sb  strings.Builder
		ids []uint32
	)

	for i, n := range names {
		id, err := b.AddGrammeme(n)
		if err != nil {
			return 0, err
		}

		if i > 0 {
			sb.WriteByte(',')
		}

		sb.WriteString(strconv.FormatUint(uint64(id), 10))
		ids = append(ids, id)
	}

	key := sb.String()

	if existing, ok := b.ancodeIdx[key]; ok {
		return existing, nil
	}

	id := uint32(len(b.ancodeKeys))
	b.ancodeIdx[key] = id
	b.ancodeKeys = append(b.ancodeKeys, ids)

	return id, nil
}
