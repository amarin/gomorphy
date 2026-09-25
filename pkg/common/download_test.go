package common_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/amarin/gomorphy/pkg/common"
)

func TestWriteFileAtomic_KeepsOldFileOnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("boom")
	err := common.WriteFileAtomic(path, func(w io.Writer) error {
		_, _ = w.Write([]byte("partial"))
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	assertFile(t, path, "old")
	assertOnlyFile(t, path)
}

func TestWriteFileAtomic_Replaces(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := common.WriteFileAtomic(path, func(w io.Writer) error {
		_, err := w.Write([]byte("new"))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	assertFile(t, path, "new")
	if st, _ := os.Stat(path); st.Mode().Perm() != 0o644 {
		t.Errorf("mode = %v, want 0644", st.Mode().Perm())
	}
}

func TestDownloadFile_CreatesDirAndWrites(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "a", "b", "f")
	if err := common.DownloadFile(srv.URL, path); err != nil {
		t.Fatal(err)
	}
	assertFile(t, path, "payload")
}

func TestDownloadFile_BadStatus(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "f")
	if err := common.DownloadFile(srv.URL, path); err == nil {
		t.Fatal("expected an error for 404")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file must not exist after a failed download, stat err = %v", err)
	}
}

func TestDownloadFile_TruncatedBodyKeepsOldFile(t *testing.T) {
	srv := httptest.NewServer(TruncatingHandler("partial"))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := common.DownloadFile(srv.URL, path); err == nil {
		t.Fatal("expected an error for a truncated body")
	}
	assertFile(t, path, "old")
	assertOnlyFile(t, path)
}

// TruncatingHandler promises more bytes than it sends, then drops the
// connection — a download interrupted midway.
func TruncatingHandler(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1000")
		_, _ = w.Write([]byte(body))
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			_ = conn.Close()
		}
	})
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}

// assertOnlyFile checks that no temporary file was left next to path.
func assertOnlyFile(t *testing.T, path string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("dir has %d entries, want only %s", len(entries), filepath.Base(path))
	}
}
