package opencorpora

import (
	"compress/bzip2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/amarin/binutils"
	"github.com/amarin/libxml"
	"github.com/sirupsen/logrus"

	"github.com/amarin/gomorphy/internal/grammemes"
	"github.com/amarin/gomorphy/internal/text"
	"github.com/amarin/gomorphy/pkg/common"
	"github.com/amarin/gomorphy/pkg/words"
)

// OpenCorporaLoader обрабатывает леммы, полученные из словаря.
type Loader struct {
	logger   *logrus.Logger
	dataPath string
}

func NewLoader(logger *logrus.Logger, dataPath string) *Loader {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	if dataPath == "" {
		dataPath = common.DomainDataPath(DomainName)
	}

	return &Loader{logger: logger, dataPath: dataPath}
}

func (loader Loader) DataPath() string {
	return path.Join(loader.dataPath)
}

func (loader *Loader) SetDataPath(dataPath string) {
	if dataPath == "" {
		dataPath = common.DomainDataPath(DomainName)
	}

	loader.dataPath = dataPath
}

func (loader Loader) filePath(fileName string) string {
	return path.Join(loader.DataPath(), fileName)
}

func lemmaToForms(index *grammemes.Index, lemma *Lemma) ([]*words.Word, error) {
	res := make([]*words.Word, len(lemma.F)+1)

	mainForm, err := lemma.L.Word(index)
	if err != nil {
		return nil, WrapOpenCorporaErrorf(err, "cant take main form word")
	}

	res[0] = mainForm

	for idx, form := range lemma.F {
		word, err := form.Word(index)
		if err != nil {
			return nil, WrapOpenCorporaErrorf(err, "cant take form word")
		}

		res[idx+1] = word
	}

	return res, nil
}

// Скомпилировать данные Lemmata из словаря XML в двоичный файл.
func (loader Loader) CompileLemmata(grammemesIndex *grammemes.Index, fromFile string, toFile string) error {
	loader.logger.Info("compile lemmata")
	// make tokens channel
	tokensChanBuffer := 50
	calculateAvgEvery := 5000
	tokensChan := make(chan Lemma, tokensChanBuffer)
	// and dont forget to close it
	defer close(tokensChan)

	loader.logger.Debug("create grammemes index")
	// setup words index
	wordsIndex := words.NewIndex(grammemesIndex)

	loader.logger.Debug("define grabber")
	// define grabber
	idx := 0
	started := time.Now()
	grabber := libxml.NewXmlTargetGrabber("lemma", "", Lemma{}, func(i interface{}) error {
		token, ok := i.(*Lemma)
		if !ok {
			return fmt.Errorf("%w: %T", common.ErrUnexpectedItem, i)
		}
		idx++
		if idx%calculateAvgEvery == 0 {
			avg := float64(idx) / float64(time.Since(started)/time.Second)
			loader.logger.Infof("Process lemma %v avg %.0f tps", idx, avg)
		}
		forms, err := lemmaToForms(grammemesIndex, token)
		if err != nil {
			return fmt.Errorf("lemma to forms: %w", err)
		}
		for _, form := range forms {
			if err := wordsIndex.AddWord(form); err != nil {
				return fmt.Errorf("add word: %w", err)
			}
		}
		return nil
	})

	loader.logger.Debug("grab lemmata data")
	// start stream parse
	err := libxml.ParseXMLFile(fromFile, grabber)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	loader.logger.Info("save lemmata index")
	// save lemmata to file
	if err = binutils.SaveBinary(toFile, wordsIndex); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	loader.logger.Info("compile lemmata")

	return nil
}

// translateToIndexGrammeme translates opencorpora_update grammeme to index grammeme.
func (loader Loader) translateToIndexGrammeme(grammeme Grammeme) grammemes.Grammeme {
	return grammemes.Grammeme{
		ParentAttr:  grammeme.ParentAttr,
		Name:        grammeme.Name,
		Alias:       text.RussianText(grammeme.Alias),
		Description: text.RussianText(grammeme.Description),
	}
}

// CompileGrammemes компилирует индекс граммем из словаря OpenCorpora XML в двоичный файл.
func (loader Loader) CompileGrammemes() error {
	var (
		ocGrammemes             Grammemes
		secondStage, thirdStage []Grammeme
		fromFile, toFile        = loader.unpackedFilePath(), loader.GrammemesIndexFilePath()
	)
	loader.logger.Info("compile grammemes index")

	// define grabber
	grammemesIdx := 0
	grabber := libxml.NewXmlTargetGrabber("grammeme", "", Grammeme{}, func(i interface{}) error {
		grammemesIdx++
		loader.logger.Debug("got grammeme ", strconv.Itoa(grammemesIdx))
		if grammeme, ok := i.(*Grammeme); ok {
			ocGrammemes.Grammeme = append(ocGrammemes.Grammeme, grammeme)
		} else {
			return fmt.Errorf("%w: %T", common.ErrUnexpectedItem, i)
		}
		return nil
	})

	loader.logger.Info("load lemmata data")
	// start stream parser
	if err := libxml.ParseXMLFile(fromFile, grabber); err != nil {
		return err
	}

	loader.logger.Info("append root grammemes")
	// init empty index
	grammemeIndex := grammemes.NewIndex()
	// add root grammemes first and depended of that roots
	for _, grammeme := range ocGrammemes.Grammeme {
		if grammeme.ParentAttr == "" {
			if err := grammemeIndex.Add(loader.translateToIndexGrammeme(*grammeme)); err != nil {
				return err
			}
		} else if err, _ := grammemeIndex.ByName(grammeme.ParentAttr); err != nil {
			secondStage = append(secondStage, *grammeme)
		} else if err := grammemeIndex.Add(loader.translateToIndexGrammeme(*grammeme)); err != nil {
			secondStage = append(secondStage, *grammeme)
		}
	}

	loader.logger.Infof("append %d secondary grammemes", len(secondStage))
	// add second stage grammemes. Some may depend of it, so store third stage list
	for _, grammeme := range secondStage {
		if err := grammemeIndex.Add(loader.translateToIndexGrammeme(grammeme)); err != nil {
			thirdStage = append(thirdStage, grammeme)
		}
	}

	loader.logger.Infof("append %d rest of grammemes", len(thirdStage))
	// add rest of grammemes.
	for _, grammeme := range thirdStage {
		if err := grammemeIndex.Add(loader.translateToIndexGrammeme(grammeme)); err != nil {
			return err
		}
	}

	loader.logger.Info("write grammemes index")
	// write grammemes index to file.
	if err := binutils.SaveBinary(toFile, grammemeIndex); err != nil {
		return err
	} else {
		return nil
	}
}

// LoadGrammemes загружает индекс граммем.
func (loader Loader) LoadGrammemes() (*grammemes.Index, error) {
	loader.logger.Info("loading grammemes index")

	grammemesIndex := grammemes.NewIndex()

	if err := binutils.LoadBinary(loader.GrammemesIndexFilePath(), grammemesIndex); err != nil {
		return nil, err
	} else {
		return grammemesIndex, nil
	}
}

// LoadLemmata загружает компилированный словарь.
func (loader Loader) LoadLemmata(grammemesIndex *grammemes.Index) (*words.Index, error) {
	loader.logger.Info("loading lemmata using grammemes index")

	loader.logger.Debug("setup words index")

	lemmata := words.NewIndex(grammemesIndex)

	loader.logger.Debug("load binary index")

	if err := binutils.LoadBinary(loader.compiledFilePath(), lemmata); err != nil {
		return nil, err
	}

	return lemmata, nil
}

// LoadLemmata загружает компилированный словарь.
func (loader Loader) Lemmata() (lemmata *words.Index, err error) {
	loader.logger.Info("take lemmata")

	var grammemesIndex *grammemes.Index
	grammemesIndex, err = loader.LoadGrammemes()

	if err != nil {
		return nil, err
	}

	lemmata, err = loader.LoadLemmata(grammemesIndex)
	if err := binutils.LoadBinary(loader.compiledFilePath(), lemmata); err != nil {
		return nil, err
	}

	return lemmata, nil
}

// unpackedFilePath returns path to unpacked lemmata file.
func (loader Loader) downloadedFilePath() string {
	return loader.filePath(LocalSourceFilename)
}

// unpackedFilePath returns path to unpacked lemmata file.
func (loader Loader) unpackedFilePath() string {
	return loader.filePath(LocalUnpackedFilename)
}

// compiledFilePath returns path to compiled lemmata file.
func (loader Loader) compiledFilePath() string {
	return loader.filePath(LocalCompiledFilename)
}

// grammemesIndexFilePath returns path to compiled grammemes index file.
func (loader Loader) GrammemesIndexFilePath() string {
	return loader.filePath(GrammemesIndexFilename)
}

func (loader Loader) IsUpdateRequired() (bool, error) {
	fileStat, err := os.Stat(loader.downloadedFilePath())
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return true, nil // no file, update required
	}
	// Get the response bytes from the url
	response, err := http.Head(RemoteURL)
	if err != nil {
		return false, err
	}
	if response.StatusCode != 200 {
		return false, fmt.Errorf("unexpected response code %v", response.StatusCode)
	}
	lastModifiedString, ok := response.Header[common.HTTPHeaderLastModified]
	if ok && len(lastModifiedString) > 0 {
		lastModified, err := time.Parse(time.RFC1123, lastModifiedString[0])
		if err != nil {
			return true, nil // cant compare lastModified, do update
		}
		if lastModified.After(fileStat.ModTime()) {
			return true, nil // site version is older then local
		}
	}

	return false, nil // site version is older then local
}

func (loader Loader) DownloadUpdate() (updated bool, err error) {
	updateRequired, err := loader.IsUpdateRequired()

	switch {
	case err != nil:
		return false, err
	case !updateRequired:
		return false, nil
	}

	if err := common.MakeDomainDataPath(DomainName); err != nil {
		return false, err
	}

	// Get the response bytes from the url
	response, err := http.Get(RemoteURL)
	if err != nil {
		return false, err
	}

	defer func() {
		if response.Body != nil {
			_ = response.Body.Close()
		}
	}()

	// Create a empty file
	file, err := os.Create(loader.downloadedFilePath())
	if err != nil {
		return false, err
	}

	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()

	if _, err = io.Copy(file, response.Body); err != nil {
		return false, err
	}

	return true, nil
}

func (loader Loader) UnpackUpdate() (err error) {
	var (
		source     io.ReadCloser
		bzipSource io.Reader
		target     io.WriteCloser
	)

	if err := common.MakeDomainDataPath(DomainName); err != nil {
		return err
	}

	if source, err = os.Open(loader.downloadedFilePath()); err != nil {
		return err
	}

	defer func() { _ = source.Close() }()

	bzipSource = bzip2.NewReader(source)

	if target, err = os.Create(loader.unpackedFilePath()); err != nil {
		return err
	}

	defer func() { _ = target.Close() }()

	if _, err = io.Copy(target, bzipSource); err != nil { // nolint:gosec
		return err
	}

	return nil
}

func (loader Loader) Update(forceRecompile bool) (err error) {
	var updated, updateRequired bool

	var grammemesIndex *grammemes.Index

	loader.logger.Info("check OpenCorpora updates")

	updateRequired, err = loader.IsUpdateRequired()

	switch {
	case err != nil:
		loader.logger.Errorf("check update: %v", err)
		return err
	case !updateRequired:
		loader.logger.Info("lemmata up to date")

		if forceRecompile {
			goto compile
		}

		return nil

	default:
		loader.logger.Info("update required, downloading")
	}

	updated, err = loader.DownloadUpdate()

	switch {
	case err != nil:
		loader.logger.Errorf("download: %v", err)
		return err
	case !updated:
		loader.logger.Warn("files not updated, no errors")
		return nil
	default:
		loader.logger.Info("downloaded, unpacking")
	}

	err = loader.UnpackUpdate()

	switch {
	case err != nil:
		loader.logger.Errorf("unpack: %v", err)
		return err
	default:
		loader.logger.Info("lemmata updated, compile")
	}

compile:
	if err = loader.CompileGrammemes(); err != nil {
		loader.logger.Errorf("compile: %v", err)
		return err
	}

	if grammemesIndex, err = loader.LoadGrammemes(); err != nil {
		loader.logger.Errorf("load grammemes index: %v", err)
		return err
	}

	if err = loader.CompileLemmata(grammemesIndex, loader.unpackedFilePath(), loader.compiledFilePath()); err != nil {
		loader.logger.Errorf("compile: %v", err)
		return err
	}

	return nil
}
