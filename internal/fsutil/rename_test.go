package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenameNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	dst := filepath.Join(dir, "b.txt")
	other := filepath.Join(dir, "c.txt")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RenameNoOverwrite(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatal(err)
	}
	if err := RenameNoOverwrite(dst, other); err == nil {
		t.Fatal("expected overwrite refusal")
	}
}
