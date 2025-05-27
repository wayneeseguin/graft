package internal

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMetricsSystem(t *testing.T) {
	Convey("Metrics System", t, func() {
		// Initialize components
		InitializeMetrics()
		InitializeMetricsRegistry(nil)
		InitializeCacheAnalytics()
		InitializeTimingAggregator()
		InitializeSlowOpDetector(nil)

		mc := GetMetricsCollector()
		registry := GetMetricsRegistry()

		Convey("Core Metrics", func() {
			Convey("Should record counter metrics", func() {
				counter := NewCounter("test_counter", nil)
				counter.Inc()
				counter.Add(5)
				So(counter.Get(), ShouldEqual, 6)
			})

			Convey("Should record gauge metrics", func() {
				gauge := NewGauge("test_gauge", nil)
				gauge.Set(10)
				gauge.Inc()
				gauge.Dec()
				So(gauge.Get(), ShouldEqual, 10)
			})

			Convey("Should record histogram metrics", func() {
				histogram := NewHistogram("test_histogram", nil)
				histogram.Observe(0.1)
				histogram.Observe(0.2)
				histogram.Observe(0.3)

				stats := histogram.GetStats()
				So(stats.Count, ShouldEqual, 3)
				So(stats.Mean, ShouldAlmostEqual, 0.2, 0.01)
			})
		})

		Convey("Metrics Collector", func() {
			Convey("Should record parse operations", func() {
				start := time.Now()
				time.Sleep(10 * time.Millisecond)
				mc.RecordParseOperation(start, nil)

				parseOps := mc.ParseOperations.GetAll()
				So(len(parseOps), ShouldBeGreaterThan, 0)
			})

			Convey("Should record cache metrics", func() {
				mc.RecordCacheHit("expression")
				mc.RecordCacheMiss("expression")
				mc.UpdateCacheSize("expression", 100)

				// Verify metrics were recorded
				snapshot := registry.GetSnapshot()
				So(snapshot, ShouldNotBeNil)
			})
		})

		Convey("Timing System", func() {
			Convey("Should track operation timing", func() {
				timer := NewTimer("test_operation")
				time.Sleep(10 * time.Millisecond)
				duration := timer.Stop()

				So(duration, ShouldBeGreaterThan, 10*time.Millisecond)
				So(timer.Duration(), ShouldEqual, duration)
			})

			Convey("Should support hierarchical timing", func() {
				parent := NewTimer("parent")
				child := parent.Child("child")
				time.Sleep(5 * time.Millisecond)
				child.Stop()
				parent.Stop()

				tree := parent.GetTree()
				So(tree.Name, ShouldEqual, "parent")
				So(len(tree.Children), ShouldEqual, 1)
				So(tree.Children[0].Name, ShouldEqual, "child")
			})
		})

		Convey("Cache Analytics", func() {
			analytics := GetCacheAnalytics()

			Convey("Should track cache performance", func() {
				analytics.RecordHit("test_cache", "key1")
				analytics.RecordHit("test_cache", "key1")
				analytics.RecordMiss("test_cache", "key2", 10*time.Millisecond)
				analytics.UpdateSize("test_cache", 2, 100)

				stats, exists := analytics.GetCacheStats("test_cache")
				So(exists, ShouldBeTrue)
				So(stats.Hits, ShouldEqual, 2)
				So(stats.Misses, ShouldEqual, 1)
				So(stats.HitRate, ShouldAlmostEqual, 0.667, 0.01)
			})

			Convey("Should track hot keys", func() {
				for i := 0; i < 10; i++ {
					analytics.RecordHit("test_cache", "hot_key")
				}

				hotKeys := analytics.GetHotKeys()
				So(len(hotKeys), ShouldBeGreaterThan, 0)
				So(hotKeys[0].Key, ShouldEqual, "hot_key")
				So(hotKeys[0].AccessCount, ShouldBeGreaterThanOrEqualTo, 10)
			})
		})

		Convey("Slow Operation Detection", func() {
			detector := GetSlowOpDetector()
			detector.SetThreshold("test_op", 5*time.Millisecond)

			Convey("Should detect slow operations", func() {
				timer := NewTimer("test_op")
				time.Sleep(10 * time.Millisecond)
				timer.Stop()

				// Check should have been called automatically
				history := detector.GetHistory(10)
				found := false
				for _, op := range history {
					if op.Name == "test_op" {
						found = true
						So(op.Duration, ShouldBeGreaterThan, 5*time.Millisecond)
						break
					}
				}
				So(found, ShouldBeTrue)
			})
		})

		Convey("Metrics Export", func() {
			exporter := NewMetricsExporter(registry)

			Convey("Should export Prometheus format", func() {
				data, err := exporter.Export(ExportFormatPrometheus)
				So(err, ShouldBeNil)
				So(string(data), ShouldContainSubstring, "# HELP")
				So(string(data), ShouldContainSubstring, "# TYPE")
			})

			Convey("Should export JSON format", func() {
				data, err := exporter.Export(ExportFormatJSON)
				So(err, ShouldBeNil)
				So(string(data), ShouldContainSubstring, "timestamp")
				So(string(data), ShouldContainSubstring, "metrics")
			})
		})
	})
}
