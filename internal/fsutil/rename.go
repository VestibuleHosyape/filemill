package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// RenameNoOverwrite renames oldpath to newpath and refuses to replace an
// existing destination. Same cleaned path is a no-op.
func RenameNoOverwrite(oldpath, newpath string) error {
	if filepath.Clean(oldpath) == filepath.Clean(newpath) {
		return nil
	}
	if _, err := os.Stat(newpath); err == nil {
		return fmt.Errorf("目标已存在: %s", newpath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("检查目标: %w", err)
	}
	if err := os.Rename(oldpath, newpath); err != nil {
		return fmt.Errorf("重命名失败: %w", err)
	}
	return nil
}
