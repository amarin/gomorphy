package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "gomorphy",
		Short: "gomorphy — морфологический анализ слов по словарям в едином формате (GMOR)",
	}
	root.SilenceErrors = true
	root.SilenceUsage = true

	registerGlobalFlags(root)

	root.AddCommand(
		newCLICommand(),
		newLookupCommand(), newLemmasCommand(), newFuzzyCommand(), newTopCommand(),
		newDownloadCommand(), newUnpackCommand(), newBuildCommand(), newUpdateCommand(),
		newMergeCommand(), newSplitCommand(),
		newVersionCommand(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
