package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// doFuzzy writes every fuzzy match within maxDist of word to w,
// tab-separated: Distance, Word, dict#Dict, and the source dictionary's
// name/version (dictLabel).
func doFuzzy(w io.Writer, m *morphology.MultiDictionary, word string, maxDist int) error {
	matches := m.Fuzzy(word, maxDist)
	for _, mt := range matches {
		_, _ = fmt.Fprintf(w, "%d\t%s\tdict#%d\t%s\n", mt.Distance, mt.Word, mt.Dict, dictLabel(m, mt.Dict))
	}
	return nil
}

func newFuzzyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "fuzzy <word> [maxDist]",
		Short: "fuzzy search within edit distance (default 2)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogging(cmd); err != nil {
				return err
			}
			maxDist := 2
			if len(args) == 2 {
				n, err := parseNonNegativeInt(args[1])
				if err != nil {
					return fmt.Errorf("invalid maxDist %q", args[1])
				}
				maxDist = n
			}
			m, err := resolveDictionaries(cmd)
			if err != nil {
				return err
			}
			defer func() { _ = m.Close() }()
			return doFuzzy(cmd.OutOrStdout(), m, args[0], maxDist)
		},
	}
}
