package worker

import (
	"errors"
	"log/slog"
	"sync"
)

// ErrClosed is returned when Submit is called after Close.
var ErrClosed = errors.New("worker pool closed")

// Pool is a bounded worker pool backed by a buffered job channel.
type Pool struct {
	workers int
	jobs    chan func()
	wg      sync.WaitGroup

	mu       sync.Mutex
	closed   bool
	inflight sync.WaitGroup // tracks Submit calls that passed the closed check
}

// New creates a pool with the given worker count.
// workers is clamped to at least 1. The job buffer size is workers*2.
func New(workers int) *Pool {
	if workers < 1 {
		workers = 1
	}
	return &Pool{
		workers: workers,
		jobs:    make(chan func(), workers*2),
	}
}

// Start launches worker goroutines. Call Close when producers are done.
func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for job := range p.jobs {
				job()
			}
		}()
	}
}

// Submit enqueues a job. It blocks if the buffer is full (back-pressure).
// After Close, Submit returns ErrClosed instead of sending on a closed channel.
func (p *Pool) Submit(job func()) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrClosed
	}
	p.inflight.Add(1)
	p.mu.Unlock()

	defer p.inflight.Done()
	p.jobs <- job
	return nil
}

// SubmitErr runs fn in a worker. On error it logs and calls onErr when non-nil,
// then continues — single-job failures never stop the pool (error isolation).
// Returns ErrClosed if the pool has already been closed.
func (p *Pool) SubmitErr(fn func() error, onErr func(error)) error {
	return p.Submit(func() {
		if err := fn(); err != nil {
			slog.Error("job failed", "err", err)
			if onErr != nil {
				onErr(err)
			}
		}
	})
}

// Close rejects new Submit calls, waits for in-flight Submit sends to finish,
// closes the job channel, then waits for workers to drain.
// Close is safe to call multiple times.
func (p *Pool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		p.wg.Wait()
		return
	}
	p.closed = true
	p.mu.Unlock()

	p.inflight.Wait()
	close(p.jobs)
	p.wg.Wait()
}

// Workers returns the configured worker count.
func (p *Pool) Workers() int { return p.workers }
