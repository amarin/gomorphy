package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/opencorpora"
	"github.com/amarin/gomorphy/pkg/pymorphy"
	"github.com/amarin/gomorphy/pkg/unimorph"
)

// runUnpack extracts the already-downloaded source archive for typ. lang
// is only consulted for typ == "unimorph" (unpack is a no-op there — see
// unimorph.Loader's doc comment — kept so every source type goes through
// the same command uniformly).
func runUnpack(cmd *cobra.Command, typ, lang string) error {
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
	case "unimorph":
		loader, err := unimorph.NewLoader(lang, "")
		if err != nil {
			return fmt.Errorf("unpack unimorph: %w", err)
		}
		if err := loader.UnpackUpdate(); err != nil {
			return fmt.Errorf("unpack unimorph: %w", err)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "unimorph: unpacked to %s\n", loader.UnpackedFilePath())
	default:
		return fmt.Errorf("unknown dictionary type %q (use opencorpora, pymorphy, or unimorph)", typ)
	}
	return nil
}

func newUnpackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unpack <type>",
		Short: "extract the downloaded source archive for a dictionary type (opencorpora, pymorphy, unimorph)",
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().String("lang", "ru", "language code (unimorph only; only \"ru\" is supported today)")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if err := configureLogging(cmd); err != nil {
			return err
		}
		lang, _ := cmd.Flags().GetString("lang")
		return runUnpack(cmd, args[0], lang)
	}
	return cmd
}
