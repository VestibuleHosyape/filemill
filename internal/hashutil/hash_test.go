package hashutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileDigests(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		MD5:    "5d41402abc4b2a76b9719d911017c592",
		SHA1:   "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d",
		SHA256: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
	}
	for algo, want := range cases {
		got, err := File(path, algo)
		if err != nil {
			t.Fatalf("%s: %v", algo, err)
		}
		if got != want {
			t.Fatalf("%s: got %s want %s", algo, got, want)
		}
	}
}

func TestNewRejectsUnknown(t *testing.T) {
	if _, err := New("blake3"); err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeAndFileErrors(t *testing.T) {
	for input, want := range map[string]string{" MD5 ": MD5, "SHA-1": SHA1, " SHA-256 ": SHA256} {
		got, err := Normalize(input)
		if err != nil || got != want {
			t.Fatalf("Normalize(%q)=%q %v", input, got, err)
		}
	}
	if _, err := Normalize("unknown"); err == nil {
		t.Fatal("accepted unknown algorithm")
	}
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := File(missing, SHA256); err == nil {
		t.Fatal("accepted missing file")
	}
	if _, err := File(missing, "unknown"); err == nil {
		t.Fatal("accepted unknown algorithm")
	}
	if _, err := File(t.TempDir(), SHA256); err == nil {
		t.Fatal("accepted directory")
	}
}
