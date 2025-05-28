package internal

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wayneeseguin/graft/internal/cache"
)

// TestWorkerPool tests the worker pool implementation
func TestWorkerPool(t *testing.T) {
	t.Run("BasicExecution", func(t *testing.T) {
		pool := NewWorkerPool(WorkerPoolConfig{
			Name:      "test",
			Workers:   4,
			QueueSize: 10,
		})
		defer pool.Shutdown()
		
		// Submit tasks
		for i := 0; i < 10; i++ {
			task := &testTask{
				id:    fmt.Sprintf("task-%d", i),
				value: i,
		}
			err := pool.Submit(task)
			if err != nil {
				t.Fatalf("Failed to submit task: %v", err)
			}
		}
		
		// Collect results
		results := make(map[string]int)
		for i := 0; i < 10; i++ {
			select {
			case result := <-pool.Results():
				if result.Err != nil {
					t.Fatalf("Task %s failed: %v", result.ID, result.Err)
				}
				results[result.ID] = result.Value.(int)
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for results")
			}
		}
		
		// Verify all tasks completed
		if len(results) != 10 {
			t.Fatalf("Expected 10 results, got %d", len(results))
		}
		
		// Verify correct values
		for i := 0; i < 10; i++ {
			id := fmt.Sprintf("task-%d", i)
			if results[id] != i*2 {
				t.Fatalf("Task %s: expected %d, got %d", id, i*2, results[id])
			}
		}
	})
	
	t.Run("RateLimiting", func(t *testing.T) {
		pool := NewWorkerPool(WorkerPoolConfig{
			Name:      "test-rate-limited",
			Workers:   4,
			QueueSize: 100,
			RateLimit: 10, // 10 per second
		})
		defer pool.Shutdown()
		
		start := time.Now()
		
		// Submit 20 tasks
		for i := 0; i < 20; i++ {
			task := &testTask{
				id:    fmt.Sprintf("task-%d", i),
				value: i,
			}
			err := pool.Submit(task)
			if err != nil {
				t.Fatalf("Failed to submit task: %v", err)
			}
		}
		
		// Collect all results
		for i := 0; i < 20; i++ {
			<-pool.Results()
		}
		
		elapsed := time.Since(start)
		// With 10/second rate limit and initial bucket of 10 tokens,
		// first 10 tasks complete immediately, then 10 more at 10/second
		// So minimum time is about 1 second for the remaining 10 tasks
		if elapsed < 900*time.Millisecond {
			t.Fatalf("Tasks completed too quickly: %v (expected ~1s)", elapsed)
		}
		if elapsed > 3000*time.Millisecond {
			t.Fatalf("Tasks took too long: %v (expected <3s)", elapsed)
		}
	})
	
	t.Run("ConcurrentAccess", func(t *testing.T) {
		pool := NewWorkerPool(WorkerPoolConfig{
			Name:      "test-concurrent",
			Workers:   8,
			QueueSize: 100,
		})
		defer pool.Shutdown()
		
		var wg sync.WaitGroup
		var submitted atomic.Uint64
		var completed atomic.Uint64
		
		// Submit tasks from multiple goroutines
		for g := 0; g < 10; g++ {
			wg.Add(1)
			go func(goroutine int) {
				defer wg.Done()
				for i := 0; i < 10; i++ {
					task := &testTask{
						id:    fmt.Sprintf("g%d-t%d", goroutine, i),
						value: i,
					}
					err := pool.Submit(task)
					if err != nil {
						t.Errorf("Failed to submit task: %v", err)
						return
					}
					submitted.Add(1)
				}
			}(g)
		}
		
		// Collect results concurrently
		wg.Add(1)
		go func() {
			defer wg.Done()
			for completed.Load() < 100 {
				select {
				case <-pool.Results():
					completed.Add(1)
				case <-time.After(10 * time.Second):
					t.Error("Timeout collecting results")
					return
				}
			}
		}()
		
		wg.Wait()
		
		if submitted.Load() != 100 {
			t.Fatalf("Expected 100 submitted tasks, got %d", submitted.Load())
		}
		if completed.Load() != 100 {
			t.Fatalf("Expected 100 completed tasks, got %d", completed.Load())
		}
	})
}

// TestConcurrentCache tests the concurrent cache implementation
func TestConcurrentCache(t *testing.T) {
	t.Run("BasicOperations", func(t *testing.T) {
		c := cache.NewConcurrentCache(cache.CacheConfig{
			Shards:   4,
			Capacity: 100,
			TTL:      0,
		})
		
		// Set and get
		c.Set("key1", "value1")
		value, found := c.Get("key1")
		if !found {
			t.Fatal("Expected to find key1")
		}
		if value != "value1" {
			t.Fatalf("Expected value1, got %v", value)
		}
		
		// Miss
		_, found = c.Get("key2")
		if found {
			t.Fatal("Expected not to find key2")
		}
		
		// Delete
		deleted := c.Delete("key1")
		if !deleted {
			t.Fatal("Expected to delete key1")
		}
		
		_, found = c.Get("key1")
		if found {
			t.Fatal("Expected not to find deleted key1")
		}
	})
	
	t.Run("TTLExpiration", func(t *testing.T) {
		c := cache.NewConcurrentCache(cache.CacheConfig{
			Shards:   4,
			Capacity: 100,
			TTL:      100 * time.Millisecond,
		})
		
		c.Set("key1", "value1")
		
		// Should exist immediately
		_, found := c.Get("key1")
		if !found {
			t.Fatal("Expected to find key1")
		}
		
		// Wait for expiration
		time.Sleep(150 * time.Millisecond)
		
		// Should be expired
		_, found = c.Get("key1")
		if found {
			t.Fatal("Expected key1 to be expired")
		}
	})
	
	t.Run("ConcurrentAccess", func(t *testing.T) {
		c := cache.NewConcurrentCache(cache.CacheConfig{
			Shards:   16,
			Capacity: 1000,
			TTL:      0,
		})
		
		var wg sync.WaitGroup
		numGoroutines := 100
		numOperations := 1000
		
		// Track operations
		var sets atomic.Uint64
		var gets atomic.Uint64
		var deletes atomic.Uint64
		
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				for j := 0; j < numOperations; j++ {
					key := fmt.Sprintf("key-%d-%d", id, j%100)
					
					switch j % 3 {
					case 0:
						c.Set(key, j)
						sets.Add(1)
					case 1:
						c.Get(key)
						gets.Add(1)
					case 2:
						c.Delete(key)
						deletes.Add(1)
					}
				}
			}(i)
		}
		
		wg.Wait()
		
		totalOps := sets.Load() + gets.Load() + deletes.Load()
		expectedOps := uint64(numGoroutines * numOperations)
		if totalOps != expectedOps {
			t.Fatalf("Expected %d operations, got %d", expectedOps, totalOps)
		}
		
		// Check metrics
		metrics := c.Metrics()
		if metrics.Sets != sets.Load() {
			t.Fatalf("Metrics mismatch: expected %d sets, got %d", sets.Load(), metrics.Sets)
		}
	})
	
	t.Run("LRUEviction", func(t *testing.T) {
		c := cache.NewConcurrentCache(cache.CacheConfig{
			Shards:   1, // Single shard for predictable behavior
			Capacity: 3,
			TTL:      0,
		})
		
		// Fill cache
		c.Set("key1", "value1")
		c.Set("key2", "value2")
		c.Set("key3", "value3")
		
		// Access key1 and key2 multiple times to increase hit count
		for i := 0; i < 5; i++ {
			c.Get("key1")
			c.Get("key2")
		}
		
		// Add key4, should evict key3 (least recently used)
		c.Set("key4", "value4")
		
		// Check that key3 was evicted
		_, found := c.Get("key3")
		if found {
			t.Fatal("Expected key3 to be evicted")
		}
		
		// Check that other keys still exist
		for _, key := range []string{"key1", "key2", "key4"} {
			_, found := c.Get(key)
			if !found {
				t.Fatalf("Expected to find %s", key)
			}
		}
	})
}

// TestRaceConditions runs tests with -race flag
func TestRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition tests in short mode")
	}
	
	t.Run("WorkerPoolRace", func(t *testing.T) {
		pool := NewWorkerPool(WorkerPoolConfig{
			Name:      "race-test",
			Workers:   10,
			QueueSize: 1000,
		})
		
		var wg sync.WaitGroup
		
		// Submit tasks concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					task := &testTask{
						id:    fmt.Sprintf("task-%d-%d", id, j),
						value: j,
					}
					pool.Submit(task)
				}
			}(i)
		}
		
		// Read results concurrently
		resultsDone := make(chan bool)
		go func() {
			count := 0
			timeout := time.After(10 * time.Second)
			for count < 1000 {
				select {
				case <-pool.Results():
					count++
				case <-timeout:
					t.Errorf("Timeout waiting for results, got %d/1000", count)
					resultsDone <- false
					return
				}
			}
			resultsDone <- true
		}()
		
		// Read metrics concurrently
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_ = pool.Metrics()
				time.Sleep(time.Millisecond)
			}
		}()
		
		// Wait for task submission to complete
		wg.Wait()
		
		// Wait for all results to be read
		success := <-resultsDone
		
		// Now safe to shutdown
		pool.Shutdown()
		
		if !success {
			t.Fatal("Failed to read all results")
		}
	})
	
	t.Run("CacheRace", func(t *testing.T) {
		c := cache.NewConcurrentCache(cache.CacheConfig{
			Shards:   16,
			Capacity: 1000,
			TTL:      time.Second,
		})
		
		var wg sync.WaitGroup
		
		// Concurrent writes
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					key := fmt.Sprintf("key-%d", j)
					c.Set(key, id)
				}
			}(i)
		}
		
		// Concurrent reads
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					key := fmt.Sprintf("key-%d", j)
					c.Get(key)
				}
			}()
		}
		
		// Concurrent deletes
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key-%d", j)
				c.Delete(key)
			}
		}()
		
		// Concurrent metrics
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_ = c.Metrics()
				_ = c.Size()
			}
		}()
		
		wg.Wait()
	})
}

// testTask is a simple task implementation for testing
type testTask struct {
	id    string
	value int
	delay time.Duration
}

func (t *testTask) ID() string {
	return t.id
}

func (t *testTask) Execute(ctx context.Context) (interface{}, error) {
	if t.delay > 0 {
		select {
		case <-time.After(t.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	// Simple computation
	return t.value * 2, nil
}

// BenchmarkConcurrentCache benchmarks the concurrent cache
func BenchmarkConcurrentCache(b *testing.B) {
	c := cache.NewConcurrentCache(cache.CacheConfig{
		Shards:   16,
		Capacity: 10000,
		TTL:      0,
	})
	
	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		c.Set(fmt.Sprintf("key-%d", i), i)
	}
	
	b.ResetTimer()
	
	b.Run("Get", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key-%d", i%1000)
				c.Get(key)
				i++
			}
		})
	})
	
	b.Run("Set", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key-%d", i%10000)
				c.Set(key, i)
				i++
			}
		})
	})
	
	b.Run("Mixed", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key-%d", i%1000)
				if i%10 == 0 {
					c.Set(key, i)
				} else {
					c.Get(key)
				}
				i++
			}
		})
	})
}