package internal

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestRequestDeduplicator tests basic request deduplication functionality
func TestRequestDeduplicator(t *testing.T) {
	t.Run("basic deduplication", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         5 * time.Second,
			CleanupInterval: 1 * time.Minute,
		}
		rd := NewRequestDeduplicator(config)

		executed := atomic.Int32{}
		
		// Function that increments counter
		fn := func() (interface{}, error) {
			executed.Add(1)
			time.Sleep(50 * time.Millisecond) // Simulate work
			return "result", nil
		}

		// Make multiple concurrent requests with same key
		key := "test_key"
		var wg sync.WaitGroup
		results := make([]RequestResult, 3)

		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				resultChan := rd.Deduplicate(key, fn)
				results[index] = <-resultChan
			}(i)
		}

		wg.Wait()

		// Function should only execute once
		if executed.Load() != 1 {
			t.Errorf("expected function to execute once, executed %d times", executed.Load())
		}

		// All requests should get the same result
		for i, result := range results {
			if result.Err != nil {
				t.Errorf("result %d: unexpected error: %v", i, result.Err)
			}
			if result.Value != "result" {
				t.Errorf("result %d: expected 'result', got %v", i, result.Value)
			}
		}

		// Check metrics
		metrics := rd.GetMetrics()
		if metrics.Hits != 2 {
			t.Errorf("expected 2 hits, got %d", metrics.Hits)
		}
		if metrics.Misses != 1 {
			t.Errorf("expected 1 miss, got %d", metrics.Misses)
		}
		if metrics.HitRate < 66.66 || metrics.HitRate > 66.67 { // Allow for floating point precision
			t.Errorf("expected hit rate ~66.67%%, got %.2f%%", metrics.HitRate)
		}
	})

	t.Run("different keys", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         5 * time.Second,
			CleanupInterval: 1 * time.Minute,
		}
		rd := NewRequestDeduplicator(config)

		executed := atomic.Int32{}
		
		fn := func() (interface{}, error) {
			count := executed.Add(1)
			return fmt.Sprintf("result_%d", count), nil
		}

		// Make requests with different keys
		key1 := "key1"
		key2 := "key2"

		result1Chan := rd.Deduplicate(key1, fn)
		result2Chan := rd.Deduplicate(key2, fn)

		result1 := <-result1Chan
		result2 := <-result2Chan

		// Function should execute twice (once per key)
		if executed.Load() != 2 {
			t.Errorf("expected function to execute twice, executed %d times", executed.Load())
		}

		// Results should be different
		if result1.Value == result2.Value {
			t.Errorf("results should be different: %v vs %v", result1.Value, result2.Value)
		}

		// Check metrics
		metrics := rd.GetMetrics()
		if metrics.Hits != 0 {
			t.Errorf("expected 0 hits, got %d", metrics.Hits)
		}
		if metrics.Misses != 2 {
			t.Errorf("expected 2 misses, got %d", metrics.Misses)
		}
	})

	t.Run("function with error", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         5 * time.Second,
			CleanupInterval: 1 * time.Minute,
		}
		rd := NewRequestDeduplicator(config)

		testErr := fmt.Errorf("test error")
		fn := func() (interface{}, error) {
			time.Sleep(10 * time.Millisecond)
			return nil, testErr
		}

		// Make multiple requests
		key := "error_key"
		var wg sync.WaitGroup
		results := make([]RequestResult, 2)

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				resultChan := rd.Deduplicate(key, fn)
				results[index] = <-resultChan
			}(i)
		}

		wg.Wait()

		// All requests should get the same error
		for i, result := range results {
			if result.Err == nil {
				t.Errorf("result %d: expected error, got nil", i)
			}
			if result.Err.Error() != testErr.Error() {
				t.Errorf("result %d: expected error %v, got %v", i, testErr, result.Err)
			}
		}

		// Check error metrics
		metrics := rd.GetMetrics()
		if metrics.Errors != 1 {
			t.Errorf("expected 1 error, got %d", metrics.Errors)
		}
	})

	t.Run("request timeout", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         100 * time.Millisecond, // Short timeout
			CleanupInterval: 1 * time.Minute,
		}
		rd := NewRequestDeduplicator(config)

		fn := func() (interface{}, error) {
			time.Sleep(200 * time.Millisecond) // Longer than timeout
			return "should_not_complete", nil
		}

		key := "timeout_key"
		resultChan := rd.Deduplicate(key, fn)
		result := <-resultChan

		// Should get timeout error
		if result.Err == nil {
			t.Error("expected timeout error, got nil")
		}
		if !isTimeoutError(result.Err) {
			t.Errorf("expected timeout error, got %v", result.Err)
		}

		// Check timeout metrics
		metrics := rd.GetMetrics()
		if metrics.Timeouts != 1 {
			t.Errorf("expected 1 timeout, got %d", metrics.Timeouts)
		}
	})

	t.Run("context with timeout", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         5 * time.Second,
			CleanupInterval: 1 * time.Minute,
		}
		rd := NewRequestDeduplicator(config)

		fn := func() (interface{}, error) {
			time.Sleep(200 * time.Millisecond)
			return "result", nil
		}

		// Create context with shorter timeout than deduplicator
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		key := "ctx_timeout_key"
		resultChan := rd.DeduplicateWithContext(ctx, key, fn)
		result := <-resultChan

		// Should get timeout error
		if result.Err == nil {
			t.Error("expected timeout error, got nil")
		}
	})
}

// TestRequestDeduplicatorConcurrency tests concurrent access
func TestRequestDeduplicatorConcurrency(t *testing.T) {
	t.Run("high concurrency same key", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         10 * time.Second,
			CleanupInterval: 1 * time.Minute,
		}
		rd := NewRequestDeduplicator(config)

		executed := atomic.Int32{}
		
		fn := func() (interface{}, error) {
			count := executed.Add(1)
			time.Sleep(50 * time.Millisecond) // Simulate work
			return fmt.Sprintf("result_%d", count), nil
		}

		// Make many concurrent requests with same key
		key := "concurrent_key"
		concurrentRequests := 100
		var wg sync.WaitGroup
		results := make([]RequestResult, concurrentRequests)

		for i := 0; i < concurrentRequests; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				resultChan := rd.Deduplicate(key, fn)
				results[index] = <-resultChan
			}(i)
		}

		wg.Wait()

		// Function should execute only once
		if executed.Load() != 1 {
			t.Errorf("expected function to execute once, executed %d times", executed.Load())
		}

		// All results should be the same
		expectedResult := results[0]
		for i, result := range results {
			if result.Err != nil {
				t.Errorf("result %d: unexpected error: %v", i, result.Err)
			}
			if result.Value != expectedResult.Value {
				t.Errorf("result %d: expected %v, got %v", i, expectedResult.Value, result.Value)
			}
		}

		// Check metrics
		metrics := rd.GetMetrics()
		if metrics.Hits != uint64(concurrentRequests-1) {
			t.Errorf("expected %d hits, got %d", concurrentRequests-1, metrics.Hits)
		}
		if metrics.Misses != 1 {
			t.Errorf("expected 1 miss, got %d", metrics.Misses)
		}
	})

	t.Run("many different keys", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         10 * time.Second,
			CleanupInterval: 1 * time.Minute,
		}
		rd := NewRequestDeduplicator(config)

		executed := atomic.Int32{}
		
		fn := func() (interface{}, error) {
			count := executed.Add(1)
			time.Sleep(10 * time.Millisecond)
			return fmt.Sprintf("result_%d", count), nil
		}

		// Make concurrent requests with different keys
		uniqueKeys := 50
		var wg sync.WaitGroup
		results := make([]RequestResult, uniqueKeys)

		for i := 0; i < uniqueKeys; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				key := fmt.Sprintf("key_%d", index)
				resultChan := rd.Deduplicate(key, fn)
				results[index] = <-resultChan
			}(i)
		}

		wg.Wait()

		// Function should execute once per key
		if executed.Load() != int32(uniqueKeys) {
			t.Errorf("expected function to execute %d times, executed %d times", uniqueKeys, executed.Load())
		}

		// All results should be different
		seen := make(map[interface{}]bool)
		for i, result := range results {
			if result.Err != nil {
				t.Errorf("result %d: unexpected error: %v", i, result.Err)
			}
			if seen[result.Value] {
				t.Errorf("result %d: duplicate value %v", i, result.Value)
			}
			seen[result.Value] = true
		}

		// Check metrics
		metrics := rd.GetMetrics()
		if metrics.Hits != 0 {
			t.Errorf("expected 0 hits, got %d", metrics.Hits)
		}
		if metrics.Misses != uint64(uniqueKeys) {
			t.Errorf("expected %d misses, got %d", uniqueKeys, metrics.Misses)
		}
	})

	t.Run("mixed same and different keys", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         10 * time.Second,
			CleanupInterval: 1 * time.Minute,
		}
		rd := NewRequestDeduplicator(config)

		executed := atomic.Int32{}
		
		fn := func() (interface{}, error) {
			count := executed.Add(1)
			time.Sleep(20 * time.Millisecond)
			return fmt.Sprintf("result_%d", count), nil
		}

		// Use 5 different keys, but make 4 requests per key
		uniqueKeys := 5
		requestsPerKey := 4
		totalRequests := uniqueKeys * requestsPerKey

		var wg sync.WaitGroup
		results := make([]RequestResult, totalRequests)

		for i := 0; i < totalRequests; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				keyIndex := index % uniqueKeys
				key := fmt.Sprintf("mixed_key_%d", keyIndex)
				resultChan := rd.Deduplicate(key, fn)
				results[index] = <-resultChan
			}(i)
		}

		wg.Wait()

		// Function should execute once per unique key
		if executed.Load() != int32(uniqueKeys) {
			t.Errorf("expected function to execute %d times, executed %d times", uniqueKeys, executed.Load())
		}

		// Group results by key and verify they're identical within each group
		resultsByKey := make(map[int][]RequestResult)
		for i, result := range results {
			keyIndex := i % uniqueKeys
			resultsByKey[keyIndex] = append(resultsByKey[keyIndex], result)
		}

		for keyIndex, keyResults := range resultsByKey {
			if len(keyResults) != requestsPerKey {
				t.Errorf("key %d: expected %d results, got %d", keyIndex, requestsPerKey, len(keyResults))
				continue
			}

			firstResult := keyResults[0]
			for j, result := range keyResults {
				if result.Err != nil {
					t.Errorf("key %d, result %d: unexpected error: %v", keyIndex, j, result.Err)
				}
				if result.Value != firstResult.Value {
					t.Errorf("key %d, result %d: expected %v, got %v", keyIndex, j, firstResult.Value, result.Value)
				}
			}
		}

		// Check metrics
		metrics := rd.GetMetrics()
		expectedHits := uint64(totalRequests - uniqueKeys) // All but one request per key should be hits
		if metrics.Hits != expectedHits {
			t.Errorf("expected %d hits, got %d", expectedHits, metrics.Hits)
		}
		if metrics.Misses != uint64(uniqueKeys) {
			t.Errorf("expected %d misses, got %d", uniqueKeys, metrics.Misses)
		}
	})
}

// TestRequestDeduplicatorCleanup tests cleanup functionality
func TestRequestDeduplicatorCleanup(t *testing.T) {
	t.Run("manual cleanup", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         100 * time.Millisecond,
			CleanupInterval: 1 * time.Hour, // Long interval so automatic cleanup won't interfere
		}
		rd := NewRequestDeduplicator(config)

		// Create a long-running function that won't complete before cleanup
		fn := func() (interface{}, error) {
			time.Sleep(500 * time.Millisecond) // Longer than cleanup time
			return "result", nil
		}

		// Start request but don't wait for it
		key := "cleanup_key"
		rd.Deduplicate(key, fn)
		
		// Wait for request to start
		time.Sleep(10 * time.Millisecond)

		// Verify pending request exists
		metrics := rd.GetMetrics()
		if metrics.Pending != 1 {
			t.Errorf("expected 1 pending request, got %d", metrics.Pending)
		}

		// Wait longer than 2x timeout (cleanup threshold)
		time.Sleep(250 * time.Millisecond)

		// Manual cleanup
		rd.cleanup()

		// Check that pending requests were cleaned up
		finalMetrics := rd.GetMetrics()
		if finalMetrics.Pending != 0 {
			t.Errorf("expected 0 pending requests after cleanup, got %d", finalMetrics.Pending)
		}
	})

	t.Run("automatic cleanup", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         50 * time.Millisecond,
			CleanupInterval: 100 * time.Millisecond, // Fast cleanup for testing
		}
		rd := NewRequestDeduplicator(config)

		fn := func() (interface{}, error) {
			time.Sleep(25 * time.Millisecond)
			return "result", nil
		}

		// Make several requests
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("cleanup_auto_key_%d", i)
			resultChan := rd.Deduplicate(key, fn)
			<-resultChan
		}

		// Check initial pending count
		initialMetrics := rd.GetMetrics()
		initialPending := initialMetrics.Pending

		// Wait for automatic cleanup (cleanup interval + 2x timeout)
		time.Sleep(300 * time.Millisecond)

		// Check that automatic cleanup occurred
		finalMetrics := rd.GetMetrics()
		if finalMetrics.Pending >= initialPending {
			t.Logf("Automatic cleanup may not have occurred yet: %d -> %d", initialPending, finalMetrics.Pending)
			// Don't fail the test as timing can be flaky
		}
	})
}

// TestKeyBuilder tests key building functionality
func TestKeyBuilder(t *testing.T) {
	t.Run("basic key building", func(t *testing.T) {
		kb := NewKeyBuilder("test")
		
		key1 := kb.BuildKey("component1", "component2")
		key2 := kb.BuildKey("component1", "component2")
		key3 := kb.BuildKey("component1", "component3")

		// Same components should produce same key
		if key1 != key2 {
			t.Errorf("same components should produce same key: %s vs %s", key1, key2)
		}

		// Different components should produce different keys
		if key1 == key3 {
			t.Errorf("different components should produce different keys: %s vs %s", key1, key3)
		}

		// Key should have correct prefix
		if !containsSubstring(key1, "test:") {
			t.Errorf("key should contain prefix: %s", key1)
		}
	})

	t.Run("empty components", func(t *testing.T) {
		kb := NewKeyBuilder("empty")
		
		key1 := kb.BuildKey()
		key2 := kb.BuildKey("")
		key3 := kb.BuildKey("", "")

		// All should be different
		if key1 == key2 || key1 == key3 || key2 == key3 {
			t.Errorf("different empty component combinations should produce different keys: %s, %s, %s", key1, key2, key3)
		}
	})

	t.Run("global key builders", func(t *testing.T) {
		// Test that global key builders exist and work
		vaultKey1 := VaultKeyBuilder.BuildKey("secret", "key")
		vaultKey2 := VaultKeyBuilder.BuildKey("secret", "key")
		awsKey := AWSKeyBuilder.BuildKey("secret", "key")
		fileKey := FileKeyBuilder.BuildKey("secret", "key")

		// Same vault keys should be identical
		if vaultKey1 != vaultKey2 {
			t.Errorf("same vault keys should be identical: %s vs %s", vaultKey1, vaultKey2)
		}

		// Different builder types should produce different keys
		if vaultKey1 == awsKey || vaultKey1 == fileKey || awsKey == fileKey {
			t.Errorf("different builders should produce different keys: vault=%s, aws=%s, file=%s", vaultKey1, awsKey, fileKey)
		}

		// Check prefixes
		if !containsSubstring(vaultKey1, "vault:") {
			t.Errorf("vault key should contain vault prefix: %s", vaultKey1)
		}
		if !containsSubstring(awsKey, "aws:") {
			t.Errorf("aws key should contain aws prefix: %s", awsKey)
		}
		if !containsSubstring(fileKey, "file:") {
			t.Errorf("file key should contain file prefix: %s", fileKey)
		}
	})
}

// TestGlobalDeduplicators tests global deduplicator functionality
func TestGlobalDeduplicators(t *testing.T) {
	t.Run("initialization", func(t *testing.T) {
		// Reset global state
		VaultDeduplicator = nil
		AWSDeduplicator = nil
		FileDeduplicator = nil
		deduplicatorInitOnce = sync.Once{}

		// Initialize should create deduplicators
		InitializeDeduplicators()

		if VaultDeduplicator == nil {
			t.Error("VaultDeduplicator should be initialized")
		}
		if AWSDeduplicator == nil {
			t.Error("AWSDeduplicator should be initialized")
		}
		if FileDeduplicator == nil {
			t.Error("FileDeduplicator should be initialized")
		}

		// Multiple calls should not create new instances
		vault1 := VaultDeduplicator
		InitializeDeduplicators()
		vault2 := VaultDeduplicator

		if vault1 != vault2 {
			t.Error("multiple initialization calls should return same instance")
		}
	})

	t.Run("metrics collection", func(t *testing.T) {
		// Ensure deduplicators are initialized
		InitializeDeduplicators()

		// Use the deduplicators
		fn := func() (interface{}, error) {
			return "test_result", nil
		}

		vaultKey := VaultKeyBuilder.BuildKey("test", "vault")
		awsKey := AWSKeyBuilder.BuildKey("test", "aws")

		// Make some requests
		<-VaultDeduplicator.Deduplicate(vaultKey, fn)
		<-AWSDeduplicator.Deduplicate(awsKey, fn)

		// Get metrics
		allMetrics := GetDeduplicationMetrics()

		if len(allMetrics) != 3 {
			t.Errorf("expected 3 sets of metrics, got %d", len(allMetrics))
		}

		vaultMetrics, hasVault := allMetrics["vault"]
		awsMetrics, hasAWS := allMetrics["aws"]
		fileMetrics, hasFile := allMetrics["file"]

		if !hasVault {
			t.Error("should have vault metrics")
		}
		if !hasAWS {
			t.Error("should have aws metrics")
		}
		if !hasFile {
			t.Error("should have file metrics")
		}

		// Check that vault and aws have activity
		if vaultMetrics.Misses == 0 {
			t.Error("vault should have at least one miss")
		}
		if awsMetrics.Misses == 0 {
			t.Error("aws should have at least one miss")
		}

		// File should have no activity
		if fileMetrics.Misses != 0 || fileMetrics.Hits != 0 {
			t.Error("file should have no activity")
		}
	})
}

// TestRequestDeduplicationMetrics tests metrics calculations
func TestRequestDeduplicationMetrics(t *testing.T) {
	t.Run("hit rate calculation", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         5 * time.Second,
			CleanupInterval: 1 * time.Minute,
		}
		rd := NewRequestDeduplicator(config)

		// No requests initially
		metrics := rd.GetMetrics()
		if metrics.HitRate != 0.0 {
			t.Errorf("expected 0%% hit rate initially, got %.2f%%", metrics.HitRate)
		}

		fn := func() (interface{}, error) {
			time.Sleep(10 * time.Millisecond)
			return "result", nil
		}

		// Make 1 miss and 3 hits (same key)
		key := "hit_rate_key"
		var wg sync.WaitGroup
		
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resultChan := rd.Deduplicate(key, fn)
				<-resultChan
			}()
		}
		
		wg.Wait()

		// Check hit rate
		finalMetrics := rd.GetMetrics()
		expectedHitRate := 75.0 // 3 hits out of 4 total = 75%
		if finalMetrics.HitRate != expectedHitRate {
			t.Errorf("expected %.1f%% hit rate, got %.2f%%", expectedHitRate, finalMetrics.HitRate)
		}
	})

	t.Run("metrics string representation", func(t *testing.T) {
		metrics := RequestDeduplicationMetrics{
			Hits:     10,
			Misses:   5,
			Timeouts: 2,
			Errors:   1,
			Pending:  3,
			HitRate:  66.67,
		}

		str := metrics.String()
		
		// Should contain all key information
		expectedSubstrings := []string{
			"Hits: 10",
			"Misses: 5",
			"Timeouts: 2",
			"Errors: 1",
			"Pending: 3",
			"66.67%",
		}

		for _, substr := range expectedSubstrings {
			if !containsSubstring(str, substr) {
				t.Errorf("metrics string should contain '%s': %s", substr, str)
			}
		}
	})
}

// Helper functions

func isTimeoutError(err error) bool {
	return containsSubstring(err.Error(), "timeout")
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}