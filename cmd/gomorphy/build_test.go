package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func TestBuildCommand_OpenCorpora_WithInput(t *testing.T) {
	xmlPath := filepath.Join(t.TempDir(), "dict.xml")
	require.NoError(t, os.WriteFile(xmlPath, []byte(fixtureXML("кот")), 0o644))
	outPath := filepath.Join(t.TempDir(), "out.dat")

	root := newTestRootCmd(newBuildCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"build", "opencorpora", "-i", xmlPath, "-o", outPath})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "saved")

	d, err := morphology.Open(outPath)
	require.NoError(t, err)
	defer func() { _ = d.Close() }()
	assert.NotEmpty(t, d.Parse("кот"))
}

func TestBuildCommand_UnknownType(t *testing.T) {
	root := newTestRootCmd(newBuildCommand())
	root.SetArgs([]string{"build", "unimorph"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown dictionary type")
}

func TestBuildCommand_MissingInputFile(t *testing.T) {
	root := newTestRootCmd(newBuildCommand())
	root.SetArgs([]string{"build", "opencorpora", "-i", "/no/such/file.xml", "-o", filepath.Join(t.TempDir(), "out.dat")})
	assert.Error(t, root.Execute())
}
