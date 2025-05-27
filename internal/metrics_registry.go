package internal

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// MetricsRegistry is the central registry for all metrics
type MetricsRegistry struct {
	collector         *MetricsCollector
	resourceCollector *ResourceCollector
	config            *MetricsConfig
	mu                sync.RWMutex
	startTime         time.Time

	// Background workers
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// MetricsConfig configures the metrics system
type MetricsConfig struct {
	Enabled              bool
	CollectionInterval   time.Duration
	ResourceInterval     time.Duration
	HistogramMaxSize     int
	EnableProfiling      bool
	EnableDetailedErrors bool
}

// DefaultMetricsConfig returns default metrics configuration
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:              true,
		CollectionInterval:   10 * time.Second,
		ResourceInterval:     5 * time.Second,
		HistogramMaxSize:     10000,
		EnableProfiling:      false,
		EnableDetailedErrors: true,
	}
}

// NewMetricsRegistry creates a new metrics registry
func NewMetricsRegistry(config *MetricsConfig) *MetricsRegistry {
	if config == nil {
		config = DefaultMetricsConfig()
	}

	registry := &MetricsRegistry{
		collector:         NewMetricsCollector(),
		resourceCollector: NewResourceCollector(),
		config:            config,
		startTime:         time.Now(),
		stopChan:          make(chan struct{}),
	}

	if config.Enabled {
		registry.Start()
	}

	return registry
}

// Start starts background metric collection
func (r *MetricsRegistry) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Start resource collection
	r.wg.Add(1)
	go r.collectResources()

	// Start throughput calculation
	r.wg.Add(1)
	go r.calculateThroughput()
}

// Stop stops background metric collection
func (r *MetricsRegistry) Stop() {
	close(r.stopChan)
	r.wg.Wait()
}

// GetCollector returns the metrics collector
func (r *MetricsRegistry) GetCollector() *MetricsCollector {
	return r.collector
}

// GetSnapshot returns a snapshot of all metrics
func (r *MetricsRegistry) GetSnapshot() *MetricsSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := &MetricsSnapshot{
		Timestamp: time.Now(),
		Uptime:    time.Since(r.startTime),
		Metrics:   make(map[string]interface{}),
	}

	// Collect all metric families
	for _, family := range r.collector.GetAllMetricFamilies() {
		familyData := make(map[string]interface{})
		familyData["help"] = family.Help
		familyData["type"] = family.Type

		metrics := make(map[string]interface{})
		for _, metric := range family.GetAll() {
			key := labelsToKey(metric.Labels())
			if key == "" {
				key = "default"
			}
			metrics[key] = metric.Value()
		}
		familyData["metrics"] = metrics

		snapshot.Metrics[family.Name] = familyData
	}

	// Add resource metrics
	snapshot.Resources = r.resourceCollector.GetSnapshot()

	return snapshot
}

// collectResources periodically collects resource metrics
func (r *MetricsRegistry) collectResources() {
	defer r.wg.Done()

	ticker := time.NewTicker(r.config.ResourceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.resourceCollector.Collect()

			// Update metrics
			stats := r.resourceCollector.GetSnapshot()
			r.collector.UpdateResourceMetrics(
				int64(stats.MemStats.HeapAlloc),
				int64(stats.MemStats.HeapObjects),
				int64(stats.NumGoroutines),
			)

			// Record GC pauses
			for _, pause := range stats.RecentGCPauses {
				r.collector.RecordGCPause(pause)
			}

		case <-r.stopChan:
			return
		}
	}
}

// calculateThroughput periodically calculates throughput metrics
func (r *MetricsRegistry) calculateThroughput() {
	defer r.wg.Done()

	ticker := time.NewTicker(r.config.CollectionInterval)
	defer ticker.Stop()

	var lastOps int64
	var lastTime time.Time = time.Now()

	for {
		select {
		case <-ticker.C:
			now := time.Now()

			// Calculate operations per second
			currentOps := r.getTotalOperations()
			elapsed := now.Sub(lastTime).Seconds()

			if elapsed > 0 {
				opsPerSecond := float64(currentOps-lastOps) / elapsed
				r.collector.UpdateOperationsPerSecond(opsPerSecond)
			}

			lastOps = currentOps
			lastTime = now

		case <-r.stopChan:
			return
		}
	}
}

// getTotalOperations returns total operations count
func (r *MetricsRegistry) getTotalOperations() int64 {
	var total int64

	// Sum all operation counters
	for _, metric := range r.collector.ParseOperations.GetAll() {
		if counter, ok := metric.(*Counter); ok {
			total += counter.Get()
		}
	}

	for _, metric := range r.collector.EvalOperations.GetAll() {
		if counter, ok := metric.(*Counter); ok {
			total += counter.Get()
		}
	}

	return total
}

// MetricsSnapshot represents a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	Uptime    time.Duration          `json:"uptime"`
	Metrics   map[string]interface{} `json:"metrics"`
	Resources *ResourceSnapshot      `json:"resources"`
}

// ResourceCollector collects system resource metrics
type ResourceCollector struct {
	mu             sync.RWMutex
	memStats       runtime.MemStats
	numGoroutines  int
	numCPU         int
	recentGCPauses []time.Duration
}

// NewResourceCollector creates a new resource collector
func NewResourceCollector() *ResourceCollector {
	return &ResourceCollector{
		numCPU:         runtime.NumCPU(),
		recentGCPauses: make([]time.Duration, 0, 10),
	}
}

// Collect collects current resource metrics
func (rc *ResourceCollector) Collect() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Collect memory stats
	runtime.ReadMemStats(&rc.memStats)

	// Collect goroutine count
	rc.numGoroutines = runtime.NumGoroutine()

	// Extract recent GC pauses
	numGC := rc.memStats.NumGC
	if numGC > 0 {
		// Clear old pauses
		rc.recentGCPauses = rc.recentGCPauses[:0]

		// Get last 10 pauses
		startIdx := 0
		if numGC > 10 {
			startIdx = int((numGC - 10) % 256)
		}

		for i := 0; i < int(numGC) && i < 10; i++ {
			idx := (startIdx + i) % 256
			rc.recentGCPauses = append(rc.recentGCPauses, time.Duration(rc.memStats.PauseNs[idx]))
		}
	}
}

// GetSnapshot returns a snapshot of resource metrics
func (rc *ResourceCollector) GetSnapshot() *ResourceSnapshot {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return &ResourceSnapshot{
		MemStats:       rc.memStats,
		NumGoroutines:  rc.numGoroutines,
		NumCPU:         rc.numCPU,
		RecentGCPauses: append([]time.Duration{}, rc.recentGCPauses...),
	}
}

// ResourceSnapshot represents system resource metrics
type ResourceSnapshot struct {
	MemStats       runtime.MemStats `json:"memory"`
	NumGoroutines  int              `json:"goroutines"`
	NumCPU         int              `json:"cpus"`
	RecentGCPauses []time.Duration  `json:"recent_gc_pauses"`
}

// Global metrics registry
var globalMetricsRegistry *MetricsRegistry
var metricsOnce sync.Once

// InitializeMetricsRegistry initializes the global metrics registry
func InitializeMetricsRegistry(config *MetricsConfig) {
	metricsOnce.Do(func() {
		globalMetricsRegistry = NewMetricsRegistry(config)
	})
}

// GetMetricsRegistry returns the global metrics registry
func GetMetricsRegistry() *MetricsRegistry {
	if globalMetricsRegistry == nil {
		InitializeMetricsRegistry(nil)
	}
	return globalMetricsRegistry
}

// MetricsEnabled returns whether metrics are enabled
func MetricsEnabled() bool {
	registry := GetMetricsRegistry()
	return registry != nil && registry.config.Enabled
}

// WithMetrics wraps a function with metric collection
func WithMetrics(name string, labels map[string]string, fn func() error) error {
	if !MetricsEnabled() {
		return fn()
	}

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	// Record metric based on name
	mc := GetMetricsCollector()
	switch name {
	case "parse":
		mc.RecordParseOperation(start, err)
	case "eval":
		mc.RecordEvalOperation(start, err)
	default:
		// Custom metric
		if family, ok := mc.CustomMetrics[name]; ok {
			if histogram, ok := family.GetOrCreate(labels).(*Histogram); ok {
				histogram.Observe(duration.Seconds())
			}
		}
	}

	return err
}

// RecordMetric records a custom metric value
func RecordMetric(name string, value float64, labels map[string]string) {
	if !MetricsEnabled() {
		return
	}

	mc := GetMetricsCollector()
	if family, ok := mc.CustomMetrics[name]; ok {
		metric := family.GetOrCreate(labels)

		switch m := metric.(type) {
		case *Counter:
			m.Add(int64(value))
		case *Gauge:
			m.Set(int64(value))
		case *Histogram:
			m.Observe(value)
		}
	}
}

// FormatSnapshot formats a metrics snapshot for display
func FormatSnapshot(snapshot *MetricsSnapshot) string {
	// This would format the snapshot for human-readable display
	// Implementation depends on desired output format
	return fmt.Sprintf("Metrics snapshot at %s (uptime: %s)\n",
		snapshot.Timestamp.Format(time.RFC3339),
		snapshot.Uptime)
}
