//go:build windows

package fsutil

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func isExistErr(err error) bool {
	return errors.Is(err, windows.ERROR_ALREADY_EXISTS) ||
		errors.Is(err, windows.ERROR_FILE_EXISTS) ||
		errors.Is(err, os.ErrExist) ||
		os.IsExist(err)
}
