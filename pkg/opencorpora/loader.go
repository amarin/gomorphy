// Package opencorpora downloads and unpacks the OpenCorpora dictionary
// source archive (dict.opcorpora.xml.bz2 from opencorpora.org).
// Compilation/loading of the unpacked dict.xml is handled externally via
// pkg/morphology (CompileFromXML/CompileFromXMLFile).
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
	dataPath  string
	remoteURL string // RemoteURL; replaced in tests
}

// NewLoader creates a new opencorpora loader instance.
// Takes path to data storage. If empty path provided, uses .data/opencorpora by default.
// Logging needs no setup: see common.NewLoaderLogger.
func NewLoader(dataPath string) *Loader {
	if dataPath == "" {
		dataPath = common.DomainDataPath(DomainName)
	}

	return &Loader{
		Logger:    common.NewLoaderLogger("loader"),
		dataPath:  dataPath,
		remoteURL: RemoteURL,
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
	expectedFile := loader.UnpackedFilePath()
	loader.Debugf("check file %v", expectedFile)
	_, err := os.Stat(expectedFile)
	return err == nil
}

// IsDownloadExists returns true if the downloaded archive exists.
func (loader *Loader) IsDownloadExists() bool {
	loader.Info("check if downloaded file exists")
	expectedFile := loader.downloadedFilePath()
	loader.Debugf("check file %v", expectedFile)
	fileStat, err := os.Stat(expectedFile)
	if err != nil {
		return false
	}
	loader.Debugf("exists: %s: modified %s: size %d",
		expectedFile, fileStat.ModTime().Format("2006-01-02T15:04:05Z07:00"), fileStat.Size())
	return true
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
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, err
	}

	loader.Debugf("check remote %v", loader.remoteURL)
	response, err := http.Head(loader.remoteURL)
	if err != nil {
		loader.Warnf("remote %v: error: %v", loader.remoteURL, err)
		return false, err
	}
	defer func() { _ = response.Body.Close() }()

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

// DownloadUpdate downloads the dictionary archive if IsUpdateRequired says
// so, and reports whether it did. The archive is replaced atomically: a
// failed download (network error, non-200 status, truncated body) keeps the
// previous archive. dataPath is created if missing.
func (loader *Loader) DownloadUpdate() (updated bool, err error) {
	updateRequired, err := loader.IsUpdateRequired()
	if err != nil {
		return false, err
	}
	if !updateRequired {
		return false, nil
	}
	if err := loader.download(); err != nil {
		return false, err
	}
	return true, nil
}

func (loader *Loader) download() error {
	if err := common.DownloadFile(loader.remoteURL, loader.downloadedFilePath()); err != nil {
		return fmt.Errorf("%w: %w", ErrOpenCorpora, err)
	}
	return nil
}

// UnpackUpdate extracts dict.xml from the downloaded bzip2 archive,
// replacing an existing dict.xml atomically (a failed unpack keeps it).
func (loader *Loader) UnpackUpdate() error {
	source, err := os.Open(loader.downloadedFilePath())
	if err != nil {
		return fmt.Errorf("%w: open archive: %w", ErrOpenCorpora, err)
	}
	defer func() { _ = source.Close() }()

	return common.WriteFileAtomic(loader.UnpackedFilePath(), func(w io.Writer) error {
		if _, err := io.Copy(w, bzip2.NewReader(source)); err != nil {
			return fmt.Errorf("%w: unpack: %w", ErrOpenCorpora, err)
		}
		return nil
	})
}

// Sync brings the unpacked dict.xml up to date without compiling it.
// Unless skipDownload is set, it asks the remote whether a newer archive
// exists and downloads it; if the remote cannot be reached but an archive
// is already on disk, Sync goes on with that one. It then unpacks the
// archive when it was just downloaded or dict.xml is missing. With
// skipDownload set, Sync makes no network requests and only unpacks an
// archive already on disk. After Sync, UnpackedFilePath is ready for
// pkg/morphology.CompileFromXMLFile.
func (loader *Loader) Sync(skipDownload bool) error {
	updated := false
	if !skipDownload {
		required, err := loader.IsUpdateRequired()
		if err != nil {
			if !loader.IsDownloadExists() {
				return err
			}
			loader.Warnf("check updates: %v; using the local archive", err)
		}
		if required {
			loader.Info("update required, downloading")
			if err := loader.download(); err != nil {
				loader.Errorf("download: %v", err)
				return err
			}
			updated = true
			loader.Info("downloaded")
		}
	}

	if loader.IsDownloadExists() && (updated || !loader.IsUnpackedExists()) {
		loader.Info("unpacking")
		if err := loader.UnpackUpdate(); err != nil {
			loader.Errorf("unpack: %v", err)
			return err
		}
		loader.Info("unpacked")
	}

	if !loader.IsUnpackedExists() {
		return fmt.Errorf("%w: no unpacked dictionary at %v", ErrOpenCorpora, loader.UnpackedFilePath())
	}

	return nil
}
