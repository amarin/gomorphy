package common

import (
	"os"
	"path"
	"path/filepath"
)

// GetDataPath returns the root directory gomorphy stores downloaded and
// built data under, relative to the current working directory.
func GetDataPath() string {
	return ".data"
}

// DomainDataPath returns the data directory for a single domain (e.g.
// "opencorpora"), rooted at GetDataPath.
func DomainDataPath(domain string) string {
	return path.Join(GetDataPath(), domain)
}

// MakeDomainDataPath ensures the data directory for domain exists,
// creating it (and any parents) if necessary.
func MakeDomainDataPath(domain string) (err error) {
	domainDataPath := DomainDataPath(domain)
	if domainDataPath, err = filepath.Abs(domainDataPath); err != nil {
		return err
	}

	if err := os.MkdirAll(domainDataPath, os.ModePerm); err != nil {
		return err
	}

	return nil
}

// DomainFilePath joins a file name onto a domain's data directory.
func DomainFilePath(domain string, fileName string) string {
	return path.Join(DomainDataPath(domain), fileName)
}
