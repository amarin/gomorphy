package pymorphy_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/amarin/gomorphy/pkg/pymorphy"
)

// wheel builds a wheel archive holding files under the data/ subtree.
func wheel(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(pymorphy.WheelDataSubtree + name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// pypi serves a PyPI JSON document announcing version with a wheel at
// /wheel; wheelStatus other than 200 makes the wheel download fail. It
// counts requests.
func pypi(t *testing.T, version string, whl []byte, wheelStatus int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path == "/wheel" {
			if wheelStatus != http.StatusOK {
				w.WriteHeader(wheelStatus)
				return
			}
			_, _ = w.Write(whl)
			return
		}
		_, _ = fmt.Fprintf(w, `{"info":{"version":%q},"urls":[{"url":%q,"packagetype":"bdist_wheel"}]}`,
			version, srv.URL+"/wheel")
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func readWords(t *testing.T, loader *pymorphy.Loader) string {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(loader.UnpackedDirPath(), "words.dawg"))
	if err != nil {
		t.Fatal(err)
	}
	return string(got)
}

func TestLoader_Sync_DownloadsIntoCustomDataPath(t *testing.T) {
	t.Chdir(t.TempDir())
	srv, _ := pypi(t, "1.0", wheel(t, map[string]string{"words.dawg": "W1"}), http.StatusOK)
	loader := pymorphy.NewLoader(filepath.Join(t.TempDir(), "custom", "dir"))
	pymorphy.SetURL(loader, srv.URL)

	if err := loader.Sync(false); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got := readWords(t, loader); got != "W1" {
		t.Errorf("words.dawg = %q, want W1", got)
	}
	if v, _ := loader.LocalVersion(); v != "1.0" {
		t.Errorf("LocalVersion() = %q, want 1.0", v)
	}
	if _, err := os.Stat(".data"); !os.IsNotExist(err) {
		t.Errorf("default ./.data created for a custom dataPath (stat err = %v)", err)
	}
}

func TestLoader_Sync_UnpacksNewReleaseOverOldCopy(t *testing.T) {
	loader := pymorphy.NewLoader(writeFixture(t)) // version 9.9.9, has paradigms.array
	if err := loader.Sync(true); err != nil {
		t.Fatal(err)
	}
	srv, _ := pypi(t, "10.0", wheel(t, map[string]string{"words.dawg": "W10"}), http.StatusOK)
	pymorphy.SetURL(loader, srv.URL)

	if err := loader.Sync(false); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got := readWords(t, loader); got != "W10" {
		t.Errorf("words.dawg = %q, want the new release W10", got)
	}
	if _, err := os.Stat(filepath.Join(loader.UnpackedDirPath(), "paradigms.array")); !os.IsNotExist(err) {
		t.Error("a file of the old release survived the re-unpack")
	}
}

func TestLoader_Sync_BadStatusKeepsOldRelease(t *testing.T) {
	loader := pymorphy.NewLoader(writeFixture(t))
	if err := loader.Sync(true); err != nil {
		t.Fatal(err)
	}
	srv, _ := pypi(t, "10.0", nil, http.StatusNotFound)
	pymorphy.SetURL(loader, srv.URL)

	if err := loader.Sync(false); err == nil {
		t.Fatal("Sync() with a failing wheel download expected to fail")
	}
	if v, _ := loader.LocalVersion(); v != "9.9.9" {
		t.Errorf("LocalVersion() = %q, want the old 9.9.9", v)
	}
	if got := readWords(t, loader); got != "FAKE-WORDS-DAWG" {
		t.Errorf("words.dawg = %q, want the old release", got)
	}
}

func TestLoader_Sync_SkipDownloadMakesNoRequests(t *testing.T) {
	srv, hits := pypi(t, "10.0", nil, http.StatusOK)
	loader := pymorphy.NewLoader(writeFixture(t))
	pymorphy.SetURL(loader, srv.URL)

	if err := loader.Sync(true); err != nil {
		t.Fatal(err)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("Sync(skipDownload=true) made %d requests, want 0", n)
	}
}
