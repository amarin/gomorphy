package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// runImportTSV imports a tab-separated wordform stream from filePath into a
// .dat at output. source fills BuildInfo.Source; ImportTSV defaults it to
// "tsv" when empty. output is required — the tsv importer has no default
// domain path to fall back on (mirrors build.go's runBuild shape).
func runImportTSV(cmd *cobra.Command, filePath, output, source string) error {
	if output == "" {
		return fmt.Errorf("import tsv: -o output path is required")
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("import tsv: %w", err)
	}
	defer func() { _ = f.Close() }()

	d, err := morphology.ImportTSV(f, morphology.BuilderOptions{Source: source})
	if err != nil {
		return fmt.Errorf("import tsv: %w", err)
	}

	if err := d.SaveTo(output); err != nil {
		return fmt.Errorf("save %s: %w", output, err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "saved %s\n", output)
	return nil
}

func newImportCommand() *cobra.Command {
	parent := &cobra.Command{
		Use:   "import",
		Short: "import a dictionary source into a .dat file",
	}

	tsv := &cobra.Command{
		Use:   "tsv <file>",
		Short: "import a tab-separated wordform TSV into a .dat",
		Args:  cobra.ExactArgs(1),
	}
	tsv.Flags().StringP("output", "o", "", "output .dat path (required)")
	tsv.Flags().String("source", "", "value for BuildInfo.Source (default: \"tsv\")")
	tsv.RunE = func(cmd *cobra.Command, args []string) error {
		if err := configureLogging(cmd); err != nil {
			return err
		}
		output, _ := cmd.Flags().GetString("output")
		source, _ := cmd.Flags().GetString("source")
		return runImportTSV(cmd, args[0], output, source)
	}

	parent.AddCommand(tsv)
	return parent
}
