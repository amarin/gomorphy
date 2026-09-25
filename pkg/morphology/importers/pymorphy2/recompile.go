// RecompileDense (below) rebuilds an imported dictionary's
// words.dawg under a dense 1-byte alphabet. See
// docs/en/superpowers/specs/2026-09-16-pymorphy2-dense-recompile-design.md.

package pymorphy2

import (
	"fmt"

	"github.com/amarin/gomorphy/pkg/morphology/internal"
)

// RecompileDense imports dir like ImportFromDir, then rebuilds Words
// (Paradigms/Suffixes/Prefixes/Prediction/Probability are copied through
// unchanged, since they don't depend on words.dawg's key encoding) under
// a dense 1-byte alphabet built from the dictionary's own wordforms — see
// internal.RecompileDense, the source-agnostic implementation this
// delegates to (pymorphy2 imports are always exactly one shard, but the
// shared logic handles any shard count). The result's Parse() must return
// identical readings to ImportFromDir's, for any word the source
// dictionary itself resolves.
func RecompileDense(dir string) (*internal.Dictionary, error) {
	d, err := ImportFromDir(dir)
	if err != nil {
		return nil, err
	}
	if err := internal.RecompileDense(d); err != nil {
		return nil, fmt.Errorf("pymorphy2: recompile: %w", err)
	}
	return d, nil
}
