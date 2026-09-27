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
// don't need to import pkg/morphology/importers/unimorph directly. Its
// CharPolicy field takes a *CharPolicy (NoCharPolicy, RussianCharPolicy,
// NewCharPolicy); nil means the language default.
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

// XMLOptions configures CompileFromXMLWithOptions.
type XMLOptions struct {
	// Progress reports import progress; nil means no reporting.
	Progress opencorpora.Progress
	// Dense recompiles every shard's words DAWG to one dense 1-byte
	// alphabet (see CompileFromXMLDense).
	Dense bool
	// NoPrediction skips the ending-based prediction for
	// out-of-dictionary words, built by default: Parse then returns nil
	// for a word the dictionary does not contain. Prediction is pruned
	// like pymorphy2's (paradigms of at least 3 lemmas, endings attested
	// at least twice, the most attested form per ending and part of
	// speech).
	NoPrediction bool
}

// CompileFromXMLWithOptions compiles an OpenCorpora dictionary from
// dict.xml. Unless opts.NoPrediction is set it builds ending-based
// prediction, so Parse returns Predicted readings for unknown words.
// Prediction is pruned like pymorphy2's (paradigms of at least 3 lemmas,
// endings attested at least twice, the most attested form per ending and
// part of speech).
func CompileFromXMLWithOptions(r interface{ Read([]byte) (int, error) }, opts XMLOptions) (*Dictionary, error) {
	d, err := opencorpora.CompileFromXML(r, opts.Progress)
	if err != nil {
		return nil, err
	}
	return finishCompiled(d, opts.Dense, !opts.NoPrediction)
}

// CompileFromXML compiles an OpenCorpora dictionary from dict.xml, with
// prediction; progress is an optional callback for reporting progress.
// See CompileFromXMLWithOptions.
func CompileFromXML(r interface{ Read([]byte) (int, error) }, progress opencorpora.Progress) (*Dictionary, error) {
	return CompileFromXMLWithOptions(r, XMLOptions{Progress: progress})
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
// docs/en/implementation/pymorphy2-dense-alphabet.md). Builds prediction
// unless told otherwise, see CompileFromXMLWithOptions.
func CompileFromXMLDense(r interface{ Read([]byte) (int, error) }, progress opencorpora.Progress) (*Dictionary, error) {
	return CompileFromXMLWithOptions(r, XMLOptions{Progress: progress, Dense: true})
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
// (lemma<TAB>wordform<TAB>bundle), with prediction. See unimorph.Options
// and docs/en/implementation/stage-16-import-unimorph.md. Builds
// prediction unless told otherwise, see CompileFromXMLWithOptions.
func CompileFromUniMorph(r interface{ Read([]byte) (int, error) }, opts UniMorphOptions) (*Dictionary, error) {
	d, err := unimorph.CompileFromTSV(r, opts)
	if err != nil {
		return nil, err
	}
	return finishCompiled(d, false, !opts.NoPrediction)
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
// use). Builds prediction unless told otherwise, see
// CompileFromXMLWithOptions.
func CompileFromUniMorphDense(r interface{ Read([]byte) (int, error) }, opts UniMorphOptions) (*Dictionary, error) {
	d, err := unimorph.CompileFromTSV(r, opts)
	if err != nil {
		return nil, err
	}
	return finishCompiled(d, true, !opts.NoPrediction)
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

// finishCompiled builds prediction (unless prediction is false) and then,
// if dense is set, recompiles the words DAWGs to a dense alphabet — the
// same order Builder uses.
func finishCompiled(d *internal.Dictionary, dense, prediction bool) (*Dictionary, error) {
	if prediction {
		if err := internal.BuildPredictionPruned(d, productive, internal.ImportPredictionPruning); err != nil {
			return nil, fmt.Errorf("morphology: build prediction: %w", err)
		}
	}
	if dense {
		if err := internal.RecompileDense(d); err != nil {
			return nil, fmt.Errorf("morphology: compile dense: %w", err)
		}
	}
	return &Dictionary{d: d}, nil
}

// Language returns the dictionary's language code ("ru" for every bundled
// importer; BuilderOptions.Language for Builder/ImportTSV, "ru" when
// empty), or "" for a nil dictionary.
func (x *Dictionary) Language() string {
	if x == nil || x.d == nil {
		return ""
	}
	return x.d.Language
}

// TagSetName returns the dictionary's TagSet.Name — the dictName
// pkg/morphology/tagmap.Map expects as its first argument to normalize
// this dictionary's tags — or "" for a nil dictionary. Importers set
// "opencorpora", "opencorpora-int" (pymorphy2) or "unimorph"; Builder and
// ImportTSV set "builder" and "tsv", which tagmap does not know
// (tagmap.Known reports false, tagmap.Map returns ok=false): their tags
// are the caller's own strings. Merge keeps the base's name.
func (x *Dictionary) TagSetName() string {
	if x == nil || x.d == nil || x.d.TagSet == nil {
		return ""
	}
	return x.d.TagSet.Name
}

// Open loads a dictionary from a GMOR file (the single on-disk format,
// SaveTo). Hot sections (words.dawg) are mapped via mmap without copying;
// the result must be closed with the Close method (see Close for the
// lifecycle rules). Not supported on Windows yet — use OpenBytes there.
func Open(path string) (*Dictionary, error) {
	mm, err := mmapx.Open(path)
	if err != nil {
		return nil, fmt.Errorf("morphology: open %s: %w", path, err)
	}
	d, err := openBytes(mm.Bytes())
	if err != nil {
		_ = mm.Close()
		return nil, fmt.Errorf("morphology: open %s: %w", path, err)
	}
	return &Dictionary{d: d, mm: mm}, nil
}

// OpenBytes opens a dictionary in the GMOR format (as written by SaveTo)
// from data — typically a file embedded with //go:embed. The checksum is
// verified like Open. data is not copied: DAWG sections whose unit array
// is 4-byte aligned in memory are used in place (zero-copy), misaligned
// ones are copied (//go:embed gives no alignment guarantee, so expect
// copies there). data must stay unmodified and reachable for as long as
// the Dictionary is used; Close releases nothing. Works on every platform,
// Windows included (no mmap involved).
func OpenBytes(data []byte) (*Dictionary, error) {
	d, err := openBytes(data)
	if err != nil {
		return nil, fmt.Errorf("morphology: open bytes: %w", err)
	}
	return &Dictionary{d: d}, nil
}

// openBytes validates a GMOR container (magic, version, checksum, catalog)
// and assembles the internal dictionary from its sections. The result may
// alias data (see internal.ParseDAWG).
func openBytes(data []byte) (*internal.Dictionary, error) {
	cont, err := internal.OpenContainer(data)
	if err != nil {
		return nil, err
	}
	return parseContainer(cont)
}

// Close releases the mmap region of a dictionary opened with Open. It is a
// no-op for dictionaries that are imported, built (Builder, ImportTSV,
// Merge) or opened with OpenBytes.
//
// Close must not be called while other goroutines may still call methods
// on the Dictionary (directly or through a MultiDictionary): in-flight
// Parse/ParseAppend/Lemma/IsKnown/Fuzzy/FuzzyTop/ContentHash calls read the
// mapping, and unmapping it under them crashes the process with SIGSEGV or
// SIGBUS — not a recoverable panic. Values already returned (Reading,
// LemmaRef, FuzzyMatch, BuildInfo and all their strings) are independent
// copies and stay valid after Close. A caller that swaps dictionaries at
// runtime must retire the old one only after its in-flight calls have
// finished (for example, hold a sync.RWMutex read lock around each call
// and take the write lock before Close). The dictionary must not be used
// after Close.
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

	plain, err := predictionSections(cont, "prediction-%d")
	if err != nil {
		return nil, err
	}
	sharded, err := predictionSections(cont, "pred-sharded-%d")
	if err != nil {
		return nil, err
	}
	switch {
	case len(plain) > 0 && len(sharded) > 0:
		return nil, fmt.Errorf("morphology: both prediction-N and pred-sharded-N sections present")
	case len(sharded) > 0:
		d.Prediction, d.PredictionSharded = sharded, true
	default:
		d.Prediction = plain
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

// predictionSections parses the consecutive sections named by format
// (N = 0, 1, …) up to the first missing one.
func predictionSections(cont *internal.Container, format string) ([]*internal.DAWG, error) {
	var out []*internal.DAWG
	for i := 0; ; i++ {
		data, _, err := cont.Section(fmt.Sprintf(format, i))
		if err != nil {
			return out, nil
		}
		pred, err := internal.ParseDAWG(data)
		if err != nil {
			return nil, err
		}
		out = append(out, pred)
	}
}
