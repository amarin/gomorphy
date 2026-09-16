package main

import (
	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <type>",
		Short: "download + unpack + build a dictionary in one step (opencorpora, pymorphy)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().StringP("output", "o", "", "output .dat path (default: .data/<type>/<type>.dat)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := configureLogging(cmd); err != nil {
			return err
		}
		typ := args[0]
		if err := runDownload(cmd, typ); err != nil {
			return err
		}
		if err := runUnpack(cmd, typ); err != nil {
			return err
		}
		output, _ := cmd.Flags().GetString("output")
		return runBuild(cmd, typ, "", output)
	}
	return cmd
}
