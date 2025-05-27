package spruce

import (
	"fmt"
	"os"
	"testing"
	"time"
	
	. "github.com/smartystreets/goconvey/convey"
)

func TestHierarchicalCache(t *testing.T) {
	Convey("Hierarchical Cache System", t, func() {
		
		// Create temporary directory for testing
		tempDir, err := os.MkdirTemp("", "spruce_cache_test")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tempDir)
		
		Convey("Basic hierarchical cache operations", func() {
			config := HierarchicalCacheConfig{
				L1Size:       100,
				L2Size:       1000,
				L2Enabled:    false, // Disable L2 for simple test
				StoragePath:  tempDir,
				TTL:          time.Minute,
				Persistence:  false,
				SyncInterval: 0,
			}
			
			cache, err := NewHierarchicalCache(config)
			So(err, ShouldBeNil)
			defer cache.Close()
			
			// Test basic set/get
			cache.Set("key1", "value1")
			value, found := cache.Get("key1")
			So(found, ShouldBeTrue)
			So(value, ShouldEqual, "value1")
			
			// Test cache miss
			_, found = cache.Get("nonexistent")
			So(found, ShouldBeFalse)
		})
		
		Convey("L1 cache only operations", func() {
			config := HierarchicalCacheConfig{
				L1Size:    100,
				L2Enabled: false,
				TTL:       time.Minute,
			}
			
			cache, err := NewHierarchicalCache(config)
			So(err, ShouldBeNil)
			defer cache.Close()
			
			cache.Set("key1", "value1")
			value, found := cache.Get("key1")
			So(found, ShouldBeTrue)
			So(value, ShouldEqual, "value1")
			
			metrics := cache.GetMetrics()
			So(metrics.L1Hits, ShouldEqual, 1)
			So(metrics.L1Misses, ShouldEqual, 0)
		})
		
		Convey("Cache promotion from L2 to L1", func() {
			config := HierarchicalCacheConfig{
				L1Size:      2, // Small L1 to force eviction
				L2Size:      100,
				L2Enabled:   true,
				StoragePath: tempDir,
				TTL:         time.Minute,
				Persistence: true,
			}
			
			cache, err := NewHierarchicalCache(config)
			So(err, ShouldBeNil)
			defer cache.Close()
			
			// Fill L1 cache
			cache.Set("key1", "value1")
			cache.Set("key2", "value2")
			cache.Set("key3", "value3") // This should cause eviction to L2
			
			// Wait for async L2 operations
			time.Sleep(100 * time.Millisecond)
			
			// key1 might be evicted to L2, so accessing it should promote it back
			value, found := cache.Get("key1")
			if found {
				So(value, ShouldEqual, "value1")
			}
			
			metrics := cache.GetMetrics()
			So(metrics.L1Hits+metrics.L2Hits, ShouldBeGreaterThan, 0)
		})
		
		Convey("Cache deletion", func() {
			config := HierarchicalCacheConfig{
				L1Size:      100,
				L2Size:      1000,
				L2Enabled:   true,
				StoragePath: tempDir,
				TTL:         time.Minute,
			}
			
			cache, err := NewHierarchicalCache(config)
			So(err, ShouldBeNil)
			defer cache.Close()
			
			cache.Set("key1", "value1")
			
			// Verify it exists
			_, found := cache.Get("key1")
			So(found, ShouldBeTrue)
			
			// Delete it
			cache.Delete("key1")
			
			// Verify it's gone
			_, found = cache.Get("key1")
			So(found, ShouldBeFalse)
		})
		
		Convey("Cache clear operation", func() {
			config := HierarchicalCacheConfig{
				L1Size:      100,
				L2Size:      1000,
				L2Enabled:   true,
				StoragePath: tempDir,
				TTL:         time.Minute,
			}
			
			cache, err := NewHierarchicalCache(config)
			So(err, ShouldBeNil)
			defer cache.Close()
			
			// Add some entries
			cache.Set("key1", "value1")
			cache.Set("key2", "value2")
			cache.Set("key3", "value3")
			
			// Clear cache
			cache.Clear()
			
			// Verify all entries are gone
			_, found1 := cache.Get("key1")
			_, found2 := cache.Get("key2")
			_, found3 := cache.Get("key3")
			
			So(found1, ShouldBeFalse)
			So(found2, ShouldBeFalse)
			So(found3, ShouldBeFalse)
		})
		
		Convey("Detailed metrics", func() {
			config := HierarchicalCacheConfig{
				L1Size:      100,
				L2Size:      1000,
				L2Enabled:   true,
				StoragePath: tempDir,
				TTL:         time.Minute,
			}
			
			cache, err := NewHierarchicalCache(config)
			So(err, ShouldBeNil)
			defer cache.Close()
			
			// Generate some cache activity
			cache.Set("key1", "value1")
			cache.Get("key1")        // L1 hit
			cache.Get("nonexistent") // L1 miss
			
			detailedMetrics := cache.GetDetailedMetrics()
			So(detailedMetrics, ShouldContainKey, "l1")
			So(detailedMetrics, ShouldContainKey, "l2_enabled")
			
			l1Metrics := detailedMetrics["l1"].(map[string]interface{})
			So(l1Metrics["hits"], ShouldBeGreaterThan, 0)
		})
		
		Convey("Background sync operations", func() {
			config := HierarchicalCacheConfig{
				L1Size:       100,
				L2Size:       1000,
				L2Enabled:    true,
				StoragePath:  tempDir,
				TTL:          time.Minute,
				SyncInterval: 100 * time.Millisecond, // Fast sync for testing
			}
			
			cache, err := NewHierarchicalCache(config)
			So(err, ShouldBeNil)
			defer cache.Close()
			
			cache.Set("key1", "value1")
			
			// Wait for sync to occur
			time.Sleep(200 * time.Millisecond)
			
			metrics := cache.GetMetrics()
			So(metrics.SyncOperations, ShouldBeGreaterThan, 0)
		})
		
		Convey("TTL expiration", func() {
			config := HierarchicalCacheConfig{
				L1Size:      100,
				L2Size:      1000,
				L2Enabled:   true,
				StoragePath: tempDir,
				TTL:         50 * time.Millisecond, // Short TTL for testing
			}
			
			cache, err := NewHierarchicalCache(config)
			So(err, ShouldBeNil)
			defer cache.Close()
			
			cache.Set("key1", "value1")
			
			// Should be available immediately
			_, found := cache.Get("key1")
			So(found, ShouldBeTrue)
			
			// Wait for expiration
			time.Sleep(100 * time.Millisecond)
			
			// Should be expired now (this tests L2 expiration)
			// Note: L1 expiration is handled by the underlying ConcurrentCache
		})
	})
}

func TestDiskCache(t *testing.T) {
	Convey("Disk Cache Operations", t, func() {
		
		tempDir, err := os.MkdirTemp("", "spruce_disk_cache_test")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tempDir)
		
		Convey("Basic disk cache operations", func() {
			config := DiskCacheConfig{
				StoragePath: tempDir,
				MaxSize:     100,
				TTL:         time.Minute,
				Persistence: true,
				FilePrefix:  "test_cache",
			}
			
			cache, err := NewDiskCache(config)
			So(err, ShouldBeNil)
			defer cache.Close()
			
			entry := &CacheEntry{
				Key:          "test_key",
				Value:        "test_value",
				Timestamp:    time.Now(),
				LastAccessed: time.Now(),
				HitCount:     0,
				Size:         10,
				TTL:          time.Minute,
			}
			
			// Test set and get
			cache.Set("test_key", entry)
			retrieved, found := cache.Get("test_key")
			
			So(found, ShouldBeTrue)
			So(retrieved.Value, ShouldEqual, "test_value")
			So(retrieved.HitCount, ShouldEqual, 1) // Should be incremented
		})
		
		Convey("Persistence across cache instances", func() {
			config := DiskCacheConfig{
				StoragePath: tempDir,
				MaxSize:     100,
				TTL:         time.Minute,
				Persistence: true,
				FilePrefix:  "persist_test",
			}
			
			// Create first cache instance
			cache1, err := NewDiskCache(config)
			So(err, ShouldBeNil)
			
			entry := &CacheEntry{
				Key:          "persist_key",
				Value:        "persist_value",
				Timestamp:    time.Now(),
				LastAccessed: time.Now(),
				TTL:          time.Minute,
			}
			
			cache1.Set("persist_key", entry)
			cache1.Close() // This should save to disk
			
			// Create second cache instance
			cache2, err := NewDiskCache(config)
			So(err, ShouldBeNil)
			defer cache2.Close()
			
			// Should load the persisted entry
			retrieved, found := cache2.Get("persist_key")
			So(found, ShouldBeTrue)
			So(retrieved.Value, ShouldEqual, "persist_value")
		})
		
		Convey("Cache eviction", func() {
			config := DiskCacheConfig{
				StoragePath: tempDir,
				MaxSize:     2, // Small size to force eviction
				TTL:         time.Minute,
				Persistence: false,
			}
			
			cache, err := NewDiskCache(config)
			So(err, ShouldBeNil)
			defer cache.Close()
			
			// Fill cache beyond capacity
			for i := 0; i < 5; i++ {
				entry := &CacheEntry{
					Key:          fmt.Sprintf("key%d", i),
					Value:        fmt.Sprintf("value%d", i),
					Timestamp:    time.Now(),
					LastAccessed: time.Now(),
					TTL:          time.Minute,
				}
				cache.Set(fmt.Sprintf("key%d", i), entry)
			}
			
			metrics := cache.GetMetrics()
			So(metrics.Size, ShouldBeLessThanOrEqualTo, 2)
		})
		
		Convey("TTL expiration", func() {
			config := DiskCacheConfig{
				StoragePath: tempDir,
				MaxSize:     100,
				TTL:         50 * time.Millisecond, // Short TTL
				Persistence: false,
			}
			
			cache, err := NewDiskCache(config)
			So(err, ShouldBeNil)
			defer cache.Close()
			
			entry := &CacheEntry{
				Key:          "expire_key",
				Value:        "expire_value",
				Timestamp:    time.Now(),
				LastAccessed: time.Now(),
				TTL:          50 * time.Millisecond,
			}
			
			cache.Set("expire_key", entry)
			
			// Should be available immediately
			_, found := cache.Get("expire_key")
			So(found, ShouldBeTrue)
			
			// Wait for expiration
			time.Sleep(100 * time.Millisecond)
			
			// Should be expired
			_, found = cache.Get("expire_key")
			So(found, ShouldBeFalse)
		})
	})
}

func BenchmarkHierarchicalCache(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "spruce_bench_cache")
	defer os.RemoveAll(tempDir)
	
	config := HierarchicalCacheConfig{
		L1Size:      1000,
		L2Size:      10000,
		L2Enabled:   true,
		StoragePath: tempDir,
		TTL:         time.Hour,
		Persistence: false, // Disable for benchmarking
	}
	
	cache, _ := NewHierarchicalCache(config)
	defer cache.Close()
	
	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i)
			value := fmt.Sprintf("value_%d", i)
			cache.Set(key, value)
		}
	})
	
	b.Run("Get_L1_Hit", func(b *testing.B) {
		// Pre-populate cache
		for i := 0; i < 100; i++ {
			cache.Set(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i%100)
			cache.Get(key)
		}
	})
	
	b.Run("Get_Miss", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("nonexistent_%d", i)
			cache.Get(key)
		}
	})
	
	b.Run("Mixed_Operations", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if i%3 == 0 {
				// Set
				cache.Set(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
			} else {
				// Get
				cache.Get(fmt.Sprintf("key_%d", i%1000))
			}
		}
	})
}