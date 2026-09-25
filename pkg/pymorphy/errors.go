package pymorphy

import "errors"

// ErrPymorphy is the sentinel error wrapped by the errors package pymorphy
// creates itself (download, unpack, missing data; see errors.Is). Plain
// file-system errors (e.g. from os.Stat) may be returned unwrapped.
var ErrPymorphy = errors.New("pymorphy")
