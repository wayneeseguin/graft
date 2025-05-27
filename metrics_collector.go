package graft

import (
	"fmt"
	"time"
)

// MetricsCollector collects metrics for Graft operations
type MetricsCollector struct {
	// Parsing metrics
	ParseOperations  *MetricFamily
	ParseDuration    *MetricFamily
	ParseErrors      *MetricFamily
	
	// Evaluation metrics
	EvalOperations   *MetricFamily
	EvalDuration     *MetricFamily
	OperatorCalls    *MetricFamily
	OperatorDuration *MetricFamily
	
	// Cache metrics
	CacheHits        *MetricFamily
	CacheMisses      *MetricFamily
	CacheEvictions   *MetricFamily
	CacheSize        *MetricFamily
	
	// I/O metrics
	ExternalCalls    *MetricFamily
	ExternalDuration *MetricFamily
	ConnectionsActive *MetricFamily
	
	// Resource metrics
	HeapAlloc        *MetricFamily
	HeapObjects      *MetricFamily
	GCPauseTime      *MetricFamily
	Goroutines       *MetricFamily
	
	// Throughput metrics
	DocumentsProcessed *MetricFamily
	BytesProcessed     *MetricFamily
	OperationsPerSecond *MetricFamily
	
	// Error metrics
	ErrorsByType     *MetricFamily
	
	// Custom metrics
	CustomMetrics    map[string]*MetricFamily
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		// Parsing metrics
		ParseOperations: NewMetricFamily(
			"graft_parse_operations_total",
			"Total number of parse operations",
			MetricTypeCounter,
		),
		ParseDuration: NewMetricFamily(
			"graft_parse_duration_seconds",
			"Parse operation duration in seconds",
			MetricTypeHistogram,
		),
		ParseErrors: NewMetricFamily(
			"graft_parse_errors_total",
			"Total number of parse errors",
			MetricTypeCounter,
		),
		
		// Evaluation metrics
		EvalOperations: NewMetricFamily(
			"graft_eval_operations_total",
			"Total number of evaluation operations",
			MetricTypeCounter,
		),
		EvalDuration: NewMetricFamily(
			"graft_eval_duration_seconds",
			"Evaluation operation duration in seconds",
			MetricTypeHistogram,
		),
		OperatorCalls: NewMetricFamily(
			"graft_operator_calls_total",
			"Total number of operator calls",
			MetricTypeCounter,
		),
		OperatorDuration: NewMetricFamily(
			"graft_operator_duration_seconds",
			"Operator execution duration in seconds",
			MetricTypeHistogram,
		),
		
		// Cache metrics
		CacheHits: NewMetricFamily(
			"graft_cache_hits_total",
			"Total number of cache hits",
			MetricTypeCounter,
		),
		CacheMisses: NewMetricFamily(
			"graft_cache_misses_total",
			"Total number of cache misses",
			MetricTypeCounter,
		),
		CacheEvictions: NewMetricFamily(
			"graft_cache_evictions_total",
			"Total number of cache evictions",
			MetricTypeCounter,
		),
		CacheSize: NewMetricFamily(
			"graft_cache_size",
			"Current cache size",
			MetricTypeGauge,
		),
		
		// I/O metrics
		ExternalCalls: NewMetricFamily(
			"graft_external_calls_total",
			"Total number of external system calls",
			MetricTypeCounter,
		),
		ExternalDuration: NewMetricFamily(
			"graft_external_duration_seconds",
			"External call duration in seconds",
			MetricTypeHistogram,
		),
		ConnectionsActive: NewMetricFamily(
			"graft_connections_active",
			"Number of active connections",
			MetricTypeGauge,
		),
		
		// Resource metrics
		HeapAlloc: NewMetricFamily(
			"graft_heap_alloc_bytes",
			"Current heap allocation in bytes",
			MetricTypeGauge,
		),
		HeapObjects: NewMetricFamily(
			"graft_heap_objects",
			"Number of allocated heap objects",
			MetricTypeGauge,
		),
		GCPauseTime: NewMetricFamily(
			"graft_gc_pause_seconds",
			"GC pause duration in seconds",
			MetricTypeHistogram,
		),
		Goroutines: NewMetricFamily(
			"graft_goroutines",
			"Number of goroutines",
			MetricTypeGauge,
		),
		
		// Throughput metrics
		DocumentsProcessed: NewMetricFamily(
			"graft_documents_processed_total",
			"Total number of documents processed",
			MetricTypeCounter,
		),
		BytesProcessed: NewMetricFamily(
			"graft_bytes_processed_total",
			"Total bytes processed",
			MetricTypeCounter,
		),
		OperationsPerSecond: NewMetricFamily(
			"graft_operations_per_second",
			"Operations per second",
			MetricTypeGauge,
		),
		
		// Error metrics
		ErrorsByType: NewMetricFamily(
			"graft_errors_total",
			"Total errors by type",
			MetricTypeCounter,
		),
		
		// Custom metrics
		CustomMetrics: make(map[string]*MetricFamily),
	}
}

// RecordParseOperation records a parse operation
func (mc *MetricsCollector) RecordParseOperation(start time.Time, err error) {
	duration := time.Since(start)
	
	mc.ParseOperations.GetOrCreate(nil).(*Counter).Inc()
	mc.ParseDuration.GetOrCreate(nil).(*Histogram).Observe(duration.Seconds())
	
	if err != nil {
		mc.ParseErrors.GetOrCreate(nil).(*Counter).Inc()
		mc.RecordError("parse", err)
	}
}

// RecordEvalOperation records an evaluation operation
func (mc *MetricsCollector) RecordEvalOperation(start time.Time, err error) {
	duration := time.Since(start)
	
	mc.EvalOperations.GetOrCreate(nil).(*Counter).Inc()
	mc.EvalDuration.GetOrCreate(nil).(*Histogram).Observe(duration.Seconds())
	
	if err != nil {
		mc.RecordError("eval", err)
	}
}

// RecordOperatorCall records an operator call
func (mc *MetricsCollector) RecordOperatorCall(operator string, start time.Time, err error) {
	duration := time.Since(start)
	
	labels := map[string]string{"operator": operator}
	mc.OperatorCalls.GetOrCreate(labels).(*Counter).Inc()
	mc.OperatorDuration.GetOrCreate(labels).(*Histogram).Observe(duration.Seconds())
	
	if err != nil {
		mc.RecordError(fmt.Sprintf("operator_%s", operator), err)
	}
}

// RecordCacheHit records a cache hit
func (mc *MetricsCollector) RecordCacheHit(cacheType string) {
	labels := map[string]string{"cache": cacheType}
	mc.CacheHits.GetOrCreate(labels).(*Counter).Inc()
}

// RecordCacheMiss records a cache miss
func (mc *MetricsCollector) RecordCacheMiss(cacheType string) {
	labels := map[string]string{"cache": cacheType}
	mc.CacheMisses.GetOrCreate(labels).(*Counter).Inc()
}

// RecordCacheEviction records a cache eviction
func (mc *MetricsCollector) RecordCacheEviction(cacheType string, count int64) {
	labels := map[string]string{"cache": cacheType}
	mc.CacheEvictions.GetOrCreate(labels).(*Counter).Add(count)
}

// UpdateCacheSize updates the cache size gauge
func (mc *MetricsCollector) UpdateCacheSize(cacheType string, size int64) {
	labels := map[string]string{"cache": cacheType}
	mc.CacheSize.GetOrCreate(labels).(*Gauge).Set(size)
}

// RecordExternalCall records an external system call
func (mc *MetricsCollector) RecordExternalCall(system string, start time.Time, err error) {
	duration := time.Since(start)
	
	labels := map[string]string{"system": system}
	mc.ExternalCalls.GetOrCreate(labels).(*Counter).Inc()
	mc.ExternalDuration.GetOrCreate(labels).(*Histogram).Observe(duration.Seconds())
	
	if err != nil {
		mc.RecordError(fmt.Sprintf("external_%s", system), err)
	}
}

// UpdateConnectionsActive updates the active connections gauge
func (mc *MetricsCollector) UpdateConnectionsActive(system string, count int64) {
	labels := map[string]string{"system": system}
	mc.ConnectionsActive.GetOrCreate(labels).(*Gauge).Set(count)
}

// UpdateResourceMetrics updates resource usage metrics
func (mc *MetricsCollector) UpdateResourceMetrics(heapAlloc, heapObjects, goroutines int64) {
	mc.HeapAlloc.GetOrCreate(nil).(*Gauge).Set(heapAlloc)
	mc.HeapObjects.GetOrCreate(nil).(*Gauge).Set(heapObjects)
	mc.Goroutines.GetOrCreate(nil).(*Gauge).Set(goroutines)
}

// RecordGCPause records a GC pause duration
func (mc *MetricsCollector) RecordGCPause(duration time.Duration) {
	mc.GCPauseTime.GetOrCreate(nil).(*Histogram).Observe(duration.Seconds())
}

// RecordDocument records a processed document
func (mc *MetricsCollector) RecordDocument(bytes int64) {
	mc.DocumentsProcessed.GetOrCreate(nil).(*Counter).Inc()
	mc.BytesProcessed.GetOrCreate(nil).(*Counter).Add(bytes)
}

// UpdateOperationsPerSecond updates the operations per second gauge
func (mc *MetricsCollector) UpdateOperationsPerSecond(ops float64) {
	mc.OperationsPerSecond.GetOrCreate(nil).(*Gauge).Set(int64(ops))
}

// RecordError records an error by type
func (mc *MetricsCollector) RecordError(errorType string, err error) {
	labels := map[string]string{"type": errorType}
	mc.ErrorsByType.GetOrCreate(labels).(*Counter).Inc()
}

// RegisterCustomMetric registers a custom metric family
func (mc *MetricsCollector) RegisterCustomMetric(name, help string, metricType MetricType) *MetricFamily {
	family := NewMetricFamily(name, help, metricType)
	mc.CustomMetrics[name] = family
	return family
}

// GetAllMetricFamilies returns all metric families
func (mc *MetricsCollector) GetAllMetricFamilies() []*MetricFamily {
	families := []*MetricFamily{
		mc.ParseOperations,
		mc.ParseDuration,
		mc.ParseErrors,
		mc.EvalOperations,
		mc.EvalDuration,
		mc.OperatorCalls,
		mc.OperatorDuration,
		mc.CacheHits,
		mc.CacheMisses,
		mc.CacheEvictions,
		mc.CacheSize,
		mc.ExternalCalls,
		mc.ExternalDuration,
		mc.ConnectionsActive,
		mc.HeapAlloc,
		mc.HeapObjects,
		mc.GCPauseTime,
		mc.Goroutines,
		mc.DocumentsProcessed,
		mc.BytesProcessed,
		mc.OperationsPerSecond,
		mc.ErrorsByType,
	}
	
	// Add custom metrics
	for _, family := range mc.CustomMetrics {
		families = append(families, family)
	}
	
	return families
}

// Reset resets all metrics
func (mc *MetricsCollector) Reset() {
	for _, family := range mc.GetAllMetricFamilies() {
		family.Reset()
	}
}

// Global metrics collector instance
var globalMetricsCollector *MetricsCollector

// InitializeMetrics initializes the global metrics collector
func InitializeMetrics() {
	globalMetricsCollector = NewMetricsCollector()
}

// GetMetricsCollector returns the global metrics collector
func GetMetricsCollector() *MetricsCollector {
	if globalMetricsCollector == nil {
		InitializeMetrics()
	}
	return globalMetricsCollector
}

// Convenience functions for common metrics

// RecordCacheMetrics records cache hit/miss in one call
func RecordCacheMetrics(cacheType string, hit bool) {
	mc := GetMetricsCollector()
	if hit {
		mc.RecordCacheHit(cacheType)
	} else {
		mc.RecordCacheMiss(cacheType)
	}
}

// TimeOperation times an operation and records metrics
func TimeOperation(metricType string, labels map[string]string, fn func() error) error {
	start := time.Now()
	err := fn()
	
	mc := GetMetricsCollector()
	switch metricType {
	case "parse":
		mc.RecordParseOperation(start, err)
	case "eval":
		mc.RecordEvalOperation(start, err)
	default:
		if operator, ok := labels["operator"]; ok {
			mc.RecordOperatorCall(operator, start, err)
		}
	}
	
	return err
}