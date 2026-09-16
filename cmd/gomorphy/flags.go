package main

import "github.com/spf13/cobra"

// registerGlobalFlags attaches the flags every dictionary-aware and
// logging-aware command shares (-d/--dictionary, -v/--verbose, -l/--log)
// as persistent flags on root, so every subcommand inherits them without
// redeclaring anything.
func registerGlobalFlags(root *cobra.Command) {
	root.PersistentFlags().StringArrayP("dictionary", "d", nil, "path to a .dat file or a directory of .dat files (repeatable)")
	root.PersistentFlags().BoolP("verbose", "v", false, "verbose logging and diagnostics")
	root.PersistentFlags().StringP("log", "l", "", "log to this file instead of stderr")
}
