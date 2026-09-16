package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func TestVersionCommand(t *testing.T) {
	root := newTestRootCmd(newVersionCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"version"})
	require.NoError(t, root.Execute())
	assert.Equal(t, morphology.Version, strings.TrimSpace(buf.String()))
}
