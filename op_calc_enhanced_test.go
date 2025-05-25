package spruce

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEnhancedCalcOperator(t *testing.T) {
	Convey("Enhanced Calc Operator", t, func() {
		ev := &Evaluator{
			Tree: map[interface{}]interface{}{
				"numbers": map[interface{}]interface{}{
					"a": 10,
					"b": 5,
					"c": 2.5,
				},
				"formula": "numbers.a * 2",
			},
		}

		// TODO: Fix this test - the enhanced parser is not properly handling nested expressions in calc
		// The issue is that calc expects a literal string argument, but with nested operators,
		// the argument might be an OperatorCall expression that needs to be evaluated first
		SkipConvey("should support nested concat expressions", func() {
			EnableEnhancedParser()
			defer func() { RegisterOp("concat", ConcatOperator{}) }()

			// Create opcall with nested concat
			opcall, err := ParseOpcall(EvalPhase, `(( calc (concat "numbers.a + " "numbers.b") ))`)
			So(err, ShouldBeNil)
			So(opcall, ShouldNotBeNil)

			// Run the operator
			resp, err := opcall.Run(ev)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Type, ShouldEqual, Replace)
			So(resp.Value, ShouldEqual, int64(15)) // 10 + 5
		})

		Convey("should support nested grab expressions", func() {
			EnableEnhancedParser()

			// Create opcall with nested grab
			opcall, err := ParseOpcall(EvalPhase, `(( calc (grab formula) ))`)
			So(err, ShouldBeNil)
			So(opcall, ShouldNotBeNil)

			// Run the operator
			resp, err := opcall.Run(ev)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp.Type, ShouldEqual, Replace)
			So(resp.Value, ShouldEqual, int64(20)) // 10 * 2
		})

		Convey("should handle literal expressions", func() {
			EnableEnhancedParser()

			opcall, err := ParseOpcall(EvalPhase, `(( calc "5 + 3 * 2" ))`)
			So(err, ShouldBeNil)
			So(opcall, ShouldNotBeNil)

			resp, err := opcall.Run(ev)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, int64(11)) // 5 + (3 * 2)
		})

		Convey("should resolve references in expressions", func() {
			EnableEnhancedParser()

			opcall, err := ParseOpcall(EvalPhase, `(( calc "numbers.a + numbers.b * numbers.c" ))`)
			So(err, ShouldBeNil)
			So(opcall, ShouldNotBeNil)

			resp, err := opcall.Run(ev)
			So(err, ShouldBeNil)
			So(resp.Value, ShouldEqual, 22.5) // 10 + (5 * 2.5)
		})

		Convey("should support math functions", func() {
			EnableEnhancedParser()

			testCases := []struct {
				expr     string
				expected float64
			}{
				{`(( calc "min(numbers.a, numbers.b)" ))`, 5},
				{`(( calc "max(numbers.a, numbers.b)" ))`, 10},
				{`(( calc "pow(numbers.b, 2)" ))`, 25},
				{`(( calc "sqrt(16)" ))`, 4},
				{`(( calc "floor(numbers.c)" ))`, 2},
				{`(( calc "ceil(numbers.c)" ))`, 3},
			}

			for _, tc := range testCases {
				opcall, err := ParseOpcall(EvalPhase, tc.expr)
				So(err, ShouldBeNil)
				So(opcall, ShouldNotBeNil)

				resp, err := opcall.Run(ev)
				So(err, ShouldBeNil)
				So(resp.Value, ShouldEqual, tc.expected)
			}
		})

		Convey("should handle errors gracefully", func() {
			EnableEnhancedParser()

			Convey("nil reference error", func() {
				ev.Tree["nil_val"] = nil
				opcall, err := ParseOpcall(EvalPhase, `(( calc "nil_val + 5" ))`)
				So(err, ShouldBeNil)

				_, err = opcall.Run(ev)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "nil")
			})

			Convey("invalid expression error", func() {
				opcall, err := ParseOpcall(EvalPhase, `(( calc "5 ++ 3" ))`)
				So(err, ShouldBeNil)

				_, err = opcall.Run(ev)
				So(err, ShouldNotBeNil)
			})

			Convey("wrong number of arguments", func() {
				opcall, err := ParseOpcall(EvalPhase, `(( calc "5 + 3" "extra" ))`)
				So(err, ShouldBeNil)

				_, err = opcall.Run(ev)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "one")
			})
		})
	})
}