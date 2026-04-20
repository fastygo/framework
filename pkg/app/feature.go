package app

import (
	"context"
	"net/http"
)

// NavItem represents a navigation entry contributed by a Feature.
type NavItem struct {
	// Label is the human-readable text displayed in navigation.
	Label string
	// Path is the URL path the entry links to (e.g. "/blog").
	Path string
	// Icon is an optional icon identifier interpreted by the UI layer
	// (UI8Kit consumers use lucide-style names; empty means no icon).
	Icon string
	// Order controls display position; lower values render first.
	// Ties are broken by registration order.
	Order int
}

// Feature is the minimum contract every pluggable module must satisfy.
//
// Features are pure compile-time plugins: the application's composition root
// constructs them, hands them to the AppBuilder, and the builder wires their
// routes and navigation entries into the shared mux.
type Feature interface {
	ID() string
	Routes(mux *http.ServeMux)
	NavItems() []NavItem
}

// NavProvider lets a feature receive the merged navigation list collected
// from every other feature in the application. Implement it when a feature
// needs to render the global navigation (for example, in a layout shell).
type NavProvider interface {
	SetNavItems([]NavItem)
}

// Initializer is an optional hook called once before the HTTP server starts.
// Use it to warm caches, validate configuration, or pre-load templates.
type Initializer interface {
	Init(ctx context.Context) error
}

// Closer is an optional hook called when the application shuts down.
// Use it to close database pools, flush logs, or release file handles.
type Closer interface {
	Close(ctx context.Context) error
}

// HealthChecker exposes a feature-specific health probe. The framework can
// expose these checks through `/healthz` style endpoints when wired up by
// the application.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// BackgroundProvider returns long-running tasks that should be supervised by
// the framework's WorkerService for the lifetime of the application.
type BackgroundProvider interface {
	BackgroundTasks() []BackgroundTask
}
