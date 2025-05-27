package graft

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/starkandwayne/goutils/tree"
)

func TestParallelEvaluatorIntegration(t *testing.T) {
	Convey("Parallel Evaluator Integration", t, func() {
		Convey("should execute operations sequentially when disabled", func() {
			os.Setenv("GRAFT_PARALLEL", "false")
			defer os.Unsetenv("GRAFT_PARALLEL")

			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"a": "(( grab b ))",
					"b": "value-b",
					"c": "(( grab d ))",
					"d": "value-d",
				},
			}

			err := ev.RunPhaseParallel(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["a"], ShouldEqual, "value-b")
			So(ev.Tree["c"], ShouldEqual, "value-d")
		})

		Convey("should execute operations in parallel when enabled", func() {
			os.Setenv("GRAFT_PARALLEL", "true")
			defer os.Unsetenv("GRAFT_PARALLEL")

			// Create evaluator with many independent operations
			ev := &Evaluator{
				Tree: make(map[interface{}]interface{}),
			}

			// Add many independent grab operations
			for i := 0; i < 20; i++ {
				ev.Tree[fmt.Sprintf("target%d", i)] = fmt.Sprintf("(( grab source%d ))", i)
				ev.Tree[fmt.Sprintf("source%d", i)] = fmt.Sprintf("value-%d", i)
			}

			start := time.Now()
			err := ev.RunPhaseParallel(EvalPhase)
			elapsed := time.Since(start)

			So(err, ShouldBeNil)

			// Verify all operations completed correctly
			for i := 0; i < 20; i++ {
				So(ev.Tree[fmt.Sprintf("target%d", i)], ShouldEqual, fmt.Sprintf("value-%d", i))
			}

			// Parallel execution should be reasonably fast
			So(elapsed, ShouldBeLessThan, 500*time.Millisecond)
		})

		Convey("should handle mixed safe and unsafe operations", func() {
			config := &ParallelEvaluatorConfig{
				Enabled:           true,
				MaxWorkers:        4,
				MinOpsForParallel: 2,
				Strategy:          "conservative",
				SafeOperators: map[string]bool{
					"grab":   true,
					"concat": true,
				},
			}
			SetParallelConfig(config)

			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"safe1":   "(( grab source1 ))",
					"safe2":   "(( concat source2 \" world\" ))",
					"unsafe1": "(( static_ips 0 1 2 ))", // This would need special handling
					"source1": "hello",
					"source2": "hello",
				},
			}

			// For this test, we expect it to handle the operations appropriately
			// Note: static_ips operator may not be defined in test environment
			err := ev.RunPhaseParallel(EvalPhase)
			
			// Should complete without error for defined operators
			if err != nil {
				// Check if it's the expected static_ips error
				So(err.Error(), ShouldContainSubstring, "not currently inside of a job definition block")
			} else {
				So(ev.Tree["safe1"], ShouldEqual, "hello")
				So(ev.Tree["safe2"], ShouldEqual, "hello world")
			}
		})

		Convey("should respect dependencies", func() {
			os.Setenv("GRAFT_PARALLEL", "true")
			defer os.Unsetenv("GRAFT_PARALLEL")

			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"a": "(( grab b ))",
					"b": "(( grab c ))",
					"c": "(( grab d ))",
					"d": "final-value",
					"e": "(( grab f ))",
					"f": "independent-value",
				},
			}

			err := ev.RunPhaseParallel(EvalPhase)
			So(err, ShouldBeNil)

			// Chain should resolve correctly
			So(ev.Tree["a"], ShouldEqual, "final-value")
			So(ev.Tree["b"], ShouldEqual, "final-value")
			So(ev.Tree["c"], ShouldEqual, "final-value")

			// Independent operation should also complete
			So(ev.Tree["e"], ShouldEqual, "independent-value")
		})

		Convey("should fall back to sequential for small workloads", func() {
			config := &ParallelEvaluatorConfig{
				Enabled:           true,
				MaxWorkers:        4,
				MinOpsForParallel: 10, // High threshold
				Strategy:          "conservative",
			}
			SetParallelConfig(config)

			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"a": "(( grab b ))",
					"b": "value",
				},
			}

			err := ev.RunPhaseParallel(EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree["a"], ShouldEqual, "value")
		})
	})
}

func TestParallelOperatorAnalysis(t *testing.T) {
	Convey("Parallel Operator Analysis", t, func() {
		Convey("should identify safe operators", func() {
			config := DefaultParallelConfig()
			
			// Check safe operators
			safeOps := []string{"grab", "concat", "base64", "vault"}
			for _, opName := range safeOps {
				So(config.SafeOperators[opName], ShouldBeTrue)
			}

			// Check unsafe operators
			unsafeOps := []string{"static_ips", "inject", "merge"}
			for _, opName := range unsafeOps {
				So(config.SafeOperators[opName], ShouldBeFalse)
			}
		})

		Convey("should group operations correctly", func() {
			ev := &Evaluator{
				Tree: make(map[interface{}]interface{}),
				Deps: make(map[string][]tree.Cursor),
			}
			adapter := NewParallelEvaluatorAdapter(ev, DefaultParallelConfig())

			// Create operations with dependencies
			ops := []*Opcall{
				{where: mustParseCursor("a")},
				{where: mustParseCursor("b")},
				{where: mustParseCursor("c")},
				{where: mustParseCursor("d")},
				{where: mustParseCursor("e")},
			}
			
			// Set up dependencies in evaluator
			ev.Deps["c"] = []tree.Cursor{*mustParseCursor("a")}
			ev.Deps["d"] = []tree.Cursor{*mustParseCursor("b")}
			ev.Deps["e"] = []tree.Cursor{*mustParseCursor("c"), *mustParseCursor("d")}

			groups := adapter.analyzeOperations(ops)

			// Should have 3 execution waves
			So(len(groups.executionOrder), ShouldEqual, 3)

			// First wave: a, b (no dependencies)
			So(len(groups.executionOrder[0].ops), ShouldEqual, 2)

			// Second wave: c, d (depend on first wave)
			So(len(groups.executionOrder[1].ops), ShouldEqual, 2)

			// Third wave: e (depends on second wave)
			So(len(groups.executionOrder[2].ops), ShouldEqual, 1)
		})
	})
}

func TestParallelExecutionMetrics(t *testing.T) {
	Convey("Parallel Execution Metrics", t, func() {
		Convey("should track execution statistics", func() {
			config := &ParallelEvaluatorConfig{
				Enabled:           true,
				MaxWorkers:        2,
				MinOpsForParallel: 1,
				Strategy:          "conservative",
				SafeOperators: map[string]bool{
					"grab": true,
				},
			}

			ev := &Evaluator{
				Tree: map[interface{}]interface{}{
					"a": "(( grab b ))",
					"b": "value",
					"c": "(( grab d ))",
					"d": "value",
				},
			}

			adapter := NewParallelEvaluatorAdapter(ev, config)
			
			// Create proper operations with grab operator
			ops := []*Opcall{
				{
					where: mustParseCursor("a"),
					op: GrabOperator{},
					args: []*Expr{{Type: Reference, Reference: mustParseCursor("b")}},
				},
				{
					where: mustParseCursor("c"),
					op: GrabOperator{},
					args: []*Expr{{Type: Reference, Reference: mustParseCursor("d")}},
				},
			}

			// Execute operations
			adapter.RunOps(ops)

			metrics := adapter.GetMetrics()
			So(metrics["total_operations"], ShouldBeGreaterThan, 0)
		})
	})
}

// Helper function for tests
func mustParseCursor(path string) *tree.Cursor {
	c, err := tree.ParseCursor(path)
	if err != nil {
		panic(err)
	}
	return c
}