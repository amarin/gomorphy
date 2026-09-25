package common_test

import (
	"path/filepath"
	"testing"

	"github.com/amarin/gomorphy/pkg/common"
	"github.com/amarin/gomorphy/pkg/opencorpora"
	"github.com/amarin/gomorphy/pkg/pymorphy"
	"github.com/amarin/gomorphy/pkg/unimorph"
)

// This package's tests deliberately never call logging.Init: they cover a
// library caller that uses the loaders without configuring logging.

func TestNewLoaderLogger_WithoutLoggingInit(t *testing.T) {
	logger := common.NewLoaderLogger("loader")
	if logger == nil {
		t.Fatal("NewLoaderLogger() = nil")
	}
	logger.Info("discarded")
	logger.Debugf("discarded %d", 1)
	logger.WithKey("k", "v").WithLevel(0).Warn("discarded")
}

func TestLoaders_WithoutLoggingInit(t *testing.T) {
	dir := t.TempDir()

	pm := pymorphy.NewLoader(filepath.Join(dir, "pymorphy"))
	if pm.IsDownloadExists() || pm.IsUnpackedExists() {
		t.Error("pymorphy: empty dir reported as populated")
	}

	oc := opencorpora.NewLoader(filepath.Join(dir, "opencorpora"))
	if oc.IsDownloadExists() || oc.IsUnpackedExists() {
		t.Error("opencorpora: empty dir reported as populated")
	}

	um, err := unimorph.NewLoader("ru", filepath.Join(dir, "unimorph"))
	if err != nil {
		t.Fatalf("unimorph.NewLoader() error = %v", err)
	}
	if um.IsDownloadExists() || um.IsUnpackedExists() {
		t.Error("unimorph: empty dir reported as populated")
	}
}
