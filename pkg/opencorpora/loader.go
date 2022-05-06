package opencorpora

import (
	"compress/bzip2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"git.media-tel.ru/railgo/logging"
	"github.com/amarin/binutils"
	"github.com/amarin/libxml"

	"github.com/amarin/gomorphy/internal/index"
	"github.com/amarin/gomorphy/pkg/common"
)

// Loader provides OpenCorpora dictionary parsing utilities.
type Loader struct {
	logging.Logger
	dataPath   string
	stringData string
}

func NewLoader(dataPath string) *Loader {
	if dataPath == "" {
		dataPath = common.DomainDataPath(DomainName)
	}

	return &Loader{
		Logger:   logging.NewNamedLogger("loader").WithLevel(logging.LevelDebug),
		dataPath: dataPath,
	}
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

// IsDownloadExists returns true if downloaded file exists at expected path.
func (loader Loader) IsDownloadExists() bool {
	loader.Info("check if downloaded file exists")
	expectedFile := loader.downloadedFilePath()
	loader.Debugf("check file %v", expectedFile)
	fileStat, err := os.Stat(expectedFile)
	switch {
	case err != nil && errors.Is(err, os.ErrNotExist):
		loader.Debugf("file not exists at %v", expectedFile)

		return false // no file
	case err != nil:
		loader.Debugf("file access: %v: %v", expectedFile, err)

		return false // no file
	default:
		modTimeFormat := "2006-01-02T15:04:05Z07:00"
		loader.Debugf(
			"exists: %s: modified %s: size %d",
			expectedFile, fileStat.ModTime().Format(modTimeFormat), fileStat.Size())

		return true
	}
}

// IsUnpackedExists returns true if downloaded and unpacked file exists at expected path.
func (loader Loader) IsUnpackedExists() bool {
	loader.Info("check if unpacked file exists")
	expectedFile := loader.unpackedFilePath()
	loader.Debugf("check file %v", expectedFile)
	fileStat, err := os.Stat(expectedFile)
	switch {
	case err != nil && errors.Is(err, os.ErrNotExist):
		loader.Debugf("file not exists at %v", expectedFile)

		return false // no file
	case err != nil:
		loader.Debugf("file access: %v: %v", expectedFile, err)

		return false // no file
	default:
		modTimeFormat := "2006-01-02T15:04:05Z07:00"
		loader.Debugf(
			"exists: %s: modified %s: size %d",
			expectedFile, fileStat.ModTime().Format(modTimeFormat), fileStat.Size())

		return true
	}
}

func (loader Loader) IsUpdateRequired() (bool, error) {
	loader.Info("check if update required")
	expectedFile := loader.downloadedFilePath()
	loader.Debugf("check file %v", expectedFile)
	fileStat, err := os.Stat(expectedFile)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		loader.Debugf("file not exists, update required: %v", expectedFile)

		return true, nil // no file, update required
	}

	loader.Debugf("check remote %v", RemoteURL)
	response, err := http.Head(RemoteURL)
	if err != nil {
		loader.Warnf("remote %v: error: %v", RemoteURL, err)
		return false, err
	}

	if response.StatusCode != 200 {
		loader.Warnf("remote %v: status: %v", RemoteURL, response.StatusCode)
		return false, fmt.Errorf("unexpected response code %v", response.StatusCode)
	}

	lastModifiedString, ok := response.Header[common.HTTPHeaderLastModified]
	if ok && len(lastModifiedString) > 0 {
		loader.Debugf("remote %v: %v", common.HTTPHeaderLastModified, lastModifiedString)
		lastModified, err := time.Parse(time.RFC1123, lastModifiedString[0])
		if err != nil {
			return true, nil // cant compare lastModified, do update
		}
		if lastModified.After(fileStat.ModTime()) {
			return true, nil // site version is older then local
		}
	} else {
		loader.Debugf("remote %v: missed, assume no update required", common.HTTPHeaderLastModified)
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

func (loader *Loader) LoadIndex() (mainIndex *index.Index, err error) {
	var reader *binutils.BinaryReader

	fromFile := loader.compiledFilePath()

	loader.Debugf("opening %v", fromFile)
	if reader, err = binutils.OpenFile(fromFile); err != nil {
		return nil, fmt.Errorf("%w: open index: %v", Error, err)
	}

	defer func() {
		loader.Debugf("loading finished %v", fromFile)
		if closeErr := reader.Close(); err != nil {
			loader.Warnf("close index: %v", closeErr)
		}

		if err != nil {
			loader.Error(err.Error())
		} else {
			loader.Infof("compiled index loaded from %v", fromFile)
		}
	}()

	loader.Debug("create index instance")
	mainIndex = index.New()

	loader.Debug("load index data")
	if _, err = mainIndex.BinaryReadFrom(reader); err != nil {
		return nil, fmt.Errorf("%w: read index: %v", Error, err)
	}

	return mainIndex, nil
}

func (loader *Loader) SaveIndex(mainIndex *index.Index, toFile string) (err error) {
	var writer *binutils.BinaryWriter

	if writer, err = binutils.CreateFile(toFile); err != nil {
		return fmt.Errorf("%w: create index: %v", Error, err)
	}

	defer func() {
		loader.Debugf("finishing %v", toFile)
		if closeErr := writer.Close(); err != nil {
			loader.Warnf("close index: %v", closeErr)
		}

		if err != nil {
			loader.Error(err.Error())

			if removeErr := os.Remove(toFile); removeErr != nil {
				loader.Warnf("remove incomplete index: %v", removeErr)
			}
		} else {
			loader.Infof("compiled index saved at %v", toFile)
		}
	}()

	if err = mainIndex.BinaryWriteTo(writer); err != nil {
		return fmt.Errorf("%w: save index: %v", Error, err)
	}

	return nil
}

func (loader *Loader) ParseUpdate(fromFile string, toFile string) (err error) {
	loader.Info("start parse")
	mainIndex := index.New()
	parser := newParser(mainIndex)
	if err = libxml.ParseXMLFile(fromFile, parser); err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	return loader.SaveIndex(mainIndex, toFile)
}

func (loader Loader) Update(forceRecompile bool) (err error) {
	var updated, updateRequired, downloadedExists, unpackedExists bool

	loader.Info("check OpenCorpora updates")

	updateRequired, err = loader.IsUpdateRequired()
	downloadedExists = loader.IsDownloadExists()
	unpackedExists = loader.IsUnpackedExists()

	switch {
	case err != nil && forceRecompile && unpackedExists:
		loader.Errorf("check update: %v, force recompile using previous unpacked file", err)
		goto compile
	case err != nil && forceRecompile && downloadedExists:
		loader.Errorf("check update: %v, force recompile using previous downloaded file", err)
		goto unpack
	case err != nil:
		loader.Errorf("check update: %v", err)
		return err
	case !updateRequired:
		loader.Info("lemmata up to date")

		if forceRecompile {
			loader.Info("do recompile as force update required")
			goto compile
		}

		return nil

	default:
		loader.Info("update required, downloading")
	}

	updated, err = loader.DownloadUpdate()

	switch {
	case err != nil:
		loader.Errorf("download: %v", err)
		return err
	case !updated:
		loader.Warn("files not updated, no errors")
		return nil
	default:
		loader.Info("downloaded, unpacking")
	}
unpack:
	err = loader.UnpackUpdate()

	switch {
	case err != nil:
		loader.Errorf("unpack: %v", err)
		return err
	default:
		loader.Info("lemmata updated, compile")
	}

compile:
	if err = loader.ParseUpdate(loader.unpackedFilePath(), loader.compiledFilePath()); err != nil {
		loader.Errorf("compile: %v", err)
		return err
	}

	return nil
}
