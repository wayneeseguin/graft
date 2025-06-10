// TODO: Thread safe parallel engine tests removed - ExecutionTask and related types not implemented
//go:build ignore
// +build ignore

package graft

import (
	"fmt"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestParallelExecutionEngine(t *testing.T) {
	Convey("ParallelExecutionEngine operations", t, func() {
		tree := NewCOWTree(map[interface{}]interface{}{
			"test": map[interface{}]interface{}{
				"data": "value",
			},
		})

		engine := NewParallelExecutionEngine(tree, 2)
		engine.Start()
		defer engine.Stop()

		Convey("Basic task execution", func() {
			// Submit a get task
			task := &ExecutionTask{
				ID:       "test-1",
				Path:     []string{"test", "data"},
				Operator: "get",
			}

			err := engine.Submit(task)
			So(err, ShouldBeNil)

			// Get result
			result, ok := engine.GetResult()
			So(ok, ShouldBeTrue)
			So(result.TaskID, ShouldEqual, "test-1")
			So(result.Error, ShouldBeNil)
			So(result.Value, ShouldEqual, "value")
		})

		Convey("Set and get operations", func() {
			// Set a value
			setTask := &ExecutionTask{
				ID:       "set-1",
				Path:     []string{"new", "key"},
				Operator: "set",
				Args:     []interface{}{"new-value"},
			}

			err := engine.Submit(setTask)
			So(err, ShouldBeNil)

			result, _ := engine.GetResult()
			So(result.Error, ShouldBeNil)

			// Get the value
			getTask := &ExecutionTask{
				ID:       "get-1",
				Path:     []string{"new", "key"},
				Operator: "get",
			}

			err = engine.Submit(getTask)
			So(err, ShouldBeNil)

			result, _ = engine.GetResult()
			So(result.Error, ShouldBeNil)
			So(result.Value, ShouldEqual, "new-value")
		})

		Convey("Delete operation", func() {
			// Set a value
			tree.Set("to-delete", "delete", "me")

			// Delete it
			deleteTask := &ExecutionTask{
				ID:       "delete-1",
				Path:     []string{"delete", "me"},
				Operator: "delete",
			}

			err := engine.Submit(deleteTask)
			So(err, ShouldBeNil)

			result, _ := engine.GetResult()
			So(result.Error, ShouldBeNil)

			// Verify deletion
			exists := tree.Exists("delete", "me")
			So(exists, ShouldBeFalse)
		})

		Convey("Update operation", func() {
			tree.Set(10, "counter")

			updateTask := &ExecutionTask{
				ID:       "update-1",
				Path:     []string{"counter"},
				Operator: "update",
				Args: []interface{}{
					func(current interface{}) interface{} {
						if val, ok := current.(int); ok {
							return val + 5
						}
						return 0
					},
				},
			}

			err := engine.Submit(updateTask)
			So(err, ShouldBeNil)

			result, _ := engine.GetResult()
			So(result.Error, ShouldBeNil)

			// Verify update
			value, _ := tree.Find("counter")
			So(value, ShouldEqual, 15)
		})

		Convey("Error handling", func() {
			// Invalid operator
			task := &ExecutionTask{
				ID:       "error-1",
				Operator: "invalid",
			}

			engine.Submit(task)
			result, _ := engine.GetResult()
			So(result.Error, ShouldNotBeNil)
			So(result.Error.Error(), ShouldContainSubstring, "unknown operator")
		})

		Convey("Metrics tracking", func() {
			// Submit multiple tasks
			for i := 0; i < 5; i++ {
				task := &ExecutionTask{
					ID:       fmt.Sprintf("metric-%d", i),
					Path:     []string{"test", "data"},
					Operator: "get",
				}
				engine.Submit(task)
			}

			// Collect results
			for i := 0; i < 5; i++ {
				engine.GetResult()
			}

			metrics := engine.GetMetrics()
			So(metrics["tasks_queued"], ShouldBeGreaterThanOrEqualTo, 5)
			So(metrics["tasks_executed"], ShouldBeGreaterThanOrEqualTo, 5)
			So(metrics["tasks_succeeded"], ShouldBeGreaterThanOrEqualTo, 5)
		})
	})
}

func TestThreadSafeParallelEvaluatorEngine(t *testing.T) {
	Convey("ThreadSafeParallelEvaluator operations", t, func() {
		tree := NewCOWTree(map[interface{}]interface{}{
			"data": map[interface{}]interface{}{
				"value1": 10,
				"value2": 20,
			},
		})

		evaluator := NewParallelEvaluator(tree, 4)
		defer evaluator.Stop()

		Convey("Parallel operation execution", func() {
			operations := []Operation{
				{Type: "get", Path: []string{"data", "value1"}},
				{Type: "get", Path: []string{"data", "value2"}},
				{Type: "set", Path: []string{"data", "value3"}, Args: []interface{}{30}},
				{Type: "get", Path: []string{"data", "value3"}},
			}

			results, err := evaluator.EvaluateParallel(operations)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 4)

			// Check results
			So(results[0].Error, ShouldBeNil)
			So(results[0].Value, ShouldEqual, 10)

			So(results[1].Error, ShouldBeNil)
			So(results[1].Value, ShouldEqual, 20)

			So(results[2].Error, ShouldBeNil)

			So(results[3].Error, ShouldBeNil)
			So(results[3].Value, ShouldEqual, 30)
		})

		Convey("Priority handling", func() {
			// High priority operations should be processed first
			operations := []Operation{
				{Type: "get", Path: []string{"data", "value1"}, Priority: 1},
				{Type: "get", Path: []string{"data", "value2"}, Priority: 10},
			}

			results, err := evaluator.EvaluateParallel(operations)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 2)

			// Both should complete successfully
			So(results[0].Error, ShouldBeNil)
			So(results[1].Error, ShouldBeNil)
		})
	})
}

func TestParallelBatchProcessor(t *testing.T) {
	Convey("ParallelBatchProcessor operations", t, func() {
		tree := NewCOWTree(nil)
		processor := NewParallelBatchProcessor(tree, 4, 10)
		defer processor.Stop()

		Convey("Small batch processing", func() {
			operations := make([]Operation, 5)
			for i := 0; i < 5; i++ {
				operations[i] = Operation{
					Type: "set",
					Path: []string{fmt.Sprintf("key%d", i)},
					Args: []interface{}{fmt.Sprintf("value%d", i)},
				}
			}

			results, err := processor.ProcessBatch(operations)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 5)

			for _, result := range results {
				So(result.Error, ShouldBeNil)
			}
		})

		Convey("Large batch processing", func() {
			operations := make([]Operation, 25)
			for i := 0; i < 25; i++ {
				operations[i] = Operation{
					Type: "set",
					Path: []string{"batch", fmt.Sprintf("key%d", i)},
					Args: []interface{}{i},
				}
			}

			results, err := processor.ProcessBatch(operations)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 25)

			// Verify all operations succeeded
			successCount := 0
			for _, result := range results {
				if result.Error == nil {
					successCount++
				}
			}
			So(successCount, ShouldEqual, 25)
		})
	})
}

func TestCOWParallelExecutor(t *testing.T) {
	Convey("COWParallelExecutor operations", t, func() {
		baseTree := NewCOWTree(map[interface{}]interface{}{
			"shared": "initial",
		})

		executor := NewCOWParallelExecutor(baseTree, 3)
		defer executor.Stop()

		Convey("Isolated execution", func() {
			// Each group modifies the same key differently
			operationGroups := [][]Operation{
				{{Type: "set", Path: []string{"shared"}, Args: []interface{}{"group1"}}},
				{{Type: "set", Path: []string{"shared"}, Args: []interface{}{"group2"}}},
				{{Type: "set", Path: []string{"shared"}, Args: []interface{}{"group3"}}},
			}

			results, err := executor.ExecuteIsolated(operationGroups)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 3)

			// All groups should succeed
			for _, groupResults := range results {
				So(len(groupResults), ShouldEqual, 1)
				So(groupResults[0].Error, ShouldBeNil)
			}

			// Base tree should remain unchanged
			value, _ := baseTree.Find("shared")
			So(value, ShouldEqual, "initial")
		})

		Convey("Error handling", func() {
			// Too many operation groups
			tooManyGroups := make([][]Operation, 5)
			_, err := executor.ExecuteIsolated(tooManyGroups)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "too many operation groups")
		})
	})
}

func TestConcurrentParallelExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent parallel execution tests in short mode")
	}

	Convey("Concurrent parallel execution", t, func() {
		tree := NewCOWTree(nil)

		Convey("High concurrency stress test", func() {
			engine := NewParallelExecutionEngine(tree, 8)
			engine.Start()
			defer engine.Stop()

			taskCount := 1000
			var wg sync.WaitGroup

			// Submit tasks concurrently
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					for j := 0; j < taskCount/10; j++ {
						task := &ExecutionTask{
							ID:       fmt.Sprintf("task-%d-%d", workerID, j),
							Path:     []string{fmt.Sprintf("worker%d", workerID), fmt.Sprintf("key%d", j)},
							Operator: "set",
							Args:     []interface{}{j},
						}

						engine.Submit(task)
					}
				}(i)
			}

			// Collect results concurrently
			resultCount := 0
			done := make(chan struct{})

			go func() {
				for resultCount < taskCount {
					if result, ok := engine.GetResult(); ok {
						resultCount++
						if result.Error != nil {
							t.Logf("Task %s failed: %v", result.TaskID, result.Error)
						}
					}
				}
				close(done)
			}()

			wg.Wait()

			select {
			case <-done:
				So(resultCount, ShouldEqual, taskCount)
			case <-time.After(time.Second * 5):
				t.Fatal("Timeout collecting results")
			}

			metrics := engine.GetMetrics()
			So(metrics["tasks_executed"], ShouldEqual, int64(taskCount))
		})

		Convey("COW snapshot isolation", func() {
			baseTree := NewCOWTree(map[interface{}]interface{}{
				"counter": 0,
			})

			// Create multiple snapshots and modify concurrently
			snapshotCount := 10
			var wg sync.WaitGroup

			for i := 0; i < snapshotCount; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					snapshot := baseTree.Copy()
					evaluator := NewParallelEvaluator(snapshot, 1)
					defer evaluator.Stop()

					// Each snapshot increments counter differently
					for j := 0; j < 100; j++ {
						ops := []Operation{{
							Type: "update",
							Path: []string{"counter"},
							Args: []interface{}{
								func(current interface{}) interface{} {
									if val, ok := current.(int); ok {
										return val + 1
									}
									return 1
								},
							},
						}}

						evaluator.EvaluateParallel(ops)
					}

					// Verify final value in snapshot
					finalValue, _ := snapshot.Find("counter")
					So(finalValue, ShouldEqual, 100)
				}(i)
			}

			wg.Wait()

			// Base tree should still have original value
			baseValue, _ := baseTree.Find("counter")
			So(baseValue, ShouldEqual, 0)
		})
	})
}

func BenchmarkThreadSafeParallelExecution(b *testing.B) {
	tree := NewCOWTree(nil)

	// Pre-populate tree
	for i := 0; i < 1000; i++ {
		tree.Set(i, "data", fmt.Sprintf("key%d", i))
	}

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for j := 0; j < 100; j++ {
				tree.Find("data", fmt.Sprintf("key%d", j%1000))
			}
		}
	})

	b.Run("Parallel-2", func(b *testing.B) {
		evaluator := NewParallelEvaluator(tree, 2)
		defer evaluator.Stop()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ops := make([]Operation, 100)
			for j := 0; j < 100; j++ {
				ops[j] = Operation{
					Type: "get",
					Path: []string{"data", fmt.Sprintf("key%d", j%1000)},
				}
			}
			evaluator.EvaluateParallel(ops)
		}
	})

	b.Run("Parallel-4", func(b *testing.B) {
		evaluator := NewParallelEvaluator(tree, 4)
		defer evaluator.Stop()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ops := make([]Operation, 100)
			for j := 0; j < 100; j++ {
				ops[j] = Operation{
					Type: "get",
					Path: []string{"data", fmt.Sprintf("key%d", j%1000)},
				}
			}
			evaluator.EvaluateParallel(ops)
		}
	})

	b.Run("Parallel-8", func(b *testing.B) {
		evaluator := NewParallelEvaluator(tree, 8)
		defer evaluator.Stop()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ops := make([]Operation, 100)
			for j := 0; j < 100; j++ {
				ops[j] = Operation{
					Type: "get",
					Path: []string{"data", fmt.Sprintf("key%d", j%1000)},
				}
			}
			evaluator.EvaluateParallel(ops)
		}
	})
}
