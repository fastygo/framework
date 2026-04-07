package app

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerServiceLifecycle(t *testing.T) {
	t.Parallel()

	var runs int32
	ws := &WorkerService{}
	ws.Add(BackgroundTask{
		Name:     "test-task",
		Interval: 20 * time.Millisecond,
		Run: func(ctx context.Context) {
			atomic.AddInt32(&runs, 1)
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	ws.Start(ctx)

	time.Sleep(70 * time.Millisecond)
	cancel()

	initialRuns := atomic.LoadInt32(&runs)
	if initialRuns == 0 {
		t.Fatal("expected worker task to run at least once")
	}

	time.Sleep(60 * time.Millisecond)
	finalRuns := atomic.LoadInt32(&runs)
	if finalRuns != initialRuns {
		t.Fatalf("expected runs to stop after cancel, got start=%d end=%d", initialRuns, finalRuns)
	}
}
