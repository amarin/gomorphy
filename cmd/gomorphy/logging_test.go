package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureLogging_DefaultsToStderrWarn(t *testing.T) {
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		return configureLogging(cmd)
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe"})
	assert.NoError(t, root.Execute())
}

func TestConfigureLogging_Verbose(t *testing.T) {
	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		return configureLogging(cmd)
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-v"})
	assert.NoError(t, root.Execute())
}

func TestConfigureLogging_LogFileExistingDir(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "gomorphy.log")

	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		return configureLogging(cmd)
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-l", logPath})
	require.NoError(t, root.Execute())
}

func TestConfigureLogging_LogFileMissingParentDir(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "does-not-exist", "gomorphy.log")

	probe := &cobra.Command{Use: "probe", RunE: func(cmd *cobra.Command, args []string) error {
		return configureLogging(cmd)
	}}
	root := newTestRootCmd(probe)
	root.SetArgs([]string{"probe", "-l", logPath})
	assert.Error(t, root.Execute())

	_, statErr := os.Stat(logPath)
	assert.True(t, os.IsNotExist(statErr), "must not create the log file when its parent dir is missing")
}
