//go:build !linux && !darwin && !windows

package fsutil

import "os"

// renameNoReplace falls back to link+unlink on platforms without a native
// no-replace rename. Link fails atomically if newpath already exists.
func renameNoReplace(oldpath, newpath string) error {
	if err := os.Link(oldpath, newpath); err != nil {
		return err
	}
	if err := os.Remove(oldpath); err != nil {
		_ = os.Remove(newpath)
		return err
	}
	return nil
}
