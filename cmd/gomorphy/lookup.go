package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

// doLookup writes every reading of word to w, tab-separated: Word, Normal,
// Tag, a para#Dict/Shard/Para composite (Dict is always 0 for a
// single-dictionary resolution, since resolveDictionaries always produces
// a MultiDictionary), and the source dictionary's name/version (dictLabel).
func doLookup(w io.Writer, m *morphology.MultiDictionary, word string) error {
	readings := m.Parse(word)
	if len(readings) == 0 {
		return fmt.Errorf("lookup %q: no readings", word)
	}
	for _, r := range readings {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\tpara#%d/%d/%d\t%s\n",
			r.Word, r.Normal, r.Tag, r.Dict, r.Shard, r.Para, dictLabel(m, r.Dict))
	}
	return nil
}

func newLookupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "lookup <word>",
		Short: "exact dictionary lookup (all readings)",
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
			return doLookup(cmd.OutOrStdout(), m, args[0])
		},
	}
}
