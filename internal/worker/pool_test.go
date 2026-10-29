package worker

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolRunsJobs(t *testing.T) {
	var n atomic.Int64
	p := New(4)
	p.Start()
	for i := 0; i < 20; i++ {
		p.Submit(func() { n.Add(1) })
	}
	p.Close()
	if got := n.Load(); got != 20 {
		t.Fatalf("want 20 jobs, got %d", got)
	}
}

func TestSubmitErrIsolatesFailures(t *testing.T) {
	var ok, fail atomic.Int64
	p := New(2)
	p.Start()
	for i := 0; i < 10; i++ {
		i := i
		p.SubmitErr(func() error {
			if i%2 == 0 {
				return errors.New("boom")
			}
			ok.Add(1)
			return nil
		}, func(error) { fail.Add(1) })
	}
	p.Close()
	if ok.Load() != 5 || fail.Load() != 5 {
		t.Fatalf("ok=%d fail=%d want 5/5", ok.Load(), fail.Load())
	}
}

func TestPoolBackPressure(t *testing.T) {
	p := New(1)
	p.Start()
	started := make(chan struct{})
	release := make(chan struct{})

	p.Submit(func() {
		close(started)
		<-release
	})
	<-started

	done := make(chan struct{})
	go func() {
		// Fill buffer (workers*2 == 2) then block on the next submit.
		p.Submit(func() {})
		p.Submit(func() {})
		p.Submit(func() {}) // should block until first job releases
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("submit should block while workers are busy and buffer is full")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for blocked submit")
	}
	p.Close()
}
