package walker

import (
	"errors"
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

func TestExcludePrunesDirectories(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.txt", ".git/config", "node_modules/pkg/a.js", "sub/skip.log", "sub/keep.txt"} {
		mustWrite(t, filepath.Join(dir, filepath.FromSlash(name)), "x")
	}
	files, err := Collect(dir, Options{Exclude: []string{".git", "node_modules", "sub/*.log"}})
	if err != nil || len(files) != 2 {
		t.Fatalf("exclusion failed: %v %v", names(files), err)
	}
}

func TestRelativeExcludeHasConsistentSlashSemantics(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "sub", "skip.log"), "x")
	mustWrite(t, filepath.Join(dir, "sub", "deep", "keep.log"), "x")
	files, err := Collect(dir, Options{Exclude: []string{"sub/*.log"}})
	if err != nil || len(files) != 1 || filepath.Base(files[0].Path) != "keep.log" {
		t.Fatalf("glob crossed directory separator: %v %v", files, err)
	}
}

func TestInvalidFiltersFailEvenOnEmptyDirectory(t *testing.T) {
	for _, opts := range []Options{{NameGlob: "["}, {Exclude: []string{"["}}, {MinSize: -1}, {MinSize: 10, MaxSize: 1}} {
		if _, err := Collect(t.TempDir(), opts); err == nil {
			t.Fatalf("accepted %+v", opts)
		}
	}
}

func TestCallbackErrorAborts(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a"), "a")
	mustWrite(t, filepath.Join(dir, "b"), "b")
	want := errors.New("stop")
	calls := 0
	err := Walk(dir, Options{}, func(FileInfo) error { calls++; return want })
	if !errors.Is(err, want) || calls != 1 {
		t.Fatalf("callback error not propagated: %v %d", err, calls)
	}
}

func TestSymlinkRootSkipped(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file")
	mustWrite(t, target, "x")
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skip(err)
	}
	files, err := Collect(link, Options{})
	if err != nil || len(files) != 0 {
		t.Fatalf("followed root symlink: %v %v", files, err)
	}
}

func TestScanErrorContinuesAccessibleSiblings(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.txt"), "a")
	mustWrite(t, filepath.Join(dir, "z.txt"), "z")
	vanishing := filepath.Join(dir, "middle")
	if err := os.Mkdir(vanishing, 0700); err != nil {
		t.Fatal(err)
	}
	var seen []string
	err := Walk(dir, Options{}, func(f FileInfo) error {
		seen = append(seen, filepath.Base(f.Path))
		if filepath.Base(f.Path) == "a.txt" {
			return os.Remove(vanishing)
		}
		return nil
	})
	if err == nil || len(seen) != 2 || seen[1] != "z.txt" {
		t.Fatalf("scan did not isolate error: %v %v", seen, err)
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
