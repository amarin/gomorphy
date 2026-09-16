package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitCommand_Help(t *testing.T) {
	root := newTestRootCmd(newSplitCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"split", "-h"})
	assert.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "split")
}

func TestSplitCommand_NotImplemented(t *testing.T) {
	root := newTestRootCmd(newSplitCommand())
	root.SetArgs([]string{"split"})
	err := root.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}
