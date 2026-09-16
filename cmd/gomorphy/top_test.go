package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoTop_ReturnsUpToN(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	m := mustResolveOne(t, path)
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	require.NoError(t, doTop(&buf, m, "кот", 1))
	assert.Contains(t, buf.String(), "кот")
}

func TestTopCommand_InvalidN(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))

	root := newTestRootCmd(newTopCommand())
	root.SetArgs([]string{"top", "-d", path, "кот", "0"})
	assert.Error(t, root.Execute())
}
