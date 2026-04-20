package app

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

// BackgroundTask describes a periodic unit of work supervised by
// WorkerService. Run is invoked once on startup and then on every Interval
// tick until the supervising context is cancelled.
type BackgroundTask struct {
	// Name identifies the task in logs and panic reports. Required.
	Name string
	// Interval is the delay between successive Run invocations. The
	// first Run fires immediately on startup; the next one fires
	// Interval after Run returns (not after the previous tick), so a
	// long Run never overlaps with itself.
	Interval time.Duration
	// Run executes one tick of work. It must respect ctx cancellation:
	// when WorkerService stops, ctx is cancelled and a misbehaving Run
	// will block graceful shutdown up to HTTPShutdownTimeout.
	Run func(ctx context.Context)
}

// WorkerService supervises a fixed set of BackgroundTasks for the lifetime
// of an App. It guarantees three properties that the framework relies on
// for safe shutdown:
//
//   - every running task is tracked via an internal sync.WaitGroup;
//   - a panic inside Run is recovered, logged with a stack trace, and the
//     ticker keeps firing — one bad task cannot crash the process;
//   - Stop blocks until every task goroutine has returned (or the caller's
//     context expires), so App.Run can drain workers before exiting.
type WorkerService struct {
	mu       sync.Mutex
	tasks    []BackgroundTask
	wg       sync.WaitGroup
	stopOnce sync.Once
}

// Add registers a task. Safe to call before Start; calls after Start are a
// no-op (the supervisor only iterates tasks captured at Start time).
func (w *WorkerService) Add(task BackgroundTask) {
	if task.Interval <= 0 || task.Run == nil {
		return
	}

	if task.Name == "" {
		task.Name = "worker"
	}

	w.mu.Lock()
	w.tasks = append(w.tasks, task)
	w.mu.Unlock()
}

// Start launches one supervised goroutine per registered task. The
// goroutines exit when ctx is cancelled. Use Stop to wait for them to
// finish draining.
func (w *WorkerService) Start(ctx context.Context) {
	w.mu.Lock()
	tasks := append([]BackgroundTask(nil), w.tasks...)
	w.mu.Unlock()

	for _, task := range tasks {
		w.wg.Add(1)
		go func(task BackgroundTask) {
			defer w.wg.Done()

			ticker := time.NewTicker(task.Interval)
			defer ticker.Stop()

			safeRun(ctx, task)

			for {
				select {
				case <-ctx.Done():
					slog.Debug("worker stopped", "name", task.Name)
					return
				case <-ticker.C:
					safeRun(ctx, task)
				}
			}
		}(task)
	}
}

// Stop blocks until every task goroutine returns or until ctx is done.
// It does not cancel the tasks itself — callers are expected to cancel the
// context they passed to Start (typically by triggering app shutdown)
// before invoking Stop. Stop is safe to call multiple times and a no-op if
// Start was never called.
func (w *WorkerService) Stop(ctx context.Context) error {
	var err error
	w.stopOnce.Do(func() {
		done := make(chan struct{})
		go func() {
			w.wg.Wait()
			close(done)
		}()

		if ctx == nil {
			<-done
			return
		}

		select {
		case <-done:
		case <-ctx.Done():
			err = ctx.Err()
		}
	})
	return err
}

// safeRun executes task.Run with a recover guard so that a single panicking
// task cannot crash the process. The panic is logged with the task name and
// a full stack trace.
func safeRun(ctx context.Context, task BackgroundTask) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("worker panic",
				"name", task.Name,
				"err", r,
				"stack", string(debug.Stack()),
			)
		}
	}()
	task.Run(ctx)
}
