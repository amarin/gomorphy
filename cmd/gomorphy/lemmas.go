package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// doLemmas writes every initial form of word to w, tab-separated: Normal,
// Tag. Unlike doLookup, no Dict component - the pre-redesign lemmas
// output never carried a Shard/Para composite either, and this spec did
// not decide to add one now.
func doLemmas(w io.Writer, m *morphology.MultiDictionary, word string) error {
	refs := m.Lemma(word)
	if len(refs) == 0 {
		return fmt.Errorf("lemmas %q: no lemmas", word)
	}
	for _, r := range refs {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", r.Normal, r.Tag)
	}
	return nil
}

func newLemmasCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "lemmas <word>",
		Short: "find initial forms (lemmas)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := configureLogging(cmd); err != nil {
				return err
			}
			m, err := resolveDictionaries(cmd)
			if err != nil {
				return err
			}
			defer func() { _ = m.Close() }()
			return doLemmas(cmd.OutOrStdout(), m, args[0])
		},
	}
}
