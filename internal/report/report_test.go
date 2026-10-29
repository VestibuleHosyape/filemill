package report

import (
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
