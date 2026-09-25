// Package pymorphy2 loads a pymorphy2 dictionary from a directory.
//
// Unlike the OpenCorpora/UniMorph importers, this one reads words.dawg and
// the other dictionary files as pre-built binary blobs (ReadDAWG,
// readParadigms) rather than raw word strings — there is no string-level
// entry point here to lower-case, so the mixed-case-unreachable class of
// bug (see the UniMorph importer's case handling) does not apply: any
// case normalization has to happen upstream, in the pymorphy2/Python
// compiler that produced words.dawg.
package pymorphy2

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// defaultPrefixes — the paradigm-prefixes.json prefixes for Russian
// dictionaries (used when the file is absent).
var defaultPrefixes = []string{"", "по", "наи"}

// ImportFromDir loads a pymorphy2 dictionary from a directory: words.dawg,
// paradigms.array, suffixes.json, paradigm-prefixes.json,
// gramtab-opencorpora-int.json. Optional: p_t_given_w.intdawg
// and prediction-suffixes-{i}.dawg. Reads directly, without conversion.
func ImportFromDir(dir string) (*internal.Dictionary, error) {
	d := internal.NewDictionary(
		"ru",
		internal.NewTagSet("opencorpora-int"),
		nil, nil, nil, nil,
		internal.RussianCharPolicy(),
	)
	d.Info = &internal.BuildInfo{Source: "pymorphy2"}
	sourceVersion, err := readMetaSourceVersion(filepath.Join(dir, "meta.json"))
	if err != nil {
		return nil, fmt.Errorf("pymorphy2: meta.json: %w", err)
	}
	d.Info.SourceVersion = sourceVersion

	tags, err := readStringArray(filepath.Join(dir, "gramtab-opencorpora-int.json"))
	if err != nil {
		return nil, fmt.Errorf("pymorphy2: gramtab: %w", err)
	}
	for _, tag := range tags {
		if _, err := d.TagSet.Add(tag); err != nil {
			return nil, fmt.Errorf("pymorphy2: gramtab: %w", err)
		}
	}

	prefixes, err := readStringArray(filepath.Join(dir, "paradigm-prefixes.json"))
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("pymorphy2: paradigm-prefixes: %w", err)
		}
		prefixes = defaultPrefixes
	}
	d.Prefixes = prefixes

	suffixes, err := readStringArray(filepath.Join(dir, "suffixes.json"))
	if err != nil {
		return nil, fmt.Errorf("pymorphy2: suffixes: %w", err)
	}
	d.Suffixes = [][]string{suffixes}

	paradigms, err := readParadigms(filepath.Join(dir, "paradigms.array"))
	if err != nil {
		return nil, fmt.Errorf("pymorphy2: paradigms: %w", err)
	}
	d.Paradigms = [][]internal.Paradigm{paradigms}

	words, err := readDAWGFile(filepath.Join(dir, "words.dawg"))
	if err != nil {
		return nil, fmt.Errorf("pymorphy2: words.dawg: %w", err)
	}
	d.Words = []*internal.DAWG{words}

	if prob, err := readDAWGFile(filepath.Join(dir, "p_t_given_w.intdawg")); err == nil {
		d.Probability = prob
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("pymorphy2: p_t_given_w.intdawg: %w", err)
	}

	for i := 0; i < len(prefixes); i++ {
		pred, err := readDAWGFile(filepath.Join(dir, fmt.Sprintf("prediction-suffixes-%d.dawg", i)))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				break
			}
			return nil, fmt.Errorf("pymorphy2: prediction-suffixes-%d.dawg: %w", i, err)
		}
		d.Prediction = append(d.Prediction, pred)
	}

	return d, nil
}

// readMetaSourceVersion reads meta.json (pymorphy2's format: a list of
// [key, value] pairs instead of an object, with values being a mix of
// strings and numbers) and returns "<source_version>/<source_revision>" if
// both keys are found and are strings; otherwise "" (including when
// meta.json is absent — not every pymorphy2 dictionary carries one).
func readMetaSourceVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	var pairs [][]json.RawMessage
	if err := json.Unmarshal(data, &pairs); err != nil {
		return "", fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}

	var version, revision string
	for _, pair := range pairs {
		if len(pair) != 2 {
			continue
		}
		var key string
		if err := json.Unmarshal(pair[0], &key); err != nil {
			continue
		}
		var value string
		if err := json.Unmarshal(pair[1], &value); err != nil {
			continue // e.g. a numeric value - not something we need here
		}
		switch key {
		case "source_version":
			version = value
		case "source_revision":
			revision = value
		}
	}

	if version == "" && revision == "" {
		return "", nil
	}
	return fmt.Sprintf("%s/%s", version, revision), nil
}

// readStringArray reads a JSON array of strings (suffixes.json, gramtab-*.json).
func readStringArray(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ss []string
	if err := json.Unmarshal(data, &ss); err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return ss, nil
}

// readDAWGFile reads a DAWG from a file (words.dawg format: dictionary + guide).
func readDAWGFile(path string) (*internal.DAWG, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return internal.ReadDAWG(f)
}

// readParadigms reads paradigms.array: uint16 count + for each paradigm
// a uint16 len + len×uint16 (LE).
func readParadigms(path string) ([]internal.Paradigm, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(data)

	var count uint16
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	paradigms := make([]internal.Paradigm, 0, count)
	for i := 0; i < int(count); i++ {
		var size uint16
		if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
			return nil, err
		}
		raw := make([]uint16, size)
		if err := binary.Read(r, binary.LittleEndian, raw); err != nil {
			return nil, err
		}
		p, err := internal.NewParadigmFromData(raw)
		if err != nil {
			return nil, err
		}
		paradigms = append(paradigms, p)
	}
	return paradigms, nil
}
