package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnpackCommand_UnknownType(t *testing.T) {
	root := newTestRootCmd(newUnpackCommand())
	root.SetArgs([]string{"unpack", "klingon"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown dictionary type")
}
