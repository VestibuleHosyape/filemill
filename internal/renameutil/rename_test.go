package renameutil

import (
	"path/filepath"
	"testing"
)

func TestApplyPrefixSuffixSeq(t *testing.T) {
	opts := Options{Prefix: "new_", Suffix: "_v", Seq: true, SeqStart: 1, SeqWidth: 3}
	if err := opts.Compile(); err != nil {
		t.Fatal(err)
	}
	got, err := opts.Apply("a.txt", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != "new_a_v_001.txt" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyRegex(t *testing.T) {
	opts := Options{Find: `^old_`, Repl: "new_"}
	if err := opts.Compile(); err != nil {
		t.Fatal(err)
	}
	got, err := opts.Apply("old_file.log", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != "new_file.log" {
		t.Fatalf("got %q", got)
	}
}

func TestPlanDetectsCollision(t *testing.T) {
	dir := t.TempDir()
	paths := []string{
		filepath.Join(dir, "a.txt"),
		filepath.Join(dir, "b.txt"),
	}
	_, err := Plan(paths, Options{Find: `.*`, Repl: "same.txt"})
	if err == nil {
		t.Fatal("expected collision error")
	}
}

func TestPlanSkipUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keep.txt")
	maps, err := Plan([]string{path}, Options{Prefix: "" /* still need a rule */, Find: `keep`, Repl: "keep"})
	if err != nil {
		t.Fatal(err)
	}
	if len(maps) != 1 || !maps[0].Skipped {
		t.Fatalf("want skipped mapping, got %+v", maps)
	}
}

func TestHasRuleRequired(t *testing.T) {
	_, err := Plan([]string{"a.txt"}, Options{})
	if err == nil {
		t.Fatal("expected error when no rules")
	}
}
