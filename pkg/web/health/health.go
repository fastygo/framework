// Package health publishes liveness and readiness HTTP endpoints driven by
// per-feature health checks.
//
// Liveness (/healthz) always returns 200 as long as the process is
// answering — its purpose is to let an orchestrator (Kubernetes, systemd)
// detect a deadlocked binary and restart it.
//
// Readiness (/readyz) aggregates a set of Checkers in parallel, each with
// a per-check timeout. If any returns an error the response is HTTP 503
// with a JSON body listing per-check status. Use it to gate traffic on
// dependencies (database, OIDC issuer, message broker).
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status is the result of a single Checker.Check call.
type Status string

// Stable status strings shared by Result and the readiness payload.
const (
	// StatusUp means the dependency answered without error within the timeout.
	StatusUp Status = "up"
	// StatusDown means the dependency returned an error or exceeded the timeout.
	StatusDown Status = "down"
)

// Result captures one check outcome for the JSON readiness payload.
type Result struct {
	// Name identifies the checker (matches Checker.Name()).
	Name string `json:"name"`
	// Status is StatusUp or StatusDown depending on the check outcome.
	Status Status `json:"status"`
	// Took is the wall time spent inside Check, in nanoseconds.
	Took time.Duration `json:"took_ns"`
	// Error is the rendered error message when Status is StatusDown,
	// empty otherwise.
	Error string `json:"error,omitempty"`
}

// Checker probes a single dependency. Implementations must be safe for
// concurrent use and must respect ctx cancellation.
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

// CheckerFunc adapts a function to the Checker interface.
type CheckerFunc struct {
	// CheckerName is returned by the Name() method.
	CheckerName string
	// Fn is the probe function invoked by Check.
	Fn func(ctx context.Context) error
}

// Name returns the checker's identifier.
func (c CheckerFunc) Name() string { return c.CheckerName }

// Check delegates to the underlying function.
func (c CheckerFunc) Check(ctx context.Context) error { return c.Fn(ctx) }

// DefaultTimeout is applied per Checker when Aggregator.Timeout is zero.
const DefaultTimeout = 5 * time.Second

// Aggregator runs every registered Checker in parallel with a per-check
// context derived from the readiness handler's request context.
type Aggregator struct {
	mu       sync.RWMutex
	checkers []Checker
	// Timeout caps each individual Checker.Check invocation. Zero means
	// DefaultTimeout. The aggregate handler still respects the inbound
	// request context as the absolute upper bound.
	Timeout time.Duration
}

// NewAggregator returns an empty Aggregator with the default per-check
// timeout.
func NewAggregator() *Aggregator {
	return &Aggregator{Timeout: DefaultTimeout}
}

// Add registers a Checker. Safe to call from multiple goroutines.
func (a *Aggregator) Add(c Checker) {
	if c == nil {
		return
	}
	a.mu.Lock()
	a.checkers = append(a.checkers, c)
	a.mu.Unlock()
}

// Run executes every Checker in parallel and returns all results in
// registration order. A check that exceeds the per-check timeout is
// reported as down with context.DeadlineExceeded.
func (a *Aggregator) Run(ctx context.Context) []Result {
	a.mu.RLock()
	snapshot := append([]Checker(nil), a.checkers...)
	timeout := a.Timeout
	a.mu.RUnlock()

	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	results := make([]Result, len(snapshot))
	var wg sync.WaitGroup
	for i, c := range snapshot {
		wg.Add(1)
		go func(i int, c Checker) {
			defer wg.Done()
			results[i] = runOne(ctx, c, timeout)
		}(i, c)
	}
	wg.Wait()
	return results
}

func runOne(parent context.Context, c Checker, timeout time.Duration) Result {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	start := time.Now()
	err := c.Check(ctx)
	took := time.Since(start)

	res := Result{Name: c.Name(), Status: StatusUp, Took: took}
	if err != nil {
		res.Status = StatusDown
		res.Error = err.Error()
	}
	return res
}

// LiveHandler always returns 200 OK with a tiny JSON body. It does not
// run any checks — its only purpose is to prove that the process can
// accept and answer HTTP traffic.
func LiveHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"up"}`))
	})
}

// ReadyHandler runs every Checker registered in a and returns 200 if all
// are up, 503 otherwise. The response body is a JSON object:
//
//	{"status":"up","checks":[{"name":"db","status":"up","took_ns":12345}]}
//
// A nil aggregator is treated as "no dependencies" and always returns up.
func ReadyHandler(a *Aggregator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var results []Result
		if a != nil {
			results = a.Run(r.Context())
		}

		overall := StatusUp
		for _, res := range results {
			if res.Status != StatusUp {
				overall = StatusDown
				break
			}
		}

		body := struct {
			Status Status   `json:"status"`
			Checks []Result `json:"checks,omitempty"`
		}{Status: overall, Checks: results}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if overall == StatusDown {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		_ = json.NewEncoder(w).Encode(body)
	})
}
