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

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/pkg/common"
)

// Loader provides OpenCorpora dictionary download and unpacking utilities.
// Compilation into runtime format is performed by pkg/dictionary, see docs/todo.md stage 7.
type Loader struct {
	logging.Logger
	dataPath string
}

// NewLoader creates new opencorpora loader instance.
// Takes path to data storage. If empty path provided, uses .data/opencorpora by default.
func NewLoader(dataPath string) *Loader {
	if dataPath == "" {
		dataPath = common.DomainDataPath(DomainName)
	}

	return &Loader{
		Logger:   logging.NewNamedLogger("loader").WithLevel(logging.LevelDebug),
		dataPath: dataPath,
	}
}

func (loader *Loader) DataPath() string {
	return path.Join(loader.dataPath)
}

func (loader *Loader) SetDataPath(dataPath string) {
	if dataPath == "" {
		dataPath = common.DomainDataPath(DomainName)
	}

	loader.dataPath = dataPath
}

func (loader *Loader) filePath(fileName string) string {
	return path.Join(loader.DataPath(), fileName)
}

// downloadedFilePath returns path to downloaded archive file.
func (loader *Loader) downloadedFilePath() string {
	return loader.filePath(LocalSourceFilename)
}

// unpackedFilePath returns path to unpacked lemmata file.
func (loader *Loader) unpackedFilePath() string {
	return loader.filePath(LocalUnpackedFilename)
}

// IsDownloadExists returns true if downloaded file exists at expected path.
func (loader *Loader) IsDownloadExists() bool {
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
func (loader *Loader) IsUnpackedExists() bool {
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

func (loader *Loader) IsUpdateRequired() (bool, error) {
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

func (loader *Loader) DownloadUpdate() (updated bool, err error) {
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
	response, err := http.Get(RemoteURL) // nolint:gosec,noctx
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

	if _, err = io.Copy(file, response.Body); err != nil { // nolint:gosec
		return false, err
	}

	return true, nil
}

func (loader *Loader) UnpackUpdate() (err error) {
	var (
		source     io.ReadCloser
		bzipSource io.Reader
		target     io.WriteCloser
	)

	if err := common.MakeDomainDataPath(DomainName); err != nil {
		return err
	}

	if source, err = os.Open(loader.downloadedFilePath()); err != nil { // nolint:gosec
		return err
	}

	defer func() { _ = source.Close() }()

	bzipSource = bzip2.NewReader(source)

	if target, err = os.Create(loader.unpackedFilePath()); err != nil { // nolint:gosec
		return err
	}

	defer func() { _ = target.Close() }()

	if _, err = io.Copy(target, bzipSource); err != nil { // nolint:gosec
		return err
	}

	return nil
}

// Update downloads and unpacks dictionary when required.
// skipDownload set uses local archive only.
// Compilation of unpacked XML into runtime dictionary file is out of scope here,
// see docs/todo.md stage 7.
func (loader *Loader) Update(skipDownload bool) error {
	downloadedExists := loader.IsDownloadExists()
	updateRequired, err := loader.IsUpdateRequired()
	if err != nil {
		loader.Warnf("check updates: %v", err)
	}

	downloadRequired := !skipDownload && (updateRequired || !downloadedExists)
	unpackRequired := false

	if downloadRequired {
		loader.Info("update required, downloading")

		updated, err := loader.DownloadUpdate()
		switch {
		case err != nil:
			loader.Errorf("download: %v", err)
			return err
		case !updated:
			loader.Warn("files not updated, no errors")
		default:
			loader.Info("downloaded")
		}
	} else {
		loader.Info("skip download")
	}

	if !loader.IsUnpackedExists() && loader.IsDownloadExists() {
		unpackRequired = true
	}

	if unpackRequired {
		loader.Info("unpacking")

		if err := loader.UnpackUpdate(); err != nil {
			loader.Errorf("unpack: %v", err)
			return err
		}
		loader.Info("unpacked")
	} else {
		loader.Info("skip unpack")
	}

	if !loader.IsUnpackedExists() {
		return fmt.Errorf("%w: no unpacked dictionary at %v", Error, loader.unpackedFilePath())
	}

	return nil
}
