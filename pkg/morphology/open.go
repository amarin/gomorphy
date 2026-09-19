package morphology

import (
	"fmt"
	"os"

	"github.com/amarin/gomorphy/internal/mmapx"
	"github.com/amarin/gomorphy/pkg/morphology/importers/opencorpora"
	"github.com/amarin/gomorphy/pkg/morphology/importers/pymorphy2"
	"github.com/amarin/gomorphy/pkg/morphology/importers/unimorph"
	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// UniMorphOptions configures CompileFromUniMorph/CompileFromUniMorphFile
// (and their Dense variants) — an alias for unimorph.Options so callers
// don't need to import pkg/morphology/importers/unimorph directly.
type UniMorphOptions = unimorph.Options

// OpenPyMorphy loads a pymorphy2 dictionary from a directory (a direct read
// of pymorphy2-format files) into an immutable Dictionary.
func OpenPyMorphy(dir string) (*Dictionary, error) {
	d, err := pymorphy2.ImportFromDir(dir)
	if err != nil {
		return nil, err
	}
	return &Dictionary{d: d}, nil
}

// OpenPyMorphyDense is like OpenPyMorphy, but recompiles words.dawg to a
// dense 1-byte alphabet before wrapping the dictionary into a Dictionary
// (see pymorphy2.RecompileDense and
// docs/en/superpowers/specs/2026-09-16-pymorphy2-dense-recompile-design.md).
func OpenPyMorphyDense(dir string) (*Dictionary, error) {
	d, err := pymorphy2.RecompileDense(dir)
	if err != nil {
		return nil, err
	}
	return &Dictionary{d: d}, nil
}

// CompileFromXML compiles an OpenCorpora dictionary from dict.xml.
// progress is an optional callback for reporting progress.
func CompileFromXML(r interface{ Read([]byte) (int, error) }, progress opencorpora.Progress) (*Dictionary, error) {
	d, err := opencorpora.CompileFromXML(r, progress)
	if err != nil {
		return nil, err
	}
	return &Dictionary{d: d}, nil
}

// CompileFromXMLFile opens path (dict.xml) and calls CompileFromXML.
func CompileFromXMLFile(path string, progress opencorpora.Progress) (*Dictionary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("morphology: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return CompileFromXML(f, progress)
}

// CompileFromXMLDense is like CompileFromXML, but recompiles every
// shard's words.dawg to one dense 1-byte alphabet shared across the whole
// dictionary before wrapping it into a Dictionary (see
// internal.RecompileDense and
// docs/en/implementation/pymorphy2-dense-alphabet.md).
func CompileFromXMLDense(r interface{ Read([]byte) (int, error) }, progress opencorpora.Progress) (*Dictionary, error) {
	d, err := opencorpora.CompileFromXML(r, progress)
	if err != nil {
		return nil, err
	}
	if err := internal.RecompileDense(d); err != nil {
		return nil, fmt.Errorf("morphology: compile dense: %w", err)
	}
	return &Dictionary{d: d}, nil
}

// CompileFromXMLFileDense opens path (dict.xml) and calls
// CompileFromXMLDense.
func CompileFromXMLFileDense(path string, progress opencorpora.Progress) (*Dictionary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("morphology: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return CompileFromXMLDense(f, progress)
}

// CompileFromUniMorph compiles a UniMorph dictionary from a TSV stream
// (lemma<TAB>wordform<TAB>bundle). See unimorph.Options and
// docs/en/implementation/stage-16-import-unimorph.md.
func CompileFromUniMorph(r interface{ Read([]byte) (int, error) }, opts UniMorphOptions) (*Dictionary, error) {
	d, err := unimorph.CompileFromTSV(r, opts)
	if err != nil {
		return nil, err
	}
	return &Dictionary{d: d}, nil
}

// CompileFromUniMorphFile opens path (a UniMorph TSV) and calls
// CompileFromUniMorph.
func CompileFromUniMorphFile(path string, opts UniMorphOptions) (*Dictionary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("morphology: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return CompileFromUniMorph(f, opts)
}

// CompileFromUniMorphDense is like CompileFromUniMorph, but recompiles
// every shard's words.dawg to one dense 1-byte alphabet shared across
// the whole dictionary before wrapping it into a Dictionary (see
// internal.RecompileDense and
// docs/en/implementation/pymorphy2-dense-alphabet.md — the same
// source-agnostic mechanism CompileFromXMLDense and OpenPyMorphyDense
// use).
func CompileFromUniMorphDense(r interface{ Read([]byte) (int, error) }, opts UniMorphOptions) (*Dictionary, error) {
	d, err := unimorph.CompileFromTSV(r, opts)
	if err != nil {
		return nil, err
	}
	if err := internal.RecompileDense(d); err != nil {
		return nil, fmt.Errorf("morphology: compile dense: %w", err)
	}
	return &Dictionary{d: d}, nil
}

// CompileFromUniMorphFileDense opens path (a UniMorph TSV) and calls
// CompileFromUniMorphDense.
func CompileFromUniMorphFileDense(path string, opts UniMorphOptions) (*Dictionary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("morphology: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return CompileFromUniMorphDense(f, opts)
}

// Language returns the dictionary's language code.
func (x *Dictionary) Language() string { return x.d.Language }

// TagSetName returns the dictionary's TagSet.Name — the dictName
// pkg/morphology/tagmap.Map expects as its first argument to normalize
// this dictionary's tags — or "" if the dictionary has no TagSet (only
// possible for a Builder-assembled dictionary that never registered
// one; every importer sets one). See
// docs/en/implementation/tag-mapping.md's "known gap" note.
func (x *Dictionary) TagSetName() string {
	if x == nil || x.d == nil || x.d.TagSet == nil {
		return ""
	}
	return x.d.TagSet.Name
}

// Open loads a dictionary from a GMOR file (the single on-disk format,
// SaveTo). Hot sections (words.dawg) are mapped via mmap without copying;
// the result must be closed with the Close method.
func Open(path string) (*Dictionary, error) {
	mm, err := mmapx.Open(path)
	if err != nil {
		return nil, fmt.Errorf("morphology: open %s: %w", path, err)
	}

	cont, err := internal.OpenContainer(mm.Bytes())
	if err != nil {
		_ = mm.Close()
		return nil, fmt.Errorf("morphology: open %s: %w", path, err)
	}

	d, err := parseContainer(cont)
	if err != nil {
		_ = mm.Close()
		return nil, fmt.Errorf("morphology: open %s: %w", path, err)
	}
	return &Dictionary{d: d, mm: mm}, nil
}

// Close releases the resources of a dictionary opened via Open (the mmap
// region). It is a no-op for importer dictionaries. The dictionary must
// not be used after Close.
func (x *Dictionary) Close() error {
	if x == nil || x.mm == nil {
		return nil
	}
	err := x.mm.Close()
	x.mm = nil
	return err
}

// parseContainer assembles the internal dictionary from a GMOR file's
// sections.
func parseContainer(cont *internal.Container) (*internal.Dictionary, error) {
	meta, _, err := cont.Section("meta")
	if err != nil {
		return nil, err
	}
	language, policy, err := internal.DecodeMeta(meta)
	if err != nil {
		return nil, err
	}

	data, _, err := cont.Section("tagset")
	if err != nil {
		return nil, err
	}
	tagSet, err := internal.DecodeTagSet(data)
	if err != nil {
		return nil, err
	}

	data, _, err = cont.Section("prefixes")
	if err != nil {
		return nil, err
	}
	prefixes, err := internal.DecodeStrings(data)
	if err != nil {
		return nil, err
	}

	var suffixes [][]string
	for i := 0; ; i++ {
		data, _, err := cont.Section(fmt.Sprintf("suffixes-%d", i))
		if err != nil {
			break
		}
		s, err := internal.DecodeStrings(data)
		if err != nil {
			return nil, err
		}
		suffixes = append(suffixes, s)
	}
	if len(suffixes) == 0 {
		return nil, fmt.Errorf("morphology: no suffixes-N sections found")
	}

	var paradigms [][]internal.Paradigm
	for i := 0; ; i++ {
		data, _, err := cont.Section(fmt.Sprintf("paradigms-%d", i))
		if err != nil {
			break
		}
		p, err := internal.DecodeParadigms(data)
		if err != nil {
			return nil, err
		}
		paradigms = append(paradigms, p)
	}
	if len(paradigms) != len(suffixes) {
		return nil, fmt.Errorf("morphology: %d suffixes-N sections but %d paradigms-N sections", len(suffixes), len(paradigms))
	}

	var words []*internal.DAWG
	for i := 0; ; i++ {
		data, _, err := cont.Section(fmt.Sprintf("words.dawg-%d", i))
		if err != nil {
			break
		}
		w, err := internal.ParseDAWG(data)
		if err != nil {
			return nil, err
		}
		words = append(words, w)
	}
	if len(words) != len(suffixes) {
		return nil, fmt.Errorf("morphology: %d suffixes-N sections but %d words.dawg-N sections", len(suffixes), len(words))
	}

	d := internal.NewDictionary(language, tagSet, suffixes, prefixes, paradigms, words, policy)

	for i := 0; ; i++ {
		predData, _, err := cont.Section(fmt.Sprintf("prediction-%d", i))
		if err != nil {
			break
		}
		pred, err := internal.ParseDAWG(predData)
		if err != nil {
			return nil, err
		}
		d.Prediction = append(d.Prediction, pred)
	}

	if probData, _, err := cont.Section("probability"); err == nil {
		prob, err := internal.ParseDAWG(probData)
		if err != nil {
			return nil, err
		}
		d.Probability = prob
	}

	if infoData, _, err := cont.Section("info"); err == nil {
		info, err := internal.DecodeBuildInfo(infoData)
		if err != nil {
			return nil, err
		}
		d.Info = info
	}

	if alphabetData, _, err := cont.Section("alphabet"); err == nil {
		alphabet, err := internal.DecodeAlphabet(alphabetData)
		if err != nil {
			return nil, err
		}
		d.Alphabet = alphabet
	}

	return d, nil
}
