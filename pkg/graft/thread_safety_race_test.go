//go:build race
// +build race

package graft

import (
	"fmt"
	"sync"
	"testing"
	"time"

	tree "github.com/geofffranks/yaml"
	. "github.com/smartystreets/goconvey/convey"
)

// TestTreeRaceConditions specifically tests for race conditions in tree operations
func TestTreeRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition tests in short mode")
	}

	Convey("Tree operations under concurrent access", t, func() {
		Convey("Concurrent reads and writes to same path", func() {
			data := make(map[interface{}]interface{})
			data["meta"] = map[interface{}]interface{}{
				"name": "initial",
			}
			tree := NewSafeTree(data)

			var wg sync.WaitGroup
			errors := make(chan error, 100)

			// Writers
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 100; j++ {
						err := tree.Replace(tree.Cursor(), map[interface{}]interface{}{
							"meta": map[interface{}]interface{}{
								"name": fmt.Sprintf("writer-%d-%d", id, j),
							},
						})
						if err != nil {
							errors <- err
						}
					}
				}(i)
			}

			// Readers
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 100; j++ {
						val := tree.Find("meta", "name")
						if val == nil {
							errors <- fmt.Errorf("reader %d: nil value at iteration %d", id, j)
						}
					}
				}(i)
			}

			wg.Wait()
			close(errors)

			// Check for errors
			var errorCount int
			for err := range errors {
				t.Logf("Race condition error: %v", err)
				errorCount++
			}

			// We expect race conditions with the current implementation
			So(errorCount, ShouldBeGreaterThan, 0)
		})

		Convey("Concurrent modifications to different paths", func() {
			data := make(map[interface{}]interface{})
			tree := NewSafeTree(data)

			var wg sync.WaitGroup

			// Each goroutine modifies its own section
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					section := fmt.Sprintf("section%d", id)

					for j := 0; j < 100; j++ {
						tree.Replace(map[interface{}]interface{}{
							section: map[interface{}]interface{}{
								fmt.Sprintf("key%d", j): fmt.Sprintf("value%d", j),
							},
						})
					}
				}(i)
			}

			wg.Wait()

			// Verify all sections exist
			for i := 0; i < 10; i++ {
				section := fmt.Sprintf("section%d", i)
				val := tree.Find(section)
				So(val, ShouldNotBeNil)
			}
		})

		Convey("Nested map concurrent access", func() {
			tree := &Tree{data: make(map[interface{}]interface{})}
			tree.data["root"] = map[interface{}]interface{}{
				"level1": map[interface{}]interface{}{
					"level2": map[interface{}]interface{}{
						"value": 0,
					},
				},
			}

			var wg sync.WaitGroup

			// Multiple goroutines accessing nested paths
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := 0; j < 50; j++ {
						// Read deep value
						val := tree.Find("root", "level1", "level2", "value")
						_ = val

						// Modify deep value
						tree.Replace(map[interface{}]interface{}{
							"root": map[interface{}]interface{}{
								"level1": map[interface{}]interface{}{
									"level2": map[interface{}]interface{}{
										"value": id*1000 + j,
									},
								},
							},
						})
					}
				}(i)
			}

			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				// Success
			case <-time.After(5 * time.Second):
				t.Fatal("Test timed out - possible deadlock")
			}
		})
	})
}

// TestEvaluatorRaceConditions tests for race conditions in evaluator
func TestEvaluatorRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition tests in short mode")
	}

	Convey("Evaluator operations under concurrent access", t, func() {
		Convey("Multiple evaluators on same tree", func() {
			yaml := `
meta:
  name: test
  count: 1
results:
  - name: (( grab meta.name ))
    value: (( meta.count + 1 ))
  - name: (( concat meta.name "-2" ))
    value: (( meta.count * 2 ))
`

			var wg sync.WaitGroup
			errors := make(chan error, 100)

			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					tree, err := ParseYAML([]byte(yaml))
					if err != nil {
						errors <- err
						return
					}

					ev := &Evaluator{Tree: tree}
					err = ev.Run([]string{}, []string{})
					if err != nil {
						errors <- fmt.Errorf("evaluator %d: %v", id, err)
					}
				}(i)
			}

			wg.Wait()
			close(errors)

			errorCount := 0
			for err := range errors {
				t.Logf("Evaluator error: %v", err)
				errorCount++
			}

			So(errorCount, ShouldEqual, 0)
		})

		Convey("Concurrent operator execution", func() {
			// This test would fail with ParallelEvaluator enabled
			t.Skip("ParallelEvaluator is currently disabled due to race conditions")
		})
	})
}

// TestOperatorRaceConditions tests individual operators for thread safety
func TestOperatorRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition tests in short mode")
	}

	Convey("Operator thread safety", t, func() {
		Convey("Grab operator concurrent execution", func() {
			tree := &Tree{data: make(map[interface{}]interface{})}
			tree.data["source"] = map[interface{}]interface{}{
				"value": "test",
			}

			ev := &Evaluator{Tree: tree}
			op := GrabOperator{}

			var wg sync.WaitGroup
			results := make(chan interface{}, 100)

			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < 10; j++ {
						args := []*Expr{{Type: Literal, Literal: "source.value"}}
						result, err := op.Run(ev, args)
						if err == nil {
							results <- result
						}
					}
				}()
			}

			wg.Wait()
			close(results)

			// All results should be "test"
			for result := range results {
				So(result, ShouldEqual, "test")
			}
		})

		Convey("Calc operator with shared state", func() {
			tree := &Tree{data: make(map[interface{}]interface{})}
			tree.data["counter"] = 0

			ev := &Evaluator{Tree: tree}

			var wg sync.WaitGroup
			var finalValue int

			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < 100; j++ {
						// Simulate increment operation
						current := tree.Find("counter")
						if val, ok := current.(int); ok {
							tree.Replace(map[interface{}]interface{}{
								"counter": val + 1,
							})
						}
					}
				}()
			}

			wg.Wait()

			finalValue = tree.Find("counter").(int)
			// With race conditions, final value will be less than 1000
			t.Logf("Final counter value: %d (expected 1000)", finalValue)
			So(finalValue, ShouldBeLessThan, 1000)
		})
	})
}

// TestDeadlockScenarios tests for potential deadlock situations
func TestDeadlockScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping deadlock tests in short mode")
	}

	Convey("Deadlock detection", t, func() {
		Convey("Circular operator dependencies", func() {
			yaml := `
a: (( grab b ))
b: (( grab c ))
c: (( grab a ))
`
			tree, err := ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			ev := &Evaluator{Tree: tree}

			done := make(chan error, 1)
			go func() {
				done <- ev.Run([]string{}, []string{})
			}()

			select {
			case err := <-done:
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "cycle")
			case <-time.After(2 * time.Second):
				// Current implementation detects cycles, so this shouldn't happen
				t.Fatal("Deadlock detected - circular dependency not caught")
			}
		})

		Convey("Concurrent evaluator with interdependencies", func() {
			yaml := `
shared:
  value: 1
workers:
  - id: 1
    value: (( shared.value + 1 ))
  - id: 2
    value: (( shared.value + 2 ))
  - id: 3
    value: (( shared.value + 3 ))
`
			tree, err := ParseYAML([]byte(yaml))
			So(err, ShouldBeNil)

			// Multiple evaluators trying to resolve same dependencies
			var wg sync.WaitGroup
			for i := 0; i < 3; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					ev := &Evaluator{Tree: tree.Copy()}
					ev.Run([]string{}, []string{})
				}()
			}

			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				// Success - no deadlock
			case <-time.After(5 * time.Second):
				t.Fatal("Potential deadlock in concurrent evaluation")
			}
		})
	})
}
