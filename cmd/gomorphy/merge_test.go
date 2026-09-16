package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeCommand_Help(t *testing.T) {
	root := newTestRootCmd(newMergeCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"merge", "-h"})
	assert.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "merge")
}

func TestMergeCommand_NotImplemented(t *testing.T) {
	root := newTestRootCmd(newMergeCommand())
	root.SetArgs([]string{"merge"})
	err := root.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}
