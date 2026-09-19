package unimorph

import "errors"

// ErrUnimorph is the sentinel error wrapped by package unimorph's
// failure modes (see errors.Is).
var ErrUnimorph = errors.New("unimorph")
