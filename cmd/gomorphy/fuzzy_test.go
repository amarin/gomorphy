package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoFuzzy_ExactMatch(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	m := mustResolveOne(t, path)
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	require.NoError(t, doFuzzy(&buf, m, "кот", 0))
	out := buf.String()
	assert.Contains(t, out, "кот")
	assert.Contains(t, out, "dict#0")
}

func TestFuzzyCommand_DefaultMaxDist(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))

	root := newTestRootCmd(newFuzzyCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"fuzzy", "-d", path, "кот"})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "кот")
}

func TestFuzzyCommand_InvalidMaxDist(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))

	root := newTestRootCmd(newFuzzyCommand())
	root.SetArgs([]string{"fuzzy", "-d", path, "кот", "not-a-number"})
	assert.Error(t, root.Execute())
}
