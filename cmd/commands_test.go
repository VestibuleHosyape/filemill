package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Each CLI run starts with fresh Cobra flags and exercises the real exit code.
func TestCLIHelper(t *testing.T) {
	if os.Getenv("FILEMILL_TEST_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			rootCmd.SetArgs(os.Args[i+1:])
			break
		}
	}
	Execute()
	os.Exit(0)
}

func cli(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	c := exec.Command(os.Args[0], append([]string{"-test.run=^TestCLIHelper$", "--"}, args...)...)
	c.Env = append(os.Environ(), "FILEMILL_TEST_HELPER=1")
	var out, stderr bytes.Buffer
	c.Stdout, c.Stderr = &out, &stderr
	err := c.Run()
	return out.String(), stderr.String(), err
}

func writeFixture(t *testing.T, dir, name, data string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDedupeCLIConfirmationMatrix(t *testing.T) {
	for _, tc := range []struct {
		name    string
		flags   []string
		deleted bool
		invalid bool
	}{
		{"report", nil, false, false},
		{"preview", []string{"--delete"}, false, false},
		{"apply", []string{"--delete", "--apply"}, true, false},
		{"yes", []string{"--delete", "--yes"}, true, false},
		{"dry-run-wins", []string{"--delete", "--apply", "--dry-run"}, false, false},
		{"missing-delete", []string{"--apply"}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			keeper := writeFixture(t, dir, "a", "same")
			duplicate := writeFixture(t, dir, "b", "same")
			out, stderr, err := cli(t, append([]string{"dedupe", dir, "--output", "json"}, tc.flags...)...)
			if tc.invalid {
				if err == nil {
					t.Fatal("expected invalid flags error")
				}
				return
			}
			if err != nil {
				t.Fatalf("CLI: %v %s", err, stderr)
			}
			var report dedupeReport
			if err := json.Unmarshal([]byte(out), &report); err != nil {
				t.Fatal(err, out)
			}
			if len(report.Groups) != 1 || report.ReclaimableBytes != 4 {
				t.Fatalf("unexpected report: %+v", report)
			}
			if data, err := os.ReadFile(keeper); err != nil || string(data) != "same" {
				t.Fatal("keeper changed")
			}
			_, statErr := os.Stat(duplicate)
			if tc.deleted != os.IsNotExist(statErr) {
				t.Fatalf("unexpected deletion: %v", statErr)
			}
		})
	}
}

func TestRenameCLIPartialFailureAndDryRun(t *testing.T) {
	for _, flags := range [][]string{nil, {"--apply"}, {"--apply", "--dry-run"}} {
		dir := t.TempDir()
		writeFixture(t, dir, "a.txt", "source")
		writeFixture(t, dir, "b.txt", "second")
		target := writeFixture(t, dir, "new_a.txt", "original")
		out, stderr, err := cli(t, append([]string{"rename", dir, "--name", "?.txt", "--prefix", "new_"}, flags...)...)
		if err == nil {
			t.Fatal("partial failure returned zero", out, stderr)
		}
		if data, err := os.ReadFile(target); err != nil || string(data) != "original" {
			t.Fatal("overwrote existing destination")
		}
		if len(flags) == 1 {
			if _, err := os.Stat(filepath.Join(dir, "new_b.txt")); err != nil {
				t.Fatal("did not continue after failure", err)
			}
		} else if _, err := os.Stat(filepath.Join(dir, "b.txt")); err != nil {
			t.Fatal("dry run modified file")
		}
	}
}

func TestHashCLIFormatsAndValidation(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "a", "data")
	for _, format := range []string{"json", "csv"} {
		out, stderr, err := cli(t, "hash", dir, "--output", format)
		if err != nil {
			t.Fatal(err, stderr)
		}
		if format == "json" {
			var records []hashRecord
			if err := json.Unmarshal([]byte(out), &records); err != nil || len(records) != 1 || len(records[0].Hash) != 64 {
				t.Fatalf("invalid JSON output: %s %v", out, err)
			}
		} else {
			rows, err := csv.NewReader(strings.NewReader(out)).ReadAll()
			if err != nil || len(rows) != 2 {
				t.Fatalf("invalid CSV: %s %v", out, err)
			}
		}
	}
	out, _, err := cli(t, "hash", t.TempDir(), "--output", "json")
	if err != nil || strings.TrimSpace(out) != "[]" {
		t.Fatalf("empty result: %q %v", out, err)
	}
	for _, args := range [][]string{{"hash", filepath.Join(dir, "missing")}, {"list", dir, "--name", "["}, {"list", dir, "--workers", "0"}, {"list", dir, "--min-size", "10", "--max-size", "1"}} {
		if _, _, err := cli(t, args...); err == nil {
			t.Fatalf("accepted invalid request: %v", args)
		}
	}
}

func TestDedupeCLIExcludesAndCSV(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "a", "same")
	writeFixture(t, dir, "b", "same")
	out, stderr, err := cli(t, "dedupe", dir, "--exclude", "b", "--output", "json")
	if err != nil {
		t.Fatal(err, stderr)
	}
	var r dedupeReport
	if err := json.Unmarshal([]byte(out), &r); err != nil || len(r.Groups) != 0 || r.Hashed != 0 {
		t.Fatalf("exclude failed: %s %v", out, err)
	}
	out, stderr, err = cli(t, "dedupe", dir, "--output", "csv")
	if err != nil {
		t.Fatal(err, stderr)
	}
	rows, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil || len(rows) != 2 || rows[1][4] != "duplicate" {
		t.Fatalf("invalid CSV: %s %v", out, err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestOutputFailurePropagates(t *testing.T) {
	if err := writeHashRecords(failingWriter{}, []hashRecord{{Path: "a"}}, "json"); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	if err := writeDedupeReport(failingWriter{}, dedupeReport{}, "json"); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}

func TestDedupeScanFailureBlocksDeletion(t *testing.T) {
	out, stderr, err := cli(t, "dedupe", filepath.Join(t.TempDir(), "missing"), "--delete", "--apply", "--output", "json")
	if err == nil {
		t.Fatal("expected nonzero exit", stderr)
	}
	var report dedupeReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatal(err, out)
	}
	if report.Mode != "blocked" || len(report.Errors) == 0 {
		t.Fatalf("not blocked: %+v", report)
	}
}
