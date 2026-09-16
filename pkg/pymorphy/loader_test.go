package pymorphy_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/pkg/pymorphy"
)

func TestMain(m *testing.M) {
	_ = logging.Init(
		logging.WithFormat(logging.FormatText),
		logging.WithTarget(logging.StdErr),
		logging.WithLevel(logging.LevelError),
	)
	os.Exit(m.Run())
}

// fixtureWheel is a valid ZIP (wheel) archive containing
// pymorphy2_dicts_ru/data/{words.dawg,paradigms.array} plus one non-data
// file (pymorphy2_dicts_ru/__init__.py) that unpack must skip.
const fixtureWheel = "UEsDBBQAAAAIAPNSMF0wxGtqEQAAAA8AAAAiAAAAcHltb3JwaHkyX2RpY3RzX3J1L2RhdGEvd29yZHMuZGF3Z3Nz9HbVDfcPcgnWdXEMdwcAUEsDBBQAAAAIAPNSMF2yQrI6EAAAAA4AAAAnAAAAcHltb3JwaHkyX2RpY3RzX3J1L2RhdGEvcGFyYWRpZ21zLmFycmF5c3P0dtUNcAxydPF09w0GAFBLAwQUAAAACADzUjBdyicAZx0AAAAbAAAAHgAAAHB5bW9ycGh5Ml9kaWN0c19ydS9fX2luaXRfXy5weVNWyMsvUUhJLEnUUcgtLS5RSEpVKM7OLChITQEAUEsBAhQDFAAAAAgA81IwXTDEa2oRAAAADwAAACIAAAAAAAAAAAAAAIABAAAAAHB5bW9ycGh5Ml9kaWN0c19ydS9kYXRhL3dvcmRzLmRhd2dQSwECFAMUAAAACADzUjBdskKyOhAAAAAOAAAAJwAAAAAAAAAAAAAAgAFRAAAAcHltb3JwaHkyX2RpY3RzX3J1L2RhdGEvcGFyYWRpZ21zLmFycmF5UEsBAhQDFAAAAAgA81IwXconAGcdAAAAGwAAAB4AAAAAAAAAAAAAAIABpgAAAHB5bW9ycGh5Ml9kaWN0c19ydS9fX2luaXRfXy5weVBLBQYAAAAAAwADAPEAAAD/AAAAAAA="

func writeFixture(t *testing.T) string {
	t.Helper()

	dataPath := filepath.Join(t.TempDir(), ".data", "pymorphy")
	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	zipBytes, err := base64.StdEncoding.DecodeString(fixtureWheel)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dataPath, pymorphy.LocalArchiveFilename), zipBytes, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataPath, pymorphy.LocalVersionFilename), []byte("9.9.9"), 0o644); err != nil {
		t.Fatalf("write version fixture: %v", err)
	}

	return dataPath
}

func TestLoader_Sync(t *testing.T) {
	dataPath := writeFixture(t)
	loader := pymorphy.NewLoader(dataPath)

	if loader.IsDownloadExists() != true {
		t.Fatal("downloaded fixture expected to exist")
	}
	if loader.IsUnpackedExists() == true {
		t.Fatal("unpacked dir must not exist before unpack")
	}

	if err := loader.Sync(true); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	words, err := os.ReadFile(filepath.Join(loader.UnpackedDirPath(), "words.dawg"))
	if err != nil {
		t.Fatalf("read unpacked words.dawg: %v", err)
	}
	if string(words) != "FAKE-WORDS-DAWG" {
		t.Errorf("words.dawg content = %q, want %q", words, "FAKE-WORDS-DAWG")
	}

	paradigms, err := os.ReadFile(filepath.Join(loader.UnpackedDirPath(), "paradigms.array"))
	if err != nil {
		t.Fatalf("read unpacked paradigms.array: %v", err)
	}
	if string(paradigms) != "FAKE-PARADIGMS" {
		t.Errorf("paradigms.array content = %q, want %q", paradigms, "FAKE-PARADIGMS")
	}

	if _, err := os.Stat(filepath.Join(loader.UnpackedDirPath(), "__init__.py")); err == nil {
		t.Error("__init__.py (outside the wheel's data/ subtree) must not be extracted")
	}
}

func TestLoader_Update_LocalOnly_OK(t *testing.T) {
	dataPath := writeFixture(t)
	loader := pymorphy.NewLoader(dataPath)

	if err := loader.Sync(true); err != nil {
		t.Fatalf("Sync(local only) error = %v", err)
	}
}

func TestLoader_Update_LocalOnly_NoArchive(t *testing.T) {
	loader := pymorphy.NewLoader(filepath.Join(t.TempDir(), "missing"))

	err := loader.Sync(true)
	if err == nil {
		t.Fatal("Sync(local only) without archive expected to fail")
	}
}

func TestLoader_LocalVersion(t *testing.T) {
	dataPath := writeFixture(t)
	loader := pymorphy.NewLoader(dataPath)

	version, ok := loader.LocalVersion()
	if !ok {
		t.Fatal("LocalVersion() ok = false, want true")
	}
	if version != "9.9.9" {
		t.Errorf("LocalVersion() = %q, want %q", version, "9.9.9")
	}
}

func TestLoader_LocalVersion_Missing(t *testing.T) {
	loader := pymorphy.NewLoader(filepath.Join(t.TempDir(), "missing"))

	if _, ok := loader.LocalVersion(); ok {
		t.Fatal("LocalVersion() ok = true, want false for missing version file")
	}
}
