package index

import (
	"fmt"
)

// ErrNilWriter indicates error when nil writer specified to BinaryWriteTo.
var ErrNilWriter = fmt.Errorf("%w: nil writer", Error)

// ErrNilReader indicates error when nil writer specified to BinaryReadFrom.
var ErrNilReader = fmt.Errorf("%w: nil reader", Error)
