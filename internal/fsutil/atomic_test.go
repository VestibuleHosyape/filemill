package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicCreatesAndReplaces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	if err := WriteFileAtomic(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "one" {
		t.Fatalf("got %q", got)
	}

	if err := WriteFileAtomic(path, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "two" {
		t.Fatalf("got %q", got)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected no leftover temps, got %d entries", len(entries))
	}
}

func TestWriteFileAtomicCreatesParentDirs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "c.txt")
	if err := WriteFileAtomic(path, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ok" {
		t.Fatalf("got %q", got)
	}
}

func TestFailedReplacementPreservesDestination(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "keep")
	if err := os.WriteFile(dst, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(filepath.Join(dir, "missing"), dst); err == nil {
		t.Fatal("expected rename failure")
	}
	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "original" {
		t.Fatalf("destination lost: %q %v", data, err)
	}
}

func TestFailedWriteCleansTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "directory")
	if err := os.Mkdir(dst, 0700); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(dst, []byte("data"), 0600); err == nil {
		t.Fatal("expected failure replacing directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || !entries[0].IsDir() {
		t.Fatalf("unexpected leftovers: %v %v", entries, err)
	}
}
