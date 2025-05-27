package spruce

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/geofffranks/spruce/log"
)

// ParallelMetricsCollector collects parallel execution metrics
type ParallelMetricsCollector struct {
	mu sync.RWMutex
	
	// Operation metrics
	opsTotal          atomic.Int64
	opsParallel       atomic.Int64
	opsSequential     atomic.Int64
	opsFailed         atomic.Int64
	
	// Timing metrics
	totalDuration     atomic.Int64 // nanoseconds
	parallelDuration  atomic.Int64
	sequentialDuration atomic.Int64
	
	// Concurrency metrics
	maxConcurrency    atomic.Int32
	currentConcurrency atomic.Int32
	
	// Lock metrics
	lockWaitTime      atomic.Int64 // nanoseconds
	lockAcquisitions  atomic.Int64
	lockContentions   atomic.Int64
	
	// Worker metrics
	workerUtilization map[int]*WorkerMetrics
	
	// Histogram data
	opDurations      []time.Duration
	lockWaitDurations []time.Duration
}

// WorkerMetrics tracks per-worker statistics
type WorkerMetrics struct {
	TasksExecuted atomic.Int64
	BusyTime      atomic.Int64 // nanoseconds
	IdleTime      atomic.Int64
}

// GlobalMetrics is the global metrics instance
var (
	GlobalMetrics     *ParallelMetricsCollector
	globalMetricsOnce sync.Once
)

// GetParallelMetrics returns the global parallel metrics collector
func GetParallelMetrics() *ParallelMetricsCollector {
	globalMetricsOnce.Do(func() {
		GlobalMetrics = NewParallelMetricsCollector()
	})
	return GlobalMetrics
}

// NewParallelMetricsCollector creates a new metrics collector
func NewParallelMetricsCollector() *ParallelMetricsCollector {
	return &ParallelMetricsCollector{
		workerUtilization: make(map[int]*WorkerMetrics),
		opDurations:       make([]time.Duration, 0, 1000),
		lockWaitDurations: make([]time.Duration, 0, 1000),
	}
}

// RecordOperation records an operation execution
func (mc *ParallelMetricsCollector) RecordOperation(parallel bool, duration time.Duration, err error) {
	mc.opsTotal.Add(1)
	
	if parallel {
		mc.opsParallel.Add(1)
		mc.parallelDuration.Add(duration.Nanoseconds())
	} else {
		mc.opsSequential.Add(1)
		mc.sequentialDuration.Add(duration.Nanoseconds())
	}
	
	if err != nil {
		mc.opsFailed.Add(1)
	}
	
	mc.totalDuration.Add(duration.Nanoseconds())
	
	// Store duration for histogram
	mc.mu.Lock()
	mc.opDurations = append(mc.opDurations, duration)
	// Keep only last 1000 durations
	if len(mc.opDurations) > 1000 {
		mc.opDurations = mc.opDurations[len(mc.opDurations)-1000:]
	}
	mc.mu.Unlock()
}

// RecordConcurrency updates concurrency metrics
func (mc *ParallelMetricsCollector) RecordConcurrency(delta int32) {
	current := mc.currentConcurrency.Add(delta)
	
	// Update max concurrency
	for {
		max := mc.maxConcurrency.Load()
		if current <= max || mc.maxConcurrency.CompareAndSwap(max, current) {
			break
		}
	}
}

// RecordLockWait records lock wait time
func (mc *ParallelMetricsCollector) RecordLockWait(duration time.Duration) {
	mc.lockWaitTime.Add(duration.Nanoseconds())
	mc.lockAcquisitions.Add(1)
	
	if duration > time.Microsecond {
		mc.lockContentions.Add(1)
	}
	
	mc.mu.Lock()
	mc.lockWaitDurations = append(mc.lockWaitDurations, duration)
	if len(mc.lockWaitDurations) > 1000 {
		mc.lockWaitDurations = mc.lockWaitDurations[len(mc.lockWaitDurations)-1000:]
	}
	mc.mu.Unlock()
}

// RecordWorkerActivity records worker activity
func (mc *ParallelMetricsCollector) RecordWorkerActivity(workerID int, busy bool, duration time.Duration) {
	mc.mu.Lock()
	worker, ok := mc.workerUtilization[workerID]
	if !ok {
		worker = &WorkerMetrics{}
		mc.workerUtilization[workerID] = worker
	}
	mc.mu.Unlock()
	
	if busy {
		worker.BusyTime.Add(duration.Nanoseconds())
		worker.TasksExecuted.Add(1)
	} else {
		worker.IdleTime.Add(duration.Nanoseconds())
	}
}

// GetSnapshot returns a snapshot of current metrics
func (mc *ParallelMetricsCollector) GetSnapshot() ParallelMetricsSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	total := mc.opsTotal.Load()
	parallel := mc.opsParallel.Load()
	sequential := mc.opsSequential.Load()
	failed := mc.opsFailed.Load()
	
	totalDuration := time.Duration(mc.totalDuration.Load())
	parallelDuration := time.Duration(mc.parallelDuration.Load())
	sequentialDuration := time.Duration(mc.sequentialDuration.Load())
	
	// Calculate speedup
	speedup := 1.0
	if sequential > 0 && parallel > 0 {
		avgSequential := float64(sequentialDuration) / float64(sequential)
		avgParallel := float64(parallelDuration) / float64(parallel)
		if avgParallel > 0 {
			speedup = avgSequential / avgParallel
		}
	}
	
	// Calculate worker utilization
	workerStats := make(map[int]float64)
	for id, worker := range mc.workerUtilization {
		busy := worker.BusyTime.Load()
		idle := worker.IdleTime.Load()
		total := busy + idle
		if total > 0 {
			workerStats[id] = float64(busy) / float64(total) * 100
		}
	}
	
	// Calculate percentiles for operation durations
	opPercentiles := calculatePercentiles(mc.opDurations)
	lockPercentiles := calculatePercentiles(mc.lockWaitDurations)
	
	return ParallelMetricsSnapshot{
		OpsTotal:           total,
		OpsParallel:        parallel,
		OpsSequential:      sequential,
		OpsFailed:          failed,
		
		TotalDuration:      totalDuration,
		ParallelDuration:   parallelDuration,
		SequentialDuration: sequentialDuration,
		
		MaxConcurrency:     mc.maxConcurrency.Load(),
		CurrentConcurrency: mc.currentConcurrency.Load(),
		
		LockWaitTime:      time.Duration(mc.lockWaitTime.Load()),
		LockAcquisitions:  mc.lockAcquisitions.Load(),
		LockContentions:   mc.lockContentions.Load(),
		
		Speedup:           speedup,
		ParallelRatio:     float64(parallel) / float64(total) * 100,
		FailureRate:       float64(failed) / float64(total) * 100,
		WorkerUtilization: workerStats,
		
		OpDurationP50:     opPercentiles.P50,
		OpDurationP95:     opPercentiles.P95,
		OpDurationP99:     opPercentiles.P99,
		
		LockWaitP50:       lockPercentiles.P50,
		LockWaitP95:       lockPercentiles.P95,
		LockWaitP99:       lockPercentiles.P99,
	}
}

// ParallelMetricsSnapshot represents a point-in-time snapshot of metrics
type ParallelMetricsSnapshot struct {
	// Operation counts
	OpsTotal      int64
	OpsParallel   int64
	OpsSequential int64
	OpsFailed     int64
	
	// Timing
	TotalDuration      time.Duration
	ParallelDuration   time.Duration
	SequentialDuration time.Duration
	
	// Concurrency
	MaxConcurrency     int32
	CurrentConcurrency int32
	
	// Lock statistics
	LockWaitTime     time.Duration
	LockAcquisitions int64
	LockContentions  int64
	
	// Calculated metrics
	Speedup           float64
	ParallelRatio     float64
	FailureRate       float64
	WorkerUtilization map[int]float64
	
	// Percentiles
	OpDurationP50 time.Duration
	OpDurationP95 time.Duration
	OpDurationP99 time.Duration
	
	LockWaitP50 time.Duration
	LockWaitP95 time.Duration
	LockWaitP99 time.Duration
}

// String returns a human-readable representation of metrics
func (ms ParallelMetricsSnapshot) String() string {
	return fmt.Sprintf(
		"Operations: %d total (%d parallel, %d sequential, %d failed)\n"+
		"Duration: %v total (parallel: %v, sequential: %v)\n"+
		"Concurrency: %d max, %d current\n"+
		"Performance: %.2fx speedup, %.1f%% parallel ratio\n"+
		"Lock Stats: %v wait time, %d acquisitions, %d contentions\n"+
		"Op Duration: P50=%v, P95=%v, P99=%v\n"+
		"Lock Wait: P50=%v, P95=%v, P99=%v",
		ms.OpsTotal, ms.OpsParallel, ms.OpsSequential, ms.OpsFailed,
		ms.TotalDuration, ms.ParallelDuration, ms.SequentialDuration,
		ms.MaxConcurrency, ms.CurrentConcurrency,
		ms.Speedup, ms.ParallelRatio,
		ms.LockWaitTime, ms.LockAcquisitions, ms.LockContentions,
		ms.OpDurationP50, ms.OpDurationP95, ms.OpDurationP99,
		ms.LockWaitP50, ms.LockWaitP95, ms.LockWaitP99,
	)
}

// Percentiles holds percentile values
type Percentiles struct {
	P50 time.Duration
	P95 time.Duration
	P99 time.Duration
}

// calculatePercentiles calculates percentiles from duration slice
func calculatePercentiles(durations []time.Duration) Percentiles {
	if len(durations) == 0 {
		return Percentiles{}
	}
	
	// Sort durations
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	
	// Simple bubble sort for small datasets
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	p50Idx := len(sorted) * 50 / 100
	p95Idx := len(sorted) * 95 / 100
	p99Idx := len(sorted) * 99 / 100
	
	// Ensure indices are valid
	if p50Idx >= len(sorted) {
		p50Idx = len(sorted) - 1
	}
	if p95Idx >= len(sorted) {
		p95Idx = len(sorted) - 1
	}
	if p99Idx >= len(sorted) {
		p99Idx = len(sorted) - 1
	}
	
	return Percentiles{
		P50: sorted[p50Idx],
		P95: sorted[p95Idx],
		P99: sorted[p99Idx],
	}
}

// MetricsServer serves metrics over HTTP
type MetricsServer struct {
	collector *ParallelMetricsCollector
	server    *http.Server
	listener  net.Listener
}

// NewMetricsServer creates a new metrics server
func NewMetricsServer(collector *ParallelMetricsCollector, port int) *MetricsServer {
	mux := http.NewServeMux()
	
	ms := &MetricsServer{
		collector: collector,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
	
	// Register handlers
	mux.HandleFunc("/metrics", ms.handleMetrics)
	mux.HandleFunc("/metrics/json", ms.handleMetricsJSON)
	mux.HandleFunc("/health", ms.handleHealth)
	
	return ms
}

// Start starts the metrics server
func (ms *MetricsServer) Start() error {
	var err error
	ms.listener, err = net.Listen("tcp", ms.server.Addr)
	if err != nil {
		return err
	}
	
	// Update server address with actual port if using port 0
	ms.server.Addr = ms.listener.Addr().String()
	DEBUG("Starting metrics server on %s", ms.server.Addr)
	
	go func() {
		if err := ms.server.Serve(ms.listener); err != nil && err != http.ErrServerClosed {
			DEBUG("Metrics server error: %v", err)
		}
	}()
	return nil
}

// Stop stops the metrics server
func (ms *MetricsServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return ms.server.Shutdown(ctx)
}

// handleMetrics serves metrics in Prometheus format
func (ms *MetricsServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	snapshot := ms.collector.GetSnapshot()
	
	// Write Prometheus-style metrics
	fmt.Fprintf(w, "# HELP spruce_operations_total Total number of operations executed\n")
	fmt.Fprintf(w, "# TYPE spruce_operations_total counter\n")
	fmt.Fprintf(w, "spruce_operations_total{type=\"parallel\"} %d\n", snapshot.OpsParallel)
	fmt.Fprintf(w, "spruce_operations_total{type=\"sequential\"} %d\n", snapshot.OpsSequential)
	fmt.Fprintf(w, "spruce_operations_total{type=\"failed\"} %d\n", snapshot.OpsFailed)
	
	fmt.Fprintf(w, "\n# HELP spruce_operation_duration_seconds Operation execution duration\n")
	fmt.Fprintf(w, "# TYPE spruce_operation_duration_seconds summary\n")
	fmt.Fprintf(w, "spruce_operation_duration_seconds{quantile=\"0.5\"} %f\n", snapshot.OpDurationP50.Seconds())
	fmt.Fprintf(w, "spruce_operation_duration_seconds{quantile=\"0.95\"} %f\n", snapshot.OpDurationP95.Seconds())
	fmt.Fprintf(w, "spruce_operation_duration_seconds{quantile=\"0.99\"} %f\n", snapshot.OpDurationP99.Seconds())
	
	fmt.Fprintf(w, "\n# HELP spruce_concurrency Current and maximum concurrency\n")
	fmt.Fprintf(w, "# TYPE spruce_concurrency gauge\n")
	fmt.Fprintf(w, "spruce_concurrency{type=\"current\"} %d\n", snapshot.CurrentConcurrency)
	fmt.Fprintf(w, "spruce_concurrency{type=\"max\"} %d\n", snapshot.MaxConcurrency)
	
	fmt.Fprintf(w, "\n# HELP spruce_speedup Parallel execution speedup factor\n")
	fmt.Fprintf(w, "# TYPE spruce_speedup gauge\n")
	fmt.Fprintf(w, "spruce_speedup %f\n", snapshot.Speedup)
}

// handleMetricsJSON serves metrics in JSON format
func (ms *MetricsServer) handleMetricsJSON(w http.ResponseWriter, r *http.Request) {
	snapshot := ms.collector.GetSnapshot()
	
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"operations": {
			"total": %d,
			"parallel": %d,
			"sequential": %d,
			"failed": %d
		},
		"duration": {
			"total": "%v",
			"parallel": "%v",
			"sequential": "%v"
		},
		"concurrency": {
			"current": %d,
			"max": %d
		},
		"performance": {
			"speedup": %.2f,
			"parallel_ratio": %.1f,
			"failure_rate": %.1f
		},
		"percentiles": {
			"operation_duration": {
				"p50": "%v",
				"p95": "%v",
				"p99": "%v"
			},
			"lock_wait": {
				"p50": "%v",
				"p95": "%v",
				"p99": "%v"
			}
		}
	}`,
		snapshot.OpsTotal, snapshot.OpsParallel, snapshot.OpsSequential, snapshot.OpsFailed,
		snapshot.TotalDuration, snapshot.ParallelDuration, snapshot.SequentialDuration,
		snapshot.CurrentConcurrency, snapshot.MaxConcurrency,
		snapshot.Speedup, snapshot.ParallelRatio, snapshot.FailureRate,
		snapshot.OpDurationP50, snapshot.OpDurationP95, snapshot.OpDurationP99,
		snapshot.LockWaitP50, snapshot.LockWaitP95, snapshot.LockWaitP99,
	)
}

// handleHealth serves health check endpoint
func (ms *MetricsServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK\n")
}