package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// fixtureXML returns a minimal valid OpenCorpora dict.xml with exactly one
// NOUN lemma, word. Real, schema-correct XML (same shape already proven
// against the real xmlscan parser by
// pkg/morphology/importers/opencorpora/import_test.go's testDictXML) -
// this package cannot reach pkg/morphology/internal or that test's own
// fixture, so it builds its own via the public morphology API only.
func fixtureXML(word string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<dictionary corpus="opencorpora" russian="yes">
 <grammemes>
  <grammeme id="NOUN">сущ</grammeme>
  <grammeme id="nomn">им.</grammeme>
  <grammeme id="anim">од.</grammeme>
  <grammeme id="masc">м.</grammeme>
  <grammeme id="sing">ед.</grammeme>
 </grammemes>
 <lemmata>
  <lemma id="1" text=%q>
   <l t=%q><g v="NOUN"/><g v="anim"/><g v="masc"/><g v="sing"/></l>
   <f t=%q><g v="nomn"/></f>
  </lemma>
 </lemmata>
</dictionary>`, word, word, word)
}

// buildFixtureDat compiles xml into a real .dat file in t.TempDir(),
// through the public morphology API only.
func buildFixtureDat(t *testing.T, xml string) string {
	t.Helper()
	d, err := morphology.CompileFromXML(strings.NewReader(xml), nil)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "fixture.dat")
	require.NoError(t, d.SaveTo(path))
	return path
}

// newTestRootCmd builds a throwaway root with the standard global flags
// registered (mirroring production's real root in main.go) and cmd
// attached as its only child - the shape every command needs to run the
// way it really will. Shared across this package's command tests.
func newTestRootCmd(cmd *cobra.Command) *cobra.Command {
	root := &cobra.Command{Use: "gomorphy"}
	registerGlobalFlags(root)
	root.AddCommand(cmd)
	return root
}

func TestResolveDictionaries_SingleFile(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))

	var got *morphology.MultiDictionary
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		got, err = resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-d", path})
	require.NoError(t, root.Execute())

	require.NotNil(t, got)
	assert.Equal(t, 1, got.Len(), "resolveDictionaries always returns a MultiDictionary, even for one path")
	assert.NotEmpty(t, got.Parse("кот"))
}

func TestResolveDictionaries_MultipleFlags(t *testing.T) {
	pathA := buildFixtureDat(t, fixtureXML("кот"))
	pathB := buildFixtureDat(t, fixtureXML("яблоко"))

	var got *morphology.MultiDictionary
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		got, err = resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-d", pathA, "-d", pathB})
	require.NoError(t, root.Execute())

	require.Equal(t, 2, got.Len())
	assert.NotEmpty(t, got.Parse("кот"))
	assert.NotEmpty(t, got.Parse("яблоко"))
}

func TestResolveDictionaries_Directory(t *testing.T) {
	dir := t.TempDir()
	d, err := morphology.CompileFromXML(strings.NewReader(fixtureXML("кот")), nil)
	require.NoError(t, err)
	require.NoError(t, d.SaveTo(filepath.Join(dir, "a.dat")))
	d2, err := morphology.CompileFromXML(strings.NewReader(fixtureXML("яблоко")), nil)
	require.NoError(t, err)
	require.NoError(t, d2.SaveTo(filepath.Join(dir, "b.dat")))
	// Non-.dat file in the same directory must be ignored, not error.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("x"), 0o644))

	var got *morphology.MultiDictionary
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		got, err = resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-d", dir})
	require.NoError(t, root.Execute())

	assert.Equal(t, 2, got.Len())
}

func TestResolveDictionaries_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		_, err := resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-d", dir})
	assert.Error(t, root.Execute())
}

func TestResolveDictionaries_NoFlagNoEnv(t *testing.T) {
	t.Setenv(dictionaryEnvVar, "")

	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		_, err := resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe"})
	assert.Error(t, root.Execute())
}

func TestResolveDictionaries_EnvFallback(t *testing.T) {
	path := buildFixtureDat(t, fixtureXML("кот"))
	t.Setenv(dictionaryEnvVar, path)

	var got *morphology.MultiDictionary
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		got, err = resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe"})
	require.NoError(t, root.Execute())

	assert.Equal(t, 1, got.Len())
	assert.NotEmpty(t, got.Parse("кот"))
}

// mustResolveOne wraps a single fixture .dat path into a *MultiDictionary
// via the real resolveDictionaries code path (not a direct
// NewMultiDictionary call) - every command test in this package needs
// exactly this, and going through resolveDictionaries is what proves the
// "always MultiDictionary, even for one path" contract at every call site
// that uses this helper.
func mustResolveOne(t *testing.T, path string) *morphology.MultiDictionary {
	t.Helper()
	var got *morphology.MultiDictionary
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		got, err = resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-d", path})
	require.NoError(t, root.Execute())
	return got
}

func TestResolveDictionaries_FlagTakesPriorityOverEnv(t *testing.T) {
	envPath := buildFixtureDat(t, fixtureXML("груша"))
	flagPath := buildFixtureDat(t, fixtureXML("кот"))
	t.Setenv(dictionaryEnvVar, envPath)

	var got *morphology.MultiDictionary
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		got, err = resolveDictionaries(cmd)
		return err
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-d", flagPath})
	require.NoError(t, root.Execute())

	assert.Equal(t, 1, got.Len())
	assert.NotEmpty(t, got.Parse("кот"))
	assert.Empty(t, got.Parse("груша"), "env path must not be used when -d was given")
}
