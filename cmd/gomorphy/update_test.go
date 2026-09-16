package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCommand_UnknownType(t *testing.T) {
	root := newTestRootCmd(newUpdateCommand())
	root.SetArgs([]string{"update", "unimorph"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown dictionary type")
}
