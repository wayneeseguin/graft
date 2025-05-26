package spruce

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSmartBatching(t *testing.T) {
	Convey("Smart Batching & Ordering", t, func() {
		Convey("Dependency Graph", func() {
			graph := NewDependencyGraph()
			
			Convey("Should build dependency graph", func() {
				// Add nodes
				graph.AddNode("a", "(( grab b ))", "grab", []string{"a"})
				graph.AddNode("b", "value", "literal", []string{"b"})
				graph.AddNode("c", "(( concat a \" suffix\" ))", "concat", []string{"c"})
				
				// Add dependencies
				err := graph.AddDependency("b", "a") // a depends on b
				So(err, ShouldBeNil)
				
				err = graph.AddDependency("a", "c") // c depends on a
				So(err, ShouldBeNil)
				
				So(graph.Size(), ShouldEqual, 3)
			})
			
			Convey("Should detect circular dependencies", func() {
				graph.AddNode("x", "(( grab y ))", "grab", []string{"x"})
				graph.AddNode("y", "(( grab z ))", "grab", []string{"y"})
				graph.AddNode("z", "(( grab x ))", "grab", []string{"z"})
				
				err := graph.AddDependency("y", "x")
				So(err, ShouldBeNil)
				
				err = graph.AddDependency("z", "y")
				So(err, ShouldBeNil)
				
				err = graph.AddDependency("x", "z")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "circular dependency")
			})
			
			Convey("Should perform topological sort", func() {
				graph.AddNode("1", "one", "literal", []string{"1"})
				graph.AddNode("2", "two", "literal", []string{"2"})
				graph.AddNode("3", "three", "literal", []string{"3"})
				
				graph.AddDependency("1", "2")
				graph.AddDependency("2", "3")
				
				sorted, err := graph.TopologicalSort()
				So(err, ShouldBeNil)
				So(len(sorted), ShouldEqual, 3)
				So(sorted[0].ID, ShouldEqual, "1")
				So(sorted[1].ID, ShouldEqual, "2")
				So(sorted[2].ID, ShouldEqual, "3")
			})
			
			Convey("Should get execution stages", func() {
				// Create diamond dependency
				graph.AddNode("top", "top", "literal", nil)
				graph.AddNode("left", "(( grab top ))", "grab", nil)
				graph.AddNode("right", "(( grab top ))", "grab", nil)
				graph.AddNode("bottom", "(( concat left right ))", "concat", nil)
				
				graph.AddDependency("top", "left")
				graph.AddDependency("top", "right")
				graph.AddDependency("left", "bottom")
				graph.AddDependency("right", "bottom")
				
				stages, err := graph.GetExecutionStages()
				So(err, ShouldBeNil)
				So(len(stages), ShouldEqual, 3)
				
				// Stage 0: top
				So(len(stages[0].Operations), ShouldEqual, 1)
				So(stages[0].Operations[0].ID, ShouldEqual, "top")
				
				// Stage 1: left and right (parallel)
				So(len(stages[1].Operations), ShouldEqual, 2)
				So(stages[1].CanParallel, ShouldBeTrue)
				
				// Stage 2: bottom
				So(len(stages[2].Operations), ShouldEqual, 1)
				So(stages[2].Operations[0].ID, ShouldEqual, "bottom")
			})
		})
		
		Convey("Dependency Analyzer", func() {
			analyzer := NewDependencyAnalyzer()
			
			Convey("Should analyze document dependencies", func() {
				doc := map[string]interface{}{
					"foo": "(( grab bar ))",
					"bar": "value",
					"baz": "(( concat foo \" suffix\" ))",
				}
				
				graph, err := analyzer.AnalyzeDocument(doc)
				So(err, ShouldBeNil)
				So(graph.Size(), ShouldEqual, 4) // 3 fields + 1 reference node
				
				// Check dependencies
				fooNode, _ := graph.GetNode("foo")
				So(len(fooNode.Dependencies), ShouldEqual, 1)
				So(fooNode.Dependencies["bar"], ShouldBeTrue)
			})
			
			Convey("Should extract operator types", func() {
				opType := analyzer.extractOperatorType("(( grab foo.bar ))")
				So(opType, ShouldEqual, "grab")
				
				opType = analyzer.extractOperatorType("(( vault \"secret/path:key\" ))")
				So(opType, ShouldEqual, "vault")
			})
		})
		
		Convey("Cost Estimator", func() {
			estimator := NewCostEstimator()
			
			Convey("Should estimate operator costs", func() {
				cost := estimator.EstimateOperatorCost("grab", "(( grab simple ))")
				So(cost, ShouldBeGreaterThan, 0)
				So(cost, ShouldBeLessThan, 1)
				
				vaultCost := estimator.EstimateOperatorCost("vault", "(( vault \"path\" ))")
				So(vaultCost, ShouldBeGreaterThan, cost)
			})
			
			Convey("Should identify batchable operations", func() {
				So(estimator.canBatch("vault"), ShouldBeTrue)
				So(estimator.canBatch("file"), ShouldBeTrue)
				So(estimator.canBatch("grab"), ShouldBeFalse)
			})
		})
		
		Convey("Operation Batcher", func() {
			batcher := NewOperationBatcher(nil)
			
			Convey("Should create batches from operations", func() {
				ops := []*DependencyNode{
					{ID: "1", OperatorType: "vault"},
					{ID: "2", OperatorType: "vault"},
					{ID: "3", OperatorType: "file"},
					{ID: "4", OperatorType: "vault"},
				}
				
				batches := batcher.CreateBatches(ops)
				So(len(batches), ShouldBeGreaterThan, 0)
				
				// Should batch vault operations together
				vaultBatchFound := false
				for _, batch := range batches {
					if batch.Type == "vault" && len(batch.Operations) > 1 {
						vaultBatchFound = true
					}
				}
				So(vaultBatchFound, ShouldBeTrue)
			})
			
			Convey("Should respect batch size limits", func() {
				config := &BatchConfig{
					MaxBatchSize: 2,
					MinBatchSize: 1,
					BatchByType:  true,
				}
				batcher := NewOperationBatcher(config)
				
				ops := make([]*DependencyNode, 5)
				for i := 0; i < 5; i++ {
					ops[i] = &DependencyNode{
						ID:           fmt.Sprintf("op%d", i),
						OperatorType: "vault",
					}
				}
				
				batches := batcher.CreateBatches(ops)
				
				// Should create multiple batches due to size limit
				So(len(batches), ShouldBeGreaterThanOrEqualTo, 2)
				for _, batch := range batches {
					So(len(batch.Operations), ShouldBeLessThanOrEqualTo, 2)
				}
			})
		})
		
		Convey("Early Termination", func() {
			graph := NewDependencyGraph()
			graph.AddNode("a", "a", "literal", []string{"a"})
			graph.AddNode("b", "(( grab a ))", "grab", []string{"b"})
			graph.AddNode("c", "(( grab b ))", "grab", []string{"c"})
			graph.AddNode("unused", "(( grab a ))", "grab", []string{"unused"})
			
			graph.AddDependency("a", "b")
			graph.AddDependency("b", "c")
			graph.AddDependency("a", "unused")
			
			terminator := NewEarlyTerminator(graph, nil)
			
			Convey("Should identify necessary operations", func() {
				// Only 'c' is needed in output
				terminator.AnalyzeNecessity([]string{"c"})
				
				So(terminator.ShouldEvaluate("a"), ShouldBeTrue)     // Needed by b
				So(terminator.ShouldEvaluate("b"), ShouldBeTrue)     // Needed by c
				So(terminator.ShouldEvaluate("c"), ShouldBeTrue)     // Output
				So(terminator.ShouldEvaluate("unused"), ShouldBeFalse) // Not needed
			})
			
			Convey("Should track unused paths", func() {
				terminator.AnalyzeNecessity([]string{"c", "unused"})
				
				// Mark some as used
				terminator.necessityTracker.MarkUsed("a")
				terminator.necessityTracker.MarkUsed("c")
				
				unused := terminator.AnalyzeUnusedPaths()
				So(len(unused), ShouldBeGreaterThan, 0)
			})
		})
		
		Convey("Execution Planner", func() {
			planner := NewExecutionPlanner(nil)
			
			Convey("Should create execution plan", func() {
				doc := map[string]interface{}{
					"a": "value",
					"b": "(( grab a ))",
					"c": "(( vault \"secret/path\" ))",
					"d": "(( vault \"secret/other\" ))",
				}
				
				ctx := context.Background()
				plan, err := planner.PlanExecution(ctx, doc)
				So(err, ShouldBeNil)
				So(plan, ShouldNotBeNil)
				So(len(plan.Stages), ShouldBeGreaterThan, 0)
				So(plan.OptimizedCost, ShouldBeLessThanOrEqualTo, plan.OriginalCost)
			})
		})
	})
}

func TestBatchCollector(t *testing.T) {
	Convey("Batch Collector", t, func() {
		config := &BatchConfig{
			MaxBatchSize: 3,
			MaxWaitTime:  50 * time.Millisecond,
			MinBatchSize: 2,
		}
		
		collector := NewBatchCollector(config)
		
		Convey("Should collect operations into batches", func() {
			ctx := context.Background()
			
			op1 := &DependencyNode{ID: "1", OperatorType: "vault"}
			op2 := &DependencyNode{ID: "2", OperatorType: "vault"}
			
			// Collect operations
			ch1 := collector.Collect(ctx, op1)
			ch2 := collector.Collect(ctx, op2)
			
			// Should receive batch after timeout
			select {
			case batch := <-ch1:
				So(batch, ShouldNotBeNil)
				So(len(batch.Operations), ShouldEqual, 2)
			case <-time.After(100 * time.Millisecond):
				So(false, ShouldBeTrue) // Should not timeout
			}
			
			// Second operation should get same batch
			select {
			case batch := <-ch2:
				So(batch, ShouldNotBeNil)
				So(len(batch.Operations), ShouldEqual, 2)
			case <-time.After(10 * time.Millisecond):
				So(false, ShouldBeTrue)
			}
		})
		
		Convey("Should flush when batch is full", func() {
			ctx := context.Background()
			
			ops := make([]*DependencyNode, 3)
			chs := make([]<-chan *OperationBatch, 3)
			
			for i := 0; i < 3; i++ {
				ops[i] = &DependencyNode{
					ID:           fmt.Sprintf("%d", i),
					OperatorType: "file",
				}
				chs[i] = collector.Collect(ctx, ops[i])
			}
			
			// Should receive batch immediately (full)
			select {
			case batch := <-chs[0]:
				So(batch, ShouldNotBeNil)
				So(len(batch.Operations), ShouldEqual, 3)
			case <-time.After(10 * time.Millisecond):
				So(false, ShouldBeTrue) // Should not timeout
			}
		})
	})
}