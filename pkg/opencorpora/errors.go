package opencorpora

import (
	"errors"
)

// ErrOpenCorpora is the sentinel error wrapped by package opencorpora's
// failure modes (see errors.Is).
var ErrOpenCorpora = errors.New("opencorpora")
