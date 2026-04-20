package app

import (
	"context"
	"time"

	"github.com/fastygo/framework/pkg/cache"
)

// CleanupTask returns a BackgroundTask that periodically evicts expired
// entries from c. Register it via AppBuilder.AddBackgroundTask so the cache
// does not grow unbounded:
//
//	htmlCache := cache.New[[]byte](10 * time.Minute)
//	builder.AddBackgroundTask(app.CleanupTask("html-cache-cleanup", time.Minute, htmlCache))
//
// The helper lives in pkg/app (not pkg/cache) to keep pkg/cache free of any
// framework-level imports.
func CleanupTask[V any](name string, interval time.Duration, c *cache.Cache[V]) BackgroundTask {
	return BackgroundTask{
		Name:     name,
		Interval: interval,
		Run: func(ctx context.Context) {
			c.Cleanup()
		},
	}
}
