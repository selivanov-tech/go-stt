package queue

import (
	"context"
	"sync"
	"testing"
)

func TestInProcRunsAllTasks(t *testing.T) {
	q := NewInProc(3, 16)

	var mu sync.Mutex
	count := 0
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		err := q.Submit(func(ctx context.Context) {
			mu.Lock()
			count++
			mu.Unlock()
			wg.Done()
		})
		if err != nil {
			t.Fatalf("submit %d: %v", i, err)
		}
	}

	wg.Wait()
	if count != 10 {
		t.Fatalf("want 10 tasks run, got %d", count)
	}

	if err := q.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := q.Submit(func(context.Context) {}); err == nil {
		t.Fatal("submit after shutdown should fail")
	}
}
