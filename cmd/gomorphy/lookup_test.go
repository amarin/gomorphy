package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoLookup_Found(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	m := mustResolveOne(t, path)
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	require.NoError(t, doLookup(&buf, m, "кот"))
	out := buf.String()
	assert.Contains(t, out, "кот")
	assert.Contains(t, out, "NOUN")
	assert.Contains(t, out, "para#0/") // Dict=0, always-MultiDictionary
}

func TestDoLookup_NotFound(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	m := mustResolveOne(t, path)
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	assert.Error(t, doLookup(&buf, m, "несуществующееслово"))
}

func TestLookupCommand_EndToEnd(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))

	root := newTestRootCmd(newLookupCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"lookup", "-d", path, "кот"})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "кот")
}
