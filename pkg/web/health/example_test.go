package health_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/fastygo/framework/pkg/web/health"
)

// ExampleAggregator wires two checks into the readiness handler and
// shows the JSON envelope returned for the all-up case. Use a
// CheckerFunc when the probe is a single function call (database
// ping, HTTP GET). Implement health.Checker on a struct when the
// probe needs configuration or per-check state.
func ExampleAggregator() {
	agg := health.NewAggregator()

	agg.Add(health.CheckerFunc{
		CheckerName: "database",
		Fn: func(ctx context.Context) error {
			// Replace with a real db.PingContext(ctx) call.
			return nil
		},
	})
	agg.Add(health.CheckerFunc{
		CheckerName: "issuer",
		Fn: func(ctx context.Context) error {
			// Replace with a real OIDC discovery probe.
			return nil
		},
	})

	rec := httptest.NewRecorder()
	health.ReadyHandler(agg).ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodGet, "/readyz", nil),
	)

	// We assert structural facts (status code + overall status string)
	// rather than the raw body, because per-check Took values are
	// non-deterministic.
	body, _ := io.ReadAll(rec.Body)
	fmt.Println("status:", rec.Code)
	fmt.Println("contains overall up:", contains(body, []byte(`"status":"up"`)))
	fmt.Println("contains database:", contains(body, []byte(`"name":"database"`)))
	fmt.Println("contains issuer:", contains(body, []byte(`"name":"issuer"`)))

	// Output:
	// status: 200
	// contains overall up: true
	// contains database: true
	// contains issuer: true
}

// ExampleAggregator_oneDown demonstrates the failure path: a single
// down check flips the overall status to "down" and the HTTP code to
// 503 Service Unavailable, telling the orchestrator to stop routing
// traffic to this instance.
func ExampleAggregator_oneDown() {
	agg := health.NewAggregator()
	agg.Add(health.CheckerFunc{
		CheckerName: "broken",
		Fn:          func(ctx context.Context) error { return errors.New("connection refused") },
	})

	rec := httptest.NewRecorder()
	health.ReadyHandler(agg).ServeHTTP(
		rec,
		httptest.NewRequest(http.MethodGet, "/readyz", nil),
	)

	body, _ := io.ReadAll(rec.Body)
	fmt.Println("status:", rec.Code)
	fmt.Println("overall down:", contains(body, []byte(`"status":"down"`)))

	// Output:
	// status: 503
	// overall down: true
}

func contains(haystack, needle []byte) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
