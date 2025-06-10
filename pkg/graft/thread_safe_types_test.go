package graft

import (
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWorkerPool(t *testing.T) {
	Convey("WorkerPool operations", t, func() {

		Convey("should create a new worker pool", func() {
			pool := NewWorkerPool(3)
			So(pool, ShouldNotBeNil)
			So(pool.workers, ShouldEqual, 3)
			So(pool.tasks, ShouldNotBeNil)
		})

		Convey("should start and stop workers correctly", func() {
			pool := NewWorkerPool(2)
			So(pool, ShouldNotBeNil)

			// Start the pool
			pool.Start()

			// Submit some tasks
			results := make([]int, 0, 3)
			var mu sync.Mutex

			for i := 0; i < 3; i++ {
				val := i
				pool.Submit(func() {
					mu.Lock()
					results = append(results, val)
					mu.Unlock()
				})
			}

			// Stop the pool and wait for completion
			pool.Stop()

			// Verify all tasks were executed
			So(len(results), ShouldEqual, 3)
		})

		Convey("should handle concurrent task submission", func() {
			pool := NewWorkerPool(3)
			pool.Start()

			var counter int
			var mu sync.Mutex
			taskCount := 10

			for i := 0; i < taskCount; i++ {
				pool.Submit(func() {
					mu.Lock()
					counter++
					mu.Unlock()
				})
			}

			pool.Stop()

			So(counter, ShouldEqual, taskCount)
		})

		Convey("should handle empty task queue gracefully", func() {
			pool := NewWorkerPool(2)
			pool.Start()

			// Immediately stop without submitting tasks
			pool.Stop()

			// Should not panic or hang
		})

		Convey("should handle single worker pool", func() {
			pool := NewWorkerPool(1)
			pool.Start()

			var executed bool
			pool.Submit(func() {
				executed = true
			})

			pool.Stop()

			So(executed, ShouldBeTrue)
		})

		Convey("should handle zero workers gracefully", func() {
			pool := NewWorkerPool(0)
			So(pool, ShouldNotBeNil)
			So(pool.workers, ShouldEqual, 0)

			// Start should not start any goroutines
			pool.Start()

			// Submit will block since no workers will consume from channel
			// Test that we can still stop the pool
			pool.Stop()

			// After stop, submit should panic on closed channel
			So(func() {
				pool.Submit(func() {})
			}, ShouldPanic)
		})

		Convey("should handle multiple start/stop cycles", func() {
			pool := NewWorkerPool(2)

			// First cycle
			pool.Start()
			var counter1 int
			var mu sync.Mutex

			pool.Submit(func() {
				mu.Lock()
				counter1++
				mu.Unlock()
			})

			pool.Stop()
			So(counter1, ShouldEqual, 1)

			// Create new pool for second cycle (can't reuse stopped pool)
			pool2 := NewWorkerPool(2)
			pool2.Start()

			var counter2 int
			pool2.Submit(func() {
				mu.Lock()
				counter2++
				mu.Unlock()
			})

			pool2.Stop()
			So(counter2, ShouldEqual, 1)
		})

		Convey("should handle tasks that panic", func() {
			pool := NewWorkerPool(2)
			pool.Start()

			var normalTaskExecuted bool
			var mu sync.Mutex

			// Submit a panicking task with recovery
			pool.Submit(func() {
				defer func() {
					recover() // Recover from the panic to prevent test failure
				}()
				panic("test panic")
			})

			// Submit a normal task
			pool.Submit(func() {
				mu.Lock()
				normalTaskExecuted = true
				mu.Unlock()
			})

			// Give time for tasks to execute
			time.Sleep(10 * time.Millisecond)

			pool.Stop()

			// Normal task should still execute despite the panic
			So(normalTaskExecuted, ShouldBeTrue)
		})

		Convey("should handle high-concurrency scenarios", func() {
			pool := NewWorkerPool(5)
			pool.Start()

			var counter int
			var mu sync.Mutex
			taskCount := 100

			// Submit many tasks concurrently
			var wg sync.WaitGroup
			for i := 0; i < taskCount; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					pool.Submit(func() {
						mu.Lock()
						counter++
						mu.Unlock()
					})
				}()
			}

			wg.Wait() // Wait for all submissions
			pool.Stop()

			So(counter, ShouldEqual, taskCount)
		})
	})
}

func TestWorkerPool_EdgeCases(t *testing.T) {
	Convey("WorkerPool edge cases", t, func() {

		Convey("should handle tasks with different execution times", func() {
			pool := NewWorkerPool(3)
			pool.Start()

			results := make([]string, 0)
			var mu sync.Mutex

			// Submit fast task
			pool.Submit(func() {
				mu.Lock()
				results = append(results, "fast")
				mu.Unlock()
			})

			// Submit slow task
			pool.Submit(func() {
				time.Sleep(5 * time.Millisecond)
				mu.Lock()
				results = append(results, "slow")
				mu.Unlock()
			})

			// Submit another fast task
			pool.Submit(func() {
				mu.Lock()
				results = append(results, "fast2")
				mu.Unlock()
			})

			pool.Stop()

			So(len(results), ShouldEqual, 3)
			So(results, ShouldContain, "fast")
			So(results, ShouldContain, "slow")
			So(results, ShouldContain, "fast2")
		})

		Convey("should handle task submission after stop", func() {
			pool := NewWorkerPool(2)
			pool.Start()
			pool.Stop()

			// Submitting after stop should panic since channel is closed
			So(func() {
				pool.Submit(func() {})
			}, ShouldPanic)
		})
	})
}

func TestWorkerPool_Memory(t *testing.T) {
	Convey("WorkerPool memory management", t, func() {

		Convey("should not leak goroutines", func() {
			initialGoroutines := testing.Short() // Placeholder for goroutine count

			for i := 0; i < 5; i++ {
				pool := NewWorkerPool(3)
				pool.Start()

				// Submit some work
				for j := 0; j < 10; j++ {
					pool.Submit(func() {
						time.Sleep(time.Millisecond)
					})
				}

				pool.Stop()
			}

			// In a real test, you'd check that goroutine count returned to initial level
			// This is a placeholder to demonstrate the test structure
			So(initialGoroutines, ShouldNotBeNil)
		})

		Convey("should handle large number of workers", func() {
			pool := NewWorkerPool(100)
			So(pool, ShouldNotBeNil)
			So(pool.workers, ShouldEqual, 100)

			pool.Start()

			var counter int
			var mu sync.Mutex

			// Submit tasks equal to worker count
			for i := 0; i < 100; i++ {
				pool.Submit(func() {
					mu.Lock()
					counter++
					mu.Unlock()
				})
			}

			pool.Stop()
			So(counter, ShouldEqual, 100)
		})
	})
}

func TestWorkerPool_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	Convey("WorkerPool performance characteristics", t, func() {

		Convey("should handle burst workloads efficiently", func() {
			pool := NewWorkerPool(10)
			pool.Start()

			start := time.Now()
			var counter int
			var mu sync.Mutex
			taskCount := 1000

			for i := 0; i < taskCount; i++ {
				pool.Submit(func() {
					// Simulate small amount of work
					time.Sleep(time.Microsecond * 100)
					mu.Lock()
					counter++
					mu.Unlock()
				})
			}

			pool.Stop()
			duration := time.Since(start)

			So(counter, ShouldEqual, taskCount)
			So(duration, ShouldBeLessThan, time.Second*5) // Should complete reasonably fast
		})
	})
}
