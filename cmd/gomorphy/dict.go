package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/amarin/gomorphy/pkg/morphology"
)

const dictionaryEnvVar = "GOMORPHY_DICTIONARY"

// resolveDictionaries opens every dictionary named by -d/--dictionary (a
// file, or a directory globbed for *.dat), falling back to
// $GOMORPHY_DICTIONARY (exactly one path, same file-or-directory handling)
// when -d was not given at all. Always returns a *MultiDictionary, even
// for a single resolved path - every dictionary-reading command goes
// through the same code path regardless of count.
func resolveDictionaries(cmd *cobra.Command) (*morphology.MultiDictionary, error) {
	paths, err := cmd.Flags().GetStringArray("dictionary")
	if err != nil {
		return nil, err
	}

	fromEnv := false
	if len(paths) == 0 {
		if envPath := os.Getenv(dictionaryEnvVar); envPath != "" {
			paths = []string{envPath}
			fromEnv = true
		}
	}

	var files []string
	for _, p := range paths {
		expanded, err := expandDictionaryPath(p)
		if err != nil {
			return nil, err
		}
		files = append(files, expanded...)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no dictionary specified: use -d/--dictionary or set $%s", dictionaryEnvVar)
	}

	if verbose, _ := cmd.Flags().GetBool("verbose"); verbose && fromEnv {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "resolved dictionary from $%s: %s\n", dictionaryEnvVar, paths[0])
	}

	dicts := make([]*morphology.Dictionary, 0, len(files))
	for _, f := range files {
		d, err := morphology.Open(f)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", f, err)
		}
		dicts = append(dicts, d)
	}
	return morphology.NewMultiDictionary(dicts...), nil
}

// expandDictionaryPath resolves one -d value: a file path as-is, or a
// directory into every *.dat file directly inside it (non-recursive).
func expandDictionaryPath(p string) ([]string, error) {
	info, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("dictionary path %s: %w", p, err)
	}
	if !info.IsDir() {
		return []string{p}, nil
	}
	matches, err := filepath.Glob(filepath.Join(p, "*.dat"))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no .dat files found in directory %s", p)
	}
	return matches, nil
}
