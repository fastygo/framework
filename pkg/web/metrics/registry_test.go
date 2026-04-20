package metrics

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCounter_TextFormat(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	c := r.Counter("requests_total", "Total requests.", "method", "status")
	c.Inc("GET", "200")
	c.Add(3, "GET", "200")
	c.Inc("POST", "500")

	var buf bytes.Buffer
	if err := r.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"# HELP requests_total Total requests.",
		"# TYPE requests_total counter",
		`requests_total{method="GET",status="200"} 4`,
		`requests_total{method="POST",status="500"} 1`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nGot:\n%s", want, out)
		}
	}
}

func TestCounter_NoLabels(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	c := r.Counter("connections_total", "Connections established.")
	c.Add(7)

	var buf bytes.Buffer
	_ = r.Write(&buf)
	if !strings.Contains(buf.String(), "connections_total 7\n") {
		t.Fatalf("expected unlabelled output, got:\n%s", buf.String())
	}
}

func TestGauge_RoundTrip(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	g := r.Gauge("queue_depth", "Items in queue.", "queue")
	g.Set(12.5, "default")
	g.Add(2.5, "default")
	g.Dec("default")

	var buf bytes.Buffer
	_ = r.Write(&buf)
	if !strings.Contains(buf.String(), `queue_depth{queue="default"} 14`) {
		t.Fatalf("unexpected gauge output:\n%s", buf.String())
	}
}

func TestHistogram_BucketCounts(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	h := r.Histogram("op_seconds", "Operation duration.",
		[]float64{0.1, 0.5, 1.0}, "op")
	h.Observe(0.05, "fast")  // -> all buckets
	h.Observe(0.3, "fast")   // -> 0.5, 1.0, +Inf
	h.Observe(2.0, "fast")   // -> +Inf only
	h.Observe(0.05, "slow")

	var buf bytes.Buffer
	_ = r.Write(&buf)
	out := buf.String()

	for _, want := range []string{
		`op_seconds_bucket{le="0.1",op="fast"} 1`,
		`op_seconds_bucket{le="0.5",op="fast"} 2`,
		`op_seconds_bucket{le="1",op="fast"} 2`,
		`op_seconds_bucket{le="+Inf",op="fast"} 3`,
		`op_seconds_count{op="fast"} 3`,
		`op_seconds_sum{op="fast"} 2.35`,
		`op_seconds_bucket{le="+Inf",op="slow"} 1`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n---\n%s", want, out)
		}
	}
}

func TestRegistry_ConcurrentSafe(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	c := r.Counter("hits_total", "Hits.")
	const goroutines = 50
	const perGoroutine = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				c.Inc()
			}
		}()
	}
	wg.Wait()

	var buf bytes.Buffer
	_ = r.Write(&buf)
	want := goroutines * perGoroutine
	if !strings.Contains(buf.String(), "hits_total ") {
		t.Fatalf("counter missing:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "10000") {
		t.Errorf("expected counter total %d in output:\n%s", want, buf.String())
	}
}

// slowWriter forces Write's I/O loop to block long enough that any
// lock held across the loop would visibly stall a concurrent Inc.
type slowWriter struct{ delay time.Duration }

func (s slowWriter) Write(p []byte) (int, error) {
	time.Sleep(s.delay)
	return len(p), nil
}

func TestRegistry_WriteDoesNotBlockObservations(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	c := r.Counter("hot_total", "Hot path counter.")
	for i := 0; i < 50; i++ {
		// Pre-register many sibling metrics so the Write loop has
		// enough iterations to expose a misplaced lock hold.
		r.Counter(fmt.Sprintf("filler_%d_total", i), "filler")
	}

	// Kick off a slow scrape; while it is sleeping inside slowWriter,
	// the hot-path Inc must complete promptly.
	scrapeDone := make(chan struct{})
	go func() {
		_ = r.Write(slowWriter{delay: 5 * time.Millisecond})
		close(scrapeDone)
	}()

	// Give the scraper time to enter the I/O loop.
	time.Sleep(2 * time.Millisecond)

	incDone := make(chan struct{})
	go func() {
		c.Inc()
		close(incDone)
	}()

	select {
	case <-incDone:
		// good — Inc was not blocked
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Inc blocked while Write held the registry lock")
	}
	<-scrapeDone
}

func TestCounter_DoubleRegistrationReturnsSame(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	a := r.Counter("x", "h", "k")
	b := r.Counter("x", "h", "k")
	if a != b {
		t.Fatal("re-registration must return the same Counter pointer")
	}
}

func TestCounter_ConflictingLabelsPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on label-schema mismatch")
		}
	}()

	r := NewRegistry()
	r.Counter("x", "h", "a")
	r.Counter("x", "h", "b") // boom
}

func TestCounter_LabelArityMismatchPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on label-arity mismatch")
		}
	}()

	r := NewRegistry()
	c := r.Counter("x", "h", "k1", "k2")
	c.Inc("only-one") // boom
}

func TestEscaping(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	c := r.Counter("e_total", "h", "label")
	c.Inc(`needs "escape" \and newline` + "\n")

	var buf bytes.Buffer
	_ = r.Write(&buf)
	if !strings.Contains(buf.String(), `\"escape\"`) {
		t.Errorf("quote not escaped:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), `\\and`) {
		t.Errorf("backslash not escaped:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), `\n`) {
		t.Errorf("newline not escaped:\n%s", buf.String())
	}
}
