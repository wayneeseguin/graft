package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCacheAnalytics(t *testing.T) {
	Convey("CacheAnalytics", t, func() {
		analytics := NewCacheAnalytics(1 * time.Hour)

		Convey("should initialize with empty state", func() {
			stats := analytics.GetAllStats()
			So(len(stats), ShouldEqual, 0)

			hotKeys := analytics.GetHotKeys()
			So(len(hotKeys), ShouldEqual, 0)

			score := analytics.GetEffectivenessScore()
			So(score, ShouldEqual, 0)
		})

		Convey("should record hits correctly", func() {
			analytics.RecordHit("test_cache", "key1")
			analytics.RecordHit("test_cache", "key1")
			analytics.RecordHit("test_cache", "key2")

			stats, exists := analytics.GetCacheStats("test_cache")
			So(exists, ShouldBeTrue)
			So(stats.Name, ShouldEqual, "test_cache")
			So(stats.Hits, ShouldEqual, 3)
			So(stats.Misses, ShouldEqual, 0)
			So(stats.HitRate, ShouldEqual, 1.0)
		})

		Convey("should record misses correctly", func() {
			loadTime1 := 50 * time.Millisecond
			loadTime2 := 100 * time.Millisecond

			analytics.RecordMiss("test_cache", "key1", loadTime1)
			analytics.RecordMiss("test_cache", "key2", loadTime2)

			stats, exists := analytics.GetCacheStats("test_cache")
			So(exists, ShouldBeTrue)
			So(stats.Hits, ShouldEqual, 0)
			So(stats.Misses, ShouldEqual, 2)
			So(stats.HitRate, ShouldEqual, 0.0)
			So(stats.TotalLoadTime, ShouldEqual, loadTime1+loadTime2)
			So(stats.AvgLoadTime, ShouldEqual, (loadTime1+loadTime2)/2)
		})

		Convey("should calculate hit rate correctly", func() {
			analytics.RecordHit("test_cache", "key1")
			analytics.RecordHit("test_cache", "key1")
			analytics.RecordMiss("test_cache", "key2", 10*time.Millisecond)

			stats, exists := analytics.GetCacheStats("test_cache")
			So(exists, ShouldBeTrue)
			So(stats.Hits, ShouldEqual, 2)
			So(stats.Misses, ShouldEqual, 1)
			So(stats.HitRate, ShouldAlmostEqual, 2.0/3.0, 0.001)
		})

		Convey("should record evictions correctly", func() {
			// First create the cache by recording a hit
			analytics.RecordHit("test_cache", "key1")
			
			analytics.RecordEviction("test_cache", 5, "capacity")
			analytics.RecordEviction("test_cache", 2, "ttl")
			analytics.RecordEviction("test_cache", 1, "capacity")

			stats, exists := analytics.GetCacheStats("test_cache")
			So(exists, ShouldBeTrue)
			So(stats.Evictions, ShouldEqual, 8)

			reasons := stats.EvictionReasons
			So(reasons["capacity"], ShouldEqual, 6)
			So(reasons["ttl"], ShouldEqual, 2)
		})

		Convey("should update size correctly", func() {
			analytics.UpdateSize("test_cache", 150, 200)

			stats, exists := analytics.GetCacheStats("test_cache")
			So(exists, ShouldBeTrue)
			So(stats.Size, ShouldEqual, 150)
			So(stats.MaxSize, ShouldEqual, 200)
			So(stats.FillRate, ShouldEqual, 0.75)
		})

		Convey("should handle multiple caches", func() {
			analytics.RecordHit("cache1", "key1")
			analytics.RecordHit("cache2", "key1")
			analytics.RecordMiss("cache1", "key2", 10*time.Millisecond)

			allStats := analytics.GetAllStats()
			So(len(allStats), ShouldEqual, 2)

			// Should be sorted by total operations (hits + misses)
			So(allStats[0].Name, ShouldEqual, "cache1") // 2 operations
			So(allStats[1].Name, ShouldEqual, "cache2") // 1 operation
		})

		Convey("should return non-existent cache correctly", func() {
			_, exists := analytics.GetCacheStats("non_existent")
			So(exists, ShouldBeFalse)
		})
	})
}

func TestHotKeyTracker(t *testing.T) {
	Convey("HotKeyTracker", t, func() {
		tracker := NewHotKeyTracker(1*time.Hour, 10)

		Convey("should initialize correctly", func() {
			So(tracker.window, ShouldEqual, 1*time.Hour)
			So(tracker.topN, ShouldEqual, 10)
			So(len(tracker.keyAccess), ShouldEqual, 0)
		})

		Convey("should track key access", func() {
			tracker.recordAccess("key1", true, 0)
			tracker.recordAccess("key1", false, 50*time.Millisecond)
			tracker.recordAccess("key2", true, 0)

			hotKeys := tracker.GetHotKeys()
			So(len(hotKeys), ShouldEqual, 2)

			// Should be sorted by access count
			So(hotKeys[0].Key, ShouldEqual, "key1")
			So(hotKeys[0].AccessCount, ShouldEqual, 2)
			So(hotKeys[1].Key, ShouldEqual, "key2")
			So(hotKeys[1].AccessCount, ShouldEqual, 1)
		})

		Convey("should calculate hit rates correctly", func() {
			// key1: 2 hits, 1 miss -> 2/3 hit rate
			tracker.recordAccess("key1", true, 0)
			tracker.recordAccess("key1", true, 0)
			tracker.recordAccess("key1", false, 30*time.Millisecond)

			hotKeys := tracker.GetHotKeys()
			So(len(hotKeys), ShouldEqual, 1)
			So(hotKeys[0].HitRate, ShouldAlmostEqual, 2.0/3.0, 0.001)
		})

		Convey("should calculate average load time correctly", func() {
			tracker.recordAccess("key1", false, 100*time.Millisecond)
			tracker.recordAccess("key1", false, 200*time.Millisecond)

			hotKeys := tracker.GetHotKeys()
			So(len(hotKeys), ShouldEqual, 1)
			So(hotKeys[0].AvgLoadTime, ShouldEqual, 150*time.Millisecond)
		})

		Convey("should limit to topN keys", func() {
			tracker = NewHotKeyTracker(1*time.Hour, 3)

			// Add 5 keys with different access counts
			for i := 0; i < 5; i++ {
				key := fmt.Sprintf("key%d", i)
				for j := 0; j <= i; j++ {
					tracker.recordAccess(key, true, 0)
				}
			}

			hotKeys := tracker.GetHotKeys()
			So(len(hotKeys), ShouldEqual, 3)

			// Should return top 3 by access count
			So(hotKeys[0].Key, ShouldEqual, "key4") // 5 accesses
			So(hotKeys[1].Key, ShouldEqual, "key3") // 4 accesses
			So(hotKeys[2].Key, ShouldEqual, "key2") // 3 accesses
		})

		Convey("should filter old entries by time window", func() {
			tracker = NewHotKeyTracker(50*time.Millisecond, 10)

			tracker.recordAccess("key1", true, 0)
			time.Sleep(25 * time.Millisecond)
			tracker.recordAccess("key2", true, 0)
			time.Sleep(60 * time.Millisecond) // Wait longer than window

			hotKeys := tracker.GetHotKeys()
			// Both keys should be filtered out due to time window
			So(len(hotKeys), ShouldEqual, 0)
		})
	})
}

func TestCacheAnalyticsConcurrency(t *testing.T) {
	Convey("CacheAnalytics concurrency", t, func() {
		analytics := NewCacheAnalytics(1 * time.Hour)

		Convey("should handle concurrent access safely", func() {
			var wg sync.WaitGroup
			goroutines := 100
			operationsPerGoroutine := 100

			// Concurrent hits
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					cacheName := fmt.Sprintf("cache_%d", id%10)
					for j := 0; j < operationsPerGoroutine; j++ {
						key := fmt.Sprintf("key_%d_%d", id, j)
						analytics.RecordHit(cacheName, key)
					}
				}(i)
			}

			// Concurrent misses
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					cacheName := fmt.Sprintf("cache_%d", id%10)
					for j := 0; j < operationsPerGoroutine; j++ {
						key := fmt.Sprintf("miss_key_%d_%d", id, j)
						analytics.RecordMiss(cacheName, key, time.Duration(j)*time.Microsecond)
					}
				}(i)
			}

			// Concurrent evictions
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					cacheName := fmt.Sprintf("cache_%d", id%10)
					analytics.RecordEviction(cacheName, int64(id), "capacity")
				}(i)
			}

			// Concurrent size updates
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					cacheName := fmt.Sprintf("cache_%d", id%10)
					analytics.UpdateSize(cacheName, int64(id*100), 1000)
				}(i)
			}

			wg.Wait()

			// Verify data integrity
			allStats := analytics.GetAllStats()
			So(len(allStats), ShouldEqual, 10) // 10 different cache names

			totalHits := int64(0)
			totalMisses := int64(0)
			for _, stats := range allStats {
				totalHits += stats.Hits
				totalMisses += stats.Misses
				So(stats.Hits, ShouldEqual, operationsPerGoroutine*goroutines/10)
				So(stats.Misses, ShouldEqual, operationsPerGoroutine*goroutines/10)
			}

			So(totalHits, ShouldEqual, operationsPerGoroutine*goroutines)
			So(totalMisses, ShouldEqual, operationsPerGoroutine*goroutines)
		})
	})
}

func TestCacheEffectivenessScore(t *testing.T) {
	Convey("Cache effectiveness score", t, func() {
		analytics := NewCacheAnalytics(1 * time.Hour)

		Convey("should return 0 for empty analytics", func() {
			score := analytics.GetEffectivenessScore()
			So(score, ShouldEqual, 0)
		})

		Convey("should calculate score for perfect cache", func() {
			// Perfect cache: 100% hit rate, 80% fill rate, no evictions, fast load time
			analytics.RecordHit("perfect_cache", "key1")
			analytics.RecordHit("perfect_cache", "key1")
			analytics.RecordHit("perfect_cache", "key1")
			analytics.UpdateSize("perfect_cache", 80, 100) // 80% fill rate

			score := analytics.GetEffectivenessScore()
			So(score, ShouldBeGreaterThan, 0.8) // Should be high
		})

		Convey("should calculate score for poor cache", func() {
			// Poor cache: low hit rate, many evictions, slow load time
			analytics.RecordMiss("poor_cache", "key1", 500*time.Millisecond) // Slow
			analytics.RecordMiss("poor_cache", "key2", 600*time.Millisecond) // Slow
			analytics.RecordHit("poor_cache", "key3")
			analytics.RecordEviction("poor_cache", 100, "capacity") // Many evictions
			analytics.UpdateSize("poor_cache", 100, 100) // 100% full

			score := analytics.GetEffectivenessScore()
			So(score, ShouldBeLessThan, 0.5) // Should be low
		})

		Convey("should weight by operations volume", func() {
			// High volume good cache
			for i := 0; i < 1000; i++ {
				analytics.RecordHit("high_volume", fmt.Sprintf("key%d", i))
			}
			analytics.UpdateSize("high_volume", 80, 100)

			// Low volume poor cache
			analytics.RecordMiss("low_volume", "key1", 1*time.Second)
			analytics.RecordEviction("low_volume", 10, "capacity")

			score := analytics.GetEffectivenessScore()
			// Should be dominated by the high-volume good cache
			So(score, ShouldBeGreaterThan, 0.7)
		})
	})
}

func TestCacheAnalyticsReport(t *testing.T) {
	Convey("Cache analytics report", t, func() {
		analytics := NewCacheAnalytics(1 * time.Hour)

		Convey("should generate empty report", func() {
			report := analytics.GenerateReport()
			So(report, ShouldNotBeNil)
			So(len(report.CacheStats), ShouldEqual, 0)
			So(len(report.HotKeys), ShouldEqual, 0)
			So(report.TotalHits, ShouldEqual, 0)
			So(report.TotalMisses, ShouldEqual, 0)
			So(report.OverallHitRate, ShouldEqual, 0)
			So(report.EffectivenessScore, ShouldEqual, 0)
		})

		Convey("should generate comprehensive report", func() {
			// Add some data
			analytics.RecordHit("cache1", "key1")
			analytics.RecordHit("cache1", "key1")
			analytics.RecordMiss("cache1", "key2", 50*time.Millisecond)
			analytics.RecordEviction("cache1", 3, "capacity")
			analytics.UpdateSize("cache1", 75, 100)

			analytics.RecordHit("cache2", "key3")
			analytics.RecordMiss("cache2", "key4", 100*time.Millisecond)
			analytics.UpdateSize("cache2", 50, 100)

			report := analytics.GenerateReport()

			So(report, ShouldNotBeNil)
			So(len(report.CacheStats), ShouldEqual, 2)
			So(len(report.HotKeys), ShouldBeGreaterThan, 0)

			// Check totals
			So(report.TotalHits, ShouldEqual, 3)
			So(report.TotalMisses, ShouldEqual, 2)
			So(report.TotalEvictions, ShouldEqual, 3)
			So(report.TotalSize, ShouldEqual, 125)
			So(report.TotalMaxSize, ShouldEqual, 200)
			So(report.OverallHitRate, ShouldEqual, 0.6)

			// Check that effectiveness score is calculated
			So(report.EffectivenessScore, ShouldBeGreaterThan, 0)

			// Verify time fields
			So(report.GeneratedAt, ShouldHappenWithin, 1*time.Second, time.Now())
			So(report.AnalyticsPeriod, ShouldBeGreaterThan, 0)
		})
	})
}

func TestGlobalCacheAnalytics(t *testing.T) {
	Convey("Global cache analytics", t, func() {
		// Reset global state for testing
		globalCacheAnalytics = nil
		cacheAnalyticsOnce = sync.Once{}

		Convey("should initialize global analytics", func() {
			InitializeCacheAnalytics()
			analytics := GetCacheAnalytics()

			So(analytics, ShouldNotBeNil)
			So(analytics.window, ShouldEqual, 1*time.Hour)
		})

		Convey("should return same instance", func() {
			analytics1 := GetCacheAnalytics()
			analytics2 := GetCacheAnalytics()

			So(analytics1, ShouldEqual, analytics2)
		})

		Convey("should initialize on first access", func() {
			// Reset again
			globalCacheAnalytics = nil
			cacheAnalyticsOnce = sync.Once{}

			analytics := GetCacheAnalytics()
			So(analytics, ShouldNotBeNil)
		})
	})
}

func TestCacheStatsCopyEvictionReasons(t *testing.T) {
	Convey("CacheStats copyEvictionReasons", t, func() {
		stats := &CacheStats{
			evictionReasons: map[string]int64{
				"capacity": 10,
				"ttl":      5,
				"manual":   2,
			},
		}

		copy := stats.copyEvictionReasons()

		Convey("should create independent copy", func() {
			So(len(copy), ShouldEqual, 3)
			So(copy["capacity"], ShouldEqual, 10)
			So(copy["ttl"], ShouldEqual, 5)
			So(copy["manual"], ShouldEqual, 2)

			// Modify original
			stats.evictionReasons["capacity"] = 20
			stats.evictionReasons["new"] = 1

			// Copy should be unchanged
			So(copy["capacity"], ShouldEqual, 10)
			So(copy["new"], ShouldEqual, 0)
		})
	})
}

func BenchmarkCacheAnalytics(b *testing.B) {
	analytics := NewCacheAnalytics(1 * time.Hour)

	b.Run("RecordHit", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			analytics.RecordHit("test_cache", fmt.Sprintf("key_%d", i%1000))
		}
	})

	b.Run("RecordMiss", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			analytics.RecordMiss("test_cache", fmt.Sprintf("key_%d", i%1000), 10*time.Microsecond)
		}
	})

	b.Run("GetAllStats", func(b *testing.B) {
		// Pre-populate with some data
		for i := 0; i < 1000; i++ {
			analytics.RecordHit("test_cache", fmt.Sprintf("key_%d", i))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			analytics.GetAllStats()
		}
	})

	b.Run("GetHotKeys", func(b *testing.B) {
		// Pre-populate with some data
		for i := 0; i < 1000; i++ {
			analytics.RecordHit("test_cache", fmt.Sprintf("key_%d", i%100))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			analytics.GetHotKeys()
		}
	})

	b.Run("GenerateReport", func(b *testing.B) {
		// Pre-populate with some data
		for i := 0; i < 1000; i++ {
			analytics.RecordHit("test_cache", fmt.Sprintf("key_%d", i))
			if i%10 == 0 {
				analytics.RecordMiss("test_cache", fmt.Sprintf("miss_key_%d", i), time.Duration(i)*time.Microsecond)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			analytics.GenerateReport()
		}
	})
}

func BenchmarkHotKeyTracker(b *testing.B) {
	tracker := NewHotKeyTracker(1*time.Hour, 100)

	b.Run("recordAccess", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tracker.recordAccess(fmt.Sprintf("key_%d", i%1000), i%2 == 0, time.Duration(i%100)*time.Microsecond)
		}
	})

	b.Run("GetHotKeys", func(b *testing.B) {
		// Pre-populate
		for i := 0; i < 1000; i++ {
			tracker.recordAccess(fmt.Sprintf("key_%d", i), true, 0)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tracker.GetHotKeys()
		}
	})
}