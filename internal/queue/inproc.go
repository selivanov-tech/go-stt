package queue

import (
	"context"
	"sync"
)

// InProc is a bounded in-process worker pool. No external dependencies — good
// for running locally. Swap for an asynq/Redis impl when deploying multi-node.
type InProc struct {
	tasks  chan Task
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	mu     sync.Mutex
	closed bool
}

func NewInProc(workers, buffer int) *InProc {
	if workers < 1 {
		workers = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	q := &InProc{
		tasks:  make(chan Task, buffer),
		ctx:    ctx,
		cancel: cancel,
	}
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker()
	}
	return q
}

func (q *InProc) worker() {
	defer q.wg.Done()
	for t := range q.tasks {
		t(q.ctx)
	}
}

func (q *InProc) Submit(t Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return context.Canceled
	}
	select {
	case q.tasks <- t:
		return nil
	case <-q.ctx.Done():
		return context.Canceled
	}
}

func (q *InProc) Shutdown(ctx context.Context) error {
	q.mu.Lock()
	if !q.closed {
		q.closed = true
		close(q.tasks)
	}
	q.mu.Unlock()

	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		q.cancel()
		return nil
	case <-ctx.Done():
		q.cancel()
		return ctx.Err()
	}
}
