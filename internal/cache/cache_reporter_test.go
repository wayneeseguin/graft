package cache

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCacheReporter(t *testing.T) {
	Convey("CacheReporter", t, func() {
		analytics := NewCacheAnalytics(1 * time.Hour)
		reporter := NewCacheReporter(analytics)

		Convey("should create reporter with analytics", func() {
			So(reporter, ShouldNotBeNil)
			So(reporter.analytics, ShouldEqual, analytics)
		})

		Convey("should generate empty text report", func() {
			var buf bytes.Buffer
			err := reporter.GenerateTextReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()
			So(output, ShouldContainSubstring, "Cache Performance Report")
			So(output, ShouldContainSubstring, "Overall Hit Rate: 0.0%")
			So(output, ShouldContainSubstring, "Effectiveness Score: 0.0/100")
			// Empty analytics will have low hit rate and effectiveness recommendations
			So(output, ShouldContainSubstring, "Recommendations:")
		})

		Convey("should generate text report with data", func() {
			// Add some test data
			analytics.RecordHit("test_cache", "key1")
			analytics.RecordHit("test_cache", "key1")
			analytics.RecordMiss("test_cache", "key2", 50*time.Millisecond)
			analytics.RecordEviction("test_cache", 2, "capacity")
			analytics.UpdateSize("test_cache", 75, 100)

			analytics.RecordHit("another_cache", "key3")
			analytics.RecordMiss("another_cache", "key4", 100*time.Millisecond)
			analytics.UpdateSize("another_cache", 50, 100)

			var buf bytes.Buffer
			err := reporter.GenerateTextReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()

			// Check header
			So(output, ShouldContainSubstring, "Cache Performance Report")
			So(output, ShouldContainSubstring, "Overall Hit Rate: 60.0%")

			// Check cache statistics table
			So(output, ShouldContainSubstring, "Cache Statistics:")
			So(output, ShouldContainSubstring, "test_cache")
			So(output, ShouldContainSubstring, "another_cache")
			So(output, ShouldContainSubstring, "Hit Rate")
			So(output, ShouldContainSubstring, "66.7%") // test_cache hit rate
			So(output, ShouldContainSubstring, "50.0%") // another_cache hit rate

			// Check hot keys section
			So(output, ShouldContainSubstring, "Hot Keys")
			So(output, ShouldContainSubstring, "key1")

			// Check recommendations
			So(output, ShouldContainSubstring, "Recommendations:")
		})

		Convey("should handle long key names in text report", func() {
			longKey := strings.Repeat("a", 50)
			analytics.RecordHit("test_cache", longKey)

			var buf bytes.Buffer
			err := reporter.GenerateTextReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()

			// Should truncate long keys with ellipsis
			So(output, ShouldContainSubstring, "...")
		})
	})
}

func TestCacheReporterCompactReport(t *testing.T) {
	Convey("Compact report", t, func() {
		analytics := NewCacheAnalytics(1 * time.Hour)
		reporter := NewCacheReporter(analytics)

		Convey("should generate empty compact report", func() {
			report := reporter.GenerateCompactReport()
			So(report, ShouldEqual, "")
		})

		Convey("should generate compact report with data", func() {
			analytics.RecordHit("cache1", "key1")
			analytics.RecordHit("cache1", "key1")
			analytics.RecordMiss("cache1", "key2", 10*time.Millisecond)

			analytics.RecordHit("cache2", "key3")

			report := reporter.GenerateCompactReport()

			So(report, ShouldContainSubstring, "cache1: 66.7% hit rate (2/3)")
			So(report, ShouldContainSubstring, "cache2: 100.0% hit rate (1/1)")
			// The compact report will have a comma between caches
			So(strings.HasSuffix(report, ", "), ShouldBeFalse)
		})

		Convey("should handle single cache correctly", func() {
			analytics.RecordHit("single_cache", "key1")
			analytics.RecordMiss("single_cache", "key2", 10*time.Millisecond)

			report := reporter.GenerateCompactReport()

			So(report, ShouldEqual, "single_cache: 50.0% hit rate (1/2)")
		})
	})
}

func TestCacheReporterMetricsReport(t *testing.T) {
	Convey("Metrics report", t, func() {
		analytics := NewCacheAnalytics(1 * time.Hour)
		reporter := NewCacheReporter(analytics)

		Convey("should generate empty metrics report", func() {
			var buf bytes.Buffer
			err := reporter.GenerateMetricsReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()

			So(output, ShouldContainSubstring, "# Cache Analytics Report")
			So(output, ShouldContainSubstring, "cache_total_hits 0")
			So(output, ShouldContainSubstring, "cache_total_misses 0")
			So(output, ShouldContainSubstring, "cache_overall_hit_rate 0.0000")
			So(output, ShouldContainSubstring, "cache_effectiveness_score 0.0000")
		})

		Convey("should generate metrics report with data", func() {
			analytics.RecordHit("test_cache", "key1")
			analytics.RecordHit("test_cache", "key1")
			analytics.RecordMiss("test_cache", "key2", 75*time.Millisecond)
			analytics.RecordEviction("test_cache", 3, "capacity")
			analytics.UpdateSize("test_cache", 80, 100)

			var buf bytes.Buffer
			err := reporter.GenerateMetricsReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()

			// Check overall metrics
			So(output, ShouldContainSubstring, "cache_total_hits 2")
			So(output, ShouldContainSubstring, "cache_total_misses 1")
			So(output, ShouldContainSubstring, "cache_total_evictions 3")
			So(output, ShouldContainSubstring, "cache_overall_hit_rate 0.6667")
			So(output, ShouldContainSubstring, "cache_total_size 80")
			So(output, ShouldContainSubstring, "cache_total_capacity 100")

			// Check per-cache metrics with labels
			So(output, ShouldContainSubstring, `cache_hits{cache="test_cache"} 2`)
			So(output, ShouldContainSubstring, `cache_misses{cache="test_cache"} 1`)
			So(output, ShouldContainSubstring, `cache_hit_rate{cache="test_cache"} 0.6667`)
			So(output, ShouldContainSubstring, `cache_fill_rate{cache="test_cache"} 0.8000`)
			So(output, ShouldContainSubstring, `cache_avg_load_time_seconds{cache="test_cache"} 0.075000`)

			// Check hot key metrics
			So(output, ShouldContainSubstring, `cache_hot_key_accesses{rank="1"}`)
			So(output, ShouldContainSubstring, `cache_hot_key_hit_rate{rank="1"}`)
		})

		Convey("should limit hot keys to top 10", func() {
			// Add more than 10 hot keys
			for i := 0; i < 15; i++ {
				key := fmt.Sprintf("key_%d", i)
				for j := 0; j <= i; j++ {
					analytics.RecordHit("test_cache", key)
				}
			}

			var buf bytes.Buffer
			err := reporter.GenerateMetricsReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()

			// Should only have ranks 1-10
			So(output, ShouldContainSubstring, `{rank="1"}`)
			So(output, ShouldContainSubstring, `{rank="10"}`)
			So(output, ShouldNotContainSubstring, `{rank="11"}`)
		})
	})
}

func TestCacheReporterRecommendations(t *testing.T) {
	Convey("Recommendations", t, func() {
		analytics := NewCacheAnalytics(1 * time.Hour)
		reporter := NewCacheReporter(analytics)

		Convey("should recommend for low overall hit rate", func() {
			// Create low hit rate scenario
			for i := 0; i < 100; i++ {
				analytics.RecordMiss("low_cache", fmt.Sprintf("key_%d", i), 10*time.Millisecond)
			}
			for i := 0; i < 50; i++ {
				analytics.RecordHit("low_cache", fmt.Sprintf("hit_key_%d", i))
			}

			var buf bytes.Buffer
			err := reporter.GenerateTextReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()
			So(output, ShouldContainSubstring, "Overall hit rate is low")
		})

		Convey("should recommend for individual cache issues", func() {
			// Low hit rate cache
			for i := 0; i < 200; i++ {
				analytics.RecordMiss("bad_cache", fmt.Sprintf("key_%d", i), 10*time.Millisecond)
			}
			for i := 0; i < 50; i++ {
				analytics.RecordHit("bad_cache", fmt.Sprintf("hit_key_%d", i))
			}

			// High eviction rate cache
			analytics.RecordHit("evicting_cache", "key1")
			analytics.RecordMiss("evicting_cache", "key2", 10*time.Millisecond)
			for i := 0; i < 150; i++ {
				analytics.RecordEviction("evicting_cache", 1, "capacity")
			}

			// Underutilized cache
			analytics.UpdateSize("underused_cache", 200, 2000)
			analytics.RecordHit("underused_cache", "key1")

			// Slow cache
			analytics.RecordMiss("slow_cache", "key1", 200*time.Millisecond)
			analytics.RecordMiss("slow_cache", "key2", 300*time.Millisecond)

			var buf bytes.Buffer
			err := reporter.GenerateTextReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()

			So(output, ShouldContainSubstring, "bad_cache cache has low hit rate")
			So(output, ShouldContainSubstring, "evicting_cache cache has high eviction rate")
			So(output, ShouldContainSubstring, "underused_cache cache is underutilized")
			So(output, ShouldContainSubstring, "slow_cache cache has slow average load time")
		})

		Convey("should recommend for hot key dominance", func() {
			// Create scenario where one key dominates
			for i := 0; i < 1000; i++ {
				analytics.RecordHit("hot_cache", "dominant_key")
			}
			for i := 0; i < 100; i++ {
				analytics.RecordHit("hot_cache", fmt.Sprintf("other_key_%d", i))
			}

			var buf bytes.Buffer
			err := reporter.GenerateTextReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()
			So(output, ShouldContainSubstring, "accounts for")
			So(output, ShouldContainSubstring, "of cache accesses")
		})

		Convey("should recommend for low effectiveness", func() {
			// Create poor overall effectiveness
			for i := 0; i < 100; i++ {
				analytics.RecordMiss("poor_cache", fmt.Sprintf("key_%d", i), 500*time.Millisecond)
			}
			analytics.RecordEviction("poor_cache", 200, "capacity")

			var buf bytes.Buffer
			err := reporter.GenerateTextReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()
			So(output, ShouldContainSubstring, "Overall cache effectiveness is low")
		})

		Convey("should give positive recommendation for good performance", func() {
			// Create good performance scenario
			for i := 0; i < 100; i++ {
				analytics.RecordHit("good_cache", fmt.Sprintf("key_%d", i))
			}
			analytics.UpdateSize("good_cache", 80, 100)

			var buf bytes.Buffer
			err := reporter.GenerateTextReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()
			So(output, ShouldContainSubstring, "No immediate optimizations needed")
		})
	})
}

func TestCacheReporterDiffReport(t *testing.T) {
	Convey("Diff report", t, func() {
		analytics1 := NewCacheAnalytics(1 * time.Hour)
		analytics2 := NewCacheAnalytics(1 * time.Hour)
		reporter := NewCacheReporter(analytics2)

		Convey("should generate diff report", func() {
			// Setup previous state
			analytics1.RecordHit("cache1", "key1")
			analytics1.RecordMiss("cache1", "key2", 50*time.Millisecond)
			time.Sleep(1 * time.Millisecond) // Ensure different timestamps

			// Setup current state (improved)
			analytics2.RecordHit("cache1", "key1")
			analytics2.RecordHit("cache1", "key1")
			analytics2.RecordHit("cache1", "key2")
			analytics2.RecordMiss("cache1", "key3", 30*time.Millisecond)

			// Add new cache in current
			analytics2.RecordHit("cache2", "new_key")

			previous := analytics1.GenerateReport()
			current := analytics2.GenerateReport()

			var buf bytes.Buffer
			err := reporter.GenerateDiffReport(previous, current, &buf)

			So(err, ShouldBeNil)
			output := buf.String()

			So(output, ShouldContainSubstring, "Cache Performance Comparison")
			So(output, ShouldContainSubstring, "Overall Metrics:")
			So(output, ShouldContainSubstring, "Hit Rate:")
			So(output, ShouldContainSubstring, "cache1:")
			So(output, ShouldContainSubstring, "cache2: NEW CACHE")
		})

		Convey("should show significant changes only", func() {
			// Setup scenarios with minor and major changes
			analytics1.RecordHit("minor_change", "key1")                       // 1 hit
			analytics1.RecordMiss("minor_change", "key2", 50*time.Millisecond) // 1 miss -> 50% hit rate

			analytics1.RecordHit("major_change", "key1")                       // 1 hit
			analytics1.RecordMiss("major_change", "key2", 50*time.Millisecond) // 1 miss
			analytics1.RecordMiss("major_change", "key3", 50*time.Millisecond) // 1 miss -> 33.3% hit rate

			// Minor change: Create scenario with <1% difference
			// Previous: 50% (1 hit, 1 miss), Current: ~50.5% (100 hits, 99 misses)
			for i := 0; i < 100; i++ {
				analytics2.RecordHit("minor_change", fmt.Sprintf("hit_%d", i))
			}
			for i := 0; i < 99; i++ {
				analytics2.RecordMiss("minor_change", fmt.Sprintf("miss_%d", i), 40*time.Millisecond)
			}

			// Major change in current (33.3% -> 80%)
			analytics2.RecordHit("major_change", "key1")
			analytics2.RecordHit("major_change", "key1")
			analytics2.RecordHit("major_change", "key1")
			analytics2.RecordHit("major_change", "key1")
			analytics2.RecordMiss("major_change", "key2", 30*time.Millisecond)

			previous := analytics1.GenerateReport()
			current := analytics2.GenerateReport()

			var buf bytes.Buffer
			err := reporter.GenerateDiffReport(previous, current, &buf)

			So(err, ShouldBeNil)
			output := buf.String()

			// Should show major change but not minor change (less than 1% difference threshold)
			So(output, ShouldContainSubstring, "major_change:")
			So(output, ShouldNotContainSubstring, "minor_change:")
		})
	})
}

func TestHelperFunctions(t *testing.T) {
	Convey("Helper functions", t, func() {
		Convey("truncateKey", func() {
			So(truncateKey("short", 10), ShouldEqual, "short")
			So(truncateKey("this_is_a_long_key", 10), ShouldEqual, "this_is...")
			So(truncateKey("exact_length", 12), ShouldEqual, "exact_length")
		})

		Convey("formatDuration", func() {
			So(formatDuration(30*time.Second), ShouldEqual, "30s")
			So(formatDuration(90*time.Second), ShouldEqual, "2m")
			So(formatDuration(2*time.Hour), ShouldEqual, "2.0h")
			So(formatDuration(25*time.Hour), ShouldEqual, "1.0d")
		})
	})
}

func TestCacheReporterEdgeCases(t *testing.T) {
	Convey("Edge cases", t, func() {
		analytics := NewCacheAnalytics(1 * time.Hour)
		reporter := NewCacheReporter(analytics)

		Convey("should handle zero load times", func() {
			analytics.RecordMiss("test_cache", "key1", 0)
			analytics.RecordMiss("test_cache", "key2", 0)

			var buf bytes.Buffer
			err := reporter.GenerateMetricsReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()
			So(output, ShouldContainSubstring, "cache_avg_load_time_seconds")
			So(output, ShouldContainSubstring, "0.000000")
		})

		Convey("should handle caches with only hits", func() {
			analytics.RecordHit("hit_only_cache", "key1")
			analytics.RecordHit("hit_only_cache", "key2")

			var buf bytes.Buffer
			err := reporter.GenerateTextReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()
			So(output, ShouldContainSubstring, "100.0%")
		})

		Convey("should handle caches with only misses", func() {
			analytics.RecordMiss("miss_only_cache", "key1", 10*time.Millisecond)
			analytics.RecordMiss("miss_only_cache", "key2", 20*time.Millisecond)

			var buf bytes.Buffer
			err := reporter.GenerateTextReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()
			So(output, ShouldContainSubstring, "0.0%")
		})

		Convey("should handle very large numbers", func() {
			// Simulate very large cache
			for i := 0; i < 1000; i++ {
				analytics.RecordHit("huge_cache", fmt.Sprintf("key_%d", i))
			}
			analytics.UpdateSize("huge_cache", 999999, 1000000)

			var buf bytes.Buffer
			err := reporter.GenerateTextReport(&buf)

			So(err, ShouldBeNil)
			output := buf.String()
			So(output, ShouldContainSubstring, "1000")
			So(output, ShouldContainSubstring, "999999/1000000")
		})
	})
}

func BenchmarkCacheReporter(b *testing.B) {
	analytics := NewCacheAnalytics(1 * time.Hour)
	reporter := NewCacheReporter(analytics)

	// Pre-populate with test data
	for i := 0; i < 1000; i++ {
		analytics.RecordHit("test_cache", fmt.Sprintf("key_%d", i%100))
		if i%10 == 0 {
			analytics.RecordMiss("test_cache", fmt.Sprintf("miss_key_%d", i), time.Duration(i)*time.Microsecond)
		}
	}

	b.Run("GenerateTextReport", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			reporter.GenerateTextReport(&buf)
		}
	})

	b.Run("GenerateCompactReport", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			reporter.GenerateCompactReport()
		}
	})

	b.Run("GenerateMetricsReport", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			reporter.GenerateMetricsReport(&buf)
		}
	})

	b.Run("GenerateDiffReport", func(b *testing.B) {
		// Create two reports for comparison
		previous := analytics.GenerateReport()
		current := analytics.GenerateReport()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			reporter.GenerateDiffReport(previous, current, &buf)
		}
	})
}
