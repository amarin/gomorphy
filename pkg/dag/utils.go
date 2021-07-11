package dag

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	indexExt = "idx"
	nodesExt = "idn"
)

func ResolveFilePath(filePath string) (dirname, filename, ext string, err error) {
	var (
		fileStat        os.FileInfo
		absPath         string
		filenameWithExt string
	)

	absPath, err = filepath.Abs(filePath)
	if err != nil {
		return "", "", "", fmt.Errorf("%w: resolve path: %v: %v", ErrIndex, filePath, err)
	}

	dirname, filenameWithExt = filepath.Split(absPath)
	ext = filepath.Ext(filenameWithExt)
	fileNameLen := len(filenameWithExt) - len(ext)
	fileName := filenameWithExt[:fileNameLen]

	fileStat, err = os.Stat(absPath)
	switch {
	case err != nil:
		return dirname, fileName, ext, fmt.Errorf("%w: path: %v: %v", ErrIndex, dirname, err)
	case fileStat.Name() != filenameWithExt:
		return dirname, fileName, ext, fmt.Errorf("%w: expected dir: %v: %v", ErrIndex, dirname, err)
	}

	return dirname, fileName, ext, nil
}
