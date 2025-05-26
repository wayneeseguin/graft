package spruce

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
	
	. "github.com/smartystreets/goconvey/convey"
)

func TestCacheWarming(t *testing.T) {
	Convey("Cache Warming System", t, func() {
		
		tempDir, err := ioutil.TempDir("", "spruce_warming_test")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tempDir)
		
		Convey("Basic cache warming", func() {
			// Create hierarchical cache
			cacheConfig := HierarchicalCacheConfig{
				L1Size:      100,
				L2Size:      1000,
				L2Enabled:   false, // Disable L2 for simpler testing
				TTL:         time.Minute,
			}
			
			cache, err := NewHierarchicalCache(cacheConfig)
			So(err, ShouldBeNil)
			defer cache.Close()
			
			// Create cache warming system
			warmingConfig := CacheWarmingConfig{
				Enabled:         true,
				Strategy:        "frequency",
				Background:      false,
				StartupTimeout:  5 * time.Second,
				WarmingInterval: 0, // Disable periodic warming for test
				TopN:            10,
				MinFrequency:    1,
			}
			
			warming := NewCacheWarming(cache, warmingConfig)
			
			// Record some usage patterns
			warming.RecordUsage("grab meta.key1", 100*time.Millisecond, "grab")
			warming.RecordUsage("grab meta.key1", 120*time.Millisecond, "grab") // Duplicate to increase frequency
			warming.RecordUsage("concat a b", 50*time.Millisecond, "concat")
			
			// Perform cache warming
			err = warming.WarmCache()
			So(err, ShouldBeNil)
			
			// Check warming stats
			stats := warming.GetWarmingStats()
			So(stats.ExpressionsWarmed, ShouldBeGreaterThan, 0)
		})
		
		Convey("Usage analytics", func() {
			analytics := NewUsageAnalytics(100, "")
			
			// Record usage patterns
			analytics.RecordUsage("grab meta.key1", 100*time.Millisecond, "grab")
			analytics.RecordUsage("grab meta.key1", 120*time.Millisecond, "grab")
			analytics.RecordUsage("concat a b", 50*time.Millisecond, "concat")
			analytics.RecordUsage("grab meta.key2", 80*time.Millisecond, "grab")
			
			// Get top patterns
			patterns := analytics.GetTopPatterns(10)
			So(len(patterns), ShouldBeGreaterThan, 0)
			
			// Most frequent should be grab meta.key1
			So(patterns[0].Expression, ShouldEqual, "grab meta.key1")
			So(patterns[0].Frequency, ShouldEqual, 2)
			So(patterns[0].AverageTime, ShouldBeGreaterThan, 0)
		})
		
		Convey("Pattern normalization", func() {
			analytics := NewUsageAnalytics(100, "")
			
			// Record similar patterns
			analytics.RecordUsage(`grab "value1"`, 100*time.Millisecond, "grab")
			analytics.RecordUsage(`grab "value2"`, 100*time.Millisecond, "grab")
			analytics.RecordUsage(`concat "hello" "world"`, 50*time.Millisecond, "concat")
			
			patterns := analytics.GetTopPatterns(10)
			
			// Verify patterns are normalized
			for _, pattern := range patterns {
				So(pattern.Pattern, ShouldNotBeEmpty)
			}
		})
		
		Convey("Persistence of usage patterns", func() {
			usageFile := tempDir + "/usage_patterns.json"
			
			// Create analytics with persistence
			analytics1 := NewUsageAnalytics(100, usageFile)
			analytics1.RecordUsage("grab meta.persistent", 100*time.Millisecond, "grab")
			analytics1.RecordUsage("grab meta.persistent", 120*time.Millisecond, "grab")
			
			// Save to disk
			err := analytics1.SaveToDisk()
			So(err, ShouldBeNil)
			
			// Create new analytics instance and load
			analytics2 := NewUsageAnalytics(100, usageFile)
			err = analytics2.LoadFromDisk()
			So(err, ShouldBeNil)
			
			// Verify data was loaded
			patterns := analytics2.GetTopPatterns(10)
			So(len(patterns), ShouldBeGreaterThan, 0)
			So(patterns[0].Expression, ShouldEqual, "grab meta.persistent")
			So(patterns[0].Frequency, ShouldEqual, 2)
		})
		
		Convey("Warming strategies", func() {
			cache, err := NewHierarchicalCache(HierarchicalCacheConfig{
				L1Size:    100,
				L2Enabled: false,
				TTL:       time.Minute,
			})
			So(err, ShouldBeNil)
			defer cache.Close()
			
			Convey("Frequency-based strategy", func() {
				warmingConfig := CacheWarmingConfig{
					Enabled:      true,
					Strategy:     "frequency",
					TopN:         5,
					MinFrequency: 2,
				}
				
				warming := NewCacheWarming(cache, warmingConfig)
				
				// Record patterns with different frequencies
				warming.RecordUsage("grab meta.high_freq", 100*time.Millisecond, "grab")
				warming.RecordUsage("grab meta.high_freq", 100*time.Millisecond, "grab")
				warming.RecordUsage("grab meta.high_freq", 100*time.Millisecond, "grab") // Frequency 3
				warming.RecordUsage("grab meta.low_freq", 100*time.Millisecond, "grab")   // Frequency 1
				
				expressions := warming.getExpressionsToWarm()
				
				// Should only include high frequency expressions
				So(len(expressions), ShouldEqual, 1)
				So(expressions[0], ShouldEqual, "grab meta.high_freq")
			})
			
			Convey("Pattern-based strategy", func() {
				warmingConfig := CacheWarmingConfig{
					Enabled:  true,
					Strategy: "pattern",
					TopN:     5,
				}
				
				warming := NewCacheWarming(cache, warmingConfig)
				
				// Record patterns with similar normalized patterns
				warming.RecordUsage(`grab "value1"`, 100*time.Millisecond, "grab")
				warming.RecordUsage(`grab "value2"`, 100*time.Millisecond, "grab")
				warming.RecordUsage(`concat "a" "b"`, 50*time.Millisecond, "concat")
				
				expressions := warming.getExpressionsToWarm()
				So(len(expressions), ShouldBeGreaterThan, 0)
			})
			
			Convey("Hybrid strategy", func() {
				warmingConfig := CacheWarmingConfig{
					Enabled:      true,
					Strategy:     "hybrid",
					TopN:         10,
					MinFrequency: 1,
				}
				
				warming := NewCacheWarming(cache, warmingConfig)
				
				warming.RecordUsage("grab meta.key1", 100*time.Millisecond, "grab")
				warming.RecordUsage("grab meta.key1", 100*time.Millisecond, "grab")
				warming.RecordUsage(`grab "value"`, 100*time.Millisecond, "grab")
				
				expressions := warming.getExpressionsToWarm()
				So(len(expressions), ShouldBeGreaterThan, 0)
			})
		})
		
		Convey("Startup warming with timeout", func() {
			cache, err := NewHierarchicalCache(HierarchicalCacheConfig{
				L1Size:    100,
				L2Enabled: false,
				TTL:       time.Minute,
			})
			So(err, ShouldBeNil)
			defer cache.Close()
			
			warmingConfig := CacheWarmingConfig{
				Enabled:        true,
				Strategy:       "frequency",
				StartupTimeout: 100 * time.Millisecond, // Short timeout
				TopN:           5,
				MinFrequency:   1,
			}
			
			warming := NewCacheWarming(cache, warmingConfig)
			warming.RecordUsage("grab meta.key", 100*time.Millisecond, "grab")
			
			// Should complete within timeout
			start := time.Now()
			err = warming.WarmStartup()
			elapsed := time.Since(start)
			
			So(err, ShouldBeNil)
			So(elapsed, ShouldBeLessThan, 200*time.Millisecond)
		})
		
		Convey("Pattern eviction when max patterns reached", func() {
			analytics := NewUsageAnalytics(2, "") // Small limit
			
			// Add patterns beyond limit
			analytics.RecordUsage("pattern1", 100*time.Millisecond, "grab")
			analytics.RecordUsage("pattern2", 100*time.Millisecond, "grab")
			analytics.RecordUsage("pattern3", 100*time.Millisecond, "grab") // Should evict oldest
			
			patterns := analytics.GetTopPatterns(10)
			So(len(patterns), ShouldBeLessThanOrEqualTo, 2)
		})
		
		Convey("Disabled warming", func() {
			cache, err := NewHierarchicalCache(HierarchicalCacheConfig{
				L1Size:    100,
				L2Enabled: false,
				TTL:       time.Minute,
			})
			So(err, ShouldBeNil)
			defer cache.Close()
			
			warmingConfig := CacheWarmingConfig{
				Enabled: false,
			}
			
			warming := NewCacheWarming(cache, warmingConfig)
			
			// Should do nothing when disabled
			err = warming.WarmCache()
			So(err, ShouldBeNil)
			
			stats := warming.GetWarmingStats()
			So(stats.ExpressionsWarmed, ShouldEqual, 0)
		})
	})
}

func TestUsageAnalytics(t *testing.T) {
	Convey("Usage Analytics", t, func() {
		
		Convey("Basic usage recording", func() {
			analytics := NewUsageAnalytics(100, "")
			
			// Record same expression multiple times
			analytics.RecordUsage("grab meta.key", 100*time.Millisecond, "grab")
			analytics.RecordUsage("grab meta.key", 200*time.Millisecond, "grab")
			analytics.RecordUsage("grab meta.key", 150*time.Millisecond, "grab")
			
			patterns := analytics.GetTopPatterns(1)
			So(len(patterns), ShouldEqual, 1)
			
			pattern := patterns[0]
			So(pattern.Expression, ShouldEqual, "grab meta.key")
			So(pattern.Frequency, ShouldEqual, 3)
			So(pattern.AverageTime, ShouldAlmostEqual, 0.15, 0.01) // (100+200+150)/3 = 150ms
			So(pattern.OperatorType, ShouldEqual, "grab")
		})
		
		Convey("Pattern ordering by frequency", func() {
			analytics := NewUsageAnalytics(100, "")
			
			// Record patterns with different frequencies
			analytics.RecordUsage("low_freq", 100*time.Millisecond, "grab")
			
			analytics.RecordUsage("high_freq", 100*time.Millisecond, "grab")
			analytics.RecordUsage("high_freq", 100*time.Millisecond, "grab")
			analytics.RecordUsage("high_freq", 100*time.Millisecond, "grab")
			
			analytics.RecordUsage("med_freq", 100*time.Millisecond, "grab")
			analytics.RecordUsage("med_freq", 100*time.Millisecond, "grab")
			
			patterns := analytics.GetTopPatterns(10)
			So(len(patterns), ShouldEqual, 3)
			
			// Should be ordered by frequency
			So(patterns[0].Expression, ShouldEqual, "high_freq")
			So(patterns[0].Frequency, ShouldEqual, 3)
			So(patterns[1].Expression, ShouldEqual, "med_freq")
			So(patterns[1].Frequency, ShouldEqual, 2)
			So(patterns[2].Expression, ShouldEqual, "low_freq")
			So(patterns[2].Frequency, ShouldEqual, 1)
		})
		
		Convey("Limited top patterns", func() {
			analytics := NewUsageAnalytics(100, "")
			
			for i := 0; i < 10; i++ {
				analytics.RecordUsage(fmt.Sprintf("pattern_%d", i), 100*time.Millisecond, "grab")
			}
			
			patterns := analytics.GetTopPatterns(5)
			So(len(patterns), ShouldEqual, 5)
		})
	})
}

func BenchmarkCacheWarming(b *testing.B) {
	cache, _ := NewHierarchicalCache(HierarchicalCacheConfig{
		L1Size:    1000,
		L2Enabled: false,
		TTL:       time.Hour,
	})
	defer cache.Close()
	
	warmingConfig := CacheWarmingConfig{
		Enabled:      true,
		Strategy:     "frequency",
		TopN:         100,
		MinFrequency: 1,
	}
	
	warming := NewCacheWarming(cache, warmingConfig)
	
	b.Run("RecordUsage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			expression := fmt.Sprintf("grab meta.key_%d", i%100)
			warming.RecordUsage(expression, 100*time.Millisecond, "grab")
		}
	})
	
	b.Run("GetTopPatterns", func(b *testing.B) {
		// Pre-populate with patterns
		for i := 0; i < 1000; i++ {
			expression := fmt.Sprintf("grab meta.key_%d", i)
			warming.RecordUsage(expression, 100*time.Millisecond, "grab")
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			warming.analytics.GetTopPatterns(50)
		}
	})
	
	b.Run("WarmCache", func(b *testing.B) {
		// Pre-populate with patterns
		for i := 0; i < 100; i++ {
			expression := fmt.Sprintf("grab meta.key_%d", i)
			warming.RecordUsage(expression, 100*time.Millisecond, "grab")
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			warming.WarmCache()
		}
	})
}