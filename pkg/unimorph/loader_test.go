package unimorph_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amarin/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/unimorph"
)

func TestMain(m *testing.M) {
	_ = logging.Init(
		logging.WithFormat(logging.FormatText),
		logging.WithTarget(logging.StdErr),
		logging.WithLevel(logging.LevelError),
	)
	os.Exit(m.Run())
}

const fixtureTSV = "кот\tкот\tN;NOM;SG\n"

func writeFixture(t *testing.T) string {
	t.Helper()
	dataPath := filepath.Join(t.TempDir(), ".data", "unimorph", "ru")
	require.NoError(t, os.MkdirAll(dataPath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dataPath, unimorph.LocalUnpackedFilename), []byte(fixtureTSV), 0o644))
	return dataPath
}

func TestNewLoader_UnsupportedLanguage(t *testing.T) {
	_, err := unimorph.NewLoader("en", "")
	assert.Error(t, err)
}

func TestNewLoader_DefaultDataPath(t *testing.T) {
	loader, err := unimorph.NewLoader("ru", "")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(".data", "unimorph", "ru"), loader.DataPath())
}

func TestLoader_Sync_LocalOnly(t *testing.T) {
	dataPath := writeFixture(t)
	loader, err := unimorph.NewLoader("ru", dataPath)
	require.NoError(t, err)

	assert.True(t, loader.IsDownloadExists())
	assert.True(t, loader.IsUnpackedExists(), "download and unpack are the same step for this source")

	require.NoError(t, loader.Sync(true))

	got, err := os.ReadFile(loader.UnpackedFilePath())
	require.NoError(t, err)
	assert.Equal(t, fixtureTSV, string(got))
}

func TestLoader_Sync_LocalOnly_NoFile(t *testing.T) {
	loader, err := unimorph.NewLoader("ru", filepath.Join(t.TempDir(), "missing"))
	require.NoError(t, err)

	err = loader.Sync(true)
	assert.Error(t, err)
}

func TestLoader_UnpackUpdate_NoOpWhenDownloaded(t *testing.T) {
	dataPath := writeFixture(t)
	loader, err := unimorph.NewLoader("ru", dataPath)
	require.NoError(t, err)

	require.NoError(t, loader.UnpackUpdate())
}

func TestLoader_UnpackUpdate_ErrorsWithoutDownload(t *testing.T) {
	loader, err := unimorph.NewLoader("ru", filepath.Join(t.TempDir(), "missing"))
	require.NoError(t, err)

	assert.Error(t, loader.UnpackUpdate())
}
