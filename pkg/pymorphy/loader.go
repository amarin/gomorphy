// Package pymorphy downloads and unpacks pre-built pymorphy2 dictionaries
// (pymorphy2-dicts-ru wheel from PyPI). Compilation/loading of the unpacked
// data is handled externally via pkg/morphology (OpenPyMorphy).
package pymorphy

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/pkg/common"
)

// Loader provides pymorphy2 dictionary download and unpacking utilities.
// A Loader is not safe for concurrent use.
type Loader struct {
	logging.Logger
	dataPath string
	pypiURL  string // PyPIJSONURL; replaced in tests
}

// NewLoader creates a new pymorphy loader instance.
// Takes path to data storage. If empty path provided, uses .data/pymorphy by default.
// Logging needs no setup: see common.NewLoaderLogger.
func NewLoader(dataPath string) *Loader {
	if dataPath == "" {
		dataPath = common.DomainDataPath(DomainName)
	}

	return &Loader{
		Logger:   common.NewLoaderLogger("loader"),
		dataPath: dataPath,
		pypiURL:  PyPIJSONURL,
	}
}

// DataPath returns the data directory path.
func (loader *Loader) DataPath() string {
	return path.Join(loader.dataPath)
}

// UnpackedDirPath returns the path dictionary sources are extracted into.
func (loader *Loader) UnpackedDirPath() string {
	return loader.filePath(LocalUnpackedDirName)
}

func (loader *Loader) filePath(name string) string {
	return path.Join(loader.dataPath, name)
}

func (loader *Loader) archiveFilePath() string {
	return loader.filePath(LocalArchiveFilename)
}

func (loader *Loader) versionFilePath() string {
	return loader.filePath(LocalVersionFilename)
}

// IsUnpackedExists returns true if the unpacked dictionary sources exist.
func (loader *Loader) IsUnpackedExists() bool {
	loader.Info("check if unpacked dir exists")
	marker := filepath.Join(loader.UnpackedDirPath(), unpackedMarkerFilename)
	loader.Debugf("check file %v", marker)
	_, err := os.Stat(marker)
	return err == nil
}

// IsDownloadExists returns true if the downloaded wheel archive exists.
func (loader *Loader) IsDownloadExists() bool {
	loader.Info("check if downloaded archive exists")
	expectedFile := loader.archiveFilePath()
	loader.Debugf("check file %v", expectedFile)
	_, err := os.Stat(expectedFile)
	return err == nil
}

// LocalVersion returns the PyPI version string of the currently downloaded
// archive, and whether one is recorded at all.
func (loader *Loader) LocalVersion() (string, bool) {
	data, err := os.ReadFile(loader.versionFilePath())
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(data)), true
}

// IsUpdateRequired checks PyPI for a release newer than the locally recorded
// version. A missing archive or version record always counts as an update.
func (loader *Loader) IsUpdateRequired() (bool, error) {
	loader.Info("check if update required")
	if !loader.IsDownloadExists() {
		return true, nil
	}
	localVersion, ok := loader.LocalVersion()
	if !ok {
		return true, nil
	}

	loader.Debugf("check remote %v", loader.pypiURL)
	remoteVersion, _, err := fetchLatestWheel(loader.pypiURL)
	if err != nil {
		loader.Warnf("remote %v: error: %v", loader.pypiURL, err)
		return false, err
	}

	return remoteVersion != localVersion, nil
}

// DownloadUpdate downloads the latest wheel from PyPI if IsUpdateRequired
// says so, and reports whether it did. The archive is replaced atomically
// and the recorded version only after it: a failed download (network
// error, non-200 status, truncated body) keeps the previous release.
// dataPath is created if missing.
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
	version, wheelURL, err := fetchLatestWheel(loader.pypiURL)
	if err != nil {
		return err
	}
	if err := common.DownloadFile(wheelURL, loader.archiveFilePath()); err != nil {
		return fmt.Errorf("%w: %w", ErrPymorphy, err)
	}
	err = common.WriteFileAtomic(loader.versionFilePath(), func(w io.Writer) error {
		_, err := io.WriteString(w, version)
		return err
	})
	if err != nil {
		return fmt.Errorf("%w: write version file: %w", ErrPymorphy, err)
	}
	return nil
}

// UnpackUpdate extracts the wheel's WheelDataSubtree into UnpackedDirPath.
// It extracts into a temporary directory first and replaces the previous
// unpacked copy only when that succeeded, so files of an older release do
// not linger and a failed unpack keeps the old copy.
func (loader *Loader) UnpackUpdate() error {
	r, err := zip.OpenReader(loader.archiveFilePath())
	if err != nil {
		return fmt.Errorf("%w: open archive: %w", ErrPymorphy, err)
	}
	defer func() { _ = r.Close() }()

	targetDir := loader.UnpackedDirPath()
	tmpDir, err := os.MkdirTemp(loader.dataPath, "."+LocalUnpackedDirName+".tmp-*")
	if err != nil {
		return fmt.Errorf("%w: %w", ErrPymorphy, err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }() // a no-op after the rename

	extracted := 0
	for _, f := range r.File {
		rel, ok := strings.CutPrefix(f.Name, WheelDataSubtree)
		if !ok || rel == "" || strings.HasSuffix(f.Name, "/") {
			continue
		}

		destPath := filepath.Join(tmpDir, filepath.FromSlash(rel))
		if !strings.HasPrefix(destPath, filepath.Clean(tmpDir)+string(os.PathSeparator)) {
			return fmt.Errorf("%w: unsafe archive path %q", ErrPymorphy, f.Name)
		}

		if err := extractZipFile(f, destPath); err != nil {
			return err
		}
		extracted++
	}

	if extracted == 0 {
		return fmt.Errorf("%w: no files found under %q in archive", ErrPymorphy, WheelDataSubtree)
	}

	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("%w: remove old unpacked copy: %w", ErrPymorphy, err)
	}
	if err := os.Rename(tmpDir, targetDir); err != nil {
		return fmt.Errorf("%w: %w", ErrPymorphy, err)
	}
	return nil
}

// Sync brings the unpacked dictionary up to date without compiling it.
// Unless skipDownload is set, it asks PyPI whether a newer release exists
// and downloads it; if PyPI cannot be reached but an archive is already on
// disk, Sync goes on with that one. It then unpacks the archive when it was
// just downloaded or nothing is unpacked yet. With skipDownload set, Sync
// makes no network requests and only unpacks an archive already on disk.
// After Sync, UnpackedDirPath is ready for pkg/morphology.OpenPyMorphy
// (raw UTF-8) or OpenPyMorphyDense (the dense 1-byte alphabet the gomorphy
// CLI uses).
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
		return fmt.Errorf("%w: no unpacked dictionary at %v", ErrPymorphy, loader.UnpackedDirPath())
	}

	return nil
}

func extractZipFile(f *zip.File, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
		return err
	}

	src, err := f.Open()
	if err != nil {
		return fmt.Errorf("%w: open %q in archive: %w", ErrPymorphy, f.Name, err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil { //nolint:gosec // size bounded by the source wheel, not attacker-controlled input
		return fmt.Errorf("%w: extract %q: %w", ErrPymorphy, f.Name, err)
	}

	return nil
}
