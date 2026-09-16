package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// doTop writes up to n nearest matches of word to w, tab-separated:
// Distance, Word, dict#Dict.
func doTop(w io.Writer, m *morphology.MultiDictionary, word string, n int) error {
	matches := m.FuzzyTop(word, n)
	for _, mt := range matches {
		_, _ = fmt.Fprintf(w, "%d\t%s\tdict#%d\n", mt.Distance, mt.Word, mt.Dict)
	}
	return nil
}

func newTopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "top <word> <N>",
		Short: "N nearest words by edit distance",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogging(cmd); err != nil {
				return err
			}
			n, err := parseNonNegativeInt(args[1])
			if err != nil || n == 0 {
				return fmt.Errorf("invalid N %q", args[1])
			}
			m, err := resolveDictionaries(cmd)
			if err != nil {
				return err
			}
			defer func() { _ = m.Close() }()
			return doTop(cmd.OutOrStdout(), m, args[0], n)
		},
	}
}
