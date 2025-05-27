package spruce

import (
	"fmt"
	"sync"
	"testing"
	
	. "github.com/smartystreets/goconvey/convey"
)

func TestThreadSafeParallelEngineSimple(t *testing.T) {
	Convey("Simple Parallel Execution Engine Tests", t, func() {
		Convey("Basic operations", func() {
			tree := NewSafeTree(map[interface{}]interface{}{
				"test": "value",
			})

			engine := NewParallelExecutionEngine(tree, 2)
			engine.Start()
			defer engine.Stop()

			// Test set operation
			task := &ExecutionTask{
				ID:       "task-1",
				Path:     []string{"new"},
				Operator: "set",
				Args:     []interface{}{"new-value"},
			}
			
			err := engine.Submit(task)
			So(err, ShouldBeNil)

			// Get result
			result, ok := engine.GetResult()
			So(ok, ShouldBeTrue)
			So(result.TaskID, ShouldEqual, "task-1")
			So(result.Error, ShouldBeNil)

			// Verify value was set
			value, err := tree.Find("new")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "new-value")
		})

		Convey("Thread-safe parallel evaluator", func() {
			tree := NewSafeTree(nil)
			evaluator := NewThreadSafeParallelEvaluator(tree, 4)
			defer evaluator.Stop()

			operations := []Operation{
				{Type: "set", Path: []string{"a"}, Args: []interface{}{"value-a"}},
				{Type: "set", Path: []string{"b"}, Args: []interface{}{"value-b"}},
			}

			results, err := evaluator.EvaluateParallel(operations)
			So(err, ShouldBeNil)
			So(len(results), ShouldEqual, 2)

			// Verify values in tree
			value, _ := tree.Find("a")
			So(value, ShouldEqual, "value-a")
			value, _ = tree.Find("b")
			So(value, ShouldEqual, "value-b")
		})

		Convey("Concurrent safety", func() {
			tree := NewSafeTree(nil)
			engine := NewParallelExecutionEngine(tree, 4)
			engine.Start()
			defer engine.Stop()

			// Submit many tasks concurrently
			var wg sync.WaitGroup
			numTasks := 50
			
			for i := 0; i < numTasks; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					
					task := &ExecutionTask{
						ID:       fmt.Sprintf("task-%d", idx),
						Path:     []string{fmt.Sprintf("key-%d", idx)},
						Operator: "set",
						Args:     []interface{}{fmt.Sprintf("value-%d", idx)},
					}
					
					engine.Submit(task)
				}(i)
			}
			
			wg.Wait()

			// Collect all results
			for i := 0; i < numTasks; i++ {
				result, ok := engine.GetResult()
				So(ok, ShouldBeTrue)
				So(result.Error, ShouldBeNil)
			}
		})
	})
}