package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// WriteFileAtomic writes data to path by first writing a temporary file in the
// same directory, then renaming it into place. On failure the temp file is removed.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建目录: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("创建临时文件: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName) // no-op after successful rename
	}()

	if runtime.GOOS != "windows" {
		if err := tmp.Chmod(perm); err != nil {
			_ = tmp.Close()
			return fmt.Errorf("设置权限: %w", err)
		}
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("写入临时文件: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("同步临时文件: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时文件: %w", err)
	}

	if err := replaceFile(tmpName, path); err != nil {
		return fmt.Errorf("原子替换: %w", err)
	}
	return nil
}

// replaceFile renames oldpath to newpath, replacing an existing newpath on Windows.
func replaceFile(oldpath, newpath string) error {
	err := os.Rename(oldpath, newpath)
	if err == nil {
		return nil
	}
	if runtime.GOOS != "windows" {
		return err
	}
	// Windows rename fails when the destination already exists.
	if rmErr := os.Remove(newpath); rmErr != nil && !os.IsNotExist(rmErr) {
		return err
	}
	return os.Rename(oldpath, newpath)
}
