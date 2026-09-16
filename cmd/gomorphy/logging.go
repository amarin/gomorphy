package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/amarin/logging"
	"github.com/spf13/cobra"
)

// configureLogging reads -v/--verbose and -l/--log and initializes the
// process-wide logger. Must be called at the top of every command's RunE,
// before any other work.
func configureLogging(cmd *cobra.Command) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	logPath, _ := cmd.Flags().GetString("log")

	level := logging.LevelWarn
	if verbose {
		level = logging.LevelInfo
	}

	target := logging.StdErr
	if logPath != "" {
		dir := filepath.Dir(logPath)
		if _, err := os.Stat(dir); err != nil {
			return fmt.Errorf("log file directory %s: %w", dir, err)
		}
		target = logging.Target(logPath)
	}

	return logging.Init(logging.WithFormat(logging.FormatText), logging.WithTarget(target), logging.WithLevel(level))
}
