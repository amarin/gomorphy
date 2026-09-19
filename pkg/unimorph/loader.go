// Package unimorph downloads a UniMorph language's raw TSV dictionary
// file (github.com/unimorph/<iso>). Compilation/loading of the
// downloaded TSV is handled externally via pkg/morphology
// (CompileFromUniMorph/CompileFromUniMorphFile).
package unimorph

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/pkg/common"
)

// Loader downloads one UniMorph language's raw TSV dictionary. Unlike
// pkg/pymorphy/pkg/opencorpora, there is no archive to unpack — the
// downloaded file is already the usable TSV — so UnpackUpdate/
// IsUnpackedExists are thin pass-throughs over the download itself,
// kept only so the CLI's download/unpack/build/update commands can
// treat every source type uniformly.
type Loader struct {
	logging.Logger
	dataPath string
	language string
	iso      string
}

// NewLoader creates a UniMorph loader for language (gomorphy's own
// code, e.g. "ru" — only "ru" is supported today, see isoCodes).
// dataPath, if empty, defaults to .data/unimorph/<language>.
func NewLoader(language, dataPath string) (*Loader, error) {
	iso, ok := isoCodes[language]
	if !ok {
		return nil, fmt.Errorf("%w: unsupported language %q", ErrUnimorph, language)
	}
	if dataPath == "" {
		dataPath = path.Join(common.DomainDataPath(DomainName), language)
	}
	return &Loader{
		Logger:   logging.NewNamedLogger("loader").WithLevel(logging.LevelDebug),
		dataPath: dataPath,
		language: language,
		iso:      iso,
	}, nil
}

// DataPath returns the data directory path.
func (loader *Loader) DataPath() string {
	return path.Join(loader.dataPath)
}

// UnpackedFilePath returns the path the downloaded TSV lives at — the
// same file DownloadUpdate writes, since there is nothing to unpack.
func (loader *Loader) UnpackedFilePath() string {
	return loader.filePath(LocalUnpackedFilename)
}

func (loader *Loader) filePath(name string) string {
	return path.Join(loader.dataPath, name)
}

// IsDownloadExists reports whether the TSV has been downloaded.
func (loader *Loader) IsDownloadExists() bool {
	loader.Info("check if downloaded file exists")
	_, err := os.Stat(loader.UnpackedFilePath())
	return err == nil
}

// IsUnpackedExists is IsDownloadExists under another name — see the
// Loader doc comment on why download and unpack are the same step here.
func (loader *Loader) IsUnpackedExists() bool {
	return loader.IsDownloadExists()
}

// IsUpdateRequired checks the remote for a newer version, the same way
// pkg/opencorpora does (a HEAD request + Last-Modified comparison
// against the local file's mtime) — GitHub's raw-file CDN serves this
// header. A missing local file always counts as an update.
func (loader *Loader) IsUpdateRequired() (bool, error) {
	loader.Info("check if update required")
	fileStat, err := os.Stat(loader.UnpackedFilePath())
	if err != nil {
		return true, nil
	}

	url := remoteURL(loader.iso)
	loader.Debugf("check remote %v", url)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(url) //nolint:gosec,noctx
	if err != nil {
		loader.Warnf("remote %v: error: %v", url, err)
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("%w: %s: unexpected status %d", ErrUnimorph, url, resp.StatusCode)
	}

	lastModified, ok := resp.Header[common.HTTPHeaderLastModified]
	if ok && len(lastModified) > 0 {
		t, err := time.Parse(time.RFC1123, lastModified[0])
		if err == nil && !t.After(fileStat.ModTime()) {
			return false, nil
		}
	}
	return true, nil
}

// DownloadUpdate downloads the language's TSV if an update is needed.
func (loader *Loader) DownloadUpdate() (updated bool, err error) {
	updateRequired, err := loader.IsUpdateRequired()
	if err != nil {
		return false, err
	}
	if !updateRequired {
		return false, nil
	}

	// os.MkdirAll on loader.dataPath directly, not common.MakeDomainDataPath
	// (which only ever creates the default .data/<domain> path, ignoring
	// a caller-supplied custom dataPath — this loader's dataPath already
	// includes the language subdirectory, whether default or custom).
	if err := os.MkdirAll(loader.dataPath, os.ModePerm); err != nil {
		return false, err
	}

	url := remoteURL(loader.iso)
	resp, err := http.Get(url) //nolint:gosec,noctx
	if err != nil {
		return false, fmt.Errorf("%w: download %s: %w", ErrUnimorph, url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("%w: %s: unexpected status %d", ErrUnimorph, url, resp.StatusCode)
	}

	file, err := os.Create(loader.UnpackedFilePath())
	if err != nil {
		return false, err
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return false, fmt.Errorf("%w: write %s: %w", ErrUnimorph, loader.UnpackedFilePath(), err)
	}

	return true, nil
}

// UnpackUpdate is a no-op — see the Loader doc comment. It exists only
// so callers that treat every source type uniformly (the CLI's
// download/unpack/build/update commands) don't need a special case for
// UniMorph.
func (loader *Loader) UnpackUpdate() error {
	if !loader.IsDownloadExists() {
		return fmt.Errorf("%w: no downloaded file at %v", ErrUnimorph, loader.UnpackedFilePath())
	}
	return nil
}

// Sync downloads the TSV if needed, without compiling. After Sync
// completes, the caller can use loader.UnpackedFilePath() to load the
// dictionary via pkg/morphology.CompileFromUniMorphFile.
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

	if !loader.IsDownloadExists() {
		return fmt.Errorf("%w: no downloaded file at %v", ErrUnimorph, loader.UnpackedFilePath())
	}

	return nil
}
