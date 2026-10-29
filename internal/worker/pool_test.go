package worker

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolRunsJobs(t *testing.T) {
	var n atomic.Int64
	p := New(4)
	p.Start()
	for i := 0; i < 20; i++ {
		if err := p.Submit(func() { n.Add(1) }); err != nil {
			t.Fatalf("Submit: %v", err)
		}
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
		if err := p.SubmitErr(func() error {
			if i%2 == 0 {
				return errors.New("boom")
			}
			ok.Add(1)
			return nil
		}, func(error) { fail.Add(1) }); err != nil {
			t.Fatalf("SubmitErr: %v", err)
		}
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

	if err := p.Submit(func() {
		close(started)
		<-release
	}); err != nil {
		t.Fatal(err)
	}
	<-started

	done := make(chan struct{})
	go func() {
		// Fill buffer (workers*2 == 2) then block on the next submit.
		_ = p.Submit(func() {})
		_ = p.Submit(func() {})
		_ = p.Submit(func() {}) // should block until first job releases
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

func TestSubmitAfterCloseReturnsErrClosed(t *testing.T) {
	p := New(2)
	p.Start()
	p.Close()

	if err := p.Submit(func() {}); !errors.Is(err, ErrClosed) {
		t.Fatalf("want ErrClosed, got %v", err)
	}
	if err := p.SubmitErr(func() error { return nil }, nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("want ErrClosed, got %v", err)
	}

	// Close must be idempotent.
	p.Close()
}

func TestSubmitCloseRaceNoPanic(t *testing.T) {
	p := New(4)
	p.Start()

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = p.Submit(func() {})
			}
		}()
	}

	time.Sleep(5 * time.Millisecond)
	p.Close()
	wg.Wait()
}
