package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func newMergeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "merge",
		Short: "merge several dictionaries into one .dat (not yet implemented)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("merge: not yet implemented")
		},
	}
}
