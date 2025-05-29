package operators

import (
	"testing"

	"github.com/geofffranks/simpleyaml"
	. "github.com/smartystreets/goconvey/convey"
	
	"github.com/wayneeseguin/graft/pkg/graft"
)

func TestTypeAwareBooleanOperators(t *testing.T) {
	Convey("Type-Aware Boolean Operators", t, func() {
		// Helper function to test boolean operations
		testBoolean := func(input string, key string, expected interface{}) {
			y, err := simpleyaml.NewYaml([]byte(input))
			So(err, ShouldBeNil)
			
			data, err := y.Map()
			So(err, ShouldBeNil)
			
			ev := &graft.Evaluator{Tree: data}
			err = ev.RunPhase(graft.EvalPhase)
			So(err, ShouldBeNil)
			So(ev.Tree[key], ShouldEqual, expected)
		}
		

		Convey("AND Operator (&&)", func() {
			Convey("should return true when both operands are truthy", func() {
				input := `
a: true
b: true
result: (( a && b ))
`
				testBoolean(input, "result", true)
			})

			Convey("should return false when left operand is falsy", func() {
				input := `
a: false
b: true
result: (( a && b ))
`
				testBoolean(input, "result", false)
			})

			Convey("should return false when right operand is falsy", func() {
				input := `
a: true
b: false
result: (( a && b ))
`
				testBoolean(input, "result", false)
			})

			Convey("should handle numeric truthiness", func() {
				input := `
zero: 0
five: 5
result1: (( zero && five ))
result2: (( five && five ))
`
				var data map[interface{}]interface{}
				y, err := simpleyaml.NewYaml([]byte(input))
				So(err, ShouldBeNil)
				data, err = y.Map()
				So(err, ShouldBeNil)

				ev := &graft.Evaluator{Tree: data}
				err = ev.RunPhase(graft.EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["result1"], ShouldEqual, false) // 0 is falsy
				So(ev.Tree["result2"], ShouldEqual, true)  // non-zero is truthy
			})

			Convey("should handle string truthiness", func() {
				input := `
empty: ""
hello: "hello"
result1: (( empty && hello ))
result2: (( hello && hello ))
`
				var data map[interface{}]interface{}
				y, err := simpleyaml.NewYaml([]byte(input))
				So(err, ShouldBeNil)
				data, err = y.Map()
				So(err, ShouldBeNil)

				ev := &graft.Evaluator{Tree: data}
				err = ev.RunPhase(graft.EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["result1"], ShouldEqual, false) // empty string is falsy
				So(ev.Tree["result2"], ShouldEqual, true)  // non-empty string is truthy
			})

			Convey("should handle nil values", func() {
				input := `
nothing: ~
something: "value"
result: (( nothing && something ))
`
				testBoolean(input, "result", false) // nil is falsy
			})
		})

		Convey("OR Operator (||) as fallback", func() {
			// Note: Currently || is implemented as fallback, not logical OR
			Convey("should return first non-nil value", func() {
				input := `
first: "hello"
second: "world"
result: (( first || second ))
`
				testBoolean(input, "result", "hello")
			})

			Convey("should return second value when first is nil", func() {
				input := `
first: ~
second: "fallback"
result: (( first || second ))
`
				testBoolean(input, "result", "fallback")
			})

			Convey("should return false when it's not nil", func() {
				input := `
first: false
second: true
result: (( first || second ))
`
				testBoolean(input, "result", false) // false is not nil, so it's returned
			})
		})

		Convey("NOT Operator (!)", func() {
			Convey("should negate boolean values", func() {
				input := `
yes: true
no: false
result1: (( !yes ))
result2: (( !no ))
`
				var data map[interface{}]interface{}
				y, err := simpleyaml.NewYaml([]byte(input))
				So(err, ShouldBeNil)
				data, err = y.Map()
				So(err, ShouldBeNil)

				ev := &graft.Evaluator{Tree: data}
				err = ev.RunPhase(graft.EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["result1"], ShouldEqual, false)
				So(ev.Tree["result2"], ShouldEqual, true)
			})

			Convey("should handle numeric truthiness", func() {
				input := `
zero: 0
five: 5
result1: (( !zero ))
result2: (( !five ))
`
				var data map[interface{}]interface{}
				y, err := simpleyaml.NewYaml([]byte(input))
				So(err, ShouldBeNil)
				data, err = y.Map()
				So(err, ShouldBeNil)

				ev := &graft.Evaluator{Tree: data}
				err = ev.RunPhase(graft.EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["result1"], ShouldEqual, true)  // !0 = true
				So(ev.Tree["result2"], ShouldEqual, false) // !5 = false
			})

			Convey("should handle string truthiness", func() {
				input := `
empty: ""
hello: "hello"
result1: (( !empty ))
result2: (( !hello ))
`
				var data map[interface{}]interface{}
				y, err := simpleyaml.NewYaml([]byte(input))
				So(err, ShouldBeNil)
				data, err = y.Map()
				So(err, ShouldBeNil)

				ev := &graft.Evaluator{Tree: data}
				err = ev.RunPhase(graft.EvalPhase)
				So(err, ShouldBeNil)
				So(ev.Tree["result1"], ShouldEqual, true)  // !"" = true
				So(ev.Tree["result2"], ShouldEqual, false) // !"hello" = false
			})
		})

		Convey("Complex Boolean Expressions", func() {
			Convey("should handle mixed operators", func() {
				input := `
a: true
b: false
c: true
result: (( a && (b || c) ))
`
				testBoolean(input, "result", true) // true && (false || true) = true && true = true
			})

			Convey("should handle comparisons with boolean operators", func() {
				input := `
x: 10
y: 5
z: 15
result: (( (x > y) && (x < z) ))
`
				testBoolean(input, "result", true) // (10 > 5) && (10 < 15) = true && true = true
			})
		})
	})
}

func TestTruthinessRules(t *testing.T) {
	Convey("Truthiness rules", t, func() {
		Convey("IsTruthy function", func() {
			// Falsy values
			So(IsTruthy(false), ShouldBeFalse)
			So(IsTruthy(nil), ShouldBeFalse)
			So(IsTruthy(0), ShouldBeFalse)
			So(IsTruthy(int64(0)), ShouldBeFalse)
			So(IsTruthy(0.0), ShouldBeFalse)
			So(IsTruthy(""), ShouldBeFalse)
			So(IsTruthy([]interface{}{}), ShouldBeFalse)
			So(IsTruthy(map[string]interface{}{}), ShouldBeFalse)
			
			// Truthy values
			So(IsTruthy(true), ShouldBeTrue)
			So(IsTruthy(1), ShouldBeTrue)
			So(IsTruthy(int64(42)), ShouldBeTrue)
			So(IsTruthy(-1), ShouldBeTrue)
			So(IsTruthy(3.14), ShouldBeTrue)
			So(IsTruthy("hello"), ShouldBeTrue)
			So(IsTruthy("0"), ShouldBeTrue) // String "0" is truthy
			So(IsTruthy("false"), ShouldBeTrue) // String "false" is truthy
			So(IsTruthy([]interface{}{1, 2, 3}), ShouldBeTrue)
			So(IsTruthy(map[string]interface{}{"key": "value"}), ShouldBeTrue)
		})
	})
}