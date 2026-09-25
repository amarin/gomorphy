package unimorph_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/unimorph"
)

// source serves body with status for every request and counts requests.
func source(t *testing.T, body string, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestLoader_Sync_SkipDownloadMakesNoRequests(t *testing.T) {
	srv, hits := source(t, "new", http.StatusOK)
	loader, err := unimorph.NewLoader("ru", writeFixture(t))
	require.NoError(t, err)
	unimorph.SetURL(loader, srv.URL)

	require.NoError(t, loader.Sync(true))
	assert.Zero(t, hits.Load(), "Sync(skipDownload=true) must not touch the network")
}

func TestLoader_DownloadUpdate_BadStatusKeepsOldFile(t *testing.T) {
	srv, _ := source(t, "error page", http.StatusNotFound)
	loader, err := unimorph.NewLoader("ru", writeFixture(t))
	require.NoError(t, err)
	unimorph.SetURL(loader, srv.URL)

	require.NoError(t, os.Chtimes(loader.UnpackedFilePath(), time.Unix(0, 0), time.Unix(0, 0)))
	_, err = loader.DownloadUpdate()
	require.Error(t, err)
	got, err := os.ReadFile(loader.UnpackedFilePath())
	require.NoError(t, err)
	assert.Equal(t, fixtureTSV, string(got))
}
