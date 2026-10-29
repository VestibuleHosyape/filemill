package worker

import (
	"log/slog"
	"sync"
)

// Pool is a bounded worker pool backed by a buffered job channel.
type Pool struct {
	workers int
	jobs    chan func()
	wg      sync.WaitGroup
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
func (p *Pool) Submit(job func()) {
	p.jobs <- job
}

// SubmitErr runs fn in a worker. On error it logs and calls onErr when non-nil,
// then continues — single-job failures never stop the pool (error isolation).
func (p *Pool) SubmitErr(fn func() error, onErr func(error)) {
	p.Submit(func() {
		if err := fn(); err != nil {
			slog.Error("job failed", "err", err)
			if onErr != nil {
				onErr(err)
			}
		}
	})
}

// Close closes the job channel and waits for all workers to finish.
func (p *Pool) Close() {
	close(p.jobs)
	p.wg.Wait()
}

// Workers returns the configured worker count.
func (p *Pool) Workers() int { return p.workers }
