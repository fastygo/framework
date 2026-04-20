package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func okChecker(name string) Checker {
	return CheckerFunc{CheckerName: name, Fn: func(context.Context) error { return nil }}
}

func errChecker(name string, err error) Checker {
	return CheckerFunc{CheckerName: name, Fn: func(context.Context) error { return err }}
}

func slowChecker(name string, sleep time.Duration) Checker {
	return CheckerFunc{
		CheckerName: name,
		Fn: func(ctx context.Context) error {
			select {
			case <-time.After(sleep):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
}

func TestAggregator_AllUp(t *testing.T) {
	t.Parallel()

	agg := NewAggregator()
	agg.Add(okChecker("a"))
	agg.Add(okChecker("b"))
	agg.Add(okChecker("c"))

	results := agg.Run(context.Background())
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	for _, r := range results {
		if r.Status != StatusUp {
			t.Errorf("checker %q status = %s, want up (err=%q)", r.Name, r.Status, r.Error)
		}
	}
}

func TestAggregator_OneDown(t *testing.T) {
	t.Parallel()

	agg := NewAggregator()
	agg.Add(okChecker("a"))
	agg.Add(errChecker("b", errors.New("db unreachable")))
	agg.Add(okChecker("c"))

	rec := httptest.NewRecorder()
	ReadyHandler(agg).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var body struct {
		Status Status   `json:"status"`
		Checks []Result `json:"checks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != StatusDown {
		t.Errorf("overall status = %s, want down", body.Status)
	}
	if !strings.Contains(rec.Body.String(), "db unreachable") {
		t.Errorf("expected error message in body, got %q", rec.Body.String())
	}
}

func TestAggregator_RespectsTimeout(t *testing.T) {
	t.Parallel()

	agg := NewAggregator()
	agg.Timeout = 50 * time.Millisecond
	agg.Add(slowChecker("slow", 500*time.Millisecond))

	start := time.Now()
	results := agg.Run(context.Background())
	elapsed := time.Since(start)

	if elapsed > 200*time.Millisecond {
		t.Fatalf("aggregator exceeded per-check timeout: %v", elapsed)
	}
	if len(results) != 1 || results[0].Status != StatusDown {
		t.Fatalf("unexpected results: %+v", results)
	}
	if !strings.Contains(results[0].Error, "deadline") {
		t.Errorf("expected deadline error, got %q", results[0].Error)
	}
}

func TestAggregator_ParallelExecution(t *testing.T) {
	t.Parallel()

	agg := NewAggregator()
	agg.Add(slowChecker("a", 100*time.Millisecond))
	agg.Add(slowChecker("b", 100*time.Millisecond))
	agg.Add(slowChecker("c", 100*time.Millisecond))

	start := time.Now()
	results := agg.Run(context.Background())
	elapsed := time.Since(start)

	if elapsed > 200*time.Millisecond {
		t.Fatalf("checks ran serially: %v (want < 200ms)", elapsed)
	}
	for _, r := range results {
		if r.Status != StatusUp {
			t.Errorf("checker %q failed: %s", r.Name, r.Error)
		}
	}
}

func TestLiveHandler_AlwaysOK(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	LiveHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"up"`) {
		t.Errorf("body = %q, want status up", rec.Body.String())
	}
}

func TestReadyHandler_NilAggregator(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	ReadyHandler(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestAggregator_ConcurrentAdd(t *testing.T) {
	t.Parallel()

	agg := NewAggregator()
	var added int32
	const total = 50
	done := make(chan struct{})
	for i := 0; i < total; i++ {
		go func() {
			agg.Add(okChecker("c"))
			atomic.AddInt32(&added, 1)
			if atomic.LoadInt32(&added) == total {
				close(done)
			}
		}()
	}
	<-done

	results := agg.Run(context.Background())
	if len(results) != total {
		t.Fatalf("len(results) = %d, want %d", len(results), total)
	}
}
