package opencorpora_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amarin/gomorphy/pkg/opencorpora"
)

// fixtureBz2v2 is a valid bzip2 stream of fixtureXMLv2 — a newer release.
const fixtureBz2v2 = "QlpoOTFBWSZTWRfazZkAAAgZgFAB6CeuJ51gIABISqaBskxAeo2kEqhpkwCNNMT0v1sKiwrOUieNA8K03iJbUZjNOXGHFEjC9nCAcHohughXk0LuSKcKEgL7WbMg"

const fixtureXMLv2 = `<?xml version="1.0"?><dictionary version="0.93"><lemmata/></dictionary>`

// source serves archive (base64) for GET, a Last-Modified of now for HEAD,
// or status for both when it is not 200; it counts requests.
func source(t *testing.T, archive string, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	body, err := base64.StdEncoding.DecodeString(archive)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		if r.Method == http.MethodGet {
			_, _ = w.Write(body)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func readUnpacked(t *testing.T, loader *opencorpora.Loader) string {
	t.Helper()
	got, err := os.ReadFile(loader.UnpackedFilePath())
	if err != nil {
		t.Fatal(err)
	}
	return string(got)
}

func TestLoader_Sync_DownloadsIntoCustomDataPath(t *testing.T) {
	t.Chdir(t.TempDir())
	srv, _ := source(t, fixtureBz2, http.StatusOK)
	loader := opencorpora.NewLoader(filepath.Join(t.TempDir(), "custom", "dir"))
	opencorpora.SetURL(loader, srv.URL)

	if err := loader.Sync(false); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got := readUnpacked(t, loader); got != fixtureXML {
		t.Errorf("dict.xml = %q, want %q", got, fixtureXML)
	}
	if _, err := os.Stat(".data"); !os.IsNotExist(err) {
		t.Errorf("default ./.data created for a custom dataPath (stat err = %v)", err)
	}
}

func TestLoader_Sync_UnpacksNewDownloadOverOldCopy(t *testing.T) {
	loader := opencorpora.NewLoader(writeFixture(t))
	if err := loader.Sync(true); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-24 * time.Hour)
	archive := filepath.Join(loader.DataPath(), opencorpora.LocalSourceFilename)
	if err := os.Chtimes(archive, old, old); err != nil {
		t.Fatal(err)
	}
	srv, _ := source(t, fixtureBz2v2, http.StatusOK)
	opencorpora.SetURL(loader, srv.URL)

	if err := loader.Sync(false); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if got := readUnpacked(t, loader); got != fixtureXMLv2 {
		t.Errorf("dict.xml = %q, want the new release %q", got, fixtureXMLv2)
	}
}

func TestLoader_Sync_BadStatus(t *testing.T) {
	srv, _ := source(t, fixtureBz2, http.StatusInternalServerError)
	loader := opencorpora.NewLoader(filepath.Join(t.TempDir(), "d"))
	opencorpora.SetURL(loader, srv.URL)

	if err := loader.Sync(false); err == nil {
		t.Fatal("Sync() with a failing server expected to fail")
	}
	if loader.IsDownloadExists() {
		t.Error("an error response must not be saved as the archive")
	}
}

func TestLoader_Sync_SkipDownloadMakesNoRequests(t *testing.T) {
	srv, hits := source(t, fixtureBz2v2, http.StatusOK)
	loader := opencorpora.NewLoader(writeFixture(t))
	opencorpora.SetURL(loader, srv.URL)

	if err := loader.Sync(true); err != nil {
		t.Fatal(err)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("Sync(skipDownload=true) made %d requests, want 0", n)
	}
}
