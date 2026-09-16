package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print gomorphy version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), morphology.Version)
			return nil
		},
	}
}
