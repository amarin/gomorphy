package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests prove the core property the MultiDictionary/resolveDictionaries
// design exists for: when multiple -d/--dictionary paths are given, each
// match's Dict index actually identifies which dictionary it came from, in
// registration order. Every other command test in this package uses exactly
// one fixture dictionary, so Dict/dict# is only ever asserted as 0 - these
// tests register two distinct fixture dictionaries and assert index 0 vs 1.

func TestLookupCommand_DictIndexTracksCorrectDictionary(t *testing.T) {
	path1 := buildFixtureDat(t, fixtureXML("кот"))
	path2 := buildFixtureDat(t, fixtureXML("яблоко"))

	root := newTestRootCmd(newLookupCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"lookup", "-d", path1, "-d", path2, "кот"})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "para#0/")

	root2 := newTestRootCmd(newLookupCommand())
	var buf2 bytes.Buffer
	root2.SetOut(&buf2)
	root2.SetArgs([]string{"lookup", "-d", path1, "-d", path2, "яблоко"})
	require.NoError(t, root2.Execute())
	assert.Contains(t, buf2.String(), "para#1/")
}

func TestFuzzyCommand_DictIndexTracksCorrectDictionary(t *testing.T) {
	path1 := buildFixtureDat(t, fixtureXML("кот"))
	path2 := buildFixtureDat(t, fixtureXML("яблоко"))

	root := newTestRootCmd(newFuzzyCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"fuzzy", "-d", path1, "-d", path2, "кот"})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "dict#0")

	root2 := newTestRootCmd(newFuzzyCommand())
	var buf2 bytes.Buffer
	root2.SetOut(&buf2)
	root2.SetArgs([]string{"fuzzy", "-d", path1, "-d", path2, "яблоко"})
	require.NoError(t, root2.Execute())
	assert.Contains(t, buf2.String(), "dict#1")
}

func TestTopCommand_DictIndexTracksCorrectDictionary(t *testing.T) {
	path1 := buildFixtureDat(t, fixtureXML("кот"))
	path2 := buildFixtureDat(t, fixtureXML("яблоко"))

	root := newTestRootCmd(newTopCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"top", "-d", path1, "-d", path2, "кот", "1"})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "dict#0")

	root2 := newTestRootCmd(newTopCommand())
	var buf2 bytes.Buffer
	root2.SetOut(&buf2)
	root2.SetArgs([]string{"top", "-d", path1, "-d", path2, "яблоко", "1"})
	require.NoError(t, root2.Execute())
	assert.Contains(t, buf2.String(), "dict#1")
}
