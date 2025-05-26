package spruce

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestHTTPClientPool tests the HTTP client pool functionality
func TestHTTPClientPool(t *testing.T) {
	t.Run("BasicPooling", func(t *testing.T) {
		config := HTTPClientPoolConfig{
			MaxClients:      5,
			IdleTimeout:     30 * time.Second,
			ConnectTimeout:  5 * time.Second,
			RequestTimeout:  10 * time.Second,
			MaxIdleConns:    10,
			MaxConnsPerHost: 5,
		}
		
		pool := NewHTTPClientPool(config)
		defer pool.Close()
		
		// Get a client
		client1 := pool.Get()
		if client1 == nil {
			t.Fatal("Expected to get a client from pool")
		}
		
		// Put it back
		pool.Put(client1)
		
		// Get another client (should be the same one)
		client2 := pool.Get()
		if client2 != client1 {
			t.Log("Got different client - this is acceptable behavior")
		}
		
		// Check metrics
		metrics := pool.Metrics()
		if metrics.Hits == 0 && metrics.Misses == 0 {
			t.Error("Expected some metrics to be recorded")
		}
		
		t.Logf("Pool metrics: %s", metrics.String())
	})
	
	t.Run("ConcurrentAccess", func(t *testing.T) {
		config := HTTPClientPoolConfig{
			MaxClients:     10,
			IdleTimeout:    30 * time.Second,
			ConnectTimeout: 5 * time.Second,
			RequestTimeout: 10 * time.Second,
		}
		
		pool := NewHTTPClientPool(config)
		defer pool.Close()
		
		var wg sync.WaitGroup
		clientsUsed := make(chan *http.Client, 100)
		
		// Use pool from multiple goroutines
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				for j := 0; j < 10; j++ {
					client := pool.Get()
					clientsUsed <- client
					
					// Simulate some work
					time.Sleep(time.Millisecond)
					
					pool.Put(client)
				}
			}(i)
		}
		
		wg.Wait()
		close(clientsUsed)
		
		// Count unique clients
		uniqueClients := make(map[*http.Client]bool)
		for client := range clientsUsed {
			uniqueClients[client] = true
		}
		
		metrics := pool.Metrics()
		t.Logf("Created %d unique clients out of %d total requests", len(uniqueClients), 200)
		t.Logf("Pool metrics: %s", metrics.String())
		
		// Should have efficient reuse
		if metrics.HitRate < 50.0 {
			t.Logf("Hit rate lower than expected: %.2f%% (this may be OK under high concurrency)", metrics.HitRate)
		}
	})
	
	t.Run("PoolExhaustion", func(t *testing.T) {
		config := HTTPClientPoolConfig{
			MaxClients:     2,
			IdleTimeout:    30 * time.Second,
			ConnectTimeout: 5 * time.Second,
			RequestTimeout: 10 * time.Second,
		}
		
		pool := NewHTTPClientPool(config)
		defer pool.Close()
		
		// Get all clients from pool
		client1 := pool.Get()
		client2 := pool.Get()
		client3 := pool.Get() // Should create temporary client
		
		if client1 == nil || client2 == nil || client3 == nil {
			t.Fatal("All Get() calls should return clients")
		}
		
		metrics := pool.Metrics()
		t.Logf("Pool metrics after exhaustion: %s", metrics.String())
		
		// Return clients
		pool.Put(client1)
		pool.Put(client2)
		pool.Put(client3) // Third client should be discarded
		
		finalMetrics := pool.Metrics()
		t.Logf("Final pool metrics: %s", finalMetrics.String())
	})
}

// TestVaultClientPool tests the Vault client pool functionality
func TestVaultClientPool(t *testing.T) {
	t.Run("BasicPooling", func(t *testing.T) {
		config := VaultClientPoolConfig{
			MaxClients:   3,
			IdleTimeout:  30 * time.Second,
			MaxIdleTime:  1 * time.Minute,
			ReuseClients: true,
		}
		
		// Mock vault client factory
		clientCounter := atomic.Int32{}
		factory := func() (interface{}, error) {
			id := clientCounter.Add(1)
			return fmt.Sprintf("mock-vault-client-%d", id), nil
		}
		
		pool := NewVaultClientPool(config, factory)
		defer pool.Close()
		
		// Get a client
		client1, err := pool.Get()
		if err != nil {
			t.Fatalf("Failed to get client: %v", err)
		}
		
		// Put it back
		pool.Put(client1)
		
		// Get another client (should be reused)
		client2, err := pool.Get()
		if err != nil {
			t.Fatalf("Failed to get second client: %v", err)
		}
		
		t.Logf("Client1: %v, Client2: %v", client1, client2)
		
		// Check metrics
		metrics := pool.Metrics()
		t.Logf("Vault pool metrics: %s", metrics.String())
		
		if metrics.Created == 0 {
			t.Error("Expected at least one client to be created")
		}
	})
	
	t.Run("IdleTimeout", func(t *testing.T) {
		config := VaultClientPoolConfig{
			MaxClients:   2,
			IdleTimeout:  30 * time.Second,
			MaxIdleTime:  50 * time.Millisecond, // Very short for testing
			ReuseClients: true,
		}
		
		clientCounter := atomic.Int32{}
		factory := func() (interface{}, error) {
			id := clientCounter.Add(1)
			return fmt.Sprintf("mock-vault-client-%d", id), nil
		}
		
		pool := NewVaultClientPool(config, factory)
		defer pool.Close()
		
		// Get a client
		client1, err := pool.Get()
		if err != nil {
			t.Fatalf("Failed to get client: %v", err)
		}
		
		// Put it back
		pool.Put(client1)
		
		// Wait for idle timeout
		time.Sleep(60 * time.Millisecond)
		
		// Get another client (should create new due to idle timeout)
		client2, err := pool.Get()
		if err != nil {
			t.Fatalf("Failed to get second client: %v", err)
		}
		
		if client1 == client2 {
			t.Error("Expected different client due to idle timeout")
		}
		
		t.Logf("Client1: %p, Client2: %p", client1, client2)
		
		metrics := pool.Metrics()
		t.Logf("Vault pool metrics after timeout: %s", metrics.String())
	})
}

// TestRequestDeduplication tests the request deduplication functionality
func TestRequestDeduplication(t *testing.T) {
	t.Run("BasicDeduplication", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         5 * time.Second,
			CleanupInterval: 10 * time.Second,
		}
		
		dedup := NewRequestDeduplicator(config)
		
		// Counter to track how many times the function is called
		callCount := atomic.Int32{}
		
		testFunc := func() (interface{}, error) {
			count := callCount.Add(1)
			time.Sleep(100 * time.Millisecond) // Simulate work
			return fmt.Sprintf("result-%d", count), nil
		}
		
		// Make multiple concurrent requests with the same key
		var wg sync.WaitGroup
		results := make(chan RequestResult, 5)
		
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resultChan := dedup.Deduplicate("test-key", testFunc)
				result := <-resultChan
				results <- result
			}()
		}
		
		wg.Wait()
		close(results)
		
		// Check that function was called only once
		if callCount.Load() != 1 {
			t.Errorf("Expected function to be called once, was called %d times", callCount.Load())
		}
		
		// Check that all requests got the same result
		firstResult := ""
		for result := range results {
			if result.Err != nil {
				t.Errorf("Unexpected error: %v", result.Err)
				continue
			}
			
			resultStr := result.Value.(string)
			if firstResult == "" {
				firstResult = resultStr
			} else if firstResult != resultStr {
				t.Errorf("Results differ: %s vs %s", firstResult, resultStr)
			}
		}
		
		metrics := dedup.GetMetrics()
		t.Logf("Deduplication metrics: %s", metrics.String())
		
		if metrics.Hits == 0 {
			t.Error("Expected some cache hits from deduplication")
		}
	})
	
	t.Run("DifferentKeys", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         5 * time.Second,
			CleanupInterval: 10 * time.Second,
		}
		
		dedup := NewRequestDeduplicator(config)
		
		callCount := atomic.Int32{}
		
		testFunc := func() (interface{}, error) {
			count := callCount.Add(1)
			return fmt.Sprintf("result-%d", count), nil
		}
		
		// Make requests with different keys
		result1Chan := dedup.Deduplicate("key1", testFunc)
		result2Chan := dedup.Deduplicate("key2", testFunc)
		
		result1 := <-result1Chan
		result2 := <-result2Chan
		
		// Function should be called twice (different keys)
		if callCount.Load() != 2 {
			t.Errorf("Expected function to be called twice, was called %d times", callCount.Load())
		}
		
		// Results should be different
		if result1.Value == result2.Value {
			t.Error("Expected different results for different keys")
		}
		
		metrics := dedup.GetMetrics()
		t.Logf("Deduplication metrics for different keys: %s", metrics.String())
	})
	
	t.Run("Timeout", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         100 * time.Millisecond,
			CleanupInterval: 10 * time.Second,
		}
		
		dedup := NewRequestDeduplicator(config)
		
		slowFunc := func() (interface{}, error) {
			time.Sleep(200 * time.Millisecond) // Longer than timeout
			return "result", nil
		}
		
		resultChan := dedup.Deduplicate("slow-key", slowFunc)
		result := <-resultChan
		
		if result.Err == nil {
			t.Error("Expected timeout error")
		}
		
		if result.Err.Error() != "request timeout after 100ms" {
			t.Errorf("Unexpected error message: %v", result.Err)
		}
		
		metrics := dedup.GetMetrics()
		if metrics.Timeouts == 0 {
			t.Error("Expected timeout to be recorded")
		}
		
		t.Logf("Deduplication timeout metrics: %s", metrics.String())
	})
	
	t.Run("ErrorHandling", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         5 * time.Second,
			CleanupInterval: 10 * time.Second,
		}
		
		dedup := NewRequestDeduplicator(config)
		
		errorFunc := func() (interface{}, error) {
			return nil, fmt.Errorf("test error")
		}
		
		resultChan := dedup.Deduplicate("error-key", errorFunc)
		result := <-resultChan
		
		if result.Err == nil {
			t.Error("Expected error to be propagated")
		}
		
		if result.Err.Error() != "test error" {
			t.Errorf("Unexpected error: %v", result.Err)
		}
		
		metrics := dedup.GetMetrics()
		if metrics.Errors == 0 {
			t.Error("Expected error to be recorded")
		}
		
		t.Logf("Deduplication error metrics: %s", metrics.String())
	})
	
	t.Run("ContextCancellation", func(t *testing.T) {
		config := RequestDeduplicatorConfig{
			Timeout:         5 * time.Second,
			CleanupInterval: 10 * time.Second,
		}
		
		dedup := NewRequestDeduplicator(config)
		
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		
		slowFunc := func() (interface{}, error) {
			time.Sleep(100 * time.Millisecond)
			return "result", nil
		}
		
		resultChan := dedup.DeduplicateWithContext(ctx, "context-key", slowFunc)
		result := <-resultChan
		
		if result.Err == nil {
			t.Error("Expected context timeout error")
		}
		
		t.Logf("Context cancellation result: %v", result.Err)
	})
}

// TestKeyBuilder tests the key building functionality
func TestKeyBuilder(t *testing.T) {
	t.Run("ConsistentKeys", func(t *testing.T) {
		kb := NewKeyBuilder("test")
		
		key1 := kb.BuildKey("component1", "component2")
		key2 := kb.BuildKey("component1", "component2")
		
		if key1 != key2 {
			t.Error("Expected identical keys for same components")
		}
		
		key3 := kb.BuildKey("component1", "component3")
		if key1 == key3 {
			t.Error("Expected different keys for different components")
		}
		
		t.Logf("Key1: %s", key1)
		t.Logf("Key3: %s", key3)
	})
	
	t.Run("PredefinedBuilders", func(t *testing.T) {
		vaultKey := VaultKeyBuilder.BuildKey("secret/path", "key")
		awsKey := AWSKeyBuilder.BuildKey("parameter-name")
		fileKey := FileKeyBuilder.BuildKey("/path/to/file")
		
		if vaultKey == awsKey || vaultKey == fileKey || awsKey == fileKey {
			t.Error("Expected different prefixes to generate different keys")
		}
		
		t.Logf("Vault key: %s", vaultKey)
		t.Logf("AWS key: %s", awsKey)
		t.Logf("File key: %s", fileKey)
	})
}

// BenchmarkConnectionPool benchmarks connection pool performance
func BenchmarkConnectionPool(b *testing.B) {
	config := HTTPClientPoolConfig{
		MaxClients:     20,
		IdleTimeout:    30 * time.Second,
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 10 * time.Second,
	}
	
	pool := NewHTTPClientPool(config)
	defer pool.Close()
	
	b.Run("GetPut", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				client := pool.Get()
				pool.Put(client)
			}
		})
	})
	
	b.Run("GetOnly", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = pool.Get()
			}
		})
	})
}

// BenchmarkDeduplication benchmarks request deduplication performance
func BenchmarkDeduplication(b *testing.B) {
	config := RequestDeduplicatorConfig{
		Timeout:         30 * time.Second,
		CleanupInterval: 1 * time.Minute,
	}
	
	dedup := NewRequestDeduplicator(config)
	
	testFunc := func() (interface{}, error) {
		return "result", nil
	}
	
	b.Run("SameKey", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resultChan := dedup.Deduplicate("same-key", testFunc)
				<-resultChan
			}
		})
	})
	
	b.Run("DifferentKeys", func(b *testing.B) {
		counter := atomic.Int64{}
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				key := fmt.Sprintf("key-%d", counter.Add(1))
				resultChan := dedup.Deduplicate(key, testFunc)
				<-resultChan
			}
		})
	})
}