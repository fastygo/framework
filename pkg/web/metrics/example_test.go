package metrics_test

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/fastygo/framework/pkg/web/metrics"
)

// ExampleRegistry walks the typical observation path: register a
// counter, record a few hits, then serialise the registry to the
// Prometheus text format that /metrics returns.
func ExampleRegistry() {
	reg := metrics.NewRegistry()
	hits := reg.Counter("page_hits_total", "Total page hits.", "route")

	hits.Inc("/")
	hits.Inc("/")
	hits.Inc("/about")

	var buf bytes.Buffer
	_ = reg.Write(&buf)

	// Print a stable subset (the metric body) so the example is
	// deterministic across Go versions and OS line endings.
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasPrefix(line, "page_hits_total") {
			fmt.Println(line)
		}
	}

	// Output:
	// page_hits_total{route="/"} 2
	// page_hits_total{route="/about"} 1
}

// ExampleRegistry_histogram shows histogram observations and the
// derived _bucket / _count / _sum series that Prometheus and Grafana
// rely on for percentile queries.
func ExampleRegistry_histogram() {
	reg := metrics.NewRegistry()
	latency := reg.Histogram(
		"op_seconds",
		"Operation duration.",
		[]float64{0.1, 0.5, 1, 5},
		"op",
	)

	latency.Observe(0.05, "fast")
	latency.Observe(0.2, "fast")
	latency.Observe(2.0, "fast")

	var buf bytes.Buffer
	_ = reg.Write(&buf)

	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasPrefix(line, "op_seconds") {
			fmt.Println(line)
		}
	}

	// Output:
	// op_seconds_bucket{le="0.1",op="fast"} 1
	// op_seconds_bucket{le="0.5",op="fast"} 2
	// op_seconds_bucket{le="1",op="fast"} 2
	// op_seconds_bucket{le="5",op="fast"} 3
	// op_seconds_bucket{le="+Inf",op="fast"} 3
	// op_seconds_sum{op="fast"} 2.25
	// op_seconds_count{op="fast"} 3
}
