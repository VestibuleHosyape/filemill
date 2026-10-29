//go:build unix

package fsutil

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func isExistErr(err error) bool {
	return errors.Is(err, unix.EEXIST) || errors.Is(err, os.ErrExist) || os.IsExist(err)
}
