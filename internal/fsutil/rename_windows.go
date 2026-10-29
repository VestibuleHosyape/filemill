//go:build windows

package fsutil

import "golang.org/x/sys/windows"

func renameNoReplace(oldpath, newpath string) error {
	from, err := windows.UTF16PtrFromString(oldpath)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(newpath)
	if err != nil {
		return err
	}
	// flags=0: do not replace an existing destination (no MOVEFILE_REPLACE_EXISTING).
	return windows.MoveFileEx(from, to, 0)
}
