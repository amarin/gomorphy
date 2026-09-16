package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoLemmas_Found(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	m := mustResolveOne(t, path)
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	require.NoError(t, doLemmas(&buf, m, "кот"))
	assert.Contains(t, buf.String(), "кот")
}

func TestDoLemmas_NotFound(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	m := mustResolveOne(t, path)
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	assert.Error(t, doLemmas(&buf, m, "несуществующееслово"))
}

func TestLemmasCommand_EndToEnd(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))

	root := newTestRootCmd(newLemmasCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"lemmas", "-d", path, "кот"})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "кот")
}
