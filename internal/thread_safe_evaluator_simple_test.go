package internal

import (
	"context"
	"fmt"
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestThreadSafeEvaluatorSimple(t *testing.T) {
	Convey("ThreadSafeEvaluatorSimple operations", t, func() {
		tree := NewSafeTree(map[interface{}]interface{}{
			"meta": map[interface{}]interface{}{
				"name":  "test",
				"value": 42,
			},
		})

		evaluator := NewThreadSafeEvaluatorSimple(tree)

		Convey("Basic evaluation", func() {
			ctx := context.Background()
			err := evaluator.Evaluate(ctx)
			So(err, ShouldBeNil)
		})

		Convey("Tree access", func() {
			safeTree := evaluator.GetTree()
			So(safeTree, ShouldNotBeNil)

			value, err := safeTree.Find("meta", "name")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "test")
		})

		Convey("Value operations", func() {
			err := evaluator.SetValue("new-value", "meta", "newkey")
			So(err, ShouldBeNil)

			value, err := evaluator.GetValue("meta", "newkey")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, "new-value")
		})
	})
}

func TestMigrationHelperSimple(t *testing.T) {
	Convey("MigrationHelperSimple operations", t, func() {
		originalData := map[interface{}]interface{}{
			"meta": map[interface{}]interface{}{
				"name": "original",
			},
		}

		helper := NewMigrationHelperSimple(originalData)

		Convey("Basic helper operations", func() {
			safeTree := helper.GetThreadSafeTree()
			So(safeTree, ShouldNotBeNil)

			evaluator := helper.GetThreadSafeEvaluator()
			So(evaluator, ShouldNotBeNil)
		})

		Convey("Evaluator export/import", func() {
			// Export to traditional evaluator
			ev, err := helper.ExportToEvaluator()
			So(err, ShouldBeNil)
			So(ev, ShouldNotBeNil)
			So(ev.Tree, ShouldNotBeNil)

			// Modify the evaluator
			ev.Tree["meta"].(map[interface{}]interface{})["modified"] = true

			// Update helper from evaluator
			err = helper.UpdateFromEvaluator(ev)
			So(err, ShouldBeNil)

			// Verify change was applied
			value, err := helper.GetThreadSafeTree().Find("meta", "modified")
			So(err, ShouldBeNil)
			So(value, ShouldEqual, true)
		})
	})
}

func TestSimpleEvaluatorConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}

	Convey("ThreadSafeEvaluatorSimple concurrent operations", t, func() {
		tree := NewSafeTree(map[interface{}]interface{}{
			"counters": map[interface{}]interface{}{},
		})

		evaluator := NewThreadSafeEvaluatorSimple(tree)

		Convey("Concurrent evaluation and tree access", func() {
			var wg sync.WaitGroup
			errors := make(chan error, 20)

			// Multiple evaluations
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					ctx := context.Background()
					if err := evaluator.Evaluate(ctx); err != nil {
						errors <- fmt.Errorf("evaluation %d: %v", id, err)
					}
				}(i)
			}

			// Multiple tree operations
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					// Set values
					key := fmt.Sprintf("counter%d", id)
					if err := evaluator.SetValue(id, "counters", key); err != nil {
						errors <- fmt.Errorf("set %d: %v", id, err)
						return
					}

					// Get values
					if _, err := evaluator.GetValue("counters", key); err != nil {
						errors <- fmt.Errorf("get %d: %v", id, err)
					}
				}(i)
			}

			wg.Wait()
			close(errors)

			// Should have no errors
			var errorCount int
			for err := range errors {
				t.Logf("Concurrency error: %v", err)
				errorCount++
			}

			So(errorCount, ShouldEqual, 0)
		})

		Convey("Migration helper under concurrent access", func() {
			helper := NewMigrationHelperSimple(map[interface{}]interface{}{
				"shared": map[interface{}]interface{}{
					"counter": 0,
				},
			})

			var wg sync.WaitGroup

			// Multiple workers using the helper
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					// Get evaluator
					ev, err := helper.ExportToEvaluator()
					if err != nil {
						return
					}

					// Modify and update back
					if shared, ok := ev.Tree["shared"].(map[interface{}]interface{}); ok {
						if counter, ok := shared["counter"].(int); ok {
							shared["counter"] = counter + 1
						}
					}

					helper.UpdateFromEvaluator(ev)
				}(i)
			}

			wg.Wait()

			// Final counter should be > 0 (exact value depends on race conditions)
			finalValue, err := helper.GetThreadSafeTree().Find("shared", "counter")
			So(err, ShouldBeNil)
			So(finalValue.(int), ShouldBeGreaterThan, 0)
		})
	})
}

// Benchmark the simplified evaluator
func BenchmarkSimpleEvaluator(b *testing.B) {
	tree := NewSafeTree(map[interface{}]interface{}{
		"meta": map[interface{}]interface{}{
			"name": "benchmark",
		},
	})

	evaluator := NewThreadSafeEvaluatorSimple(tree)
	ctx := context.Background()

	b.Run("Evaluate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evaluator.Evaluate(ctx)
		}
	})

	b.Run("SetValue", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			evaluator.SetValue(i, "data", fmt.Sprintf("key%d", i))
		}
	})

	b.Run("GetValue", func(b *testing.B) {
		evaluator.SetValue("test", "benchmark", "key")
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			evaluator.GetValue("benchmark", "key")
		}
	})
}
