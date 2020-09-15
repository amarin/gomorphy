package common

import (
	"errors"
)

var (
	ErrUnmarshal      = errors.New("unpack error")
	ErrUnknownNode    = errors.New("unknown node")
	ErrEmptyValue     = errors.New("empty value")
	ErrChildrenError  = errors.New("node children")
	ErrIndexSize      = errors.New("index size")
	ErrUnexpectedItem = errors.New("unexpected item")
)
