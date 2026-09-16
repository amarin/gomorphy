package pymorphy

import "errors"

// ErrPymorphy is the sentinel error wrapped by package pymorphy's failure
// modes (see errors.Is).
var ErrPymorphy = errors.New("pymorphy")
