package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/common"
	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/amarin/gomorphy/pkg/opencorpora"
	"github.com/amarin/gomorphy/pkg/pymorphy"
)

// runBuild compiles typ's source into a .dat file. input, if non-empty,
// is compiled directly (skipping the loader's own unpacked-file/dir
// path); output, if empty, defaults to .data/<type>/<type>.dat.
func runBuild(cmd *cobra.Command, typ, input, output string) error {
	progress := newProgressReporterTo(cmd.OutOrStdout(), isInteractive())

	var d *morphology.Dictionary
	var err error
	var defaultOut string

	switch typ {
	case "opencorpora":
		xmlPath := input
		if xmlPath == "" {
			xmlPath = opencorpora.NewLoader("").UnpackedFilePath()
		}
		f, openErr := os.Open(xmlPath)
		if openErr != nil {
			return fmt.Errorf("build opencorpora: %w", openErr)
		}
		defer func() { _ = f.Close() }()
		d, err = morphology.CompileFromXML(f, progress)
		defaultOut = common.DomainFilePath(opencorpora.DomainName, "opencorpora.dat")
	case "pymorphy":
		dir := input
		if dir == "" {
			dir = pymorphy.NewLoader("").UnpackedDirPath()
		}
		// The CLI always builds a dense 1-byte alphabet, no opt-out flag —
		// an agreed design decision (see
		// docs/en/implementation/pymorphy2-dense-alphabet.md, "Agreed
		// design decisions", item 2). The raw/non-dense variant
		// (morphology.OpenPyMorphy) is Go-API-only, for embedders who want
		// it directly.
		d, err = morphology.OpenPyMorphyDense(dir)
		defaultOut = common.DomainFilePath(pymorphy.DomainName, "pymorphy.dat")
	default:
		return fmt.Errorf("unknown dictionary type %q (use opencorpora or pymorphy)", typ)
	}
	if err != nil {
		return fmt.Errorf("build %s: %w", typ, err)
	}

	if output == "" {
		output = defaultOut
	}
	if err := d.SaveTo(output); err != nil {
		return fmt.Errorf("save %s: %w", output, err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "saved %s\n", output)
	return nil
}

func newBuildCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build <type>",
		Short: "compile a dictionary source into a .dat file (opencorpora, pymorphy)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().StringP("input", "i", "", "compile this source path directly, skipping the loader")
	cmd.Flags().StringP("output", "o", "", "output .dat path (default: .data/<type>/<type>.dat)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := configureLogging(cmd); err != nil {
			return err
		}
		input, _ := cmd.Flags().GetString("input")
		output, _ := cmd.Flags().GetString("output")
		return runBuild(cmd, args[0], input, output)
	}
	return cmd
}
