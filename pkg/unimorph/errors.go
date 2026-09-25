package unimorph

import "errors"

// ErrUnimorph is the sentinel error wrapped by the errors package unimorph
// creates itself (download, unpack, missing data; see errors.Is). Plain
// file-system errors (e.g. from os.Stat) may be returned unwrapped.
var ErrUnimorph = errors.New("unimorph")
