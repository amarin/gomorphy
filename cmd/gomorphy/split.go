package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func newSplitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "split",
		Short: "split one dictionary into several .dat files (not yet implemented)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("split: not yet implemented")
		},
	}
}
