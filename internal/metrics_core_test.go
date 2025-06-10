package internal

import (
	"fmt"
	"math"
	"sync"
	"testing"
	"time"
)

// TestCounter tests counter metric functionality
func TestCounter(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		labels := map[string]string{"test": "value"}
		counter := NewCounter("test_counter", labels)

		if counter.Name() != "test_counter" {
			t.Errorf("expected name 'test_counter', got %s", counter.Name())
		}

		if counter.Type() != MetricTypeCounter {
			t.Errorf("expected type %s, got %s", MetricTypeCounter, counter.Type())
		}

		if counter.Labels()["test"] != "value" {
			t.Error("labels not preserved")
		}

		// Initial value should be 0
		if counter.Get() != 0 {
			t.Errorf("expected initial value 0, got %d", counter.Get())
		}

		// Test increment
		counter.Inc()
		if counter.Get() != 1 {
			t.Errorf("expected value 1 after Inc(), got %d", counter.Get())
		}

		// Test add
		counter.Add(5)
		if counter.Get() != 6 {
			t.Errorf("expected value 6 after Add(5), got %d", counter.Get())
		}

		// Test Value() interface
		value := counter.Value()
		if v, ok := value.(int64); !ok || v != 6 {
			t.Errorf("expected Value() to return int64(6), got %v", value)
		}
	})

	t.Run("concurrent operations", func(t *testing.T) {
		counter := NewCounter("concurrent_test", nil)
		var wg sync.WaitGroup

		goroutines := 100
		incrementsPerGoroutine := 1000

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < incrementsPerGoroutine; j++ {
					counter.Inc()
				}
			}()
		}

		wg.Wait()

		expected := int64(goroutines * incrementsPerGoroutine)
		if counter.Get() != expected {
			t.Errorf("expected %d, got %d", expected, counter.Get())
		}
	})

	t.Run("reset functionality", func(t *testing.T) {
		counter := NewCounter("reset_test", nil)
		counter.Add(42)

		if counter.Get() != 42 {
			t.Errorf("expected 42 before reset, got %d", counter.Get())
		}

		counter.Reset()
		if counter.Get() != 0 {
			t.Errorf("expected 0 after reset, got %d", counter.Get())
		}
	})
}

// TestGauge tests gauge metric functionality
func TestGauge(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		labels := map[string]string{"env": "test"}
		gauge := NewGauge("test_gauge", labels)

		if gauge.Name() != "test_gauge" {
			t.Errorf("expected name 'test_gauge', got %s", gauge.Name())
		}

		if gauge.Type() != MetricTypeGauge {
			t.Errorf("expected type %s, got %s", MetricTypeGauge, gauge.Type())
		}

		// Test set
		gauge.Set(42)
		if gauge.Get() != 42 {
			t.Errorf("expected value 42 after Set(42), got %d", gauge.Get())
		}

		// Test increment
		gauge.Inc()
		if gauge.Get() != 43 {
			t.Errorf("expected value 43 after Inc(), got %d", gauge.Get())
		}

		// Test decrement
		gauge.Dec()
		if gauge.Get() != 42 {
			t.Errorf("expected value 42 after Dec(), got %d", gauge.Get())
		}

		// Test add positive
		gauge.Add(8)
		if gauge.Get() != 50 {
			t.Errorf("expected value 50 after Add(8), got %d", gauge.Get())
		}

		// Test add negative (subtract)
		gauge.Add(-10)
		if gauge.Get() != 40 {
			t.Errorf("expected value 40 after Add(-10), got %d", gauge.Get())
		}
	})

	t.Run("negative values", func(t *testing.T) {
		gauge := NewGauge("negative_test", nil)

		gauge.Set(-100)
		if gauge.Get() != -100 {
			t.Errorf("expected -100, got %d", gauge.Get())
		}

		gauge.Dec()
		if gauge.Get() != -101 {
			t.Errorf("expected -101 after Dec(), got %d", gauge.Get())
		}
	})

	t.Run("concurrent operations", func(t *testing.T) {
		gauge := NewGauge("concurrent_gauge", nil)
		var wg sync.WaitGroup

		// Half increment, half decrement
		operations := 1000
		for i := 0; i < operations/2; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				gauge.Inc()
			}()
			go func() {
				defer wg.Done()
				gauge.Dec()
			}()
		}

		wg.Wait()

		// Should end up at 0 (or very close due to timing)
		final := gauge.Get()
		if final < -10 || final > 10 {
			t.Errorf("expected final value near 0, got %d", final)
		}
	})
}

// TestHistogram tests histogram metric functionality
func TestHistogram(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		histogram := NewHistogram("test_histogram", nil)

		if histogram.Name() != "test_histogram" {
			t.Errorf("expected name 'test_histogram', got %s", histogram.Name())
		}

		if histogram.Type() != MetricTypeHistogram {
			t.Errorf("expected type %s, got %s", MetricTypeHistogram, histogram.Type())
		}

		// Test observations
		histogram.Observe(1.0)
		histogram.Observe(2.0)
		histogram.Observe(3.0)
		histogram.Observe(4.0)
		histogram.Observe(5.0)

		stats := histogram.GetStats()
		if stats.Count != 5 {
			t.Errorf("expected count 5, got %d", stats.Count)
		}

		if stats.Sum != 15.0 {
			t.Errorf("expected sum 15.0, got %f", stats.Sum)
		}

		if stats.Mean != 3.0 {
			t.Errorf("expected mean 3.0, got %f", stats.Mean)
		}

		if stats.Min != 1.0 {
			t.Errorf("expected min 1.0, got %f", stats.Min)
		}

		if stats.Max != 5.0 {
			t.Errorf("expected max 5.0, got %f", stats.Max)
		}

		// Test percentiles (with sorted values 1,2,3,4,5)
		if stats.P50 != 3.0 {
			t.Errorf("expected P50 3.0, got %f", stats.P50)
		}

		// For array [1,2,3,4,5], P90 at index int(4*0.9) = 3, so value = 4.0
		if stats.P90 != 4.0 {
			t.Errorf("expected P90 4.0, got %f", stats.P90)
		}
	})

	t.Run("observe duration", func(t *testing.T) {
		histogram := NewHistogram("duration_test", nil)

		start := time.Now()
		time.Sleep(10 * time.Millisecond)
		histogram.ObserveDuration(start)

		stats := histogram.GetStats()
		if stats.Count != 1 {
			t.Errorf("expected count 1, got %d", stats.Count)
		}

		// Should be approximately 0.01 seconds
		if stats.Mean < 0.005 || stats.Mean > 0.050 {
			t.Errorf("expected mean around 0.01s, got %f", stats.Mean)
		}
	})

	t.Run("empty histogram", func(t *testing.T) {
		histogram := NewHistogram("empty_test", nil)
		stats := histogram.GetStats()

		if stats.Count != 0 {
			t.Errorf("expected count 0 for empty histogram, got %d", stats.Count)
		}

		if stats.Sum != 0 {
			t.Errorf("expected sum 0 for empty histogram, got %f", stats.Sum)
		}
	})

	t.Run("value trimming", func(t *testing.T) {
		histogram := NewHistogram("trim_test", nil)

		// Add more than 10000 values to test trimming
		for i := 0; i < 12000; i++ {
			histogram.Observe(float64(i))
		}

		stats := histogram.GetStats()
		if stats.Count != 12000 {
			t.Errorf("expected count 12000, got %d", stats.Count)
		}

		// Min should be from the kept values (last 10000)
		if stats.Min < 2000 {
			t.Errorf("expected min >= 2000 due to trimming, got %f", stats.Min)
		}
	})

	t.Run("concurrent observations", func(t *testing.T) {
		histogram := NewHistogram("concurrent_hist", nil)
		var wg sync.WaitGroup

		goroutines := 50
		observationsPerGoroutine := 100

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < observationsPerGoroutine; j++ {
					histogram.Observe(float64(goroutineID*100 + j))
				}
			}(i)
		}

		wg.Wait()

		stats := histogram.GetStats()
		expectedCount := int64(goroutines * observationsPerGoroutine)
		if stats.Count != expectedCount {
			t.Errorf("expected count %d, got %d", expectedCount, stats.Count)
		}
	})

	t.Run("reset functionality", func(t *testing.T) {
		histogram := NewHistogram("reset_hist", nil)
		histogram.Observe(1.0)
		histogram.Observe(2.0)

		stats := histogram.GetStats()
		if stats.Count != 2 {
			t.Errorf("expected count 2 before reset, got %d", stats.Count)
		}

		histogram.Reset()
		stats = histogram.GetStats()
		if stats.Count != 0 {
			t.Errorf("expected count 0 after reset, got %d", stats.Count)
		}
	})
}

// TestSummary tests summary metric functionality
func TestSummary(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		quantiles := []float64{0.5, 0.9, 0.95, 0.99}
		summary := NewSummary("test_summary", nil, quantiles)

		if summary.Type() != MetricTypeSummary {
			t.Errorf("expected type %s, got %s", MetricTypeSummary, summary.Type())
		}

		// Add observations
		for i := 1; i <= 100; i++ {
			summary.Observe(float64(i))
		}

		quantileValues := summary.GetQuantiles()
		if quantileValues == nil {
			t.Fatal("expected quantile values, got nil")
		}

		// Check that we have all requested quantiles
		for _, q := range quantiles {
			if _, exists := quantileValues[q]; !exists {
				t.Errorf("missing quantile %f", q)
			}
		}

		// P50 should be around 50
		if q50 := quantileValues[0.5]; math.Abs(q50-50.0) > 5.0 {
			t.Errorf("expected P50 around 50, got %f", q50)
		}

		// P99 should be around 99
		if q99 := quantileValues[0.99]; math.Abs(q99-99.0) > 5.0 {
			t.Errorf("expected P99 around 99, got %f", q99)
		}
	})

	t.Run("empty summary", func(t *testing.T) {
		summary := NewSummary("empty_summary", nil, []float64{0.5, 0.9})
		quantiles := summary.GetQuantiles()

		if quantiles != nil {
			t.Error("expected nil quantiles for empty summary")
		}
	})

	t.Run("value interface", func(t *testing.T) {
		summary := NewSummary("value_test", nil, []float64{0.5})
		summary.Observe(42.0)

		value := summary.Value()
		valueMap, ok := value.(map[string]interface{})
		if !ok {
			t.Fatalf("expected Value() to return map[string]interface{}, got %T", value)
		}

		if _, hasStats := valueMap["stats"]; !hasStats {
			t.Error("expected 'stats' key in value map")
		}

		if _, hasQuantiles := valueMap["quantiles"]; !hasQuantiles {
			t.Error("expected 'quantiles' key in value map")
		}
	})
}

// TestMetricFamily tests metric family functionality
func TestMetricFamily(t *testing.T) {
	t.Run("counter family", func(t *testing.T) {
		family := NewMetricFamily("test_counters", "Test counter family", MetricTypeCounter)

		if family.Name != "test_counters" {
			t.Errorf("expected name 'test_counters', got %s", family.Name)
		}

		if family.Type != MetricTypeCounter {
			t.Errorf("expected type %s, got %s", MetricTypeCounter, family.Type)
		}

		// Get metrics with different labels
		labels1 := map[string]string{"env": "prod", "service": "api"}
		labels2 := map[string]string{"env": "dev", "service": "api"}

		metric1 := family.GetOrCreate(labels1)
		metric2 := family.GetOrCreate(labels2)

		if metric1 == metric2 {
			t.Error("different labels should create different metrics")
		}

		// Same labels should return same metric
		metric1Again := family.GetOrCreate(labels1)
		if metric1 != metric1Again {
			t.Error("same labels should return same metric")
		}

		// Test type assertion
		counter1, ok := metric1.(*Counter)
		if !ok {
			t.Error("metric should be a Counter")
		}

		counter1.Inc()
		if counter1.Get() != 1 {
			t.Error("counter should work normally")
		}

		// Test GetAll
		allMetrics := family.GetAll()
		if len(allMetrics) != 2 {
			t.Errorf("expected 2 metrics, got %d", len(allMetrics))
		}
	})

	t.Run("histogram family", func(t *testing.T) {
		family := NewMetricFamily("test_histograms", "Test histogram family", MetricTypeHistogram)

		labels := map[string]string{"operation": "parse"}
		metric := family.GetOrCreate(labels)

		histogram, ok := metric.(*Histogram)
		if !ok {
			t.Error("metric should be a Histogram")
		}

		histogram.Observe(1.5)
		stats := histogram.GetStats()
		if stats.Count != 1 {
			t.Error("histogram should work normally")
		}
	})

	t.Run("concurrent access", func(t *testing.T) {
		family := NewMetricFamily("concurrent_family", "Test", MetricTypeCounter)
		var wg sync.WaitGroup

		// Create metrics concurrently
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				labels := map[string]string{"id": fmt.Sprintf("%d", id)}
				metric := family.GetOrCreate(labels)
				counter := metric.(*Counter)
				counter.Inc()
			}(i)
		}

		wg.Wait()

		allMetrics := family.GetAll()
		if len(allMetrics) != 100 {
			t.Errorf("expected 100 metrics, got %d", len(allMetrics))
		}
	})

	t.Run("family reset", func(t *testing.T) {
		family := NewMetricFamily("reset_family", "Test", MetricTypeCounter)

		labels := map[string]string{"test": "value"}
		metric := family.GetOrCreate(labels)
		counter := metric.(*Counter)
		counter.Add(42)

		if counter.Get() != 42 {
			t.Errorf("expected 42 before reset, got %d", counter.Get())
		}

		family.Reset()

		if counter.Get() != 0 {
			t.Errorf("expected 0 after family reset, got %d", counter.Get())
		}
	})

	t.Run("invalid metric type", func(t *testing.T) {
		family := NewMetricFamily("invalid_family", "Test", MetricType("invalid"))

		// Should panic on invalid type
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for invalid metric type")
			}
		}()

		family.GetOrCreate(nil)
	})
}

// TestLabelsToKey tests label key generation
func TestLabelsToKey(t *testing.T) {
	t.Run("empty labels", func(t *testing.T) {
		key := labelsToKey(nil)
		if key != "" {
			t.Errorf("expected empty key for nil labels, got %s", key)
		}

		key = labelsToKey(map[string]string{})
		if key != "" {
			t.Errorf("expected empty key for empty labels, got %s", key)
		}
	})

	t.Run("single label", func(t *testing.T) {
		labels := map[string]string{"env": "prod"}
		key := labelsToKey(labels)
		expected := "env=prod"
		if key != expected {
			t.Errorf("expected key %s, got %s", expected, key)
		}
	})

	t.Run("multiple labels ordered", func(t *testing.T) {
		labels := map[string]string{
			"env":     "prod",
			"service": "api",
			"version": "1.0",
		}
		key := labelsToKey(labels)

		// Should be sorted by key name
		expected := "env=prod,service=api,version=1.0"
		if key != expected {
			t.Errorf("expected key %s, got %s", expected, key)
		}
	})

	t.Run("consistent ordering", func(t *testing.T) {
		labels1 := map[string]string{"b": "2", "a": "1", "c": "3"}
		labels2 := map[string]string{"c": "3", "a": "1", "b": "2"}

		key1 := labelsToKey(labels1)
		key2 := labelsToKey(labels2)

		if key1 != key2 {
			t.Errorf("keys should be identical for same labels: %s vs %s", key1, key2)
		}
	})
}

// TestHelperFunctions tests utility functions
func TestHelperFunctions(t *testing.T) {
	t.Run("percentile function", func(t *testing.T) {
		// Test with sorted array
		values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

		p50 := percentile(values, 0.5)
		if p50 != 5.0 {
			t.Errorf("expected P50 5.0, got %f", p50)
		}

		p90 := percentile(values, 0.9)
		if p90 != 9.0 {
			t.Errorf("expected P90 9.0, got %f", p90)
		}

		p100 := percentile(values, 1.0)
		if p100 != 10.0 {
			t.Errorf("expected P100 10.0, got %f", p100)
		}

		// Test with empty array
		p50Empty := percentile([]float64{}, 0.5)
		if p50Empty != 0.0 {
			t.Errorf("expected 0.0 for empty array, got %f", p50Empty)
		}
	})

	t.Run("quicksort function", func(t *testing.T) {
		values := []float64{3.5, 1.2, 4.8, 2.1, 5.9, 0.5}
		expected := []float64{0.5, 1.2, 2.1, 3.5, 4.8, 5.9}

		quickSort(values)

		for i, v := range values {
			if v != expected[i] {
				t.Errorf("at index %d: expected %f, got %f", i, expected[i], v)
			}
		}
	})

	t.Run("quicksort strings", func(t *testing.T) {
		strings := []string{"charlie", "alpha", "delta", "bravo"}
		expected := []string{"alpha", "bravo", "charlie", "delta"}

		quickSortStrings(strings)

		for i, s := range strings {
			if s != expected[i] {
				t.Errorf("at index %d: expected %s, got %s", i, expected[i], s)
			}
		}
	})

	t.Run("quicksort edge cases", func(t *testing.T) {
		// Empty array
		empty := []float64{}
		quickSort(empty)
		if len(empty) != 0 {
			t.Error("empty array should remain empty")
		}

		// Single element
		single := []float64{42.0}
		quickSort(single)
		if len(single) != 1 || single[0] != 42.0 {
			t.Error("single element array should remain unchanged")
		}

		// Already sorted
		sorted := []float64{1.0, 2.0, 3.0, 4.0}
		original := make([]float64, len(sorted))
		copy(original, sorted)
		quickSort(sorted)
		for i, v := range sorted {
			if v != original[i] {
				t.Error("already sorted array should remain unchanged")
			}
		}
	})
}
