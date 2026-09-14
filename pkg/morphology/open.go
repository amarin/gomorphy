package morphology

import (
	"fmt"
	"os"

	"github.com/amarin/gomorphy/internal/mmapx"
	"github.com/amarin/gomorphy/pkg/morphology/importers/opencorpora"
	"github.com/amarin/gomorphy/pkg/morphology/importers/pymorphy2"
	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// OpenPyMorphy загружает словарь pymorphy2 из директории (прямое чтение
// файлов формата pymorphy2) в иммутабельный Dictionary.
func OpenPyMorphy(dir string) (*Dictionary, error) {
	d, err := pymorphy2.ImportFromDir(dir)
	if err != nil {
		return nil, err
	}
	return &Dictionary{d: d}, nil
}

// CompileFromXML компилирует словарь OpenCorpora из dict.xml.
// progress — необязательный callback для вывода прогресса.
func CompileFromXML(r interface{ Read([]byte) (int, error) }, progress opencorpora.Progress) (*Dictionary, error) {
	d, err := opencorpora.CompileFromXML(r, progress)
	if err != nil {
		return nil, err
	}
	return &Dictionary{d: d}, nil
}

// CompileFromXMLFile открывает path (dict.xml) и вызывает CompileFromXML.
func CompileFromXMLFile(path string, progress opencorpora.Progress) (*Dictionary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("morphology: open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return CompileFromXML(f, progress)
}

// Language возвращает код языка словаря.
func (x *Dictionary) Language() string { return x.d.Language }

// Open загружает словарь из файла GMOR (единый дисковый формат, SaveTo).
// Горячие секции (words.dawg) отображаются через mmap без копирования;
// результат необходимо закрывать методом Close.
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

// Close освобождает ресурсы словаря, открытого через Open (mmap-регион).
// Для словарей импортёров — no-op. После Close словарь использовать нельзя.
func (x *Dictionary) Close() error {
	if x == nil || x.mm == nil {
		return nil
	}
	err := x.mm.Close()
	x.mm = nil
	return err
}

// parseContainer собирает внутренний словарь из секций GMOR-файла.
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

	return d, nil
}
