package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/opencorpora"
	"github.com/amarin/gomorphy/pkg/pymorphy"
)

// runUnpack extracts the already-downloaded source archive for typ.
func runUnpack(cmd *cobra.Command, typ string) error {
	switch typ {
	case "opencorpora":
		loader := opencorpora.NewLoader("")
		if err := loader.UnpackUpdate(); err != nil {
			return fmt.Errorf("unpack opencorpora: %w", err)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "opencorpora: unpacked to %s\n", loader.UnpackedFilePath())
	case "pymorphy":
		loader := pymorphy.NewLoader("")
		if err := loader.UnpackUpdate(); err != nil {
			return fmt.Errorf("unpack pymorphy: %w", err)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "pymorphy: unpacked to %s\n", loader.UnpackedDirPath())
	default:
		return fmt.Errorf("unknown dictionary type %q (use opencorpora or pymorphy)", typ)
	}
	return nil
}

func newUnpackCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unpack <type>",
		Short: "extract the downloaded source archive for a dictionary type (opencorpora, pymorphy)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogging(cmd); err != nil {
				return err
			}
			return runUnpack(cmd, args[0])
		},
	}
}
