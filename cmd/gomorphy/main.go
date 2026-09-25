// Command gomorphy is the CLI for morphological analysis over dictionaries
// compiled into the unified GMOR format (lookup/lemmas/fuzzy/top), plus
// utilities to download, unpack, and build dictionaries from their sources
// (opencorpora, pymorphy2, unimorph) or a wordform TSV (import), and to merge
// compiled dictionaries. See docs/en/cli.md for the full command reference.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "gomorphy",
		Short: "gomorphy - morphological analysis over dictionaries in a unified format (GMOR)",
	}
	root.SilenceErrors = true
	root.SilenceUsage = true

	registerGlobalFlags(root)

	root.AddCommand(
		newCLICommand(),
		newLookupCommand(), newLemmasCommand(), newFuzzyCommand(), newTopCommand(),
		newDownloadCommand(), newUnpackCommand(), newBuildCommand(), newUpdateCommand(),
		newImportCommand(), newMergeCommand(), newSplitCommand(),
		newVersionCommand(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
