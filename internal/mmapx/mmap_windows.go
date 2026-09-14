//go:build windows

// Package mmapx provides minimal read-only memory-mapped file access.
package mmapx

import "fmt"

// Region is a read-only memory mapping of a whole file.
type Region struct{}

// Open is not implemented on Windows yet — gomorphy currently targets Unix
// platforms only (see docs/todo.md for planned Windows mmap support).
func Open(path string) (*Region, error) {
	return nil, fmt.Errorf("mmapx: %s: not supported on windows yet", path)
}

// Bytes returns the mapped contents.
func (r *Region) Bytes() []byte { return nil }

// Len returns the mapped size.
func (r *Region) Len() int { return 0 }

// Close unmaps the region and closes the file.
func (r *Region) Close() error { return nil }
