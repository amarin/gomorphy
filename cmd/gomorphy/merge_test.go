package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// Grammeme tags the merge CLI fixture corpora use.
const (
	mergeCLIBaseTag    = "NOUN,anim,masc,sing,nomn"
	mergeCLIOverlayTag = "VERB,impf,trans"
)

// buildBuilderDat builds a dictionary from (word, lemma, tag) triples via the
// public morphology.Builder and saves it to a .dat in t.TempDir() — the
// Builder-based fixture helper for the import/merge CLI tests (the XML-based
// buildFixtureDat stays for the opencorpora-based tests).
func buildBuilderDat(t *testing.T, triples ...[3]string) string {
	t.Helper()
	b := morphology.NewBuilder(morphology.BuilderOptions{})
	for _, tr := range triples {
		require.NoError(t, b.AddForm(tr[0], tr[1], tr[2]))
	}
	d, err := b.Build()
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "fixture.dat")
	require.NoError(t, d.SaveTo(path))
	return path
}

// openTags opens a .dat and returns the tags of word's readings, in order.
func openTags(t *testing.T, path, word string) []string {
	t.Helper()
	d, err := morphology.Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, d.Close()) }()
	var tags []string
	for _, r := range d.Parse(word) {
		tags = append(tags, r.Tag)
	}
	return tags
}

func TestMergeCommand_Help(t *testing.T) {
	root := newTestRootCmd(newMergeCommand())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"merge", "-h"})
	assert.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "merge")
}

func TestMergeCommand_MissingOutput(t *testing.T) {
	base := buildBuilderDat(t, [3]string{"кот", "кот", mergeCLIBaseTag})

	root := newTestRootCmd(newMergeCommand())
	root.SetArgs([]string{"merge", "--mode", "add", base, base})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "-o", "error must name the missing -o flag")
}

func TestMergeCommand_Add(t *testing.T) {
	base := buildBuilderDat(t,
		[3]string{"кот", "кот", mergeCLIBaseTag},
		[3]string{"кота", "кот", "NOUN,anim,masc,sing,gent"},
		[3]string{"база", "база", mergeCLIBaseTag},
	)
	overlay := buildBuilderDat(t,
		[3]string{"кот", "кот", mergeCLIOverlayTag}, // in base: dropped under add
		[3]string{"оверлей", "оверлей", mergeCLIOverlayTag},
	)
	out := filepath.Join(t.TempDir(), "merged.dat")

	root := newTestRootCmd(newMergeCommand())
	root.SetArgs([]string{"merge", "--mode", "add", "-o", out, base, overlay})
	require.NoError(t, root.Execute())

	assert.Equal(t, []string{mergeCLIBaseTag}, openTags(t, out, "кот"), "shared word keeps only the base reading")
	assert.Equal(t, []string{mergeCLIBaseTag}, openTags(t, out, "база"), "base-only word preserved")
	assert.Equal(t, []string{mergeCLIOverlayTag}, openTags(t, out, "оверлей"), "overlay-only word added")
}

func TestMergeCommand_AddCaseInsensitive(t *testing.T) {
	base := buildBuilderDat(t, [3]string{"кот", "кот", mergeCLIBaseTag})
	overlay := buildBuilderDat(t, [3]string{"оверлей", "оверлей", mergeCLIOverlayTag})
	out := filepath.Join(t.TempDir(), "merged.dat")

	root := newTestRootCmd(newMergeCommand())
	root.SetArgs([]string{"merge", "--mode", "ADD", "-o", out, base, overlay})
	require.NoError(t, root.Execute(), "--mode must be validated case-insensitively")

	assert.Equal(t, []string{mergeCLIBaseTag}, openTags(t, out, "кот"))
	assert.Equal(t, []string{mergeCLIOverlayTag}, openTags(t, out, "оверлей"))
}

func TestMergeCommand_Replace(t *testing.T) {
	base := buildBuilderDat(t,
		[3]string{"кот", "кот", mergeCLIBaseTag},
		[3]string{"база", "база", mergeCLIBaseTag},
	)
	overlay := buildBuilderDat(t,
		[3]string{"кот", "кот", mergeCLIOverlayTag}, // in base: replaced
		[3]string{"оверлей", "оверлей", mergeCLIOverlayTag},
	)
	out := filepath.Join(t.TempDir(), "merged.dat")

	root := newTestRootCmd(newMergeCommand())
	root.SetArgs([]string{"merge", "--mode", "replace", "-o", out, base, overlay})
	require.NoError(t, root.Execute())

	assert.Equal(t, []string{mergeCLIOverlayTag}, openTags(t, out, "кот"), "shared word keeps only the overlay reading")
	assert.Equal(t, []string{mergeCLIBaseTag}, openTags(t, out, "база"), "base-only word preserved")
	assert.Equal(t, []string{mergeCLIOverlayTag}, openTags(t, out, "оверлей"), "overlay-only word preserved")
}

func TestMergeCommand_TooFewArgs(t *testing.T) {
	root := newTestRootCmd(newMergeCommand())
	root.SetArgs([]string{"merge", "--mode", "add", "-o", filepath.Join(t.TempDir(), "out.dat"), "only-one.dat"})
	err := root.Execute()
	require.Error(t, err, "fewer than two dictionaries must be rejected")
}

func TestMergeCommand_MissingMode(t *testing.T) {
	base := buildBuilderDat(t, [3]string{"кот", "кот", mergeCLIBaseTag})
	out := filepath.Join(t.TempDir(), "merged.dat")

	root := newTestRootCmd(newMergeCommand())
	root.SetArgs([]string{"merge", "-o", out, base, base})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mode")
}

func TestMergeCommand_UnknownMode(t *testing.T) {
	base := buildBuilderDat(t, [3]string{"кот", "кот", mergeCLIBaseTag})
	out := filepath.Join(t.TempDir(), "merged.dat")

	root := newTestRootCmd(newMergeCommand())
	root.SetArgs([]string{"merge", "--mode", "nope", "-o", out, base, base})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nope", "error must name the offending mode")
}

func TestMergeCommand_OutputOverwritesInput(t *testing.T) {
	base := buildBuilderDat(t, [3]string{"кот", "кот", mergeCLIBaseTag})

	root := newTestRootCmd(newMergeCommand())
	root.SetArgs([]string{"merge", "--mode", "add", "-o", base, base, base})
	err := root.Execute()
	require.Error(t, err, "-o equal to an input .dat must be rejected, not silently overwrite it")
}
