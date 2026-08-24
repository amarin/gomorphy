package dictionary

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/amarin/gomorphy/internal/build"
	"github.com/amarin/gomorphy/internal/xmlscan"
)

// Expected volumes of the full OpenCorpora dictionary, used to pre-size
// builder structures and avoid most reallocation during compilation.
const (
	expectedTexts  = 3_100_000
	expectedLemmas = 400_000
	expectedPairs  = 5_500_000
)

// xmlFeed adapts xmlscan events to Builder calls.
type xmlFeed struct {
	b        *build.Builder
	curText  string
	curLemma int
	lGrams   []string
	fGrams   []string
	haveForm bool
	err      error
}

func (f *xmlFeed) fail(err error) {
	if f.err == nil {
		f.err = err
	}
}

func (f *xmlFeed) OnGrammeme(_, name []byte) error {
	if len(name) > 0 {
		if _, err := f.b.AddGrammeme(string(name)); err != nil {
			f.fail(err)
		}
	}

	return nil
}

func (f *xmlFeed) OnGrammemeRef(v []byte) error {
	if f.haveForm {
		f.fGrams = append(f.fGrams, string(v))
	} else {
		f.lGrams = append(f.lGrams, string(v))
	}

	return nil
}

func (f *xmlFeed) OnLemma(_ uint32, text []byte) error {
	f.curText = string(text)
	f.lGrams = f.lGrams[:0]

	return nil
}

func (f *xmlFeed) OnLemmaEnd() error {
	id, err := f.b.AddLemma(f.curText, f.lGrams...)
	if err != nil {
		f.fail(err)

		return nil
	}

	f.curLemma = id

	return nil
}

func (f *xmlFeed) OnForm(text []byte) error {
	f.curText = string(text)
	f.fGrams = f.fGrams[:0]
	f.haveForm = true

	return nil
}

func (f *xmlFeed) OnFormEnd() error {
	if err := f.b.AddForm(f.curLemma, f.curText, f.fGrams...); err != nil {
		f.fail(err)
	}

	f.haveForm = false

	return nil
}

// CompileFromXML streams an OpenCorpora dict.xml stream and compiles it into
// a Dictionary snapshot (FT7). The input is read once; memory use is bounded
// by the builder scratch space, not by the file size.
func CompileFromXML(r io.Reader) (*Dictionary, error) {
	b := build.NewBuilderWithCapacity(expectedTexts, expectedLemmas, expectedPairs)

	feed := &xmlFeed{b: b}

	if err := xmlscan.New(r, feed).Scan(); err != nil {
		return nil, fmt.Errorf("dictionary: scan dict.xml: %w", err)
	}

	if feed.err != nil {
		return nil, fmt.Errorf("dictionary: build: %w", feed.err)
	}

	return &Dictionary{snap: b.Build()}, nil
}

// CompileFromXMLFile opens path and calls CompileFromXML.
func CompileFromXMLFile(path string) (*Dictionary, error) {
	f, err := os.Open(path) //nolint:gosec // path comes from caller by design
	if err != nil {
		return nil, fmt.Errorf("dictionary: open %s: %w", path, err)
	}

	defer func() { _ = f.Close() }()

	return CompileFromXML(bufReader(f))
}

func bufReader(f *os.File) io.Reader { return f }

// SaveToAtomic writes the snapshot to path via a temporary file in the same
// directory followed by rename, so readers never observe a partial file.
func (d *Dictionary) SaveToAtomic(path string) error {
	tmp, err := os.CreateTemp(dirOf(path), "."+baseOf(path)+".tmp*")
	if err != nil {
		return fmt.Errorf("dictionary: temp file: %w", err)
	}

	tmpName := tmp.Name()

	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("dictionary: temp file: %w", err)
	}

	if err := d.SaveTo(tmpName); err != nil {
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("dictionary: rename %s -> %s: %w", tmpName, path, err)
	}

	return nil
}

func dirOf(path string) string {
	if i := strings.LastIndexByte(path, '/'); i > 0 {
		return path[:i]
	}

	return "."
}

func baseOf(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}

	return path
}
