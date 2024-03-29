package common

import (
	"errors"
	"os"
	"path"
	"path/filepath"
)

// ErrPath represents filepath related errors.
var ErrPath = errors.New("path")

func GetDataPath() string {
	return ".data"
}

func DomainDataPath(domain string) string {
	return path.Join(GetDataPath(), domain)
}

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

func DomainFilePath(domain string, fileName string) string {
	return path.Join(DomainDataPath(domain), fileName)
}
