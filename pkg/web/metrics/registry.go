// Package metrics implements a minimal Prometheus-compatible metrics
// registry with no external dependencies.
//
// Three metric primitives are supported: Counter (monotonic), Gauge
// (settable), Histogram (bucketed observations with sum and count).
// Each metric carries an immutable schema (name, help, label keys) and
// stores per-label-set state under a string-hashed key. Output follows
// the Prometheus text exposition format version 0.0.4 so a /metrics
// endpoint can be scraped by any standard collector.
//
// Design constraints:
//
//   - Zero external dependencies. The framework's go.mod stays clean.
//   - Cardinality is the caller's responsibility. The registry does not
//     bound the number of label-value combinations; high-cardinality
//     labels (user_id, request_path) will eventually exhaust memory.
//   - Concurrent-safe via sync.Map and atomic operations. No global mutex
//     on the hot path.
//
// See pkg/web/middleware/metrics.go for the http.Handler middleware
// that uses this registry.
package metrics

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// float64bits / float64frombits delegate to math but are aliased for
// readability inside the atomic CAS loops below.
var (
	float64bits     = math.Float64bits
	float64frombits = math.Float64frombits
)

// Registry owns a set of named metrics and exposes them in Prometheus
// text format via Write. Metric registration is lock-protected; the
// hot-path observation calls (Inc/Add/Observe/Set) are lock-free.
type Registry struct {
	mu      sync.RWMutex
	metrics map[string]metric
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{metrics: make(map[string]metric)}
}

// metric is the internal type-erased contract every primitive satisfies
// so Write can iterate the registry without knowing the concrete type.
type metric interface {
	name() string
	writeTo(w io.Writer) error
}

// Counter registers (or returns an existing) monotonic counter under
// the given name. labelKeys defines which labels every observation must
// supply, in fixed order.
//
// Re-registering the same name with different labelKeys panics: a
// metric's schema is immutable after first registration.
func (r *Registry) Counter(name, help string, labelKeys ...string) *Counter {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.metrics[name]; ok {
		c, ok := existing.(*Counter)
		if !ok {
			panic(fmt.Sprintf("metrics: %q already registered as %T", name, existing))
		}
		assertLabels(name, c.labelKeys, labelKeys)
		return c
	}

	c := &Counter{
		metricName: name,
		help:       help,
		labelKeys:  append([]string(nil), labelKeys...),
	}
	r.metrics[name] = c
	return c
}

// Gauge registers (or returns an existing) settable gauge.
func (r *Registry) Gauge(name, help string, labelKeys ...string) *Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.metrics[name]; ok {
		g, ok := existing.(*Gauge)
		if !ok {
			panic(fmt.Sprintf("metrics: %q already registered as %T", name, existing))
		}
		assertLabels(name, g.labelKeys, labelKeys)
		return g
	}

	g := &Gauge{
		metricName: name,
		help:       help,
		labelKeys:  append([]string(nil), labelKeys...),
	}
	r.metrics[name] = g
	return g
}

// Histogram registers (or returns an existing) histogram with the given
// upper-bound buckets. Buckets must be sorted ascending; an implicit +Inf
// bucket is always appended. Pass nil to use DefaultBuckets.
func (r *Registry) Histogram(name, help string, buckets []float64, labelKeys ...string) *Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.metrics[name]; ok {
		h, ok := existing.(*Histogram)
		if !ok {
			panic(fmt.Sprintf("metrics: %q already registered as %T", name, existing))
		}
		assertLabels(name, h.labelKeys, labelKeys)
		return h
	}

	if buckets == nil {
		buckets = DefaultBuckets
	}
	if !isSorted(buckets) {
		panic(fmt.Sprintf("metrics: histogram %q buckets must be sorted ascending", name))
	}

	h := &Histogram{
		metricName: name,
		help:       help,
		labelKeys:  append([]string(nil), labelKeys...),
		buckets:    append([]float64(nil), buckets...),
	}
	r.metrics[name] = h
	return h
}

// Write serializes every registered metric in Prometheus text format
// version 0.0.4. Output is sorted by metric name for deterministic
// scrapes (and easier diffing in tests).
//
// The method is intentionally named Write rather than WriteTo to avoid
// satisfying io.WriterTo (which mandates a (int64, error) return type
// and is flagged by go vet -stdmethods otherwise).
//
// Locking strategy: take the registry RLock exactly once, copy both
// the names and the *metric pointers into local slices, then drop the
// lock before any I/O. Per-metric observation paths (Counter.Add,
// Gauge.Set, Histogram.Observe) remain lock-free, so writers are
// never blocked by a long-running scrape — and a scrape on a registry
// holding 1000 metrics no longer dances on the lock 1000 times.
func (r *Registry) Write(w io.Writer) error {
	r.mu.RLock()
	names := make([]string, 0, len(r.metrics))
	for name := range r.metrics {
		names = append(names, name)
	}
	sort.Strings(names)

	snapshot := make([]metric, len(names))
	for i, name := range names {
		snapshot[i] = r.metrics[name]
	}
	r.mu.RUnlock()

	for _, m := range snapshot {
		if err := m.writeTo(w); err != nil {
			return err
		}
	}
	return nil
}

// DefaultBuckets matches the canonical Prometheus client-go defaults,
// suitable for HTTP request duration in seconds.
var DefaultBuckets = []float64{
	.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10,
}

// assertLabels panics with a clear message when label schemas do not
// match. Defensive only — callers should not rely on the panic, but it
// catches typos at registration time which is much better than silently
// emitting two metrics with the same name.
func assertLabels(name string, want, got []string) {
	if !equalStrings(want, got) {
		panic(fmt.Sprintf(
			"metrics: %q already registered with labels %v, refusing to redefine as %v",
			name, want, got,
		))
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isSorted(s []float64) bool {
	for i := 1; i < len(s); i++ {
		if s[i] <= s[i-1] {
			return false
		}
	}
	return true
}

// labelKey hashes a label-value tuple into a stable key for sync.Map
// lookups. Uses NUL as a separator so common ASCII values cannot
// collide. Order matches the schema (labelKeys), so callers must pass
// values in registration order.
func labelKey(values []string) string {
	if len(values) == 0 {
		return ""
	}
	var sb strings.Builder
	size := len(values) - 1
	for _, v := range values {
		size += len(v)
	}
	sb.Grow(size)
	for i, v := range values {
		if i > 0 {
			sb.WriteByte(0)
		}
		sb.WriteString(v)
	}
	return sb.String()
}

// labelPairsFor reconstructs a (key, value) sequence for serialization
// from the immutable label keys and the per-observation label values.
type labelPair struct{ Key, Value string }

func zip(keys, values []string) []labelPair {
	out := make([]labelPair, len(keys))
	for i := range keys {
		out[i] = labelPair{Key: keys[i], Value: values[i]}
	}
	return out
}

// formatLabels renders a sorted "{k1=\"v1\",k2=\"v2\"}" suffix or empty
// string when there are no labels. Sorting keeps Write output stable
// regardless of label registration order.
func formatLabels(pairs []labelPair, extra ...labelPair) string {
	all := append([]labelPair{}, pairs...)
	all = append(all, extra...)
	if len(all) == 0 {
		return ""
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Key < all[j].Key })

	var sb strings.Builder
	sb.WriteByte('{')
	for i, p := range all {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(p.Key)
		sb.WriteByte('=')
		sb.WriteByte('"')
		writeEscaped(&sb, p.Value)
		sb.WriteByte('"')
	}
	sb.WriteByte('}')
	return sb.String()
}

// writeEscaped escapes per the Prometheus text format spec: backslash,
// double-quote, and newline get a leading backslash.
func writeEscaped(sb *strings.Builder, v string) {
	for _, r := range v {
		switch r {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		case '\n':
			sb.WriteString(`\n`)
		default:
			sb.WriteRune(r)
		}
	}
}

// counterValue is the per-label-set state for a Counter.
type counterValue struct {
	labels []string
	v      atomic.Uint64
}

// Counter is a monotonic uint64 counter sharded by label-value tuple.
type Counter struct {
	metricName string
	help       string
	labelKeys  []string
	values     sync.Map // key: labelKey(values), value: *counterValue
}

// Inc increments the counter for the given label values by 1.
func (c *Counter) Inc(labelValues ...string) { c.Add(1, labelValues...) }

// Add increments the counter by delta. Negative deltas are silently
// ignored to preserve monotonicity (Prometheus rate() relies on it).
func (c *Counter) Add(delta uint64, labelValues ...string) {
	if delta == 0 {
		return
	}
	if len(labelValues) != len(c.labelKeys) {
		panic(fmt.Sprintf("metrics: counter %q expects %d labels, got %d",
			c.metricName, len(c.labelKeys), len(labelValues)))
	}
	key := labelKey(labelValues)
	cv, _ := c.values.LoadOrStore(key, &counterValue{labels: append([]string(nil), labelValues...)})
	cv.(*counterValue).v.Add(delta)
}

func (c *Counter) name() string { return c.metricName }

func (c *Counter) writeTo(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n", c.metricName, c.help, c.metricName); err != nil {
		return err
	}
	rows := c.snapshot()
	for _, row := range rows {
		if _, err := fmt.Fprintf(w, "%s%s %d\n", c.metricName, formatLabels(row.pairs), row.value); err != nil {
			return err
		}
	}
	return nil
}

type counterRow struct {
	pairs []labelPair
	value uint64
}

func (c *Counter) snapshot() []counterRow {
	var rows []counterRow
	c.values.Range(func(_, v any) bool {
		cv := v.(*counterValue)
		rows = append(rows, counterRow{
			pairs: zip(c.labelKeys, cv.labels),
			value: cv.v.Load(),
		})
		return true
	})
	sort.Slice(rows, func(i, j int) bool {
		return formatLabels(rows[i].pairs) < formatLabels(rows[j].pairs)
	})
	return rows
}

// gaugeValue is the per-label-set state for a Gauge. Stored as int64
// bits via atomic.Int64; we use math.Float64bits to round-trip floats.
type gaugeValue struct {
	labels []string
	v      atomic.Uint64
}

// Gauge is a settable float64 metric sharded by label-value tuple.
type Gauge struct {
	metricName string
	help       string
	labelKeys  []string
	values     sync.Map
}

// Set replaces the current value.
func (g *Gauge) Set(value float64, labelValues ...string) {
	if len(labelValues) != len(g.labelKeys) {
		panic(fmt.Sprintf("metrics: gauge %q expects %d labels, got %d",
			g.metricName, len(g.labelKeys), len(labelValues)))
	}
	key := labelKey(labelValues)
	gv, _ := g.values.LoadOrStore(key, &gaugeValue{labels: append([]string(nil), labelValues...)})
	gv.(*gaugeValue).v.Store(float64bits(value))
}

// Inc adds 1 to the current value.
func (g *Gauge) Inc(labelValues ...string) { g.Add(1, labelValues...) }

// Dec subtracts 1 from the current value.
func (g *Gauge) Dec(labelValues ...string) { g.Add(-1, labelValues...) }

// Add adds delta (which may be negative) to the current value.
func (g *Gauge) Add(delta float64, labelValues ...string) {
	if len(labelValues) != len(g.labelKeys) {
		panic(fmt.Sprintf("metrics: gauge %q expects %d labels, got %d",
			g.metricName, len(g.labelKeys), len(labelValues)))
	}
	key := labelKey(labelValues)
	gv, _ := g.values.LoadOrStore(key, &gaugeValue{labels: append([]string(nil), labelValues...)})
	for {
		old := gv.(*gaugeValue).v.Load()
		newVal := float64bits(float64frombits(old) + delta)
		if gv.(*gaugeValue).v.CompareAndSwap(old, newVal) {
			return
		}
	}
}

func (g *Gauge) name() string { return g.metricName }

func (g *Gauge) writeTo(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n", g.metricName, g.help, g.metricName); err != nil {
		return err
	}
	type row struct {
		pairs []labelPair
		value float64
	}
	var rows []row
	g.values.Range(func(_, v any) bool {
		gv := v.(*gaugeValue)
		rows = append(rows, row{
			pairs: zip(g.labelKeys, gv.labels),
			value: float64frombits(gv.v.Load()),
		})
		return true
	})
	sort.Slice(rows, func(i, j int) bool {
		return formatLabels(rows[i].pairs) < formatLabels(rows[j].pairs)
	})
	for _, r := range rows {
		if _, err := fmt.Fprintf(w, "%s%s %s\n", g.metricName, formatLabels(r.pairs), formatFloat(r.value)); err != nil {
			return err
		}
	}
	return nil
}

// histogramValue is the per-label-set state for a Histogram.
// buckets[i] = cumulative count for buckets <= upperBounds[i].
type histogramValue struct {
	labels    []string
	buckets   []atomic.Uint64
	sumBits   atomic.Uint64 // float64 sum stored as bits
	count     atomic.Uint64
}

// Histogram is a bucketed float64 observation tracker compatible with
// Prometheus histogram_quantile() server-side aggregation.
type Histogram struct {
	metricName string
	help       string
	labelKeys  []string
	buckets    []float64
	values     sync.Map
}

// Observe records a single observation under the given label values.
func (h *Histogram) Observe(value float64, labelValues ...string) {
	if len(labelValues) != len(h.labelKeys) {
		panic(fmt.Sprintf("metrics: histogram %q expects %d labels, got %d",
			h.metricName, len(h.labelKeys), len(labelValues)))
	}
	key := labelKey(labelValues)
	hv, ok := h.values.Load(key)
	if !ok {
		fresh := &histogramValue{
			labels:  append([]string(nil), labelValues...),
			buckets: make([]atomic.Uint64, len(h.buckets)),
		}
		hv, _ = h.values.LoadOrStore(key, fresh)
	}
	v := hv.(*histogramValue)

	for i, ub := range h.buckets {
		if value <= ub {
			v.buckets[i].Add(1)
		}
	}
	v.count.Add(1)
	for {
		old := v.sumBits.Load()
		newVal := float64bits(float64frombits(old) + value)
		if v.sumBits.CompareAndSwap(old, newVal) {
			return
		}
	}
}

func (h *Histogram) name() string { return h.metricName }

func (h *Histogram) writeTo(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s histogram\n", h.metricName, h.help, h.metricName); err != nil {
		return err
	}
	type row struct {
		pairs   []labelPair
		buckets []uint64
		sum     float64
		count   uint64
	}
	var rows []row
	h.values.Range(func(_, v any) bool {
		hv := v.(*histogramValue)
		bs := make([]uint64, len(hv.buckets))
		for i := range hv.buckets {
			bs[i] = hv.buckets[i].Load()
		}
		rows = append(rows, row{
			pairs:   zip(h.labelKeys, hv.labels),
			buckets: bs,
			sum:     float64frombits(hv.sumBits.Load()),
			count:   hv.count.Load(),
		})
		return true
	})
	sort.Slice(rows, func(i, j int) bool {
		return formatLabels(rows[i].pairs) < formatLabels(rows[j].pairs)
	})
	for _, r := range rows {
		for i, ub := range h.buckets {
			label := formatLabels(r.pairs, labelPair{Key: "le", Value: formatFloat(ub)})
			if _, err := fmt.Fprintf(w, "%s_bucket%s %d\n", h.metricName, label, r.buckets[i]); err != nil {
				return err
			}
		}
		infLabel := formatLabels(r.pairs, labelPair{Key: "le", Value: "+Inf"})
		if _, err := fmt.Fprintf(w, "%s_bucket%s %d\n", h.metricName, infLabel, r.count); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s_sum%s %s\n", h.metricName, formatLabels(r.pairs), formatFloat(r.sum)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s_count%s %d\n", h.metricName, formatLabels(r.pairs), r.count); err != nil {
			return err
		}
	}
	return nil
}

// formatFloat renders a float64 using the shortest representation
// Prometheus accepts. Integers come out as "1234", non-integers use
// strconv's 'g' format which avoids trailing zeros.
func formatFloat(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%g", v)
}
