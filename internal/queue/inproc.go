package queue

import (
	"context"
	"errors"
	"sync"
)

// ErrClosed is returned by Submit after the queue has shut down.
var ErrClosed = errors.New("queue is shut down")

// InProc is a bounded in-process worker pool. The tasks channel is never closed;
// shutdown is signalled by cancelling ctx. That way no Submit pins a lock across
// a blocking send, so Shutdown's deadline always wins.
type InProc struct {
	tasks  chan Task
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
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
	for {
		select {
		case t := <-q.tasks:
			t(q.ctx)
		case <-q.ctx.Done():
			return
		}
	}
}

func (q *InProc) Submit(t Task) error {
	if q.ctx.Err() != nil {
		return ErrClosed
	}
	select {
	case q.tasks <- t:
		return nil
	case <-q.ctx.Done():
		return ErrClosed
	}
}

func (q *InProc) Shutdown(ctx context.Context) error {
	q.cancel()
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
