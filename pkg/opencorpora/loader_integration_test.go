//go:build integration

// Integration tests require network access to opencorpora.org
// and overwrite files at .data/opencorpora. Run explicitly:
//
//	go test -tags=integration ./pkg/opencorpora/
package opencorpora_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/amarin/gomorphy/pkg/opencorpora"
)

func TestLoader_DownloadUpdate_Integration(t *testing.T) {
	skipIfOpencorporaDown(t)

	loader := opencorpora.NewLoader("")
	if _, err := loader.DownloadUpdate(); err != nil {
		t.Errorf("DownloadUpdate() error = %v", err)
	}
}

// skipIfOpencorporaDown skips the test when the upstream site is
// unreachable or answering with a server-side failure: the dictionary
// mirror is a third-party resource and its outages must not fail CI.
func skipIfOpencorporaDown(t *testing.T) {
	t.Helper()

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Head("https://dict.opencorpora.org/")
	if err != nil {
		t.Skipf("dict.opencorpora.org unreachable, skipping: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 500 {
		t.Skipf("dict.opencorpora.org unhealthy (HTTP %d), skipping", resp.StatusCode)
	}
}
