package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/opencorpora"
	"github.com/amarin/gomorphy/pkg/pymorphy"
)

// runDownload downloads the source archive for typ ("opencorpora" or
// "pymorphy"), reporting the result on cmd's output writer.
func runDownload(cmd *cobra.Command, typ string) error {
	switch typ {
	case "opencorpora":
		loader := opencorpora.NewLoader("")
		updated, err := loader.DownloadUpdate()
		if err != nil {
			return fmt.Errorf("download opencorpora: %w", err)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "opencorpora: updated=%v, %s\n", updated, loader.DataPath())
	case "pymorphy":
		loader := pymorphy.NewLoader("")
		updated, err := loader.DownloadUpdate()
		if err != nil {
			return fmt.Errorf("download pymorphy: %w", err)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "pymorphy: updated=%v, %s\n", updated, loader.DataPath())
	default:
		return fmt.Errorf("unknown dictionary type %q (use opencorpora or pymorphy)", typ)
	}
	return nil
}

func newDownloadCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "download <type>",
		Short: "download the source archive for a dictionary type (opencorpora, pymorphy)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogging(cmd); err != nil {
				return err
			}
			return runDownload(cmd, args[0])
		},
	}
}
