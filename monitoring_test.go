package spruce

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestParallelMetricsCollector(t *testing.T) {
	Convey("ParallelMetricsCollector", t, func() {
		Convey("should track operation metrics", func() {
			mc := NewParallelMetricsCollector()
			
			// Record some operations
			mc.RecordOperation(true, 10*time.Millisecond, nil)
			mc.RecordOperation(true, 15*time.Millisecond, nil)
			mc.RecordOperation(false, 5*time.Millisecond, nil)
			mc.RecordOperation(false, 8*time.Millisecond, fmt.Errorf("test error"))
			
			snapshot := mc.GetSnapshot()
			
			So(snapshot.OpsTotal, ShouldEqual, 4)
			So(snapshot.OpsParallel, ShouldEqual, 2)
			So(snapshot.OpsSequential, ShouldEqual, 2)
			So(snapshot.OpsFailed, ShouldEqual, 1)
		})

		Convey("should track concurrency metrics", func() {
			mc := NewParallelMetricsCollector()
			
			// Simulate concurrent operations
			mc.RecordConcurrency(5)
			So(mc.GetSnapshot().CurrentConcurrency, ShouldEqual, 5)
			So(mc.GetSnapshot().MaxConcurrency, ShouldEqual, 5)
			
			mc.RecordConcurrency(3)
			So(mc.GetSnapshot().CurrentConcurrency, ShouldEqual, 8)
			So(mc.GetSnapshot().MaxConcurrency, ShouldEqual, 8)
			
			mc.RecordConcurrency(-5)
			So(mc.GetSnapshot().CurrentConcurrency, ShouldEqual, 3)
			So(mc.GetSnapshot().MaxConcurrency, ShouldEqual, 8)
		})

		Convey("should track lock wait times", func() {
			mc := NewParallelMetricsCollector()
			
			mc.RecordLockWait(100 * time.Microsecond)
			mc.RecordLockWait(200 * time.Microsecond)
			mc.RecordLockWait(1 * time.Millisecond)
			
			snapshot := mc.GetSnapshot()
			
			So(snapshot.LockAcquisitions, ShouldEqual, 3)
			So(snapshot.LockContentions, ShouldEqual, 3) // All waits > 1μs count as contention
			So(snapshot.LockWaitTime, ShouldBeGreaterThan, 1*time.Millisecond)
		})

		Convey("should calculate percentiles", func() {
			mc := NewParallelMetricsCollector()
			
			// Record various operation durations
			durations := []time.Duration{
				1 * time.Millisecond,
				2 * time.Millisecond,
				3 * time.Millisecond,
				4 * time.Millisecond,
				5 * time.Millisecond,
				10 * time.Millisecond,
				20 * time.Millisecond,
				50 * time.Millisecond,
				100 * time.Millisecond,
				200 * time.Millisecond,
			}
			
			for _, d := range durations {
				mc.RecordOperation(true, d, nil)
			}
			
			snapshot := mc.GetSnapshot()
			
			// P50 should be around 5-20ms (with 10 values, P50 is the 5th)
			So(snapshot.OpDurationP50, ShouldBeBetween, 5*time.Millisecond, 20*time.Millisecond)
			
			// P95 should be around 100-200ms  
			So(snapshot.OpDurationP95, ShouldBeBetweenOrEqual, 100*time.Millisecond, 200*time.Millisecond)
			
			// P99 should be close to 200ms
			So(snapshot.OpDurationP99, ShouldBeGreaterThanOrEqualTo, 100*time.Millisecond)
		})

		Convey("should calculate speedup correctly", func() {
			mc := NewParallelMetricsCollector()
			
			// Record sequential operations: 10 ops @ 10ms each = 100ms total
			for i := 0; i < 10; i++ {
				mc.RecordOperation(false, 10*time.Millisecond, nil)
			}
			
			// Record parallel operations: 10 ops @ 2ms each = 20ms total
			for i := 0; i < 10; i++ {
				mc.RecordOperation(true, 2*time.Millisecond, nil)
			}
			
			snapshot := mc.GetSnapshot()
			
			// Speedup should be approximately 5x (10ms/2ms)
			So(snapshot.Speedup, ShouldBeBetween, 4.5, 5.5)
			
			// Parallel ratio should be 50%
			So(snapshot.ParallelRatio, ShouldBeBetween, 49, 51)
		})
	})
}

func TestMetricsServer(t *testing.T) {
	Convey("MetricsServer", t, func() {
		mc := NewParallelMetricsCollector()
		
		// Record some test data
		mc.RecordOperation(true, 10*time.Millisecond, nil)
		mc.RecordOperation(false, 5*time.Millisecond, nil)
		mc.RecordConcurrency(5)
		
		server := NewMetricsServer(mc, 0) // Use port 0 for automatic assignment
		err := server.Start()
		So(err, ShouldBeNil)
		defer server.Stop()
		
		// Give server time to start
		time.Sleep(100 * time.Millisecond)
		
		// Get the actual port from the listener
		_, port, _ := net.SplitHostPort(server.listener.Addr().String())
		baseURL := fmt.Sprintf("http://localhost:%s", port)
		
		Convey("should serve health endpoint", func() {
			resp, err := http.Get(baseURL + "/health")
			So(err, ShouldBeNil)
			defer resp.Body.Close()
			
			So(resp.StatusCode, ShouldEqual, http.StatusOK)
		})
		
		Convey("should serve Prometheus metrics", func() {
			resp, err := http.Get(baseURL + "/metrics")
			So(err, ShouldBeNil)
			defer resp.Body.Close()
			
			So(resp.StatusCode, ShouldEqual, http.StatusOK)
			
			// Read response body
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			content := string(body[:n])
			
			// Check for expected metrics
			So(content, ShouldContainSubstring, "spruce_operations_total")
			So(content, ShouldContainSubstring, "spruce_operation_duration_seconds")
			So(content, ShouldContainSubstring, "spruce_concurrency")
			So(content, ShouldContainSubstring, "spruce_speedup")
		})
		
		Convey("should serve JSON metrics", func() {
			resp, err := http.Get(baseURL + "/metrics/json")
			So(err, ShouldBeNil)
			defer resp.Body.Close()
			
			So(resp.StatusCode, ShouldEqual, http.StatusOK)
			So(resp.Header.Get("Content-Type"), ShouldEqual, "application/json")
			
			// Read response body
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			content := string(body[:n])
			
			// Check for JSON structure
			So(content, ShouldContainSubstring, `"operations"`)
			So(content, ShouldContainSubstring, `"duration"`)
			So(content, ShouldContainSubstring, `"concurrency"`)
			So(content, ShouldContainSubstring, `"performance"`)
		})
	})
}

func TestPercentileCalculation(t *testing.T) {
	Convey("Percentile calculation", t, func() {
		Convey("should handle empty slice", func() {
			p := calculatePercentiles([]time.Duration{})
			So(p.P50, ShouldEqual, 0)
			So(p.P95, ShouldEqual, 0)
			So(p.P99, ShouldEqual, 0)
		})
		
		Convey("should calculate correct percentiles", func() {
			// Create 100 durations from 1ms to 100ms
			durations := make([]time.Duration, 100)
			for i := 0; i < 100; i++ {
				durations[i] = time.Duration(i+1) * time.Millisecond
			}
			
			p := calculatePercentiles(durations)
			
			// P50 should be around 50ms (index 50 = 51ms)
			So(p.P50, ShouldBeBetween, 50*time.Millisecond, 52*time.Millisecond)
			
			// P95 should be around 95ms (index 95 = 96ms)
			So(p.P95, ShouldBeBetween, 95*time.Millisecond, 97*time.Millisecond)
			
			// P99 should be around 99ms (index 99 = 100ms)
			So(p.P99, ShouldBeBetween, 99*time.Millisecond, 101*time.Millisecond)
		})
	})
}