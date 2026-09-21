package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// runMerge combines args[0] (the base dictionary) with the overlay
// dictionaries args[1:] into a new .dat at output, under mergeMode
// ("add" or "replace"). output is required (merge must never silently
// overwrite an input — the abs-ified output path is rejected when it
// equals any input path), and the mode is validated case-insensitively.
func runMerge(cmd *cobra.Command, args []string, output, mergeMode string) error {
	if output == "" {
		return fmt.Errorf("merge: -o output path is required")
	}

	var mode morphology.MergeMode
	switch strings.ToLower(mergeMode) {
	case "add":
		mode = morphology.MergeAdd
	case "replace":
		mode = morphology.MergeReplace
	case "":
		return fmt.Errorf("merge: --mode add|replace is required")
	default:
		return fmt.Errorf("merge: unknown mode %q (use add or replace)", mergeMode)
	}

	// Reject an output path that aliases an input: a merge must not
	// silently overwrite one of its own sources.
	absOut, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("merge: resolve output path: %w", err)
	}
	for _, in := range args {
		absIn, err := filepath.Abs(in)
		if err != nil {
			return fmt.Errorf("merge: resolve input %s: %w", in, err)
		}
		if absOut == absIn {
			return fmt.Errorf("merge: output %q must not overwrite input %s", output, in)
		}
	}

	// Open the inputs as immutable mmap'd dictionaries (they are opened
	// read-only by morphology.Open and closed when Merge — the only
	// consumer — has finished collecting their entries).
	dicts := make([]*morphology.Dictionary, 0, len(args))
	for _, in := range args {
		d, err := morphology.Open(in)
		if err != nil {
			return fmt.Errorf("merge: open %s: %w", in, err)
		}
		dicts = append(dicts, d)
	}
	defer func() {
		for _, d := range dicts {
			_ = d.Close()
		}
	}()

	merged, err := morphology.Merge(dicts[0], dicts[1:], mode)
	if err != nil {
		return fmt.Errorf("merge: %w", err)
	}

	if err := merged.SaveTo(output); err != nil {
		return fmt.Errorf("save %s: %w", output, err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "saved %s\n", output)
	return nil
}

func newMergeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge",
		Short: "merge several dictionaries into one .dat (add or replace)",
		Args:  cobra.MinimumNArgs(2),
	}
	cmd.Flags().StringP("output", "o", "", "output .dat path (required)")
	cmd.Flags().String("mode", "", "merge mode: add or replace (required)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := configureLogging(cmd); err != nil {
			return err
		}
		output, _ := cmd.Flags().GetString("output")
		mode, _ := cmd.Flags().GetString("mode")
		return runMerge(cmd, args, output, mode)
	}
	return cmd
}
