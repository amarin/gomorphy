package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/opencorpora"
	"github.com/amarin/gomorphy/pkg/pymorphy"
	"github.com/amarin/gomorphy/pkg/unimorph"
)

// runDownload downloads the source archive for typ ("opencorpora",
// "pymorphy", or "unimorph"), reporting the result on cmd's output
// writer. lang is only consulted for typ == "unimorph".
func runDownload(cmd *cobra.Command, typ, lang string) error {
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
	case "unimorph":
		loader, err := unimorph.NewLoader(lang, "")
		if err != nil {
			return fmt.Errorf("download unimorph: %w", err)
		}
		updated, err := loader.DownloadUpdate()
		if err != nil {
			return fmt.Errorf("download unimorph: %w", err)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "unimorph: updated=%v, %s\n", updated, loader.DataPath())
	default:
		return fmt.Errorf("unknown dictionary type %q (use opencorpora, pymorphy, or unimorph)", typ)
	}
	return nil
}

func newDownloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <type>",
		Short: "download the source archive for a dictionary type (opencorpora, pymorphy, unimorph)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().String("lang", "ru", "language code (unimorph only; only \"ru\" is supported today)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := configureLogging(cmd); err != nil {
			return err
		}
		lang, _ := cmd.Flags().GetString("lang")
		return runDownload(cmd, args[0], lang)
	}
	return cmd
}
