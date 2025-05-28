package operators

import (
	"testing"

	"github.com/geofffranks/simpleyaml"
	"github.com/starkandwayne/goutils/tree"
	. "github.com/smartystreets/goconvey/convey"
	
	"github.com/wayneeseguin/graft/pkg/graft"
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
				ev := &graft.Evaluator{Tree: YAML(`{}`)}

				// Integer equality
				op := &ComparisonOperator{op: "=="}
				resp, err := op.Run(ev, []*graft.Expr{
					&graft.Expr{Type: graft.Literal, Literal: int64(42)},
					&graft.Expr{Type: graft.Literal, Literal: int64(42)},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)

				// Float equality
				resp, err = op.Run(ev, []*graft.Expr{
					&graft.Expr{Type: graft.Literal, Literal: 3.14},
					&graft.Expr{Type: graft.Literal, Literal: 3.14},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)

				// Mixed numeric types
				resp, err = op.Run(ev, []*graft.Expr{
					&graft.Expr{Type: graft.Literal, Literal: int64(5)},
					&graft.Expr{Type: graft.Literal, Literal: 5.0},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
			})

			Convey("compares strings correctly", func() {
				ev := &graft.Evaluator{Tree: YAML(`{}`)}
				op := &ComparisonOperator{op: "=="}

				resp, err := op.Run(ev, []*graft.Expr{
					&graft.Expr{Type: graft.Literal, Literal: "hello"},
					&graft.Expr{Type: graft.Literal, Literal: "hello"},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)

				resp, err = op.Run(ev, []*graft.Expr{
					&graft.Expr{Type: graft.Literal, Literal: "hello"},
					&graft.Expr{Type: graft.Literal, Literal: "world"},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, false)
			})

			Convey("compares booleans correctly", func() {
				ev := &graft.Evaluator{Tree: YAML(`{}`)}
				op := &ComparisonOperator{op: "=="}

				resp, err := op.Run(ev, []*graft.Expr{
					&graft.Expr{Type: graft.Literal, Literal: true},
					&graft.Expr{Type: graft.Literal, Literal: true},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
			})

			Convey("compares nil correctly", func() {
				ev := &graft.Evaluator{Tree: YAML(`{}`)}
				op := &ComparisonOperator{op: "=="}

				resp, err := op.Run(ev, []*graft.Expr{
					&graft.Expr{Type: graft.Literal, Literal: nil},
					&graft.Expr{Type: graft.Literal, Literal: nil},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
			})
		})

		Convey("Inequality (!=) operator", func() {
			ev := &graft.Evaluator{Tree: YAML(`{}`)}
			op := &ComparisonOperator{op: "!="}

			resp, err := op.Run(ev, []*graft.Expr{
				&graft.Expr{Type: graft.Literal, Literal: int64(42)},
				&graft.Expr{Type: graft.Literal, Literal: int64(43)},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)

			resp, err = op.Run(ev, []*graft.Expr{
				&graft.Expr{Type: graft.Literal, Literal: "hello"},
				&graft.Expr{Type: graft.Literal, Literal: "hello"},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, false)
		})

		Convey("Less than (<) operator", func() {
			ev := &graft.Evaluator{Tree: YAML(`{}`)}
			op := &ComparisonOperator{op: "<"}

			Convey("compares numbers", func() {
				resp, err := op.Run(ev, []*graft.Expr{
					&graft.Expr{Type: graft.Literal, Literal: int64(5)},
					&graft.Expr{Type: graft.Literal, Literal: int64(10)},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)

				resp, err = op.Run(ev, []*graft.Expr{
					&graft.Expr{Type: graft.Literal, Literal: 3.14},
					&graft.Expr{Type: graft.Literal, Literal: 2.71},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, false)
			})

			Convey("compares strings", func() {
				resp, err := op.Run(ev, []*graft.Expr{
					&graft.Expr{Type: graft.Literal, Literal: "apple"},
					&graft.Expr{Type: graft.Literal, Literal: "banana"},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
			})
		})

		Convey("Greater than (>) operator", func() {
			ev := &graft.Evaluator{Tree: YAML(`{}`)}
			op := &ComparisonOperator{op: ">"}

			resp, err := op.Run(ev, []*graft.Expr{
				&graft.Expr{Type: graft.Literal, Literal: int64(10)},
				&graft.Expr{Type: graft.Literal, Literal: int64(5)},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)
		})

		Convey("Less than or equal (<=) operator", func() {
			ev := &graft.Evaluator{Tree: YAML(`{}`)}
			op := &ComparisonOperator{op: "<="}

			resp, err := op.Run(ev, []*graft.Expr{
				&graft.Expr{Type: graft.Literal, Literal: int64(5)},
				&graft.Expr{Type: graft.Literal, Literal: int64(5)},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)

			resp, err = op.Run(ev, []*graft.Expr{
				&graft.Expr{Type: graft.Literal, Literal: int64(5)},
				&graft.Expr{Type: graft.Literal, Literal: int64(10)},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)
		})

		Convey("Greater than or equal (>=) operator", func() {
			ev := &graft.Evaluator{Tree: YAML(`{}`)}
			op := &ComparisonOperator{op: ">="}

			resp, err := op.Run(ev, []*graft.Expr{
				&graft.Expr{Type: graft.Literal, Literal: int64(10)},
				&graft.Expr{Type: graft.Literal, Literal: int64(5)},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, true)
		})

		Convey("handles references", func() {
			ev := &graft.Evaluator{Tree: YAML(`
meta:
  value: 42
  name: test
`)}
			op := &ComparisonOperator{op: "=="}

			cursor, _ := tree.ParseCursor("meta.value")
			resp, err := op.Run(ev, []*graft.Expr{
				&graft.Expr{Type: graft.Reference, Reference: cursor},
				&graft.Expr{Type: graft.Literal, Literal: int64(42)},
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
		Convey("work with  parser", func() {
			ev := &graft.Evaluator{Tree: YAML(`
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
func parseAndEvaluateExpression(ev *graft.Evaluator, expr string) (interface{}, error) {
	// Enable  parser for tests
	// Enhanced parser is now the default

	opcall, err := graft.ParseOpcallCompat(graft.EvalPhase, expr)
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
