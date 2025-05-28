package operators

import (
	"testing"

	"github.com/geofffranks/simpleyaml"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBooleanOperators(t *testing.T) {
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}
	Convey("Boolean Operators", t, func() {
		Convey("Logical AND (&&) operator", func() {
			ev := &Evaluator{Tree: YAML(`{}`)}
			op := BooleanAndOperator{}

			Convey("returns true when both operands are truthy", func() {
				resp, err := op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: true},
					&Expr{Type: Literal, Literal: true},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)

				// Non-zero numbers are truthy
				resp, err = op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: int64(1)},
					&Expr{Type: Literal, Literal: "hello"},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
			})

			Convey("returns false when any operand is falsy", func() {
				resp, err := op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: true},
					&Expr{Type: Literal, Literal: false},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, false)

				// Zero is falsy
				resp, err = op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: int64(0)},
					&Expr{Type: Literal, Literal: true},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, false)

				// Empty string is falsy
				resp, err = op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: ""},
					&Expr{Type: Literal, Literal: true},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, false)

				// nil is falsy
				resp, err = op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: nil},
					&Expr{Type: Literal, Literal: true},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, false)
			})

			Convey("short-circuits on first falsy value", func() {
				// This would error if evaluated, but shouldn't be evaluated
				// because first operand is false
				errorExpr := &Expr{Type: Reference, Reference: nil} // Invalid reference

				resp, err := op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: false},
					errorExpr,
				})
				So(err, ShouldBeNil) // No error because second operand not evaluated
				So(resp.Value, ShouldEqual, false)
			})
		})

		Convey("Logical OR (||) operator", func() {
			ev := &Evaluator{Tree: YAML(`{}`)}
			op := BooleanOrOperator{}

			Convey("returns true when any operand is truthy", func() {
				resp, err := op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: false},
					&Expr{Type: Literal, Literal: true},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)

				resp, err = op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: int64(0)},
					&Expr{Type: Literal, Literal: "hello"},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, true)
			})

			Convey("returns false when both operands are falsy", func() {
				resp, err := op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: false},
					&Expr{Type: Literal, Literal: false},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, false)

				resp, err = op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: int64(0)},
					&Expr{Type: Literal, Literal: ""},
				})
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, false)
			})

			Convey("short-circuits on first truthy value", func() {
				// This would error if evaluated
				errorExpr := &Expr{Type: Reference, Reference: nil}

				resp, err := op.Run(ev, []*Expr{
					&Expr{Type: Literal, Literal: true},
					errorExpr,
				})
				So(err, ShouldBeNil) // No error because second operand not evaluated
				So(resp.Value, ShouldEqual, true)
			})
		})
	})
}

func TestTruthiness(t *testing.T) {
	Convey("Truthiness rules", t, func() {
		Convey("falsy values", func() {
			So(isTruthy(false), ShouldBeFalse)
			So(isTruthy(nil), ShouldBeFalse)
			So(isTruthy(int64(0)), ShouldBeFalse)
			So(isTruthy(0.0), ShouldBeFalse)
			So(isTruthy(""), ShouldBeFalse)
			So(isTruthy([]interface{}{}), ShouldBeFalse)
			So(isTruthy(map[string]interface{}{}), ShouldBeFalse)
		})

		Convey("truthy values", func() {
			So(isTruthy(true), ShouldBeTrue)
			So(isTruthy(int64(1)), ShouldBeTrue)
			So(isTruthy(-1.5), ShouldBeTrue)
			So(isTruthy("hello"), ShouldBeTrue)
			So(isTruthy("0"), ShouldBeTrue)     // String "0" is truthy
			So(isTruthy("false"), ShouldBeTrue) // String "false" is truthy
			So(isTruthy([]interface{}{1}), ShouldBeTrue)
			So(isTruthy(map[string]interface{}{"a": 1}), ShouldBeTrue)
		})
	})
}

func TestBooleanIntegration(t *testing.T) {
	YAML := func(s string) map[interface{}]interface{} {
		y, err := simpleyaml.NewYaml([]byte(s))
		So(err, ShouldBeNil)

		data, err := y.Map()
		So(err, ShouldBeNil)

		return data
	}
	Convey("Boolean operators in expressions", t, func() {
		ev := &Evaluator{Tree: YAML(`
enabled: true
debug: false
count: 5
name: test
empty: ""
`)}

		Convey("work with  parser", func() {
			// Test AND
			result, err := parseAndEvaluateExpression(ev, `(( enabled && count > 0 ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, true)

			// Test fallback (|| is fallback, not boolean OR)
			result, err = parseAndEvaluateExpression(ev, `(( debug || name == "test" ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, false) // debug evaluates to false, so that's returned

			// Test complex expression
			result, err = parseAndEvaluateExpression(ev, `(( (enabled && !debug) || empty ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, true) // (enabled && !debug) evaluates to true
		})

		Convey("handle precedence correctly", func() {
			// && has higher precedence than ||
			result, err := parseAndEvaluateExpression(ev, `(( debug || enabled && count > 0 ))`)
			So(err, ShouldBeNil)
			So(result, ShouldEqual, false) // || is fallback, so returns debug's value (false)
		})
	})
}
