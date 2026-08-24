package opencorpora_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/pkg/opencorpora"
)

func TestMain(m *testing.M) {
	_ = logging.Init(
		logging.WithFormat(logging.FormatText),
		logging.WithTarget(logging.StdErr),
		logging.WithLevel(logging.LevelError),
	)
	os.Exit(m.Run())
}

// fixtureBz2 is a valid bzip2 stream of fixtureXML.
const fixtureBz2 = "QlpoOTFBWSZTWffJbtMAAAgZgFAB8CeuJ51gIABISqaBskxAeo2kEqhpkwCNNMT0v1sKiwrOUieNA8K03iJbUZjNOXGHFEjC9nCAcHohughXk0LuSKcKEh75Ldpg"

const fixtureXML = `<?xml version="1.0"?><dictionary version="0.92"><lemmata/></dictionary>`

func writeFixture(t *testing.T) string {
	t.Helper()

	dataPath := filepath.Join(t.TempDir(), ".data", "opencorpora")
	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	bz, err := base64.StdEncoding.DecodeString(fixtureBz2)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dataPath, opencorpora.LocalSourceFilename), bz, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	return dataPath
}

func TestLoader_UnpackUpdate(t *testing.T) {
	dataPath := writeFixture(t)
	loader := opencorpora.NewLoader(dataPath)

	if loader.IsDownloadExists() != true {
		t.Fatal("downloaded fixture expected to exist")
	}
	if loader.IsUnpackedExists() == true {
		t.Fatal("unpacked file must not exist before unpack")
	}

	if err := loader.UnpackUpdate(); err != nil {
		t.Fatalf("UnpackUpdate() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(loader.DataPath(), opencorpora.LocalUnpackedFilename))
	if err != nil {
		t.Fatalf("read unpacked: %v", err)
	}
	if string(got) != fixtureXML {
		t.Errorf("unpacked content = %q, want %q", got, fixtureXML)
	}
}

func TestLoader_Update_LocalOnly_OK(t *testing.T) {
	dataPath := writeFixture(t)
	loader := opencorpora.NewLoader(dataPath)

	if err := loader.Update(true); err != nil {
		t.Fatalf("Update(local only) error = %v", err)
	}
}

func TestLoader_Update_LocalOnly_NoArchive(t *testing.T) {
	loader := opencorpora.NewLoader(filepath.Join(t.TempDir(), "missing"))

	err := loader.Update(true)
	if err == nil {
		t.Fatal("Update(local only) without archive expected to fail")
	}
}
