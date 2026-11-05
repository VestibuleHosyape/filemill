package dedupe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"filemill/internal/walker"
)

func fixture(t *testing.T, values map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range values {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func scan(t *testing.T, dir string, workers int) Result {
	t.Helper()
	files, err := walker.Collect(dir, walker.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return Find(files, workers)
}

func TestFindStableAndHashesOnlyCandidates(t *testing.T) {
	dir := fixture(t, map[string]string{"b": "same", "a": "same", "c": "diff", "unique": "longer", "empty-a": "", "empty-b": ""})
	r := scan(t, dir, 1)
	r2 := scan(t, dir, 4)
	j1, _ := json.Marshal(r)
	j2, _ := json.Marshal(r2)
	if string(j1) != string(j2) {
		t.Fatalf("unstable results: %+v / %+v", r, r2)
	}
	if r.Scanned != 6 || r.Hashed != 5 || len(r.Groups) != 2 || len(r.Errors) != 0 || r.ReclaimableBytes != 4 {
		t.Fatalf("unexpected report: %+v", r)
	}
	if filepath.Base(r.Groups[0].Keep.Path) != "a" {
		t.Fatalf("unexpected keeper: %+v", r.Groups[0])
	}
}

func TestRemoveVerified(t *testing.T) {
	dir := fixture(t, map[string]string{"a": "same", "b": "same", "c": "same"})
	r := scan(t, dir, 2)
	g := r.Groups[0]
	for _, f := range g.Duplicates {
		if err := RemoveVerified(g, f); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "a" {
		t.Fatalf("keeper not preserved: %v %v", entries, err)
	}
	if err := RemoveVerified(g, g.Keep); err == nil {
		t.Fatal("must reject keeper")
	}
}

func TestRemoveRefusesChangedFiles(t *testing.T) {
	for _, changed := range []string{"a", "b"} {
		t.Run(changed, func(t *testing.T) {
			dir := fixture(t, map[string]string{"a": "same", "b": "same"})
			g := scan(t, dir, 2).Groups[0]
			path := filepath.Join(dir, changed)
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("diff"), 0600); err != nil {
				t.Fatal(err)
			}
			// Same size and restored mtime must still fail the content check.
			if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
				t.Fatal(err)
			}
			if err := RemoveVerified(g, g.Duplicates[0]); err == nil {
				t.Fatal("accepted changed content")
			}
			if _, err := os.Stat(g.Duplicates[0].Path); err != nil {
				t.Fatal("duplicate removed", err)
			}
		})
	}
}

func TestRemoveRefusesReplacedKeeper(t *testing.T) {
	dir := fixture(t, map[string]string{"a": "same", "b": "same"})
	g := scan(t, dir, 2).Groups[0]
	if err := os.Rename(g.Keep.Path, filepath.Join(dir, "original")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(g.Keep.Path, []byte("same"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RemoveVerified(g, g.Duplicates[0]); err == nil {
		t.Fatal("accepted replaced keeper")
	}
}

func TestHardlinkAndSymlink(t *testing.T) {
	dir := fixture(t, map[string]string{"a": "same"})
	if err := os.Link(filepath.Join(dir, "a"), filepath.Join(dir, "b")); err != nil {
		t.Skip(err)
	}
	if r := scan(t, dir, 2); len(r.Groups) != 0 {
		t.Fatal("keeper hardlink reported as duplicate")
	}
	if err := os.Symlink(filepath.Join(dir, "a"), filepath.Join(dir, "c")); err != nil {
		t.Skip(err)
	}
	if r := scan(t, dir, 2); r.Scanned != 2 || len(r.Groups) != 0 {
		t.Fatalf("followed symlink: %+v", r)
	}
}

func TestHashFailureIsReported(t *testing.T) {
	dir := fixture(t, map[string]string{"a": "same", "b": "same", "c": "same"})
	files, err := walker.Collect(dir, walker.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "c")); err != nil {
		t.Fatal(err)
	}
	r := Find(files, 2)
	if len(r.Errors) != 1 || len(r.Groups) != 1 {
		t.Fatalf("unexpected result: %+v", r)
	}
}
