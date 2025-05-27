package graft

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// TimingAggregator aggregates timing statistics
type TimingAggregator struct {
	mu         sync.RWMutex
	operations map[string]*OperationStats
	window     time.Duration
	maxSamples int
}

// OperationStats tracks statistics for an operation type
type OperationStats struct {
	Name         string
	Count        int64
	TotalTime    time.Duration
	MinTime      time.Duration
	MaxTime      time.Duration
	samples      []time.Duration
	lastUpdated  time.Time
	mu           sync.RWMutex
}

// NewTimingAggregator creates a new timing aggregator
func NewTimingAggregator(window time.Duration, maxSamples int) *TimingAggregator {
	return &TimingAggregator{
		operations: make(map[string]*OperationStats),
		window:     window,
		maxSamples: maxSamples,
	}
}

// Record records a timing sample
func (ta *TimingAggregator) Record(name string, duration time.Duration) {
	ta.mu.Lock()
	stats, exists := ta.operations[name]
	if !exists {
		stats = &OperationStats{
			Name:    name,
			MinTime: duration,
			MaxTime: duration,
			samples: make([]time.Duration, 0, ta.maxSamples),
		}
		ta.operations[name] = stats
	}
	ta.mu.Unlock()
	
	stats.record(duration, ta.maxSamples)
}

// record adds a sample to operation stats
func (os *OperationStats) record(duration time.Duration, maxSamples int) {
	os.mu.Lock()
	defer os.mu.Unlock()
	
	os.Count++
	os.TotalTime += duration
	os.lastUpdated = time.Now()
	
	if duration < os.MinTime {
		os.MinTime = duration
	}
	if duration > os.MaxTime {
		os.MaxTime = duration
	}
	
	os.samples = append(os.samples, duration)
	if len(os.samples) > maxSamples {
		// Keep only recent samples
		os.samples = os.samples[len(os.samples)-maxSamples:]
	}
}

// GetStats returns current statistics
func (os *OperationStats) GetStats() TimingStatistics {
	os.mu.RLock()
	defer os.mu.RUnlock()
	
	if os.Count == 0 {
		return TimingStatistics{Name: os.Name}
	}
	
	stats := TimingStatistics{
		Name:      os.Name,
		Count:     os.Count,
		TotalTime: os.TotalTime,
		MinTime:   os.MinTime,
		MaxTime:   os.MaxTime,
		MeanTime:  os.TotalTime / time.Duration(os.Count),
	}
	
	// Calculate percentiles if we have samples
	if len(os.samples) > 0 {
		sorted := make([]time.Duration, len(os.samples))
		copy(sorted, os.samples)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i] < sorted[j]
		})
		
		stats.P50 = percentileDuration(sorted, 0.50)
		stats.P90 = percentileDuration(sorted, 0.90)
		stats.P95 = percentileDuration(sorted, 0.95)
		stats.P99 = percentileDuration(sorted, 0.99)
		
		// Calculate standard deviation
		var sumSquares float64
		meanNanos := float64(stats.MeanTime.Nanoseconds())
		for _, d := range os.samples {
			diff := float64(d.Nanoseconds()) - meanNanos
			sumSquares += diff * diff
		}
		varianceNanos := sumSquares / float64(len(os.samples))
		stats.StdDev = time.Duration(math.Sqrt(varianceNanos))
	}
	
	return stats
}

// GetAllStats returns statistics for all operations
func (ta *TimingAggregator) GetAllStats() []TimingStatistics {
	ta.mu.RLock()
	defer ta.mu.RUnlock()
	
	stats := make([]TimingStatistics, 0, len(ta.operations))
	for _, op := range ta.operations {
		stats = append(stats, op.GetStats())
	}
	
	// Sort by total time descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].TotalTime > stats[j].TotalTime
	})
	
	return stats
}

// GetStats returns statistics for a specific operation
func (ta *TimingAggregator) GetStats(name string) (TimingStatistics, bool) {
	ta.mu.RLock()
	defer ta.mu.RUnlock()
	
	if stats, exists := ta.operations[name]; exists {
		return stats.GetStats(), true
	}
	return TimingStatistics{}, false
}

// GetTopOperations returns the top N operations by total time
func (ta *TimingAggregator) GetTopOperations(n int) []TimingStatistics {
	allStats := ta.GetAllStats()
	if n > len(allStats) {
		n = len(allStats)
	}
	return allStats[:n]
}

// Clean removes old entries outside the window
func (ta *TimingAggregator) Clean() {
	if ta.window <= 0 {
		return // No cleaning if window is not set
	}
	
	ta.mu.Lock()
	defer ta.mu.Unlock()
	
	cutoff := time.Now().Add(-ta.window)
	for name, stats := range ta.operations {
		stats.mu.RLock()
		if stats.lastUpdated.Before(cutoff) {
			stats.mu.RUnlock()
			delete(ta.operations, name)
		} else {
			stats.mu.RUnlock()
		}
	}
}

// Reset resets all statistics
func (ta *TimingAggregator) Reset() {
	ta.mu.Lock()
	defer ta.mu.Unlock()
	ta.operations = make(map[string]*OperationStats)
}

// TimingStatistics represents timing statistics for an operation
type TimingStatistics struct {
	Name      string        `json:"name"`
	Count     int64         `json:"count"`
	TotalTime time.Duration `json:"total_time"`
	MinTime   time.Duration `json:"min_time"`
	MaxTime   time.Duration `json:"max_time"`
	MeanTime  time.Duration `json:"mean_time"`
	StdDev    time.Duration `json:"std_dev"`
	P50       time.Duration `json:"p50"`
	P90       time.Duration `json:"p90"`
	P95       time.Duration `json:"p95"`
	P99       time.Duration `json:"p99"`
}

// String returns a formatted string representation
func (ts TimingStatistics) String() string {
	return fmt.Sprintf(
		"%s: count=%d, total=%v, mean=%v, min=%v, max=%v, p50=%v, p95=%v, p99=%v",
		ts.Name, ts.Count, ts.TotalTime, ts.MeanTime,
		ts.MinTime, ts.MaxTime, ts.P50, ts.P95, ts.P99,
	)
}

// percentileDuration calculates percentile for duration slice
func percentileDuration(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	
	index := int(float64(len(sorted)-1) * p)
	return sorted[index]
}

// TimingSummary provides a summary of all timings
type TimingSummary struct {
	TotalOperations int64                `json:"total_operations"`
	TotalTime       time.Duration        `json:"total_time"`
	Operations      []TimingStatistics   `json:"operations"`
	ByCategory      map[string][]TimingStatistics `json:"by_category"`
}

// GetSummary returns a comprehensive timing summary
func (ta *TimingAggregator) GetSummary() *TimingSummary {
	allStats := ta.GetAllStats()
	
	summary := &TimingSummary{
		Operations: allStats,
		ByCategory: make(map[string][]TimingStatistics),
	}
	
	// Calculate totals and categorize
	for _, stats := range allStats {
		summary.TotalOperations += stats.Count
		summary.TotalTime += stats.TotalTime
		
		// Categorize by prefix
		category := "other"
		if len(stats.Name) > 0 {
			for _, prefix := range []string{"parse", "eval", "operator", "cache", "io"} {
				if stats.Name[:min(len(prefix), len(stats.Name))] == prefix {
					category = prefix
					break
				}
			}
		}
		
		summary.ByCategory[category] = append(summary.ByCategory[category], stats)
	}
	
	return summary
}

// Global timing aggregator
var globalTimingAggregator *TimingAggregator
var timingAggregatorOnce sync.Once

// InitializeTimingAggregator initializes the global timing aggregator
func InitializeTimingAggregator() {
	timingAggregatorOnce.Do(func() {
		globalTimingAggregator = NewTimingAggregator(1*time.Hour, 1000)
	})
}

// GetTimingAggregator returns the global timing aggregator
func GetTimingAggregator() *TimingAggregator {
	if globalTimingAggregator == nil {
		InitializeTimingAggregator()
	}
	return globalTimingAggregator
}

// RecordTiming records a timing to the global aggregator
func RecordTiming(name string, duration time.Duration) {
	if aggregator := GetTimingAggregator(); aggregator != nil {
		aggregator.Record(name, duration)
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}