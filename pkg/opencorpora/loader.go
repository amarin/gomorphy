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
// Compilation is handled externally via pkg/morphology.
type Loader struct {
	logging.Logger
	dataPath string
}

// NewLoader creates a new opencorpora loader instance.
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

// DataPath returns the data directory path.
func (loader *Loader) DataPath() string {
	return path.Join(loader.dataPath)
}

// UnpackedFilePath returns path to the unpacked dict.xml.
func (loader *Loader) UnpackedFilePath() string {
	return loader.filePath(LocalUnpackedFilename)
}

func (loader *Loader) filePath(fileName string) string {
	return path.Join(loader.dataPath, fileName)
}

// IsUnpackedExists returns true if the unpacked dict.xml exists.
func (loader *Loader) IsUnpackedExists() bool {
	loader.Info("check if unpacked file exists")
	expectedFile := loader.unpackedFilePath()
	loader.Debugf("check file %v", expectedFile)
	_, err := os.Stat(expectedFile)
	return err == nil
}

// UnpackedFilePath returns the path to the unpacked file (internal alias).
func (loader *Loader) unpackedFilePath() string {
	return loader.filePath(LocalUnpackedFilename)
}

// IsDownloadExists returns true if the downloaded archive exists.
func (loader *Loader) IsDownloadExists() bool {
	loader.Info("check if downloaded file exists")
	expectedFile := loader.downloadedFilePath()
	loader.Debugf("check file %v", expectedFile)
	fileStat, err := os.Stat(expectedFile)
	switch {
	case err != nil && errors.Is(err, os.ErrNotExist):
		return false
	case err != nil:
		return false
	default:
		loader.Debugf("exists: %s: modified %s: size %d",
			expectedFile, fileStat.ModTime().Format("2006-01-02T15:04:05Z07:00"), fileStat.Size())
		return true
	}
}

func (loader *Loader) downloadedFilePath() string {
	return loader.filePath(LocalSourceFilename)
}

// IsUpdateRequired checks the remote for a newer version.
func (loader *Loader) IsUpdateRequired() (bool, error) {
	loader.Info("check if update required")
	expectedFile := loader.downloadedFilePath()
	loader.Debugf("check file %v", expectedFile)
	fileStat, err := os.Stat(expectedFile)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return true, nil
	}

	loader.Debugf("check remote %v", RemoteURL)
	response, err := http.Head(RemoteURL)
	if err != nil {
		loader.Warnf("remote %v: error: %v", RemoteURL, err)
		return false, err
	}

	if response.StatusCode != 200 {
		return false, fmt.Errorf("unexpected response code %v", response.StatusCode)
	}

	lastModifiedString, ok := response.Header[common.HTTPHeaderLastModified]
	if ok && len(lastModifiedString) > 0 {
		lastModified, err := time.Parse(time.RFC1123, lastModifiedString[0])
		if err != nil {
			return true, nil
		}
		if lastModified.After(fileStat.ModTime()) {
			return true, nil
		}
	}

	return false, nil
}

// DownloadUpdate downloads the dictionary archive if update is needed.
func (loader *Loader) DownloadUpdate() (updated bool, err error) {
	updateRequired, err := loader.IsUpdateRequired()
	if err != nil {
		return false, err
	}
	if !updateRequired {
		return false, nil
	}

	if err := common.MakeDomainDataPath(DomainName); err != nil {
		return false, err
	}

	response, err := http.Get(RemoteURL) // nolint:gosec,noctx
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	file, err := os.Create(loader.downloadedFilePath())
	if err != nil {
		return false, err
	}
	defer file.Close()

	if _, err = io.Copy(file, response.Body); err != nil {
		return false, err
	}

	return true, nil
}

// UnpackUpdate extracts dict.xml from the bzip2 archive.
func (loader *Loader) UnpackUpdate() error {
	if err := common.MakeDomainDataPath(DomainName); err != nil {
		return err
	}

	source, err := os.Open(loader.downloadedFilePath())
	if err != nil {
		return err
	}
	defer source.Close()

	bzipSource := bzip2.NewReader(source)

	target, err := os.Create(loader.unpackedFilePath())
	if err != nil {
		return err
	}
	defer target.Close()

	if _, err = io.Copy(target, bzipSource); err != nil {
		return err
	}

	return nil
}

// Sync downloads and unpacks the dictionary archive if needed, without compiling.
// After Sync completes, the caller can use loader.UnpackedFilePath() to get the
// path to dict.xml and compile it via pkg/morphology.
func (loader *Loader) Sync(skipDownload bool) error {
	downloadedExists := loader.IsDownloadExists()
	updateRequired, err := loader.IsUpdateRequired()
	if err != nil {
		loader.Warnf("check updates: %v", err)
	}

	downloadRequired := !skipDownload && (updateRequired || !downloadedExists)

	if downloadRequired {
		loader.Info("update required, downloading")
		updated, err := loader.DownloadUpdate()
		if err != nil {
			loader.Errorf("download: %v", err)
			return err
		}
		if updated {
			loader.Info("downloaded")
		} else {
			loader.Warn("files not updated, no errors")
		}
	} else {
		loader.Info("skip download")
	}

	if !loader.IsUnpackedExists() && loader.IsDownloadExists() {
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
