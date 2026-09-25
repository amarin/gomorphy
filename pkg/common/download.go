package common

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes path through write: the data goes to a temporary
// file in path's directory (which must exist), and that file replaces path
// only after write returned nil and the file was closed. On any error path
// keeps its previous content, or stays absent, and the temporary file is
// removed. The result has mode 0644.
func WriteFileAtomic(path string, write func(w io.Writer) error) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()

	if err = write(tmp); err != nil {
		return err
	}
	if err = tmp.Chmod(0o644); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// DownloadFile fetches url with GET and stores the body at path, creating
// path's directory if needed. The write is atomic (see WriteFileAtomic): a
// non-200 status, a network error or a truncated body leave path as it
// was.
func DownloadFile(url, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}

	resp, err := http.Get(url) //nolint:gosec,noctx // URL is the loader's configured source
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected status %d", url, resp.StatusCode)
	}

	return WriteFileAtomic(path, func(w io.Writer) error {
		if _, err := io.Copy(w, resp.Body); err != nil {
			return fmt.Errorf("download %s: %w", url, err)
		}
		return nil
	})
}
