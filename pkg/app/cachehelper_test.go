package app

import (
	"context"
	"testing"
	"time"

	"github.com/fastygo/framework/pkg/cache"
)

// TestCleanupTask_RunsOnInterval wires CleanupTask into WorkerService and
// asserts that expired entries are eventually evicted without any explicit
// Cleanup call from user code.
func TestCleanupTask_RunsOnInterval(t *testing.T) {
	t.Parallel()

	c := cache.New[int](5 * time.Millisecond)
	for i := 0; i < 100; i++ {
		c.Set(string(rune('a'+(i%26))), i)
	}

	ws := &WorkerService{}
	ws.Add(CleanupTask("test-cleanup", 10*time.Millisecond, c))

	ctx, cancel := context.WithCancel(context.Background())
	ws.Start(ctx)

	time.Sleep(80 * time.Millisecond)
	cancel()

	stopCtx, cancelStop := context.WithTimeout(context.Background(), time.Second)
	defer cancelStop()
	if err := ws.Stop(stopCtx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	for i := 0; i < 26; i++ {
		key := string(rune('a' + i))
		if _, ok := c.Get(key); ok {
			t.Fatalf("expected key %q to be evicted by background cleanup", key)
		}
	}
}

// TestCleanupTask_DefaultsApplied checks that CleanupTask returns a
// usable BackgroundTask with the expected name and interval.
func TestCleanupTask_DefaultsApplied(t *testing.T) {
	t.Parallel()

	c := cache.New[int](time.Minute)
	task := CleanupTask("my-cache", 30*time.Second, c)

	if task.Name != "my-cache" {
		t.Fatalf("Name = %q, want %q", task.Name, "my-cache")
	}
	if task.Interval != 30*time.Second {
		t.Fatalf("Interval = %v, want %v", task.Interval, 30*time.Second)
	}
	if task.Run == nil {
		t.Fatal("Run must not be nil")
	}
}
