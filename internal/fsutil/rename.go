package fsutil

import (
	"fmt"
	"path/filepath"
)

// RenameNoOverwrite renames oldpath to newpath and refuses to replace an
// existing destination. Same cleaned path is a no-op.
//
// Existence is enforced by a platform-native no-replace rename (or an
// equivalent link+unlink fallback), avoiding a Stat/Rename TOCTOU race.
func RenameNoOverwrite(oldpath, newpath string) error {
	if filepath.Clean(oldpath) == filepath.Clean(newpath) {
		return nil
	}
	if err := renameNoReplace(oldpath, newpath); err != nil {
		if isExistErr(err) {
			return fmt.Errorf("目标已存在: %s", newpath)
		}
		return fmt.Errorf("重命名失败: %w", err)
	}
	return nil
}
