//go:build integration

// Integration tests require network access to raw.githubusercontent.com
// and write to .data/unimorph/ru. Run explicitly:
//
//	go test -tags=integration ./pkg/unimorph/
package unimorph_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/amarin/gomorphy/pkg/unimorph"
)

func TestLoader_DownloadUpdate_Integration(t *testing.T) {
	skipIfUnimorphDown(t)

	loader, err := unimorph.NewLoader("ru", "")
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}
	if _, err := loader.DownloadUpdate(); err != nil {
		t.Errorf("DownloadUpdate() error = %v", err)
	}
}

// skipIfUnimorphDown skips the test when raw.githubusercontent.com is
// unreachable or answering with a server-side failure: GitHub's raw CDN
// is a third-party resource and its outages must not fail CI.
func skipIfUnimorphDown(t *testing.T) {
	t.Helper()

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Head("https://raw.githubusercontent.com/unimorph/rus/master/rus")
	if err != nil {
		t.Skipf("raw.githubusercontent.com unreachable, skipping: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 500 {
		t.Skipf("raw.githubusercontent.com unhealthy (HTTP %d), skipping", resp.StatusCode)
	}
}
