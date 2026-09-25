package opencorpora

import (
	"errors"
)

// ErrOpenCorpora is the sentinel error wrapped by the errors package opencorpora
// creates itself (download, unpack, missing data; see errors.Is). Plain
// file-system errors (e.g. from os.Stat) may be returned unwrapped.
var ErrOpenCorpora = errors.New("opencorpora")
