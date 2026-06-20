// Package queue is a job queue behind an interface, so the in-process pool can
// be swapped for a Redis-backed queue (asynq) later without changing callers.
package queue

import "context"

type Task func(ctx context.Context)

type Queue interface {
	Submit(t Task) error
	Shutdown(ctx context.Context) error
}
