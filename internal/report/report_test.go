package report

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestStatsSummary(t *testing.T) {
	s := &Stats{}
	s.Start()
	s.AddTotal(3)
	s.IncSuccess()
	s.IncSuccess()
	s.IncFailed()
	time.Sleep(2 * time.Millisecond)

	sum := s.Summary()
	if !strings.Contains(sum, "total=3") || !strings.Contains(sum, "success=2") || !strings.Contains(sum, "failed=1") {
		t.Fatalf("unexpected summary: %s", sum)
	}
	if s.Elapsed() <= 0 {
		t.Fatal("expected positive elapsed")
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestSummaryAndFailureStatus(t *testing.T) {
	s := &Stats{}
	if s.Err() != nil || s.Elapsed() != 0 {
		t.Fatal("empty stats should not fail")
	}
	s.IncFailed()
	if s.Err() == nil {
		t.Fatal("partial failure must be reported")
	}
	var out strings.Builder
	if err := s.WriteSummary(&out); err != nil || !strings.Contains(out.String(), "failed=1") {
		t.Fatal(out.String(), err)
	}
	if err := s.WriteSummary(brokenWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("lost output error", err)
	}
}
