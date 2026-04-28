package app

import (
	"context"
	"errors"
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

	stopCtx, cancelStop := context.WithTimeout(context.Background(), time.Second)
	defer cancelStop()
	if err := ws.Stop(stopCtx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

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

func TestWorkerServiceStartIsIdempotent(t *testing.T) {
	t.Parallel()

	var runs int32
	release := make(chan struct{})
	ws := &WorkerService{}
	ws.Add(BackgroundTask{
		Name:     "single-start",
		Interval: time.Hour,
		Run: func(ctx context.Context) {
			atomic.AddInt32(&runs, 1)
			<-release
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	ws.Start(ctx)
	for atomic.LoadInt32(&runs) == 0 {
		time.Sleep(time.Millisecond)
	}

	ws.Start(ctx)
	time.Sleep(20 * time.Millisecond)
	if got := atomic.LoadInt32(&runs); got != 1 {
		t.Fatalf("Start must not duplicate worker goroutines, got %d runs", got)
	}

	cancel()
	close(release)
	stopCtx, cancelStop := context.WithTimeout(context.Background(), time.Second)
	defer cancelStop()
	if err := ws.Stop(stopCtx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

func TestWorkerServiceAddAfterStartIsIgnored(t *testing.T) {
	t.Parallel()

	var runs int32
	ws := &WorkerService{}
	ctx, cancel := context.WithCancel(context.Background())
	ws.Start(ctx)

	ws.Add(BackgroundTask{
		Name:     "too-late",
		Interval: time.Millisecond,
		Run: func(ctx context.Context) {
			atomic.AddInt32(&runs, 1)
		},
	})
	time.Sleep(20 * time.Millisecond)
	cancel()

	if err := ws.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	if got := atomic.LoadInt32(&runs); got != 0 {
		t.Fatalf("Add after Start must be ignored, got %d runs", got)
	}
}

// TestWorker_StopWaitsForRunningTask ensures Stop blocks until an in-flight
// task has actually returned. Without WaitGroup tracking the goroutine
// would be orphaned and Stop would return immediately.
func TestWorker_StopWaitsForRunningTask(t *testing.T) {
	t.Parallel()

	released := make(chan struct{})
	finished := make(chan struct{})

	ws := &WorkerService{}
	ws.Add(BackgroundTask{
		Name:     "slow-task",
		Interval: time.Hour,
		Run: func(ctx context.Context) {
			<-released
			close(finished)
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	ws.Start(ctx)

	time.Sleep(20 * time.Millisecond)
	cancel()

	stopReturned := make(chan error, 1)
	go func() {
		stopReturned <- ws.Stop(context.Background())
	}()

	select {
	case <-stopReturned:
		t.Fatal("Stop returned before task finished")
	case <-time.After(40 * time.Millisecond):
	}

	close(released)

	select {
	case err := <-stopReturned:
		if err != nil {
			t.Fatalf("Stop returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Stop did not return after task released")
	}

	select {
	case <-finished:
	default:
		t.Fatal("task did not finish before Stop returned")
	}
}

// TestWorker_PanicDoesNotKillService verifies that a panic inside Run is
// recovered and the ticker keeps firing, so a buggy task cannot take down
// the entire process.
func TestWorker_PanicDoesNotKillService(t *testing.T) {
	t.Parallel()

	var calls int32
	ws := &WorkerService{}
	ws.Add(BackgroundTask{
		Name:     "panicky",
		Interval: 10 * time.Millisecond,
		Run: func(ctx context.Context) {
			n := atomic.AddInt32(&calls, 1)
			if n == 1 {
				panic("boom")
			}
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	ws.Start(ctx)

	time.Sleep(80 * time.Millisecond)
	cancel()

	stopCtx, cancelStop := context.WithTimeout(context.Background(), time.Second)
	defer cancelStop()
	if err := ws.Stop(stopCtx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	if got := atomic.LoadInt32(&calls); got < 2 {
		t.Fatalf("expected at least 2 invocations after panic, got %d", got)
	}
}

// TestWorker_StopHonoursContextDeadline ensures Stop returns the ctx error
// when a task ignores cancellation and exceeds the shutdown budget.
func TestWorker_StopHonoursContextDeadline(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	defer close(release)

	ws := &WorkerService{}
	ws.Add(BackgroundTask{
		Name:     "stubborn",
		Interval: time.Hour,
		Run: func(ctx context.Context) {
			<-release // ignores ctx on purpose
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	ws.Start(ctx)

	time.Sleep(20 * time.Millisecond)
	cancel()

	stopCtx, cancelStop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancelStop()

	err := ws.Stop(stopCtx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestWorker_StopCanWaitAgainAfterDeadline(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	ws := &WorkerService{}
	ws.Add(BackgroundTask{
		Name:     "retry-stop",
		Interval: time.Hour,
		Run: func(ctx context.Context) {
			<-release
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	ws.Start(ctx)
	time.Sleep(20 * time.Millisecond)
	cancel()

	deadlineCtx, cancelDeadline := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancelDeadline()
	if err := ws.Stop(deadlineCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}

	close(release)
	stopCtx, cancelStop := context.WithTimeout(context.Background(), time.Second)
	defer cancelStop()
	if err := ws.Stop(stopCtx); err != nil {
		t.Fatalf("second Stop should wait for drained workers, got %v", err)
	}
}

// TestWorker_StopBeforeStartIsNoop checks the Stop-without-Start path.
func TestWorker_StopBeforeStartIsNoop(t *testing.T) {
	t.Parallel()

	ws := &WorkerService{}
	if err := ws.Stop(context.Background()); err != nil {
		t.Fatalf("Stop on empty service returned %v, want nil", err)
	}
}
