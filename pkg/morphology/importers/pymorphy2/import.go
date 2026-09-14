// Package pymorphy2 загружает словарь pymorphy2 из директории.
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

// defaultPrefixes — префиксы paradigm-prefixes.json для русских словарей
// (используются, если файл отсутствует).
var defaultPrefixes = []string{"", "по", "наи"}

// ImportFromDir загружает словарь pymorphy2 из директории: words.dawg,
// paradigms.array, suffixes.json, paradigm-prefixes.json,
// gramtab-opencorpora-int.json. Опционально — p_t_given_w.intdawg
// и prediction-suffixes-{i}.dawg. Чтение прямое, без конвертации.
func ImportFromDir(dir string) (*internal.Dictionary, error) {
	d := internal.NewDictionary(
		"ru",
		internal.NewTagSet("opencorpora-int"),
		nil, nil, nil, nil,
		internal.RussianCharPolicy(),
	)
	d.Info = &internal.BuildInfo{Source: "pymorphy2"}

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

// readStringArray читает JSON-массив строк (suffixes.json, gramtab-*.json).
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

// readDAWGFile читает DAWG из файла (формат words.dawg: dictionary + guide).
func readDAWGFile(path string) (*internal.DAWG, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return internal.ReadDAWG(f)
}

// readParadigms читает paradigms.array: uint16 count + для каждого парадигмы
// uint16 len + len×uint16 (LE).
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
