package internal

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestMetricsCollector tests the main metrics collector functionality
func TestMetricsCollector(t *testing.T) {
	t.Run("basic collector creation", func(t *testing.T) {
		mc := NewMetricsCollector()

		// Verify all metric families are created
		if mc.ParseOperations == nil {
			t.Error("ParseOperations should be initialized")
		}
		if mc.EvalOperations == nil {
			t.Error("EvalOperations should be initialized")
		}
		if mc.CacheHits == nil {
			t.Error("CacheHits should be initialized")
		}
		if mc.CustomMetrics == nil {
			t.Error("CustomMetrics should be initialized")
		}

		// Test metric family properties
		if mc.ParseOperations.Type != MetricTypeCounter {
			t.Errorf("expected ParseOperations to be counter, got %s", mc.ParseOperations.Type)
		}

		if mc.ParseDuration.Type != MetricTypeHistogram {
			t.Errorf("expected ParseDuration to be histogram, got %s", mc.ParseDuration.Type)
		}
	})

	t.Run("parse operation recording", func(t *testing.T) {
		mc := NewMetricsCollector()
		start := time.Now()
		time.Sleep(1 * time.Millisecond)

		// Record successful parse
		mc.RecordParseOperation(start, nil)

		parseOps := mc.ParseOperations.GetOrCreate(nil).(*Counter)
		if parseOps.Get() != 1 {
			t.Errorf("expected 1 parse operation, got %d", parseOps.Get())
		}

		parseDuration := mc.ParseDuration.GetOrCreate(nil).(*Histogram)
		stats := parseDuration.GetStats()
		if stats.Count != 1 {
			t.Errorf("expected 1 duration observation, got %d", stats.Count)
		}
		if stats.Mean <= 0 {
			t.Errorf("expected positive duration, got %f", stats.Mean)
		}

		// Record parse with error
		testErr := &testError{"parse error"}
		mc.RecordParseOperation(start, testErr)

		if parseOps.Get() != 2 {
			t.Errorf("expected 2 parse operations after error, got %d", parseOps.Get())
		}

		parseErrors := mc.ParseErrors.GetOrCreate(nil).(*Counter)
		if parseErrors.Get() != 1 {
			t.Errorf("expected 1 parse error, got %d", parseErrors.Get())
		}

		errorsByType := mc.ErrorsByType.GetOrCreate(map[string]string{"type": "parse"}).(*Counter)
		if errorsByType.Get() != 1 {
			t.Errorf("expected 1 error by type, got %d", errorsByType.Get())
		}
	})

	t.Run("evaluation operation recording", func(t *testing.T) {
		mc := NewMetricsCollector()
		start := time.Now()
		time.Sleep(1 * time.Millisecond)

		mc.RecordEvalOperation(start, nil)

		evalOps := mc.EvalOperations.GetOrCreate(nil).(*Counter)
		if evalOps.Get() != 1 {
			t.Errorf("expected 1 eval operation, got %d", evalOps.Get())
		}

		evalDuration := mc.EvalDuration.GetOrCreate(nil).(*Histogram)
		stats := evalDuration.GetStats()
		if stats.Count != 1 {
			t.Errorf("expected 1 duration observation, got %d", stats.Count)
		}
	})

	t.Run("operator call recording", func(t *testing.T) {
		mc := NewMetricsCollector()
		start := time.Now()

		mc.RecordOperatorCall("grab", start, nil)
		mc.RecordOperatorCall("concat", start, nil)
		mc.RecordOperatorCall("grab", start, nil)

		// Check grab operator calls
		grabLabels := map[string]string{"operator": "grab"}
		grabCalls := mc.OperatorCalls.GetOrCreate(grabLabels).(*Counter)
		if grabCalls.Get() != 2 {
			t.Errorf("expected 2 grab calls, got %d", grabCalls.Get())
		}

		// Check concat operator calls
		concatLabels := map[string]string{"operator": "concat"}
		concatCalls := mc.OperatorCalls.GetOrCreate(concatLabels).(*Counter)
		if concatCalls.Get() != 1 {
			t.Errorf("expected 1 concat call, got %d", concatCalls.Get())
		}

		// Check operator duration
		grabDuration := mc.OperatorDuration.GetOrCreate(grabLabels).(*Histogram)
		grabStats := grabDuration.GetStats()
		if grabStats.Count != 2 {
			t.Errorf("expected 2 grab duration observations, got %d", grabStats.Count)
		}
	})

	t.Run("cache metrics recording", func(t *testing.T) {
		mc := NewMetricsCollector()

		// Record cache operations
		mc.RecordCacheHit("memory")
		mc.RecordCacheHit("memory")
		mc.RecordCacheMiss("memory")
		mc.RecordCacheEviction("memory", 5)
		mc.UpdateCacheSize("memory", 1024)

		memoryLabels := map[string]string{"cache": "memory"}

		cacheHits := mc.CacheHits.GetOrCreate(memoryLabels).(*Counter)
		if cacheHits.Get() != 2 {
			t.Errorf("expected 2 cache hits, got %d", cacheHits.Get())
		}

		cacheMisses := mc.CacheMisses.GetOrCreate(memoryLabels).(*Counter)
		if cacheMisses.Get() != 1 {
			t.Errorf("expected 1 cache miss, got %d", cacheMisses.Get())
		}

		cacheEvictions := mc.CacheEvictions.GetOrCreate(memoryLabels).(*Counter)
		if cacheEvictions.Get() != 5 {
			t.Errorf("expected 5 cache evictions, got %d", cacheEvictions.Get())
		}

		cacheSize := mc.CacheSize.GetOrCreate(memoryLabels).(*Gauge)
		if cacheSize.Get() != 1024 {
			t.Errorf("expected cache size 1024, got %d", cacheSize.Get())
		}
	})

	t.Run("external call recording", func(t *testing.T) {
		mc := NewMetricsCollector()
		start := time.Now()

		mc.RecordExternalCall("vault", start, nil)
		mc.RecordExternalCall("aws", start, &testError{"connection failed"})
		mc.UpdateConnectionsActive("vault", 3)

		vaultLabels := map[string]string{"system": "vault"}
		awsLabels := map[string]string{"system": "aws"}

		vaultCalls := mc.ExternalCalls.GetOrCreate(vaultLabels).(*Counter)
		if vaultCalls.Get() != 1 {
			t.Errorf("expected 1 vault call, got %d", vaultCalls.Get())
		}

		awsCalls := mc.ExternalCalls.GetOrCreate(awsLabels).(*Counter)
		if awsCalls.Get() != 1 {
			t.Errorf("expected 1 aws call, got %d", awsCalls.Get())
		}

		vaultConnections := mc.ConnectionsActive.GetOrCreate(vaultLabels).(*Gauge)
		if vaultConnections.Get() != 3 {
			t.Errorf("expected 3 active vault connections, got %d", vaultConnections.Get())
		}

		// Check error recording
		awsErrors := mc.ErrorsByType.GetOrCreate(map[string]string{"type": "external_aws"}).(*Counter)
		if awsErrors.Get() != 1 {
			t.Errorf("expected 1 aws error, got %d", awsErrors.Get())
		}
	})

	t.Run("resource metrics", func(t *testing.T) {
		mc := NewMetricsCollector()

		mc.UpdateResourceMetrics(1024*1024, 5000, 50)
		mc.RecordGCPause(5 * time.Millisecond)

		heapAlloc := mc.HeapAlloc.GetOrCreate(nil).(*Gauge)
		if heapAlloc.Get() != 1024*1024 {
			t.Errorf("expected heap alloc 1048576, got %d", heapAlloc.Get())
		}

		heapObjects := mc.HeapObjects.GetOrCreate(nil).(*Gauge)
		if heapObjects.Get() != 5000 {
			t.Errorf("expected heap objects 5000, got %d", heapObjects.Get())
		}

		goroutines := mc.Goroutines.GetOrCreate(nil).(*Gauge)
		if goroutines.Get() != 50 {
			t.Errorf("expected goroutines 50, got %d", goroutines.Get())
		}

		gcPause := mc.GCPauseTime.GetOrCreate(nil).(*Histogram)
		stats := gcPause.GetStats()
		if stats.Count != 1 {
			t.Errorf("expected 1 GC pause observation, got %d", stats.Count)
		}
		if stats.Mean < 0.004 || stats.Mean > 0.006 {
			t.Errorf("expected GC pause around 0.005s, got %f", stats.Mean)
		}
	})

	t.Run("throughput metrics", func(t *testing.T) {
		mc := NewMetricsCollector()

		mc.RecordDocument(1024)
		mc.RecordDocument(2048)
		mc.UpdateOperationsPerSecond(150.5)

		docsProcessed := mc.DocumentsProcessed.GetOrCreate(nil).(*Counter)
		if docsProcessed.Get() != 2 {
			t.Errorf("expected 2 documents processed, got %d", docsProcessed.Get())
		}

		bytesProcessed := mc.BytesProcessed.GetOrCreate(nil).(*Counter)
		if bytesProcessed.Get() != 3072 {
			t.Errorf("expected 3072 bytes processed, got %d", bytesProcessed.Get())
		}

		opsPerSec := mc.OperationsPerSecond.GetOrCreate(nil).(*Gauge)
		if opsPerSec.Get() != 150 { // int64 conversion
			t.Errorf("expected operations per second 150, got %d", opsPerSec.Get())
		}
	})

	t.Run("custom metrics", func(t *testing.T) {
		mc := NewMetricsCollector()

		// Register custom metric
		customFamily := mc.RegisterCustomMetric("custom_test_metric", "A test metric", MetricTypeCounter)
		if customFamily == nil {
			t.Fatal("expected custom metric family to be created")
		}

		if customFamily.Name != "custom_test_metric" {
			t.Errorf("expected custom metric name 'custom_test_metric', got %s", customFamily.Name)
		}

		// Use custom metric
		customMetric := customFamily.GetOrCreate(map[string]string{"label": "value"}).(*Counter)
		customMetric.Inc()

		if customMetric.Get() != 1 {
			t.Errorf("expected custom metric value 1, got %d", customMetric.Get())
		}

		// Verify it's in custom metrics map
		if storedFamily, exists := mc.CustomMetrics["custom_test_metric"]; !exists || storedFamily != customFamily {
			t.Error("custom metric should be stored in CustomMetrics map")
		}
	})

	t.Run("get all metric families", func(t *testing.T) {
		mc := NewMetricsCollector()
		mc.RegisterCustomMetric("custom1", "Custom 1", MetricTypeCounter)
		mc.RegisterCustomMetric("custom2", "Custom 2", MetricTypeGauge)

		allFamilies := mc.GetAllMetricFamilies()

		// Should include all standard families plus custom ones
		expectedMinimum := 22 + 2 // standard families + 2 custom
		if len(allFamilies) < expectedMinimum {
			t.Errorf("expected at least %d families, got %d", expectedMinimum, len(allFamilies))
		}

		// Verify custom families are included
		customFound := 0
		for _, family := range allFamilies {
			if family.Name == "custom1" || family.Name == "custom2" {
				customFound++
			}
		}
		if customFound != 2 {
			t.Errorf("expected 2 custom families in result, found %d", customFound)
		}
	})

	t.Run("reset functionality", func(t *testing.T) {
		mc := NewMetricsCollector()

		// Add some data
		mc.RecordParseOperation(time.Now(), nil)
		mc.RecordCacheHit("memory")
		mc.UpdateCacheSize("memory", 100)

		// Verify data exists
		parseOps := mc.ParseOperations.GetOrCreate(nil).(*Counter)
		if parseOps.Get() == 0 {
			t.Error("should have parse operations before reset")
		}

		cacheHits := mc.CacheHits.GetOrCreate(map[string]string{"cache": "memory"}).(*Counter)
		if cacheHits.Get() == 0 {
			t.Error("should have cache hits before reset")
		}

		cacheSize := mc.CacheSize.GetOrCreate(map[string]string{"cache": "memory"}).(*Gauge)
		if cacheSize.Get() == 0 {
			t.Error("should have cache size before reset")
		}

		// Reset all metrics
		mc.Reset()

		// Verify data is cleared
		if parseOps.Get() != 0 {
			t.Errorf("expected 0 parse operations after reset, got %d", parseOps.Get())
		}

		if cacheHits.Get() != 0 {
			t.Errorf("expected 0 cache hits after reset, got %d", cacheHits.Get())
		}

		if cacheSize.Get() != 0 {
			t.Errorf("expected 0 cache size after reset, got %d", cacheSize.Get())
		}
	})
}

// TestMetricsCollectorConcurrency tests concurrent access to metrics collector
func TestMetricsCollectorConcurrency(t *testing.T) {
	t.Run("concurrent parse recording", func(t *testing.T) {
		mc := NewMetricsCollector()
		var wg sync.WaitGroup

		goroutines := 50
		recordsPerGoroutine := 100

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < recordsPerGoroutine; j++ {
					mc.RecordParseOperation(time.Now(), nil)
				}
			}()
		}

		wg.Wait()

		parseOps := mc.ParseOperations.GetOrCreate(nil).(*Counter)
		expected := int64(goroutines * recordsPerGoroutine)
		if parseOps.Get() != expected {
			t.Errorf("expected %d parse operations, got %d", expected, parseOps.Get())
		}
	})

	t.Run("concurrent operator recording", func(t *testing.T) {
		mc := NewMetricsCollector()
		var wg sync.WaitGroup

		operators := []string{"grab", "concat", "vault", "calc"}
		recordsPerOperator := 250

		for _, op := range operators {
			wg.Add(1)
			go func(operator string) {
				defer wg.Done()
				for i := 0; i < recordsPerOperator; i++ {
					mc.RecordOperatorCall(operator, time.Now(), nil)
				}
			}(op)
		}

		wg.Wait()

		// Verify each operator has correct count
		for _, op := range operators {
			labels := map[string]string{"operator": op}
			opCalls := mc.OperatorCalls.GetOrCreate(labels).(*Counter)
			if opCalls.Get() != int64(recordsPerOperator) {
				t.Errorf("operator %s: expected %d calls, got %d", op, recordsPerOperator, opCalls.Get())
			}
		}
	})

	t.Run("concurrent cache metrics", func(t *testing.T) {
		mc := NewMetricsCollector()
		var wg sync.WaitGroup

		operations := 1000

		// Mix of hits and misses
		for i := 0; i < operations; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				mc.RecordCacheHit("memory")
			}()
			go func() {
				defer wg.Done()
				mc.RecordCacheMiss("memory")
			}()
		}

		wg.Wait()

		memoryLabels := map[string]string{"cache": "memory"}
		hits := mc.CacheHits.GetOrCreate(memoryLabels).(*Counter)
		misses := mc.CacheMisses.GetOrCreate(memoryLabels).(*Counter)

		if hits.Get() != int64(operations) {
			t.Errorf("expected %d cache hits, got %d", operations, hits.Get())
		}

		if misses.Get() != int64(operations) {
			t.Errorf("expected %d cache misses, got %d", operations, misses.Get())
		}
	})

	t.Run("concurrent custom metrics", func(t *testing.T) {
		mc := NewMetricsCollector()
		var wg sync.WaitGroup

		// Register custom metrics concurrently
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				metricName := fmt.Sprintf("custom_metric_%d", id)
				family := mc.RegisterCustomMetric(metricName, "Test metric", MetricTypeCounter)

				// Use the metric
				metric := family.GetOrCreate(nil).(*Counter)
				metric.Inc()
			}(i)
		}

		wg.Wait()

		// Verify all custom metrics were created
		if len(mc.CustomMetrics) != 50 {
			t.Errorf("expected 50 custom metrics, got %d", len(mc.CustomMetrics))
		}
	})
}

// TestGlobalMetricsCollector tests global collector functionality
func TestGlobalMetricsCollector(t *testing.T) {
	t.Run("global initialization", func(t *testing.T) {
		// Reset global state
		globalMetricsCollector = nil

		mc1 := GetMetricsCollector()
		if mc1 == nil {
			t.Fatal("global metrics collector should be initialized")
		}

		mc2 := GetMetricsCollector()
		if mc1 != mc2 {
			t.Error("should return same global instance")
		}
	})

	t.Run("convenience functions", func(t *testing.T) {
		// Reset global state
		globalMetricsCollector = nil

		// Test cache metrics recording
		RecordCacheMetrics("test_cache", true)
		RecordCacheMetrics("test_cache", false)

		mc := GetMetricsCollector()
		labels := map[string]string{"cache": "test_cache"}

		hits := mc.CacheHits.GetOrCreate(labels).(*Counter)
		if hits.Get() != 1 {
			t.Errorf("expected 1 cache hit, got %d", hits.Get())
		}

		misses := mc.CacheMisses.GetOrCreate(labels).(*Counter)
		if misses.Get() != 1 {
			t.Errorf("expected 1 cache miss, got %d", misses.Get())
		}
	})

	t.Run("time operation function", func(t *testing.T) {
		// Reset global state
		globalMetricsCollector = nil

		// Test parse operation timing
		err := TimeOperation("parse", nil, func() error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		mc := GetMetricsCollector()
		parseOps := mc.ParseOperations.GetOrCreate(nil).(*Counter)
		if parseOps.Get() != 1 {
			t.Errorf("expected 1 parse operation, got %d", parseOps.Get())
		}

		// Test eval operation timing
		err = TimeOperation("eval", nil, func() error {
			return &testError{"eval error"}
		})

		if err == nil {
			t.Error("expected error to be returned")
		}

		evalOps := mc.EvalOperations.GetOrCreate(nil).(*Counter)
		if evalOps.Get() != 1 {
			t.Errorf("expected 1 eval operation, got %d", evalOps.Get())
		}

		// Test operator timing
		labels := map[string]string{"operator": "test_op"}
		err = TimeOperation("operator", labels, func() error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		opCalls := mc.OperatorCalls.GetOrCreate(labels).(*Counter)
		if opCalls.Get() != 1 {
			t.Errorf("expected 1 operator call, got %d", opCalls.Get())
		}
	})
}

// TestMetricsCollectorErrorTypes tests different error type recording
func TestMetricsCollectorErrorTypes(t *testing.T) {
	t.Run("various error types", func(t *testing.T) {
		mc := NewMetricsCollector()

		// Record different types of errors
		mc.RecordError("parse", &testError{"syntax error"})
		mc.RecordError("eval", &testError{"runtime error"})
		mc.RecordError("operator_grab", &testError{"path not found"})
		mc.RecordError("external_vault", &testError{"connection timeout"})

		// Verify each error type is recorded
		errorTypes := []string{"parse", "eval", "operator_grab", "external_vault"}
		for _, errorType := range errorTypes {
			labels := map[string]string{"type": errorType}
			errorCount := mc.ErrorsByType.GetOrCreate(labels).(*Counter)
			if errorCount.Get() != 1 {
				t.Errorf("error type %s: expected 1 error, got %d", errorType, errorCount.Get())
			}
		}
	})

	t.Run("multiple errors of same type", func(t *testing.T) {
		mc := NewMetricsCollector()

		// Record multiple parse errors
		for i := 0; i < 5; i++ {
			mc.RecordError("parse", &testError{"parse error"})
		}

		labels := map[string]string{"type": "parse"}
		errorCount := mc.ErrorsByType.GetOrCreate(labels).(*Counter)
		if errorCount.Get() != 5 {
			t.Errorf("expected 5 parse errors, got %d", errorCount.Get())
		}
	})
}
