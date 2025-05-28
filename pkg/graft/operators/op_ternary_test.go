package operators

import (
	"testing"

	"github.com/geofffranks/simpleyaml"
	"github.com/starkandwayne/goutils/tree"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTernaryOperator(t *testing.T) {
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}
	Convey("Ternary Operator (?:)", t, func() {
		ev := &Evaluator{Tree: YAML(`{}`)}
		op := TernaryOperator{}

		Convey("selects true branch when condition is truthy", func() {
			resp, err := op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: true},
				&Expr{Type: Literal, Literal: "yes"},
				&Expr{Type: Literal, Literal: "no"},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "yes")

			// Non-zero number is truthy
			resp, err = op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: int64(1)},
				&Expr{Type: Literal, Literal: "positive"},
				&Expr{Type: Literal, Literal: "zero or negative"},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "positive")
		})

		Convey("selects false branch when condition is falsy", func() {
			resp, err := op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: false},
				&Expr{Type: Literal, Literal: "yes"},
				&Expr{Type: Literal, Literal: "no"},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "no")

			// Zero is falsy
			resp, err = op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: int64(0)},
				&Expr{Type: Literal, Literal: "truthy"},
				&Expr{Type: Literal, Literal: "falsy"},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "falsy")
		})

		Convey("only evaluates the selected branch", func() {
			// This would error if evaluated
			errorExpr := &Expr{Type: Reference, Reference: nil}

			// Condition is true, so false branch shouldn't be evaluated
			resp, err := op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: true},
				&Expr{Type: Literal, Literal: "success"},
				errorExpr,
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "success")

			// Condition is false, so true branch shouldn't be evaluated
			resp, err = op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: false},
				errorExpr,
				&Expr{Type: Literal, Literal: "fallback"},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, "fallback")
		})

		Convey("handles references in branches", func() {
			ev := &Evaluator{Tree: YAML(`
production: false
dev_port: 8080
prod_port: 80
`)}

			prodCursor, _ := tree.ParseCursor("production")
			devPortCursor, _ := tree.ParseCursor("dev_port")
			prodPortCursor, _ := tree.ParseCursor("prod_port")

			resp, err := op.Run(ev, []*Expr{
				&Expr{Type: Reference, Reference: prodCursor},
				&Expr{Type: Reference, Reference: prodPortCursor},
				&Expr{Type: Reference, Reference: devPortCursor},
			})
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, int64(8080))
		})

		Convey("requires exactly 3 arguments", func() {
			_, err := op.Run(ev, []*Expr{
				&Expr{Type: Literal, Literal: true},
				&Expr{Type: Literal, Literal: "yes"},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "3 arguments")
		})
	})
}

func TestTernaryIntegration(t *testing.T) {
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}
	Convey("Ternary operator in expressions", t, func() {
		ev := &Evaluator{Tree: YAML(`
age: 25
score: 85
environment: production
debug: false
`)}

		Convey("works with enhanced parser", func() {
			// Simple ternary
			result, err := parseAndEvaluateExpression(ev, `(( age >= 18 ? "adult" : "minor" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "adult")

			// Ternary with references
			result, err = parseAndEvaluateExpression(ev, `(( environment == "production" ? 80 : 8080 ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, int64(80))

			// Nested ternary
			result, err = parseAndEvaluateExpression(ev, `(( score >= 90 ? "A" : score >= 80 ? "B" : "C" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "B")
		})

		Convey("handles complex expressions", func() {
			// Ternary with boolean operators
			result, err := parseAndEvaluateExpression(ev, `(( environment == "production" && !debug ? "optimized" : "debug" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "optimized")

			// Ternary with arithmetic
			result, err = parseAndEvaluateExpression(ev, `(( score > 80 ? score * 1.1 : score * 0.9 ))`)
			So(err, ShouldBeNil)
			So(result.(float64), ShouldAlmostEqual, 93.5, 0.001)
		})

		Convey("respects precedence", func() {
			// Ternary has lowest precedence
			result, err := parseAndEvaluateExpression(ev, `(( 1 + 2 > 2 ? "yes" : "no" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, "yes") // (1 + 2) > 2 ? "yes" : "no"
		})
	})
}
