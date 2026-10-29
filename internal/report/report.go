package report

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// Stats tracks batch job progress. Methods are safe for concurrent use.
type Stats struct {
	Total   atomic.Int64
	Success atomic.Int64
	Failed  atomic.Int64
	started time.Time
}

// Start begins timing. Call once before work starts.
func (s *Stats) Start() {
	s.started = time.Now()
}

// Elapsed returns time since Start. If Start was not called, returns 0.
func (s *Stats) Elapsed() time.Duration {
	if s.started.IsZero() {
		return 0
	}
	return time.Since(s.started)
}

// AddTotal increments the planned/seen total.
func (s *Stats) AddTotal(n int64) { s.Total.Add(n) }

// IncSuccess marks one successful item.
func (s *Stats) IncSuccess() { s.Success.Add(1) }

// IncFailed marks one failed item.
func (s *Stats) IncFailed() { s.Failed.Add(1) }

// Summary formats total / success / failed / elapsed.
func (s *Stats) Summary() string {
	return fmt.Sprintf(
		"total=%d success=%d failed=%d elapsed=%s",
		s.Total.Load(),
		s.Success.Load(),
		s.Failed.Load(),
		s.Elapsed().Round(time.Millisecond),
	)
}

// WriteSummary writes Summary to w with a trailing newline.
func (s *Stats) WriteSummary(w io.Writer) error {
	_, err := fmt.Fprintln(w, s.Summary())
	return err
}
