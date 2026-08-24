//go:build integration

// Integration tests require network access to opencorpora.org
// and overwrite files at .data/opencorpora. Run explicitly:
//
//	go test -tags=integration ./pkg/opencorpora/
package opencorpora_test

import (
	"testing"

	"github.com/amarin/gomorphy/pkg/opencorpora"
)

func TestLoader_DownloadUpdate_Integration(t *testing.T) {
	loader := opencorpora.NewLoader("")
	if _, err := loader.DownloadUpdate(); err != nil {
		t.Errorf("DownloadUpdate() error = %v", err)
	}
}
