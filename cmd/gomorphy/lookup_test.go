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

// lookupPredictedDat saves a tiny Builder dictionary (кот/кота). Builder
// dictionaries carry a prediction DAWG, so "бота" is predicted.
func lookupPredictedDat(t *testing.T) string {
	t.Helper()
	return buildBuilderDat(t,
		[3]string{"кот", "кот", "NOUN,anim,masc,sing,nomn"},
		[3]string{"кота", "кот", "NOUN,anim,masc,sing,gent"},
	)
}

func TestDoLookup_MarksPredicted(t *testing.T) {
	m := mustResolveOne(t, lookupPredictedDat(t))
	defer func() { _ = m.Close() }()

	var buf bytes.Buffer
	require.NoError(t, doLookup(&buf, m, "бота"))
	assert.Contains(t, buf.String(), "\t(predicted)\n")

	buf.Reset()
	require.NoError(t, doLookup(&buf, m, "кота"))
	assert.NotContains(t, buf.String(), "(predicted)")
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
