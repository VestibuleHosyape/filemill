package walker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectExtAndGlob(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.txt"), "hello")
	mustWrite(t, filepath.Join(root, "b.log"), "log")
	mustWrite(t, filepath.Join(root, "sub", "c.txt"), "world")
	mustWrite(t, filepath.Join(root, "sub", "d.md"), "md")

	files, err := Collect(root, Options{Exts: []string{".txt"}})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("ext=.txt want 2 files, got %d: %+v", len(files), names(files))
	}

	files, err = Collect(root, Options{NameGlob: "*.log"})
	if err != nil {
		t.Fatalf("Collect glob: %v", err)
	}
	if len(files) != 1 || filepath.Base(files[0].Path) != "b.log" {
		t.Fatalf("glob=*.log want b.log, got %+v", names(files))
	}
}

func TestCollectRegexAndSize(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "keep_1.dat"), "12345")
	mustWrite(t, filepath.Join(root, "keep_2.dat"), "1")
	mustWrite(t, filepath.Join(root, "skip.dat"), "xxxxx")

	files, err := Collect(root, Options{
		NameRegex: `^keep_`,
		MinSize:   3,
	})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(files) != 1 || filepath.Base(files[0].Path) != "keep_1.dat" {
		t.Fatalf("want keep_1.dat, got %+v", names(files))
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func names(files []FileInfo) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = filepath.Base(f.Path)
	}
	return out
}
