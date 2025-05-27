package internal

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// SlowOperationDetector detects and reports slow operations
type SlowOperationDetector struct {
	mu              sync.RWMutex
	thresholds      map[string]time.Duration
	defaultThreshold time.Duration
	handlers        []SlowOperationHandler
	enabled         bool
	history         *SlowOperationHistory
}

// SlowOperationHandler handles slow operation events
type SlowOperationHandler func(op *SlowOperation)

// SlowOperation represents a slow operation
type SlowOperation struct {
	Name      string                 `json:"name"`
	Duration  time.Duration          `json:"duration"`
	Threshold time.Duration          `json:"threshold"`
	Start     time.Time              `json:"start"`
	End       time.Time              `json:"end"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Stack     string                 `json:"stack,omitempty"`
}

// SlowOperationHistory tracks slow operations
type SlowOperationHistory struct {
	mu         sync.RWMutex
	operations []*SlowOperation
	maxSize    int
}

// NewSlowOperationDetector creates a new slow operation detector
func NewSlowOperationDetector(defaultThreshold time.Duration) *SlowOperationDetector {
	return &SlowOperationDetector{
		thresholds:       make(map[string]time.Duration),
		defaultThreshold: defaultThreshold,
		enabled:          true,
		history:          NewSlowOperationHistory(1000),
		handlers: []SlowOperationHandler{
			defaultSlowOperationLogger,
		},
	}
}

// NewSlowOperationHistory creates a new history tracker
func NewSlowOperationHistory(maxSize int) *SlowOperationHistory {
	return &SlowOperationHistory{
		operations: make([]*SlowOperation, 0, maxSize),
		maxSize:    maxSize,
	}
}

// SetThreshold sets threshold for a specific operation
func (sod *SlowOperationDetector) SetThreshold(operation string, threshold time.Duration) {
	sod.mu.Lock()
	defer sod.mu.Unlock()
	sod.thresholds[operation] = threshold
}

// GetThreshold gets threshold for an operation
func (sod *SlowOperationDetector) GetThreshold(operation string) time.Duration {
	sod.mu.RLock()
	defer sod.mu.RUnlock()
	
	if threshold, exists := sod.thresholds[operation]; exists {
		return threshold
	}
	return sod.defaultThreshold
}

// AddHandler adds a slow operation handler
func (sod *SlowOperationDetector) AddHandler(handler SlowOperationHandler) {
	sod.mu.Lock()
	defer sod.mu.Unlock()
	sod.handlers = append(sod.handlers, handler)
}

// SetEnabled enables or disables detection
func (sod *SlowOperationDetector) SetEnabled(enabled bool) {
	sod.mu.Lock()
	defer sod.mu.Unlock()
	sod.enabled = enabled
}

// Check checks if an operation is slow
func (sod *SlowOperationDetector) Check(timer *Timer) {
	sod.mu.RLock()
	if !sod.enabled {
		sod.mu.RUnlock()
		return
	}
	sod.mu.RUnlock()
	
	duration := timer.Duration()
	threshold := sod.GetThreshold(timer.Name())
	
	if duration > threshold {
		slowOp := &SlowOperation{
			Name:      timer.Name(),
			Duration:  duration,
			Threshold: threshold,
			Start:     timer.start,
			End:       timer.end,
			Metadata:  timer.copyMetadata(),
		}
		
		// Add to history
		sod.history.Add(slowOp)
		
		// Call handlers
		sod.mu.RLock()
		handlers := append([]SlowOperationHandler{}, sod.handlers...)
		sod.mu.RUnlock()
		
		for _, handler := range handlers {
			handler(slowOp)
		}
	}
}

// GetHistory returns slow operation history
func (sod *SlowOperationDetector) GetHistory(limit int) []*SlowOperation {
	return sod.history.GetRecent(limit)
}

// ClearHistory clears the history
func (sod *SlowOperationDetector) ClearHistory() {
	sod.history.Clear()
}

// Add adds a slow operation to history
func (soh *SlowOperationHistory) Add(op *SlowOperation) {
	soh.mu.Lock()
	defer soh.mu.Unlock()
	
	soh.operations = append(soh.operations, op)
	if len(soh.operations) > soh.maxSize {
		// Remove oldest
		soh.operations = soh.operations[len(soh.operations)-soh.maxSize:]
	}
}

// GetRecent returns recent slow operations
func (soh *SlowOperationHistory) GetRecent(limit int) []*SlowOperation {
	soh.mu.RLock()
	defer soh.mu.RUnlock()
	
	if limit <= 0 || limit > len(soh.operations) {
		limit = len(soh.operations)
	}
	
	start := len(soh.operations) - limit
	if start < 0 {
		start = 0
	}
	
	result := make([]*SlowOperation, limit)
	copy(result, soh.operations[start:])
	return result
}

// Clear clears the history
func (soh *SlowOperationHistory) Clear() {
	soh.mu.Lock()
	defer soh.mu.Unlock()
	soh.operations = soh.operations[:0]
}

// GetStats returns statistics about slow operations
func (soh *SlowOperationHistory) GetStats() *SlowOperationStats {
	soh.mu.RLock()
	defer soh.mu.RUnlock()
	
	stats := &SlowOperationStats{
		ByOperation: make(map[string]*SlowOpTypeStats),
	}
	
	for _, op := range soh.operations {
		stats.Total++
		
		typeStats, exists := stats.ByOperation[op.Name]
		if !exists {
			typeStats = &SlowOpTypeStats{
				Operation: op.Name,
				MinTime:   op.Duration,
				MaxTime:   op.Duration,
			}
			stats.ByOperation[op.Name] = typeStats
		}
		
		typeStats.Count++
		typeStats.TotalTime += op.Duration
		if op.Duration < typeStats.MinTime {
			typeStats.MinTime = op.Duration
		}
		if op.Duration > typeStats.MaxTime {
			typeStats.MaxTime = op.Duration
		}
	}
	
	// Calculate averages
	for _, typeStats := range stats.ByOperation {
		if typeStats.Count > 0 {
			typeStats.AvgTime = typeStats.TotalTime / time.Duration(typeStats.Count)
		}
	}
	
	return stats
}

// SlowOperationStats contains statistics about slow operations
type SlowOperationStats struct {
	Total       int                          `json:"total"`
	ByOperation map[string]*SlowOpTypeStats  `json:"by_operation"`
}

// SlowOpTypeStats contains stats for a specific operation type
type SlowOpTypeStats struct {
	Operation string        `json:"operation"`
	Count     int           `json:"count"`
	TotalTime time.Duration `json:"total_time"`
	MinTime   time.Duration `json:"min_time"`
	MaxTime   time.Duration `json:"max_time"`
	AvgTime   time.Duration `json:"avg_time"`
}

// Default handlers

// defaultSlowOperationLogger logs slow operations
func defaultSlowOperationLogger(op *SlowOperation) {
	extra := ""
	if err, ok := op.Metadata["error"]; ok {
		extra = fmt.Sprintf(" (error: %v)", err)
	}
	
	log.Printf("[SLOW] Operation '%s' took %v (threshold: %v)%s",
		op.Name, op.Duration, op.Threshold, extra)
}

// Global slow operation detector
var slowOpDetector *SlowOperationDetector
var slowOpOnce sync.Once

// InitializeSlowOpDetector initializes the global slow operation detector
func InitializeSlowOpDetector(config *PerformanceConfig) {
	slowOpOnce.Do(func() {
		threshold := time.Duration(100) * time.Millisecond
		if config != nil && config.Performance.Monitoring.SlowOperationThresholdMs > 0 {
			threshold = time.Duration(config.Performance.Monitoring.SlowOperationThresholdMs) * time.Millisecond
		}
		
		slowOpDetector = NewSlowOperationDetector(threshold)
		
		// Set specific thresholds
		slowOpDetector.SetThreshold("parse", 50*time.Millisecond)
		slowOpDetector.SetThreshold("eval", 100*time.Millisecond)
		slowOpDetector.SetThreshold("vault", 1*time.Second)
		slowOpDetector.SetThreshold("file", 200*time.Millisecond)
		slowOpDetector.SetThreshold("awsparam", 1*time.Second)
		slowOpDetector.SetThreshold("awssecret", 1*time.Second)
		
		// Add metrics handler
		if MetricsEnabled() {
			slowOpDetector.AddHandler(func(op *SlowOperation) {
				mc := GetMetricsCollector()
				labels := map[string]string{
					"operation": op.Name,
					"slow":      "true",
				}
				
				// Register slow operation metric if not exists
				metricName := "graft_slow_operations_total"
				if _, exists := mc.CustomMetrics[metricName]; !exists {
					mc.RegisterCustomMetric(metricName, "Total number of slow operations", MetricTypeCounter)
				}
				
				if family, ok := mc.CustomMetrics[metricName]; ok {
					family.GetOrCreate(labels).(*Counter).Inc()
				}
			})
		}
	})
}

// GetSlowOpDetector returns the global slow operation detector
func GetSlowOpDetector() *SlowOperationDetector {
	if slowOpDetector == nil {
		InitializeSlowOpDetector(nil)
	}
	return slowOpDetector
}

// ReportSlowOperation reports a slow operation manually
func ReportSlowOperation(name string, duration time.Duration, metadata map[string]interface{}) {
	detector := GetSlowOpDetector()
	if detector == nil || !detector.enabled {
		return
	}
	
	threshold := detector.GetThreshold(name)
	if duration > threshold {
		slowOp := &SlowOperation{
			Name:      name,
			Duration:  duration,
			Threshold: threshold,
			Start:     time.Now().Add(-duration),
			End:       time.Now(),
			Metadata:  metadata,
		}
		
		detector.history.Add(slowOp)
		
		for _, handler := range detector.handlers {
			handler(slowOp)
		}
	}
}