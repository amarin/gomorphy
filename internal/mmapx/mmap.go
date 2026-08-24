// Package mmapx provides minimal read-only memory-mapped file access.
package mmapx

import (
	"fmt"
	"os"
	"syscall"
)

// Region is a read-only memory mapping of a whole file.
// The mapping stays valid until Close; slices derived from Bytes must not
// outlive it.
type Region struct {
	file *os.File
	data []byte
}

// Open maps path read-only.
func Open(path string) (*Region, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("mmapx: open %s: %w", path, err)
	}

	info, err := f.Stat()
	if err != nil {
		_ = f.Close()

		return nil, fmt.Errorf("mmapx: stat %s: %w", path, err)
	}

	if info.Size() == 0 {
		_ = f.Close()

		return nil, fmt.Errorf("mmapx: %s is empty", path)
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(info.Size()), syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		_ = f.Close()

		return nil, fmt.Errorf("mmapx: mmap %s: %w", path, err)
	}

	return &Region{file: f, data: data}, nil
}

// Bytes returns the mapped contents.
func (r *Region) Bytes() []byte { return r.data }

// Len returns the mapped size.
func (r *Region) Len() int { return len(r.data) }

// Close unmaps the region and closes the file.
func (r *Region) Close() error {
	err := syscall.Munmap(r.data)

	if ferr := r.file.Close(); err == nil {
		err = ferr
	}

	return err
}
