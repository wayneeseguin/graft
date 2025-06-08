package graft

import (
	"testing"
	
	. "github.com/smartystreets/goconvey/convey"
	"github.com/wayneeseguin/graft/internal/utils/tree"
)

func TestLazyEvaluator(t *testing.T) {
	Convey("Lazy Evaluation System", t, func() {
		
		Convey("Basic lazy expression creation", func() {
			expr := &Expr{
				Type:    Literal,
				Literal: "test value",
			}
			
			evaluator := &Evaluator{Tree: make(map[interface{}]interface{})}
			lazy := NewLazyExpression(expr, evaluator)
			
			So(lazy, ShouldNotBeNil)
			So(lazy.IsEvaluated(), ShouldBeFalse)
		})
		
		Convey("Lazy expression evaluation", func() {
			expr := &Expr{
				Type:    Literal,
				Literal: 42,
			}
			
			evaluator := &Evaluator{Tree: make(map[interface{}]interface{})}
			lazy := NewLazyExpression(expr, evaluator)
			
			// Should not be evaluated initially
			So(lazy.IsEvaluated(), ShouldBeFalse)
			
			// Evaluate the expression
			result, err := lazy.Evaluate()
			So(err, ShouldBeNil)
			So(result, ShouldEqual, 42)
			So(lazy.IsEvaluated(), ShouldBeTrue)
			
			// Second evaluation should return cached result
			result2, err2 := lazy.Evaluate()
			So(err2, ShouldBeNil)
			So(result2, ShouldEqual, 42)
		})
		
		Convey("Reference expression lazy evaluation", func() {
			// Create a test tree
			testTree := map[interface{}]interface{}{
				"meta": map[interface{}]interface{}{
					"key": "test-value",
				},
			}
			
			cursor, err := tree.ParseCursor("meta.key")
			So(err, ShouldBeNil)
			
			expr := &Expr{
				Type:      Reference,
				Reference: cursor,
			}
			
			evaluator := &Evaluator{Tree: testTree}
			lazy := NewLazyExpression(expr, evaluator)
			
			result, err := lazy.Evaluate()
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "test-value")
		})
		
		Convey("Lazy evaluator wrapping", func() {
			lev := NewLazyEvaluator()
			
			expr := &Expr{
				Type:    Literal,
				Literal: "wrapped",
			}
			
			evaluator := &Evaluator{Tree: make(map[interface{}]interface{})}
			lazy := lev.WrapExpression(expr, evaluator)
			
			So(lazy, ShouldNotBeNil)
			
			stats := lev.GetStats()
			So(stats.TotalExpressions, ShouldEqual, 1)
		})
		
		Convey("Dependency handling", func() {
			expr1 := &Expr{Type: Literal, Literal: "dep1"}
			expr2 := &Expr{Type: Literal, Literal: "dep2"}
			expr3 := &Expr{Type: Literal, Literal: "main"}
			
			evaluator := &Evaluator{Tree: make(map[interface{}]interface{})}
			
			lazy1 := NewLazyExpression(expr1, evaluator)
			lazy2 := NewLazyExpression(expr2, evaluator)
			lazy3 := NewLazyExpression(expr3, evaluator)
			
			// Set up dependencies
			lazy3.AddDependency(lazy1)
			lazy3.AddDependency(lazy2)
			
			// Evaluating lazy3 should evaluate dependencies first
			result, err := lazy3.Evaluate()
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "main")
			
			// Dependencies should be evaluated
			So(lazy1.IsEvaluated(), ShouldBeTrue)
			So(lazy2.IsEvaluated(), ShouldBeTrue)
			So(lazy3.IsEvaluated(), ShouldBeTrue)
		})
		
		Convey("Statistics tracking", func() {
			lev := NewLazyEvaluator()
			
			expr1 := &Expr{Type: Literal, Literal: "test1"}
			expr2 := &Expr{Type: Literal, Literal: "test2"}
			
			evaluator := &Evaluator{Tree: make(map[interface{}]interface{})}
			
			// Wrap and evaluate expressions
			lazy1 := lev.WrapExpression(expr1, evaluator)
			lazy2 := lev.WrapExpression(expr2, evaluator)
			
			lazy1.Evaluate()
			lazy2.Evaluate()
			
			stats := lev.GetStats()
			So(stats.TotalExpressions, ShouldEqual, 2)
			So(stats.EvaluatedCount, ShouldEqual, 2)
			So(stats.EvaluationTime, ShouldBeGreaterThan, 0)
		})
		
		// TODO: Test uses NewOperatorCall which is not implemented
		/*
		Convey("Expensive operation detection", func() {
			// Create operator expressions for expensive operations
			vaultExpr := NewOperatorCall("vault", []*Expr{
				{Type: Literal, Literal: "secret/path"},
			})
			
			concatExpr := NewOperatorCall("concat", []*Expr{
				{Type: Literal, Literal: "hello"},
				{Type: Literal, Literal: "world"},
			})
			
			So(ShouldUseLazyEvaluation(vaultExpr), ShouldBeTrue)
			So(ShouldUseLazyEvaluation(concatExpr), ShouldBeFalse)
		})
		*/
		
		Convey("Lazy evaluator reset", func() {
			lev := NewLazyEvaluator()
			
			expr := &Expr{Type: Literal, Literal: "test"}
			evaluator := &Evaluator{Tree: make(map[interface{}]interface{})}
			
			lev.WrapExpression(expr, evaluator)
			
			statsBefore := lev.GetStats()
			So(statsBefore.TotalExpressions, ShouldEqual, 1)
			
			lev.Reset()
			
			statsAfter := lev.GetStats()
			So(statsAfter.TotalExpressions, ShouldEqual, 0)
		})
		
		Convey("Thread safety", func() {
			lev := NewLazyEvaluator()
			expr := &Expr{Type: Literal, Literal: "concurrent"}
			evaluator := &Evaluator{Tree: make(map[interface{}]interface{})}
			
			lazy := lev.WrapExpression(expr, evaluator)
			
			// Evaluate concurrently
			done := make(chan bool, 2)
			results := make(chan interface{}, 2)
			errors := make(chan error, 2)
			
			go func() {
				result, err := lazy.Evaluate()
				results <- result
				errors <- err
				done <- true
			}()
			
			go func() {
				result, err := lazy.Evaluate()
				results <- result
				errors <- err
				done <- true
			}()
			
			// Wait for both goroutines
			<-done
			<-done
			
			// Check results
			result1 := <-results
			result2 := <-results
			err1 := <-errors
			err2 := <-errors
			
			So(err1, ShouldBeNil)
			So(err2, ShouldBeNil)
			So(result1, ShouldEqual, "concurrent")
			So(result2, ShouldEqual, "concurrent")
			So(lazy.IsEvaluated(), ShouldBeTrue)
		})
	})
}

func BenchmarkLazyEvaluation(b *testing.B) {
	lev := NewLazyEvaluator()
	evaluator := &Evaluator{Tree: make(map[interface{}]interface{})}
	
	b.Run("LiteralExpression", func(b *testing.B) {
		expr := &Expr{Type: Literal, Literal: "benchmark"}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lazy := lev.WrapExpression(expr, evaluator)
			lazy.Evaluate()
		}
	})
	
	// TODO: Benchmark uses NewOperatorCall which is not implemented
	/*
	b.Run("OperatorExpression", func(b *testing.B) {
		expr := NewOperatorCall("concat", []*Expr{
			{Type: Literal, Literal: "hello"},
			{Type: Literal, Literal: "world"},
		})
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lazy := lev.WrapExpression(expr, evaluator)
			// Note: This will fail since we don't have full operator integration
			// but it tests the lazy evaluation framework
			lazy.Evaluate()
		}
	})
	
	b.Run("MultipleExpressions", func(b *testing.B) {
		expressions := make([]*Expr, 100)
		for i := range expressions {
			expressions[i] = &Expr{Type: Literal, Literal: i}
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, expr := range expressions {
				lazy := lev.WrapExpression(expr, evaluator)
				lazy.Evaluate()
			}
		}
	})
	*/
}