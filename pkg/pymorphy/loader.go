// Package pymorphy downloads and unpacks pre-built pymorphy2 dictionaries
// (pymorphy2-dicts-ru wheel from PyPI). Compilation/loading of the unpacked
// data is handled externally via pkg/morphology (OpenPyMorphy).
package pymorphy

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/pkg/common"
)

// Loader provides pymorphy2 dictionary download and unpacking utilities.
type Loader struct {
	logging.Logger
	dataPath string
}

// NewLoader creates a new pymorphy loader instance.
// Takes path to data storage. If empty path provided, uses .data/pymorphy by default.
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

	loader.Debugf("check remote %v", PyPIJSONURL)
	remoteVersion, _, err := fetchLatestWheel(PyPIJSONURL)
	if err != nil {
		loader.Warnf("remote %v: error: %v", PyPIJSONURL, err)
		return false, err
	}

	return remoteVersion != localVersion, nil
}

// DownloadUpdate downloads the latest wheel from PyPI if an update is needed.
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

	version, wheelURL, err := fetchLatestWheel(PyPIJSONURL)
	if err != nil {
		return false, err
	}

	if err := downloadFile(wheelURL, loader.archiveFilePath()); err != nil {
		return false, err
	}

	if err := os.WriteFile(loader.versionFilePath(), []byte(version), 0o644); err != nil {
		return false, fmt.Errorf("%w: write version file: %w", ErrPymorphy, err)
	}

	return true, nil
}

// UnpackUpdate extracts the wheel's WheelDataSubtree into UnpackedDirPath.
func (loader *Loader) UnpackUpdate() error {
	if err := common.MakeDomainDataPath(DomainName); err != nil {
		return err
	}

	r, err := zip.OpenReader(loader.archiveFilePath())
	if err != nil {
		return fmt.Errorf("%w: open archive: %w", ErrPymorphy, err)
	}
	defer func() { _ = r.Close() }()

	targetDir := loader.UnpackedDirPath()
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return err
	}

	extracted := 0
	for _, f := range r.File {
		rel, ok := strings.CutPrefix(f.Name, WheelDataSubtree)
		if !ok || rel == "" || strings.HasSuffix(f.Name, "/") {
			continue
		}

		destPath := filepath.Join(targetDir, filepath.FromSlash(rel))
		if !strings.HasPrefix(destPath, filepath.Clean(targetDir)+string(os.PathSeparator)) {
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

	return nil
}

// Sync downloads and unpacks the dictionary archive if needed, without
// compiling. After Sync completes, the caller can use loader.UnpackedDirPath()
// to load the dictionary via pkg/morphology.OpenPyMorphy (raw UTF-8) or
// OpenPyMorphyDense (the dense 1-byte alphabet gomorphy's own CLI
// defaults to, see cmd/gomorphy's build command).
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

func downloadFile(url, destPath string) error {
	resp, err := http.Get(url) //nolint:gosec,noctx
	if err != nil {
		return fmt.Errorf("%w: download %s: %w", ErrPymorphy, url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %s: unexpected status %d", ErrPymorphy, url, resp.StatusCode)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("%w: write %s: %w", ErrPymorphy, destPath, err)
	}

	return nil
}
