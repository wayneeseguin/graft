package internal

import (
	"math"
	"sync"
	"testing"
	"time"
)

// TestTimingAggregator tests timing aggregation functionality
func TestTimingAggregator(t *testing.T) {
	t.Run("basic aggregation", func(t *testing.T) {
		ta := NewTimingAggregator(1*time.Hour, 100)
		
		// Record some samples
		ta.Record("operation1", 10*time.Millisecond)
		ta.Record("operation1", 20*time.Millisecond)
		ta.Record("operation1", 30*time.Millisecond)
		
		stats, exists := ta.GetStats("operation1")
		if !exists {
			t.Fatal("operation1 stats should exist")
		}

		if stats.Name != "operation1" {
			t.Errorf("expected name 'operation1', got %s", stats.Name)
		}

		if stats.Count != 3 {
			t.Errorf("expected count 3, got %d", stats.Count)
		}

		expectedTotal := 60 * time.Millisecond
		if stats.TotalTime != expectedTotal {
			t.Errorf("expected total time %v, got %v", expectedTotal, stats.TotalTime)
		}

		expectedMean := 20 * time.Millisecond
		if stats.MeanTime != expectedMean {
			t.Errorf("expected mean time %v, got %v", expectedMean, stats.MeanTime)
		}

		if stats.MinTime != 10*time.Millisecond {
			t.Errorf("expected min time 10ms, got %v", stats.MinTime)
		}

		if stats.MaxTime != 30*time.Millisecond {
			t.Errorf("expected max time 30ms, got %v", stats.MaxTime)
		}
	})

	t.Run("percentile calculations", func(t *testing.T) {
		ta := NewTimingAggregator(1*time.Hour, 100)
		
		// Record samples for percentile testing (1ms to 100ms)
		for i := 1; i <= 100; i++ {
			ta.Record("percentile_test", time.Duration(i)*time.Millisecond)
		}

		stats, exists := ta.GetStats("percentile_test")
		if !exists {
			t.Fatal("percentile_test stats should exist")
		}

		// P50 should be around 50ms
		if absInt64(int64(stats.P50-50*time.Millisecond)) > int64(5*time.Millisecond) {
			t.Errorf("expected P50 ~50ms, got %v", stats.P50)
		}

		// P90 should be around 90ms
		if absInt64(int64(stats.P90-90*time.Millisecond)) > int64(5*time.Millisecond) {
			t.Errorf("expected P90 ~90ms, got %v", stats.P90)
		}

		// P95 should be around 95ms
		if absInt64(int64(stats.P95-95*time.Millisecond)) > int64(5*time.Millisecond) {
			t.Errorf("expected P95 ~95ms, got %v", stats.P95)
		}

		// P99 should be around 99ms
		if absInt64(int64(stats.P99-99*time.Millisecond)) > int64(5*time.Millisecond) {
			t.Errorf("expected P99 ~99ms, got %v", stats.P99)
		}
	})

	t.Run("standard deviation calculation", func(t *testing.T) {
		ta := NewTimingAggregator(1*time.Hour, 100)
		
		// Record samples with known std dev
		// Values: 10, 20, 30, 40, 50 (mean = 30, population std dev = √200 ≈ 14.14)
		ta.Record("stddev_test", 10*time.Millisecond)
		ta.Record("stddev_test", 20*time.Millisecond)
		ta.Record("stddev_test", 30*time.Millisecond)
		ta.Record("stddev_test", 40*time.Millisecond)
		ta.Record("stddev_test", 50*time.Millisecond)

		stats, exists := ta.GetStats("stddev_test")
		if !exists {
			t.Fatal("stddev_test stats should exist")
		}

		expectedStdDev := 14.14 * float64(time.Millisecond)
		actualStdDev := float64(stats.StdDev)
		
		// Allow 10% tolerance
		tolerance := expectedStdDev * 0.1
		if math.Abs(actualStdDev-expectedStdDev) > tolerance {
			t.Errorf("expected std dev ~%.2f, got %.2f", expectedStdDev, actualStdDev)
		}
	})

	t.Run("multiple operations", func(t *testing.T) {
		ta := NewTimingAggregator(1*time.Hour, 100)
		
		// Record different operations
		ta.Record("parse", 5*time.Millisecond)
		ta.Record("parse", 10*time.Millisecond)
		ta.Record("eval", 15*time.Millisecond)
		ta.Record("eval", 25*time.Millisecond)
		ta.Record("cache", 1*time.Millisecond)

		allStats := ta.GetAllStats()
		if len(allStats) != 3 {
			t.Errorf("expected 3 operations, got %d", len(allStats))
		}

		// Should be sorted by total time descending
		// eval: 40ms total, parse: 15ms total, cache: 1ms total
		if allStats[0].Name != "eval" {
			t.Errorf("expected first operation to be 'eval', got %s", allStats[0].Name)
		}

		if allStats[1].Name != "parse" {
			t.Errorf("expected second operation to be 'parse', got %s", allStats[1].Name)
		}

		if allStats[2].Name != "cache" {
			t.Errorf("expected third operation to be 'cache', got %s", allStats[2].Name)
		}
	})

	t.Run("top operations", func(t *testing.T) {
		ta := NewTimingAggregator(1*time.Hour, 100)
		
		// Record many operations
		for i := 0; i < 10; i++ {
			ta.Record("op1", time.Duration(i+1)*time.Millisecond)
			ta.Record("op2", time.Duration(i+2)*time.Millisecond)
			ta.Record("op3", time.Duration(i+3)*time.Millisecond)
		}

		top2 := ta.GetTopOperations(2)
		if len(top2) != 2 {
			t.Errorf("expected 2 top operations, got %d", len(top2))
		}

		// op3 should have highest total time
		if top2[0].Name != "op3" {
			t.Errorf("expected top operation to be 'op3', got %s", top2[0].Name)
		}

		// Test requesting more than available
		top10 := ta.GetTopOperations(10)
		if len(top10) != 3 {
			t.Errorf("expected 3 operations (all available), got %d", len(top10))
		}
	})

	t.Run("sample limit", func(t *testing.T) {
		ta := NewTimingAggregator(1*time.Hour, 5) // limit to 5 samples
		
		// Record more samples than limit
		for i := 1; i <= 10; i++ {
			ta.Record("limited", time.Duration(i)*time.Millisecond)
		}

		stats, exists := ta.GetStats("limited")
		if !exists {
			t.Fatal("limited stats should exist")
		}

		// Count should be 10 (all recordings)
		if stats.Count != 10 {
			t.Errorf("expected count 10, got %d", stats.Count)
		}

		// But percentiles should be calculated from only recent samples (6-10ms)
		// P50 should be around 8ms
		if stats.P50 < 7*time.Millisecond || stats.P50 > 9*time.Millisecond {
			t.Errorf("expected P50 around 8ms with sample limit, got %v", stats.P50)
		}
	})

	t.Run("reset functionality", func(t *testing.T) {
		ta := NewTimingAggregator(1*time.Hour, 100)
		
		ta.Record("operation", 10*time.Millisecond)
		ta.Record("operation", 20*time.Millisecond)

		// Verify stats exist
		_, exists := ta.GetStats("operation")
		if !exists {
			t.Fatal("operation stats should exist before reset")
		}

		// Reset and verify cleared
		ta.Reset()
		_, exists = ta.GetStats("operation")
		if exists {
			t.Error("operation stats should not exist after reset")
		}

		allStats := ta.GetAllStats()
		if len(allStats) != 0 {
			t.Errorf("expected 0 operations after reset, got %d", len(allStats))
		}
	})
}

// TestTimingAggregatorConcurrency tests concurrent access
func TestTimingAggregatorConcurrency(t *testing.T) {
	t.Run("concurrent recording", func(t *testing.T) {
		ta := NewTimingAggregator(1*time.Hour, 1000)
		var wg sync.WaitGroup
		
		goroutines := 50
		recordsPerGoroutine := 100

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < recordsPerGoroutine; j++ {
					ta.Record("concurrent", time.Duration(j+1)*time.Microsecond)
				}
			}(i)
		}

		wg.Wait()

		stats, exists := ta.GetStats("concurrent")
		if !exists {
			t.Fatal("concurrent stats should exist")
		}

		expectedCount := int64(goroutines * recordsPerGoroutine)
		if stats.Count != expectedCount {
			t.Errorf("expected count %d, got %d", expectedCount, stats.Count)
		}
	})

	t.Run("concurrent stats retrieval", func(t *testing.T) {
		ta := NewTimingAggregator(1*time.Hour, 100)
		
		// Prime with some data
		for i := 0; i < 10; i++ {
			ta.Record("test_op", time.Duration(i+1)*time.Millisecond)
		}

		var wg sync.WaitGroup
		errors := make(chan error, 50)

		// Multiple goroutines reading stats
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				
				// Try various read operations
				_, exists := ta.GetStats("test_op")
				if !exists {
					errors <- &testError{"stats should exist"}
					return
				}

				allStats := ta.GetAllStats()
				if len(allStats) == 0 {
					errors <- &testError{"should have stats"}
					return
				}

				topOps := ta.GetTopOperations(5)
				if len(topOps) == 0 {
					errors <- &testError{"should have top operations"}
					return
				}
			}()
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Error(err)
		}
	})

	t.Run("concurrent recording different operations", func(t *testing.T) {
		ta := NewTimingAggregator(1*time.Hour, 100)
		var wg sync.WaitGroup
		
		operations := []string{"parse", "eval", "cache", "io", "network"}
		
		for _, op := range operations {
			wg.Add(1)
			go func(operation string) {
				defer wg.Done()
				for i := 0; i < 20; i++ {
					ta.Record(operation, time.Duration(i+1)*time.Millisecond)
				}
			}(op)
		}

		wg.Wait()

		allStats := ta.GetAllStats()
		if len(allStats) != len(operations) {
			t.Errorf("expected %d operations, got %d", len(operations), len(allStats))
		}

		// Verify each operation has correct count
		for _, stats := range allStats {
			if stats.Count != 20 {
				t.Errorf("operation %s: expected count 20, got %d", stats.Name, stats.Count)
			}
		}
	})
}

// TestTimingAggregatorSummary tests summary functionality
func TestTimingAggregatorSummary(t *testing.T) {
	t.Run("basic summary", func(t *testing.T) {
		ta := NewTimingAggregator(1*time.Hour, 100)
		
		// Record operations with known totals
		ta.Record("parse", 10*time.Millisecond)
		ta.Record("parse", 20*time.Millisecond) // total: 30ms, count: 2
		ta.Record("eval", 15*time.Millisecond)
		ta.Record("eval", 25*time.Millisecond)  // total: 40ms, count: 2
		ta.Record("cache", 5*time.Millisecond)  // total: 5ms, count: 1

		summary := ta.GetSummary()
		
		if summary.TotalOperations != 5 {
			t.Errorf("expected total operations 5, got %d", summary.TotalOperations)
		}

		expectedTotalTime := 75 * time.Millisecond
		if summary.TotalTime != expectedTotalTime {
			t.Errorf("expected total time %v, got %v", expectedTotalTime, summary.TotalTime)
		}

		if len(summary.Operations) != 3 {
			t.Errorf("expected 3 operations in summary, got %d", len(summary.Operations))
		}
	})

	t.Run("categorization", func(t *testing.T) {
		ta := NewTimingAggregator(1*time.Hour, 100)
		
		// Record operations that should be categorized
		ta.Record("parse_yaml", 10*time.Millisecond)
		ta.Record("parse_json", 15*time.Millisecond)
		ta.Record("eval_expression", 20*time.Millisecond)
		ta.Record("eval_operator", 25*time.Millisecond)
		ta.Record("cache_get", 5*time.Millisecond)
		ta.Record("io_read", 30*time.Millisecond)
		ta.Record("unknown_operation", 10*time.Millisecond)

		summary := ta.GetSummary()
		
		// Check categories exist
		if _, exists := summary.ByCategory["parse"]; !exists {
			t.Error("parse category should exist")
		}
		if _, exists := summary.ByCategory["eval"]; !exists {
			t.Error("eval category should exist")
		}
		if _, exists := summary.ByCategory["cache"]; !exists {
			t.Error("cache category should exist")
		}
		if _, exists := summary.ByCategory["io"]; !exists {
			t.Error("io category should exist")
		}
		if _, exists := summary.ByCategory["other"]; !exists {
			t.Error("other category should exist")
		}

		// Check category contents
		parseOps := summary.ByCategory["parse"]
		if len(parseOps) != 2 {
			t.Errorf("expected 2 parse operations, got %d", len(parseOps))
		}

		evalOps := summary.ByCategory["eval"]
		if len(evalOps) != 2 {
			t.Errorf("expected 2 eval operations, got %d", len(evalOps))
		}

		otherOps := summary.ByCategory["other"]
		if len(otherOps) != 1 {
			t.Errorf("expected 1 other operation, got %d", len(otherOps))
		}
	})
}

// TestTimingAggregatorCleaning tests cleanup functionality
func TestTimingAggregatorCleaning(t *testing.T) {
	t.Run("window-based cleaning", func(t *testing.T) {
		// Short window for testing
		ta := NewTimingAggregator(50*time.Millisecond, 100)
		
		// Record operation
		ta.Record("old_operation", 10*time.Millisecond)
		
		// Verify it exists
		_, exists := ta.GetStats("old_operation")
		if !exists {
			t.Fatal("operation should exist initially")
		}

		// Wait for window to expire
		time.Sleep(60 * time.Millisecond)
		
		// Clean and verify removed
		ta.Clean()
		_, exists = ta.GetStats("old_operation")
		if exists {
			t.Error("operation should be cleaned after window expires")
		}
	})

	t.Run("recent operations preserved", func(t *testing.T) {
		ta := NewTimingAggregator(100*time.Millisecond, 100)
		
		// Record old operation
		ta.Record("old_operation", 10*time.Millisecond)
		
		// Wait a bit but not past window
		time.Sleep(20 * time.Millisecond)
		
		// Record recent operation
		ta.Record("recent_operation", 15*time.Millisecond)
		
		// Wait for old operation to expire
		time.Sleep(90 * time.Millisecond)
		
		ta.Clean()
		
		// Old should be gone, recent should remain
		_, oldExists := ta.GetStats("old_operation")
		_, recentExists := ta.GetStats("recent_operation")
		
		if oldExists {
			t.Error("old operation should be cleaned")
		}
		if !recentExists {
			t.Error("recent operation should be preserved")
		}
	})

	t.Run("no cleaning with zero window", func(t *testing.T) {
		ta := NewTimingAggregator(0, 100) // zero window = no cleaning
		
		ta.Record("persistent", 10*time.Millisecond)
		
		// Wait and clean
		time.Sleep(50 * time.Millisecond)
		ta.Clean()
		
		// Should still exist
		_, exists := ta.GetStats("persistent")
		if !exists {
			t.Error("operation should persist with zero window")
		}
	})
}

// TestGlobalTimingAggregator tests global aggregator functionality
func TestGlobalTimingAggregator(t *testing.T) {
	t.Run("global aggregator initialization", func(t *testing.T) {
		// Reset global state for testing
		globalTimingAggregator = nil
		timingAggregatorOnce = sync.Once{}
		
		// First call should initialize
		ta1 := GetTimingAggregator()
		if ta1 == nil {
			t.Fatal("global timing aggregator should be initialized")
		}

		// Second call should return same instance
		ta2 := GetTimingAggregator()
		if ta1 != ta2 {
			t.Error("should return same global instance")
		}
	})

	t.Run("global recording", func(t *testing.T) {
		// Record through global function
		RecordTiming("global_test", 25*time.Millisecond)
		
		ta := GetTimingAggregator()
		stats, exists := ta.GetStats("global_test")
		if !exists {
			t.Fatal("global_test stats should exist")
		}

		if stats.Count != 1 {
			t.Errorf("expected count 1, got %d", stats.Count)
		}

		if stats.TotalTime != 25*time.Millisecond {
			t.Errorf("expected total time 25ms, got %v", stats.TotalTime)
		}
	})
}

// TestTimingStatistics tests statistics formatting and calculations
func TestTimingStatistics(t *testing.T) {
	t.Run("string formatting", func(t *testing.T) {
		stats := TimingStatistics{
			Name:      "test_op",
			Count:     100,
			TotalTime: 500 * time.Millisecond,
			MinTime:   1 * time.Millisecond,
			MaxTime:   10 * time.Millisecond,
			MeanTime:  5 * time.Millisecond,
			P50:       4 * time.Millisecond,
			P95:       9 * time.Millisecond,
			P99:       10 * time.Millisecond,
		}

		str := stats.String()
		
		// Should contain key information
		if !contains(str, "test_op") {
			t.Error("string should contain operation name")
		}
		if !contains(str, "count=100") {
			t.Error("string should contain count")
		}
		if !contains(str, "500ms") {
			t.Error("string should contain total time")
		}
	})

	t.Run("zero values", func(t *testing.T) {
		stats := TimingStatistics{Name: "empty"}
		str := stats.String()
		
		// Should not panic with zero values
		if !contains(str, "empty") {
			t.Error("string should contain name even with zero values")
		}
	})
}

// Helper functions
func absInt64(a int64) int64 {
	if a < 0 {
		return -a
	}
	return a
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr ||
		   len(s) > len(substr) && contains(s[1:], substr)
}