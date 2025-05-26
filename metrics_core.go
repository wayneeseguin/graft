package spruce

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric is the base interface for all metrics
type Metric interface {
	Name() string
	Type() MetricType
	Labels() map[string]string
	Value() interface{}
	Reset()
}

// Counter is a monotonically increasing metric
type Counter struct {
	name   string
	labels map[string]string
	value  int64
}

// NewCounter creates a new counter metric
func NewCounter(name string, labels map[string]string) *Counter {
	return &Counter{
		name:   name,
		labels: labels,
	}
}

// Inc increments the counter by 1
func (c *Counter) Inc() {
	atomic.AddInt64(&c.value, 1)
}

// Add adds the given value to the counter
func (c *Counter) Add(delta int64) {
	atomic.AddInt64(&c.value, delta)
}

// Get returns the current value
func (c *Counter) Get() int64 {
	return atomic.LoadInt64(&c.value)
}

// Name returns the metric name
func (c *Counter) Name() string { return c.name }

// Type returns the metric type
func (c *Counter) Type() MetricType { return MetricTypeCounter }

// Labels returns the metric labels
func (c *Counter) Labels() map[string]string { return c.labels }

// Value returns the current value as interface{}
func (c *Counter) Value() interface{} { return c.Get() }

// Reset resets the counter to zero
func (c *Counter) Reset() {
	atomic.StoreInt64(&c.value, 0)
}

// Gauge is a metric that can go up and down
type Gauge struct {
	name   string
	labels map[string]string
	value  int64
}

// NewGauge creates a new gauge metric
func NewGauge(name string, labels map[string]string) *Gauge {
	return &Gauge{
		name:   name,
		labels: labels,
	}
}

// Set sets the gauge to the given value
func (g *Gauge) Set(value int64) {
	atomic.StoreInt64(&g.value, value)
}

// Inc increments the gauge by 1
func (g *Gauge) Inc() {
	atomic.AddInt64(&g.value, 1)
}

// Dec decrements the gauge by 1
func (g *Gauge) Dec() {
	atomic.AddInt64(&g.value, -1)
}

// Add adds the given value to the gauge
func (g *Gauge) Add(delta int64) {
	atomic.AddInt64(&g.value, delta)
}

// Get returns the current value
func (g *Gauge) Get() int64 {
	return atomic.LoadInt64(&g.value)
}

// Name returns the metric name
func (g *Gauge) Name() string { return g.name }

// Type returns the metric type
func (g *Gauge) Type() MetricType { return MetricTypeGauge }

// Labels returns the metric labels
func (g *Gauge) Labels() map[string]string { return g.labels }

// Value returns the current value as interface{}
func (g *Gauge) Value() interface{} { return g.Get() }

// Reset resets the gauge to zero
func (g *Gauge) Reset() {
	atomic.StoreInt64(&g.value, 0)
}

// Histogram tracks the distribution of values
type Histogram struct {
	name   string
	labels map[string]string
	mu     sync.RWMutex
	values []float64
	sum    float64
	count  int64
}

// NewHistogram creates a new histogram metric
func NewHistogram(name string, labels map[string]string) *Histogram {
	return &Histogram{
		name:   name,
		labels: labels,
		values: make([]float64, 0, 1000),
	}
}

// Observe adds a value to the histogram
func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.values = append(h.values, value)
	h.sum += value
	h.count++
	
	// Keep only last 10000 values to prevent unbounded growth
	if len(h.values) > 10000 {
		h.values = h.values[len(h.values)-10000:]
	}
}

// ObserveDuration observes the duration since the given time
func (h *Histogram) ObserveDuration(start time.Time) {
	h.Observe(time.Since(start).Seconds())
}

// GetStats returns histogram statistics
func (h *Histogram) GetStats() HistogramStats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if h.count == 0 {
		return HistogramStats{}
	}
	
	// Calculate percentiles
	sorted := make([]float64, len(h.values))
	copy(sorted, h.values)
	quickSort(sorted)
	
	return HistogramStats{
		Count:  h.count,
		Sum:    h.sum,
		Min:    sorted[0],
		Max:    sorted[len(sorted)-1],
		Mean:   h.sum / float64(h.count),
		P50:    percentile(sorted, 0.50),
		P90:    percentile(sorted, 0.90),
		P95:    percentile(sorted, 0.95),
		P99:    percentile(sorted, 0.99),
	}
}

// Name returns the metric name
func (h *Histogram) Name() string { return h.name }

// Type returns the metric type
func (h *Histogram) Type() MetricType { return MetricTypeHistogram }

// Labels returns the metric labels
func (h *Histogram) Labels() map[string]string { return h.labels }

// Value returns the histogram statistics
func (h *Histogram) Value() interface{} { return h.GetStats() }

// Reset resets the histogram
func (h *Histogram) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values = h.values[:0]
	h.sum = 0
	h.count = 0
}

// HistogramStats represents histogram statistics
type HistogramStats struct {
	Count int64   `json:"count"`
	Sum   float64 `json:"sum"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Mean  float64 `json:"mean"`
	P50   float64 `json:"p50"`
	P90   float64 `json:"p90"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
}

// Summary tracks value distribution with configurable quantiles
type Summary struct {
	*Histogram
	quantiles []float64
}

// NewSummary creates a new summary metric
func NewSummary(name string, labels map[string]string, quantiles []float64) *Summary {
	return &Summary{
		Histogram: NewHistogram(name, labels),
		quantiles: quantiles,
	}
}

// GetQuantiles returns the configured quantiles
func (s *Summary) GetQuantiles() map[float64]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.count == 0 {
		return nil
	}
	
	sorted := make([]float64, len(s.values))
	copy(sorted, s.values)
	quickSort(sorted)
	
	result := make(map[float64]float64)
	for _, q := range s.quantiles {
		result[q] = percentile(sorted, q)
	}
	
	return result
}

// Type returns the metric type
func (s *Summary) Type() MetricType { return MetricTypeSummary }

// Value returns the summary statistics
func (s *Summary) Value() interface{} {
	return map[string]interface{}{
		"stats":     s.GetStats(),
		"quantiles": s.GetQuantiles(),
	}
}

// MetricFamily groups related metrics
type MetricFamily struct {
	Name        string
	Help        string
	Type        MetricType
	metrics     map[string]Metric
	mu          sync.RWMutex
}

// NewMetricFamily creates a new metric family
func NewMetricFamily(name, help string, metricType MetricType) *MetricFamily {
	return &MetricFamily{
		Name:    name,
		Help:    help,
		Type:    metricType,
		metrics: make(map[string]Metric),
	}
}

// GetOrCreate gets or creates a metric with the given labels
func (f *MetricFamily) GetOrCreate(labels map[string]string) Metric {
	key := labelsToKey(labels)
	
	f.mu.RLock()
	metric, exists := f.metrics[key]
	f.mu.RUnlock()
	
	if exists {
		return metric
	}
	
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Double-check after acquiring write lock
	if metric, exists := f.metrics[key]; exists {
		return metric
	}
	
	// Create new metric
	switch f.Type {
	case MetricTypeCounter:
		metric = NewCounter(f.Name, labels)
	case MetricTypeGauge:
		metric = NewGauge(f.Name, labels)
	case MetricTypeHistogram:
		metric = NewHistogram(f.Name, labels)
	case MetricTypeSummary:
		metric = NewSummary(f.Name, labels, []float64{0.5, 0.9, 0.95, 0.99})
	default:
		panic(fmt.Sprintf("unknown metric type: %s", f.Type))
	}
	
	f.metrics[key] = metric
	return metric
}

// GetAll returns all metrics in the family
func (f *MetricFamily) GetAll() []Metric {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	metrics := make([]Metric, 0, len(f.metrics))
	for _, m := range f.metrics {
		metrics = append(metrics, m)
	}
	return metrics
}

// Reset resets all metrics in the family
func (f *MetricFamily) Reset() {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	for _, m := range f.metrics {
		m.Reset()
	}
}

// Helper functions

func labelsToKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	
	// Sort label keys for consistent ordering
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	quickSortStrings(keys)
	
	// Build key
	var key string
	for i, k := range keys {
		if i > 0 {
			key += ","
		}
		key += fmt.Sprintf("%s=%s", k, labels[k])
	}
	
	return key
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	
	index := int(float64(len(sorted)-1) * p)
	return sorted[index]
}

func quickSort(arr []float64) {
	if len(arr) < 2 {
		return
	}
	
	left, right := 0, len(arr)-1
	pivot := len(arr) / 2
	
	arr[pivot], arr[right] = arr[right], arr[pivot]
	
	for i := range arr {
		if arr[i] < arr[right] {
			arr[left], arr[i] = arr[i], arr[left]
			left++
		}
	}
	
	arr[left], arr[right] = arr[right], arr[left]
	
	quickSort(arr[:left])
	quickSort(arr[left+1:])
}

func quickSortStrings(arr []string) {
	if len(arr) < 2 {
		return
	}
	
	left, right := 0, len(arr)-1
	pivot := len(arr) / 2
	
	arr[pivot], arr[right] = arr[right], arr[pivot]
	
	for i := range arr {
		if arr[i] < arr[right] {
			arr[left], arr[i] = arr[i], arr[left]
			left++
		}
	}
	
	arr[left], arr[right] = arr[right], arr[left]
	
	quickSortStrings(arr[:left])
	quickSortStrings(arr[left+1:])
}