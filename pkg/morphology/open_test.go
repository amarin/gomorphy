package morphology_test

import (
	"path/filepath"
	"testing"

	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenPyMorphy(t *testing.T) {
	d := parseDict(t)

	assert.Equal(t, "ru", d.Language())

	readings := d.Parse("кот")
	require.NotEmpty(t, readings)
	assert.Equal(t, "кот", readings[0].Word)
}

func TestOpenPyMorphyError(t *testing.T) {
	_, err := morphology.OpenPyMorphy(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
}
