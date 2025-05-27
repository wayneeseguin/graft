package graft

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/geofffranks/simpleyaml"
	"github.com/starkandwayne/goutils/tree"
)

func TestComparisonOperators(t *testing.T) {
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}
	Convey("Comparison Operators", t, func() {
		Convey("Equality (==) operator", func() {
			Convey("compares numbers correctly", func() {
				ev := &Evaluator{Tree: YAML(`{}`)}
				
				// Integer equality
				op := &ComparisonOperator{op: "=="}
				resp, err := op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: int64(42)},
					&Expr{Type: Literal, Literal: int64(42)},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
				
				// Float equality
				resp, err = op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: 3.14},
					&Expr{Type: Literal, Literal: 3.14},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
				
				// Mixed numeric types
				resp, err = op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: int64(5)},
					&Expr{Type: Literal, Literal: 5.0},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
			})
			
			Convey("compares strings correctly", func() {
				ev := &Evaluator{Tree: YAML(`{}`)}
				op := &ComparisonOperator{op: "=="}
				
				resp, err := op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: "hello"},
					&Expr{Type: Literal, Literal: "hello"},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
				
				resp, err = op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: "hello"},
					&Expr{Type: Literal, Literal: "world"},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, false)
			})
			
			Convey("compares booleans correctly", func() {
				ev := &Evaluator{Tree: YAML(`{}`)}
				op := &ComparisonOperator{op: "=="}
				
				resp, err := op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: true},
					&Expr{Type: Literal, Literal: true},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
			})
			
			Convey("compares nil correctly", func() {
				ev := &Evaluator{Tree: YAML(`{}`)}
				op := &ComparisonOperator{op: "=="}
				
				resp, err := op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: nil},
					&Expr{Type: Literal, Literal: nil},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
			})
		})
		
		Convey("Inequality (!=) operator", func() {
			ev := &Evaluator{Tree: YAML(`{}`)}
			op := &ComparisonOperator{op: "!="}
			
			resp, err := op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: int64(42)},
				&Expr{Type: Literal, Literal: int64(43)},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)
			
			resp, err = op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: "hello"},
				&Expr{Type: Literal, Literal: "hello"},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, false)
		})
		
		Convey("Less than (<) operator", func() {
			ev := &Evaluator{Tree: YAML(`{}`)}
			op := &ComparisonOperator{op: "<"}
			
			Convey("compares numbers", func() {
				resp, err := op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: int64(5)},
					&Expr{Type: Literal, Literal: int64(10)},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
				
				resp, err = op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: 3.14},
					&Expr{Type: Literal, Literal: 2.71},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, false)
			})
			
			Convey("compares strings", func() {
				resp, err := op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: "apple"},
					&Expr{Type: Literal, Literal: "banana"},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
			})
		})
		
		Convey("Greater than (>) operator", func() {
			ev := &Evaluator{Tree: YAML(`{}`)}
			op := &ComparisonOperator{op: ">"}
			
			resp, err := op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: int64(10)},
				&Expr{Type: Literal, Literal: int64(5)},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)
		})
		
		Convey("Less than or equal (<=) operator", func() {
			ev := &Evaluator{Tree: YAML(`{}`)}
			op := &ComparisonOperator{op: "<="}
			
			resp, err := op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: int64(5)},
				&Expr{Type: Literal, Literal: int64(5)},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)
			
			resp, err = op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: int64(5)},
				&Expr{Type: Literal, Literal: int64(10)},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)
		})
		
		Convey("Greater than or equal (>=) operator", func() {
			ev := &Evaluator{Tree: YAML(`{}`)}
			op := &ComparisonOperator{op: ">="}
			
			resp, err := op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: int64(10)},
				&Expr{Type: Literal, Literal: int64(5)},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)
		})
		
		Convey("handles references", func() {
			ev := &Evaluator{Tree: YAML(`
meta:
  value: 42
  name: test
`)}
			op := &ComparisonOperator{op: "=="}
			
			cursor, _ := tree.ParseCursor("meta.value")
			resp, err := op.Run(ev, []*Expr{
				&Expr{Type: Reference, Reference: cursor},
				&Expr{Type: Literal, Literal: int64(42)},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)
		})
	})
}

func TestComparisonIntegration(t *testing.T) {
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}
	Convey("Comparison operators in expressions", t, func() {
		Convey("work with enhanced parser", func() {
			ev := &Evaluator{Tree: YAML(`
name: alice
age: 30
score: 85
`)}
			
			// Test comparison in expression
			result, err := parseAndEvaluateExpression(ev, `(( age > 25 ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, true)
			
			// Test with string comparison
			result, err = parseAndEvaluateExpression(ev, `(( name == "alice" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, true)
			
			// Test complex expression
			result, err = parseAndEvaluateExpression(ev, `(( score >= 80 && score <= 90 ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, true)
		})
	})
}

// Helper function for testing
func parseAndEvaluateExpression(ev *Evaluator, expr string) (interface{}, error) {
	// Enable enhanced parser for tests
	oldUseEnhancedParser := UseEnhancedParser
	UseEnhancedParser = true
	defer func() { UseEnhancedParser = oldUseEnhancedParser }()
	
	opcall, err := ParseOpcallIntegrated(EvalPhase, expr)
	if err != nil {
		return nil, err
	}
	if opcall == nil {
		return nil, nil
	}
	
	resp, err := opcall.Run(ev)
	if err != nil {
		return nil, err
	}
	return resp.Value, nil
}