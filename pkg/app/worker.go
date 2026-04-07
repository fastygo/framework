package app

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type BackgroundTask struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context)
}

type WorkerService struct {
	mu    sync.Mutex
	tasks []BackgroundTask
}

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

func (w *WorkerService) Start(ctx context.Context) {
	w.mu.Lock()
	tasks := append([]BackgroundTask(nil), w.tasks...)
	w.mu.Unlock()

	for _, task := range tasks {
		go func(task BackgroundTask) {
			ticker := time.NewTicker(task.Interval)
			defer ticker.Stop()

			task.Run(ctx)

			for {
				select {
				case <-ctx.Done():
					slog.Debug("worker stopped", "name", task.Name)
					return
				case <-ticker.C:
					task.Run(ctx)
				}
			}
		}(task)
	}
}
