package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadCommand_UnknownType(t *testing.T) {
	root := newTestRootCmd(newDownloadCommand())
	root.SetArgs([]string{"download", "unimorph"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown dictionary type")
}

func TestDownloadCommand_RequiresExactlyOneArg(t *testing.T) {
	root := newTestRootCmd(newDownloadCommand())
	root.SetArgs([]string{"download"})
	assert.Error(t, root.Execute())

	root = newTestRootCmd(newDownloadCommand())
	root.SetArgs([]string{"download", "opencorpora", "pymorphy"})
	assert.Error(t, root.Execute())
}
